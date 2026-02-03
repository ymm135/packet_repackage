package api

import (
	"net/http"
	"packet-repackage/database"
	"packet-repackage/models"
	"packet-repackage/network"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// ListInterfaces returns all available network interfaces
func ListInterfaces(c *gin.Context) {
	interfaces, err := network.ListInterfaces()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": interfaces})
}

// GetInterface returns info about a specific interface
func GetInterface(c *gin.Context) {
	name := c.Param("name")
	iface, err := network.GetInterfaceByName(name)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": iface})
}

// ConfigureVLAN configures VLAN on an interface
func ConfigureVLAN(c *gin.Context) {
	var req struct {
		Interface   string `json:"interface" binding:"required"`
		LinkType    string `json:"link_type" binding:"required"`
		VlanId      string `json:"vlan_id"`
		TrunkVlanId string `json:"trunk_vlan_id"`
		DefaultId   string `json:"default_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Handle VLAN ID = 0 as deletion
	if req.LinkType == "access" && req.VlanId == "0" {
		err := network.RemoveVlan(req.Interface, database.DB)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{"message": "VLAN removed successfully"})
		return
	}

	// Create interface config
	ifaceConfig := models.InterfaceConfig{
		OutInterface: req.Interface,
		LinkType:     req.LinkType,
		VlanId:       req.VlanId,
		TrunkVlanId:  req.TrunkVlanId,
		DefaultId:    req.DefaultId,
	}
	
	// Pass database to AddVlan for automatic VLAN interface creation
	err := network.AddVlan(ifaceConfig, req.Interface, database.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Save configuration to database for persistence
	// Check if record exists (including soft-deleted ones)
	var existingConfig models.InterfaceConfig
	result := database.DB.Unscoped().Where("out_interface = ?", req.Interface).First(&existingConfig)
	
	if result.Error == nil {
		// Record exists (possibly soft-deleted), update it
		existingConfig.LinkType = req.LinkType
		existingConfig.VlanId = req.VlanId
		existingConfig.TrunkVlanId = req.TrunkVlanId
		existingConfig.DefaultId = req.DefaultId
		existingConfig.DeletedAt = gorm.DeletedAt{} // Restore if soft-deleted
		if err := database.DB.Unscoped().Save(&existingConfig).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update config: " + err.Error()})
			return
		}
	} else {
		// No record found, create new
		if err := database.DB.Create(&ifaceConfig).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create config: " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "VLAN configured successfully"})
}

// GetVLANConfig returns current VLAN configuration for an interface
func GetVLANConfig(c *gin.Context) {
	interfaceName := c.Param("interface")
	
	var config models.InterfaceConfig
	result := database.DB.Where("out_interface = ?", interfaceName).First(&config)
	
	if result.Error != nil {
		// No configuration found, return default (vlan_id = 0)
		c.JSON(http.StatusOK, gin.H{
			"data": gin.H{
				"interface":     interfaceName,
				"link_type":     "access",
				"vlan_id":       "0",
				"trunk_vlan_id": "",
				"default_id":    "1",
			},
		})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"interface":     config.OutInterface,
			"link_type":     config.LinkType,
			"vlan_id":       config.VlanId,
			"trunk_vlan_id": config.TrunkVlanId,
			"default_id":    config.DefaultId,
		},
	})
}

// AddVLANIP adds IP address to VLAN interface
func AddVLANIP(c *gin.Context) {
	var req struct {
		VlanInterface string   `json:"vlan_interface" binding:"required"`
		IpAddresses   []string `json:"ip_addresses" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := network.VlanIfIpAdd(req.IpAddresses, req.VlanInterface)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "IP addresses added successfully"})
}

// RemoveVLANIP removes all IP addresses from VLAN interface
func RemoveVLANIP(c *gin.Context) {
	vlanInterface := c.Param("interface")
	
	err := network.VlanIfIpFlush(vlanInterface)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "IP addresses removed successfully"})
}

// SetInterfaceStatus sets interface up or down
func SetInterfaceStatus(c *gin.Context) {
	var req struct {
		Interface string `json:"interface" binding:"required"`
		Status    string `json:"status" binding:"required"` // "up" or "down"
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := network.VlanIfUpAndDown(req.Interface, req.Status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Interface status updated"})
}
