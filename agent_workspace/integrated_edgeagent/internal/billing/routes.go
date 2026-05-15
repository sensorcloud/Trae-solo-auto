package billing

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func SetupRoutes(r *gin.RouterGroup, logger *zap.Logger) {
	bills := r.Group("/bills")
	bills.Use(authMiddleware())
	{
		bills.GET("", listBillsHandler)
		bills.GET("/:id", getBillHandler)
		bills.POST("/pay", payBillHandler)
	}

	metering := r.Group("/metering")
	metering.Use(authMiddleware())
	{
		metering.GET("/usage", getUsageHandler)
		metering.POST("/record", recordUsageHandler)
	}
}