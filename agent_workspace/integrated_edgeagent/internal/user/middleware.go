package user

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gitcode.com/ywtech/EdgeAgent-Hub/pkg/middleware"
)

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1003, "message": "未授权，请先登录"})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1003, "message": "Token格式错误"})
			return
		}

		claims, err := middleware.ValidateAccessToken(parts[1])
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1003, "message": "Token无效或已过期"})
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)

		c.Next()
	}
}

func requireRole(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		userRole := c.GetString("role")

		for _, role := range roles {
			if userRole == role {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{"code": 1004, "message": "权限不足"})
	}
}