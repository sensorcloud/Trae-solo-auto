package iot

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type DeviceQuery struct {
	IDs         []uuid.UUID
	Name        string
	DeviceType  DeviceType
	Status      DeviceStatus
	Protocol    ProtocolType
	ProfileID   *uuid.UUID
	ParentID    *uuid.UUID
	GatewayID   *uuid.UUID
	Labels      map[string]string
	Tags        []string
	Enabled     *bool
	Search      string
	OrderBy     string
	Order       string
	Limit       int
	Offset      int
}

type DeviceManager struct {
	db       *gorm.DB
	tenantID uuid.UUID

	deviceCache  map[uuid.UUID]*Device
	profileCache map[uuid.UUID]*DeviceProfile
	cacheMu      sync.RWMutex

	onlineDevices  map[uuid.UUID]time.Time
	onlineMu       sync.RWMutex

	eventHandlers  []DeviceEventHandler
	handlerMu      sync.RWMutex
}

type DeviceEventHandler func(event *DeviceEvent)

type DeviceEvent struct {
	Type      string
	DeviceID  uuid.UUID
	Timestamp time.Time
	Data      interface{}
}

func NewDeviceManager(db *gorm.DB, tenantID uuid.UUID) *DeviceManager {
	return &DeviceManager{
		db:            db,
		tenantID:      tenantID,
		deviceCache:   make(map[uuid.UUID]*Device),
		profileCache:  make(map[uuid.UUID]*DeviceProfile),
		onlineDevices: make(map[uuid.UUID]time.Time),
	}
}

func (m *DeviceManager) RegisterDevice(device *Device) error {
	if device.TenantID == uuid.Nil {
		device.TenantID = m.tenantID
	}

	if device.ID == uuid.Nil {
		device.ID = uuid.New()
	}

	if device.Status == "" {
		device.Status = DeviceStatusPending
	}

	if device.Labels == nil {
		device.Labels = make(DeviceLabels)
	}

	var existingDevice Device
	err := m.db.Where("tenant_id = ? AND name = ?", device.TenantID, device.Name).
		First(&existingDevice).Error
	if err == nil {
		return fmt.Errorf("设备名称已存在: %s", device.Name)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询设备失败: %w", err)
	}

	if device.ProfileID != nil {
		profile, err := m.GetDeviceProfile(*device.ProfileID)
		if err != nil {
			return fmt.Errorf("获取设备配置文件失败: %w", err)
		}
		if profile.TenantID != device.TenantID {
			return errors.New("设备配置文件不属于当前租户")
		}
	}

	if err := m.db.Create(device).Error; err != nil {
		return fmt.Errorf("创建设备失败: %w", err)
	}

	m.cacheDevice(device)

	m.emitEvent(&DeviceEvent{
		Type:      "device_registered",
		DeviceID:  device.ID,
		Timestamp: time.Now(),
		Data:      device,
	})

	log.Printf("[DeviceManager] 设备注册成功: %s (%s)", device.Name, device.ID)
	return nil
}

func (m *DeviceManager) RegisterDeviceBatch(devices []*Device) error {
	return m.db.Transaction(func(tx *gorm.DB) error {
		for _, device := range devices {
			if device.TenantID == uuid.Nil {
				device.TenantID = m.tenantID
			}
			if device.ID == uuid.Nil {
				device.ID = uuid.New()
			}
			if device.Status == "" {
				device.Status = DeviceStatusPending
			}
			if device.Labels == nil {
				device.Labels = make(DeviceLabels)
			}
		}

		if err := tx.CreateInBatches(devices, 100).Error; err != nil {
			return fmt.Errorf("批量创建设备失败: %w", err)
		}

		for _, device := range devices {
			m.cacheDevice(device)
			m.emitEvent(&DeviceEvent{
				Type:      "device_registered",
				DeviceID:  device.ID,
				Timestamp: time.Now(),
				Data:      device,
			})
		}

		log.Printf("[DeviceManager] 批量注册%d个设备成功", len(devices))
		return nil
	})
}

func (m *DeviceManager) UnregisterDevice(deviceID uuid.UUID) error {
	device, err := m.GetDevice(deviceID)
	if err != nil {
		return err
	}

	if err := m.db.Delete(&Device{}, "id = ?", deviceID).Error; err != nil {
		return fmt.Errorf("删除设备失败: %w", err)
	}

	m.removeFromCache(deviceID)
	m.removeFromOnline(deviceID)

	m.emitEvent(&DeviceEvent{
		Type:      "device_unregistered",
		DeviceID:  deviceID,
		Timestamp: time.Now(),
		Data:      device,
	})

	log.Printf("[DeviceManager] 设备注销成功: %s", device.Name)
	return nil
}

func (m *DeviceManager) GetDevice(deviceID uuid.UUID) (*Device, error) {
	if device := m.getFromCache(deviceID); device != nil {
		return device, nil
	}

	var device Device
	err := m.db.Where("id = ? AND tenant_id = ?", deviceID, m.tenantID).
		Preload("Profile").First(&device).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("设备不存在: %s", deviceID)
		}
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}

	m.cacheDevice(&device)
	return &device, nil
}

func (m *DeviceManager) GetDeviceByName(name string) (*Device, error) {
	var device Device
	err := m.db.Where("tenant_id = ? AND name = ?", m.tenantID, name).
		First(&device).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("设备不存在: %s", name)
		}
		return nil, fmt.Errorf("查询设备失败: %w", err)
	}

	return &device, nil
}

func (m *DeviceManager) ListDevices(query *DeviceQuery) ([]*Device, error) {
	db := m.db.Model(&Device{}).Where("tenant_id = ?", m.tenantID)

	if len(query.IDs) > 0 {
		db = db.Where("id IN ?", query.IDs)
	}
	if query.Name != "" {
		db = db.Where("name = ?", query.Name)
	}
	if query.DeviceType != "" {
		db = db.Where("device_type = ?", query.DeviceType)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.Protocol != "" {
		db = db.Where("protocol = ?", query.Protocol)
	}
	if query.ProfileID != nil {
		db = db.Where("profile_id = ?", *query.ProfileID)
	}
	if query.ParentID != nil {
		db = db.Where("parent_id = ?", *query.ParentID)
	}
	if query.GatewayID != nil {
		db = db.Where("gateway_id = ?", *query.GatewayID)
	}
	if query.Enabled != nil {
		db = db.Where("enabled = ?", *query.Enabled)
	}
	if query.Search != "" {
		search := "%" + query.Search + "%"
		db = db.Where("name LIKE ? OR display_name LIKE ? OR description LIKE ?",
			search, search, search)
	}
	if len(query.Labels) > 0 {
		for k, v := range query.Labels {
			db = db.Where("labels->>? = ?", k, v)
		}
	}

	if query.OrderBy != "" {
		order := "ASC"
		if query.Order != "" {
			order = query.Order
		}
		db = db.Order(fmt.Sprintf("%s %s", query.OrderBy, order))
	} else {
		db = db.Order("created_at DESC")
	}

	if query.Limit > 0 {
		db = db.Limit(query.Limit)
	}
	if query.Offset > 0 {
		db = db.Offset(query.Offset)
	}

	var devices []*Device
	if err := db.Find(&devices).Error; err != nil {
		return nil, fmt.Errorf("查询设备列表失败: %w", err)
	}

	return devices, nil
}

func (m *DeviceManager) CountDevices(query *DeviceQuery) (int, error) {
	db := m.db.Model(&Device{}).Where("tenant_id = ?", m.tenantID)

	if query.DeviceType != "" {
		db = db.Where("device_type = ?", query.DeviceType)
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.Protocol != "" {
		db = db.Where("protocol = ?", query.Protocol)
	}
	if query.Enabled != nil {
		db = db.Where("enabled = ?", *query.Enabled)
	}

	var count int64
	if err := db.Count(&count).Error; err != nil {
		return 0, fmt.Errorf("统计设备数量失败: %w", err)
	}

	return int(count), nil
}

func (m *DeviceManager) UpdateDevice(deviceID uuid.UUID, updates map[string]interface{}) error {
	result := m.db.Model(&Device{}).
		Where("id = ? AND tenant_id = ?", deviceID, m.tenantID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("更新设备失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("设备不存在: %s", deviceID)
	}

	m.invalidateCache(deviceID)

	m.emitEvent(&DeviceEvent{
		Type:      "device_updated",
		DeviceID:  deviceID,
		Timestamp: time.Now(),
		Data:      updates,
	})

	return nil
}

func (m *DeviceManager) UpdateDeviceStatus(deviceID uuid.UUID, status DeviceStatus) error {
	updates := map[string]interface{}{
		"status": status,
	}

	now := time.Now()
	switch status {
	case DeviceStatusOnline:
		updates["last_online_at"] = now
		m.addToOnline(deviceID)
	case DeviceStatusOffline:
		updates["last_offline_at"] = now
		m.removeFromOnline(deviceID)
	}

	if err := m.UpdateDevice(deviceID, updates); err != nil {
		return err
	}

	m.emitEvent(&DeviceEvent{
		Type:      "device_status_changed",
		DeviceID:  deviceID,
		Timestamp: now,
		Data:      status,
	})

	return nil
}

func (m *DeviceManager) UpdateHeartbeat(deviceID uuid.UUID) error {
	now := time.Now()
	result := m.db.Model(&Device{}).
		Where("id = ? AND tenant_id = ?", deviceID, m.tenantID).
		Update("last_heartbeat_at", now)

	if result.Error != nil {
		return fmt.Errorf("更新心跳失败: %w", result.Error)
	}

	m.addToOnline(deviceID)

	return nil
}

func (m *DeviceManager) SetDeviceLabels(deviceID uuid.UUID, labels DeviceLabels) error {
	return m.UpdateDevice(deviceID, map[string]interface{}{
		"labels": labels,
	})
}

func (m *DeviceManager) SetDeviceTags(deviceID uuid.UUID, tags []string) error {
	return m.UpdateDevice(deviceID, map[string]interface{}{
		"tags": tags,
	})
}

func (m *DeviceManager) EnableDevice(deviceID uuid.UUID) error {
	return m.UpdateDevice(deviceID, map[string]interface{}{
		"enabled": true,
	})
}

func (m *DeviceManager) DisableDevice(deviceID uuid.UUID) error {
	return m.UpdateDevice(deviceID, map[string]interface{}{
		"enabled": false,
	})
}

func (m *DeviceManager) CreateDeviceProfile(profile *DeviceProfile) error {
	if profile.TenantID == uuid.Nil {
		profile.TenantID = m.tenantID
	}
	if profile.ID == uuid.Nil {
		profile.ID = uuid.New()
	}

	var existingProfile DeviceProfile
	err := m.db.Where("tenant_id = ? AND name = ?", profile.TenantID, profile.Name).
		First(&existingProfile).Error
	if err == nil {
		return fmt.Errorf("设备配置文件名称已存在: %s", profile.Name)
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("查询设备配置文件失败: %w", err)
	}

	if err := m.db.Create(profile).Error; err != nil {
		return fmt.Errorf("创建设备配置文件失败: %w", err)
	}

	m.cacheProfile(profile)

	log.Printf("[DeviceManager] 设备配置文件创建成功: %s", profile.Name)
	return nil
}

func (m *DeviceManager) GetDeviceProfile(profileID uuid.UUID) (*DeviceProfile, error) {
	if profile := m.getProfileFromCache(profileID); profile != nil {
		return profile, nil
	}

	var profile DeviceProfile
	err := m.db.Where("id = ?", profileID).First(&profile).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("设备配置文件不存在: %s", profileID)
		}
		return nil, fmt.Errorf("查询设备配置文件失败: %w", err)
	}

	m.cacheProfile(&profile)
	return &profile, nil
}

func (m *DeviceManager) ListDeviceProfiles() ([]*DeviceProfile, error) {
	var profiles []*DeviceProfile
	err := m.db.Where("tenant_id = ?", m.tenantID).Find(&profiles).Error
	if err != nil {
		return nil, fmt.Errorf("查询设备配置文件列表失败: %w", err)
	}
	return profiles, nil
}

func (m *DeviceManager) UpdateDeviceProfile(profileID uuid.UUID, updates map[string]interface{}) error {
	result := m.db.Model(&DeviceProfile{}).
		Where("id = ? AND tenant_id = ?", profileID, m.tenantID).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("更新设备配置文件失败: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return fmt.Errorf("设备配置文件不存在: %s", profileID)
	}

	m.invalidateProfileCache(profileID)

	return nil
}

func (m *DeviceManager) DeleteDeviceProfile(profileID uuid.UUID) error {
	var count int64
	m.db.Model(&Device{}).Where("profile_id = ?", profileID).Count(&count)
	if count > 0 {
		return fmt.Errorf("设备配置文件正在被%d个设备使用，无法删除", count)
	}

	if err := m.db.Delete(&DeviceProfile{}, "id = ?", profileID).Error; err != nil {
		return fmt.Errorf("删除设备配置文件失败: %w", err)
	}

	m.invalidateProfileCache(profileID)

	log.Printf("[DeviceManager] 设备配置文件删除成功: %s", profileID)
	return nil
}

func (m *DeviceManager) GetOnlineDevices() []uuid.UUID {
	m.onlineMu.RLock()
	defer m.onlineMu.RUnlock()

	devices := make([]uuid.UUID, 0, len(m.onlineDevices))
	for id := range m.onlineDevices {
		devices = append(devices, id)
	}
	return devices
}

func (m *DeviceManager) GetOnlineDeviceCount() int {
	m.onlineMu.RLock()
	defer m.onlineMu.RUnlock()
	return len(m.onlineDevices)
}

func (m *DeviceManager) CheckDeviceHeartbeat(timeout time.Duration) ([]uuid.UUID, error) {
	m.onlineMu.Lock()
	defer m.onlineMu.Unlock()

	var timedOutDevices []uuid.UUID
	now := time.Now()

	for deviceID, lastHeartbeat := range m.onlineDevices {
		if now.Sub(lastHeartbeat) > timeout {
			timedOutDevices = append(timedOutDevices, deviceID)
			delete(m.onlineDevices, deviceID)
		}
	}

	for _, deviceID := range timedOutDevices {
		if err := m.UpdateDeviceStatus(deviceID, DeviceStatusOffline); err != nil {
			log.Printf("[DeviceManager] 更新设备状态失败: %v", err)
		}
	}

	return timedOutDevices, nil
}

func (m *DeviceManager) AddEventHandler(handler DeviceEventHandler) {
	m.handlerMu.Lock()
	defer m.handlerMu.Unlock()
	m.eventHandlers = append(m.eventHandlers, handler)
}

func (m *DeviceManager) emitEvent(event *DeviceEvent) {
	m.handlerMu.RLock()
	defer m.handlerMu.RUnlock()

	for _, handler := range m.eventHandlers {
		go handler(event)
	}
}

func (m *DeviceManager) cacheDevice(device *Device) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	m.deviceCache[device.ID] = device
}

func (m *DeviceManager) getFromCache(deviceID uuid.UUID) *Device {
	m.cacheMu.RLock()
	defer m.cacheMu.RUnlock()
	return m.deviceCache[deviceID]
}

func (m *DeviceManager) removeFromCache(deviceID uuid.UUID) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	delete(m.deviceCache, deviceID)
}

func (m *DeviceManager) invalidateCache(deviceID uuid.UUID) {
	m.removeFromCache(deviceID)
}

func (m *DeviceManager) cacheProfile(profile *DeviceProfile) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	m.profileCache[profile.ID] = profile
}

func (m *DeviceManager) getProfileFromCache(profileID uuid.UUID) *DeviceProfile {
	m.cacheMu.RLock()
	defer m.cacheMu.RUnlock()
	return m.profileCache[profileID]
}

func (m *DeviceManager) invalidateProfileCache(profileID uuid.UUID) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	delete(m.profileCache, profileID)
}

func (m *DeviceManager) addToOnline(deviceID uuid.UUID) {
	m.onlineMu.Lock()
	defer m.onlineMu.Unlock()
	m.onlineDevices[deviceID] = time.Now()
}

func (m *DeviceManager) removeFromOnline(deviceID uuid.UUID) {
	m.onlineMu.Lock()
	defer m.onlineMu.Unlock()
	delete(m.onlineDevices, deviceID)
}

func (m *DeviceManager) AutoDiscoverDevices(ctx context.Context, protocol ProtocolType) ([]*Device, error) {
	discoveredDevices := make([]*Device, 0)

	log.Printf("[DeviceManager] 开始自动发现设备，协议: %s", protocol)

	return discoveredDevices, nil
}

func (m *DeviceManager) BindDeviceToGateway(deviceID, gatewayID uuid.UUID) error {
	return m.UpdateDevice(deviceID, map[string]interface{}{
		"gateway_id": gatewayID,
	})
}

func (m *DeviceManager) UnbindDeviceFromGateway(deviceID uuid.UUID) error {
	return m.UpdateDevice(deviceID, map[string]interface{}{
		"gateway_id": nil,
	})
}

func (m *DeviceManager) GetChildDevices(parentID uuid.UUID) ([]*Device, error) {
	return m.ListDevices(&DeviceQuery{
		ParentID: &parentID,
	})
}

func (m *DeviceManager) GetGatewayDevices(gatewayID uuid.UUID) ([]*Device, error) {
	return m.ListDevices(&DeviceQuery{
		GatewayID: &gatewayID,
	})
}

func (m *DeviceManager) ImportDevicesFromConfig(configs []map[string]interface{}) ([]*Device, error) {
	devices := make([]*Device, 0, len(configs))

	for _, config := range configs {
		device := &Device{
			TenantID:    m.tenantID,
			Name:        config["name"].(string),
			DisplayName: getString(config, "display_name"),
			Description: getString(config, "description"),
			DeviceType:  DeviceType(getString(config, "device_type")),
			Protocol:    ProtocolType(getString(config, "protocol")),
			Labels:      make(DeviceLabels),
		}

		if err := m.RegisterDevice(device); err != nil {
			log.Printf("[DeviceManager] 导入设备失败: %v", err)
			continue
		}

		devices = append(devices, device)
	}

	return devices, nil
}

func (m *DeviceManager) ExportDevicesToConfig(deviceIDs []uuid.UUID) ([]map[string]interface{}, error) {
	devices, err := m.ListDevices(&DeviceQuery{
		IDs: deviceIDs,
	})
	if err != nil {
		return nil, err
	}

	configs := make([]map[string]interface{}, 0, len(devices))
	for _, device := range devices {
		config := map[string]interface{}{
			"name":         device.Name,
			"display_name": device.DisplayName,
			"description":  device.Description,
			"device_type":  device.DeviceType,
			"protocol":     device.Protocol,
			"labels":       device.Labels,
			"connection_info": device.ConnectionInfo,
		}
		configs = append(configs, config)
	}

	return configs, nil
}

func getString(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func (m *DeviceManager) UpsertDevice(device *Device) error {
	return m.db.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "tenant_id"}, {Name: "name"}},
		DoUpdates: clause.AssignmentColumns([]string{"display_name", "description", "status", "labels", "updated_at"}),
	}).Create(device).Error
}
