package nfqueue

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"packet-repackage/database"
	"packet-repackage/engine"
	"packet-repackage/models"
	"strings"
	"time"

	"sync"

	"github.com/florianl/go-nfqueue"
	"go.uber.org/zap"
)

type NFQueueManager struct {
	queues map[uint16]*nfqueue.Nfqueue
	ctx    context.Context
	cancel context.CancelFunc
}

var Manager *NFQueueManager

type configCache struct {
	sync.RWMutex
	fields []models.Field
	rules  []models.Rule
}

var cache = &configCache{}

// ReloadConfig loads fields and enabled rules from the database into memory
func ReloadConfig() error {
	var fields []models.Field
	if err := database.DB.Find(&fields).Error; err != nil {
		return fmt.Errorf("failed to load fields: %w", err)
	}

	var rules []models.Rule
	if err := database.DB.Where("enabled = ?", true).Order("priority DESC").Find(&rules).Error; err != nil {
		return fmt.Errorf("failed to load rules: %w", err)
	}

	cache.Lock()
	cache.fields = fields
	cache.rules = rules
	cache.Unlock()

	database.Logger.Info("Configuration reloaded",
		zap.Int("fields_count", len(fields)),
		zap.Int("rules_count", len(rules)))

	return nil
}

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
			AfFamily:     2, // AF_INET - more reliable than AF_BRIDGE which can have nil PacketID issues
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
	// Critical: In AF_BRIDGE mode, PacketID can sometimes be nil even when Payload is present
	// This appears to be a known issue with the nfqueue library when using bridge netfilter
	// Without a PacketID, we cannot issue a verdict, so we must skip processing
	if attr.PacketID == nil {
		if attr.Payload != nil {
			database.Logger.Warn("Received packet with nil PacketID but valid payload - AF_BRIDGE mode issue",
				zap.Int("payload_len", len(*attr.Payload)),
				zap.Any("indev", attr.InDev),
				zap.Any("outdev", attr.OutDev))
		}
		// Cannot process without PacketID - return and let kernel handle it
		return 0
	}

	if attr.Payload == nil {
		database.Logger.Warn("Received packet with nil payload", zap.Uint32("packet_id", *attr.PacketID))
		nfq.SetVerdict(*attr.PacketID, nfqueue.NfAccept)
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

	lines := strings.Split(strings.TrimSpace(engine.HexDump(rawPacket)), "\n")
	for _, line := range lines {
		database.Logger.Debug("Packet HexDump",
			zap.Uint32("packet_id", packetID),
			zap.String("line", line))
	}

	// Get configurations from cache
	cache.RLock()
	fields := cache.fields
	rules := cache.rules
	cache.RUnlock()

	// Extract field values
	engine.ExtractAllFields(ctx, fields)

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

	// Populate 5-tuple info
	if ctx.IPv4Layer != nil {
		logEntry.SrcIP = ctx.IPv4Layer.SrcIP.String()
		logEntry.DstIP = ctx.IPv4Layer.DstIP.String()
		logEntry.Protocol = ctx.IPv4Layer.Protocol.String()
	}
	if ctx.TCPLayer != nil {
		logEntry.SrcPort = int(ctx.TCPLayer.SrcPort)
		logEntry.DstPort = int(ctx.TCPLayer.DstPort)
		logEntry.Protocol = "TCP"
	} else if ctx.UDPLayer != nil {
		logEntry.SrcPort = int(ctx.UDPLayer.SrcPort)
		logEntry.DstPort = int(ctx.UDPLayer.DstPort)
		logEntry.Protocol = "UDP"
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
		modifiedPacket, err = engine.RepackagePacket(matchedRule.OutputOptions, ctx, fields)
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
