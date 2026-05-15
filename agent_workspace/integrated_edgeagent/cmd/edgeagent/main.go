package main

import (
	"flag"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"github.com/ywtech/edgeagent-hub/internal/agent"
	"github.com/ywtech/edgeagent-hub/internal/billing"
	"github.com/ywtech/edgeagent-hub/internal/coordination"
	"github.com/ywtech/edgeagent-hub/internal/energy"
	"github.com/ywtech/edgeagent-hub/internal/iot"
	"github.com/ywtech/edgeagent-hub/internal/market"
	"github.com/ywtech/edgeagent-hub/internal/monitor"
	"github.com/ywtech/edgeagent-hub/internal/user"
	"github.com/ywtech/edgeagent-hub/pkg/config"
	"github.com/ywtech/edgeagent-hub/pkg/database"
	"github.com/ywtech/edgeagent-hub/pkg/logging"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "", "Path to config file")
	flag.Parse()

	if err := config.LoadConfig(configPath); err != nil {
		panic(err)
	}

	logger := logging.NewLogger()

	db, err := database.InitDB()
	if err != nil {
		logger.Fatal("Failed to connect to database", zap.Error(err))
		os.Exit(1)
	}

	if err := database.AutoMigrate(db); err != nil {
		logger.Fatal("Failed to migrate database", zap.Error(err))
		os.Exit(1)
	}

	database.InitModules(db)

	r := gin.Default()

	setupRoutes(r, logger)

	port := viper.GetString("server.port")
	logger.Info("Starting server on port " + port)

	if err := r.Run(":" + port); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
		os.Exit(1)
	}
}

func setupRoutes(r *gin.Engine, logger *zap.Logger) {
	api := r.Group("/api/v1")

	user.SetupRoutes(api, logger)
	agent.SetupRoutes(api, logger)
	market.SetupRoutes(api, logger)
	energy.SetupRoutes(api, logger)
	coordination.SetupRoutes(api, logger)
	iot.SetupRoutes(api, logger)
	billing.SetupRoutes(api, logger)
	monitor.SetupRoutes(api, logger)
}