package iot

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockIoTRepository struct {
	connectors  map[uuid.UUID]*Connector
	devices     map[uuid.UUID]*Device
	dataPoints  map[uuid.UUID]*DataPoint
	rules       map[uuid.UUID]*Rule
	alarms      map[uuid.UUID]*Alarm
	connections map[uuid.UUID]*Connection
}

func newMockIoTRepository() *mockIoTRepository {
	return &mockIoTRepository{
		connectors:  make(map[uuid.UUID]*Connector),
		devices:     make(map[uuid.UUID]*Device),
		dataPoints:  make(map[uuid.UUID]*DataPoint),
		rules:       make(map[uuid.UUID]*Rule),
		alarms:      make(map[uuid.UUID]*Alarm),
		connections: make(map[uuid.UUID]*Connection),
	}
}

func (m *mockIoTRepository) CreateConnector(ctx context.Context, connector *Connector) error {
	m.connectors[connector.ID] = connector
	return nil
}

func (m *mockIoTRepository) GetConnector(ctx context.Context, id uuid.UUID) (*Connector, error) {
	if connector, ok := m.connectors[id]; ok {
		return connector, nil
	}
	return nil, fmt.Errorf("connector not found")
}

func (m *mockIoTRepository) ListConnectors(ctx context.Context, filter *ConnectorFilter) ([]*Connector, error) {
	result := make([]*Connector, 0)
	for _, connector := range m.connectors {
		if filter != nil {
			if filter.Type != "" && connector.Type != filter.Type {
				continue
			}
			if filter.Status != "" && connector.Status != filter.Status {
				continue
			}
		}
		result = append(result, connector)
	}
	return result, nil
}

func (m *mockIoTRepository) UpdateConnector(ctx context.Context, connector *Connector) error {
	m.connectors[connector.ID] = connector
	return nil
}

func (m *mockIoTRepository) DeleteConnector(ctx context.Context, id uuid.UUID) error {
	delete(m.connectors, id)
	return nil
}

func (m *mockIoTRepository) CreateDevice(ctx context.Context, device *Device) error {
	m.devices[device.ID] = device
	return nil
}

func (m *mockIoTRepository) GetDevice(ctx context.Context, id uuid.UUID) (*Device, error) {
	if device, ok := m.devices[id]; ok {
		return device, nil
	}
	return nil, fmt.Errorf("device not found")
}

func (m *mockIoTRepository) GetDeviceBySerial(ctx context.Context, serial string) (*Device, error) {
	for _, device := range m.devices {
		if device.SerialNumber == serial {
			return device, nil
		}
	}
	return nil, fmt.Errorf("device not found")
}

func (m *mockIoTRepository) ListDevices(ctx context.Context, filter *DeviceFilter) ([]*Device, error) {
	result := make([]*Device, 0)
	for _, device := range m.devices {
		if filter != nil {
			if filter.Status != "" && device.Status != filter.Status {
				continue
			}
			if filter.ConnectorID != uuid.Nil && device.ConnectorID != filter.ConnectorID {
				continue
			}
		}
		result = append(result, device)
	}
	return result, nil
}

func (m *mockIoTRepository) UpdateDevice(ctx context.Context, device *Device) error {
	m.devices[device.ID] = device
	return nil
}

func (m *mockIoTRepository) DeleteDevice(ctx context.Context, id uuid.UUID) error {
	delete(m.devices, id)
	return nil
}

func (m *mockIoTRepository) CreateDataPoint(ctx context.Context, point *DataPoint) error {
	m.dataPoints[point.ID] = point
	return nil
}

func (m *mockIoTRepository) GetDataPoints(ctx context.Context, deviceID uuid.UUID, start, end time.Time) ([]*DataPoint, error) {
	result := make([]*DataPoint, 0)
	for _, point := range m.dataPoints {
		if point.DeviceID == deviceID {
			if (start.IsZero() || point.Timestamp.After(start)) && (end.IsZero() || point.Timestamp.Before(end)) {
				result = append(result, point)
			}
		}
	}
	return result, nil
}

func (m *mockIoTRepository) GetLatestDataPoint(ctx context.Context, deviceID uuid.UUID, key string) (*DataPoint, error) {
	for _, point := range m.dataPoints {
		if point.DeviceID == deviceID && point.Key == key {
			return point, nil
		}
	}
	return nil, fmt.Errorf("data point not found")
}

func (m *mockIoTRepository) CreateRule(ctx context.Context, rule *Rule) error {
	m.rules[rule.ID] = rule
	return nil
}

func (m *mockIoTRepository) GetRule(ctx context.Context, id uuid.UUID) (*Rule, error) {
	if rule, ok := m.rules[id]; ok {
		return rule, nil
	}
	return nil, fmt.Errorf("rule not found")
}

func (m *mockIoTRepository) ListRules(ctx context.Context, filter *RuleFilter) ([]*Rule, error) {
	result := make([]*Rule, 0)
	for _, rule := range m.rules {
		result = append(result, rule)
	}
	return result, nil
}

func (m *mockIoTRepository) UpdateRule(ctx context.Context, rule *Rule) error {
	m.rules[rule.ID] = rule
	return nil
}

func (m *mockIoTRepository) DeleteRule(ctx context.Context, id uuid.UUID) error {
	delete(m.rules, id)
	return nil
}

func (m *mockIoTRepository) CreateAlarm(ctx context.Context, alarm *Alarm) error {
	m.alarms[alarm.ID] = alarm
	return nil
}

func (m *mockIoTRepository) GetAlarm(ctx context.Context, id uuid.UUID) (*Alarm, error) {
	if alarm, ok := m.alarms[id]; ok {
		return alarm, nil
	}
	return nil, fmt.Errorf("alarm not found")
}

func (m *mockIoTRepository) ListAlarms(ctx context.Context, filter *AlarmFilter) ([]*Alarm, error) {
	result := make([]*Alarm, 0)
	for _, alarm := range m.alarms {
		if filter != nil {
			if filter.Status != "" && alarm.Status != filter.Status {
				continue
			}
		}
		result = append(result, alarm)
	}
	return result, nil
}

func (m *mockIoTRepository) UpdateAlarm(ctx context.Context, alarm *Alarm) error {
	m.alarms[alarm.ID] = alarm
	return nil
}

func (m *mockIoTRepository) CreateConnection(ctx context.Context, conn *Connection) error {
	m.connections[conn.ID] = conn
	return nil
}

func (m *mockIoTRepository) GetConnection(ctx context.Context, id uuid.UUID) (*Connection, error) {
	if conn, ok := m.connections[id]; ok {
		return conn, nil
	}
	return nil, fmt.Errorf("connection not found")
}

func (m *mockIoTRepository) ListConnections(ctx context.Context, filter *ConnectionFilter) ([]*Connection, error) {
	result := make([]*Connection, 0)
	for _, conn := range m.connections {
		result = append(result, conn)
	}
	return result, nil
}

func (m *mockIoTRepository) UpdateConnection(ctx context.Context, conn *Connection) error {
	m.connections[conn.ID] = conn
	return nil
}

func TestNewConnectorManager(t *testing.T) {
	tests := []struct {
		name   string
		config *ConnectorConfig
	}{
		{
			name:   "nil config uses defaults",
			config: nil,
		},
		{
			name: "custom config",
			config: &ConnectorConfig{
				MaxConnectors:      100,
				ReconnectInterval:  30 * time.Second,
				HeartbeatInterval:  10 * time.Second,
				Timeout:            5 * time.Second,
				EnableAutoReconnect: true,
				EnableTLS:          true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockIoTRepository()
			manager := NewConnectorManager(repo, tt.config)

			if manager == nil {
				t.Error("expected non-nil ConnectorManager")
			}
		})
	}
}

func TestCreateConnector(t *testing.T) {
	tests := []struct {
		name      string
		connector *Connector
		wantErr   bool
	}{
		{
			name: "valid MQTT connector",
			connector: &Connector{
				Name:     "mqtt-connector-1",
				Type:     ConnectorTypeMQTT,
				Protocol: "mqtt",
				Endpoint: "tcp://localhost:1883",
				Config: ConnectorConfigData{
					Username: "user",
					Password: "pass",
					ClientID: "client-1",
				},
				TenantID: uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "valid Modbus connector",
			connector: &Connector{
				Name:     "modbus-connector-1",
				Type:     ConnectorTypeModbus,
				Protocol: "modbus-tcp",
				Endpoint: "192.168.1.100:502",
				Config: ConnectorConfigData{
					SlaveID: 1,
				},
				TenantID: uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "connector with existing ID",
			connector: &Connector{
				ID:       uuid.New(),
				Name:     "existing-connector",
				Type:     ConnectorTypeOPCUA,
				Protocol: "opcua",
				Endpoint: "opc.tcp://localhost:4840",
				TenantID: uuid.New(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := newMockIoTRepository()
			manager := NewConnectorManager(repo, nil)

			err := manager.CreateConnector(ctx, tt.connector)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateConnector() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.connector.ID == uuid.Nil {
					t.Error("expected ID to be set")
				}
				if tt.connector.CreatedAt.IsZero() {
					t.Error("expected CreatedAt to be set")
				}
				if tt.connector.Status == "" {
					t.Error("expected default status to be set")
				}
			}
		})
	}
}

func TestGetConnector(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewConnectorManager(repo, nil)

	connector := &Connector{
		Name:     "test-connector",
		Type:     ConnectorTypeMQTT,
		Protocol: "mqtt",
		Endpoint: "tcp://localhost:1883",
		TenantID: uuid.New(),
	}

	if err := manager.CreateConnector(ctx, connector); err != nil {
		t.Fatalf("failed to create connector: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing connector",
			id:      connector.ID,
			wantErr: false,
		},
		{
			name:    "non-existing connector",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := manager.GetConnector(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetConnector() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.ID != tt.id {
					t.Errorf("expected ID %s, got %s", tt.id, got.ID)
				}
			}
		})
	}
}

func TestUpdateConnector(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewConnectorManager(repo, nil)

	connector := &Connector{
		Name:     "test-connector",
		Type:     ConnectorTypeMQTT,
		Protocol: "mqtt",
		Endpoint: "tcp://localhost:1883",
		TenantID: uuid.New(),
	}

	if err := manager.CreateConnector(ctx, connector); err != nil {
		t.Fatalf("failed to create connector: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		updates *ConnectorUpdate
		wantErr bool
	}{
		{
			name: "update endpoint",
			id:   connector.ID,
			updates: &ConnectorUpdate{
				Endpoint: strPtr("tcp://newhost:1883"),
			},
			wantErr: false,
		},
		{
			name: "update status",
			id:   connector.ID,
			updates: &ConnectorUpdate{
				Status: connectorStatusPtr(ConnectorStatusConnected),
			},
			wantErr: false,
		},
		{
			name:    "non-existing connector",
			id:      uuid.New(),
			updates: &ConnectorUpdate{Endpoint: strPtr("tcp://host:1883")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.UpdateConnector(ctx, tt.id, tt.updates)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateConnector() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				updated, err := manager.GetConnector(ctx, tt.id)
				if err != nil {
					t.Fatalf("failed to get updated connector: %v", err)
				}

				if tt.updates.Endpoint != nil && updated.Endpoint != *tt.updates.Endpoint {
					t.Errorf("expected Endpoint %s, got %s", *tt.updates.Endpoint, updated.Endpoint)
				}
			}
		})
	}
}

func TestDeleteConnector(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewConnectorManager(repo, nil)

	connector := &Connector{
		Name:     "test-connector",
		Type:     ConnectorTypeMQTT,
		Protocol: "mqtt",
		Endpoint: "tcp://localhost:1883",
		TenantID: uuid.New(),
	}

	if err := manager.CreateConnector(ctx, connector); err != nil {
		t.Fatalf("failed to create connector: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "delete existing connector",
			id:      connector.ID,
			wantErr: false,
		},
		{
			name:    "delete non-existing connector",
			id:      uuid.New(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.DeleteConnector(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteConnector() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConnectConnector(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewConnectorManager(repo, nil)

	connector := &Connector{
		Name:     "test-connector",
		Type:     ConnectorTypeMQTT,
		Protocol: "mqtt",
		Endpoint: "tcp://localhost:1883",
		Status:   ConnectorStatusDisconnected,
		TenantID: uuid.New(),
	}

	if err := manager.CreateConnector(ctx, connector); err != nil {
		t.Fatalf("failed to create connector: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "connect valid connector",
			id:      connector.ID,
			wantErr: false,
		},
		{
			name:    "connect non-existing connector",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.ConnectConnector(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("ConnectConnector() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDisconnectConnector(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewConnectorManager(repo, nil)

	connector := &Connector{
		Name:     "test-connector",
		Type:     ConnectorTypeMQTT,
		Protocol: "mqtt",
		Endpoint: "tcp://localhost:1883",
		Status:   ConnectorStatusConnected,
		TenantID: uuid.New(),
	}

	if err := manager.CreateConnector(ctx, connector); err != nil {
		t.Fatalf("failed to create connector: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "disconnect connected connector",
			id:      connector.ID,
			wantErr: false,
		},
		{
			name:    "disconnect non-existing connector",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.DisconnectConnector(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("DisconnectConnector() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetConnectorStatus(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewConnectorManager(repo, nil)

	connector := &Connector{
		Name:     "test-connector",
		Type:     ConnectorTypeMQTT,
		Protocol: "mqtt",
		Endpoint: "tcp://localhost:1883",
		Status:   ConnectorStatusConnected,
		TenantID: uuid.New(),
	}

	if err := manager.CreateConnector(ctx, connector); err != nil {
		t.Fatalf("failed to create connector: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing connector",
			id:      connector.ID,
			wantErr: false,
		},
		{
			name:    "non-existing connector",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := manager.GetConnectorStatus(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetConnectorStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if status == nil {
					t.Error("expected non-nil status")
					return
				}
				if status.ID != tt.id {
					t.Errorf("expected ID %s, got %s", tt.id, status.ID)
				}
			}
		})
	}
}

func TestListConnectors(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewConnectorManager(repo, nil)

	connector1 := &Connector{
		Name:     "connector-1",
		Type:     ConnectorTypeMQTT,
		Protocol: "mqtt",
		Endpoint: "tcp://localhost:1883",
		Status:   ConnectorStatusConnected,
		TenantID: uuid.New(),
	}
	connector2 := &Connector{
		Name:     "connector-2",
		Type:     ConnectorTypeModbus,
		Protocol: "modbus-tcp",
		Endpoint: "192.168.1.100:502",
		Status:   ConnectorStatusDisconnected,
		TenantID: uuid.New(),
	}

	manager.CreateConnector(ctx, connector1)
	manager.CreateConnector(ctx, connector2)

	tests := []struct {
		name   string
		filter *ConnectorFilter
		count  int
	}{
		{
			name:   "list all connectors",
			filter: nil,
			count:  2,
		},
		{
			name:   "filter by MQTT type",
			filter: &ConnectorFilter{Type: ConnectorTypeMQTT},
			count:  1,
		},
		{
			name:   "filter by connected status",
			filter: &ConnectorFilter{Status: ConnectorStatusConnected},
			count:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connectors, err := manager.ListConnectors(ctx, tt.filter)

			if err != nil {
				t.Errorf("ListConnectors() error = %v", err)
				return
			}

			if len(connectors) != tt.count {
				t.Errorf("expected %d connectors, got %d", tt.count, len(connectors))
			}
		})
	}
}

func TestTestConnectorConnection(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewConnectorManager(repo, nil)

	connector := &Connector{
		Name:     "test-connector",
		Type:     ConnectorTypeMQTT,
		Protocol: "mqtt",
		Endpoint: "tcp://localhost:1883",
		TenantID: uuid.New(),
	}

	if err := manager.CreateConnector(ctx, connector); err != nil {
		t.Fatalf("failed to create connector: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "test existing connector",
			id:      connector.ID,
			wantErr: false,
		},
		{
			name:    "test non-existing connector",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.TestConnectorConnection(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("TestConnectorConnection() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == nil {
					t.Error("expected non-nil result")
					return
				}
				if result.ConnectorID != tt.id {
					t.Errorf("expected ConnectorID %s, got %s", tt.id, result.ConnectorID)
				}
			}
		})
	}
}

func TestGetConnectorMetrics(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewConnectorManager(repo, nil)

	connector := &Connector{
		Name:     "test-connector",
		Type:     ConnectorTypeMQTT,
		Protocol: "mqtt",
		Endpoint: "tcp://localhost:1883",
		Status:   ConnectorStatusConnected,
		TenantID: uuid.New(),
	}

	if err := manager.CreateConnector(ctx, connector); err != nil {
		t.Fatalf("failed to create connector: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing connector",
			id:      connector.ID,
			wantErr: false,
		},
		{
			name:    "non-existing connector",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			metrics, err := manager.GetConnectorMetrics(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetConnectorMetrics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if metrics == nil {
					t.Error("expected non-nil metrics")
					return
				}
				if metrics.ConnectorID != tt.id {
					t.Errorf("expected ConnectorID %s, got %s", tt.id, metrics.ConnectorID)
				}
			}
		})
	}
}

func TestConnectorTypes(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewConnectorManager(repo, nil)

	tests := []struct {
		name     string
		connType ConnectorType
		wantErr  bool
	}{
		{
			name:     "MQTT connector",
			connType: ConnectorTypeMQTT,
			wantErr:  false,
		},
		{
			name:     "Modbus connector",
			connType: ConnectorTypeModbus,
			wantErr:  false,
		},
		{
			name:     "OPCUA connector",
			connType: ConnectorTypeOPCUA,
			wantErr:  false,
		},
		{
			name:     "HTTP connector",
			connType: ConnectorTypeHTTP,
			wantErr:  false,
		},
		{
			name:     "CoAP connector",
			connType: ConnectorTypeCoAP,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connector := &Connector{
				Name:     string(tt.connType) + "-connector",
				Type:     tt.connType,
				Protocol: string(tt.connType),
				Endpoint: "localhost",
				TenantID: uuid.New(),
			}

			err := manager.CreateConnector(ctx, connector)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateConnector() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestConnectorStatusTransitions(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewConnectorManager(repo, nil)

	tests := []struct {
		name          string
		initialStatus ConnectorStatus
		action        func(uuid.UUID) error
		wantErr       bool
	}{
		{
			name:          "connect from disconnected",
			initialStatus: ConnectorStatusDisconnected,
			action: func(id uuid.UUID) error {
				return manager.ConnectConnector(ctx, id)
			},
			wantErr: false,
		},
		{
			name:          "disconnect from connected",
			initialStatus: ConnectorStatusConnected,
			action: func(id uuid.UUID) error {
				return manager.DisconnectConnector(ctx, id)
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			connector := &Connector{
				Name:     "test-connector",
				Type:     ConnectorTypeMQTT,
				Protocol: "mqtt",
				Endpoint: "tcp://localhost:1883",
				Status:   tt.initialStatus,
				TenantID: uuid.New(),
			}

			if err := manager.CreateConnector(ctx, connector); err != nil {
				t.Fatalf("failed to create connector: %v", err)
			}

			err := tt.action(connector.ID)

			if (err != nil) != tt.wantErr {
				t.Errorf("action() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func connectorStatusPtr(s ConnectorStatus) *ConnectorStatus {
	return &s
}
