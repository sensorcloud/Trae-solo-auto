package iot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

type TelemetryManager struct {
	db       *gorm.DB
	tenantID uuid.UUID
	redis    *redis.Client

	realtimeCache map[uuid.UUID]map[string]*TelemetryData
	cacheMu       sync.RWMutex

	subscribers  map[string][]chan *TelemetryData
	subMu        sync.RWMutex
}

func NewTelemetryManager(db *gorm.DB, tenantID uuid.UUID) *TelemetryManager {
	return &TelemetryManager{
		db:            db,
		tenantID:      tenantID,
		realtimeCache: make(map[uuid.UUID]map[string]*TelemetryData),
		subscribers:   make(map[string][]chan *TelemetryData),
	}
}

func (m *TelemetryManager) SetRedis(client *redis.Client) {
	m.redis = client
}

func (m *TelemetryManager) Store(telemetry *TelemetryData) error {
	if telemetry.ID == uuid.Nil {
		telemetry.ID = uuid.New()
	}
	if telemetry.TenantID == uuid.Nil {
		telemetry.TenantID = m.tenantID
	}
	if telemetry.Timestamp.IsZero() {
		telemetry.Timestamp = time.Now()
	}
	if telemetry.Quality == "" {
		telemetry.Quality = "good"
	}

	if err := m.db.Create(telemetry).Error; err != nil {
		return fmt.Errorf("存储遥测数据失败: %w", err)
	}

	m.updateRealtimeCache(telemetry)
	m.notifySubscribers(telemetry)

	return nil
}

func (m *TelemetryManager) StoreBatch(batch []*TelemetryData) error {
	if len(batch) == 0 {
		return nil
	}

	for _, t := range batch {
		if t.ID == uuid.Nil {
			t.ID = uuid.New()
		}
		if t.TenantID == uuid.Nil {
			t.TenantID = m.tenantID
		}
		if t.Timestamp.IsZero() {
			t.Timestamp = time.Now()
		}
		if t.Quality == "" {
			t.Quality = "good"
		}
	}

	if err := m.db.CreateInBatches(batch, 1000).Error; err != nil {
		return fmt.Errorf("批量存储遥测数据失败: %w", err)
	}

	for _, t := range batch {
		m.updateRealtimeCache(t)
		m.notifySubscribers(t)
	}

	return nil
}

func (m *TelemetryManager) StoreBatchFromBatch(batch *TelemetryBatch) error {
	telemetryList := make([]*TelemetryData, 0, len(batch.Values))
	now := batch.Timestamp
	if now.IsZero() {
		now = time.Now()
	}

	for _, item := range batch.Values {
		telemetry := &TelemetryData{
			DeviceID:  batch.DeviceID,
			TenantID:  batch.TenantID,
			Timestamp: now,
			Property:  item.Property,
			Value:     item.Value,
			DataType:  item.DataType,
			Unit:      item.Unit,
			Quality:   item.Quality,
		}
		if telemetry.Quality == "" {
			telemetry.Quality = "good"
		}
		telemetryList = append(telemetryList, telemetry)
	}

	return m.StoreBatch(telemetryList)
}

func (m *TelemetryManager) Query(query *TelemetryQuery) ([]*TelemetryQueryResult, error) {
	db := m.db.Model(&TelemetryData{}).Where("tenant_id = ?", m.tenantID)

	if len(query.DeviceIDs) > 0 {
		db = db.Where("device_id IN ?", query.DeviceIDs)
	}
	if len(query.Properties) > 0 {
		db = db.Where("property IN ?", query.Properties)
	}
	if query.StartTime != nil {
		db = db.Where("timestamp >= ?", *query.StartTime)
	}
	if query.EndTime != nil {
		db = db.Where("timestamp <= ?", *query.EndTime)
	}

	if query.Aggregate != "" && query.Interval > 0 {
		return m.queryWithAggregation(db, query)
	}

	order := "DESC"
	if query.Order != "" {
		order = query.Order
	}
	db = db.Order("timestamp " + order)

	if query.Limit > 0 {
		db = db.Limit(query.Limit)
	}
	if query.Offset > 0 {
		db = db.Offset(query.Offset)
	}

	var results []TelemetryData
	if err := db.Find(&results).Error; err != nil {
		return nil, fmt.Errorf("查询遥测数据失败: %w", err)
	}

	groupedResults := make(map[uuid.UUID]map[string][]TelemetryPoint)
	for _, r := range results {
		if _, ok := groupedResults[r.DeviceID]; !ok {
			groupedResults[r.DeviceID] = make(map[string][]TelemetryPoint)
		}
		groupedResults[r.DeviceID][r.Property] = append(
			groupedResults[r.DeviceID][r.Property],
			TelemetryPoint{
				Timestamp: r.Timestamp,
				Value:     r.Value,
				Quality:   r.Quality,
			},
		)
	}

	queryResults := make([]*TelemetryQueryResult, 0)
	for deviceID, props := range groupedResults {
		for prop, points := range props {
			queryResults = append(queryResults, &TelemetryQueryResult{
				DeviceID: deviceID,
				Property: prop,
				Points:   points,
			})
		}
	}

	return queryResults, nil
}

func (m *TelemetryManager) queryWithAggregation(db *gorm.DB, query *TelemetryQuery) ([]*TelemetryQueryResult, error) {
	intervalSeconds := query.Interval

	var selectField string
	switch query.Aggregate {
	case "avg":
		selectField = "AVG(value::float)"
	case "min":
		selectField = "MIN(value::float)"
	case "max":
		selectField = "MAX(value::float)"
	case "sum":
		selectField = "SUM(value::float)"
	case "count":
		selectField = "COUNT(*)"
	default:
		selectField = "AVG(value::float)"
	}

	sql := fmt.Sprintf(`
		SELECT 
			device_id,
			property,
			%s as value,
			date_trunc('second', timestamp) - 
			(EXTRACT(EPOCH FROM date_trunc('second', timestamp))::int %% %d) * interval '1 second' as time_bucket
		FROM telemetry_data
		WHERE tenant_id = ?
	`, selectField, intervalSeconds)

	var args []interface{}
	args = append(args, m.tenantID)

	if len(query.DeviceIDs) > 0 {
		sql += " AND device_id IN ?"
		args = append(args, query.DeviceIDs)
	}
	if len(query.Properties) > 0 {
		sql += " AND property IN ?"
		args = append(args, query.Properties)
	}
	if query.StartTime != nil {
		sql += " AND timestamp >= ?"
		args = append(args, *query.StartTime)
	}
	if query.EndTime != nil {
		sql += " AND timestamp <= ?"
		args = append(args, *query.EndTime)
	}

	sql += " GROUP BY device_id, property, time_bucket ORDER BY time_bucket DESC"

	if query.Limit > 0 {
		sql += " LIMIT ?"
		args = append(args, query.Limit)
	}

	type aggResult struct {
		DeviceID  uuid.UUID
		Property  string
		Value     interface{}
		TimeBucket time.Time
	}

	var results []aggResult
	if err := m.db.Raw(sql, args...).Scan(&results).Error; err != nil {
		return nil, fmt.Errorf("聚合查询遥测数据失败: %w", err)
	}

	groupedResults := make(map[uuid.UUID]map[string][]TelemetryPoint)
	for _, r := range results {
		if _, ok := groupedResults[r.DeviceID]; !ok {
			groupedResults[r.DeviceID] = make(map[string][]TelemetryPoint)
		}
		groupedResults[r.DeviceID][r.Property] = append(
			groupedResults[r.DeviceID][r.Property],
			TelemetryPoint{
				Timestamp: r.TimeBucket,
				Value:     r.Value,
			},
		)
	}

	queryResults := make([]*TelemetryQueryResult, 0)
	for deviceID, props := range groupedResults {
		for prop, points := range props {
			queryResults = append(queryResults, &TelemetryQueryResult{
				DeviceID: deviceID,
				Property: prop,
				Points:   points,
			})
		}
	}

	return queryResults, nil
}

func (m *TelemetryManager) GetLatest(deviceID uuid.UUID, properties []string) (map[string]*TelemetryData, error) {
	m.cacheMu.RLock()
	defer m.cacheMu.RUnlock()

	if deviceCache, ok := m.realtimeCache[deviceID]; ok {
		result := make(map[string]*TelemetryData)
		if len(properties) == 0 {
			for prop, data := range deviceCache {
				result[prop] = data
			}
		} else {
			for _, prop := range properties {
				if data, ok := deviceCache[prop]; ok {
					result[prop] = data
				}
			}
		}
		return result, nil
	}

	db := m.db.Model(&TelemetryData{}).
		Where("tenant_id = ? AND device_id = ?", m.tenantID, deviceID)
	if len(properties) > 0 {
		db = db.Where("property IN ?", properties)
	}

	var results []TelemetryData
	if err := db.Order("timestamp DESC").
		Distinct("property").
		Find(&results).Error; err != nil {
		return nil, fmt.Errorf("查询最新遥测数据失败: %w", err)
	}

	result := make(map[string]*TelemetryData)
	for i := range results {
		result[results[i].Property] = &results[i]
	}

	return result, nil
}

func (m *TelemetryManager) GetLatestAll(deviceIDs []uuid.UUID) (map[uuid.UUID]map[string]*TelemetryData, error) {
	result := make(map[uuid.UUID]map[string]*TelemetryData)

	m.cacheMu.RLock()
	for _, deviceID := range deviceIDs {
		if deviceCache, ok := m.realtimeCache[deviceID]; ok {
			result[deviceID] = make(map[string]*TelemetryData)
			for prop, data := range deviceCache {
				result[deviceID][prop] = data
			}
		}
	}
	m.cacheMu.RUnlock()

	return result, nil
}

func (m *TelemetryManager) DeleteBefore(deviceID uuid.UUID, before time.Time) error {
	result := m.db.Where("device_id = ? AND timestamp < ?", deviceID, before).
		Delete(&TelemetryData{})

	if result.Error != nil {
		return fmt.Errorf("删除历史遥测数据失败: %w", result.Error)
	}

	log.Printf("[TelemetryManager] 已删除设备 %s 在 %s 之前的 %d 条遥测数据",
		deviceID, before.Format(time.RFC3339), result.RowsAffected)

	return nil
}

func (m *TelemetryManager) GetStatistics(deviceID uuid.UUID, property string, startTime, endTime time.Time) (*TelemetryStatistics, error) {
	var stats TelemetryStatistics

	err := m.db.Model(&TelemetryData{}).
		Where("tenant_id = ? AND device_id = ? AND property = ? AND timestamp >= ? AND timestamp <= ?",
			m.tenantID, deviceID, property, startTime, endTime).
		Select(`
			COUNT(*) as count,
			MIN(value::float) as min_value,
			MAX(value::float) as max_value,
			AVG(value::float) as avg_value,
			STDDEV(value::float) as std_dev
		`).
		Scan(&stats).Error

	if err != nil {
		return nil, fmt.Errorf("获取遥测统计失败: %w", err)
	}

	stats.DeviceID = deviceID
	stats.Property = property
	stats.StartTime = startTime
	stats.EndTime = endTime

	return &stats, nil
}

func (m *TelemetryManager) Subscribe(deviceID uuid.UUID, properties []string) (<-chan *TelemetryData, error) {
	ch := make(chan *TelemetryData, 100)

	m.subMu.Lock()
	defer m.subMu.Unlock()

	key := m.subscriptionKey(deviceID, properties)
	m.subscribers[key] = append(m.subscribers[key], ch)

	return ch, nil
}

func (m *TelemetryManager) Unsubscribe(deviceID uuid.UUID, properties []string, ch <-chan *TelemetryData) {
	m.subMu.Lock()
	defer m.subMu.Unlock()

	key := m.subscriptionKey(deviceID, properties)
	subs := m.subscribers[key]
	for i, sub := range subs {
		if sub == ch {
			m.subscribers[key] = append(subs[:i], subs[i+1:]...)
			close(sub)
			break
		}
	}
}

func (m *TelemetryManager) subscriptionKey(deviceID uuid.UUID, properties []string) string {
	if deviceID == uuid.Nil && len(properties) == 0 {
		return "*"
	}
	if deviceID == uuid.Nil {
		return fmt.Sprintf("props:%v", properties)
	}
	if len(properties) == 0 {
		return fmt.Sprintf("device:%s", deviceID)
	}
	return fmt.Sprintf("device:%s:props:%v", deviceID, properties)
}

func (m *TelemetryManager) notifySubscribers(telemetry *TelemetryData) {
	m.subMu.RLock()
	defer m.subMu.RUnlock()

	keys := []string{
		"*",
		fmt.Sprintf("device:%s", telemetry.DeviceID),
		fmt.Sprintf("props:[%s]", telemetry.Property),
		fmt.Sprintf("device:%s:props:[%s]", telemetry.DeviceID, telemetry.Property),
	}

	for _, key := range keys {
		if subs, ok := m.subscribers[key]; ok {
			for _, sub := range subs {
				select {
				case sub <- telemetry:
				default:
					log.Printf("[TelemetryManager] 订阅通道已满，丢弃数据")
				}
			}
		}
	}
}

func (m *TelemetryManager) updateRealtimeCache(telemetry *TelemetryData) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()

	if _, ok := m.realtimeCache[telemetry.DeviceID]; !ok {
		m.realtimeCache[telemetry.DeviceID] = make(map[string]*TelemetryData)
	}
	m.realtimeCache[telemetry.DeviceID][telemetry.Property] = telemetry
}

type TelemetryStatistics struct {
	DeviceID  uuid.UUID `json:"device_id"`
	Property  string    `json:"property"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Count     int64     `json:"count"`
	MinValue  float64   `json:"min_value"`
	MaxValue  float64   `json:"max_value"`
	AvgValue  float64   `json:"avg_value"`
	StdDev    float64   `json:"std_dev"`
}

type ShadowManager struct {
	db       *gorm.DB
	redis    *redis.Client
	tenantID uuid.UUID

	shadowCache map[uuid.UUID]*DeviceShadow
	cacheMu     sync.RWMutex

	deltaHandlers []DeltaHandler
	handlerMu     sync.RWMutex
}

type DeltaHandler func(deviceID uuid.UUID, delta map[string]interface{}) error

func NewShadowManager(db *gorm.DB, tenantID uuid.UUID) *ShadowManager {
	return &ShadowManager{
		db:           db,
		tenantID:     tenantID,
		shadowCache:  make(map[uuid.UUID]*DeviceShadow),
		deltaHandlers: make([]DeltaHandler, 0),
	}
}

func (m *ShadowManager) SetRedis(client *redis.Client) {
	m.redis = client
}

func (m *ShadowManager) GetShadow(deviceID uuid.UUID) (*DeviceShadow, error) {
	if shadow := m.getFromCache(deviceID); shadow != nil {
		return shadow, nil
	}

	shadow, err := m.loadOrCreateShadow(deviceID)
	if err != nil {
		return nil, err
	}

	m.cacheShadow(shadow)
	return shadow, nil
}

func (m *ShadowManager) loadOrCreateShadow(deviceID uuid.UUID) (*DeviceShadow, error) {
	shadow := &DeviceShadow{
		DeviceID: deviceID,
		Reported: ShadowState{
			Properties: make(map[string]interface{}),
			Tags:       make(map[string]string),
		},
		Desired: ShadowState{
			Properties: make(map[string]interface{}),
			Tags:       make(map[string]string),
		},
		Delta: ShadowState{
			Properties: make(map[string]interface{}),
		},
		Metadata: ShadowMetadata{
			Reported: make(map[string]PropertyMetadata),
			Desired:  make(map[string]PropertyMetadata),
		},
		Version:   0,
		UpdatedAt: time.Now(),
	}

	if m.redis != nil {
		ctx := context.Background()
		key := m.shadowKey(deviceID)
		data, err := m.redis.Get(ctx, key).Result()
		if err == nil {
			if err := json.Unmarshal([]byte(data), shadow); err == nil {
				return shadow, nil
			}
		}
	}

	return shadow, nil
}

func (m *ShadowManager) UpdateReported(deviceID uuid.UUID, property string, value interface{}) error {
	shadow, err := m.GetShadow(deviceID)
	if err != nil {
		return err
	}

	shadow.Reported.Properties[property] = value
	shadow.Metadata.Reported[property] = PropertyMetadata{
		Timestamp: time.Now(),
		Version:   shadow.Version + 1,
	}
	shadow.Version++
	shadow.UpdatedAt = time.Now()

	m.calculateDelta(shadow, property)

	if err := m.saveShadow(shadow); err != nil {
		return err
	}

	m.cacheShadow(shadow)
	return nil
}

func (m *ShadowManager) UpdateReportedBatch(deviceID uuid.UUID, properties map[string]interface{}) error {
	shadow, err := m.GetShadow(deviceID)
	if err != nil {
		return err
	}

	now := time.Now()
	for prop, value := range properties {
		shadow.Reported.Properties[prop] = value
		shadow.Metadata.Reported[prop] = PropertyMetadata{
			Timestamp: now,
			Version:   shadow.Version + 1,
		}
		m.calculateDelta(shadow, prop)
	}

	shadow.Version++
	shadow.UpdatedAt = now

	if err := m.saveShadow(shadow); err != nil {
		return err
	}

	m.cacheShadow(shadow)
	return nil
}

func (m *ShadowManager) UpdateDesired(deviceID uuid.UUID, properties map[string]interface{}) error {
	shadow, err := m.GetShadow(deviceID)
	if err != nil {
		return err
	}

	now := time.Now()
	for prop, value := range properties {
		shadow.Desired.Properties[prop] = value
		shadow.Metadata.Desired[prop] = PropertyMetadata{
			Timestamp: now,
			Version:   shadow.Version + 1,
		}
		m.calculateDelta(shadow, prop)
	}

	shadow.Version++
	shadow.UpdatedAt = now

	if err := m.saveShadow(shadow); err != nil {
		return err
	}

	m.cacheShadow(shadow)

	m.notifyDelta(deviceID, shadow.Delta.Properties)

	return nil
}

func (m *ShadowManager) DeleteDesired(deviceID uuid.UUID, properties []string) error {
	shadow, err := m.GetShadow(deviceID)
	if err != nil {
		return err
	}

	for _, prop := range properties {
		delete(shadow.Desired.Properties, prop)
		delete(shadow.Metadata.Desired, prop)
		m.calculateDelta(shadow, prop)
	}

	shadow.Version++
	shadow.UpdatedAt = time.Now()

	if err := m.saveShadow(shadow); err != nil {
		return err
	}

	m.cacheShadow(shadow)
	return nil
}

func (m *ShadowManager) ClearDelta(deviceID uuid.UUID) error {
	shadow, err := m.GetShadow(deviceID)
	if err != nil {
		return err
	}

	shadow.Delta = ShadowState{
		Properties: make(map[string]interface{}),
	}
	shadow.Version++
	shadow.UpdatedAt = time.Now()

	if err := m.saveShadow(shadow); err != nil {
		return err
	}

	m.cacheShadow(shadow)
	return nil
}

func (m *ShadowManager) calculateDelta(shadow *DeviceShadow, property string) {
	desiredValue, hasDesired := shadow.Desired.Properties[property]
	reportedValue, hasReported := shadow.Reported.Properties[property]

	if hasDesired {
		if !hasReported || !m.valuesEqual(desiredValue, reportedValue) {
			shadow.Delta.Properties[property] = desiredValue
		} else {
			delete(shadow.Delta.Properties, property)
		}
	} else {
		delete(shadow.Delta.Properties, property)
	}
}

func (m *ShadowManager) valuesEqual(a, b interface{}) bool {
	aJSON, err := json.Marshal(a)
	if err != nil {
		return false
	}
	bJSON, err := json.Marshal(b)
	if err != nil {
		return false
	}
	return string(aJSON) == string(bJSON)
}

func (m *ShadowManager) saveShadow(shadow *DeviceShadow) error {
	if m.redis == nil {
		return nil
	}

	ctx := context.Background()
	key := m.shadowKey(shadow.DeviceID)
	data, err := json.Marshal(shadow)
	if err != nil {
		return fmt.Errorf("序列化设备影子失败: %w", err)
	}

	return m.redis.Set(ctx, key, data, 0).Err()
}

func (m *ShadowManager) shadowKey(deviceID uuid.UUID) string {
	return fmt.Sprintf("shadow:%s:%s", m.tenantID, deviceID)
}

func (m *ShadowManager) getFromCache(deviceID uuid.UUID) *DeviceShadow {
	m.cacheMu.RLock()
	defer m.cacheMu.RUnlock()
	return m.shadowCache[deviceID]
}

func (m *ShadowManager) cacheShadow(shadow *DeviceShadow) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	m.shadowCache[shadow.DeviceID] = shadow
}

func (m *ShadowManager) invalidateCache(deviceID uuid.UUID) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	delete(m.shadowCache, deviceID)
}

func (m *ShadowManager) AddDeltaHandler(handler DeltaHandler) {
	m.handlerMu.Lock()
	defer m.handlerMu.Unlock()
	m.deltaHandlers = append(m.deltaHandlers, handler)
}

func (m *ShadowManager) notifyDelta(deviceID uuid.UUID, delta map[string]interface{}) {
	if len(delta) == 0 {
		return
	}

	m.handlerMu.RLock()
	defer m.handlerMu.RUnlock()

	for _, handler := range m.deltaHandlers {
		go func(h DeltaHandler) {
			if err := h(deviceID, delta); err != nil {
				log.Printf("[ShadowManager] Delta处理器执行失败: %v", err)
			}
		}(handler)
	}
}

func (m *ShadowManager) GetShadowVersion(deviceID uuid.UUID) (int64, error) {
	shadow, err := m.GetShadow(deviceID)
	if err != nil {
		return 0, err
	}
	return shadow.Version, nil
}

func (m *ShadowManager) SyncShadow(deviceID uuid.UUID) error {
	shadow, err := m.GetShadow(deviceID)
	if err != nil {
		return err
	}

	if len(shadow.Delta.Properties) > 0 {
		m.notifyDelta(deviceID, shadow.Delta.Properties)
	}

	return nil
}

func (m *ShadowManager) DeleteShadow(deviceID uuid.UUID) error {
	m.invalidateCache(deviceID)

	if m.redis != nil {
		ctx := context.Background()
		key := m.shadowKey(deviceID)
		return m.redis.Del(ctx, key).Err()
	}

	return nil
}

func (m *ShadowManager) ListShadows(deviceIDs []uuid.UUID) ([]*DeviceShadow, error) {
	shadows := make([]*DeviceShadow, 0, len(deviceIDs))

	for _, deviceID := range deviceIDs {
		shadow, err := m.GetShadow(deviceID)
		if err != nil {
			continue
		}
		shadows = append(shadows, shadow)
	}

	return shadows, nil
}

func (m *ShadowManager) GetDelta(deviceID uuid.UUID) (map[string]interface{}, error) {
	shadow, err := m.GetShadow(deviceID)
	if err != nil {
		return nil, err
	}
	return shadow.Delta.Properties, nil
}

func (m *ShadowManager) HasDelta(deviceID uuid.UUID) (bool, error) {
	shadow, err := m.GetShadow(deviceID)
	if err != nil {
		return false, err
	}
	return len(shadow.Delta.Properties) > 0, nil
}

func (m *ShadowManager) UpdateShadowTags(deviceID uuid.UUID, tags map[string]string, isDesired bool) error {
	shadow, err := m.GetShadow(deviceID)
	if err != nil {
		return err
	}

	if isDesired {
		for k, v := range tags {
			shadow.Desired.Tags[k] = v
		}
	} else {
		for k, v := range tags {
			shadow.Reported.Tags[k] = v
		}
	}

	shadow.Version++
	shadow.UpdatedAt = time.Now()

	if err := m.saveShadow(shadow); err != nil {
		return err
	}

	m.cacheShadow(shadow)
	return nil
}

func (m *ShadowManager) GetShadowHistory(deviceID uuid.UUID, limit int) ([]*DeviceShadow, error) {
	return nil, errors.New("影子历史记录功能暂未实现")
}
