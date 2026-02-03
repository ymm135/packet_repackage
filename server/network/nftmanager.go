package network

import (
	"fmt"
	"packet-repackage/database"
	"packet-repackage/models"
	"packet-repackage/utils/command"
	"strings"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// ApplyNFTRules applies all enabled NFT rules from database to nftables
func ApplyNFTRules(db *gorm.DB) error {
	database.Logger.Info("Applying NFTables rules from database")

	// Ensure table and chain exist first
	err := EnsureNFTInfrastructure()
	if err != nil {
		return fmt.Errorf("failed to ensure NFT infrastructure: %w", err)
	}

	// Clear existing rules in base-rule-chain
	err = ClearNFTRules()
	if err != nil {
		return fmt.Errorf("failed to clear existing rules: %w", err)
	}

	// Load all enabled rules from database
	var rules []models.NFTRule
	result := db.Where("enabled = ?", true).Order("priority ASC, id ASC").Find(&rules)
	if result.Error != nil {
		return fmt.Errorf("failed to load rules: %w", result.Error)
	}

	database.Logger.Info("Found enabled rules to apply", zap.Int("count", len(rules)))

	// Apply each rule
	successCount := 0
	for _, rule := range rules {
		cmd := BuildNFTCommand(rule)
		database.Logger.Info("Applying rule", 
			zap.String("name", rule.Name),
			zap.String("command", cmd))

		err := command.GoLinuxShell(cmd)
		if err != nil {
			database.Logger.Error("Failed to apply rule",
				zap.String("name", rule.Name),
				zap.Error(err))
			continue
		}
		successCount++
	}

	database.Logger.Info("NFTables rules applied",
		zap.Int("success", successCount),
		zap.Int("failed", len(rules)-successCount))

	return nil
}

// EnsureNFTInfrastructure creates the nftables table and chain if they don't exist
func EnsureNFTInfrastructure() error {
	database.Logger.Info("Ensuring NFTables infrastructure exists")

	// Try to create table (ignore error if exists)
	// Remove old IP table if exists to avoid confusion
	_ = command.GoLinuxShell("nft delete table ip netvine-table")

	// Try to create table (ignore error if exists)
	cmd := "nft add table bridge netvine-table"
	_ = command.GoLinuxShell(cmd) // Ignore error, table might exist

	// Try to create chain (ignore error if exists)
	cmd = "nft add chain bridge netvine-table base-rule-chain { type filter hook forward priority 0\\; policy accept\\; }"
	_ = command.GoLinuxShell(cmd) // Ignore error, chain might exist

	database.Logger.Info("NFTables infrastructure ready")
	return nil
}

// ClearNFTRules removes all rules from base-rule-chain
func ClearNFTRules() error {
	database.Logger.Info("Clearing existing NFTables rules")

	// Flush all rules in base-rule-chain
	cmd := "nft flush chain bridge netvine-table base-rule-chain"
	err := command.GoLinuxShell(cmd)
	if err != nil {
		return fmt.Errorf("failed to flush chain: %w", err)
	}

	return nil
}

// BuildNFTCommand builds an nft command from a rule
func BuildNFTCommand(rule models.NFTRule) string {
	var parts []string

	// Start with base command
	parts = append(parts, "nft add rule bridge netvine-table base-rule-chain")

	// Add IP filters FIRST (before protocol)
	if rule.SrcIP != "" {
		parts = append(parts, fmt.Sprintf("ip saddr %s", rule.SrcIP))
	}

	if rule.DstIP != "" {
		parts = append(parts, fmt.Sprintf("ip daddr %s", rule.DstIP))
	}

	// Then add protocol and ports
	if rule.Protocol != "" && rule.Protocol != "any" {
		// For TCP/UDP, add protocol-specific port filters
		if rule.Protocol == "tcp" || rule.Protocol == "udp" {
			if rule.SrcPort != "" {
				parts = append(parts, fmt.Sprintf("%s sport %s", rule.Protocol, rule.SrcPort))
			}
			if rule.DstPort != "" {
				parts = append(parts, fmt.Sprintf("%s dport %s", rule.Protocol, rule.DstPort))
			}
			// If no ports specified, just add protocol
			if rule.SrcPort == "" && rule.DstPort == "" {
				parts = append(parts, rule.Protocol)
			}
		} else {
			// For ICMP or other protocols, just add protocol
			parts = append(parts, rule.Protocol)
		}
	}

	// Add logging if enabled
	if rule.LogEnabled {
		prefix := rule.LogPrefix
		if prefix == "" {
			prefix = rule.Name
		}
		// NFTables log prefix should not have quotes or special chars in the command
		// The prefix itself will be shown in logs
		parts = append(parts, fmt.Sprintf("log prefix \"%s\"", prefix))
	}

	// Add action
	switch rule.Action {
	case "accept":
		parts = append(parts, "accept")
	case "drop":
		parts = append(parts, "drop")
	case "queue":
		if rule.QueueNum != "" {
			// Support range like "0-3" or single number like "2"
			parts = append(parts, fmt.Sprintf("queue num %s bypass", rule.QueueNum))
		} else {
			// Default to 0-3 if not specified
			parts = append(parts, "queue num 0-3 bypass")
		}
	}

	return strings.Join(parts, " ")
}

// GetRuleSummary returns a human-readable summary of the rule's filters
func GetRuleSummary(rule models.NFTRule) string {
	srcIP := rule.SrcIP
	if srcIP == "" {
		srcIP = "any"
	}

	dstIP := rule.DstIP
	if dstIP == "" {
		dstIP = "any"
	}

	srcPort := rule.SrcPort
	if srcPort == "" {
		srcPort = "*"
	}

	dstPort := rule.DstPort
	if dstPort == "" {
		dstPort = "*"
	}

	proto := rule.Protocol
	if proto == "" || proto == "any" {
		proto = "any"
	}

	return fmt.Sprintf("%s:%s → %s:%s (%s)", srcIP, srcPort, dstIP, dstPort, proto)
}
