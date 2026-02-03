package api

import (
	"net/http"
	"packet-repackage/database"
	"packet-repackage/models"
	"strconv"

	"github.com/gin-gonic/gin"
)

// ListLogs returns paginated processing logs
func ListLogs(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	
	ruleID := c.Query("rule_id")
	result := c.Query("result")

	offset := (page - 1) * pageSize
	
	var logs []models.ProcessLog
	var total int64
	
	query := database.DB.Model(&models.ProcessLog{})
	
	// Apply filters
	if ruleID != "" {
		query = query.Where("rule_id = ?", ruleID)
	}
	if result != "" {
		query = query.Where("result = ?", result)
	}
	
	// Get total count
	query.Count(&total)
	
	// Get paginated results
	query.Order("processed_at DESC").
		Limit(pageSize).
		Offset(offset).
		Find(&logs)
	
	c.JSON(http.StatusOK, gin.H{
		"data": logs,
		"pagination": gin.H{
			"page":       page,
			"page_size":  pageSize,
			"total":      total,
			"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
		},
	})
}

// GetLog returns a specific log entry with full details
func GetLog(c *gin.Context) {
	id := c.Param("id")
	var log models.ProcessLog
	
	if err := database.DB.First(&log, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Log not found"})
		return
	}
	
	c.JSON(http.StatusOK, gin.H{"data": log})
}

// ClearLogs deletes all logs or logs older than specified time
func ClearLogs(c *gin.Context) {
	days := c.Query("days")
	
	if days != "" {
		daysInt, err := strconv.Atoi(days)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid days parameter"})
			return
		}
		
		// Delete logs older than X days
		database.DB.Where("processed_at < datetime('now', '-' || ? || ' days')", daysInt).
			Delete(&models.ProcessLog{})
	} else {
		// Delete all logs
		database.DB.Unscoped().Delete(&models.ProcessLog{}, "1=1")
	}
	
	c.JSON(http.StatusOK, gin.H{"message": "Logs cleared successfully"})
}
