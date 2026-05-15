package agent

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func SetupRoutes(r *gin.RouterGroup, logger *zap.Logger) {
	agents := r.Group("/agents")
	agents.Use(authMiddleware())
	{
		agents.POST("", createAgentHandler)
		agents.GET("", listAgentsHandler)
		agents.GET("/:id", getAgentHandler)
		agents.PUT("/:id", updateAgentHandler)
		agents.DELETE("/:id", deleteAgentHandler)
		agents.POST("/:id/start", startAgentHandler)
		agents.POST("/:id/stop", stopAgentHandler)
		agents.POST("/:id/execute", executeAgentHandler)
	}
}