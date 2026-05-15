package agent

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gitcode.com/ywtech/EdgeAgent-Hub/pkg/middleware"
	"gorm.io/gorm"
)

var db *gorm.DB

func InitDB(database *gorm.DB) {
	db = database
}

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
		c.Next()
	}
}

func createAgentHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req CreateAgentRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	agent := Agent{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Runtime:     req.Runtime,
		Config:      req.Config,
		Code:        req.Code,
		Resources:   req.Resources,
	}

	if err := db.Create(&agent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "创建成功", "data": agent.ToResponse()})
}

func listAgentsHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var agents []Agent

	if err := db.Where("user_id = ?", userID).Find(&agents).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	response := make([]AgentResponse, len(agents))
	for i, agent := range agents {
		response[i] = agent.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": response})
}

func getAgentHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	agentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var agent Agent
	if err := db.Where("id = ? AND user_id = ?", agentID, userID).First(&agent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "Agent不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": agent.ToResponse()})
}

func updateAgentHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	agentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var req UpdateAgentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	var agent Agent
	if err := db.Where("id = ? AND user_id = ?", agentID, userID).First(&agent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "Agent不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	if req.Name != "" {
		agent.Name = req.Name
	}
	if req.Description != "" {
		agent.Description = req.Description
	}
	if req.Config != "" {
		agent.Config = req.Config
	}
	if req.Code != "" {
		agent.Code = req.Code
	}

	if err := db.Save(&agent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功", "data": agent.ToResponse()})
}

func deleteAgentHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	agentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	if err := db.Where("id = ? AND user_id = ?", agentID, userID).Delete(&Agent{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func startAgentHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	agentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var agent Agent
	if err := db.Where("id = ? AND user_id = ?", agentID, userID).First(&agent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "Agent不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	if agent.Status == "running" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 2003, "message": "Agent已在运行中"})
		return
	}

	if err := startAgentSandbox(&agent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "启动失败", "data": err.Error()})
		return
	}

	agent.Status = "running"
	if err := db.Save(&agent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "启动成功", "data": agent.ToResponse()})
}

func stopAgentHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	agentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var agent Agent
	if err := db.Where("id = ? AND user_id = ?", agentID, userID).First(&agent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "Agent不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	if agent.Status != "running" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 2003, "message": "Agent未在运行"})
		return
	}

	if err := stopAgentSandbox(&agent); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "停止失败", "data": err.Error()})
		return
	}

	agent.Status = "stopped"
	if err := db.Save(&agent).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "停止成功", "data": agent.ToResponse()})
}

func executeAgentHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	agentID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var req ExecuteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	var agent Agent
	if err := db.Where("id = ? AND user_id = ?", agentID, userID).First(&agent).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "Agent不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	if agent.Status != "running" {
		c.JSON(http.StatusBadRequest, gin.H{"code": 2003, "message": "Agent未运行，请先启动"})
		return
	}

	result, err := executeAgent(&agent, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "执行失败", "data": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": result})
}