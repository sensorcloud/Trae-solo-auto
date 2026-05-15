package iot

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewDeviceManager(t *testing.T) {
	tests := []struct {
		name   string
		config *DeviceConfig
	}{
		{
			name:   "nil config uses defaults",
			config: nil,
		},
		{
			name: "custom config",
			config: &DeviceConfig{
				MaxDevices:         10000,
				DataRetentionDays:  30,
				BatchSize:          100,
				FlushInterval:      5 * time.Second,
				EnableAutoRegister: true,
				EnableDataCache:    true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockIoTRepository()
			manager := NewDeviceManager(repo, tt.config)

			if manager == nil {
				t.Error("expected non-nil DeviceManager")
			}
		})
	}
}

func TestCreateDevice(t *testing.T) {
	connectorID := uuid.New()

	tests := []struct {
		name    string
		device  *Device
		wantErr bool
	}{
		{
			name: "valid device",
			device: &Device{
				Name:         "temperature-sensor-1",
				SerialNumber: "SN001",
				ConnectorID:  connectorID,
				Type:         DeviceTypeSensor,
				Model:        "TMP-100",
				Manufacturer: "Acme",
				TenantID:     uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "device with properties",
			device: &Device{
				Name:         "smart-meter-1",
				SerialNumber: "SN002",
				ConnectorID:  connectorID,
				Type:         DeviceTypeMeter,
				Model:        "SM-200",
				Properties: map[string]interface{}{
					"voltage":    220,
					"frequency":  50,
					"max_power":  10000,
				},
				TenantID: uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "device with existing ID",
			device: &Device{
				ID:           uuid.New(),
				Name:         "existing-device",
				SerialNumber: "SN003",
				ConnectorID:  connectorID,
				Type:         DeviceTypeActuator,
				TenantID:     uuid.New(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := newMockIoTRepository()
			manager := NewDeviceManager(repo, nil)

			err := manager.CreateDevice(ctx, tt.device)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateDevice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.device.ID == uuid.Nil {
					t.Error("expected ID to be set")
				}
				if tt.device.CreatedAt.IsZero() {
					t.Error("expected CreatedAt to be set")
				}
				if tt.device.Status == "" {
					t.Error("expected default status to be set")
				}
			}
		})
	}
}

func TestGetDevice(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewDeviceManager(repo, nil)

	device := &Device{
		Name:         "test-device",
		SerialNumber: "SN001",
		ConnectorID:  uuid.New(),
		Type:         DeviceTypeSensor,
		TenantID:     uuid.New(),
	}

	if err := manager.CreateDevice(ctx, device); err != nil {
		t.Fatalf("failed to create device: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing device",
			id:      device.ID,
			wantErr: false,
		},
		{
			name:    "non-existing device",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := manager.GetDevice(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetDevice() error = %v, wantErr %v", err, tt.wantErr)
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

func TestGetDeviceBySerial(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewDeviceManager(repo, nil)

	device := &Device{
		Name:         "test-device",
		SerialNumber: "SN001",
		ConnectorID:  uuid.New(),
		Type:         DeviceTypeSensor,
		TenantID:     uuid.New(),
	}

	if err := manager.CreateDevice(ctx, device); err != nil {
		t.Fatalf("failed to create device: %v", err)
	}

	tests := []struct {
		name    string
		serial  string
		wantErr bool
	}{
		{
			name:    "existing serial",
			serial:  "SN001",
			wantErr: false,
		},
		{
			name:    "non-existing serial",
			serial:  "SN999",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := manager.GetDeviceBySerial(ctx, tt.serial)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetDeviceBySerial() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if got.SerialNumber != tt.serial {
					t.Errorf("expected SerialNumber %s, got %s", tt.serial, got.SerialNumber)
				}
			}
		})
	}
}

func TestUpdateDevice(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewDeviceManager(repo, nil)

	device := &Device{
		Name:         "test-device",
		SerialNumber: "SN001",
		ConnectorID:  uuid.New(),
		Type:         DeviceTypeSensor,
		TenantID:     uuid.New(),
	}

	if err := manager.CreateDevice(ctx, device); err != nil {
		t.Fatalf("failed to create device: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		updates *DeviceUpdate
		wantErr bool
	}{
		{
			name: "update name",
			id:   device.ID,
			updates: &DeviceUpdate{
				Name: strPtr("updated-device"),
			},
			wantErr: false,
		},
		{
			name: "update status",
			id:   device.ID,
			updates: &DeviceUpdate{
				Status: deviceStatusPtr(DeviceStatusOnline),
			},
			wantErr: false,
		},
		{
			name:    "non-existing device",
			id:      uuid.New(),
			updates: &DeviceUpdate{Name: strPtr("new-name")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.UpdateDevice(ctx, tt.id, tt.updates)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateDevice() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				updated, err := manager.GetDevice(ctx, tt.id)
				if err != nil {
					t.Fatalf("failed to get updated device: %v", err)
				}

				if tt.updates.Name != nil && updated.Name != *tt.updates.Name {
					t.Errorf("expected Name %s, got %s", *tt.updates.Name, updated.Name)
				}
			}
		})
	}
}

func TestDeleteDevice(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewDeviceManager(repo, nil)

	device := &Device{
		Name:         "test-device",
		SerialNumber: "SN001",
		ConnectorID:  uuid.New(),
		Type:         DeviceTypeSensor,
		TenantID:     uuid.New(),
	}

	if err := manager.CreateDevice(ctx, device); err != nil {
		t.Fatalf("failed to create device: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "delete existing device",
			id:      device.ID,
			wantErr: false,
		},
		{
			name:    "delete non-existing device",
			id:      uuid.New(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.DeleteDevice(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteDevice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestListDevices(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewDeviceManager(repo, nil)

	connectorID := uuid.New()

	device1 := &Device{
		Name:         "device-1",
		SerialNumber: "SN001",
		ConnectorID:  connectorID,
		Type:         DeviceTypeSensor,
		Status:       DeviceStatusOnline,
		TenantID:     uuid.New(),
	}
	device2 := &Device{
		Name:         "device-2",
		SerialNumber: "SN002",
		ConnectorID:  connectorID,
		Type:         DeviceTypeActuator,
		Status:       DeviceStatusOffline,
		TenantID:     uuid.New(),
	}
	device3 := &Device{
		Name:         "device-3",
		SerialNumber: "SN003",
		ConnectorID:  uuid.New(),
		Type:         DeviceTypeMeter,
		Status:       DeviceStatusOnline,
		TenantID:     uuid.New(),
	}

	manager.CreateDevice(ctx, device1)
	manager.CreateDevice(ctx, device2)
	manager.CreateDevice(ctx, device3)

	tests := []struct {
		name   string
		filter *DeviceFilter
		count  int
	}{
		{
			name:   "list all devices",
			filter: nil,
			count:  3,
		},
		{
			name:   "filter by connector",
			filter: &DeviceFilter{ConnectorID: connectorID},
			count:  2,
		},
		{
			name:   "filter by online status",
			filter: &DeviceFilter{Status: DeviceStatusOnline},
			count:  2,
		},
		{
			name:   "filter by offline status",
			filter: &DeviceFilter{Status: DeviceStatusOffline},
			count:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			devices, err := manager.ListDevices(ctx, tt.filter)

			if err != nil {
				t.Errorf("ListDevices() error = %v", err)
				return
			}

			if len(devices) != tt.count {
				t.Errorf("expected %d devices, got %d", tt.count, len(devices))
			}
		})
	}
}

func TestReportDeviceData(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewDeviceManager(repo, nil)

	device := &Device{
		Name:         "test-device",
		SerialNumber: "SN001",
		ConnectorID:  uuid.New(),
		Type:         DeviceTypeSensor,
		Status:       DeviceStatusOnline,
		TenantID:     uuid.New(),
	}

	if err := manager.CreateDevice(ctx, device); err != nil {
		t.Fatalf("failed to create device: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		data    *DeviceData
		wantErr bool
	}{
		{
			name: "valid data report",
			id:   device.ID,
			data: &DeviceData{
				Timestamp: time.Now(),
				Values: map[string]interface{}{
					"temperature": 25.5,
					"humidity":    60.0,
				},
			},
			wantErr: false,
		},
		{
			name: "data with metadata",
			id:   device.ID,
			data: &DeviceData{
				Timestamp: time.Now(),
				Values: map[string]interface{}{
					"power": 1500.0,
				},
				Metadata: map[string]string{
					"unit":  "W",
					"quality": "good",
				},
			},
			wantErr: false,
		},
		{
			name: "non-existing device",
			id:   uuid.New(),
			data: &DeviceData{
				Timestamp: time.Now(),
				Values:    map[string]interface{}{"value": 1.0},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.ReportDeviceData(ctx, tt.id, tt.data)

			if (err != nil) != tt.wantErr {
				t.Errorf("ReportDeviceData() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetDeviceData(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewDeviceManager(repo, nil)

	device := &Device{
		Name:         "test-device",
		SerialNumber: "SN001",
		ConnectorID:  uuid.New(),
		Type:         DeviceTypeSensor,
		Status:       DeviceStatusOnline,
		TenantID:     uuid.New(),
	}

	if err := manager.CreateDevice(ctx, device); err != nil {
		t.Fatalf("failed to create device: %v", err)
	}

	now := time.Now()
	start := now.Add(-24 * time.Hour)
	end := now

	tests := []struct {
		name    string
		id      uuid.UUID
		start   time.Time
		end     time.Time
		wantErr bool
	}{
		{
			name:    "valid time range",
			id:      device.ID,
			start:   start,
			end:     end,
			wantErr: false,
		},
		{
			name:    "non-existing device",
			id:      uuid.New(),
			start:   start,
			end:     end,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := manager.GetDeviceData(ctx, tt.id, tt.start, tt.end)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetDeviceData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && data == nil {
				t.Error("expected non-nil data")
			}
		})
	}
}

func TestGetLatestDeviceData(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewDeviceManager(repo, nil)

	device := &Device{
		Name:         "test-device",
		SerialNumber: "SN001",
		ConnectorID:  uuid.New(),
		Type:         DeviceTypeSensor,
		Status:       DeviceStatusOnline,
		TenantID:     uuid.New(),
	}

	if err := manager.CreateDevice(ctx, device); err != nil {
		t.Fatalf("failed to create device: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		key     string
		wantErr bool
	}{
		{
			name:    "existing device",
			id:      device.ID,
			key:     "temperature",
			wantErr: false,
		},
		{
			name:    "non-existing device",
			id:      uuid.New(),
			key:     "temperature",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := manager.GetLatestDeviceData(ctx, tt.id, tt.key)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetLatestDeviceData() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && data == nil {
				t.Error("expected non-nil data")
			}
		})
	}
}

func TestSendDeviceCommand(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewDeviceManager(repo, nil)

	device := &Device{
		Name:         "test-device",
		SerialNumber: "SN001",
		ConnectorID:  uuid.New(),
		Type:         DeviceTypeActuator,
		Status:       DeviceStatusOnline,
		TenantID:     uuid.New(),
	}

	if err := manager.CreateDevice(ctx, device); err != nil {
		t.Fatalf("failed to create device: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		command *DeviceCommand
		wantErr bool
	}{
		{
			name: "valid command",
			id:   device.ID,
			command: &DeviceCommand{
				Name: "set_power",
				Params: map[string]interface{}{
					"power": 1000,
				},
			},
			wantErr: false,
		},
		{
			name: "command with timeout",
			id:   device.ID,
			command: &DeviceCommand{
				Name:    "reset",
				Timeout: 30 * time.Second,
			},
			wantErr: false,
		},
		{
			name: "non-existing device",
			id:   uuid.New(),
			command: &DeviceCommand{
				Name: "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := manager.SendDeviceCommand(ctx, tt.id, tt.command)

			if (err != nil) != tt.wantErr {
				t.Errorf("SendDeviceCommand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == nil {
					t.Error("expected non-nil result")
					return
				}
				if result.DeviceID != tt.id {
					t.Errorf("expected DeviceID %s, got %s", tt.id, result.DeviceID)
				}
			}
		})
	}
}

func TestGetDeviceStatus(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewDeviceManager(repo, nil)

	device := &Device{
		Name:         "test-device",
		SerialNumber: "SN001",
		ConnectorID:  uuid.New(),
		Type:         DeviceTypeSensor,
		Status:       DeviceStatusOnline,
		TenantID:     uuid.New(),
	}

	if err := manager.CreateDevice(ctx, device); err != nil {
		t.Fatalf("failed to create device: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing device",
			id:      device.ID,
			wantErr: false,
		},
		{
			name:    "non-existing device",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			status, err := manager.GetDeviceStatus(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetDeviceStatus() error = %v, wantErr %v", err, tt.wantErr)
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

func TestSetDeviceProperties(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewDeviceManager(repo, nil)

	device := &Device{
		Name:         "test-device",
		SerialNumber: "SN001",
		ConnectorID:  uuid.New(),
		Type:         DeviceTypeSensor,
		TenantID:     uuid.New(),
	}

	if err := manager.CreateDevice(ctx, device); err != nil {
		t.Fatalf("failed to create device: %v", err)
	}

	tests := []struct {
		name       string
		id         uuid.UUID
		properties map[string]interface{}
		wantErr    bool
	}{
		{
			name: "valid properties",
			id:   device.ID,
			properties: map[string]interface{}{
				"interval":  10,
				"threshold": 100,
			},
			wantErr: false,
		},
		{
			name:       "empty properties",
			id:         device.ID,
			properties: map[string]interface{}{},
			wantErr:    false,
		},
		{
			name: "non-existing device",
			id:   uuid.New(),
			properties: map[string]interface{}{
				"key": "value",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.SetDeviceProperties(ctx, tt.id, tt.properties)

			if (err != nil) != tt.wantErr {
				t.Errorf("SetDeviceProperties() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetDeviceStatistics(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewDeviceManager(repo, nil)

	device := &Device{
		Name:         "test-device",
		SerialNumber: "SN001",
		ConnectorID:  uuid.New(),
		Type:         DeviceTypeSensor,
		Status:       DeviceStatusOnline,
		TenantID:     uuid.New(),
	}

	if err := manager.CreateDevice(ctx, device); err != nil {
		t.Fatalf("failed to create device: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing device",
			id:      device.ID,
			wantErr: false,
		},
		{
			name:    "non-existing device",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := manager.GetDeviceStatistics(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetDeviceStatistics() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if stats == nil {
					t.Error("expected non-nil statistics")
					return
				}
				if stats.DeviceID != tt.id {
					t.Errorf("expected DeviceID %s, got %s", tt.id, stats.DeviceID)
				}
			}
		})
	}
}

func TestDeviceTypes(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewDeviceManager(repo, nil)

	tests := []struct {
		name      string
		deviceType DeviceType
		wantErr   bool
	}{
		{
			name:      "sensor device",
			deviceType: DeviceTypeSensor,
			wantErr:   false,
		},
		{
			name:      "actuator device",
			deviceType: DeviceTypeActuator,
			wantErr:   false,
		},
		{
			name:      "meter device",
			deviceType: DeviceTypeMeter,
			wantErr:   false,
		},
		{
			name:      "gateway device",
			deviceType: DeviceTypeGateway,
			wantErr:   false,
		},
		{
			name:      "controller device",
			deviceType: DeviceTypeController,
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := &Device{
				Name:         string(tt.deviceType) + "-device",
				SerialNumber: string(tt.deviceType) + "-SN",
				ConnectorID:  uuid.New(),
				Type:         tt.deviceType,
				TenantID:     uuid.New(),
			}

			err := manager.CreateDevice(ctx, device)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateDevice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDeviceStatusTransitions(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewDeviceManager(repo, nil)

	tests := []struct {
		name          string
		initialStatus DeviceStatus
		newStatus     DeviceStatus
		wantErr       bool
	}{
		{
			name:          "offline to online",
			initialStatus: DeviceStatusOffline,
			newStatus:     DeviceStatusOnline,
			wantErr:       false,
		},
		{
			name:          "online to offline",
			initialStatus: DeviceStatusOnline,
			newStatus:     DeviceStatusOffline,
			wantErr:       false,
		},
		{
			name:          "online to maintenance",
			initialStatus: DeviceStatusOnline,
			newStatus:     DeviceStatusMaintenance,
			wantErr:       false,
		},
		{
			name:          "maintenance to online",
			initialStatus: DeviceStatusMaintenance,
			newStatus:     DeviceStatusOnline,
			wantErr:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			device := &Device{
				Name:         "test-device",
				SerialNumber: "SN-" + tt.name,
				ConnectorID:  uuid.New(),
				Type:         DeviceTypeSensor,
				Status:       tt.initialStatus,
				TenantID:     uuid.New(),
			}

			if err := manager.CreateDevice(ctx, device); err != nil {
				t.Fatalf("failed to create device: %v", err)
			}

			err := manager.UpdateDevice(ctx, device.ID, &DeviceUpdate{
				Status: &tt.newStatus,
			})

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateDevice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestBatchDeviceOperations(t *testing.T) {
	ctx := context.Background()
	repo := newMockIoTRepository()
	manager := NewDeviceManager(repo, nil)

	connectorID := uuid.New()

	devices := []*Device{
		{
			Name:         "batch-1",
			SerialNumber: "BATCH-SN-001",
			ConnectorID:  connectorID,
			Type:         DeviceTypeSensor,
			TenantID:     uuid.New(),
		},
		{
			Name:         "batch-2",
			SerialNumber: "BATCH-SN-002",
			ConnectorID:  connectorID,
			Type:         DeviceTypeSensor,
			TenantID:     uuid.New(),
		},
		{
			Name:         "batch-3",
			SerialNumber: "BATCH-SN-003",
			ConnectorID:  connectorID,
			Type:         DeviceTypeSensor,
			TenantID:     uuid.New(),
		},
	}

	tests := []struct {
		name    string
		devices []*Device
		wantErr bool
	}{
		{
			name:    "valid batch",
			devices: devices,
			wantErr: false,
		},
		{
			name:    "empty batch",
			devices: []*Device{},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ids, err := manager.CreateBatchDevices(ctx, tt.devices)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateBatchDevices() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(ids) != len(tt.devices) {
					t.Errorf("expected %d IDs, got %d", len(tt.devices), len(ids))
				}
			}
		})
	}
}

func deviceStatusPtr(s DeviceStatus) *DeviceStatus {
	return &s
}
