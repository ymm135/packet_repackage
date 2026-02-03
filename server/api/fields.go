package api

import (
	"net/http"
	"packet-repackage/database"
	"packet-repackage/models"

	"github.com/gin-gonic/gin"
)

// ListFields returns all field definitions
func ListFields(c *gin.Context) {
	var fields []models.Field
	database.DB.Find(&fields)
	c.JSON(http.StatusOK, gin.H{"data": fields})
}

// GetField returns a specific field
func GetField(c *gin.Context) {
	id := c.Param("id")
	var field models.Field
	
	if err := database.DB.First(&field, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Field not found"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"data": field})
}

// CreateField creates a new field definition
func CreateField(c *gin.Context) {
	var field models.Field
	
	if err := c.ShouldBindJSON(&field); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := database.DB.Create(&field).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": field})
}

// UpdateField updates an existing field
func UpdateField(c *gin.Context) {
	id := c.Param("id")
	var field models.Field
	
	if err := database.DB.First(&field, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Field not found"})
		return
	}

	var updates models.Field
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update fields
	field.Name = updates.Name
	field.Offset = updates.Offset
	field.Length = updates.Length
	field.Type = updates.Type

	if err := database.DB.Save(&field).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": field})
}

// DeleteField deletes a field
func DeleteField(c *gin.Context) {
	id := c.Param("id")
	
	if err := database.DB.Delete(&models.Field{}, id).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Field deleted successfully"})
}
