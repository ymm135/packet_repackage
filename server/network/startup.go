package network

import (
	"packet-repackage/database"
	"packet-repackage/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// LoadAndApplyConfigs loads all interface configurations from database and applies them
func LoadAndApplyConfigs(db *gorm.DB) error {
	database.Logger.Info("Loading network configurations from database")
	
	// Ensure bridge is initialized once
	if err := EnsureBridgeExists(); err != nil {
		database.Logger.Error("Failed to initialize Bridge", zap.Error(err))
		return err
	}
	var configs []models.InterfaceConfig
	result := db.Find(&configs)
	
	if result.Error != nil {
		database.Logger.Error("Failed to load configurations from database", zap.Error(result.Error))
		return result.Error
	}
	
	database.Logger.Info("Found configurations to apply", zap.Int("count", len(configs)))
	
	successCount := 0
	failCount := 0
	
	for _, config := range configs {
		database.Logger.Info("Applying configuration",
			zap.String("interface", config.OutInterface),
			zap.String("link_type", config.LinkType),
			zap.String("vlan_id", config.VlanId))
		
		err := AddVlan(config, config.OutInterface, db)
		if err != nil {
			database.Logger.Error("Failed to apply configuration",
				zap.String("interface", config.OutInterface),
				zap.Error(err))
			failCount++
			continue
		}
		
		successCount++
		database.Logger.Info("Configuration applied successfully",
			zap.String("interface", config.OutInterface))
	}
	
	database.Logger.Info("Configuration loading complete",
		zap.Int("success", successCount),
		zap.Int("failed", failCount))
	
	return nil
}
