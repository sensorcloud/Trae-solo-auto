package iot

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/edgehub/edgehub/internal/iot/protocol"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

type ConnectorConfig struct {
	TenantID        uuid.UUID
	EnableMQTT      bool
	EnableModbus    bool
	EnableOPCUA     bool
	MQTTConfig      *protocol.MQTTConfig
	ModbusConfig    *protocol.ModbusConfig
	OPCUAConfig     *protocol.OPCUAConfig
	MaxDevices      int
	TelemetryBuffer int
	WorkerCount     int
}

type IoTConnector struct {
	config       *ConnectorConfig
	db           *gorm.DB

	deviceManager    *DeviceManager
	telemetryManager *TelemetryManager
	shadowManager    *ShadowManager

	mqttAdapter   protocol.ProtocolAdapter
	modbusAdapter protocol.ProtocolAdapter
	opcuaAdapter  protocol.ProtocolAdapter

	adapters     map[ProtocolType]protocol.ProtocolAdapter
	statusMap    map[ProtocolType]*ConnectorStatus

	telemetryChan chan *TelemetryData
	eventChan     chan *DeviceEventRecord
	alarmChan     chan *DeviceAlarmRecord
	commandChan   chan *DeviceCommandRequest
	responseChan  chan *DeviceCommandResponse

	stopCh chan struct{}
	wg     sync.WaitGroup

	mu     sync.RWMutex
	status string
}

func NewIoTConnector(db *gorm.DB, config *ConnectorConfig) *IoTConnector {
	bufferSize := config.TelemetryBuffer
	if bufferSize <= 0 {
		bufferSize = 10000
	}

	return &IoTConnector{
		config:    config,
		db:        db,
		adapters:  make(map[ProtocolType]protocol.ProtocolAdapter),
		statusMap: make(map[ProtocolType]*ConnectorStatus),
		telemetryChan: make(chan *TelemetryData, bufferSize),
		eventChan:     make(chan *DeviceEventRecord, 1000),
		alarmChan:     make(chan *DeviceAlarmRecord, 1000),
		commandChan:   make(chan *DeviceCommandRequest, 100),
		responseChan:  make(chan *DeviceCommandResponse, 100),
		stopCh:        make(chan struct{}),
		status:        "initializing",
	}
}

func (c *IoTConnector) Start(ctx context.Context) error {
	log.Println("[IoTConnector] 正在启动IoT连接器...")

	c.deviceManager = NewDeviceManager(c.db, c.config.TenantID)
	c.telemetryManager = NewTelemetryManager(c.db, c.config.TenantID)
	c.shadowManager = NewShadowManager(c.db, c.config.TenantID)

	if err := c.initAdapters(ctx); err != nil {
		return fmt.Errorf("初始化协议适配器失败: %w", err)
	}

	c.wg.Add(1)
	go c.telemetryProcessor(ctx)

	c.wg.Add(1)
	go c.eventProcessor(ctx)

	c.wg.Add(1)
	go c.alarmProcessor(ctx)

	c.wg.Add(1)
	go c.commandProcessor(ctx)

	c.wg.Add(1)
	go c.healthCheckLoop(ctx)

	c.wg.Add(1)
	go c.metricsCollector(ctx)

	c.setStatus("running")
	log.Println("[IoTConnector] IoT连接器启动成功")
	return nil
}

func (c *IoTConnector) Stop() error {
	log.Println("[IoTConnector] 正在停止IoT连接器...")
	close(c.stopCh)
	c.wg.Wait()

	for proto, adapter := range c.adapters {
		if err := adapter.Stop(); err != nil {
			log.Printf("[IoTConnector] 停止%s适配器失败: %v", proto, err)
		}
	}

	c.setStatus("stopped")
	log.Println("[IoTConnector] IoT连接器已停止")
	return nil
}

func (c *IoTConnector) initAdapters(ctx context.Context) error {
	if c.config.EnableMQTT && c.config.MQTTConfig != nil {
		mqttAdapter := protocol.NewMQTTAdapter(c.config.MQTTConfig)
		mqttAdapter.SetMessageHandler(c.handleMQTTMessage)
		if err := mqttAdapter.Start(ctx); err != nil {
			log.Printf("[IoTConnector] 启动MQTT适配器失败: %v", err)
		} else {
			c.adapters[ProtocolMQTT] = mqttAdapter
			c.mqttAdapter = mqttAdapter
			c.statusMap[ProtocolMQTT] = &ConnectorStatus{
				Protocol: ProtocolMQTT,
				Status:   "connected",
			}
			log.Println("[IoTConnector] MQTT适配器启动成功")
		}
	}

	if c.config.EnableModbus && c.config.ModbusConfig != nil {
		modbusAdapter := protocol.NewModbusAdapter(c.config.ModbusConfig)
		modbusAdapter.SetMessageHandler(c.handleProtocolMessage)
		if err := modbusAdapter.Start(ctx); err != nil {
			log.Printf("[IoTConnector] 启动Modbus适配器失败: %v", err)
		} else {
			c.adapters[ProtocolModbus] = modbusAdapter
			c.modbusAdapter = modbusAdapter
			c.statusMap[ProtocolModbus] = &ConnectorStatus{
				Protocol: ProtocolModbus,
				Status:   "connected",
			}
			log.Println("[IoTConnector] Modbus适配器启动成功")
		}
	}

	if c.config.EnableOPCUA && c.config.OPCUAConfig != nil {
		opcuaAdapter := protocol.NewOPCUAAdapter(c.config.OPCUAConfig)
		opcuaAdapter.SetMessageHandler(c.handleProtocolMessage)
		if err := opcuaAdapter.Start(ctx); err != nil {
			log.Printf("[IoTConnector] 启动OPC-UA适配器失败: %v", err)
		} else {
			c.adapters[ProtocolOPCUA] = opcuaAdapter
			c.opcuaAdapter = opcuaAdapter
			c.statusMap[ProtocolOPCUA] = &ConnectorStatus{
				Protocol: ProtocolOPCUA,
				Status:   "connected",
			}
			log.Println("[IoTConnector] OPC-UA适配器启动成功")
		}
	}

	return nil
}

func (c *IoTConnector) handleMQTTMessage(topic string, payload []byte) error {
	deviceID, err := c.extractDeviceIDFromTopic(topic)
	if err != nil {
		return err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		return fmt.Errorf("解析MQTT消息失败: %w", err)
	}

	now := time.Now()
	for prop, value := range data {
		telemetry := &TelemetryData{
			DeviceID:  deviceID,
			TenantID:  c.config.TenantID,
			Timestamp: now,
			Property:  prop,
			Value:     value,
			Quality:   "good",
		}
		c.telemetryChan <- telemetry
	}

	c.updateConnectorStatus(ProtocolMQTT, func(s *ConnectorStatus) {
		s.MessagesIn++
		s.BytesIn += int64(len(payload))
	})

	return nil
}

func (c *IoTConnector) handleProtocolMessage(topic string, payload []byte) error {
	deviceID, property, err := c.parseProtocolTopic(topic)
	if err != nil {
		return err
	}

	var data map[string]interface{}
	if err := json.Unmarshal(payload, &data); err != nil {
		data = map[string]interface{}{
			"value": string(payload),
		}
	}

	now := time.Now()
	if property != "" {
		telemetry := &TelemetryData{
			DeviceID:  deviceID,
			TenantID:  c.config.TenantID,
			Timestamp: now,
			Property:  property,
			Value:     data,
			Quality:   "good",
		}
		c.telemetryChan <- telemetry
	} else {
		for prop, value := range data {
			telemetry := &TelemetryData{
				DeviceID:  deviceID,
				TenantID:  c.config.TenantID,
				Timestamp: now,
				Property:  prop,
				Value:     value,
				Quality:   "good",
			}
			c.telemetryChan <- telemetry
		}
	}

	return nil
}

func (c *IoTConnector) telemetryProcessor(ctx context.Context) {
	defer c.wg.Done()

	batch := make([]*TelemetryData, 0, 100)
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			if len(batch) > 0 {
				c.flushTelemetryBatch(batch)
			}
			return
		case telemetry := <-c.telemetryChan:
			batch = append(batch, telemetry)
			if len(batch) >= 100 {
				c.flushTelemetryBatch(batch)
				batch = make([]*TelemetryData, 0, 100)
			}
		case <-ticker.C:
			if len(batch) > 0 {
				c.flushTelemetryBatch(batch)
				batch = make([]*TelemetryData, 0, 100)
			}
		}
	}
}

func (c *IoTConnector) flushTelemetryBatch(batch []*TelemetryData) {
	if len(batch) == 0 {
		return
	}

	if err := c.telemetryManager.StoreBatch(batch); err != nil {
		log.Printf("[IoTConnector] 存储遥测数据失败: %v", err)
		return
	}

	for _, t := range batch {
		if err := c.shadowManager.UpdateReported(t.DeviceID, t.Property, t.Value); err != nil {
			log.Printf("[IoTConnector] 更新设备影子失败: %v", err)
		}
	}

	log.Printf("[IoTConnector] 已存储%d条遥测数据", len(batch))
}

func (c *IoTConnector) eventProcessor(ctx context.Context) {
	defer c.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case event := <-c.eventChan:
			if err := c.processEvent(event); err != nil {
				log.Printf("[IoTConnector] 处理事件失败: %v", err)
			}
		}
	}
}

func (c *IoTConnector) processEvent(event *DeviceEventRecord) error {
	if err := c.db.Create(event).Error; err != nil {
		return fmt.Errorf("存储事件记录失败: %w", err)
	}

	log.Printf("[IoTConnector] 设备 %s 触发事件: %s", event.DeviceID, event.EventName)
	return nil
}

func (c *IoTConnector) alarmProcessor(ctx context.Context) {
	defer c.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case alarm := <-c.alarmChan:
			if err := c.processAlarm(alarm); err != nil {
				log.Printf("[IoTConnector] 处理告警失败: %v", err)
			}
		}
	}
}

func (c *IoTConnector) processAlarm(alarm *DeviceAlarmRecord) error {
	var existingAlarm DeviceAlarmRecord
	err := c.db.Where("device_id = ? AND alarm_name = ? AND status = ?",
		alarm.DeviceID, alarm.AlarmName, "active").First(&existingAlarm).Error

	if err == gorm.ErrRecordNotFound {
		if err := c.db.Create(alarm).Error; err != nil {
			return fmt.Errorf("存储告警记录失败: %w", err)
		}
		log.Printf("[IoTConnector] 设备 %s 触发告警: %s [%s]", alarm.DeviceID, alarm.AlarmName, alarm.Severity)
	}

	return nil
}

func (c *IoTConnector) commandProcessor(ctx context.Context) {
	defer c.wg.Done()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case cmd := <-c.commandChan:
			if err := c.executeCommand(cmd); err != nil {
				log.Printf("[IoTConnector] 执行命令失败: %v", err)
			}
		}
	}
}

func (c *IoTConnector) executeCommand(cmd *DeviceCommandRequest) error {
	device, err := c.deviceManager.GetDevice(cmd.DeviceID)
	if err != nil {
		return fmt.Errorf("获取设备信息失败: %w", err)
	}

	var adapter protocol.ProtocolAdapter
	var proto ProtocolType
	switch device.Protocol {
	case ProtocolMQTT:
		adapter = c.mqttAdapter
		proto = ProtocolMQTT
	case ProtocolModbus:
		adapter = c.modbusAdapter
		proto = ProtocolModbus
	case ProtocolOPCUA:
		adapter = c.opcuaAdapter
		proto = ProtocolOPCUA
	default:
		return fmt.Errorf("不支持的协议类型: %s", device.Protocol)
	}

	if adapter == nil {
		return fmt.Errorf("协议适配器未初始化: %s", device.Protocol)
	}

	connectionInfo := &protocol.DeviceConnectionInfo{
		TopicPrefix: device.ConnectionInfo.TopicPrefix,
		SlaveID:     device.ConnectionInfo.SlaveID,
	}

	protoCmd := &protocol.DeviceCommandRequest{
		DeviceID:      cmd.DeviceID,
		CommandName:   cmd.CommandName,
		Parameters:    cmd.Parameters,
		Timeout:       cmd.Timeout,
		Async:         cmd.Async,
		CorrelationID: cmd.CorrelationID,
	}

	result, err := adapter.SendCommand(cmd.DeviceID, connectionInfo, protoCmd)
	response := &DeviceCommandResponse{
		DeviceID:      cmd.DeviceID,
		CommandName:   cmd.CommandName,
		CorrelationID: cmd.CorrelationID,
		Timestamp:     time.Now(),
	}

	if err != nil {
		response.Status = "failed"
		response.ErrorMessage = err.Error()
	} else {
		response.Status = "success"
		response.Result = result
	}

	c.responseChan <- response

	c.updateConnectorStatus(proto, func(s *ConnectorStatus) {
		s.MessagesOut++
	})

	return nil
}

func (c *IoTConnector) healthCheckLoop(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.checkAdapterHealth()
		}
	}
}

func (c *IoTConnector) checkAdapterHealth() {
	for proto, adapter := range c.adapters {
		if !adapter.IsConnected() {
			log.Printf("[IoTConnector] %s适配器连接断开，尝试重连...", proto)
			if err := adapter.Reconnect(); err != nil {
				log.Printf("[IoTConnector] %s适配器重连失败: %v", proto, err)
				c.updateConnectorStatus(proto, func(s *ConnectorStatus) {
					s.Status = "disconnected"
					s.LastError = err.Error()
					s.ReconnectCount++
				})
			} else {
				log.Printf("[IoTConnector] %s适配器重连成功", proto)
				c.updateConnectorStatus(proto, func(s *ConnectorStatus) {
					s.Status = "connected"
					now := time.Now()
					s.ConnectedAt = &now
				})
			}
		}
	}
}

func (c *IoTConnector) metricsCollector(ctx context.Context) {
	defer c.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.collectMetrics()
		}
	}
}

func (c *IoTConnector) collectMetrics() {
	for proto := range c.adapters {
		devices, err := c.deviceManager.ListDevices(&DeviceQuery{
			Protocol: proto,
			Status:   DeviceStatusOnline,
		})
		if err == nil {
			c.updateConnectorStatus(proto, func(s *ConnectorStatus) {
				s.DevicesOnline = len(devices)
			})
		}

		totalDevices, err := c.deviceManager.CountDevices(&DeviceQuery{
			Protocol: proto,
		})
		if err == nil {
			c.updateConnectorStatus(proto, func(s *ConnectorStatus) {
				s.DevicesTotal = totalDevices
			})
		}
	}
}

func (c *IoTConnector) extractDeviceIDFromTopic(topic string) (uuid.UUID, error) {
	return uuid.New(), nil
}

func (c *IoTConnector) parseProtocolTopic(topic string) (uuid.UUID, string, error) {
	return uuid.New(), "", nil
}

func (c *IoTConnector) updateConnectorStatus(proto ProtocolType, update func(*ConnectorStatus)) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if status, ok := c.statusMap[proto]; ok {
		update(status)
	}
}

func (c *IoTConnector) setStatus(status string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.status = status
}

func (c *IoTConnector) GetStatus() string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.status
}

func (c *IoTConnector) GetConnectorStatus(proto ProtocolType) (*ConnectorStatus, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	status, ok := c.statusMap[proto]
	if !ok {
		return nil, fmt.Errorf("协议适配器不存在: %s", proto)
	}
	return status, nil
}

func (c *IoTConnector) GetAllConnectorStatus() map[ProtocolType]*ConnectorStatus {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make(map[ProtocolType]*ConnectorStatus)
	for k, v := range c.statusMap {
		statusCopy := *v
		result[k] = &statusCopy
	}
	return result
}

func (c *IoTConnector) RegisterDevice(device *Device) error {
	return c.deviceManager.RegisterDevice(device)
}

func (c *IoTConnector) UnregisterDevice(deviceID uuid.UUID) error {
	return c.deviceManager.UnregisterDevice(deviceID)
}

func (c *IoTConnector) GetDevice(deviceID uuid.UUID) (*Device, error) {
	return c.deviceManager.GetDevice(deviceID)
}

func (c *IoTConnector) ListDevices(query *DeviceQuery) ([]*Device, error) {
	return c.deviceManager.ListDevices(query)
}

func (c *IoTConnector) SendCommand(cmd *DeviceCommandRequest) (<-chan *DeviceCommandResponse, error) {
	responseChan := make(chan *DeviceCommandResponse, 1)

	go func() {
		c.commandChan <- cmd

		timeout := time.After(time.Duration(cmd.Timeout) * time.Second)
		if cmd.Timeout == 0 {
			timeout = time.After(30 * time.Second)
		}

		select {
		case resp := <-c.responseChan:
			if resp.CorrelationID == cmd.CorrelationID {
				responseChan <- resp
			}
		case <-timeout:
			responseChan <- &DeviceCommandResponse{
				DeviceID:      cmd.DeviceID,
				CommandName:   cmd.CommandName,
				CorrelationID: cmd.CorrelationID,
				Status:        "timeout",
				ErrorMessage:  "命令执行超时",
				Timestamp:     time.Now(),
			}
		}
		close(responseChan)
	}()

	return responseChan, nil
}

func (c *IoTConnector) GetDeviceShadow(deviceID uuid.UUID) (*DeviceShadow, error) {
	return c.shadowManager.GetShadow(deviceID)
}

func (c *IoTConnector) UpdateDesiredState(deviceID uuid.UUID, properties map[string]interface{}) error {
	return c.shadowManager.UpdateDesired(deviceID, properties)
}

func (c *IoTConnector) QueryTelemetry(query *TelemetryQuery) ([]*TelemetryQueryResult, error) {
	return c.telemetryManager.Query(query)
}

func (c *IoTConnector) SubscribeDeviceEvents(deviceID uuid.UUID) (<-chan *DeviceEventRecord, error) {
	eventChan := make(chan *DeviceEventRecord, 100)

	go func() {
		<-c.stopCh
		close(eventChan)
	}()

	return eventChan, nil
}

func (c *IoTConnector) SubscribeDeviceAlarms(deviceID uuid.UUID) (<-chan *DeviceAlarmRecord, error) {
	alarmChan := make(chan *DeviceAlarmRecord, 100)

	go func() {
		<-c.stopCh
		close(alarmChan)
	}()

	return alarmChan, nil
}

func (c *IoTConnector) PublishEvent(event *DeviceEventRecord) {
	c.eventChan <- event
}

func (c *IoTConnector) PublishAlarm(alarm *DeviceAlarmRecord) {
	c.alarmChan <- alarm
}
