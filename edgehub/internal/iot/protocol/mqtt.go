package protocol

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

type MessageHandler func(topic string, payload []byte) error

type ProtocolAdapter interface {
	Start(ctx context.Context) error
	Stop() error
	Reconnect() error
	IsConnected() bool
	SetMessageHandler(handler MessageHandler)
	SendCommand(deviceID uuid.UUID, connectionInfo interface{}, cmd *DeviceCommandRequest) (map[string]interface{}, error)
	Subscribe(topic string) error
	Unsubscribe(topic string) error
	Publish(topic string, payload []byte) error
}

type DeviceCommandRequest struct {
	DeviceID      uuid.UUID              `json:"device_id"`
	CommandName   string                 `json:"command_name"`
	Parameters    map[string]interface{} `json:"parameters,omitempty"`
	Timeout       int                    `json:"timeout,omitempty"`
	Async         bool                   `json:"async"`
	CorrelationID string                 `json:"correlation_id,omitempty"`
}

type MQTTConfig struct {
	Broker         string `json:"broker"`
	Port           int    `json:"port"`
	ClientID       string `json:"client_id,omitempty"`
	Username       string `json:"username,omitempty"`
	Password       string `json:"-"`
	CleanSession   bool   `json:"clean_session"`
	KeepAlive      int    `json:"keep_alive"`
	QoS            int    `json:"qos"`
	TopicPrefix    string `json:"topic_prefix"`
	TLSEnabled     bool   `json:"tls_enabled"`
	TLSSkipVerify  bool   `json:"tls_skip_verify"`
	Certificate    string `json:"-"`
	PrivateKey     string `json:"-"`
	AutoReconnect  bool   `json:"auto_reconnect"`
	MaxReconnect   int    `json:"max_reconnect"`
	ReconnectDelay int    `json:"reconnect_delay"`
	WillTopic      string `json:"will_topic,omitempty"`
	WillPayload    string `json:"will_payload,omitempty"`
	WillQoS        int    `json:"will_qos"`
	WillRetained   bool   `json:"will_retained"`
}

type DeviceConnectionInfo struct {
	Endpoint     string            `json:"endpoint,omitempty"`
	Port         int               `json:"port,omitempty"`
	TopicPrefix  string            `json:"topic_prefix,omitempty"`
	ClientID     string            `json:"client_id,omitempty"`
	KeepAlive    int               `json:"keep_alive,omitempty"`
	CleanSession bool              `json:"clean_session,omitempty"`
	QoS          int               `json:"qos,omitempty"`
	SlaveID      int               `json:"slave_id,omitempty"`
	UnitID       int               `json:"unit_id,omitempty"`
	Extra        map[string]string `json:"extra,omitempty"`
}

type MQTTAdapter struct {
	config    *MQTTConfig
	client    MQTTClient
	handler   MessageHandler

	connected     bool
	connectMu     sync.RWMutex
	stopCh        chan struct{}
	wg            sync.WaitGroup

	reconnectCount int
	lastError      error
}

type MQTTClient interface {
	Connect() error
	Disconnect(quiesce uint)
	IsConnected() bool
	Subscribe(topic string, qos byte, callback MQTTMessageHandler) error
	Unsubscribe(topics ...string) error
	Publish(topic string, qos byte, retained bool, payload interface{}) error
}

type MQTTMessageHandler func(topic string, payload []byte)

func NewMQTTAdapter(config *MQTTConfig) *MQTTAdapter {
	return &MQTTAdapter{
		config: config,
		stopCh: make(chan struct{}),
	}
}

func (a *MQTTAdapter) Start(ctx context.Context) error {
	client, err := a.createMQTTClient()
	if err != nil {
		return fmt.Errorf("创建MQTT客户端失败: %w", err)
	}

	a.client = client

	if err := a.client.Connect(); err != nil {
		return fmt.Errorf("连接MQTT Broker失败: %w", err)
	}

	a.setConnected(true)
	log.Printf("[MQTTAdapter] 已连接到MQTT Broker: %s:%d", a.config.Broker, a.config.Port)

	a.subscribeDefaultTopics()

	return nil
}

func (a *MQTTAdapter) createMQTTClient() (MQTTClient, error) {
	return &mockMQTTClient{
		config: a.config,
	}, nil
}

func (a *MQTTAdapter) subscribeDefaultTopics() {
	topics := []string{
		fmt.Sprintf("%s/+/telemetry", a.config.TopicPrefix),
		fmt.Sprintf("%s/+/events", a.config.TopicPrefix),
		fmt.Sprintf("%s/+/status", a.config.TopicPrefix),
	}

	for _, topic := range topics {
		if err := a.Subscribe(topic); err != nil {
			log.Printf("[MQTTAdapter] 订阅主题失败: %v", err)
		} else {
			log.Printf("[MQTTAdapter] 已订阅主题: %s", topic)
		}
	}
}

func (a *MQTTAdapter) Stop() error {
	close(a.stopCh)
	a.wg.Wait()

	if a.client != nil && a.client.IsConnected() {
		a.client.Disconnect(250)
	}

	a.setConnected(false)
	log.Println("[MQTTAdapter] 已断开MQTT连接")
	return nil
}

func (a *MQTTAdapter) Reconnect() error {
	if a.client == nil {
		return fmt.Errorf("MQTT客户端未初始化")
	}

	if a.client.IsConnected() {
		a.client.Disconnect(100)
	}

	if err := a.client.Connect(); err != nil {
		a.lastError = err
		return fmt.Errorf("重连MQTT Broker失败: %w", err)
	}

	a.reconnectCount++
	a.setConnected(true)
	log.Printf("[MQTTAdapter] MQTT重连成功，重连次数: %d", a.reconnectCount)
	return nil
}

func (a *MQTTAdapter) IsConnected() bool {
	a.connectMu.RLock()
	defer a.connectMu.RUnlock()
	return a.connected && a.client != nil && a.client.IsConnected()
}

func (a *MQTTAdapter) SetMessageHandler(handler MessageHandler) {
	a.handler = handler
}

func (a *MQTTAdapter) Subscribe(topic string) error {
	if !a.IsConnected() {
		return fmt.Errorf("MQTT未连接")
	}

	if err := a.client.Subscribe(topic, byte(a.config.QoS), a.handleMessage); err != nil {
		return fmt.Errorf("订阅主题失败: %w", err)
	}

	log.Printf("[MQTTAdapter] 已订阅主题: %s", topic)
	return nil
}

func (a *MQTTAdapter) Unsubscribe(topic string) error {
	if !a.IsConnected() {
		return fmt.Errorf("MQTT未连接")
	}

	if err := a.client.Unsubscribe(topic); err != nil {
		return fmt.Errorf("取消订阅主题失败: %w", err)
	}

	log.Printf("[MQTTAdapter] 已取消订阅主题: %s", topic)
	return nil
}

func (a *MQTTAdapter) Publish(topic string, payload []byte) error {
	if !a.IsConnected() {
		return fmt.Errorf("MQTT未连接")
	}

	if err := a.client.Publish(topic, byte(a.config.QoS), false, payload); err != nil {
		return fmt.Errorf("发布消息失败: %w", err)
	}

	return nil
}

func (a *MQTTAdapter) SendCommand(deviceID uuid.UUID, connectionInfo interface{}, cmd *DeviceCommandRequest) (map[string]interface{}, error) {
	if !a.IsConnected() {
		return nil, fmt.Errorf("MQTT未连接")
	}

	var topicPrefix string
	if ci, ok := connectionInfo.(*DeviceConnectionInfo); ok && ci.TopicPrefix != "" {
		topicPrefix = ci.TopicPrefix
	} else {
		topicPrefix = a.config.TopicPrefix
	}

	commandTopic := fmt.Sprintf("%s/%s/command/%s", topicPrefix, deviceID, cmd.CommandName)
	payload, err := json.Marshal(map[string]interface{}{
		"command":        cmd.CommandName,
		"parameters":     cmd.Parameters,
		"correlation_id": cmd.CorrelationID,
		"timestamp":      time.Now().Unix(),
	})
	if err != nil {
		return nil, fmt.Errorf("序列化命令失败: %w", err)
	}

	if err := a.Publish(commandTopic, payload); err != nil {
		return nil, err
	}

	log.Printf("[MQTTAdapter] 已发送命令 %s 到设备 %s", cmd.CommandName, deviceID)
	return map[string]interface{}{
		"status":         "sent",
		"correlation_id": cmd.CorrelationID,
	}, nil
}

func (a *MQTTAdapter) handleMessage(topic string, payload []byte) {
	log.Printf("[MQTTAdapter] 收到消息 - 主题: %s", topic)

	if a.handler != nil {
		if err := a.handler(topic, payload); err != nil {
			log.Printf("[MQTTAdapter] 处理消息失败: %v", err)
		}
	}
}

func (a *MQTTAdapter) setConnected(connected bool) {
	a.connectMu.Lock()
	defer a.connectMu.Unlock()
	a.connected = connected
}

func (a *MQTTAdapter) SubscribeDeviceTelemetry(deviceID uuid.UUID) error {
	topic := fmt.Sprintf("%s/%s/telemetry", a.config.TopicPrefix, deviceID)
	return a.Subscribe(topic)
}

func (a *MQTTAdapter) SubscribeDeviceEvents(deviceID uuid.UUID) error {
	topic := fmt.Sprintf("%s/%s/events", a.config.TopicPrefix, deviceID)
	return a.Subscribe(topic)
}

func (a *MQTTAdapter) SubscribeDeviceStatus(deviceID uuid.UUID) error {
	topic := fmt.Sprintf("%s/%s/status", a.config.TopicPrefix, deviceID)
	return a.Subscribe(topic)
}

func (a *MQTTAdapter) PublishCommand(deviceID uuid.UUID, command string, params map[string]interface{}) error {
	topic := fmt.Sprintf("%s/%s/command/%s", a.config.TopicPrefix, deviceID, command)
	payload, err := json.Marshal(params)
	if err != nil {
		return err
	}
	return a.Publish(topic, payload)
}

func (a *MQTTAdapter) PublishDesiredState(deviceID uuid.UUID, state map[string]interface{}) error {
	topic := fmt.Sprintf("%s/%s/desired", a.config.TopicPrefix, deviceID)
	payload, err := json.Marshal(state)
	if err != nil {
		return err
	}
	return a.Publish(topic, payload)
}

func (a *MQTTAdapter) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"connected":       a.IsConnected(),
		"reconnect_count": a.reconnectCount,
		"last_error":      a.lastError,
	}
}

type MQTTMessage struct {
	Topic       string                 `json:"topic"`
	Payload     map[string]interface{} `json:"payload"`
	Timestamp   int64                  `json:"timestamp"`
	DeviceID    string                 `json:"device_id,omitempty"`
	MessageType string                 `json:"message_type"`
}

func ParseMQTTMessage(topic string, payload []byte) (*MQTTMessage, error) {
	msg := &MQTTMessage{
		Topic:     topic,
		Timestamp: time.Now().Unix(),
	}

	if err := json.Unmarshal(payload, &msg.Payload); err != nil {
		return nil, fmt.Errorf("解析MQTT消息失败: %w", err)
	}

	return msg, nil
}

func (a *MQTTAdapter) RequestResponse(ctx context.Context, deviceID uuid.UUID, command string, params map[string]interface{}, timeout time.Duration) (map[string]interface{}, error) {
	correlationID := uuid.New().String()
	responseTopic := fmt.Sprintf("%s/%s/response/%s", a.config.TopicPrefix, deviceID, correlationID)

	responseChan := make(chan []byte, 1)

	handler := func(topic string, payload []byte) {
		responseChan <- payload
	}

	if err := a.client.Subscribe(responseTopic, byte(a.config.QoS), handler); err != nil {
		return nil, fmt.Errorf("订阅响应主题失败: %w", err)
	}
	defer a.client.Unsubscribe(responseTopic)

	commandTopic := fmt.Sprintf("%s/%s/command/%s", a.config.TopicPrefix, deviceID, command)
	payload, _ := json.Marshal(map[string]interface{}{
		"command":        command,
		"parameters":     params,
		"correlation_id": correlationID,
		"response_topic": responseTopic,
	})

	if err := a.client.Publish(commandTopic, byte(a.config.QoS), false, payload); err != nil {
		return nil, fmt.Errorf("发送命令失败: %w", err)
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, fmt.Errorf("等待响应超时")
	case response := <-responseChan:
		var result map[string]interface{}
		if err := json.Unmarshal(response, &result); err != nil {
			return nil, fmt.Errorf("解析响应失败: %w", err)
		}
		return result, nil
	}
}

type mockMQTTClient struct {
	config      *MQTTConfig
	connected   bool
	connectMu   sync.RWMutex
	subscribers map[string]MQTTMessageHandler
	subMu       sync.RWMutex
}

func (c *mockMQTTClient) Connect() error {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()

	c.connected = true
	c.subscribers = make(map[string]MQTTMessageHandler)

	log.Printf("[MockMQTTClient] 模拟连接到: %s:%d", c.config.Broker, c.config.Port)
	return nil
}

func (c *mockMQTTClient) Disconnect(quiesce uint) {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()

	c.connected = false
}

func (c *mockMQTTClient) IsConnected() bool {
	c.connectMu.RLock()
	defer c.connectMu.RUnlock()
	return c.connected
}

func (c *mockMQTTClient) Subscribe(topic string, qos byte, callback MQTTMessageHandler) error {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	c.subscribers[topic] = callback
	return nil
}

func (c *mockMQTTClient) Unsubscribe(topics ...string) error {
	c.subMu.Lock()
	defer c.subMu.Unlock()

	for _, topic := range topics {
		delete(c.subscribers, topic)
	}
	return nil
}

func (c *mockMQTTClient) Publish(topic string, qos byte, retained bool, payload interface{}) error {
	c.subMu.RLock()
	defer c.subMu.RUnlock()

	var payloadBytes []byte
	switch p := payload.(type) {
	case []byte:
		payloadBytes = p
	case string:
		payloadBytes = []byte(p)
	default:
		payloadBytes = []byte(fmt.Sprintf("%v", p))
	}

	for t, handler := range c.subscribers {
		if matchTopic(t, topic) {
			go handler(topic, payloadBytes)
		}
	}

	return nil
}

func matchTopic(pattern, topic string) bool {
	if pattern == topic {
		return true
	}
	if pattern == "+" || pattern == "#" {
		return true
	}
	return true
}
