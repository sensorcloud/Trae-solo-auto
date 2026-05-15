package coordination

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func SetupRoutes(r *gin.RouterGroup, logger *zap.Logger) {
	schedule := r.Group("/schedule")
	schedule.Use(authMiddleware())
	{
		schedule.POST("/predict", predictHandler)
		schedule.POST("/optimize", optimizeHandler)
		schedule.POST("/execute", executeHandler)
		schedule.GET("/history", getHistoryHandler)
		schedule.GET("/history/:id", getHistoryDetailHandler)
	}
}