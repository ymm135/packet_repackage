package main

import (
	"flag"
	"fmt"
	"packet-repackage/api"
	"packet-repackage/database"
	"packet-repackage/network"
	"packet-repackage/nfqueue"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func main() {
	// Parse command line flags
	dbPath := flag.String("db", "./data/packet_repackage.db", "Path to SQLite database")
	port := flag.String("port", "8080", "Web server port")
	queueDefs := flag.String("queues", "0", "NFQueue numbers to listen on (e.g. '0', '0-3', '0,1,2')")
	noQueue := flag.Bool("no-queue", false, "Disable NFQueue (API only mode)")
	logPath := flag.String("log-path", "./log/backend.log", "Path to log file")
	logLevel := flag.String("log-level", "debug", "Log level (debug, info, warn, error)")
	flag.Parse()

	// Initialize logger
	err := database.InitLogger(*logPath, *logLevel)
	if err != nil {
		panic(fmt.Sprintf("Failed to initialize logger: %v", err))
	}
	defer database.Logger.Sync()

	// Initialize database
	err = database.InitDatabase(*dbPath)
	if err != nil {
		database.Logger.Fatal("Failed to initialize database", zap.Error(err))
	}

	database.Logger.Info("Packet Repackage Server starting",
		zap.String("db", *dbPath),
		zap.String("port", *port),
		zap.String("queues", *queueDefs))

	// Load and apply network configurations from database
	database.Logger.Info("Loading network configurations from database")
	err = network.LoadAndApplyConfigs(database.DB)
	if err != nil {
		database.Logger.Error("Failed to load network configurations", zap.Error(err))
		// Continue startup even if config loading fails
	}

	// Apply NFTables rules from database (only if not in no-queue mode)
	if !*noQueue {
		database.Logger.Info("Applying NFTables rules from database")
		err = network.ApplyNFTRules(database.DB)
		if err != nil {
			database.Logger.Error("Failed to apply NFT rules", zap.Error(err))
			// Continue startup even if rule application fails
		}
	}

	// Parse queue definitions
	var queues []uint16
	if !*noQueue {
		queues, err = parseQueueDefs(*queueDefs)
		if err != nil {
			database.Logger.Fatal("Failed to parse queue definitions", zap.Error(err))
		}
	}

	// Start NFQueue handler if enabled
	if !*noQueue {
		err = nfqueue.Start(queues)
		if err != nil {
			database.Logger.Error("Failed to start NFQueue, continuing in API-only mode",
				zap.Error(err))
		} else {
			defer nfqueue.Stop()
		}
	}

	// Setup Gin router
	router := gin.Default()

	// Add request/response logging middleware
	router.Use(func(c *gin.Context) {
		// Log request
		database.Logger.Info("Incoming request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("client_ip", c.ClientIP()))
		
		c.Next()
		
		// Log response
		database.Logger.Info("Request completed",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.Int("status", c.Writer.Status()))
		
		// Log errors if any
		if len(c.Errors) > 0 {
			for _, err := range c.Errors {
				database.Logger.Error("Request error",
					zap.String("path", c.Request.URL.Path),
					zap.Error(err))
			}
		}
	})

	// Enable CORS for frontend
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		
		c.Next()
	})

	// API routes
	apiGroup := router.Group("/api")
	{
		// Network configuration
		apiGroup.GET("/interfaces", api.ListInterfaces)
		apiGroup.GET("/interfaces/:name", api.GetInterface)
		apiGroup.POST("/vlan", api.ConfigureVLAN)
		apiGroup.GET("/vlan/:interface", api.GetVLANConfig)
		apiGroup.POST("/vlan/ip", api.AddVLANIP)
		apiGroup.DELETE("/vlan/ip/:interface", api.RemoveVLANIP)
		apiGroup.POST("/interface/status", api.SetInterfaceStatus)

		// Field management
		apiGroup.GET("/fields", api.ListFields)
		apiGroup.GET("/fields/:id", api.GetField)
		apiGroup.POST("/fields", api.CreateField)
		apiGroup.PUT("/fields/:id", api.UpdateField)
		apiGroup.DELETE("/fields/:id", api.DeleteField)

		// Rule management
		apiGroup.GET("/rules", api.ListRules)
		apiGroup.GET("/rules/:id", api.GetRule)
		apiGroup.POST("/rules", api.CreateRule)
		apiGroup.PUT("/rules/:id", api.UpdateRule)
		apiGroup.DELETE("/rules/:id", api.DeleteRule)
		apiGroup.POST("/rules/:id/toggle", api.ToggleRule)

		// NFTables rule management
		apiGroup.GET("/nftrules", api.ListNFTRules)
		apiGroup.GET("/nftrules/:id", api.GetNFTRule)
		apiGroup.POST("/nftrules", api.CreateNFTRule)
		apiGroup.PUT("/nftrules/:id", api.UpdateNFTRule)
		apiGroup.DELETE("/nftrules/:id", api.DeleteNFTRule)
		apiGroup.POST("/nftrules/:id/toggle", api.ToggleNFTRule)
		apiGroup.POST("/nftrules/apply", api.ApplyNFTRulesAPI)

		// Test mode
		apiGroup.POST("/test", api.TestRule)

		// Logs
		apiGroup.GET("/logs", api.ListLogs)
		apiGroup.GET("/logs/:id", api.GetLog)
		apiGroup.DELETE("/logs", api.ClearLogs)
	}

	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Start server
	addr := ":" + *port
	database.Logger.Info("Server listening", zap.String("addr", addr))
	if err := router.Run(addr); err != nil {
		database.Logger.Fatal("Failed to start server", zap.Error(err))
	}
}

func parseQueueDefs(def string) ([]uint16, error) {
	var queues []uint16
	
	// Split by comma first
	parts := strings.Split(def, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		// Check for range
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return nil, fmt.Errorf("invalid range format: %s", part)
			}
			
			start, err := strconv.ParseUint(strings.TrimSpace(rangeParts[0]), 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid start of range: %s", rangeParts[0])
			}
			
			end, err := strconv.ParseUint(strings.TrimSpace(rangeParts[1]), 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid end of range: %s", rangeParts[1])
			}
			
			if start > end {
				return nil, fmt.Errorf("invalid range: start %d > end %d", start, end)
			}
			
			for i := start; i <= end; i++ {
				queues = append(queues, uint16(i))
			}
		} else {
			// Single number
			q, err := strconv.ParseUint(part, 10, 16)
			if err != nil {
				return nil, fmt.Errorf("invalid queue number: %s", part)
			}
			queues = append(queues, uint16(q))
		}
	}
	
	if len(queues) == 0 {
		return nil, fmt.Errorf("no valid queues specified")
	}
	
	return queues, nil
}
