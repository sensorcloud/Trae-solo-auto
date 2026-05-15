package energy

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestNewStorageManager(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "default config",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockEnergyRepository()
			core := NewEnergyMarketCore(repo, nil)
			manager := NewStorageManager(repo, core)

			if manager == nil {
				t.Error("expected non-nil StorageManager")
			}
		})
	}
}

func TestCreateStorageDevice(t *testing.T) {
	tests := []struct {
		name    string
		device  *StorageDevice
		wantErr bool
	}{
		{
			name: "valid battery storage",
			device: &StorageDevice{
				Name:             "Battery 1",
				Capacity:         1000,
				SOC:              50,
				MaxChargeRate:    500,
				MaxDischargeRate: 500,
				Location:         Location{Region: "cn-east"},
				TenantID:         uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "storage with existing ID",
			device: &StorageDevice{
				Name:             "Existing Battery",
				Capacity:         500,
				SOC:              50,
				MaxChargeRate:    250,
				MaxDischargeRate: 250,
				TenantID:         uuid.New(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := newMockEnergyRepository()
			core := NewEnergyMarketCore(repo, nil)
			manager := NewStorageManager(repo, core)

			err := manager.CreateStorageDevice(ctx, tt.device)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateStorageDevice() error = %v, wantErr %v", err, tt.wantErr)
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

func TestGetStorageDevice(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	manager := NewStorageManager(repo, core)

	device := &StorageDevice{
		Name:             "Test Battery",
		Capacity:         1000,
		SOC:              50,
		MaxChargeRate:    500,
		MaxDischargeRate: 500,
		TenantID:         uuid.New(),
	}

	if err := manager.CreateStorageDevice(ctx, device); err != nil {
		t.Fatalf("failed to create storage device: %v", err)
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
			got, err := manager.GetStorageDevice(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetStorageDevice() error = %v, wantErr %v", err, tt.wantErr)
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

func TestUpdateStorageDevice(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	manager := NewStorageManager(repo, core)

	device := &StorageDevice{
		Name:             "Test Battery",
		Capacity:         1000,
		SOC:              50,
		MaxChargeRate:    500,
		MaxDischargeRate: 500,
		TenantID:         uuid.New(),
	}

	if err := manager.CreateStorageDevice(ctx, device); err != nil {
		t.Fatalf("failed to create storage device: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		updates *StorageUpdate
		wantErr bool
	}{
		{
			name: "update max charge rate",
			id:   device.ID,
			updates: &StorageUpdate{
				MaxChargeRate: float64Ptr(600),
			},
			wantErr: false,
		},
		{
			name: "update status",
			id:   device.ID,
			updates: &StorageUpdate{
				Status: storageStatusPtr(StorageStatusCharging),
			},
			wantErr: false,
		},
		{
			name:    "non-existing device",
			id:      uuid.New(),
			updates: &StorageUpdate{MaxChargeRate: float64Ptr(600)},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.UpdateStorageDevice(ctx, tt.id, tt.updates)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateStorageDevice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChargeStorage(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	manager := NewStorageManager(repo, core)

	device := &StorageDevice{
		Name:             "Test Battery",
		Capacity:         1000,
		SOC:              30,
		MinSOC:           10,
		MaxSOC:           95,
		MaxChargeRate:    500,
		MaxDischargeRate: 500,
		Status:           StorageStatusIdle,
		TenantID:         uuid.New(),
	}

	if err := manager.CreateStorageDevice(ctx, device); err != nil {
		t.Fatalf("failed to create storage device: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		power   float64
		wantErr bool
	}{
		{
			name:    "valid charge",
			id:      device.ID,
			power:   200,
			wantErr: false,
		},
		{
			name:    "charge exceeds max rate",
			id:      device.ID,
			power:   1000,
			wantErr: false,
		},
		{
			name:    "non-existing device",
			id:      uuid.New(),
			power:   100,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.Charge(ctx, tt.id, tt.power)

			if (err != nil) != tt.wantErr {
				t.Errorf("Charge() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDischargeStorage(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	manager := NewStorageManager(repo, core)

	device := &StorageDevice{
		Name:             "Test Battery",
		Capacity:         1000,
		SOC:              80,
		MinSOC:           10,
		MaxSOC:           95,
		MaxChargeRate:    500,
		MaxDischargeRate: 500,
		Status:           StorageStatusIdle,
		TenantID:         uuid.New(),
	}

	if err := manager.CreateStorageDevice(ctx, device); err != nil {
		t.Fatalf("failed to create storage device: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		power   float64
		wantErr bool
	}{
		{
			name:    "valid discharge",
			id:      device.ID,
			power:   200,
			wantErr: false,
		},
		{
			name:    "discharge exceeds max rate",
			id:      device.ID,
			power:   1000,
			wantErr: false,
		},
		{
			name:    "non-existing device",
			id:      uuid.New(),
			power:   100,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.Discharge(ctx, tt.id, tt.power)

			if (err != nil) != tt.wantErr {
				t.Errorf("Discharge() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetStorageStatus(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	manager := NewStorageManager(repo, core)

	device := &StorageDevice{
		Name:             "Test Battery",
		Capacity:         1000,
		SOC:              50,
		MaxChargeRate:    500,
		MaxDischargeRate: 500,
		Status:           StorageStatusIdle,
		TenantID:         uuid.New(),
	}

	if err := manager.CreateStorageDevice(ctx, device); err != nil {
		t.Fatalf("failed to create storage device: %v", err)
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
			status, err := manager.GetStorageStatus(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetStorageStatus() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if status == nil {
					t.Error("expected non-nil status")
					return
				}
				if status.DeviceID != tt.id {
					t.Errorf("expected DeviceID %s, got %s", tt.id, status.DeviceID)
				}
			}
		})
	}
}

func TestListStorageDevices(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	manager := NewStorageManager(repo, core)

	device1 := &StorageDevice{
		Name:             "Battery 1",
		Capacity:         1000,
		SOC:              50,
		MaxChargeRate:    500,
		MaxDischargeRate: 500,
		Status:           StorageStatusIdle,
		Location:         Location{Region: "cn-east"},
		TenantID:         uuid.New(),
	}
	device2 := &StorageDevice{
		Name:             "Battery 2",
		Capacity:         2000,
		SOC:              75,
		MaxChargeRate:    1000,
		MaxDischargeRate: 1000,
		Status:           StorageStatusCharging,
		Location:         Location{Region: "cn-east"},
		TenantID:         uuid.New(),
	}

	manager.CreateStorageDevice(ctx, device1)
	manager.CreateStorageDevice(ctx, device2)

	devices, err := manager.ListStorageDevices(ctx, nil)

	if err != nil {
		t.Errorf("ListStorageDevices() error = %v", err)
		return
	}

	if len(devices) != 2 {
		t.Errorf("expected 2 devices, got %d", len(devices))
	}
}

func TestDeleteStorageDevice(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	manager := NewStorageManager(repo, core)

	device := &StorageDevice{
		Name:             "Test Battery",
		Capacity:         1000,
		SOC:              50,
		MaxChargeRate:    500,
		MaxDischargeRate: 500,
		TenantID:         uuid.New(),
	}

	if err := manager.CreateStorageDevice(ctx, device); err != nil {
		t.Fatalf("failed to create storage device: %v", err)
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
			err := manager.DeleteStorageDevice(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("DeleteStorageDevice() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestStopChargeDischarge(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	manager := NewStorageManager(repo, core)

	device := &StorageDevice{
		Name:             "Test Battery",
		Capacity:         1000,
		SOC:              50,
		MaxChargeRate:    500,
		MaxDischargeRate: 500,
		Status:           StorageStatusCharging,
		TenantID:         uuid.New(),
	}

	if err := manager.CreateStorageDevice(ctx, device); err != nil {
		t.Fatalf("failed to create storage device: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "stop existing device",
			id:      device.ID,
			wantErr: false,
		},
		{
			name:    "stop non-existing device",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.StopChargeDischarge(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("StopChargeDischarge() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestUpdateSOC(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	manager := NewStorageManager(repo, core)

	device := &StorageDevice{
		Name:             "Test Battery",
		Capacity:         1000,
		SOC:              50,
		MinSOC:           10,
		MaxSOC:           95,
		MaxChargeRate:    500,
		MaxDischargeRate: 500,
		TenantID:         uuid.New(),
	}

	if err := manager.CreateStorageDevice(ctx, device); err != nil {
		t.Fatalf("failed to create storage device: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		soc     float64
		wantErr bool
	}{
		{
			name:    "update to 80%",
			id:      device.ID,
			soc:     80,
			wantErr: false,
		},
		{
			name:    "update to 0%",
			id:      device.ID,
			soc:     0,
			wantErr: false,
		},
		{
			name:    "update to 100%",
			id:      device.ID,
			soc:     100,
			wantErr: false,
		},
		{
			name:    "non-existing device",
			id:      uuid.New(),
			soc:     50,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := manager.UpdateSOC(ctx, tt.id, tt.soc)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateSOC() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetAvailableCapacity(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	manager := NewStorageManager(repo, core)

	device1 := &StorageDevice{
		Name:             "Battery 1",
		Capacity:         1000,
		SOC:              50,
		MinSOC:           10,
		MaxChargeRate:    500,
		MaxDischargeRate: 500,
		Location:         Location{Region: "cn-east"},
		TenantID:         uuid.New(),
	}
	device2 := &StorageDevice{
		Name:             "Battery 2",
		Capacity:         2000,
		SOC:              75,
		MinSOC:           10,
		MaxChargeRate:    1000,
		MaxDischargeRate: 1000,
		Location:         Location{Region: "cn-east"},
		TenantID:         uuid.New(),
	}

	manager.CreateStorageDevice(ctx, device1)
	manager.CreateStorageDevice(ctx, device2)

	tests := []struct {
		name   string
		region string
	}{
		{
			name:   "specific region",
			region: "cn-east",
		},
		{
			name:   "all regions",
			region: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			capacity, err := manager.GetAvailableCapacity(ctx, tt.region)

			if err != nil {
				t.Errorf("GetAvailableCapacity() error = %v", err)
				return
			}

			if capacity < 0 {
				t.Error("expected non-negative capacity")
			}
		})
	}
}

func TestStorageSOCBoundaryConditions(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	manager := NewStorageManager(repo, core)

	tests := []struct {
		name             string
		device           *StorageDevice
		chargePower      float64
		dischargePower   float64
		wantChargeErr    bool
		wantDischargeErr bool
	}{
		{
			name: "device at min SOC",
			device: &StorageDevice{
				Name:             "Min SOC Battery",
				Capacity:         1000,
				SOC:              10,
				MinSOC:           10,
				MaxSOC:           95,
				MaxChargeRate:    500,
				MaxDischargeRate: 500,
				Status:           StorageStatusIdle,
				TenantID:         uuid.New(),
			},
			chargePower:      100,
			dischargePower:   100,
			wantChargeErr:    false,
			wantDischargeErr: true,
		},
		{
			name: "device at max SOC",
			device: &StorageDevice{
				Name:             "Max SOC Battery",
				Capacity:         1000,
				SOC:              95,
				MinSOC:           10,
				MaxSOC:           95,
				MaxChargeRate:    500,
				MaxDischargeRate: 500,
				Status:           StorageStatusIdle,
				TenantID:         uuid.New(),
			},
			chargePower:      100,
			dischargePower:   100,
			wantChargeErr:    true,
			wantDischargeErr: false,
		},
		{
			name: "device at zero SOC",
			device: &StorageDevice{
				Name:             "Zero SOC Battery",
				Capacity:         1000,
				SOC:              0,
				MinSOC:           0,
				MaxSOC:           100,
				MaxChargeRate:    500,
				MaxDischargeRate: 500,
				Status:           StorageStatusIdle,
				TenantID:         uuid.New(),
			},
			chargePower:      100,
			dischargePower:   100,
			wantChargeErr:    false,
			wantDischargeErr: true,
		},
		{
			name: "device at full SOC",
			device: &StorageDevice{
				Name:             "Full SOC Battery",
				Capacity:         1000,
				SOC:              100,
				MinSOC:           0,
				MaxSOC:           100,
				MaxChargeRate:    500,
				MaxDischargeRate: 500,
				Status:           StorageStatusIdle,
				TenantID:         uuid.New(),
			},
			chargePower:      100,
			dischargePower:   100,
			wantChargeErr:    true,
			wantDischargeErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := manager.CreateStorageDevice(ctx, tt.device); err != nil {
				t.Fatalf("failed to create device: %v", err)
			}

			chargeErr := manager.Charge(ctx, tt.device.ID, tt.chargePower)

			if (chargeErr != nil) != tt.wantChargeErr {
				t.Errorf("Charge() error = %v, wantChargeErr %v", chargeErr, tt.wantChargeErr)
			}

			dischargeErr := manager.Discharge(ctx, tt.device.ID, tt.dischargePower)

			if (dischargeErr != nil) != tt.wantDischargeErr {
				t.Errorf("Discharge() error = %v, wantDischargeErr %v", dischargeErr, tt.wantDischargeErr)
			}
		})
	}
}

func storageStatusPtr(s StorageDeviceStatus) *StorageDeviceStatus {
	return &s
}
