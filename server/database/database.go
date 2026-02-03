package database

import (
	"packet-repackage/models"

	"go.uber.org/zap"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var DB *gorm.DB
var Logger *zap.Logger

// InitDatabase initializes the SQLite database and runs migrations
func InitDatabase(dbPath string) error {
	var err error
	
	// Open database connection
	DB, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return err
	}

	// Auto-migrate all models
	err = DB.AutoMigrate(
		&models.Field{},
		&models.Rule{},
		&models.InterfaceConfig{},
		&models.VlanConfig{},
		&models.VlanConfigIP{},
		&models.ProcessLog{},
		&models.NFTRule{},
	)
	if err != nil {
		return err
	}

	Logger.Info("Database initialized successfully")
	return nil
}

// InitLogger initializes the zap logger with file and console output
func InitLogger(logPath string, logLevel string) error {
	// Parse log level
	level, err := zap.ParseAtomicLevel(logLevel)
	if err != nil {
		return err
	}

	// Create logger configuration
	config := zap.Config{
		Level:            level,
		Development:      false,
		Encoding:         "json",
		EncoderConfig:    zap.NewProductionEncoderConfig(),
		OutputPaths:      []string{"stdout", logPath},
		ErrorOutputPaths: []string{"stderr", logPath},
	}

	// Build logger
	Logger, err = config.Build()
	if err != nil {
		return err
	}
	
	return nil
}
