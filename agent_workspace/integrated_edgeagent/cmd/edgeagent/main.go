package main

import (
	"flag"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"

	"gitcode.com/ywtech/EdgeAgent-Hub/internal/agent"
	"gitcode.com/ywtech/EdgeAgent-Hub/internal/billing"
	"gitcode.com/ywtech/EdgeAgent-Hub/internal/coordination"
	"gitcode.com/ywtech/EdgeAgent-Hub/internal/energy"
	"gitcode.com/ywtech/EdgeAgent-Hub/internal/iot"
	"gitcode.com/ywtech/EdgeAgent-Hub/internal/market"
	"gitcode.com/ywtech/EdgeAgent-Hub/internal/monitor"
	"gitcode.com/ywtech/EdgeAgent-Hub/internal/user"
	"gitcode.com/ywtech/EdgeAgent-Hub/pkg/config"
	"gitcode.com/ywtech/EdgeAgent-Hub/pkg/database"
	"gitcode.com/ywtech/EdgeAgent-Hub/pkg/logging"
	"gitcode.com/ywtech/EdgeAgent-Hub/pkg/middleware"
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

	middleware.InitJWTSecrets()

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