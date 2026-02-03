package network

import (
	"errors"
	"fmt"
	"packet-repackage/database"
	"packet-repackage/models"
	"packet-repackage/utils/command"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	LinkTypeAccess       = "access"
	LinkTypeTrunk        = "trunk"
	Default_VlanId       = "1"
	Vlan_Filter          = "ip link add Bridge up type bridge vlan_filtering 1"
	Cmd_Br_Vlan_Create   = "bridge vlan add vid %s dev Bridge self"
	Cmd_Add_Port         = "ip link set %s master Bridge"
	Cmd_Del_Port         = "ip link set %s nomaster"
	Cmd_Access           = "bridge vlan add vid %s dev %s master pvid untagged"
	Cmd_Trunk            = "bridge vlan add vid %s dev %s master"
	Cmd_Vlan_If          = "ip link add link Bridge name %s %s type vlan id %s"
	Cmd_Vlan_If_Delete   = "ip link del %s"
	Cmd_Vlan_If_Ip       = "ip addr add %s dev %s"
	Cmd_Vlan_If_Ip_Flush = "ip -4 addr flush dev %s"
	Cmd_Vlan_If_Up_Down  = "ip link set dev %s %s"
)

// EnsureBridgeExists ensures the Bridge exists and is configured correctly
func EnsureBridgeExists() error {
	database.Logger.Info("Ensuring Bridge exists and is configured")
	
	// Try to create bridge (ignore error if exists)
	command.GoLinuxShell("ip link add Bridge type bridge vlan_filtering 1")
	
	// Enforce vlan_filtering setting (in case it existed but was off)
	err := command.GoLinuxShell("ip link set Bridge type bridge vlan_filtering 1")
	if err != nil {
		database.Logger.Warn("Failed to enforce vlan_filtering on Bridge", zap.Error(err))
	}

	// Enforce UP state
	err = command.GoLinuxShell("ip link set Bridge up")
	if err != nil {
		return fmt.Errorf("failed to bring Bridge up: %w", err)
	}
	
	return nil
}

// AddVlan adds VLAN configuration to an interface
func AddVlan(interfaceConfig models.InterfaceConfig, interfaceName string, db *gorm.DB) error {
	database.Logger.Info("Configuring VLAN on interface",
		zap.String("interface", interfaceName),
		zap.String("link_type", interfaceConfig.LinkType),
		zap.String("vlan_id", interfaceConfig.VlanId))
	
	// Check if already in bridge
	isSlave, _ := checkInterfaceMaster(interfaceName, "Bridge")
	
	var err error

	if !isSlave {
		// Reset/Remove from any other master
		ResetDefaultVlan(interfaceName)

		// Add interface to bridge
		addPortCmd := fmt.Sprintf(Cmd_Add_Port, interfaceName)
		err = command.GoLinuxShell(addPortCmd)
		if err != nil {
			return fmt.Errorf("failed to add interface to bridge: %w", err)
		}

		// Ensure interface is UP
		upCmd := fmt.Sprintf("ip link set %s up", interfaceName)
		err = command.GoLinuxShell(upCmd)
		if err != nil {
			return fmt.Errorf("failed to bring interface up: %w", err)
		}
		
		// Wait for interface to be recognized as bridge port
		maxRetries := 50
		for i := 0; i < maxRetries; i++ {
			isSlaveNow, err := checkInterfaceMaster(interfaceName, "Bridge")
			if err == nil && isSlaveNow {
				break
			}
			if i == maxRetries-1 {
				database.Logger.Warn("Interface did not become bridge slave in time, proceeding anyway", 
					zap.String("interface", interfaceName))
			}
			time.Sleep(100 * time.Millisecond)
		}
	} else {
		database.Logger.Info("Interface already in Bridge, skipping attachment", zap.String("interface", interfaceName))
		// Still ensure it is UP
		command.GoLinuxShell(fmt.Sprintf("ip link set %s up", interfaceName))
	}

	// Access mode
	if interfaceConfig.LinkType == LinkTypeAccess {
		createVlanCmd := fmt.Sprintf(Cmd_Br_Vlan_Create, interfaceConfig.VlanId)
		err = command.GoLinuxShell(createVlanCmd)
		if err != nil {
			return fmt.Errorf("failed to create VLAN: %w", err)
		}

		accessCmd := fmt.Sprintf(Cmd_Access, interfaceConfig.VlanId, interfaceName)
		err = command.GoLinuxShell(accessCmd)
		if err != nil {
			return fmt.Errorf("failed to set access VLAN: %w", err)
		}

	} else if interfaceConfig.LinkType == LinkTypeTrunk {
		// Trunk mode
		if len(interfaceConfig.TrunkVlanId) > 0 {
			vlanIds := strings.Split(interfaceConfig.TrunkVlanId, ",")
			for _, vlan := range vlanIds {
				trunkCmd := fmt.Sprintf(Cmd_Trunk, vlan, interfaceName)
				err = command.GoLinuxShell(trunkCmd)
				if err != nil {
					return fmt.Errorf("failed to set trunk VLAN: %w", err)
				}
			}
		}

		// Default ID
		trunkDefaultIdCmd := fmt.Sprintf(Cmd_Access, interfaceConfig.DefaultId, interfaceName)
		err = command.GoLinuxShell(trunkDefaultIdCmd)
		if err != nil {
			return fmt.Errorf("failed to set trunk default VLAN: %w", err)
		}
	}
	
	// Automatically create VLAN interfaces
	database.Logger.Info("Creating VLAN interfaces", zap.String("interface", interfaceName))
	err = AddVlanIf(db, interfaceConfig)
	if err != nil {
		database.Logger.Error("Failed to create VLAN interfaces", zap.Error(err))
		return fmt.Errorf("failed to create VLAN interfaces: %w", err)
	}

	return nil
}

// ResetDefaultVlan removes interface from bridge
func ResetDefaultVlan(interfaceName string) error {
	delVlanCmd := fmt.Sprintf(Cmd_Del_Port, interfaceName)
	err := command.GoLinuxShell(delVlanCmd)
	if err != nil {
		return fmt.Errorf("failed to reset VLAN: %w", err)
	}
	return nil
}

// ListBridge checks if bridge exists
func ListBridge(bridgeName string) bool {
	err := command.GoLinuxShell("ip link show", bridgeName)
	return err == nil
}

// DelBridge deletes a bridge
func DelBridge(bridgeName string) error {
	exist := ListBridge(bridgeName)
	if exist {
		return command.GoLinuxShell("ip link del", bridgeName)
	}
	return nil
}

// ValidateVlanInterface checks if a VLAN interface exists
func ValidateVlanInterface(vlanInterface string) error {
	err := command.GoLinuxShell("ip", "link", "show", vlanInterface)
	if err != nil {
		return fmt.Errorf("VLAN interface %s does not exist", vlanInterface)
	}
	return nil
}

// VlanIfIpAdd adds IP addresses to VLAN interface
func VlanIfIpAdd(ipList []string, vlanInterface string) error {
	database.Logger.Info("Adding IP addresses to VLAN interface",
		zap.String("interface", vlanInterface),
		zap.Strings("ips", ipList))
	
	// Validate interface exists
	if err := ValidateVlanInterface(vlanInterface); err != nil {
		database.Logger.Error("VLAN interface validation failed",
			zap.String("interface", vlanInterface),
			zap.Error(err))
		return err
	}
	
	for _, v := range ipList {
		cmd := fmt.Sprintf(Cmd_Vlan_If_Ip, v, vlanInterface)
		database.Logger.Info("Executing IP add command",
			zap.String("command", cmd),
			zap.String("ip", v))
		
		err := command.GoLinuxShell(cmd)
		if err != nil {
			database.Logger.Error("Failed to add IP address",
				zap.String("interface", vlanInterface),
				zap.String("ip", v),
				zap.Error(err))
			return fmt.Errorf("failed to add IP %s to %s: %w", v, vlanInterface, err)
		}
		
		database.Logger.Info("Successfully added IP address",
			zap.String("interface", vlanInterface),
			zap.String("ip", v))
	}
	
	database.Logger.Info("All IP addresses added successfully",
		zap.String("interface", vlanInterface))
	return nil
}

// VlanIfIpFlush removes all IPs from VLAN interface
func VlanIfIpFlush(vlanInterface string) error {
	cmd := fmt.Sprintf(Cmd_Vlan_If_Ip_Flush, vlanInterface)
	return command.GoLinuxShell(cmd)
}

// VlanIfUpAndDown changes interface status
func VlanIfUpAndDown(vlanInterface, action string) error {
	cmd := fmt.Sprintf(Cmd_Vlan_If_Up_Down, vlanInterface, action)
	return command.GoLinuxShell(cmd)
}

// SplitAndAddList parses trunk VLAN ID ranges and adds default ID
func SplitAndAddList(trunkVlanId, defaultID string) ([]string, error) {
	var result []string

	ranges := strings.Split(trunkVlanId, ",")
	for _, r := range ranges {
		r = strings.TrimSpace(r)
		if strings.Contains(r, "-") {
			nums := strings.Split(r, "-")
			if len(nums) != 2 {
				return nil, errors.New("invalid trunk VLAN ID format")
			}

			start, err := strconv.Atoi(strings.TrimSpace(nums[0]))
			if err != nil {
				return nil, err
			}

			end, err := strconv.Atoi(strings.TrimSpace(nums[1]))
			if err != nil {
				return nil, err
			}

			for i := start; i <= end; i++ {
				result = append(result, strconv.Itoa(i))
			}
		} else {
			num, err := strconv.Atoi(r)
			if err != nil {
				return nil, err
			}
			result = append(result, strconv.Itoa(num))
		}
	}

	// Add default ID if not present
	found := false
	for _, num := range result {
		if num == defaultID {
			found = true
			break
		}
	}
	if !found {
		result = append(result, defaultID)
	}

	return result, nil
}

// AddVlanIf creates VLAN interface and manages database entry
func AddVlanIf(db *gorm.DB, interfaceConfig models.InterfaceConfig) error {
	if interfaceConfig.LinkType == "access" {
		var vlanConfig models.VlanConfig
		vlanInterface := "vlan_" + interfaceConfig.VlanId
		db.Model(&models.VlanConfig{}).Where("out_interface = ?", vlanInterface).Find(&vlanConfig)
		if vlanConfig.ID == 0 {
			return addVlanIfAdd(db, interfaceConfig.OutInterface, interfaceConfig.VlanId, vlanInterface)
		} else {
			return addVlanIfUpdate(db, vlanConfig, interfaceConfig.OutInterface)
		}
	} else if interfaceConfig.LinkType == "trunk" {
		// Get all existing VLAN interfaces
		var vlanIfList []models.VlanConfig
		db.Model(&models.VlanConfig{}).Find(&vlanIfList)
		
		vlanMap := make(map[string]models.VlanConfig)
		for _, v := range vlanIfList {
			vlanMap[v.OutInterface] = v
		}

		// Parse trunk IDs
		arr, err := SplitAndAddList(interfaceConfig.TrunkVlanId, interfaceConfig.DefaultId)
		if err != nil {
			return err
		}

		// Create or update VLAN interfaces
		for _, trunkId := range arr {
			vlanInterface := "vlan_" + trunkId
			if v, ok := vlanMap[vlanInterface]; !ok {
				err = addVlanIfAdd(db, interfaceConfig.OutInterface, trunkId, vlanInterface)
			} else {
				err = addVlanIfUpdate(db, v, interfaceConfig.OutInterface)
			}
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func addVlanIfAdd(db *gorm.DB, eth string, vlanId string, vlanInterface string) error {
	vlanIntId, _ := strconv.Atoi(vlanId)
	vlanConfig := models.VlanConfig{
		VlanId:            vlanIntId,
		OutInterface:      vlanInterface,
		Type:              "2",
		IsManager:         0,
		PhysicalInterface: eth,
		Status:            1,
	}
	
	err := db.Create(&vlanConfig).Error
	if err != nil {
		return fmt.Errorf("failed to add vlanIf: %w", err)
	}
	
	vlanIfCmd := fmt.Sprintf(Cmd_Vlan_If, vlanInterface, "up", vlanId)
	command.GoLinuxShell(vlanIfCmd)
	return nil
}

func addVlanIfUpdate(db *gorm.DB, vlanConfig models.VlanConfig, id string) error {
	newPhysicalInterface := combineAndSortStrings(vlanConfig.PhysicalInterface, id)
	err := db.Model(&vlanConfig).Update("physical_interface", newPhysicalInterface).Error
	if err != nil {
		return fmt.Errorf("failed to update vlanIf: %w", err)
	}
	return nil
}

func combineAndSortStrings(A, B string) string {
	AList := strings.Split(A, ",")
	BList := strings.Split(B, ",")

	for _, b := range BList {
		found := false
		for _, a := range AList {
			if strings.TrimSpace(a) == strings.TrimSpace(b) {
				found = true
				break
			}
		}
		if !found {
			AList = append(AList, b)
		}
	}

	sort.Strings(AList)
	return strings.Join(AList, ",")
}

// RemoveVlan removes VLAN configuration from an interface
func RemoveVlan(interfaceName string, db *gorm.DB) error {
	database.Logger.Info("Removing VLAN from interface", zap.String("interface", interfaceName))
	
	// Get current interface configuration to find associated VLANs
	var ifaceConfig models.InterfaceConfig
	err := db.Where("out_interface = ?", interfaceName).First(&ifaceConfig).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to find interface config: %w", err)
	}
	
	// Remove interface from bridge
	err = ResetDefaultVlan(interfaceName)
	if err != nil {
		database.Logger.Error("Failed to remove from bridge", zap.Error(err))
		return err
	}
	
	// Remove interface from VLAN tracking
	if ifaceConfig.ID != 0 {
		if ifaceConfig.LinkType == LinkTypeAccess {
			err = RemoveVlanIf(db, interfaceName, []string{ifaceConfig.VlanId})
		} else if ifaceConfig.LinkType == LinkTypeTrunk {
			vlanIds, _ := SplitAndAddList(ifaceConfig.TrunkVlanId, ifaceConfig.DefaultId)
			err = RemoveVlanIf(db, interfaceName, vlanIds)
		}
		if err != nil {
			database.Logger.Error("Failed to cleanup VLAN interfaces", zap.Error(err))
		}
		
		// Delete interface config from database
		db.Delete(&ifaceConfig)
	}
	
	database.Logger.Info("Successfully removed VLAN configuration", zap.String("interface", interfaceName))
	return nil
}

// RemoveVlanIf removes physical interface from VLAN tracking and deletes unused vlan_X interfaces
func RemoveVlanIf(db *gorm.DB, physicalInterface string, vlanIds []string) error {
	database.Logger.Info("Removing physical interface from VLAN tracking",
		zap.String("physical_interface", physicalInterface),
		zap.Strings("vlan_ids", vlanIds))
	
	for _, vlanId := range vlanIds {
		vlanInterface := "vlan_" + vlanId
		var vlanConfig models.VlanConfig
		
		err := db.Where("out_interface = ?", vlanInterface).First(&vlanConfig).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				database.Logger.Warn("VLAN interface not found in database",
					zap.String("vlan_interface", vlanInterface))
				continue
			}
			return fmt.Errorf("failed to find VLAN interface %s: %w", vlanInterface, err)
		}
		
		// Remove physical interface from the list
		physicalList := strings.Split(vlanConfig.PhysicalInterface, ",")
		var newList []string
		for _, iface := range physicalList {
			if strings.TrimSpace(iface) != physicalInterface {
				newList = append(newList, strings.TrimSpace(iface))
			}
		}
		
		// If no physical interfaces remain, delete the vlan_X interface
		if len(newList) == 0 {
			database.Logger.Info("No interfaces using VLAN, deleting vlan interface",
				zap.String("vlan_interface", vlanInterface))
			
			// Delete the network interface
			delCmd := fmt.Sprintf(Cmd_Vlan_If_Delete, vlanInterface)
			err = command.GoLinuxShell(delCmd)
			if err != nil {
				database.Logger.Warn("Failed to delete vlan interface (may not exist)",
					zap.String("vlan_interface", vlanInterface),
					zap.Error(err))
			}
			
			// Delete from database
			err = db.Delete(&vlanConfig).Error
			if err != nil {
				return fmt.Errorf("failed to delete vlan config from database: %w", err)
			}
			
			database.Logger.Info("Deleted VLAN interface",
				zap.String("vlan_interface", vlanInterface))
		} else {
			// Update the physical interface list
			newPhysicalInterface := strings.Join(newList, ",")
			database.Logger.Info("Updating VLAN interface physical list",
				zap.String("vlan_interface", vlanInterface),
				zap.String("new_list", newPhysicalInterface))
			
			err = db.Model(&vlanConfig).Update("physical_interface", newPhysicalInterface).Error
			if err != nil {
				return fmt.Errorf("failed to update vlan interface %s: %w", vlanInterface, err)
			}
		}
	}
	
	return nil
}

// checkInterfaceMaster checks if the interface has the specified master
func checkInterfaceMaster(interfaceName, masterName string) (bool, error) {
	// We use ip -d link show because it contains "master <name>"
	// output format example: ... master Bridge ...
	cmd := fmt.Sprintf("ip -d link show %s", interfaceName)
	output, err := command.GoLinuxShellWithResult(cmd)
	if err != nil {
		return false, err
	}
	
	return strings.Contains(output, fmt.Sprintf("master %s", masterName)), nil
}
