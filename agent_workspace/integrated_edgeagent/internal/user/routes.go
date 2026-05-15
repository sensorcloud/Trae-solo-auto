package user

import (
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func SetupRoutes(r *gin.RouterGroup, logger *zap.Logger) {
	auth := r.Group("/auth")
	{
		auth.POST("/register", registerHandler)
		auth.POST("/login", loginHandler)
		auth.POST("/logout", logoutHandler)
		auth.POST("/refresh", refreshHandler)
	}

	users := r.Group("/users")
	users.Use(authMiddleware())
	{
		users.GET("/me", getUserHandler)
		users.PUT("/me", updateUserHandler)
		users.DELETE("/me", deleteUserHandler)
	}
}