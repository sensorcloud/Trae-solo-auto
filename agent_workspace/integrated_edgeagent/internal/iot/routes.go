package iot

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func SetupRoutes(r *gin.RouterGroup, logger *zap.Logger) {
	devices := r.Group("/devices")
	devices.Use(authMiddleware())
	{
		devices.POST("", createDeviceHandler)
		devices.GET("", listDevicesHandler)
		devices.GET("/:id", getDeviceHandler)
		devices.PUT("/:id", updateDeviceHandler)
		devices.DELETE("/:id", deleteDeviceHandler)
		devices.GET("/:id/status", getDeviceStatusHandler)
	}

	telemetry := r.Group("/telemetry")
	telemetry.Use(authMiddleware())
	{
		telemetry.GET("/:device_id", getTelemetryHandler)
		telemetry.POST("/:device_id", submitTelemetryHandler)
	}
}