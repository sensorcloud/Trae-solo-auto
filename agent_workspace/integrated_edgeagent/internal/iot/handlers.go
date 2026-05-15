package iot

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/ywtech/edgeagent-hub/pkg/utils"
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

func createDeviceHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req CreateDeviceRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	device := Device{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Protocol:    req.Protocol,
		Address:     req.Address,
		Port:        req.Port,
		Metadata:    req.Metadata,
	}

	if err := db.Create(&device).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "创建成功", "data": device.ToResponse()})
}

func listDevicesHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var devices []Device

	if err := db.Where("user_id = ?", userID).Find(&devices).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	response := make([]DeviceResponse, len(devices))
	for i, device := range devices {
		response[i] = device.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": response})
}

func getDeviceHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	deviceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var device Device
	if err := db.Where("id = ? AND user_id = ?", deviceID, userID).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "设备不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": device.ToResponse()})
}

func updateDeviceHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	deviceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var device Device
	if err := db.Where("id = ? AND user_id = ?", deviceID, userID).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "设备不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	var req CreateDeviceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	device.Name = req.Name
	device.Description = req.Description
	device.Protocol = req.Protocol
	device.Address = req.Address
	device.Port = req.Port
	device.Metadata = req.Metadata

	if err := db.Save(&device).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功", "data": device.ToResponse()})
}

func deleteDeviceHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	deviceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	if err := db.Where("id = ? AND user_id = ?", deviceID, userID).Delete(&Device{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func getDeviceStatusHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	deviceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var device Device
	if err := db.Where("id = ? AND user_id = ?", deviceID, userID).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "设备不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "success",
		"data": gin.H{
			"id":        device.ID,
			"name":      device.Name,
			"status":    device.Status,
			"last_seen": device.LastSeen,
			"protocol":  device.Protocol,
		},
	})
}

func getTelemetryHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	deviceID, err := strconv.ParseUint(c.Param("device_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var device Device
	if err := db.Where("id = ? AND user_id = ?", deviceID, userID).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "设备不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	var telemetries []Telemetry
	if err := db.Where("device_id = ?", deviceID).Order("timestamp DESC").Limit(100).Find(&telemetries).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	response := make([]TelemetryResponse, len(telemetries))
	for i, telemetry := range telemetries {
		response[i] = telemetry.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": response})
}

func submitTelemetryHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	deviceID, err := strconv.ParseUint(c.Param("device_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var device Device
	if err := db.Where("id = ? AND user_id = ?", deviceID, userID).First(&device).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "设备不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	var req SubmitTelemetryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	timestamp := req.Timestamp
	if timestamp == 0 {
		timestamp = utils.GetCurrentTimestamp()
	}

	telemetry := Telemetry{
		DeviceID:  uint(deviceID),
		Timestamp: timestamp,
		Metric:    req.Metric,
		Value:     req.Value,
		Unit:      req.Unit,
	}

	if err := db.Create(&telemetry).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	device.Status = "online"
	device.LastSeen = timestamp
	db.Save(&device)

	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "提交成功", "data": telemetry.ToResponse()})
}