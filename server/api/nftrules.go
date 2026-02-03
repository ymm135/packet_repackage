package api

import (
	"net/http"
	"packet-repackage/database"
	"packet-repackage/models"
	"packet-repackage/network"

	"github.com/gin-gonic/gin"
)

// ListNFTRules returns all NFT rules
func ListNFTRules(c *gin.Context) {
	var rules []models.NFTRule
	result := database.DB.Order("priority ASC, id ASC").Find(&rules)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	// Add summary for each rule
	type RuleWithSummary struct {
		models.NFTRule
		Summary string `json:"summary"`
	}

	var rulesWithSummary []RuleWithSummary
	for _, rule := range rules {
		rulesWithSummary = append(rulesWithSummary, RuleWithSummary{
			NFTRule: rule,
			Summary: network.GetRuleSummary(rule),
		})
	}

	c.JSON(http.StatusOK, gin.H{"data": rulesWithSummary})
}

// GetNFTRule returns a single NFT rule
func GetNFTRule(c *gin.Context) {
	id := c.Param("id")
	var rule models.NFTRule
	
	result := database.DB.First(&rule, id)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": rule})
}

// CreateNFTRule creates a new NFT rule
func CreateNFTRule(c *gin.Context) {
	var rule models.NFTRule
	
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate action
	if rule.Action != "accept" && rule.Action != "drop" && rule.Action != "queue" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Action must be accept, drop, or queue"})
		return
	}

	// Create in database
	if err := database.DB.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create rule: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "Rule created successfully",
		"data":    rule,
	})
}

// UpdateNFTRule updates an existing NFT rule
func UpdateNFTRule(c *gin.Context) {
	id := c.Param("id")
	
	var rule models.NFTRule
	result := database.DB.First(&rule, id)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
		return
	}

	// Parse update data
	var updateData models.NFTRule
	if err := c.ShouldBindJSON(&updateData); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate action
	if updateData.Action != "accept" && updateData.Action != "drop" && updateData.Action != "queue" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Action must be accept, drop, or queue"})
		return
	}

	// Update fields
	rule.Name = updateData.Name
	rule.Enabled = updateData.Enabled
	rule.Priority = updateData.Priority
	rule.SrcIP = updateData.SrcIP
	rule.DstIP = updateData.DstIP
	rule.SrcPort = updateData.SrcPort
	rule.DstPort = updateData.DstPort
	rule.Protocol = updateData.Protocol
	rule.LogEnabled = updateData.LogEnabled
	rule.LogPrefix = updateData.LogPrefix
	rule.Action = updateData.Action
	rule.QueueNum = updateData.QueueNum

	if err := database.DB.Save(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update rule: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Rule updated successfully",
		"data":    rule,
	})
}

// DeleteNFTRule deletes an NFT rule
func DeleteNFTRule(c *gin.Context) {
	id := c.Param("id")
	
	result := database.DB.Delete(&models.NFTRule{}, id)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rule deleted successfully"})
}

// ToggleNFTRule toggles a rule's enabled status
func ToggleNFTRule(c *gin.Context) {
	id := c.Param("id")
	
	var rule models.NFTRule
	result := database.DB.First(&rule, id)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
		return
	}

	// Toggle enabled status
	rule.Enabled = !rule.Enabled
	
	if err := database.DB.Save(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to toggle rule: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Rule toggled successfully",
		"data":    rule,
	})
}

// ApplyNFTRules applies all enabled rules to nftables
func ApplyNFTRulesAPI(c *gin.Context) {
	err := network.ApplyNFTRules(database.DB)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to apply rules: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rules applied successfully"})
}
