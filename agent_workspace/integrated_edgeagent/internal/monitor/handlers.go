package monitor

import (
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"gitcode.com/ywtech/EdgeAgent-Hub/internal/agent"
)

var db *gorm.DB

func InitDB(database *gorm.DB) {
	db = database
}

func authMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 1003, "message": "未授权，请先登录"})
			return
		}
		c.Set("user_id", userID.(uint))
		c.Next()
	}
}

func healthCheckHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": HealthResponse{
		Status:  "healthy",
		Message: "EdgeAgent-Hub is running",
		Uptime:  time.Since(startTime).String(),
	}})
}

var startTime = time.Now()

func systemMetricsHandler(c *gin.Context) {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	cpuPercent := getCPUPercent()
	memPercent := float64(m.Alloc) / float64(m.TotalAlloc) * 100

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": SystemMetricsResponse{
		CPU:         cpuPercent,
		Memory:      memPercent,
		Disk:        getDiskUsage(),
		Network:     0,
		ActiveAgents: getActiveAgentCount(),
		ActiveOrders: 0,
	}})
}

func getCPUPercent() float64 {
	return 0
}

func getDiskUsage() float64 {
	var stat os.FileInfo
	stat, _ = os.Stat("/")
	fs := stat.Sys()
	if fs == nil {
		return 0
	}
	return 0
}

func getActiveAgentCount() int {
	return len(agent.GetActiveAgents())
}

func listAlertsHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var alerts []Alert

	if err := db.Where("user_id = ?", userID).Find(&alerts).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	response := make([]AlertResponse, len(alerts))
	for i, alert := range alerts {
		response[i] = alert.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": response})
}

func createAlertHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req CreateAlertRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	alert := Alert{
		UserID:       userID,
		Name:         req.Name,
		Description:  req.Description,
		Metric:       req.Metric,
		Operator:     req.Operator,
		Threshold:    req.Threshold,
		Severity:     req.Severity,
		Notification: req.Notification,
	}

	if err := db.Create(&alert).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "创建成功", "data": alert.ToResponse()})
}

func getAlertHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	alertID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var alert Alert
	if err := db.Where("id = ? AND user_id = ?", alertID, userID).First(&alert).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "告警规则不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": alert.ToResponse()})
}

func updateAlertHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	alertID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var alert Alert
	if err := db.Where("id = ? AND user_id = ?", alertID, userID).First(&alert).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "告警规则不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	var req UpdateAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	if req.Name != "" {
		alert.Name = req.Name
	}
	if req.Description != "" {
		alert.Description = req.Description
	}
	if req.Metric != "" {
		alert.Metric = req.Metric
	}
	if req.Operator != "" {
		alert.Operator = req.Operator
	}
	if req.Threshold != 0 {
		alert.Threshold = req.Threshold
	}
	if req.Severity != "" {
		alert.Severity = req.Severity
	}
	if req.Notification != "" {
		alert.Notification = req.Notification
	}

	if err := db.Save(&alert).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功", "data": alert.ToResponse()})
}

func deleteAlertHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	alertID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	if err := db.Where("id = ? AND user_id = ?", alertID, userID).Delete(&Alert{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}