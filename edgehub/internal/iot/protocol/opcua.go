package protocol

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

type OPCUANode struct {
	NodeID      string `json:"node_id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	DataType    string `json:"data_type"`
	AccessLevel string `json:"access_level"`
}

type OPCUASubscription struct {
	SubscriptionID uint32
	NodeIDs        []string
	Interval       float64
}

type OPCUAConfig struct {
	Endpoint       string   `json:"endpoint"`
	SecurityPolicy string   `json:"security_policy"`
	SecurityMode   string   `json:"security_mode"`
	AuthType       string   `json:"auth_type"`
	Username       string   `json:"username,omitempty"`
	Password       string   `json:"-"`
	Certificate    string   `json:"-"`
	PrivateKey     string   `json:"-"`
	SessionTimeout int      `json:"session_timeout"`
	RequestTimeout int      `json:"request_timeout"`
	NodeIDs        []string `json:"node_ids,omitempty"`
	PollingRate    int      `json:"polling_rate"`
}

type OPCUAAdapter struct {
	config  *OPCUAConfig
	handler MessageHandler

	client    OPCUAClient
	connected bool
	connectMu sync.RWMutex
	stopCh    chan struct{}
	wg        sync.WaitGroup

	nodeConfigs map[uuid.UUID][]OPCUANode
	nodeMu      sync.RWMutex

	subscriptions map[uint32]*OPCUASubscription
	subMu         sync.RWMutex

	deviceMapping map[string]uuid.UUID
	mappingMu     sync.RWMutex
}

type OPCUAClient interface {
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool
	ReadNode(nodeID string) (interface{}, error)
	WriteNode(nodeID string, value interface{}) error
	Subscribe(nodeIDs []string, interval float64, handler OPCUADataChangeHandler) (uint32, error)
	Unsubscribe(subscriptionID uint32) error
	Browse(nodeID string) ([]OPCUANode, error)
}

type OPCUADataChangeHandler func(nodeID string, value interface{}, quality uint32, timestamp time.Time)

func NewOPCUAAdapter(config *OPCUAConfig) *OPCUAAdapter {
	return &OPCUAAdapter{
		config:        config,
		stopCh:        make(chan struct{}),
		nodeConfigs:   make(map[uuid.UUID][]OPCUANode),
		subscriptions: make(map[uint32]*OPCUASubscription),
		deviceMapping: make(map[string]uuid.UUID),
	}
}

func (a *OPCUAAdapter) Start(ctx context.Context) error {
	client, err := a.createClient()
	if err != nil {
		return fmt.Errorf("创建OPC-UA客户端失败: %w", err)
	}

	a.client = client

	if err := a.client.Connect(ctx); err != nil {
		return fmt.Errorf("连接OPC-UA服务器失败: %w", err)
	}

	a.setConnected(true)
	log.Printf("[OPCUAAdapter] 已连接到OPC-UA服务器: %s", a.config.Endpoint)

	a.wg.Add(1)
	go a.healthCheckLoop(ctx)

	return nil
}

func (a *OPCUAAdapter) createClient() (OPCUAClient, error) {
	return &mockOPCUAClient{
		endpoint: a.config.Endpoint,
		config:   a.config,
	}, nil
}

func (a *OPCUAAdapter) Stop() error {
	close(a.stopCh)
	a.wg.Wait()

	a.subMu.Lock()
	for subID := range a.subscriptions {
		if a.client != nil {
			a.client.Unsubscribe(subID)
		}
	}
	a.subMu.Unlock()

	if a.client != nil {
		a.client.Disconnect()
	}

	a.setConnected(false)
	log.Println("[OPCUAAdapter] OPC-UA连接已关闭")
	return nil
}

func (a *OPCUAAdapter) Reconnect() error {
	if a.client != nil {
		a.client.Disconnect()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.client.Connect(ctx); err != nil {
		return fmt.Errorf("重连OPC-UA服务器失败: %w", err)
	}

	a.setConnected(true)
	log.Println("[OPCUAAdapter] OPC-UA重连成功")

	a.resubscribeAll()

	return nil
}

func (a *OPCUAAdapter) IsConnected() bool {
	a.connectMu.RLock()
	defer a.connectMu.RUnlock()
	return a.connected && a.client != nil && a.client.IsConnected()
}

func (a *OPCUAAdapter) SetMessageHandler(handler MessageHandler) {
	a.handler = handler
}

func (a *OPCUAAdapter) Subscribe(topic string) error {
	return nil
}

func (a *OPCUAAdapter) Unsubscribe(topic string) error {
	return nil
}

func (a *OPCUAAdapter) Publish(topic string, payload []byte) error {
	return nil
}

func (a *OPCUAAdapter) SendCommand(deviceID uuid.UUID, connectionInfo interface{}, cmd *DeviceCommandRequest) (map[string]interface{}, error) {
	if !a.IsConnected() {
		return nil, fmt.Errorf("OPC-UA未连接")
	}

	switch cmd.CommandName {
	case "write_node":
		return a.writeNode(cmd.Parameters)
	case "write_nodes":
		return a.writeNodes(cmd.Parameters)
	case "read_node":
		return a.readNode(cmd.Parameters)
	case "read_nodes":
		return a.readNodes(cmd.Parameters)
	case "browse":
		return a.browseNode(cmd.Parameters)
	default:
		return nil, fmt.Errorf("不支持的命令: %s", cmd.CommandName)
	}
}

func (a *OPCUAAdapter) RegisterDevice(deviceID uuid.UUID, nodes []OPCUANode, pollRate int) {
	a.nodeMu.Lock()
	a.nodeConfigs[deviceID] = nodes
	a.nodeMu.Unlock()

	a.mappingMu.Lock()
	for _, node := range nodes {
		a.deviceMapping[node.NodeID] = deviceID
	}
	a.mappingMu.Unlock()

	if pollRate > 0 {
		go a.startPolling(deviceID, nodes, pollRate)
	}

	log.Printf("[OPCUAAdapter] 已注册设备 %s，节点数量: %d", deviceID, len(nodes))
}

func (a *OPCUAAdapter) UnregisterDevice(deviceID uuid.UUID) {
	a.nodeMu.Lock()
	nodes, ok := a.nodeConfigs[deviceID]
	if ok {
		a.mappingMu.Lock()
		for _, node := range nodes {
			delete(a.deviceMapping, node.NodeID)
		}
		a.mappingMu.Unlock()
	}
	delete(a.nodeConfigs, deviceID)
	a.nodeMu.Unlock()

	log.Printf("[OPCUAAdapter] 已注销设备 %s", deviceID)
}

func (a *OPCUAAdapter) startPolling(deviceID uuid.UUID, nodes []OPCUANode, pollRate int) {
	ticker := time.NewTicker(time.Duration(pollRate) * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.pollNodes(deviceID, nodes)
		}
	}
}

func (a *OPCUAAdapter) pollNodes(deviceID uuid.UUID, nodes []OPCUANode) {
	if !a.IsConnected() {
		return
	}

	data := make(map[string]interface{})

	for _, node := range nodes {
		value, err := a.client.ReadNode(node.NodeID)
		if err != nil {
			log.Printf("[OPCUAAdapter] 读取节点 %s 失败: %v", node.NodeID, err)
			continue
		}
		data[node.Name] = value
	}

	if a.handler != nil && len(data) > 0 {
		payload := []byte(fmt.Sprintf("%v", data))
		a.handler(fmt.Sprintf("opcua/%s/data", deviceID), payload)
	}
}

func (a *OPCUAAdapter) CreateSubscription(nodeIDs []string, interval float64) (uint32, error) {
	if !a.IsConnected() {
		return 0, fmt.Errorf("OPC-UA未连接")
	}

	subID, err := a.client.Subscribe(nodeIDs, interval, a.handleDataChange)
	if err != nil {
		return 0, fmt.Errorf("创建订阅失败: %w", err)
	}

	a.subMu.Lock()
	a.subscriptions[subID] = &OPCUASubscription{
		SubscriptionID: subID,
		NodeIDs:        nodeIDs,
		Interval:       interval,
	}
	a.subMu.Unlock()

	log.Printf("[OPCUAAdapter] 已创建订阅 %d，节点数量: %d，采样间隔: %.2f秒", subID, len(nodeIDs), interval)
	return subID, nil
}

func (a *OPCUAAdapter) DeleteSubscription(subscriptionID uint32) error {
	if !a.IsConnected() {
		return fmt.Errorf("OPC-UA未连接")
	}

	if err := a.client.Unsubscribe(subscriptionID); err != nil {
		return fmt.Errorf("删除订阅失败: %w", err)
	}

	a.subMu.Lock()
	delete(a.subscriptions, subscriptionID)
	a.subMu.Unlock()

	log.Printf("[OPCUAAdapter] 已删除订阅 %d", subscriptionID)
	return nil
}

func (a *OPCUAAdapter) handleDataChange(nodeID string, value interface{}, quality uint32, timestamp time.Time) {
	a.mappingMu.RLock()
	deviceID, ok := a.deviceMapping[nodeID]
	a.mappingMu.RUnlock()

	if !ok {
		deviceID = uuid.Nil
	}

	if a.handler != nil {
		payload := []byte(fmt.Sprintf(`{"node_id":"%s","device_id":"%s","value":%v,"quality":%d,"timestamp":"%s"}`,
			nodeID, deviceID, value, quality, timestamp.Format(time.RFC3339)))
		a.handler(nodeID, payload)
	}

	log.Printf("[OPCUAAdapter] 节点 %s 数据变化: %v (质量: %d)", nodeID, value, quality)
}

func (a *OPCUAAdapter) resubscribeAll() {
	a.subMu.RLock()
	subs := make(map[uint32]*OPCUASubscription)
	for k, v := range a.subscriptions {
		subs[k] = v
	}
	a.subMu.RUnlock()

	for _, sub := range subs {
		_, err := a.CreateSubscription(sub.NodeIDs, sub.Interval)
		if err != nil {
			log.Printf("[OPCUAAdapter] 重新订阅失败: %v", err)
		}
	}
}

func (a *OPCUAAdapter) writeNode(params map[string]interface{}) (map[string]interface{}, error) {
	nodeID, ok := params["node_id"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少node_id参数")
	}

	value, ok := params["value"]
	if !ok {
		return nil, fmt.Errorf("缺少value参数")
	}

	if err := a.client.WriteNode(nodeID, value); err != nil {
		return nil, fmt.Errorf("写入节点失败: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"node_id": nodeID,
		"value":   value,
	}, nil
}

func (a *OPCUAAdapter) writeNodes(params map[string]interface{}) (map[string]interface{}, error) {
	nodes, ok := params["nodes"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("缺少nodes参数")
	}

	results := make([]map[string]interface{}, 0)
	for _, n := range nodes {
		node, ok := n.(map[string]interface{})
		if !ok {
			continue
		}

		nodeID, _ := node["node_id"].(string)
		value := node["value"]

		err := a.client.WriteNode(nodeID, value)
		results = append(results, map[string]interface{}{
			"node_id": nodeID,
			"success": err == nil,
			"error":   err,
		})
	}

	return map[string]interface{}{
		"success": true,
		"results": results,
	}, nil
}

func (a *OPCUAAdapter) readNode(params map[string]interface{}) (map[string]interface{}, error) {
	nodeID, ok := params["node_id"].(string)
	if !ok {
		return nil, fmt.Errorf("缺少node_id参数")
	}

	value, err := a.client.ReadNode(nodeID)
	if err != nil {
		return nil, fmt.Errorf("读取节点失败: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"node_id": nodeID,
		"value":   value,
	}, nil
}

func (a *OPCUAAdapter) readNodes(params map[string]interface{}) (map[string]interface{}, error) {
	nodeIDs, ok := params["node_ids"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("缺少node_ids参数")
	}

	results := make([]map[string]interface{}, 0)
	for _, n := range nodeIDs {
		nodeID, ok := n.(string)
		if !ok {
			continue
		}

		value, err := a.client.ReadNode(nodeID)
		results = append(results, map[string]interface{}{
			"node_id": nodeID,
			"value":   value,
			"error":   err,
		})
	}

	return map[string]interface{}{
		"success": true,
		"results": results,
	}, nil
}

func (a *OPCUAAdapter) browseNode(params map[string]interface{}) (map[string]interface{}, error) {
	nodeID, _ := params["node_id"].(string)
	if nodeID == "" {
		nodeID = "i=85"
	}

	nodes, err := a.client.Browse(nodeID)
	if err != nil {
		return nil, fmt.Errorf("浏览节点失败: %w", err)
	}

	return map[string]interface{}{
		"success": true,
		"node_id": nodeID,
		"nodes":   nodes,
	}, nil
}

func (a *OPCUAAdapter) healthCheckLoop(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			if !a.client.IsConnected() {
				log.Println("[OPCUAAdapter] 检测到连接断开，尝试重连...")
				if err := a.Reconnect(); err != nil {
					log.Printf("[OPCUAAdapter] 重连失败: %v", err)
				}
			}
		}
	}
}

func (a *OPCUAAdapter) setConnected(connected bool) {
	a.connectMu.Lock()
	defer a.connectMu.Unlock()
	a.connected = connected
}

func (a *OPCUAAdapter) Browse(nodeID string) ([]OPCUANode, error) {
	if !a.IsConnected() {
		return nil, fmt.Errorf("OPC-UA未连接")
	}
	return a.client.Browse(nodeID)
}

func (a *OPCUAAdapter) ReadNode(nodeID string) (interface{}, error) {
	if !a.IsConnected() {
		return nil, fmt.Errorf("OPC-UA未连接")
	}
	return a.client.ReadNode(nodeID)
}

func (a *OPCUAAdapter) WriteNode(nodeID string, value interface{}) error {
	if !a.IsConnected() {
		return fmt.Errorf("OPC-UA未连接")
	}
	return a.client.WriteNode(nodeID, value)
}

func (a *OPCUAAdapter) GetNamespaceIndex(namespaceURI string) (uint16, error) {
	return 0, nil
}

func (a *OPCUAAdapter) BuildNodeID(namespaceIndex uint16, identifier string) string {
	return fmt.Sprintf("ns=%d;s=%s", namespaceIndex, identifier)
}

type mockOPCUAClient struct {
	endpoint  string
	config    *OPCUAConfig
	connected bool
	connectMu sync.RWMutex
	nodeValues map[string]interface{}
	valueMu    sync.RWMutex
}

func (c *mockOPCUAClient) Connect(ctx context.Context) error {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()

	c.connected = true
	c.nodeValues = make(map[string]interface{})

	log.Printf("[MockOPCUAClient] 模拟连接到: %s", c.endpoint)
	return nil
}

func (c *mockOPCUAClient) Disconnect() error {
	c.connectMu.Lock()
	defer c.connectMu.Unlock()

	c.connected = false
	return nil
}

func (c *mockOPCUAClient) IsConnected() bool {
	c.connectMu.RLock()
	defer c.connectMu.RUnlock()
	return c.connected
}

func (c *mockOPCUAClient) ReadNode(nodeID string) (interface{}, error) {
	c.valueMu.RLock()
	defer c.valueMu.RUnlock()

	if val, ok := c.nodeValues[nodeID]; ok {
		return val, nil
	}

	return fmt.Sprintf("mock_value_%s", nodeID), nil
}

func (c *mockOPCUAClient) WriteNode(nodeID string, value interface{}) error {
	c.valueMu.Lock()
	defer c.valueMu.Unlock()

	c.nodeValues[nodeID] = value
	return nil
}

func (c *mockOPCUAClient) Subscribe(nodeIDs []string, interval float64, handler OPCUADataChangeHandler) (uint32, error) {
	subID := uint32(time.Now().Unix())

	go func() {
		ticker := time.NewTicker(time.Duration(interval) * time.Second)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				for _, nodeID := range nodeIDs {
					value, _ := c.ReadNode(nodeID)
					handler(nodeID, value, 0, time.Now())
				}
			}
		}
	}()

	return subID, nil
}

func (c *mockOPCUAClient) Unsubscribe(subscriptionID uint32) error {
	return nil
}

func (c *mockOPCUAClient) Browse(nodeID string) ([]OPCUANode, error) {
	return []OPCUANode{
		{NodeID: "ns=2;s=Device1.Temperature", Name: "Temperature", DataType: "Float"},
		{NodeID: "ns=2;s=Device1.Pressure", Name: "Pressure", DataType: "Float"},
		{NodeID: "ns=2;s=Device1.Status", Name: "Status", DataType: "Boolean"},
	}, nil
}
