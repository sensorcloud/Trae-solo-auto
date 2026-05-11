package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/edgehub/edgehub/internal/api/handlers"
	"github.com/edgehub/edgehub/internal/api/middleware"
	"github.com/edgehub/edgehub/internal/config"
	"github.com/edgehub/edgehub/internal/repository"
	"github.com/edgehub/edgehub/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	db, err := repository.NewPostgresDB(cfg.Database)
	if err != nil {
		log.Printf("Warning: Database connection failed: %v (continuing without DB)", err)
		db = nil
	}

	redisClient := repository.NewRedisClient(cfg.Redis)
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Printf("Warning: Redis connection failed: %v", err)
	}

	nodeService := service.NewNodeService(db)
	jobService := service.NewJobService(db)
	marketService := service.NewMarketService(db)
	billingService := service.NewBillingService(db, redisClient)
	monitorService := service.NewMonitorService(db, redisClient)
	authService := service.NewAuthService(db, &service.JWTConfig{
		Secret:     cfg.JWT.Secret,
		Expiration: cfg.JWT.Expiration,
		RefreshExp: cfg.JWT.RefreshExp,
	})

	nodeHandler := handlers.NewNodeHandler(nodeService)
	jobHandler := handlers.NewJobHandler(jobService)
	marketHandler := handlers.NewMarketHandler(marketService)
	billingHandler := handlers.NewBillingHandler(billingService)
	monitorHandler := handlers.NewMonitorHandler(monitorService)
	authHandler := handlers.NewAuthHandler(authService)

	if cfg.Mode == "release" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	router.Use(middleware.CORS())
	router.Use(middleware.RequestID())
	router.Use(middleware.Metrics())
	router.Use(middleware.Tracing())

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "healthy", "version": "v1.0.0"})
	})

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	api := router.Group("/api/v1")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/register", authHandler.Register)
			auth.POST("/refresh", authHandler.Refresh)
			auth.POST("/logout", middleware.Authenticate(cfg.JWT.Secret), authHandler.Logout)
		}

		protected := api.Group("")
		protected.Use(middleware.Authenticate(cfg.JWT.Secret))
		{
			nodes := protected.Group("/nodes")
			{
				nodes.GET("", nodeHandler.List)
				nodes.POST("", nodeHandler.Register)
				nodes.GET("/:id", nodeHandler.Get)
				nodes.PUT("/:id", nodeHandler.Update)
				nodes.DELETE("/:id", nodeHandler.Delete)
				nodes.POST("/:id/heartbeat", nodeHandler.Heartbeat)
				nodes.GET("/:id/metrics", nodeHandler.GetMetrics)
			}

			clusters := protected.Group("/clusters")
			{
				clusters.GET("", handlers.ListClusters)
				clusters.POST("", handlers.CreateCluster)
				clusters.GET("/:id", handlers.GetCluster)
				clusters.DELETE("/:id", handlers.DeleteCluster)
			}

			jobs := protected.Group("/jobs")
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

			market := protected.Group("/market")
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

			billing := protected.Group("/billing")
			{
				billing.GET("/bills", billingHandler.List)
				billing.GET("/bills/:id", billingHandler.Get)
				billing.GET("/bills/summary", billingHandler.GetSummary)
				billing.GET("/bills/export", billingHandler.Export)
			}

			monitoring := protected.Group("/monitoring")
			{
				monitoring.GET("/metrics", monitorHandler.GetMetrics)
				monitoring.GET("/metrics/query", monitorHandler.QueryMetrics)
				monitoring.GET("/alerts", monitorHandler.ListAlerts)
				monitoring.POST("/alerts", monitorHandler.CreateAlert)
				monitoring.PUT("/alerts/:id", monitorHandler.UpdateAlert)
			}
		}
	}

	srv := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      router,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		log.Printf("Starting EdgeHub API server on port %d", cfg.Server.Port)
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down server...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer shutdownCancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Fatalf("Server forced to shutdown: %v", err)
	}

	log.Println("Server exited properly")
}
