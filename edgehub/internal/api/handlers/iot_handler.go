package handlers

import (
	"time"

	"github.com/edgehub/edgehub/internal/iot"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type IoTHandler struct {
	deviceMgr    *iot.DeviceManager
	telemetryMgr *iot.TelemetryManager
	connector    *iot.ConnectorManager
}

func NewIoTHandler(deviceMgr *iot.DeviceManager, telemetryMgr *iot.TelemetryManager, connector *iot.ConnectorManager) *IoTHandler {
	return &IoTHandler{
		deviceMgr:    deviceMgr,
		telemetryMgr: telemetryMgr,
		connector:    connector,
	}
}

// ==================== 设备管理 API ====================

// ListDevices 获取设备列表
// @Summary 获取设备列表
// @Description 获取所有IoT设备列表，支持分页、过滤和排序
// @Tags IoT设备管理
// @Accept json
// @Produce json
// @Param device_type query string false "设备类型 (sensor, actuator, gateway, controller, energy_meter, compute_node, smart_meter, inverter, ups, pdu)"
// @Param status query string false "设备状态 (online, offline, pending, inactive, error, maintain)"
// @Param protocol query string false "协议类型 (mqtt, modbus, opcua, http, coap)"
// @Param gateway_id query string false "网关ID"
// @Param enabled query bool false "是否启用"
// @Param search query string false "搜索关键词"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param sort_by query string false "排序字段"
// @Param order query string false "排序方向 (asc, desc)" default(desc)
// @Success 200 {object} PagedResponse
// @Router /api/v1/iot/devices [get]
func (h *IoTHandler) ListDevices(c *gin.Context) {
	pagination := GetPagination(c)
	sort := GetSortParams(c)

	enabledStr := c.Query("enabled")
	var enabled *bool
	if enabledStr != "" {
		v := ParseBool(enabledStr)
		enabled = &v
	}

	query := &iot.DeviceQuery{
		DeviceType: iot.DeviceType(c.Query("device_type")),
		Status:     iot.DeviceStatus(c.Query("status")),
		Protocol:   iot.ProtocolType(c.Query("protocol")),
		Enabled:    enabled,
		Search:     c.Query("search"),
		OrderBy:    sort.SortBy,
		Order:      sort.Order,
		Limit:      pagination.PageSize,
		Offset:     (pagination.Page - 1) * pagination.PageSize,
	}

	if gatewayID := c.Query("gateway_id"); gatewayID != "" {
		id := ParseUUID(gatewayID)
		query.GatewayID = &id
	}

	devices, err := h.deviceMgr.ListDevices(query)
	if err != nil {
		InternalError(c, "获取设备列表失败: "+err.Error())
		return
	}

	total, err := h.deviceMgr.CountDevices(query)
	if err != nil {
		InternalError(c, "统计设备数量失败: "+err.Error())
		return
	}

	PagedSuccess(c, devices, int64(total), pagination.Page, pagination.PageSize)
}

// CreateDevice 创建设备
// @Summary 创建设备
// @Description 创建新的IoT设备
// @Tags IoT设备管理
// @Accept json
// @Produce json
// @Param device body iot.Device true "设备信息"
// @Success 201 {object} Response
// @Router /api/v1/iot/devices [post]
func (h *IoTHandler) CreateDevice(c *gin.Context) {
	var device iot.Device
	if err := c.ShouldBindJSON(&device); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	device.TenantID = GetTenantID(c)
	if device.ID == uuid.Nil {
		device.ID = uuid.New()
	}

	if err := h.deviceMgr.RegisterDevice(&device); err != nil {
		InternalError(c, "创建设备失败: "+err.Error())
		return
	}

	Created(c, device)
}

// CreateDeviceBatch 批量创建设备
// @Summary 批量创建设备
// @Description 批量创建IoT设备
// @Tags IoT设备管理
// @Accept json
// @Produce json
// @Param devices body []iot.Device true "设备列表"
// @Success 201 {object} Response
// @Router /api/v1/iot/devices/batch [post]
func (h *IoTHandler) CreateDeviceBatch(c *gin.Context) {
	var devices []*iot.Device
	if err := c.ShouldBindJSON(&devices); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	tenantID := GetTenantID(c)
	for _, device := range devices {
		device.TenantID = tenantID
		if device.ID == uuid.Nil {
			device.ID = uuid.New()
		}
	}

	if err := h.deviceMgr.RegisterDeviceBatch(devices); err != nil {
		InternalError(c, "批量创建设备失败: "+err.Error())
		return
	}

	Created(c, gin.H{
		"count":   len(devices),
		"devices": devices,
	})
}

// GetDevice 获取设备详情
// @Summary 获取设备详情
// @Description 根据ID获取设备详细信息
// @Tags IoT设备管理
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Success 200 {object} Response
// @Router /api/v1/iot/devices/{id} [get]
func (h *IoTHandler) GetDevice(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的设备ID")
		return
	}

	device, err := h.deviceMgr.GetDevice(id)
	if err != nil {
		NotFound(c, "设备不存在")
		return
	}

	Success(c, device)
}

// UpdateDevice 更新设备
// @Summary 更新设备
// @Description 更新设备信息
// @Tags IoT设备管理
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Param updates body map[string]interface{} true "更新信息"
// @Success 200 {object} Response
// @Router /api/v1/iot/devices/{id} [put]
func (h *IoTHandler) UpdateDevice(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的设备ID")
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	if err := h.deviceMgr.UpdateDevice(id, updates); err != nil {
		InternalError(c, "更新设备失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "设备更新成功", nil)
}

// DeleteDevice 删除设备
// @Summary 删除设备
// @Description 删除设备
// @Tags IoT设备管理
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Success 204 "无内容"
// @Router /api/v1/iot/devices/{id} [delete]
func (h *IoTHandler) DeleteDevice(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的设备ID")
		return
	}

	if err := h.deviceMgr.UnregisterDevice(id); err != nil {
		InternalError(c, "删除设备失败: "+err.Error())
		return
	}

	NoContent(c)
}

// UpdateDeviceStatus 更新设备状态
// @Summary 更新设备状态
// @Description 更新设备在线状态
// @Tags IoT设备管理
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Param status query string true "设备状态"
// @Success 200 {object} Response
// @Router /api/v1/iot/devices/{id}/status [put]
func (h *IoTHandler) UpdateDeviceStatus(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的设备ID")
		return
	}

	status := iot.DeviceStatus(c.Query("status"))
	if status == "" {
		BadRequest(c, "状态参数不能为空")
		return
	}

	if err := h.deviceMgr.UpdateDeviceStatus(id, status); err != nil {
		InternalError(c, "更新设备状态失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "设备状态已更新", nil)
}

// EnableDevice 启用设备
// @Summary 启用设备
// @Description 启用设备
// @Tags IoT设备管理
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Success 200 {object} Response
// @Router /api/v1/iot/devices/{id}/enable [post]
func (h *IoTHandler) EnableDevice(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的设备ID")
		return
	}

	if err := h.deviceMgr.EnableDevice(id); err != nil {
		InternalError(c, "启用设备失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "设备已启用", nil)
}

// DisableDevice 禁用设备
// @Summary 禁用设备
// @Description 禁用设备
// @Tags IoT设备管理
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Success 200 {object} Response
// @Router /api/v1/iot/devices/{id}/disable [post]
func (h *IoTHandler) DisableDevice(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的设备ID")
		return
	}

	if err := h.deviceMgr.DisableDevice(id); err != nil {
		InternalError(c, "禁用设备失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "设备已禁用", nil)
}

// SetDeviceLabels 设置设备标签
// @Summary 设置设备标签
// @Description 设置设备的标签信息
// @Tags IoT设备管理
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Param labels body map[string]string true "标签"
// @Success 200 {object} Response
// @Router /api/v1/iot/devices/{id}/labels [put]
func (h *IoTHandler) SetDeviceLabels(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的设备ID")
		return
	}

	var labels iot.DeviceLabels
	if err := c.ShouldBindJSON(&labels); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	if err := h.deviceMgr.SetDeviceLabels(id, labels); err != nil {
		InternalError(c, "设置标签失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "标签已更新", nil)
}

// BindDeviceToGateway 绑定设备到网关
// @Summary 绑定设备到网关
// @Description 将设备绑定到指定网关
// @Tags IoT设备管理
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Param gateway_id path string true "网关ID"
// @Success 200 {object} Response
// @Router /api/v1/iot/devices/{id}/bind/{gateway_id} [post]
func (h *IoTHandler) BindDeviceToGateway(c *gin.Context) {
	deviceID := ParseUUID(c.Param("id"))
	gatewayID := ParseUUID(c.Param("gateway_id"))

	if deviceID == uuid.Nil || gatewayID == uuid.Nil {
		BadRequest(c, "无效的ID参数")
		return
	}

	if err := h.deviceMgr.BindDeviceToGateway(deviceID, gatewayID); err != nil {
		InternalError(c, "绑定设备失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "设备已绑定到网关", nil)
}

// UnbindDeviceFromGateway 解绑设备
// @Summary 解绑设备
// @Description 解除设备与网关的绑定
// @Tags IoT设备管理
// @Accept json
// @Produce json
// @Param id path string true "设备ID"
// @Success 200 {object} Response
// @Router /api/v1/iot/devices/{id}/unbind [post]
func (h *IoTHandler) UnbindDeviceFromGateway(c *gin.Context) {
	deviceID := ParseUUID(c.Param("id"))
	if deviceID == uuid.Nil {
		BadRequest(c, "无效的设备ID")
		return
	}

	if err := h.deviceMgr.UnbindDeviceFromGateway(deviceID); err != nil {
		InternalError(c, "解绑设备失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "设备已解绑", nil)
}

// GetOnlineDevices 获取在线设备
// @Summary 获取在线设备列表
// @Description 获取所有在线设备的ID列表
// @Tags IoT设备管理
// @Accept json
// @Produce json
// @Success 200 {object} Response
// @Router /api/v1/iot/devices/online [get]
func (h *IoTHandler) GetOnlineDevices(c *gin.Context) {
	devices := h.deviceMgr.GetOnlineDevices()
	count := h.deviceMgr.GetOnlineDeviceCount()

	Success(c, gin.H{
		"device_ids": devices,
		"count":      count,
	})
}

// ==================== 设备配置文件 API ====================

// ListDeviceProfiles 获取设备配置文件列表
// @Summary 获取设备配置文件列表
// @Description 获取所有设备配置文件列表
// @Tags IoT设备管理
// @Accept json
// @Produce json
// @Success 200 {object} Response
// @Router /api/v1/iot/profiles [get]
func (h *IoTHandler) ListDeviceProfiles(c *gin.Context) {
	profiles, err := h.deviceMgr.ListDeviceProfiles()
	if err != nil {
		InternalError(c, "获取配置文件列表失败: "+err.Error())
		return
	}

	Success(c, profiles)
}

// CreateDeviceProfile 创建设备配置文件
// @Summary 创建设备配置文件
// @Description 创建新的设备配置文件
// @Tags IoT设备管理
// @Accept json
// @Produce json
// @Param profile body iot.DeviceProfile true "配置文件信息"
// @Success 201 {object} Response
// @Router /api/v1/iot/profiles [post]
func (h *IoTHandler) CreateDeviceProfile(c *gin.Context) {
	var profile iot.DeviceProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	profile.TenantID = GetTenantID(c)
	if profile.ID == uuid.Nil {
		profile.ID = uuid.New()
	}

	if err := h.deviceMgr.CreateDeviceProfile(&profile); err != nil {
		InternalError(c, "创建配置文件失败: "+err.Error())
		return
	}

	Created(c, profile)
}

// GetDeviceProfile 获取设备配置文件详情
// @Summary 获取设备配置文件详情
// @Description 根据ID获取设备配置文件详细信息
// @Tags IoT设备管理
// @Accept json
// @Produce json
// @Param id path string true "配置文件ID"
// @Success 200 {object} Response
// @Router /api/v1/iot/profiles/{id} [get]
func (h *IoTHandler) GetDeviceProfile(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的配置文件ID")
		return
	}

	profile, err := h.deviceMgr.GetDeviceProfile(id)
	if err != nil {
		NotFound(c, "配置文件不存在")
		return
	}

	Success(c, profile)
}

// UpdateDeviceProfile 更新设备配置文件
// @Summary 更新设备配置文件
// @Description 更新设备配置文件信息
// @Tags IoT设备管理
// @Accept json
// @Produce json
// @Param id path string true "配置文件ID"
// @Param updates body map[string]interface{} true "更新信息"
// @Success 200 {object} Response
// @Router /api/v1/iot/profiles/{id} [put]
func (h *IoTHandler) UpdateDeviceProfile(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的配置文件ID")
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	if err := h.deviceMgr.UpdateDeviceProfile(id, updates); err != nil {
		InternalError(c, "更新配置文件失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "配置文件更新成功", nil)
}

// DeleteDeviceProfile 删除设备配置文件
// @Summary 删除设备配置文件
// @Description 删除设备配置文件
// @Tags IoT设备管理
// @Accept json
// @Produce json
// @Param id path string true "配置文件ID"
// @Success 204 "无内容"
// @Router /api/v1/iot/profiles/{id} [delete]
func (h *IoTHandler) DeleteDeviceProfile(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的配置文件ID")
		return
	}

	if err := h.deviceMgr.DeleteDeviceProfile(id); err != nil {
		InternalError(c, "删除配置文件失败: "+err.Error())
		return
	}

	NoContent(c)
}

// ==================== 遥测数据 API ====================

// GetTelemetry 获取遥测数据
// @Summary 获取遥测数据
// @Description 获取设备的遥测数据
// @Tags IoT遥测
// @Accept json
// @Produce json
// @Param device_id query string false "设备ID"
// @Param properties query string false "属性列表(逗号分隔)"
// @Param start_time query string false "开始时间(RFC3339)"
// @Param end_time query string false "结束时间(RFC3339)"
// @Param limit query int false "数量限制" default(100)
// @Param order query string false "排序方向 (asc, desc)" default(desc)
// @Success 200 {object} Response
// @Router /api/v1/iot/telemetry [get]
func (h *IoTHandler) GetTelemetry(c *gin.Context) {
	deviceID := ParseUUID(c.Query("device_id"))
	properties := splitString(c.Query("properties"), ",")
	timeRange := GetTimeRange(c)
	limit := parseIntDefault(c.Query("limit"), 100)
	order := c.DefaultQuery("order", "desc")

	query := &iot.TelemetryQuery{
		DeviceIDs:  nil,
		Properties: properties,
		StartTime:  timeRange.StartTime,
		EndTime:    timeRange.EndTime,
		Limit:      limit,
		Order:      order,
	}

	if deviceID != uuid.Nil {
		query.DeviceIDs = []uuid.UUID{deviceID}
	}

	result, err := h.telemetryMgr.QueryTelemetry(c.Request.Context(), query)
	if err != nil {
		InternalError(c, "获取遥测数据失败: "+err.Error())
		return
	}

	Success(c, result)
}

// GetTelemetryLatest 获取最新遥测数据
// @Summary 获取最新遥测数据
// @Description 获取设备的最新遥测数据
// @Tags IoT遥测
// @Accept json
// @Produce json
// @Param device_id path string true "设备ID"
// @Success 200 {object} Response
// @Router /api/v1/iot/telemetry/{device_id}/latest [get]
func (h *IoTHandler) GetTelemetryLatest(c *gin.Context) {
	deviceID := ParseUUID(c.Param("device_id"))
	if deviceID == uuid.Nil {
		BadRequest(c, "无效的设备ID")
		return
	}

	data, err := h.telemetryMgr.GetLatestTelemetry(c.Request.Context(), deviceID)
	if err != nil {
		InternalError(c, "获取最新遥测数据失败: "+err.Error())
		return
	}

	Success(c, data)
}

// GetTelemetryHistory 获取遥测历史数据
// @Summary 获取遥测历史数据
// @Description 获取设备的遥测历史数据
// @Tags IoT遥测
// @Accept json
// @Produce json
// @Param device_id path string true "设备ID"
// @Param property query string false "属性名称"
// @Param start_time query string false "开始时间(RFC3339)"
// @Param end_time query string false "结束时间(RFC3339)"
// @Param interval query int false "聚合间隔(秒)"
// @Param aggregate query string false "聚合函数 (avg, min, max, sum)"
// @Success 200 {object} Response
// @Router /api/v1/iot/telemetry/{device_id}/history [get]
func (h *IoTHandler) GetTelemetryHistory(c *gin.Context) {
	deviceID := ParseUUID(c.Param("device_id"))
	if deviceID == uuid.Nil {
		BadRequest(c, "无效的设备ID")
		return
	}

	property := c.Query("property")
	timeRange := GetTimeRange(c)
	interval := parseIntDefault(c.Query("interval"), 0)
	aggregate := c.Query("aggregate")

	query := &iot.TelemetryQuery{
		DeviceIDs:  []uuid.UUID{deviceID},
		Properties: []string{property},
		StartTime:  timeRange.StartTime,
		EndTime:    timeRange.EndTime,
		Interval:   interval,
		Aggregate:  aggregate,
		Limit:      1000,
	}

	result, err := h.telemetryMgr.QueryTelemetry(c.Request.Context(), query)
	if err != nil {
		InternalError(c, "获取遥测历史数据失败: "+err.Error())
		return
	}

	Success(c, result)
}

// SubmitTelemetry 提交遥测数据
// @Summary 提交遥测数据
// @Description 设备提交遥测数据
// @Tags IoT遥测
// @Accept json
// @Produce json
// @Param telemetry body iot.TelemetryBatch true "遥测数据"
// @Success 200 {object} Response
// @Router /api/v1/iot/telemetry [post]
func (h *IoTHandler) SubmitTelemetry(c *gin.Context) {
	var batch iot.TelemetryBatch
	if err := c.ShouldBindJSON(&batch); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	batch.TenantID = GetTenantID(c)
	batch.Timestamp = time.Now()

	if err := h.telemetryMgr.SubmitTelemetry(c.Request.Context(), &batch); err != nil {
		InternalError(c, "提交遥测数据失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "遥测数据已提交", nil)
}

// GetDeviceShadow 获取设备影子
// @Summary 获取设备影子
// @Description 获取设备的影子状态
// @Tags IoT遥测
// @Accept json
// @Produce json
// @Param device_id path string true "设备ID"
// @Success 200 {object} Response
// @Router /api/v1/iot/devices/{device_id}/shadow [get]
func (h *IoTHandler) GetDeviceShadow(c *gin.Context) {
	deviceID := ParseUUID(c.Param("device_id"))
	if deviceID == uuid.Nil {
		BadRequest(c, "无效的设备ID")
		return
	}

	shadow, err := h.telemetryMgr.GetDeviceShadow(c.Request.Context(), deviceID)
	if err != nil {
		InternalError(c, "获取设备影子失败: "+err.Error())
		return
	}

	Success(c, shadow)
}

// UpdateDeviceShadow 更新设备影子
// @Summary 更新设备影子
// @Description 更新设备的影子状态
// @Tags IoT遥测
// @Accept json
// @Produce json
// @Param device_id path string true "设备ID"
// @Param shadow body iot.DeviceShadow true "影子状态"
// @Success 200 {object} Response
// @Router /api/v1/iot/devices/{device_id}/shadow [put]
func (h *IoTHandler) UpdateDeviceShadow(c *gin.Context) {
	deviceID := ParseUUID(c.Param("device_id"))
	if deviceID == uuid.Nil {
		BadRequest(c, "无效的设备ID")
		return
	}

	var shadow iot.DeviceShadow
	if err := c.ShouldBindJSON(&shadow); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	shadow.DeviceID = deviceID

	if err := h.telemetryMgr.UpdateDeviceShadow(c.Request.Context(), &shadow); err != nil {
		InternalError(c, "更新设备影子失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "设备影子已更新", nil)
}

// ==================== 设备命令 API ====================

// ExecuteCommand 执行设备命令
// @Summary 执行设备命令
// @Description 向设备发送命令并执行
// @Tags IoT命令
// @Accept json
// @Produce json
// @Param command body iot.DeviceCommandRequest true "命令请求"
// @Success 200 {object} Response
// @Router /api/v1/iot/commands [post]
func (h *IoTHandler) ExecuteCommand(c *gin.Context) {
	var request iot.DeviceCommandRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		BadRequest(c, "无效的请求参数: "+err.Error())
		return
	}

	response, err := h.connector.ExecuteCommand(c.Request.Context(), &request)
	if err != nil {
		InternalError(c, "执行命令失败: "+err.Error())
		return
	}

	Success(c, response)
}

// GetCommandStatus 获取命令状态
// @Summary 获取命令执行状态
// @Description 获取异步命令的执行状态
// @Tags IoT命令
// @Accept json
// @Produce json
// @Param correlation_id path string true "关联ID"
// @Success 200 {object} Response
// @Router /api/v1/iot/commands/{correlation_id} [get]
func (h *IoTHandler) GetCommandStatus(c *gin.Context) {
	correlationID := c.Param("correlation_id")
	if correlationID == "" {
		BadRequest(c, "关联ID不能为空")
		return
	}

	response, err := h.connector.GetCommandStatus(c.Request.Context(), correlationID)
	if err != nil {
		NotFound(c, "命令不存在")
		return
	}

	Success(c, response)
}

// ==================== 连接器管理 API ====================

// GetConnectorStatus 获取连接器状态
// @Summary 获取连接器状态
// @Description 获取协议连接器的状态信息
// @Tags IoT连接器
// @Accept json
// @Produce json
// @Param protocol path string true "协议类型 (mqtt, modbus, opcua)"
// @Success 200 {object} Response
// @Router /api/v1/iot/connectors/{protocol}/status [get]
func (h *IoTHandler) GetConnectorStatus(c *gin.Context) {
	protocol := iot.ProtocolType(c.Param("protocol"))

	status, err := h.connector.GetStatus(c.Request.Context(), protocol)
	if err != nil {
		InternalError(c, "获取连接器状态失败: "+err.Error())
		return
	}

	Success(c, status)
}

// StartConnector 启动连接器
// @Summary 启动连接器
// @Description 启动协议连接器
// @Tags IoT连接器
// @Accept json
// @Produce json
// @Param protocol path string true "协议类型"
// @Success 200 {object} Response
// @Router /api/v1/iot/connectors/{protocol}/start [post]
func (h *IoTHandler) StartConnector(c *gin.Context) {
	protocol := iot.ProtocolType(c.Param("protocol"))

	if err := h.connector.Start(c.Request.Context(), protocol); err != nil {
		InternalError(c, "启动连接器失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "连接器已启动", nil)
}

// StopConnector 停止连接器
// @Summary 停止连接器
// @Description 停止协议连接器
// @Tags IoT连接器
// @Accept json
// @Produce json
// @Param protocol path string true "协议类型"
// @Success 200 {object} Response
// @Router /api/v1/iot/connectors/{protocol}/stop [post]
func (h *IoTHandler) StopConnector(c *gin.Context) {
	protocol := iot.ProtocolType(c.Param("protocol"))

	if err := h.connector.Stop(c.Request.Context(), protocol); err != nil {
		InternalError(c, "停止连接器失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "连接器已停止", nil)
}

// ==================== 告警管理 API ====================

// ListDeviceAlarms 获取设备告警列表
// @Summary 获取设备告警列表
// @Description 获取设备的告警记录
// @Tags IoT告警
// @Accept json
// @Produce json
// @Param device_id query string false "设备ID"
// @Param status query string false "告警状态 (active, cleared, acknowledged)"
// @Param severity query string false "严重程度"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} PagedResponse
// @Router /api/v1/iot/alarms [get]
func (h *IoTHandler) ListDeviceAlarms(c *gin.Context) {
	pagination := GetPagination(c)

	deviceID := ParseUUID(c.Query("device_id"))
	status := c.Query("status")
	severity := c.Query("severity")

	alarms, err := h.telemetryMgr.ListDeviceAlarms(c.Request.Context(), &iot.AlarmFilter{
		DeviceID: deviceID,
		Status:   status,
		Severity: severity,
		TenantID: GetTenantID(c),
		Limit:    pagination.PageSize,
		Offset:   (pagination.Page - 1) * pagination.PageSize,
	})
	if err != nil {
		InternalError(c, "获取告警列表失败: "+err.Error())
		return
	}

	PagedSuccess(c, alarms, int64(len(alarms)), pagination.Page, pagination.PageSize)
}

// AcknowledgeAlarm 确认告警
// @Summary 确认告警
// @Description 确认设备告警
// @Tags IoT告警
// @Accept json
// @Produce json
// @Param id path string true "告警ID"
// @Success 200 {object} Response
// @Router /api/v1/iot/alarms/{id}/acknowledge [post]
func (h *IoTHandler) AcknowledgeAlarm(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的告警ID")
		return
	}

	userID := GetUserID(c)
	if err := h.telemetryMgr.AcknowledgeAlarm(c.Request.Context(), id, userID); err != nil {
		InternalError(c, "确认告警失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "告警已确认", nil)
}

// ClearAlarm 清除告警
// @Summary 清除告警
// @Description 清除设备告警
// @Tags IoT告警
// @Accept json
// @Produce json
// @Param id path string true "告警ID"
// @Success 200 {object} Response
// @Router /api/v1/iot/alarms/{id}/clear [post]
func (h *IoTHandler) ClearAlarm(c *gin.Context) {
	id := ParseUUID(c.Param("id"))
	if id == uuid.Nil {
		BadRequest(c, "无效的告警ID")
		return
	}

	if err := h.telemetryMgr.ClearAlarm(c.Request.Context(), id); err != nil {
		InternalError(c, "清除告警失败: "+err.Error())
		return
	}

	SuccessWithMessage(c, "告警已清除", nil)
}
