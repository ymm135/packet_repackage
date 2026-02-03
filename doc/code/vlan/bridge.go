package network

import (
	"errors"
	fmt "fmt"
	"sort"
	"strconv"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
	"netvine.com/firewall/server/global"
	"netvine.com/firewall/server/model/netManage"
	"netvine.com/firewall/server/utils/command"
	firewall_error "netvine.com/firewall/server/utils/errors"
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
	Cmd_Vlan_If_Flush    = "ip link show | grep \"vlan_\" | awk -F: '{print $2}'|awk -F@ '{print $1}' | xargs -I {} sudo ip link delete {}"
)

// AddVlan 增加vlan
// bridge vlan { add | del } dev DEV vid VID [ tunnel_info TUNNEL_ID ] [ pvid ] [ untagged ] [ self ] [ master ]
// bridge vlan [ show | tunnelshow ] [ dev DEV ]
func AddVlan(interfaceConfig netManage.InterfaceConfig, interfaceName string) error {
	// 开启
	command.GoLinuxShell(Vlan_Filter)

	// 先删除之前的信息
	err := ResetDefaultVlan(interfaceName)
	if err != nil {
		return firewall_error.CreateError("清除VLAN信息失败!")
	}

	// 添加接口到桥
	addPortCmd := fmt.Sprintf(Cmd_Add_Port, interfaceName)
	err = command.GoLinuxShell(addPortCmd)
	if err != nil {
		return firewall_error.CreateError("创建VLAN时,添加接口失败!")
	}

	// 如果是access模式
	if interfaceConfig.LinkType == LinkTypeAccess {
		// 创建vlan
		createVlanCmd := fmt.Sprintf(Cmd_Br_Vlan_Create, interfaceConfig.VlanId)
		err = command.GoLinuxShell(createVlanCmd)
		if err != nil {
			return firewall_error.CreateError("创建VLAN失败!")
		}

		// vlan id
		accessCmd := fmt.Sprintf(Cmd_Access, interfaceConfig.VlanId, interfaceName)
		err = command.GoLinuxShell(accessCmd)
		if err != nil {
			global.NETVINE_LOG.Error("access", zap.Error(err))
			return firewall_error.CreateError("Access VLAN设置失败!")
		}

	} else if interfaceConfig.LinkType == LinkTypeTrunk { // trunk 模式

		if len(interfaceConfig.TrunkVlanId) > 0 {
			valnIds := strings.Split(interfaceConfig.TrunkVlanId, ",")
			for _, vlan := range valnIds { // 只能一个一个设置
				trunkCmd := fmt.Sprintf(Cmd_Trunk, vlan, interfaceName)
				err = command.GoLinuxShell(trunkCmd)
				if err != nil {
					global.NETVINE_LOG.Error("trunk", zap.Error(err))
					return firewall_error.CreateError("Trunk VLAN设置失败!")
				}
			}
		}

		// default id
		trunkDefaultIdCmd := fmt.Sprintf(Cmd_Access, interfaceConfig.DefaultId, interfaceName)
		err = command.GoLinuxShell(trunkDefaultIdCmd)
		if err != nil {
			global.NETVINE_LOG.Error("trunk", zap.Error(err))
			return firewall_error.CreateError("Trunk VLAN设置失败!")
		}
	}

	result, _ := command.GoLinuxShellWithResult("bridge vlan")
	global.NETVINE_LOG.Debug("show", zap.String("vlan", result))
	return err
}

func ResetDefaultVlan(interfaceName string) error {

	delVlanCmd := fmt.Sprintf(Cmd_Del_Port, interfaceName)
	err := command.GoLinuxShell(delVlanCmd)
	if err != nil {
		global.NETVINE_LOG.Error("access", zap.Error(err))
		return firewall_error.CreateError("删除VLAN重置失败!")
	}
	return err
}

func ListBridge(bridgeName string) bool {
	// get bridge
	// Device "ETH9" does not exist.
	result := true
	err := command.GoLinuxShell("ip link list", bridgeName)
	if err != nil {
		result = false
	}
	return result
}

func DelBridge(bridgeName string) error {
	exist := ListBridge(bridgeName)
	var err error
	if exist {
		err = command.GoLinuxShell("ip link del", bridgeName)
	}
	return err
}

func AddVlanIf(interfaceConfig netManage.InterfaceConfig) error {
	if interfaceConfig.LinkType == "access" {
		var vlanConfig netManage.VlanConfig
		vlanInterface := "vlan_" + interfaceConfig.VlanId
		global.NETVINE_DB.Model(&netManage.VlanConfig{}).Where("out_interface", vlanInterface).Find(&vlanConfig)
		if vlanConfig.Id == 0 {
			return AddVlanIfAdd(interfaceConfig.OutInterface, interfaceConfig.VlanId, vlanInterface)
		} else {
			return AddVlanIfUpdate(vlanConfig, interfaceConfig.OutInterface)
		}
	} else if interfaceConfig.LinkType == "trunk" {
		//先把所有数据捞出来
		var vlanIfList []netManage.VlanConfig
		global.NETVINE_DB.Model(&netManage.VlanConfig{}).Where("1=1").Find(&vlanIfList)
		//map
		vlanMap := make(map[string]netManage.VlanConfig)
		for _, v := range vlanIfList {
			vlanMap[v.OutInterface] = v
		}
		//解析trunkId
		arr, _ := splitAndAddList(interfaceConfig.TrunkVlanId, interfaceConfig.DefaultId)
		//根据trunkId去修改或者新增
		for _, trunkId := range arr {
			vlanInterface := "vlan_" + trunkId
			var tempErr error
			if v, ok := vlanMap[vlanInterface]; !ok {
				//新增
				tempErr = AddVlanIfAdd(interfaceConfig.OutInterface, trunkId, vlanInterface)
			} else {
				//修改
				tempErr = AddVlanIfUpdate(v, interfaceConfig.OutInterface)
			}
			if tempErr != nil {
				return tempErr
			}
		}
	}
	return nil
}

func AddVlanIfAdd(eth string, vlanId string, vlanInterface string) error {
	var vlanConfig netManage.VlanConfig
	vlanIntId, _ := strconv.Atoi(vlanId)
	vlanConfig.VlanId = vlanIntId
	vlanConfig.OutInterface = vlanInterface
	vlanConfig.Type = "2"
	vlanConfig.IsManager = 0
	vlanConfig.PhysicalInterface = eth
	vlanConfig.Status = 1
	err := global.NETVINE_DB.Create(&vlanConfig).Error
	if err != nil {
		global.NETVINE_LOG.Error("AddVlanIfAdd", zap.Error(err))
		return errors.New("增加vlanIf失败")
	}
	vlanIfCmd := fmt.Sprintf(Cmd_Vlan_If, vlanInterface, "up", vlanId)
	command.GoLinuxShell(vlanIfCmd)
	return nil
}

func AddVlanIfUpdate(vlanConfig netManage.VlanConfig, id string) error {
	newPhysicalInterface := combineAndSortStrings(vlanConfig.PhysicalInterface, id)
	err := global.NETVINE_DB.Model(&vlanConfig).Update("physical_interface", newPhysicalInterface).Error
	if err != nil {
		global.NETVINE_LOG.Error("AddVlanIfUpdate", zap.Error(err))
		return errors.New("更新vlanIf失败")
	}
	return nil
}

func DelVlanIf(physicalInterface string, tx *gorm.DB) error {
	if tx == nil {
		tx = global.NETVINE_DB
	}

	var oldConfig netManage.InterfaceConfig
	global.NETVINE_DB.Model(&[]netManage.InterfaceConfig{}).Where("out_interface = ?", physicalInterface).Find(&oldConfig)
	var arr []string
	if len(oldConfig.VlanId) != 0 {
		arr = append(arr, oldConfig.VlanId)
	} else {
		arr, _ = splitAndAddList(oldConfig.TrunkVlanId, oldConfig.DefaultId)
	}

	var vlanList []string
	for _, v := range arr {
		vlanList = append(vlanList, "vlan_"+v)
	}

	for _, v := range vlanList {
		var vlanConfig netManage.VlanConfig
		tx.Model(&netManage.VlanConfig{}).Where("out_interface = ?", v).Find(&vlanConfig)
		if vlanConfig.PhysicalInterface == physicalInterface {
			//说明此vlan的物理接口只有一个，删
			err := tx.Unscoped().Delete(&vlanConfig).Error
			if err != nil {
				return err
			}
			delVlanCmd := fmt.Sprintf(Cmd_Vlan_If_Delete, v)
			err = command.GoLinuxShell(delVlanCmd)
			if err != nil {
				return err
			}
			//清理ip
			err = tx.Unscoped().Delete(&[]netManage.VlanConfigArray{}, "out_interface = ?", v).Error
			if err != nil {
				global.NETVINE_LOG.Error("清空vlanIp,"+v, zap.Error(err))
				return err
			}
		} else if len(vlanConfig.PhysicalInterface) > 0 {
			//改
			newInterface := removeStringFromSlice(vlanConfig.PhysicalInterface, physicalInterface)
			err := tx.Model(&vlanConfig).Update("physical_interface", newInterface).Error
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func combineAndSortStrings(A, B string) string {
	// 将A和B按逗号分隔拆分为字符串切片
	AList := strings.Split(A, ",")
	BList := strings.Split(B, ",")

	// 将B中不存在于A中的元素添加到A中
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

	// 对切片进行排序
	sort.Strings(AList)

	// 将切片元素拼接为字符串
	result := strings.Join(AList, ",")

	return result
}

func splitAndAddList(trunkVlanId, defaultID string) ([]string, error) {
	var result []string

	ranges := strings.Split(trunkVlanId, ",")
	for _, r := range ranges {
		if strings.Contains(r, "-") {
			nums := strings.Split(r, "-")
			if len(nums) != 2 {
				return nil, errors.New("Trunkd的VlanId不合法")
			}

			start, err := strconv.Atoi(nums[0])
			if err != nil {
				return nil, err
			}

			end, err := strconv.Atoi(nums[1])
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

func removeStringFromSlice(sliceString string, strToRemove string) string {
	var result []string

	slice := strings.Split(sliceString, ",")
	for _, str := range slice {
		if str != strToRemove {
			result = append(result, str)
		}
	}
	// 将结果拼接为逗号分隔的字符串
	return strings.Join(result, ",")
}

func VlanIfIpAdd(ipList []string, vlanInterface string) error {
	for _, v := range ipList {
		cmd := fmt.Sprintf(Cmd_Vlan_If_Ip, v, vlanInterface)
		err := command.GoLinuxShell(cmd)
		if err != nil {
			global.NETVINE_LOG.Error("VlanIfIpAddAndDel设置失败,"+v+vlanInterface, zap.Error(err))
			return firewall_error.CreateError("Vlan Ip设置失败!")
		}
	}
	return nil
}

func VlanIfIpFlush(vlanInterface string) error {
	cmd := fmt.Sprintf(Cmd_Vlan_If_Ip_Flush, vlanInterface)
	err := command.GoLinuxShell(cmd)
	if err != nil {
		global.NETVINE_LOG.Error("VlanIfIpFlush失败,"+vlanInterface, zap.Error(err))
		return firewall_error.CreateError("清空Vlan Ip设置失败!")
	}
	err = global.NETVINE_DB.Unscoped().Delete(&[]netManage.VlanConfigArray{}, "out_interface = ?", vlanInterface).Error
	if err != nil {
		global.NETVINE_LOG.Error("VlanIfIpFlush失败2,"+vlanInterface, zap.Error(err))
		return firewall_error.CreateError("清空VlanIp设置失败!")
	}
	return nil
}

func VlanIfUpAndDown(vlanInterface, action string) error {
	cmd := fmt.Sprintf(Cmd_Vlan_If_Up_Down, vlanInterface, action)
	err := command.GoLinuxShell(cmd)
	if err != nil {
		global.NETVINE_LOG.Error("VlanIfUpAndDown失败,"+vlanInterface+action, zap.Error(err))
		return firewall_error.CreateError("更改网卡状态失败!")
	}
	return nil
}

func InterfaceIpFlush(interfaceName string) error {
	cmd := fmt.Sprintf(Cmd_Vlan_If_Ip_Flush, interfaceName)
	err := command.GoLinuxShell(cmd)
	if err != nil {
		global.NETVINE_LOG.Error("接口清除ip失败,"+interfaceName, zap.Error(err))
		return firewall_error.CreateError("清空Vlan Ip设置失败!")
	}
	return nil
}
