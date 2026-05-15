package energy

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
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

func createPowerSourceHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req CreatePowerSourceRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	powerSource := PowerSource{
		UserID:          userID,
		Name:            req.Name,
		Description:     req.Description,
		Type:            req.Type,
		Capacity:        req.Capacity,
		OutputPower:     req.OutputPower,
		Location:        req.Location,
		PricePerKWh:     req.PricePerKWh,
		CarbonIntensity: req.CarbonIntensity,
	}

	if err := db.Create(&powerSource).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "创建成功", "data": powerSource.ToResponse()})
}

func listPowerSourcesHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var sources []PowerSource

	if err := db.Where("user_id = ?", userID).Find(&sources).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	response := make([]PowerSourceResponse, len(sources))
	for i, source := range sources {
		response[i] = source.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": response})
}

func getPowerSourceHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	sourceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var source PowerSource
	if err := db.Where("id = ? AND user_id = ?", sourceID, userID).First(&source).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "电源不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": source.ToResponse()})
}

func updatePowerSourceHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	sourceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var source PowerSource
	if err := db.Where("id = ? AND user_id = ?", sourceID, userID).First(&source).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "电源不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	var req CreatePowerSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	source.Name = req.Name
	source.Description = req.Description
	source.Type = req.Type
	source.Capacity = req.Capacity
	source.OutputPower = req.OutputPower
	source.Location = req.Location
	source.PricePerKWh = req.PricePerKWh
	source.CarbonIntensity = req.CarbonIntensity

	if err := db.Save(&source).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功", "data": source.ToResponse()})
}

func deletePowerSourceHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	sourceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	if err := db.Where("id = ? AND user_id = ?", sourceID, userID).Delete(&PowerSource{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func getPowerStatusHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	sourceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var source PowerSource
	if err := db.Where("id = ? AND user_id = ?", sourceID, userID).First(&source).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "电源不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code": 0,
		"message": "success",
		"data": gin.H{
			"id":      source.ID,
			"name":    source.Name,
			"status":  source.Status,
			"output":  source.OutputPower,
			"price":   source.PricePerKWh,
			"carbon":  source.CarbonIntensity,
		},
	})
}

func createStorageHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req CreateStorageRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	storage := Storage{
		UserID:           userID,
		Name:             req.Name,
		Description:      req.Description,
		Capacity:         req.Capacity,
		MaxChargeRate:    req.MaxChargeRate,
		MaxDischargeRate: req.MaxDischargeRate,
		Efficiency:       req.Efficiency,
	}

	if err := db.Create(&storage).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "创建成功", "data": storage.ToResponse()})
}

func listStoragesHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var storages []Storage

	if err := db.Where("user_id = ?", userID).Find(&storages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	response := make([]StorageResponse, len(storages))
	for i, storage := range storages {
		response[i] = storage.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": response})
}

func getStorageHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	storageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var storage Storage
	if err := db.Where("id = ? AND user_id = ?", storageID, userID).First(&storage).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "储能设备不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": storage.ToResponse()})
}

func updateStorageHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	storageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var storage Storage
	if err := db.Where("id = ? AND user_id = ?", storageID, userID).First(&storage).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "储能设备不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	var req CreateStorageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	storage.Name = req.Name
	storage.Description = req.Description
	storage.Capacity = req.Capacity
	storage.MaxChargeRate = req.MaxChargeRate
	storage.MaxDischargeRate = req.MaxDischargeRate
	storage.Efficiency = req.Efficiency

	if err := db.Save(&storage).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功", "data": storage.ToResponse()})
}

func chargeStorageHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	storageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var req ChargeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	var storage Storage
	if err := db.Where("id = ? AND user_id = ?", storageID, userID).First(&storage).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "储能设备不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	chargeAmount := req.Amount * storage.Efficiency
	newSOC := storage.CurrentSOC + chargeAmount

	if newSOC > storage.Capacity {
		newSOC = storage.Capacity
	}

	storage.CurrentSOC = newSOC
	if err := db.Save(&storage).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "充电成功", "data": storage.ToResponse()})
}

func dischargeStorageHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	storageID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var req DischargeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	var storage Storage
	if err := db.Where("id = ? AND user_id = ?", storageID, userID).First(&storage).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "储能设备不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	dischargeAmount := req.Amount / storage.Efficiency
	newSOC := storage.CurrentSOC - dischargeAmount

	if newSOC < 0 {
		newSOC = 0
	}

	storage.CurrentSOC = newSOC
	if err := db.Save(&storage).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "放电成功", "data": storage.ToResponse()})
}

type CreateVPPRequest struct {
	Name            string `json:"name" validate:"required,max=100"`
	Description     string `json:"description" validate:"max=500"`
	Type            string `json:"type" validate:"required,oneof=aggregate demand_response emergency"`
	ControlStrategy string `json:"control_strategy"`
}

type UpdateVPPRequest struct {
	Name            string `json:"name" validate:"omitempty,max=100"`
	Description     string `json:"description" validate:"omitempty,max=500"`
	ControlStrategy string `json:"control_strategy"`
}

type DispatchVPPRequest struct {
	TargetPower float64 `json:"target_power" validate:"required"`
	Duration    int     `json:"duration" validate:"required,min=1"`
}

func createVPPHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var req CreateVPPRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	vpp := VirtualPowerPlant{
		UserID:          userID,
		Name:            req.Name,
		Description:     req.Description,
		Type:            req.Type,
		ControlStrategy: req.ControlStrategy,
	}

	if err := db.Create(&vpp).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"code": 0, "message": "VPP创建成功", "data": vpp.ToResponse()})
}

func listVPPsHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	var vpps []VirtualPowerPlant

	if err := db.Where("user_id = ?", userID).Find(&vpps).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	response := make([]VPPResponse, len(vpps))
	for i, vpp := range vpps {
		response[i] = vpp.ToResponse()
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": response})
}

func getVPPHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	vppID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var vpp VirtualPowerPlant
	if err := db.Where("id = ? AND user_id = ?", vppID, userID).First(&vpp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "VPP不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "success", "data": vpp.ToResponse()})
}

func updateVPPHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	vppID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var vpp VirtualPowerPlant
	if err := db.Where("id = ? AND user_id = ?", vppID, userID).First(&vpp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "VPP不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	var req UpdateVPPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	if req.Name != "" {
		vpp.Name = req.Name
	}
	if req.Description != "" {
		vpp.Description = req.Description
	}
	if req.ControlStrategy != "" {
		vpp.ControlStrategy = req.ControlStrategy
	}

	if err := db.Save(&vpp).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "更新成功", "data": vpp.ToResponse()})
}

func deleteVPPHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	vppID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	if err := db.Where("id = ? AND user_id = ?", vppID, userID).Delete(&VirtualPowerPlant{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"code": 0, "message": "删除成功"})
}

func dispatchVPPHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	vppID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var vpp VirtualPowerPlant
	if err := db.Where("id = ? AND user_id = ?", vppID, userID).First(&vpp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "VPP不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	var req DispatchVPPRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败"})
		return
	}

	if err := validate.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数校验失败", "data": err.Error()})
		return
	}

	vpp.TotalCapacity = req.TargetPower
	if err := db.Save(&vpp).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "调度成功",
		"data": gin.H{
			"vpp_id":       vpp.ID,
			"target_power": req.TargetPower,
			"duration":     req.Duration,
			"status":       "dispatched",
		},
	})
}

func getVPPCapacityHandler(c *gin.Context) {
	userID := c.GetUint("user_id")
	vppID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"code": 1001, "message": "参数错误"})
		return
	}

	var vpp VirtualPowerPlant
	if err := db.Where("id = ? AND user_id = ?", vppID, userID).First(&vpp).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"code": 2001, "message": "VPP不存在"})
		} else {
			c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		}
		return
	}

	var powerSources []PowerSource
	if err := db.Where("user_id = ?", userID).Find(&powerSources).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	var storages []Storage
	if err := db.Where("user_id = ?", userID).Find(&storages).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"code": 3001, "message": "服务异常"})
		return
	}

	totalPower := 0.0
	for _, ps := range powerSources {
		totalPower += ps.OutputPower
	}

	totalStorage := 0.0
	for _, s := range storages {
		totalStorage += s.CurrentSOC
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"vpp_id":         vpp.ID,
			"vpp_name":       vpp.Name,
			"total_power":    totalPower,
			"total_storage":  totalStorage,
			"available_power": vpp.TotalCapacity,
			"power_sources":  len(powerSources),
			"storage_devices": len(storages),
		},
	})
}
