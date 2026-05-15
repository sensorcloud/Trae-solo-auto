package market

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func SetupRoutes(r *gin.RouterGroup, logger *zap.Logger) {
	assets := r.Group("/assets")
	assets.Use(authMiddleware())
	{
		assets.POST("", createAssetHandler)
		assets.GET("", listAssetsHandler)
		assets.GET("/:id", getAssetHandler)
		assets.PUT("/:id", updateAssetHandler)
		assets.DELETE("/:id", deleteAssetHandler)
	}

	orders := r.Group("/orders")
	orders.Use(authMiddleware())
	{
		orders.POST("", createOrderHandler)
		orders.GET("", listOrdersHandler)
		orders.GET("/:id", getOrderHandler)
		orders.PUT("/:id/pay", payOrderHandler)
		orders.PUT("/:id/cancel", cancelOrderHandler)
	}
}