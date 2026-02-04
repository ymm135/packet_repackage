package api

import (
	"net/http"
	"packet-repackage/database"
	"packet-repackage/models"

	"github.com/gin-gonic/gin"
)

// ListRules returns all rules
func ListRules(c *gin.Context) {
	var rules []models.Rule
	database.DB.Order("priority DESC").Find(&rules)
	c.JSON(http.StatusOK, gin.H{"data": rules})
}

// GetRule returns a specific rule
func GetRule(c *gin.Context) {
	id := c.Param("id")
	var rule models.Rule

	if err := database.DB.First(&rule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": rule})
}

// CreateRule creates a new rule
func CreateRule(c *gin.Context) {
	var rule models.Rule

	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Create(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": rule})
}

// UpdateRule updates an existing rule
func UpdateRule(c *gin.Context) {
	id := c.Param("id")
	var rule models.Rule

	if err := database.DB.First(&rule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
		return
	}

	var updates models.Rule
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	rule.Name = updates.Name
	rule.Enabled = updates.Enabled
	rule.MatchCondition = updates.MatchCondition
	rule.Actions = updates.Actions
	rule.OutputOptions = updates.OutputOptions
	rule.Priority = updates.Priority

	if err := database.DB.Save(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": rule})
}

// DeleteRule deletes a rule
func DeleteRule(c *gin.Context) {
	id := c.Param("id")

	if err := database.DB.Delete(&models.Rule{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rule deleted successfully"})
}

// ToggleRule enables or disables a rule
func ToggleRule(c *gin.Context) {
	id := c.Param("id")
	var rule models.Rule

	if err := database.DB.First(&rule, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rule not found"})
		return
	}

	rule.Enabled = !rule.Enabled

	if err := database.DB.Save(&rule).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": rule})
}
