package api

import (
	"encoding/hex"
	"net/http"
	"packet-repackage/database"
	"packet-repackage/engine"
	"packet-repackage/models"

	"github.com/gin-gonic/gin"
)

// TestRequest represents a test mode request
type TestRequest struct {
	HexPacket string `json:"hex_packet" binding:"required"`
	RuleID    uint   `json:"rule_id"`
}

// TestResponse represents a test mode response
type TestResponse struct {
	OriginalPacket  string                 `json:"original_packet"`
	ParsedFields    map[string]string      `json:"parsed_fields"`
	MatchedRule     *models.Rule           `json:"matched_rule"`
	ModifiedFields  map[string]interface{} `json:"modified_fields"`
	ModifiedPacket  string                 `json:"modified_packet"`
	ProcessingSteps []string               `json:"processing_steps"`
	Error           string                 `json:"error,omitempty"`

	// 5-Tuple info
	SrcIP    string `json:"src_ip"`
	DstIP    string `json:"dst_ip"`
	SrcPort  int    `json:"src_port"`
	DstPort  int    `json:"dst_port"`
	Protocol string `json:"protocol"`
}

// TestRule tests a rule against a hex packet input
func TestRule(c *gin.Context) {
	var req TestRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Decode hex packet
	rawPacket, err := hex.DecodeString(req.HexPacket)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid hex packet: " + err.Error()})
		return
	}

	response := TestResponse{
		OriginalPacket:  req.HexPacket,
		ParsedFields:    make(map[string]string),
		ModifiedFields:  make(map[string]interface{}),
		ProcessingSteps: []string{},
	}

	// Parse packet
	ctx, err := engine.ParsePacket(rawPacket)
	if err != nil {
		response.Error = "Failed to parse packet: " + err.Error()
		c.JSON(http.StatusOK, response)
		return
	}
	response.ProcessingSteps = append(response.ProcessingSteps, "Packet parsed successfully")

	// Populate 5-tuple info
	if ctx.IPv4Layer != nil {
		response.SrcIP = ctx.IPv4Layer.SrcIP.String()
		response.DstIP = ctx.IPv4Layer.DstIP.String()
		response.Protocol = ctx.IPv4Layer.Protocol.String()
	}
	if ctx.TCPLayer != nil {
		response.SrcPort = int(ctx.TCPLayer.SrcPort)
		response.DstPort = int(ctx.TCPLayer.DstPort)
		response.Protocol = "TCP"
	} else if ctx.UDPLayer != nil {
		response.SrcPort = int(ctx.UDPLayer.SrcPort)
		response.DstPort = int(ctx.UDPLayer.DstPort)
		response.Protocol = "UDP"
	}

	// Get all fields
	var fields []models.Field
	database.DB.Find(&fields)

	// Extract field values
	err = engine.ExtractAllFields(ctx, fields)
	if err != nil {
		response.Error = "Failed to extract fields: " + err.Error()
		c.JSON(http.StatusOK, response)
		return
	}

	// Format fields for display
	for _, field := range fields {
		if ctx.Fields[field.Name] != nil {
			response.ParsedFields[field.Name] = engine.FormatFieldValue(ctx.Fields[field.Name], field.Type)
		}
	}
	response.ProcessingSteps = append(response.ProcessingSteps, "Extracted fields from packet")

	// Get rule to test
	var rule models.Rule
	if req.RuleID > 0 {
		// Test specific rule
		if err := database.DB.First(&rule, req.RuleID).Error; err != nil {
			response.Error = "Rule not found"
			c.JSON(http.StatusNotFound, response)
			return
		}
	} else {
		// Try to match any enabled rule
		var rules []models.Rule
		database.DB.Where("enabled = ?", true).Order("priority DESC").Find(&rules)

		for _, r := range rules {
			matched, err := engine.EvaluateCondition(r.MatchCondition, ctx, fields)
			if err != nil {
				continue
			}
			if matched {
				rule = r
				break
			}
		}

		if rule.ID == 0 {
			response.ProcessingSteps = append(response.ProcessingSteps, "No matching rule found")
			c.JSON(http.StatusOK, response)
			return
		}
	}

	response.MatchedRule = &rule
	response.ProcessingSteps = append(response.ProcessingSteps, "Matched rule: "+rule.Name)

	// Evaluate condition
	matched, err := engine.EvaluateCondition(rule.MatchCondition, ctx, fields)
	if err != nil {
		response.Error = "Failed to evaluate condition: " + err.Error()
		c.JSON(http.StatusOK, response)
		return
	}

	if !matched {
		response.ProcessingSteps = append(response.ProcessingSteps, "Rule condition not matched")
		c.JSON(http.StatusOK, response)
		return
	}
	response.ProcessingSteps = append(response.ProcessingSteps, "Rule condition matched")

	// Store original values
	originalFields := make(map[string]interface{})
	for k, v := range ctx.Fields {
		originalFields[k] = v
	}

	// Execute actions
	err = engine.ExecuteActions(rule.Actions, ctx)
	if err != nil {
		response.Error = "Failed to execute actions: " + err.Error()
		c.JSON(http.StatusOK, response)
		return
	}
	response.ProcessingSteps = append(response.ProcessingSteps, "Executed rule actions")

	// Build modified fields comparison
	for k, v := range ctx.Fields {
		if originalFields[k] != v {
			response.ModifiedFields[k] = map[string]interface{}{
				"before": originalFields[k],
				"after":  v,
			}
		}
	}

	// Repackage packet
	modifiedPacket, err := engine.RepackagePacket(rule.OutputOptions, ctx, fields)
	if err != nil {
		response.Error = "Failed to repackage packet: " + err.Error()
		c.JSON(http.StatusOK, response)
		return
	}
	response.ModifiedPacket = hex.EncodeToString(modifiedPacket)
	response.ProcessingSteps = append(response.ProcessingSteps, "Packet repackaged successfully")

	c.JSON(http.StatusOK, response)
}
