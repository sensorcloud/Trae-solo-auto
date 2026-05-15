package monitor

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func SetupRoutes(r *gin.RouterGroup, logger *zap.Logger) {
	metrics := r.Group("/metrics")
	{
		metrics.GET("/health", healthCheckHandler)
		metrics.GET("/system", systemMetricsHandler)
	}

	alerts := r.Group("/alerts")
	alerts.Use(authMiddleware())
	{
		alerts.GET("", listAlertsHandler)
		alerts.POST("", createAlertHandler)
		alerts.GET("/:id", getAlertHandler)
		alerts.PUT("/:id", updateAlertHandler)
		alerts.DELETE("/:id", deleteAlertHandler)
	}
}