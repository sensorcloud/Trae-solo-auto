package energy

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
)

type mockEnergyRepository struct {
	powerSources  map[uuid.UUID]*PowerSource
	vpps          map[uuid.UUID]*VirtualPowerPlant
	orders        map[uuid.UUID]*EnergyOrder
	priceQuotes   map[string]*PriceQuote
	transactions  map[uuid.UUID]*EnergyTransaction
	greenCerts    map[uuid.UUID]*GreenCertificate
	storageDevices map[uuid.UUID]*StorageDevice
	loadProfiles  map[uuid.UUID]*LoadProfile
}

func newMockEnergyRepository() *mockEnergyRepository {
	return &mockEnergyRepository{
		powerSources:   make(map[uuid.UUID]*PowerSource),
		vpps:           make(map[uuid.UUID]*VirtualPowerPlant),
		orders:         make(map[uuid.UUID]*EnergyOrder),
		priceQuotes:    make(map[string]*PriceQuote),
		transactions:   make(map[uuid.UUID]*EnergyTransaction),
		greenCerts:     make(map[uuid.UUID]*GreenCertificate),
		storageDevices: make(map[uuid.UUID]*StorageDevice),
		loadProfiles:   make(map[uuid.UUID]*LoadProfile),
	}
}

func (m *mockEnergyRepository) CreatePowerSource(ctx context.Context, source *PowerSource) error {
	m.powerSources[source.ID] = source
	return nil
}

func (m *mockEnergyRepository) GetPowerSource(ctx context.Context, id uuid.UUID) (*PowerSource, error) {
	if source, ok := m.powerSources[id]; ok {
		return source, nil
	}
	return nil, fmt.Errorf("power source not found")
}

func (m *mockEnergyRepository) ListPowerSources(ctx context.Context, filter *PowerSourceFilter) ([]*PowerSource, error) {
	result := make([]*PowerSource, 0)
	for _, source := range m.powerSources {
		if filter != nil {
			if filter.Type != "" && source.Type != filter.Type {
				continue
			}
			if filter.Status != "" && source.Status != filter.Status {
				continue
			}
		}
		result = append(result, source)
	}
	return result, nil
}

func (m *mockEnergyRepository) UpdatePowerSource(ctx context.Context, source *PowerSource) error {
	m.powerSources[source.ID] = source
	return nil
}

func (m *mockEnergyRepository) DeletePowerSource(ctx context.Context, id uuid.UUID) error {
	delete(m.powerSources, id)
	return nil
}

func (m *mockEnergyRepository) CreateStorageDevice(ctx context.Context, device *StorageDevice) error {
	m.storageDevices[device.ID] = device
	return nil
}

func (m *mockEnergyRepository) GetStorageDevice(ctx context.Context, id uuid.UUID) (*StorageDevice, error) {
	if device, ok := m.storageDevices[id]; ok {
		return device, nil
	}
	return nil, fmt.Errorf("storage device not found")
}

func (m *mockEnergyRepository) ListStorageDevices(ctx context.Context, filter *StorageFilter) ([]*StorageDevice, error) {
	result := make([]*StorageDevice, 0)
	for _, device := range m.storageDevices {
		result = append(result, device)
	}
	return result, nil
}

func (m *mockEnergyRepository) UpdateStorageDevice(ctx context.Context, device *StorageDevice) error {
	m.storageDevices[device.ID] = device
	return nil
}

func (m *mockEnergyRepository) DeleteStorageDevice(ctx context.Context, id uuid.UUID) error {
	delete(m.storageDevices, id)
	return nil
}

func (m *mockEnergyRepository) CreateOrder(ctx context.Context, order *EnergyOrder) error {
	m.orders[order.ID] = order
	return nil
}

func (m *mockEnergyRepository) GetOrder(ctx context.Context, id uuid.UUID) (*EnergyOrder, error) {
	if order, ok := m.orders[id]; ok {
		return order, nil
	}
	return nil, fmt.Errorf("order not found")
}

func (m *mockEnergyRepository) GetOrderByNo(ctx context.Context, orderNo string) (*EnergyOrder, error) {
	for _, order := range m.orders {
		if order.OrderNo == orderNo {
			return order, nil
		}
	}
	return nil, fmt.Errorf("order not found")
}

func (m *mockEnergyRepository) ListOrders(ctx context.Context, filter *OrderFilter) ([]*EnergyOrder, error) {
	result := make([]*EnergyOrder, 0)
	for _, order := range m.orders {
		if filter != nil {
			if filter.Status != "" && order.Status != filter.Status {
				continue
			}
			if filter.Type != "" && order.Type != filter.Type {
				continue
			}
		}
		result = append(result, order)
	}
	return result, nil
}

func (m *mockEnergyRepository) UpdateOrder(ctx context.Context, order *EnergyOrder) error {
	m.orders[order.ID] = order
	return nil
}

func (m *mockEnergyRepository) CreatePriceQuote(ctx context.Context, quote *PriceQuote) error {
	key := quote.Region + "_" + string(quote.EnergyType)
	m.priceQuotes[key] = quote
	return nil
}

func (m *mockEnergyRepository) GetLatestPriceQuote(ctx context.Context, region string, energyType EnergyOrderType) (*PriceQuote, error) {
	key := region + "_" + string(energyType)
	if quote, ok := m.priceQuotes[key]; ok {
		return quote, nil
	}
	return nil, fmt.Errorf("price quote not found")
}

func (m *mockEnergyRepository) ListPriceQuotes(ctx context.Context, region string, energyType EnergyOrderType, limit int) ([]*PriceQuote, error) {
	result := make([]*PriceQuote, 0)
	for _, quote := range m.priceQuotes {
		result = append(result, quote)
		if len(result) >= limit {
			break
		}
	}
	return result, nil
}

func (m *mockEnergyRepository) CreateVPP(ctx context.Context, vpp *VirtualPowerPlant) error {
	m.vpps[vpp.ID] = vpp
	return nil
}

func (m *mockEnergyRepository) GetVPP(ctx context.Context, id uuid.UUID) (*VirtualPowerPlant, error) {
	if vpp, ok := m.vpps[id]; ok {
		return vpp, nil
	}
	return nil, fmt.Errorf("vpp not found")
}

func (m *mockEnergyRepository) ListVPPs(ctx context.Context, filter *VPPFilter) ([]*VirtualPowerPlant, error) {
	result := make([]*VirtualPowerPlant, 0)
	for _, vpp := range m.vpps {
		result = append(result, vpp)
	}
	return result, nil
}

func (m *mockEnergyRepository) UpdateVPP(ctx context.Context, vpp *VirtualPowerPlant) error {
	m.vpps[vpp.ID] = vpp
	return nil
}

func (m *mockEnergyRepository) DeleteVPP(ctx context.Context, id uuid.UUID) error {
	delete(m.vpps, id)
	return nil
}

func (m *mockEnergyRepository) CreateTransaction(ctx context.Context, tx *EnergyTransaction) error {
	m.transactions[tx.ID] = tx
	return nil
}

func (m *mockEnergyRepository) GetTransaction(ctx context.Context, id uuid.UUID) (*EnergyTransaction, error) {
	if tx, ok := m.transactions[id]; ok {
		return tx, nil
	}
	return nil, fmt.Errorf("transaction not found")
}

func (m *mockEnergyRepository) ListTransactions(ctx context.Context, filter *TransactionFilter) ([]*EnergyTransaction, error) {
	result := make([]*EnergyTransaction, 0)
	for _, tx := range m.transactions {
		result = append(result, tx)
	}
	return result, nil
}

func (m *mockEnergyRepository) UpdateTransaction(ctx context.Context, tx *EnergyTransaction) error {
	m.transactions[tx.ID] = tx
	return nil
}

func (m *mockEnergyRepository) CreateGreenCertificate(ctx context.Context, cert *GreenCertificate) error {
	m.greenCerts[cert.ID] = cert
	return nil
}

func (m *mockEnergyRepository) GetGreenCertificate(ctx context.Context, id uuid.UUID) (*GreenCertificate, error) {
	if cert, ok := m.greenCerts[id]; ok {
		return cert, nil
	}
	return nil, fmt.Errorf("green certificate not found")
}

func (m *mockEnergyRepository) ListGreenCertificates(ctx context.Context, filter *GreenCertFilter) ([]*GreenCertificate, error) {
	result := make([]*GreenCertificate, 0)
	for _, cert := range m.greenCerts {
		if filter != nil {
			if filter.Status != "" && cert.Status != filter.Status {
				continue
			}
		}
		result = append(result, cert)
	}
	return result, nil
}

func (m *mockEnergyRepository) UpdateGreenCertificate(ctx context.Context, cert *GreenCertificate) error {
	m.greenCerts[cert.ID] = cert
	return nil
}

func (m *mockEnergyRepository) CreateLoadProfile(ctx context.Context, profile *LoadProfile) error {
	m.loadProfiles[profile.ID] = profile
	return nil
}

func (m *mockEnergyRepository) GetLoadProfile(ctx context.Context, id uuid.UUID) (*LoadProfile, error) {
	if profile, ok := m.loadProfiles[id]; ok {
		return profile, nil
	}
	return nil, fmt.Errorf("load profile not found")
}

func (m *mockEnergyRepository) ListLoadProfiles(ctx context.Context, filter *LoadFilter) ([]*LoadProfile, error) {
	result := make([]*LoadProfile, 0)
	for _, profile := range m.loadProfiles {
		result = append(result, profile)
	}
	return result, nil
}

func (m *mockEnergyRepository) UpdateLoadProfile(ctx context.Context, profile *LoadProfile) error {
	m.loadProfiles[profile.ID] = profile
	return nil
}

func (m *mockEnergyRepository) DeleteLoadProfile(ctx context.Context, id uuid.UUID) error {
	delete(m.loadProfiles, id)
	return nil
}

func TestNewEnergyMarketCore(t *testing.T) {
	tests := []struct {
		name   string
		config *EnergyMarketConfig
	}{
		{
			name:   "nil config uses defaults",
			config: nil,
		},
		{
			name: "custom config",
			config: &EnergyMarketConfig{
				Region:              "cn-east",
				SpotMarketEnabled:   false,
				MinOrderQuantity:    10,
				MaxOrderQuantity:    500000,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newMockEnergyRepository()
			core := NewEnergyMarketCore(repo, tt.config)

			if core == nil {
				t.Fatal("expected non-nil EnergyMarketCore")
			}
		})
	}
}

func TestCreatePowerSource(t *testing.T) {
	tests := []struct {
		name    string
		source  *PowerSource
		wantErr bool
	}{
		{
			name: "valid solar power source",
			source: &PowerSource{
				Name:     "Solar Farm 1",
				Type:     PowerSourceSolar,
				Capacity: 1000,
				Unit:     "kW",
				Location: Location{
					Region:    "cn-east",
					Zone:      "zone-a",
					Latitude:  31.2,
					Longitude: 121.5,
				},
				TenantID: uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "valid wind power source",
			source: &PowerSource{
				Name:     "Wind Farm 1",
				Type:     PowerSourceWind,
				Capacity: 2000,
				Unit:     "kW",
				Location: Location{
					Region: "cn-north",
				},
				TenantID: uuid.New(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := newMockEnergyRepository()
			core := NewEnergyMarketCore(repo, nil)

			err := core.CreatePowerSource(ctx, tt.source)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreatePowerSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.source.ID == uuid.Nil {
					t.Error("expected ID to be set")
				}
				if tt.source.CreatedAt.IsZero() {
					t.Error("expected CreatedAt to be set")
				}
				if tt.source.Status == "" {
					t.Error("expected default status to be set")
				}
			}
		})
	}
}

func TestGetPowerSource(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)

	source := &PowerSource{
		Name:     "Test Source",
		Type:     PowerSourceSolar,
		Capacity: 1000,
		TenantID: uuid.New(),
	}

	if err := core.CreatePowerSource(ctx, source); err != nil {
		t.Fatalf("failed to create power source: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing power source",
			id:      source.ID,
			wantErr: false,
		},
		{
			name:    "non-existing power source",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := core.GetPowerSource(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetPowerSource() error = %v, wantErr %v", err, tt.wantErr)
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

func TestUpdatePowerSource(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)

	source := &PowerSource{
		Name:     "Test Source",
		Type:     PowerSourceSolar,
		Capacity: 1000,
		TenantID: uuid.New(),
	}

	if err := core.CreatePowerSource(ctx, source); err != nil {
		t.Fatalf("failed to create power source: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		updates *PowerSourceUpdate
		wantErr bool
	}{
		{
			name: "update name and capacity",
			id:   source.ID,
			updates: &PowerSourceUpdate{
				Name:     strPtr("Updated Source"),
				Capacity: float64Ptr(1500),
			},
			wantErr: false,
		},
		{
			name: "update status",
			id:   source.ID,
			updates: &PowerSourceUpdate{
				Status: powerSourceStatusPtr(PowerSourceStatusMaintenance),
			},
			wantErr: false,
		},
		{
			name:    "non-existing power source",
			id:      uuid.New(),
			updates: &PowerSourceUpdate{Name: strPtr("New Name")},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := core.UpdatePowerSource(ctx, tt.id, tt.updates)

			if (err != nil) != tt.wantErr {
				t.Errorf("UpdatePowerSource() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				updated, err := core.GetPowerSource(ctx, tt.id)
				if err != nil {
					t.Fatalf("failed to get updated source: %v", err)
				}

				if tt.updates.Name != nil && updated.Name != *tt.updates.Name {
					t.Errorf("expected name %s, got %s", *tt.updates.Name, updated.Name)
				}
				if tt.updates.Capacity != nil && updated.Capacity != *tt.updates.Capacity {
					t.Errorf("expected capacity %f, got %f", *tt.updates.Capacity, updated.Capacity)
				}
			}
		})
	}
}

func TestDeletePowerSource(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)

	source := &PowerSource{
		Name:     "Test Source",
		Type:     PowerSourceSolar,
		Capacity: 1000,
		TenantID: uuid.New(),
	}

	if err := core.CreatePowerSource(ctx, source); err != nil {
		t.Fatalf("failed to create power source: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "delete existing power source",
			id:      source.ID,
			wantErr: false,
		},
		{
			name:    "delete non-existing power source",
			id:      uuid.New(),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := core.DeletePowerSource(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("DeletePowerSource() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCreateVPP(t *testing.T) {
	tests := []struct {
		name    string
		vpp     *VirtualPowerPlant
		wantErr bool
	}{
		{
			name: "valid VPP",
			vpp: &VirtualPowerPlant{
				Name:          "VPP 1",
				Type:          VPPTypesDistributed,
				TotalCapacity: 5000,
				Region:        "cn-east",
				TenantID:      uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "VPP with existing ID",
			vpp: &VirtualPowerPlant{
				EnergyBaseModel: EnergyBaseModel{
					ID: uuid.New(),
				},
				Name:          "VPP 2",
				Type:          VPPTypesCentralized,
				TotalCapacity: 3000,
				TenantID:      uuid.New(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			repo := newMockEnergyRepository()
			core := NewEnergyMarketCore(repo, nil)

			err := core.CreateVPP(ctx, tt.vpp)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateVPP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.vpp.ID == uuid.Nil {
					t.Error("expected ID to be set")
				}
				if tt.vpp.Status == "" {
					t.Error("expected default status to be set")
				}
			}
		})
	}
}

func TestDispatchVPP(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)

	vpp := &VirtualPowerPlant{
		Name:              "Test VPP",
		Type:              VPPTypesDistributed,
		TotalCapacity:     5000,
		AvailableCapacity: 4000,
		Status:            VPPStatusActive,
		Region:            "cn-east",
		TenantID:          uuid.New(),
	}

	if err := core.CreateVPP(ctx, vpp); err != nil {
		t.Fatalf("failed to create VPP: %v", err)
	}

	tests := []struct {
		name    string
		vppID   uuid.UUID
		request *DispatchRequest
		wantErr bool
	}{
		{
			name:  "valid dispatch request",
			vppID: vpp.ID,
			request: &DispatchRequest{
				Power:    1000,
				Duration: 60,
			},
			wantErr: false,
		},
		{
			name:  "dispatch power exceeds capacity",
			vppID: vpp.ID,
			request: &DispatchRequest{
				Power:    10000,
				Duration: 60,
			},
			wantErr: true,
		},
		{
			name:  "non-existing VPP",
			vppID: uuid.New(),
			request: &DispatchRequest{
				Power:    100,
				Duration: 30,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := core.DispatchVPP(ctx, tt.vppID, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("DispatchVPP() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if result == nil {
					t.Error("expected non-nil result")
					return
				}
				if result.VPPID != tt.vppID {
					t.Errorf("expected VPPID %s, got %s", tt.vppID, result.VPPID)
				}
				if result.Status != "dispatched" {
					t.Errorf("expected status 'dispatched', got %s", result.Status)
				}
			}
		})
	}
}

func TestGetMarketOverview(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)

	source1 := &PowerSource{
		Name:           "Solar 1",
		Type:           PowerSourceSolar,
		Capacity:       1000,
		RealtimeOutput: 800,
		Location:       Location{Region: "cn-east"},
		TenantID:       uuid.New(),
	}
	source2 := &PowerSource{
		Name:           "Wind 1",
		Type:           PowerSourceWind,
		Capacity:       2000,
		RealtimeOutput: 1500,
		Location:       Location{Region: "cn-east"},
		TenantID:       uuid.New(),
	}

	core.CreatePowerSource(ctx, source1)
	core.CreatePowerSource(ctx, source2)

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
			overview, err := core.GetMarketOverview(ctx, tt.region)

			if err != nil {
				t.Errorf("GetMarketOverview() error = %v", err)
				return
			}

			if overview == nil {
				t.Error("expected non-nil overview")
				return
			}

			if overview.Region != tt.region {
				t.Errorf("expected region %s, got %s", tt.region, overview.Region)
			}
		})
	}
}

func TestGetOptimalTimeSlot(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)

	tests := []struct {
		name    string
		request *OptimalTimeRequest
		wantErr bool
	}{
		{
			name: "cost optimization",
			request: &OptimalTimeRequest{
				EstimatedPower:   100,
				Duration:         60,
				Region:           "cn-east",
				OptimizationGoal: "cost",
			},
			wantErr: false,
		},
		{
			name: "green optimization",
			request: &OptimalTimeRequest{
				EstimatedPower:   200,
				Duration:         120,
				Region:           "cn-east",
				OptimizationGoal: "green",
			},
			wantErr: false,
		},
		{
			name: "balanced optimization",
			request: &OptimalTimeRequest{
				EstimatedPower:   150,
				Duration:         90,
				Region:           "cn-east",
				OptimizationGoal: "balanced",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slot, err := core.GetOptimalTimeSlot(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetOptimalTimeSlot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if slot == nil {
					t.Error("expected non-nil slot")
					return
				}
				if slot.StartTime == "" {
					t.Error("expected StartTime to be set")
				}
				if slot.EndTime == "" {
					t.Error("expected EndTime to be set")
				}
			}
		})
	}
}

func TestGetEnergyForecast(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)

	tests := []struct {
		name    string
		region  string
		horizon int
	}{
		{
			name:    "short horizon",
			region:  "cn-east",
			horizon: 24,
		},
		{
			name:    "medium horizon",
			region:  "cn-east",
			horizon: 96,
		},
		{
			name:    "long horizon",
			region:  "cn-east",
			horizon: 192,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			forecast, err := core.GetEnergyForecast(ctx, tt.region, tt.horizon)

			if err != nil {
				t.Errorf("GetEnergyForecast() error = %v", err)
				return
			}

			if len(forecast) != tt.horizon {
				t.Errorf("expected %d forecast points, got %d", tt.horizon, len(forecast))
			}

			for i, point := range forecast {
				if point.Timestamp.IsZero() {
					t.Errorf("forecast point %d has zero timestamp", i)
				}
				if point.Confidence < 0 || point.Confidence > 1 {
					t.Errorf("forecast point %d has invalid confidence: %f", i, point.Confidence)
				}
			}
		})
	}
}

func TestIsPeakHour(t *testing.T) {
	repo := newMockEnergyRepository()
	config := &EnergyMarketConfig{
		PeakHours: []TimeRange{
			{Start: "08:00", End: "12:00"},
			{Start: "18:00", End: "22:00"},
		},
		ValleyHours: []TimeRange{
			{Start: "00:00", End: "06:00"},
		},
	}
	core := NewEnergyMarketCore(repo, config)

	tests := []struct {
		name     string
		time     time.Time
		isPeak   bool
		isValley bool
	}{
		{
			name:     "morning peak",
			time:     time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
			isPeak:   true,
			isValley: false,
		},
		{
			name:     "evening peak",
			time:     time.Date(2024, 1, 1, 19, 0, 0, 0, time.UTC),
			isPeak:   true,
			isValley: false,
		},
		{
			name:     "valley hour",
			time:     time.Date(2024, 1, 1, 3, 0, 0, 0, time.UTC),
			isPeak:   false,
			isValley: true,
		},
		{
			name:     "regular hour",
			time:     time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
			isPeak:   false,
			isValley: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isPeak := core.IsPeakHour(tt.time)
			isValley := core.IsValleyHour(tt.time)

			if isPeak != tt.isPeak {
				t.Errorf("IsPeakHour() = %v, want %v", isPeak, tt.isPeak)
			}
			if isValley != tt.isValley {
				t.Errorf("IsValleyHour() = %v, want %v", isValley, tt.isValley)
			}
		})
	}
}

func TestScheduleComputeWithEnergy(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)

	tests := []struct {
		name    string
		request *ComputeEnergyRequest
		wantErr bool
	}{
		{
			name: "valid compute request",
			request: &ComputeEnergyRequest{
				ComputeJobID:   uuid.New(),
				EstimatedPower: 100,
				Duration:       60,
				Region:         "cn-east",
				TenantID:       uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "high power compute request",
			request: &ComputeEnergyRequest{
				ComputeJobID:   uuid.New(),
				EstimatedPower: 1000,
				Duration:       120,
				Region:         "cn-east",
				TenantID:       uuid.New(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			coordination, err := core.ScheduleComputeWithEnergy(ctx, tt.request)

			if (err != nil) != tt.wantErr {
				t.Errorf("ScheduleComputeWithEnergy() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if coordination == nil {
					t.Error("expected non-nil coordination")
					return
				}
				if coordination.ComputeJobID != tt.request.ComputeJobID {
					t.Errorf("expected ComputeJobID %s, got %s", tt.request.ComputeJobID, coordination.ComputeJobID)
				}
				if coordination.Status != "scheduled" {
					t.Errorf("expected status 'scheduled', got %s", coordination.Status)
				}
			}
		})
	}
}

func TestListPowerSources(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)

	source1 := &PowerSource{
		Name:     "Solar 1",
		Type:     PowerSourceSolar,
		Capacity: 1000,
		Status:   PowerSourceStatusOnline,
		TenantID: uuid.New(),
	}
	source2 := &PowerSource{
		Name:     "Wind 1",
		Type:     PowerSourceWind,
		Capacity: 2000,
		Status:   PowerSourceStatusOffline,
		TenantID: uuid.New(),
	}

	core.CreatePowerSource(ctx, source1)
	core.CreatePowerSource(ctx, source2)

	tests := []struct {
		name   string
		filter *PowerSourceFilter
		count  int
	}{
		{
			name:   "list all",
			filter: nil,
			count:  2,
		},
		{
			name:   "filter by solar type",
			filter: &PowerSourceFilter{Type: PowerSourceSolar},
			count:  1,
		},
		{
			name:   "filter by online status",
			filter: &PowerSourceFilter{Status: PowerSourceStatusOnline},
			count:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sources, err := core.ListPowerSources(ctx, tt.filter)

			if err != nil {
				t.Errorf("ListPowerSources() error = %v", err)
				return
			}

			if len(sources) != tt.count {
				t.Errorf("expected %d sources, got %d", tt.count, len(sources))
			}
		})
	}
}

func strPtr(s string) *string {
	return &s
}

func float64Ptr(f float64) *float64 {
	return &f
}

func powerSourceStatusPtr(s PowerSourceStatus) *PowerSourceStatus {
	return &s
}
