package api

import (
	"net/http"
	"time"

	"github.com/edgehub/edgehub/internal/api/handlers"
	"github.com/edgehub/edgehub/internal/api/middleware"
	"github.com/edgehub/edgehub/internal/config"
	"github.com/edgehub/edgehub/internal/iot"
	"github.com/edgehub/edgehub/internal/repository"
	"github.com/edgehub/edgehub/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"gorm.io/gorm"
)

type Router struct {
	engine          *gin.Engine
	config          *config.Config
	db              *gorm.DB
	redis           *repository.RedisClient
	jwtSecret       string
}

func NewRouter(cfg *config.Config, db *gorm.DB, redis *repository.RedisClient) *Router {
	if cfg.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	engine := gin.New()
	engine.Use(gin.Logger())
	engine.Use(gin.Recovery())

	return &Router{
		engine:    engine,
		config:    cfg,
		db:        db,
		redis:     redis,
		jwtSecret: cfg.JWT.Secret,
	}
}

func (r *Router) Setup() *gin.Engine {
	r.setupGlobalMiddleware()
	r.setupHealthRoutes()
	r.setupMetricsRoutes()
	r.setupAuthRoutes()
	r.setupProtectedRoutes()

	return r.engine
}

func (r *Router) setupGlobalMiddleware() {
	r.engine.Use(middleware.CORS())
	r.engine.Use(middleware.RequestID())
	r.engine.Use(middleware.Metrics())
	r.engine.Use(middleware.Tracing())
	r.engine.Use(middleware.TenantMiddleware())
}

func (r *Router) setupHealthRoutes() {
	r.engine.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"version":   "v1.0.0",
			"timestamp": time.Now().Unix(),
		})
	})

	r.engine.GET("/ready", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "ready",
		})
	})
}

func (r *Router) setupMetricsRoutes() {
	r.engine.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

func (r *Router) setupAuthRoutes() {
	authService := service.NewAuthService(r.db, &service.JWTConfig{
		Secret:     r.config.JWT.Secret,
		Expiration: r.config.JWT.Expiration,
		RefreshExp: r.config.JWT.RefreshExp,
	})
	authHandler := handlers.NewAuthHandler(authService)

	auth := r.engine.Group("/api/v1/auth")
	{
		auth.POST("/login", authHandler.Login)
		auth.POST("/register", authHandler.Register)
		auth.POST("/refresh", authHandler.Refresh)
		auth.POST("/logout", middleware.Authenticate(r.jwtSecret), authHandler.Logout)
	}
}

func (r *Router) setupProtectedRoutes() {
	protected := r.engine.Group("/api/v1")
	protected.Use(middleware.Authenticate(r.jwtSecret))
	{
		r.setupNodeRoutes(protected)
		r.setupClusterRoutes(protected)
		r.setupJobRoutes(protected)
		r.setupMarketRoutes(protected)
		r.setupBillingRoutes(protected)
		r.setupMonitoringRoutes(protected)
		r.setupEnergyRoutes(protected)
		r.setupStorageRoutes(protected)
		r.setupTradingRoutes(protected)
		r.setupVPPRoutes(protected)
		r.setupIoTRoutes(protected)
		r.setupAgentRoutes(protected)
		r.setupCoordinationRoutes(protected)
	}
}

func (r *Router) setupNodeRoutes(rg *gin.RouterGroup) {
	nodeService := service.NewNodeService(r.db)
	nodeHandler := handlers.NewNodeHandler(nodeService)

	nodes := rg.Group("/nodes")
	{
		nodes.GET("", nodeHandler.List)
		nodes.POST("", nodeHandler.Register)
		nodes.GET("/:id", nodeHandler.Get)
		nodes.PUT("/:id", nodeHandler.Update)
		nodes.DELETE("/:id", nodeHandler.Delete)
		nodes.POST("/:id/heartbeat", nodeHandler.Heartbeat)
		nodes.GET("/:id/metrics", nodeHandler.GetMetrics)
	}
}

func (r *Router) setupClusterRoutes(rg *gin.RouterGroup) {
	clusters := rg.Group("/clusters")
	{
		clusters.GET("", handlers.ListClusters)
		clusters.POST("", handlers.CreateCluster)
		clusters.GET("/:id", handlers.GetCluster)
		clusters.DELETE("/:id", handlers.DeleteCluster)
	}
}

func (r *Router) setupJobRoutes(rg *gin.RouterGroup) {
	jobService := service.NewJobService(r.db)
	jobHandler := handlers.NewJobHandler(jobService)

	jobs := rg.Group("/jobs")
	{
		jobs.GET("", jobHandler.List)
		jobs.POST("", jobHandler.Submit)
		jobs.GET("/:id", jobHandler.Get)
		jobs.PUT("/:id", jobHandler.Update)
		jobs.DELETE("/:id", jobHandler.Delete)
		jobs.POST("/:id/stop", jobHandler.Stop)
		jobs.GET("/:id/logs", jobHandler.GetLogs)
		jobs.GET("/:id/metrics", jobHandler.GetMetrics)
	}
}

func (r *Router) setupMarketRoutes(rg *gin.RouterGroup) {
	marketService := service.NewMarketService(r.db)
	marketHandler := handlers.NewMarketHandler(marketService)

	market := rg.Group("/market")
	{
		market.GET("/offers", marketHandler.ListOffers)
		market.POST("/offers", marketHandler.CreateOffer)
		market.GET("/offers/:id", marketHandler.GetOffer)
		market.DELETE("/offers/:id", marketHandler.DeleteOffer)
		market.POST("/orders", marketHandler.CreateOrder)
		market.GET("/orders/:id", marketHandler.GetOrder)
		market.GET("/prices", marketHandler.GetPrices)
		market.GET("/prices/recommend", marketHandler.GetRecommendation)
	}
}

func (r *Router) setupBillingRoutes(rg *gin.RouterGroup) {
	billingService := service.NewBillingService(r.db, r.redis)
	billingHandler := handlers.NewBillingHandler(billingService)

	billing := rg.Group("/billing")
	{
		billing.GET("/bills", billingHandler.List)
		billing.GET("/bills/:id", billingHandler.Get)
		billing.GET("/bills/summary", billingHandler.GetSummary)
		billing.GET("/bills/export", billingHandler.Export)
	}
}

func (r *Router) setupMonitoringRoutes(rg *gin.RouterGroup) {
	monitorService := service.NewMonitorService(r.db, r.redis)
	monitorHandler := handlers.NewMonitorHandler(monitorService)

	monitoring := rg.Group("/monitoring")
	{
		monitoring.GET("/metrics", monitorHandler.GetMetrics)
		monitoring.GET("/metrics/query", monitorHandler.QueryMetrics)
		monitoring.GET("/alerts", monitorHandler.ListAlerts)
		monitoring.POST("/alerts", monitorHandler.CreateAlert)
		monitoring.PUT("/alerts/:id", monitorHandler.UpdateAlert)
	}
}

func (r *Router) setupEnergyRoutes(rg *gin.RouterGroup) {
	energyHandler := handlers.NewEnergyHandler(
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
		nil,
	)

	energy := rg.Group("/energy")
	{
		energy.GET("/power-sources", energyHandler.ListPowerSources)
		energy.POST("/power-sources", energyHandler.CreatePowerSource)
		energy.GET("/power-sources/:id", energyHandler.GetPowerSource)
		energy.PUT("/power-sources/:id", energyHandler.UpdatePowerSource)
		energy.DELETE("/power-sources/:id", energyHandler.DeletePowerSource)
		energy.GET("/power-sources/:id/stats", energyHandler.GetPowerGenerationStats)

		energy.GET("/loads", energyHandler.ListLoadProfiles)
		energy.POST("/loads", energyHandler.CreateLoadProfile)
		energy.GET("/loads/:id", energyHandler.GetLoadProfile)
		energy.GET("/loads/:id/forecast", energyHandler.ForecastLoad)
		energy.POST("/loads/:id/adjust", energyHandler.AdjustLoad)

		energy.GET("/market/overview", energyHandler.GetMarketOverview)
		energy.GET("/market/volume", energyHandler.GetTradingVolume)
		energy.GET("/market/depth", energyHandler.GetMarketDepth)
	}
}

func (r *Router) setupStorageRoutes(rg *gin.RouterGroup) {
	storageHandler := handlers.NewEnergyHandler(nil, nil, nil, nil, nil, nil, nil)

	storage := rg.Group("/storage")
	{
		storage.GET("/devices", storageHandler.ListStorageDevices)
		storage.POST("/devices", storageHandler.CreateStorageDevice)
		storage.GET("/devices/:id", storageHandler.GetStorageDevice)
		storage.PUT("/devices/:id", storageHandler.UpdateStorageDevice)
		storage.DELETE("/devices/:id", storageHandler.DeleteStorageDevice)
		storage.POST("/devices/:id/charge", storageHandler.ChargeStorage)
		storage.POST("/devices/:id/discharge", storageHandler.DischargeStorage)
		storage.POST("/devices/:id/stop", storageHandler.StopStorage)
		storage.GET("/devices/:id/status", storageHandler.GetStorageStatus)
		storage.POST("/devices/:id/optimize", storageHandler.OptimizeStorageSchedule)
	}
}

func (r *Router) setupTradingRoutes(rg *gin.RouterGroup) {
	tradingHandler := handlers.NewEnergyHandler(nil, nil, nil, nil, nil, nil, nil)

	trading := rg.Group("/trading")
	{
		trading.GET("/orders", tradingHandler.ListOrders)
		trading.POST("/orders", tradingHandler.CreateOrder)
		trading.GET("/orders/:id", tradingHandler.GetOrder)
		trading.POST("/orders/:id/cancel", tradingHandler.CancelOrder)
		trading.POST("/orders/:id/submit", tradingHandler.SubmitOrder)
		trading.GET("/prices", tradingHandler.GetPriceQuote)
		trading.GET("/prices/history", tradingHandler.GetPriceHistory)
		trading.GET("/green-certificates", tradingHandler.ListGreenCertificates)
		trading.POST("/green-certificates/:id/transfer", tradingHandler.TransferGreenCertificate)
	}
}

func (r *Router) setupVPPRoutes(rg *gin.RouterGroup) {
	vppHandler := handlers.NewEnergyHandler(nil, nil, nil, nil, nil, nil, nil)

	vpp := rg.Group("/vpp")
	{
		vpp.GET("", vppHandler.ListVPPs)
		vpp.POST("", vppHandler.CreateVPP)
		vpp.GET("/:id", vppHandler.GetVPP)
		vpp.PUT("/:id", vppHandler.UpdateVPP)
		vpp.DELETE("/:id", vppHandler.DeleteVPP)
		vpp.POST("/:id/dispatch", vppHandler.DispatchVPP)
		vpp.GET("/:id/dispatch-status", vppHandler.GetVPPDispatchStatus)
		vpp.GET("/:id/capacity", vppHandler.AggregateVPPCapacity)
		vpp.POST("/:id/power-sources/:source_id", vppHandler.AddPowerSourceToVPP)
		vpp.POST("/:id/storages/:storage_id", vppHandler.AddStorageToVPP)
	}
}

func (r *Router) setupIoTRoutes(rg *gin.RouterGroup) {
	deviceMgr := iot.NewDeviceManager(r.db, uuid.Nil)
	telemetryMgr := iot.NewTelemetryManager(r.db, uuid.Nil)
	connectorMgr := iot.NewConnectorManager(nil)

	iotHandler := handlers.NewIoTHandler(deviceMgr, telemetryMgr, connectorMgr)

	iot := rg.Group("/iot")
	{
		iot.GET("/devices", iotHandler.ListDevices)
		iot.POST("/devices", iotHandler.CreateDevice)
		iot.POST("/devices/batch", iotHandler.CreateDeviceBatch)
		iot.GET("/devices/:id", iotHandler.GetDevice)
		iot.PUT("/devices/:id", iotHandler.UpdateDevice)
		iot.DELETE("/devices/:id", iotHandler.DeleteDevice)
		iot.PUT("/devices/:id/status", iotHandler.UpdateDeviceStatus)
		iot.POST("/devices/:id/enable", iotHandler.EnableDevice)
		iot.POST("/devices/:id/disable", iotHandler.DisableDevice)
		iot.PUT("/devices/:id/labels", iotHandler.SetDeviceLabels)
		iot.POST("/devices/:id/bind/:gateway_id", iotHandler.BindDeviceToGateway)
		iot.POST("/devices/:id/unbind", iotHandler.UnbindDeviceFromGateway)
		iot.GET("/devices/online", iotHandler.GetOnlineDevices)
		iot.GET("/devices/:device_id/shadow", iotHandler.GetDeviceShadow)
		iot.PUT("/devices/:device_id/shadow", iotHandler.UpdateDeviceShadow)

		iot.GET("/profiles", iotHandler.ListDeviceProfiles)
		iot.POST("/profiles", iotHandler.CreateDeviceProfile)
		iot.GET("/profiles/:id", iotHandler.GetDeviceProfile)
		iot.PUT("/profiles/:id", iotHandler.UpdateDeviceProfile)
		iot.DELETE("/profiles/:id", iotHandler.DeleteDeviceProfile)

		iot.GET("/telemetry", iotHandler.GetTelemetry)
		iot.POST("/telemetry", iotHandler.SubmitTelemetry)
		iot.GET("/telemetry/:device_id/latest", iotHandler.GetTelemetryLatest)
		iot.GET("/telemetry/:device_id/history", iotHandler.GetTelemetryHistory)

		iot.POST("/commands", iotHandler.ExecuteCommand)
		iot.GET("/commands/:correlation_id", iotHandler.GetCommandStatus)

		iot.GET("/connectors/:protocol/status", iotHandler.GetConnectorStatus)
		iot.POST("/connectors/:protocol/start", iotHandler.StartConnector)
		iot.POST("/connectors/:protocol/stop", iotHandler.StopConnector)

		iot.GET("/alarms", iotHandler.ListDeviceAlarms)
		iot.POST("/alarms/:id/acknowledge", iotHandler.AcknowledgeAlarm)
		iot.POST("/alarms/:id/clear", iotHandler.ClearAlarm)
	}
}

func (r *Router) setupAgentRoutes(rg *gin.RouterGroup) {
	agentHandler := handlers.NewAgentHandler(nil)

	agents := rg.Group("/agents")
	{
		agents.GET("/sandboxes", agentHandler.ListSandboxes)
		agents.POST("/sandboxes", agentHandler.CreateSandbox)
		agents.GET("/sandboxes/:id", agentHandler.GetSandbox)
		agents.POST("/sandboxes/:id/pause", agentHandler.PauseSandbox)
		agents.POST("/sandboxes/:id/resume", agentHandler.ResumeSandbox)
		agents.POST("/sandboxes/:id/stop", agentHandler.StopSandbox)
		agents.DELETE("/sandboxes/:id", agentHandler.DeleteSandbox)
		agents.GET("/sandboxes/:id/metrics", agentHandler.GetSandboxMetrics)

		agents.GET("", agentHandler.ListAgents)
		agents.POST("/sandboxes/:sandbox_id/agents", agentHandler.CreateAgent)
		agents.GET("/:id", agentHandler.GetAgent)
		agents.DELETE("/:id", agentHandler.DeleteAgent)

		agents.POST("/execute", agentHandler.Execute)
		agents.POST("/:agent_id/execute/code", agentHandler.ExecuteCode)
		agents.POST("/:agent_id/execute/shell", agentHandler.ExecuteShell)
		agents.GET("/executions/:id", agentHandler.GetExecution)
		agents.POST("/executions/:id/cancel", agentHandler.CancelExecution)

		agents.GET("/:agent_id/tools", agentHandler.ListTools)
		agents.POST("/:agent_id/tools", agentHandler.RegisterTool)
		agents.DELETE("/:agent_id/tools/:tool_name", agentHandler.UnregisterTool)
		agents.POST("/:agent_id/tools/:tool_name/execute", agentHandler.ExecuteTool)

		agents.PUT("/sandboxes/:sandbox_id/security", agentHandler.ApplySecurityPolicy)
		agents.PUT("/sandboxes/:sandbox_id/network-policy", agentHandler.ApplyNetworkPolicy)

		agents.GET("/statistics", agentHandler.GetStatistics)
	}
}

func (r *Router) setupCoordinationRoutes(rg *gin.RouterGroup) {
	coordHandler := handlers.NewCoordinationHandler(nil)

	coordination := rg.Group("/coordination")
	{
		coordination.POST("/schedule", coordHandler.ScheduleComputeWithEnergy)
		coordination.GET("/optimal-time", coordHandler.GetOptimalTimeSlot)
		coordination.GET("/forecast", coordHandler.GetEnergyForecast)
		coordination.GET("/carbon-intensity", coordHandler.GetCarbonIntensity)
		coordination.GET("/green-ratio", coordHandler.GetGreenEnergyRatio)
		coordination.POST("/workloads/optimize", coordHandler.OptimizeWorkload)
		coordination.GET("/workloads/:workload_id/schedule", coordHandler.GetWorkloadSchedule)
		coordination.GET("/metrics", coordHandler.GetEnergyMetrics)
		coordination.GET("/status", coordHandler.GetRealtimeStatus)
		coordination.POST("/simulate", coordHandler.SimulateSchedule)
		coordination.GET("/policies", coordHandler.GetPolicies)
		coordination.PUT("/policies/:id", coordHandler.UpdatePolicy)
	}
}

func SetupRouter(cfg *config.Config, db *gorm.DB, redis *repository.RedisClient) *gin.Engine {
	router := NewRouter(cfg, db, redis)
	return router.Setup()
}
