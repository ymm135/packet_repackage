package nfqueue

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"packet-repackage/database"
	"packet-repackage/engine"
	"packet-repackage/models"
	"time"

	nfqueue "github.com/florianl/go-nfqueue"
	"go.uber.org/zap"
)

type NFQueueManager struct {
	queues map[uint16]*nfqueue.Nfqueue
	ctx    context.Context
	cancel context.CancelFunc
}

var Manager *NFQueueManager

// Start initializes and starts the NFQueue packet processing
func Start(queueNums []uint16) error {
	ctx, cancel := context.WithCancel(context.Background())

	manager := &NFQueueManager{
		queues: make(map[uint16]*nfqueue.Nfqueue),
		ctx:    ctx,
		cancel: cancel,
	}

	for _, qNum := range queueNums {
		config := nfqueue.Config{
			NfQueue:      qNum,
			MaxPacketLen: 0xFFFF,
			MaxQueueLen:  0xFF,
			Copymode:     nfqueue.NfQnlCopyPacket,
			WriteTimeout: 100 * time.Millisecond,
			AfFamily:     7, // AF_BRIDGE (patched library now supports this)
		}

		nfq, err := nfqueue.Open(&config)
		if err != nil {
			cancel()
			return fmt.Errorf("failed to open nfqueue %d: %w", qNum, err)
		}

		// Register packet callback
		// We capture the nfq instance in a closure so handlePacket knows which queue to use
		// This avoids relying on attr.QueueId which might not be available
		currentNFQ := nfq
		err = nfq.RegisterWithErrorFunc(ctx, func(attr nfqueue.Attribute) int {
			return handlePacket(attr, currentNFQ)
		}, handleError)
		if err != nil {
			nfq.Close()
			cancel()
			return fmt.Errorf("failed to register callback for queue %d: %w", qNum, err)
		}

		manager.queues[qNum] = nfq
		database.Logger.Info("NFQueue started", zap.Uint16("queue", qNum))
	}

	Manager = manager
	return nil
}

// Stop closes the NFQueue connection
func Stop() error {
	if Manager == nil {
		return nil
	}

	Manager.cancel()

	var lastErr error
	for qNum, nfq := range Manager.queues {
		err := nfq.Close()
		if err != nil {
			database.Logger.Error("Failed to close queue",
				zap.Uint16("queue", qNum),
				zap.Error(err))
			lastErr = err
		}
	}

	Manager = nil
	database.Logger.Info("NFQueue stopped")
	return lastErr
}

// handlePacket processes each packet from the queue
func handlePacket(attr nfqueue.Attribute, nfq *nfqueue.Nfqueue) int {
	if attr.PacketID == nil || attr.Payload == nil {
		if attr.PacketID != nil {
			database.Logger.Warn("Received packet with nil payload", zap.Uint32("packet_id", *attr.PacketID))
			nfq.SetVerdict(*attr.PacketID, nfqueue.NfAccept)
		}
		return 0
	}

	packetID := *attr.PacketID
	rawPacket := *attr.Payload

	// Default verdict is ACCEPT (pass through)
	verdict := nfqueue.NfAccept

	// Parse packet
	ctx, err := engine.ParsePacket(rawPacket)
	if err != nil {
		database.Logger.Error("Failed to parse packet", zap.Error(err))
		nfq.SetVerdict(packetID, verdict)
		return 0
	}

	database.Logger.Debug("Packet received",
		zap.Uint32("packet_id", packetID),
		zap.String("5-tuple", ctx.Get5Tuple()))

	database.Logger.Debug("Packet HexDump",
		zap.Uint32("packet_id", packetID),
		zap.String("dump", engine.HexDump(rawPacket)))

	// Get all fields
	var fields []models.Field
	database.DB.Find(&fields)

	// Extract field values
	engine.ExtractAllFields(ctx, fields)

	// Get enabled rules ordered by priority
	var rules []models.Rule
	database.DB.Where("enabled = ?", true).Order("priority DESC").Find(&rules)

	// Try to match rules
	var matchedRule *models.Rule
	for _, rule := range rules {
		matched, err := engine.EvaluateCondition(rule.MatchCondition, ctx, fields)
		if err != nil {
			database.Logger.Error("Failed to evaluate condition",
				zap.String("rule", rule.Name),
				zap.Error(err))
			continue
		}

		if matched {
			matchedRule = &rule
			break
		}
	}

	// Process matched rule
	var modifiedPacket []byte
	logEntry := models.ProcessLog{
		ProcessedAt:    time.Now(),
		OriginalPacket: hex.EncodeToString(rawPacket),
	}

	if matchedRule != nil {
		logEntry.RuleID = matchedRule.ID
		logEntry.RuleName = matchedRule.Name

		// Store original field values
		originalFields := make(map[string]interface{})
		for k, v := range ctx.Fields {
			originalFields[k] = v
		}

		// Execute actions
		err = engine.ExecuteActions(matchedRule.Actions, ctx)
		if err != nil {
			database.Logger.Error("Failed to execute actions",
				zap.String("rule", matchedRule.Name),
				zap.Error(err))
			logEntry.Result = "error"
			logEntry.ErrorMessage = err.Error()
			database.DB.Create(&logEntry)
			nfq.SetVerdict(packetID, verdict)
			return 0
		}

		// Repackage packet
		modifiedPacket, err = engine.RepackagePacket(matchedRule.OutputTemplate, ctx, fields)
		if err != nil {
			database.Logger.Error("Failed to repackage packet",
				zap.String("rule", matchedRule.Name),
				zap.Error(err))
			logEntry.Result = "error"
			logEntry.ErrorMessage = err.Error()
			database.DB.Create(&logEntry)
			nfq.SetVerdict(packetID, verdict)
			return 0
		}

		// Build field values comparison
		fieldComparison := make(map[string]map[string]interface{})
		for k, v := range ctx.Fields {
			fieldComparison[k] = map[string]interface{}{
				"before": originalFields[k],
				"after":  v,
			}
		}
		fieldValuesJSON, _ := json.Marshal(fieldComparison)
		logEntry.FieldValues = string(fieldValuesJSON)
		logEntry.ModifiedPacket = hex.EncodeToString(modifiedPacket)
		logEntry.Result = "success"

		// For modified packets, we need to set the verdict with the new packet data
		err = nfq.SetVerdictModPacket(packetID, verdict, modifiedPacket)
		if err != nil {
			database.Logger.Error("Failed to set verdict with modified packet",
				zap.Uint32("packet_id", packetID),
				zap.Error(err))
			// Fallback to accepting original if modification fails
			nfq.SetVerdict(packetID, verdict)
		} else {
			database.Logger.Info("Packet modified and sent",
				zap.String("rule", matchedRule.Name),
				zap.Int("original_size", len(rawPacket)),
				zap.Int("modified_size", len(modifiedPacket)))
		}

		// Log the processing
		database.DB.Create(&logEntry)

		return 0
	}

	// No rule matched, pass through unchanged
	nfq.SetVerdict(packetID, verdict)
	return 0
}

func handleError(err error) int {
	database.Logger.Error("NFQueue error", zap.Error(err))
	return 0
}
