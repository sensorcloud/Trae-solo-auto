package handlers

import (
	"time"

	"github.com/edgehub/edgehub/internal/energy"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type EnergyHandler struct {
	powerSourceSvc energy.PowerSourceService
	storageSvc     energy.StorageService
	tradingSvc     energy.TradingService
	vppSvc         energy.VPPService
	loadSvc        energy.LoadService
	marketSvc      energy.EnergyMarketService
	coordSvc       energy.ComputeEnergyCoordinationService
}

func NewEnergyHandler(
	powerSourceSvc energy.PowerSourceService,
	storageSvc energy.StorageService,
	tradingSvc energy.TradingService,
	vppSvc energy.VPPService,
	loadSvc energy.LoadService,
	marketSvc energy.EnergyMarketService,
	coordSvc energy.ComputeEnergyCoordinationService,
) *EnergyHandler {
	return &EnergyHandler{
		powerSourceSvc: powerSourceSvc,
		storageSvc:     storageSvc,
		tradingSvc:     tradingSvc,
		vppSvc:         vppSvc,
		loadSvc:        loadSvc,
		marketSvc:      marketSvc,
		coordSvc:       coordSvc,
	}
}

// ==================== 电源管理 API ====================

// ListPowerSources 获取电源列表
// @Summary 获取电源列表
// @Description 获取所有电源设备列表，支持分页、过滤和排序
// @Tags 能源管理-电源
// @Accept json
// @Produce json
// @Param type query string false "电源类型 (solar, wind, hydro, biomass, grid, storage, generator)"
// @Param status query string false "电源状态 (online, offline, maintenance, fault)"
// @Param region query string false "区域"
// @Param min_capacity query number false "最小容量(kW)"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param sort_by query string false "排序字段"
// @Param order query string false "排序方向 (asc, desc)" default(desc)
// @Success 200 {object} PagedResponse
// @Router /api/v1/energy/power-sources [get]
func (h *EnergyHandler) ListPowerSources(c *gin.Context) {
	pagination := GetPagination(c)

	filter := &energy.PowerSourceFilter{
		Type:        energy.PowerSourceType(c.Query("type")),
		Status:      energy.PowerSourceStatus(c.Query("status")),
		Region:      c.Query("region"),
		MinCapacity: parseFloatDefault(c.Query("min_capacity"), 0),
		TenantID:    GetTenantID(c),
	}

	sources, err := h.powerSourceSvc.ListPowerSources(c.Request.Context(), filter)
	if err != nil {
		InternalError(c, "获取电源列表失败: "+err.Error())
		return
	}

	PagedSuccess(c, sources, int64(len(sources)), pagination.Page, pagination.PageSize)
}

// CreatePowerSource 创建电源
// @Summary 创建电源
// @Description 创建新的电源设备
// @Tags 能源管理-电源
// @Accept json
// @Produce json
// @Param power_source body energy.PowerSource true "电源信息"
// @Success 201 {object} Response
// @Router /api/v1/energy/power-sources [post]
func (h *EnergyHandler) CreatePowerSource(c *gin.Context) {
	var source energy.PowerSource
	if err := c.ShouldBindJSON(&source); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	source.TenantID = GetTenantID(c)
	if source.ID == uuid.Nil {
		source.ID = uuid.New()
	}

	if err := h.powerSourceSvc.CreatePowerSource(c.Request.Context(), &source); err != nil {
		InternalError(c, "创建电源失败: "+err.Error())
		return
	}

	Created(c, source)
}

// GetPowerSource 获取电源详情
// @Summary 获取电源详情
// @Description 根据ID获取电源详细信息
// @Tags 能源管理-电源
// @Accept json
// @Produce json
// @Param id path string true "电源ID"
// @Success 200 {object} Response
// @Router /api/v1/energy/power-sources/{id} [get]
func (h *EnergyHandler) GetPowerSource(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的电源ID")
		return
	}

	source, err := h.powerSourceSvc.GetPowerSource(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "电源不存在")
		return
	}

	Success(c, source)
}

// UpdatePowerSource 更新电源
// @Summary 更新电源
// @Description 更新电源设备信息
// @Tags 能源管理-电源
// @Accept json
// @Produce json
// @Param id path string true "电源ID"
// @Param updates body energy.PowerSourceUpdate true "更新信息"
// @Success 200 {object} Response
// @Router /api/v1/energy/power-sources/{id} [put]
func (h *EnergyHandler) UpdatePowerSource(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的电源ID")
		return
	}

	var updates energy.PowerSourceUpdate
	if err := c.ShouldBindJSON(&updates); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	if err := h.powerSourceSvc.UpdatePowerSource(c.Request.Context(), id, &updates); err != nil {
		InternalError(c, "更新电源失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "电源更新成功", nil)
}

// DeletePowerSource 删除电源
// @Summary 删除电源
// @Description 删除电源设备
// @Tags 能源管理-电源
// @Accept json
// @Produce json
// @Param id path string true "电源ID"
// @Success 204 "无内容"
// @Router /api/v1/energy/power-sources/{id} [delete]
func (h *EnergyHandler) DeletePowerSource(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的电源ID")
		return
	}

	if err := h.powerSourceSvc.DeletePowerSource(c.Request.Context(), id); err != nil {
		InternalError(c, "删除电源失败: "+err.Error())
		return
	}

	NoContent(c)
}

// GetPowerGenerationStats 获取发电统计
// @Summary 获取发电统计
// @Description 获取电源发电统计数据
// @Tags 能源管理-电源
// @Accept json
// @Produce json
// @Param id path string true "电源ID"
// @Param period query string false "统计周期 (day, week, month, year)" default(day)
// @Success 200 {object} Response
// @Router /api/v1/energy/power-sources/{id}/stats [get]
func (h *EnergyHandler) GetPowerGenerationStats(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的电源ID")
		return
	}

	period := c.DefaultQuery("period", "day")

	stats, err := h.powerSourceSvc.GetPowerGenerationStats(c.Request.Context(), id, period)
	if err != nil {
		InternalError(c, "获取发电统计失败: "+err.Error())
		return
	}

	Success(c, stats)
}

// ==================== 储能管理 API ====================

// ListStorageDevices 获取储能设备列表
// @Summary 获取储能设备列表
// @Description 获取所有储能设备列表，支持分页和过滤
// @Tags 能源管理-储能
// @Accept json
// @Produce json
// @Param status query string false "设备状态 (idle, charging, discharging, maintenance, fault)"
// @Param region query string false "区域"
// @Param min_capacity query number false "最小容量(kWh)"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} PagedResponse
// @Router /api/v1/storage/devices [get]
func (h *EnergyHandler) ListStorageDevices(c *gin.Context) {
	pagination := GetPagination(c)

	filter := &energy.StorageFilter{
		Status:      energy.StorageDeviceStatus(c.Query("status")),
		Region:      c.Query("region"),
		MinCapacity: parseFloatDefault(c.Query("min_capacity"), 0),
		TenantID:    GetTenantID(c),
	}

	devices, err := h.storageSvc.ListStorageDevices(c.Request.Context(), filter)
	if err != nil {
		InternalError(c, "获取储能设备列表失败: "+err.Error())
		return
	}

	PagedSuccess(c, devices, int64(len(devices)), pagination.Page, pagination.PageSize)
}

// CreateStorageDevice 创建储能设备
// @Summary 创建储能设备
// @Description 创建新的储能设备
// @Tags 能源管理-储能
// @Accept json
// @Produce json
// @Param device body energy.StorageDevice true "储能设备信息"
// @Success 201 {object} Response
// @Router /api/v1/storage/devices [post]
func (h *EnergyHandler) CreateStorageDevice(c *gin.Context) {
	var device energy.StorageDevice
	if err := c.ShouldBindJSON(&device); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	device.TenantID = GetTenantID(c)
	if device.ID == uuid.Nil {
		device.ID = uuid.New()
	}

	if err := h.storageSvc.CreateStorageDevice(c.Request.Context(), &device); err != nil {
		InternalError(c, "创建储能设备失败: "+err.Error())
		return
	}

	Created(c, device)
}

// GetStorageDevice 获取储能设备详情
// @Summary 获取储能设备详情
// @Description 根据ID获取储能设备详细信息
// @Tags 能源管理-储能
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Success 200 {object} Response
// @Router /api/v1/storage/devices/{id} [get]
func (h *EnergyHandler) GetStorageDevice(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的设备ID")
		return
	}

	device, err := h.storageSvc.GetStorageDevice(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "储能设备不存在")
		return
	}

	Success(c, device)
}

// UpdateStorageDevice 更新储能设备
// @Summary 更新储能设备
// @Description 更新储能设备信息
// @Tags 能源管理-储能
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Param updates body energy.StorageUpdate true "更新信息"
// @Success 200 {object} Response
// @Router /api/v1/storage/devices/{id} [put]
func (h *EnergyHandler) UpdateStorageDevice(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的设备ID")
		return
	}

	var updates energy.StorageUpdate
	if err := c.ShouldBindJSON(&updates); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	if err := h.storageSvc.UpdateStorageDevice(c.Request.Context(), id, &updates); err != nil {
		InternalError(c, "更新储能设备失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "储能设备更新成功", nil)
}

// DeleteStorageDevice 删除储能设备
// @Summary 删除储能设备
// @Description 删除储能设备
// @Tags 能源管理-储能
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Success 204 "无内容"
// @Router /api/v1/storage/devices/{id} [delete]
func (h *EnergyHandler) DeleteStorageDevice(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的设备ID")
		return
	}

	if err := h.storageSvc.DeleteStorageDevice(c.Request.Context(), id); err != nil {
		InternalError(c, "删除储能设备失败: "+err.Error())
		return
	}

	NoContent(c)
}

// ChargeStorage 充电操作
// @Summary 储能设备充电
// @Description 控制储能设备进行充电
// @Tags 能源管理-储能
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Param power query number true "充电功率(kW)"
// @Success 200 {object} Response
// @Router /api/v1/storage/devices/{id}/charge [post]
func (h *EnergyHandler) ChargeStorage(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的设备ID")
		return
	}

	power := parseFloatDefault(c.Query("power"), 0)
	if power <= 0 {
		BadRequest(c, "充电功率必须大于0")
		return
	}

	if err := h.storageSvc.Charge(c.Request.Context(), id, power); err != nil {
		InternalError(c, "充电操作失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "充电操作已执行", nil)
}

// DischargeStorage 放电操作
// @Summary 储能设备放电
// @Description 控制储能设备进行放电
// @Tags 能源管理-储能
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Param power query number true "放电功率(kW)"
// @Success 200 {object} Response
// @Router /api/v1/storage/devices/{id}/discharge [post]
func (h *EnergyHandler) DischargeStorage(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的设备ID")
		return
	}

	power := parseFloatDefault(c.Query("power"), 0)
	if power <= 0 {
		BadRequest(c, "放电功率必须大于0")
		return
	}

	if err := h.storageSvc.Discharge(c.Request.Context(), id, power); err != nil {
		InternalError(c, "放电操作失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "放电操作已执行", nil)
}

// StopStorage 停止充放电
// @Summary 停止储能设备充放电
// @Description 停止储能设备的充放电操作
// @Tags 能源管理-储能
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Success 200 {object} Response
// @Router /api/v1/storage/devices/{id}/stop [post]
func (h *EnergyHandler) StopStorage(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的设备ID")
		return
	}

	if err := h.storageSvc.StopChargeDischarge(c.Request.Context(), id); err != nil {
		InternalError(c, "停止操作失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "充放电已停止", nil)
}

// GetStorageStatus 获取储能状态
// @Summary 获取储能设备状态
// @Description 获取储能设备实时状态
// @Tags 能源管理-储能
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Success 200 {object} Response
// @Router /api/v1/storage/devices/{id}/status [get]
func (h *EnergyHandler) GetStorageStatus(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的设备ID")
		return
	}

	status, err := h.storageSvc.GetStorageStatus(c.Request.Context(), id)
	if err != nil {
		InternalError(c, "获取储能状态失败: "+err.Error())
		return
	}

	Success(c, status)
}

// OptimizeStorageSchedule 优化储能调度
// @Summary 优化储能调度计划
// @Description 基于电价预测优化储能设备的充放电计划
// @Tags 能源管理-储能
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Success 200 {object} Response
// @Router /api/v1/storage/devices/{id}/optimize [post]
func (h *EnergyHandler) OptimizeStorageSchedule(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的设备ID")
		return
	}

	result, err := h.storageSvc.OptimizeSchedule(c.Request.Context(), id, nil)
	if err != nil {
		InternalError(c, "优化调度失败: "+err.Error())
		return
	}

	Success(c, result)
}

// ==================== 电力交易 API ====================

// ListOrders 获取订单列表
// @Summary 获取电力订单列表
// @Description 获取电力交易订单列表，支持分页和过滤
// @Tags 能源管理-交易
// @Accept json
// @Produce json
// @Param type query string false "订单类型 (buy, sell)"
// @Param energy_type query string false "能源类型 (spot, forward, green_cert)"
// @Param status query string false "订单状态 (pending, submitted, filled, partial, cancelled, expired)"
// @Param is_green query bool false "是否绿电"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} PagedResponse
// @Router /api/v1/trading/orders [get]
func (h *EnergyHandler) ListOrders(c *gin.Context) {
	pagination := GetPagination(c)

	isGreen := c.Query("is_green")
	var isGreenPtr *bool
	if isGreen != "" {
		v := ParseBool(isGreen)
		isGreenPtr = &v
	}

	filter := &energy.OrderFilter{
		Type:       energy.OrderType(c.Query("type")),
		EnergyType: energy.EnergyOrderType(c.Query("energy_type")),
		Status:     energy.OrderStatus(c.Query("status")),
		BuyerID:    GetTenantID(c),
		SellerID:   GetTenantID(c),
		TenantID:   GetTenantID(c),
		Region:     c.Query("region"),
		IsGreen:    isGreenPtr,
	}

	orders, err := h.tradingSvc.ListOrders(c.Request.Context(), filter)
	if err != nil {
		InternalError(c, "获取订单列表失败: "+err.Error())
		return
	}

	PagedSuccess(c, orders, int64(len(orders)), pagination.Page, pagination.PageSize)
}

// CreateOrder 创建订单
// @Summary 创建电力交易订单
// @Description 创建新的电力交易订单
// @Tags 能源管理-交易
// @Accept json
// @Produce json
// @Param order body energy.EnergyOrder true "订单信息"
// @Success 201 {object} Response
// @Router /api/v1/trading/orders [post]
func (h *EnergyHandler) CreateOrder(c *gin.Context) {
	var order energy.EnergyOrder
	if err := c.ShouldBindJSON(&order); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	order.TenantID = GetTenantID(c)
	if order.ID == uuid.Nil {
		order.ID = uuid.New()
	}
	order.OrderNo = generateOrderNo()

	if err := h.tradingSvc.CreateOrder(c.Request.Context(), &order); err != nil {
		InternalError(c, "创建订单失败: "+err.Error())
		return
	}

	Created(c, order)
}

// GetOrder 获取订单详情
// @Summary 获取订单详情
// @Description 根据ID获取订单详细信息
// @Tags 能源管理-交易
// @Accept json
// @Produce json
// @Param id path string true "订单ID"
// @Success 200 {object} Response
// @Router /api/v1/trading/orders/{id} [get]
func (h *EnergyHandler) GetOrder(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的订单ID")
		return
	}

	order, err := h.tradingSvc.GetOrder(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "订单不存在")
		return
	}

	Success(c, order)
}

// CancelOrder 取消订单
// @Summary 取消订单
// @Description 取消电力交易订单
// @Tags 能源管理-交易
// @Accept json
// @Produce json
// @Param id path string true "订单ID"
// @Param reason query string false "取消原因"
// @Success 200 {object} Response
// @Router /api/v1/trading/orders/{id}/cancel [post]
func (h *EnergyHandler) CancelOrder(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的订单ID")
		return
	}

	reason := c.Query("reason")

	if err := h.tradingSvc.CancelOrder(c.Request.Context(), id, reason); err != nil {
		InternalError(c, "取消订单失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "订单已取消", nil)
}

// SubmitOrder 提交订单
// @Summary 提交订单到市场
// @Description 将订单提交到电力市场进行撮合
// @Tags 能源管理-交易
// @Accept json
// @Produce json
// @Param id path string true "订单ID"
// @Success 200 {object} Response
// @Router /api/v1/trading/orders/{id}/submit [post]
func (h *EnergyHandler) SubmitOrder(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的订单ID")
		return
	}

	if err := h.tradingSvc.SubmitOrder(c.Request.Context(), id); err != nil {
		InternalError(c, "提交订单失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "订单已提交", nil)
}

// GetPriceQuote 获取电价报价
// @Summary 获取电价报价
// @Description 获取指定区域的电价报价信息
// @Tags 能源管理-交易
// @Accept json
// @Produce json
// @Param region query string true "区域"
// @Param energy_type query string false "能源类型" default(spot)
// @Success 200 {object} Response
// @Router /api/v1/trading/prices [get]
func (h *EnergyHandler) GetPriceQuote(c *gin.Context) {
	region := c.Query("region")
	if region == "" {
		BadRequest(c, "区域参数不能为空")
		return
	}

	energyType := energy.EnergyOrderType(c.DefaultQuery("energy_type", "spot"))

	quote, err := h.tradingSvc.GetPriceQuote(c.Request.Context(), region, energyType)
	if err != nil {
		InternalError(c, "获取电价报价失败: "+err.Error())
		return
	}

	Success(c, quote)
}

// GetPriceHistory 获取电价历史
// @Summary 获取电价历史
// @Description 获取指定区域的电价历史数据
// @Tags 能源管理-交易
// @Accept json
// @Produce json
// @Param region query string true "区域"
// @Param energy_type query string false "能源类型" default(spot)
// @Param period query string false "时间范围 (day, week, month)" default(day)
// @Success 200 {object} Response
// @Router /api/v1/trading/prices/history [get]
func (h *EnergyHandler) GetPriceHistory(c *gin.Context) {
	region := c.Query("region")
	if region == "" {
		BadRequest(c, "区域参数不能为空")
		return
	}

	energyType := energy.EnergyOrderType(c.DefaultQuery("energy_type", "spot"))
	period := c.DefaultQuery("period", "day")

	history, err := h.tradingSvc.GetPriceHistory(c.Request.Context(), region, energyType, period)
	if err != nil {
		InternalError(c, "获取电价历史失败: "+err.Error())
		return
	}

	Success(c, history)
}

// ListGreenCertificates 获取绿证列表
// @Summary 获取绿色证书列表
// @Description 获取绿色电力证书列表
// @Tags 能源管理-交易
// @Accept json
// @Produce json
// @Param source_type query string false "能源类型 (solar, wind, hydro, biomass)"
// @Param status query string false "证书状态"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} PagedResponse
// @Router /api/v1/trading/green-certificates [get]
func (h *EnergyHandler) ListGreenCertificates(c *gin.Context) {
	pagination := GetPagination(c)

	filter := &energy.GreenCertFilter{
		SourceType: energy.PowerSourceType(c.Query("source_type")),
		OwnerID:    GetTenantID(c),
		Status:     c.Query("status"),
		TenantID:   GetTenantID(c),
	}

	certs, err := h.tradingSvc.ListGreenCertificates(c.Request.Context(), filter)
	if err != nil {
		InternalError(c, "获取绿证列表失败: "+err.Error())
		return
	}

	PagedSuccess(c, certs, int64(len(certs)), pagination.Page, pagination.PageSize)
}

// TransferGreenCertificate 转让绿证
// @Summary 转让绿色证书
// @Description 将绿色证书转让给其他用户
// @Tags 能源管理-交易
// @Accept json
// @Produce json
// @Param id path string true "证书ID"
// @Param to_owner_id query string true "目标用户ID"
// @Success 200 {object} Response
// @Router /api/v1/trading/green-certificates/{id}/transfer [post]
func (h *EnergyHandler) TransferGreenCertificate(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的证书ID")
		return
	}

	toOwnerID := ParseUUID(c.Query("to_owner_id"))
	if toOwnerID == uuid.Nil {
		BadRequest(c, "目标用户ID不能为空")
		return
	}

	if err := h.tradingSvc.TransferGreenCertificate(c.Request.Context(), id, toOwnerID); err != nil {
		InternalError(c, "转让绿证失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "绿证转让成功", nil)
}

// ==================== 虚拟电厂 API ====================

// ListVPPs 获取虚拟电厂列表
// @Summary 获取虚拟电厂列表
// @Description 获取所有虚拟电厂列表
// @Tags 能源管理-VPP
// @Accept json
// @Produce json
// @Param type query string false "VPP类型 (distributed, centralized, hybrid)"
// @Param status query string false "VPP状态 (active, inactive, dispatching)"
// @Param region query string false "区域"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} PagedResponse
// @Router /api/v1/vpp [get]
func (h *EnergyHandler) ListVPPs(c *gin.Context) {
	pagination := GetPagination(c)

	filter := &energy.VPPFilter{
		Type:     energy.VPPTypes(c.Query("type")),
		Status:   energy.VPPStatus(c.Query("status")),
		Region:   c.Query("region"),
		TenantID: GetTenantID(c),
	}

	vpps, err := h.vppSvc.ListVPPs(c.Request.Context(), filter)
	if err != nil {
		InternalError(c, "获取VPP列表失败: "+err.Error())
		return
	}

	PagedSuccess(c, vpps, int64(len(vpps)), pagination.Page, pagination.PageSize)
}

// CreateVPP 创建虚拟电厂
// @Summary 创建虚拟电厂
// @Description 创建新的虚拟电厂
// @Tags 能源管理-VPP
// @Accept json
// @Produce json
// @Param vpp body energy.VirtualPowerPlant true "VPP信息"
// @Success 201 {object} Response
// @Router /api/v1/vpp [post]
func (h *EnergyHandler) CreateVPP(c *gin.Context) {
	var vpp energy.VirtualPowerPlant
	if err := c.ShouldBindJSON(&vpp); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	vpp.TenantID = GetTenantID(c)
	if vpp.ID == uuid.Nil {
		vpp.ID = uuid.New()
	}

	if err := h.vppSvc.CreateVPP(c.Request.Context(), &vpp); err != nil {
		InternalError(c, "创建VPP失败: "+err.Error())
		return
	}

	Created(c, vpp)
}

// GetVPP 获取虚拟电厂详情
// @Summary 获取虚拟电厂详情
// @Description 根据ID获取虚拟电厂详细信息
// @Tags 能源管理-VPP
// @Accept json
// @Produce json
// @Param id path string true "VPP ID"
// @Success 200 {object} Response
// @Router /api/v1/vpp/{id} [get]
func (h *EnergyHandler) GetVPP(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的VPP ID")
		return
	}

	vpp, err := h.vppSvc.GetVPP(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "虚拟电厂不存在")
		return
	}

	Success(c, vpp)
}

// UpdateVPP 更新虚拟电厂
// @Summary 更新虚拟电厂
// @Description 更新虚拟电厂信息
// @Tags 能源管理-VPP
// @Accept json
// @Produce json
// @Param id path string true "VPP ID"
// @Param updates body energy.VPPUpdate true "更新信息"
// @Success 200 {object} Response
// @Router /api/v1/vpp/{id} [put]
func (h *EnergyHandler) UpdateVPP(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的VPP ID")
		return
	}

	var updates energy.VPPUpdate
	if err := c.ShouldBindJSON(&updates); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	if err := h.vppSvc.UpdateVPP(c.Request.Context(), id, &updates); err != nil {
		InternalError(c, "更新VPP失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "VPP更新成功", nil)
}

// DeleteVPP 删除虚拟电厂
// @Summary 删除虚拟电厂
// @Description 删除虚拟电厂
// @Tags 能源管理-VPP
// @Accept json
// @Produce json
// @Param id path string true "VPP ID"
// @Success 204 "无内容"
// @Router /api/v1/vpp/{id} [delete]
func (h *EnergyHandler) DeleteVPP(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的VPP ID")
		return
	}

	if err := h.vppSvc.DeleteVPP(c.Request.Context(), id); err != nil {
		InternalError(c, "删除VPP失败: "+err.Error())
		return
	}

	NoContent(c)
}

// DispatchVPP VPP调度
// @Summary 虚拟电厂调度
// @Description 对虚拟电厂进行调度操作
// @Tags 能源管理-VPP
// @Accept json
// @Produce json
// @Param id path string true "VPP ID"
// @Param request body energy.DispatchRequest true "调度请求"
// @Success 200 {object} Response
// @Router /api/v1/vpp/{id}/dispatch [post]
func (h *EnergyHandler) DispatchVPP(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的VPP ID")
		return
	}

	var request energy.DispatchRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	result, err := h.vppSvc.Dispatch(c.Request.Context(), id, &request)
	if err != nil {
		InternalError(c, "调度失败: "+err.Error())
		return
	}

	Success(c, result)
}

// GetVPPDispatchStatus 获取VPP调度状态
// @Summary 获取虚拟电厂调度状态
// @Description 获取虚拟电厂当前调度状态
// @Tags 能源管理-VPP
// @Accept json
// @Produce json
// @Param id path string true "VPP ID"
// @Success 200 {object} Response
// @Router /api/v1/vpp/{id}/dispatch-status [get]
func (h *EnergyHandler) GetVPPDispatchStatus(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的VPP ID")
		return
	}

	status, err := h.vppSvc.GetDispatchStatus(c.Request.Context(), id)
	if err != nil {
		InternalError(c, "获取调度状态失败: "+err.Error())
		return
	}

	Success(c, status)
}

// AggregateVPPCapacity 聚合VPP容量
// @Summary 聚合虚拟电厂容量
// @Description 获取虚拟电厂的聚合容量信息
// @Tags 能源管理-VPP
// @Accept json
// @Produce json
// @Param id path string true "VPP ID"
// @Success 200 {object} Response
// @Router /api/v1/vpp/{id}/capacity [get]
func (h *EnergyHandler) AggregateVPPCapacity(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的VPP ID")
		return
	}

	capacity, err := h.vppSvc.AggregateCapacity(c.Request.Context(), id)
	if err != nil {
		InternalError(c, "聚合容量失败: "+err.Error())
		return
	}

	Success(c, capacity)
}

// AddPowerSourceToVPP 添加电源到VPP
// @Summary 添加电源到虚拟电厂
// @Description 将电源添加到虚拟电厂
// @Tags 能源管理-VPP
// @Accept json
// @Produce json
// @Param id path string true "VPP ID"
// @Param source_id path string true "电源ID"
// @Success 200 {object} Response
// @Router /api/v1/vpp/{id}/power-sources/{source_id} [post]
func (h *EnergyHandler) AddPowerSourceToVPP(c *gin.Context) {
	vppID := ParseUUID(c.Param("id"))
	sourceID := ParseUUID(c.Param("source_id"))

	if vppID == uuid.Nil || sourceID == uuid.Nil {
		BadRequest(c, "无效的ID参数")
		return
	}

	if err := h.vppSvc.AddPowerSource(c.Request.Context(), vppID, sourceID); err != nil {
		InternalError(c, "添加电源失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "电源已添加到VPP", nil)
}

// AddStorageToVPP 添加储能到VPP
// @Summary 添加储能到虚拟电厂
// @Description 将储能设备添加到虚拟电厂
// @Tags 能源管理-VPP
// @Accept json
// @Produce json
// @Param id path string true "VPP ID"
// @Param storage_id path string true "储能ID"
// @Success 200 {object} Response
// @Router /api/v1/vpp/{id}/storages/{storage_id} [post]
func (h *EnergyHandler) AddStorageToVPP(c *gin.Context) {
	vppID := ParseUUID(c.Param("id"))
	storageID := ParseUUID(c.Param("storage_id"))

	if vppID == uuid.Nil || storageID == uuid.Nil {
		BadRequest(c, "无效的ID参数")
		return
	}

	if err := h.vppSvc.AddStorageDevice(c.Request.Context(), vppID, storageID); err != nil {
		InternalError(c, "添加储能失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "储能已添加到VPP", nil)
}

// ==================== 负荷管理 API ====================

// ListLoadProfiles 获取负荷配置列表
// @Summary 获取负荷配置列表
// @Description 获取所有负荷配置列表
// @Tags 能源管理-负荷
// @Accept json
// @Produce json
// @Param type query string false "负荷类型 (computing, cooling, lighting, other)"
// @Param cluster_id query string false "集群ID"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} PagedResponse
// @Router /api/v1/energy/loads [get]
func (h *EnergyHandler) ListLoadProfiles(c *gin.Context) {
	pagination := GetPagination(c)

	filter := &energy.LoadFilter{
		Type:      energy.LoadType(c.Query("type")),
		TenantID:  GetTenantID(c),
		ClusterID: ParseUUID(c.Query("cluster_id")),
	}

	profiles, err := h.loadSvc.ListLoadProfiles(c.Request.Context(), filter)
	if err != nil {
		InternalError(c, "获取负荷配置列表失败: "+err.Error())
		return
	}

	PagedSuccess(c, profiles, int64(len(profiles)), pagination.Page, pagination.PageSize)
}

// CreateLoadProfile 创建负荷配置
// @Summary 创建负荷配置
// @Description 创建新的负荷配置
// @Tags 能源管理-负荷
// @Accept json
// @Produce json
// @Param profile body energy.LoadProfile true "负荷配置信息"
// @Success 201 {object} Response
// @Router /api/v1/energy/loads [post]
func (h *EnergyHandler) CreateLoadProfile(c *gin.Context) {
	var profile energy.LoadProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	profile.TenantID = GetTenantID(c)
	if profile.ID == uuid.Nil {
		profile.ID = uuid.New()
	}

	if err := h.loadSvc.CreateLoadProfile(c.Request.Context(), &profile); err != nil {
		InternalError(c, "创建负荷配置失败: "+err.Error())
		return
	}

	Created(c, profile)
}

// GetLoadProfile 获取负荷配置详情
// @Summary 获取负荷配置详情
// @Description 根据ID获取负荷配置详细信息
// @Tags 能源管理-负荷
// @Accept json
// @Produce json
// @Param id path string true "负荷ID"
// @Success 200 {object} Response
// @Router /api/v1/energy/loads/{id} [get]
func (h *EnergyHandler) GetLoadProfile(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的负荷ID")
		return
	}

	profile, err := h.loadSvc.GetLoadProfile(c.Request.Context(), id)
	if err != nil {
		NotFound(c, "负荷配置不存在")
		return
	}

	Success(c, profile)
}

// ForecastLoad 负荷预测
// @Summary 负荷预测
// @Description 对负荷进行预测
// @Tags 能源管理-负荷
// @Accept json
// @Produce json
// @Param id path string true "负荷ID"
// @Param horizon query int false "预测时间范围(小时)" default(24)
// @Success 200 {object} Response
// @Router /api/v1/energy/loads/{id}/forecast [get]
func (h *EnergyHandler) ForecastLoad(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的负荷ID")
		return
	}

	horizon := parseIntDefault(c.Query("horizon"), 24)

	forecast, err := h.loadSvc.ForecastLoad(c.Request.Context(), id, horizon)
	if err != nil {
		InternalError(c, "负荷预测失败: "+err.Error())
		return
	}

	Success(c, forecast)
}

// AdjustLoad 调整负荷
// @Summary 调整负荷
// @Description 调整负荷到目标值
// @Tags 能源管理-负荷
// @Accept json
// @Produce json
// @Param id path string true "负荷ID"
// @Param target_load query number true "目标负荷(kW)"
// @Success 200 {object} Response
// @Router /api/v1/energy/loads/{id}/adjust [post]
func (h *EnergyHandler) AdjustLoad(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的负荷ID")
		return
	}

	targetLoad := parseFloatDefault(c.Query("target_load"), 0)
	if targetLoad <= 0 {
		BadRequest(c, "目标负荷必须大于0")
		return
	}

	if err := h.loadSvc.AdjustLoad(c.Request.Context(), id, targetLoad); err != nil {
		InternalError(c, "调整负荷失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "负荷调整已执行", nil)
}

// ==================== 市场概览 API ====================

// GetMarketOverview 获取市场概览
// @Summary 获取电力市场概览
// @Description 获取电力市场的整体情况
// @Tags 能源管理-市场
// @Accept json
// @Produce json
// @Param region query string true "区域"
// @Success 200 {object} Response
// @Router /api/v1/energy/market/overview [get]
func (h *EnergyHandler) GetMarketOverview(c *gin.Context) {
	region := c.Query("region")
	if region == "" {
		BadRequest(c, "区域参数不能为空")
		return
	}

	overview, err := h.marketSvc.GetMarketOverview(c.Request.Context(), region)
	if err != nil {
		InternalError(c, "获取市场概览失败: "+err.Error())
		return
	}

	Success(c, overview)
}

// GetTradingVolume 获取交易量
// @Summary 获取交易量统计
// @Description 获取电力市场的交易量统计
// @Tags 能源管理-市场
// @Accept json
// @Produce json
// @Param region query string true "区域"
// @Param period query string false "时间范围 (day, week, month)" default(day)
// @Success 200 {object} Response
// @Router /api/v1/energy/market/volume [get]
func (h *EnergyHandler) GetTradingVolume(c *gin.Context) {
	region := c.Query("region")
	if region == "" {
		BadRequest(c, "区域参数不能为空")
		return
	}

	period := c.DefaultQuery("period", "day")

	volume, err := h.marketSvc.GetTradingVolume(c.Request.Context(), region, period)
	if err != nil {
		InternalError(c, "获取交易量失败: "+err.Error())
		return
	}

	Success(c, volume)
}

// GetMarketDepth 获取市场深度
// @Summary 获取市场深度
// @Description 获取电力市场的买卖盘深度
// @Tags 能源管理-市场
// @Accept json
// @Produce json
// @Param region query string true "区域"
// @Param energy_type query string false "能源类型" default(spot)
// @Success 200 {object} Response
// @Router /api/v1/energy/market/depth [get]
func (h *EnergyHandler) GetMarketDepth(c *gin.Context) {
	region := c.Query("region")
	if region == "" {
		BadRequest(c, "区域参数不能为空")
		return
	}

	energyType := energy.EnergyOrderType(c.DefaultQuery("energy_type", "spot"))

	depth, err := h.marketSvc.GetMarketDepth(c.Request.Context(), region, energyType)
	if err != nil {
		InternalError(c, "获取市场深度失败: "+err.Error())
		return
	}

	Success(c, depth)
}

func generateOrderNo() string {
	return "ORD" + time.Now().Format("20060102150405") + uuid.New().String()[:8]
}
