package energy

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func SetupRoutes(r *gin.RouterGroup, logger *zap.Logger) {
	power := r.Group("/power")
	power.Use(authMiddleware())
	{
		power.POST("", createPowerSourceHandler)
		power.GET("", listPowerSourcesHandler)
		power.GET("/:id", getPowerSourceHandler)
		power.PUT("/:id", updatePowerSourceHandler)
		power.DELETE("/:id", deletePowerSourceHandler)
		power.GET("/:id/status", getPowerStatusHandler)
	}

	storage := r.Group("/storage")
	storage.Use(authMiddleware())
	{
		storage.POST("", createStorageHandler)
		storage.GET("", listStoragesHandler)
		storage.GET("/:id", getStorageHandler)
		storage.PUT("/:id", updateStorageHandler)
		storage.POST("/:id/charge", chargeStorageHandler)
		storage.POST("/:id/discharge", dischargeStorageHandler)
	}
}