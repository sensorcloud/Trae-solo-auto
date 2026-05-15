package energy

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestNewTradingEngine(t *testing.T) {
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
			engine := NewTradingEngine(repo, core)

			if engine == nil {
				t.Error("expected non-nil TradingEngine")
			}
		})
	}
}

func TestCreateOrder(t *testing.T) {
	tests := []struct {
		name    string
		order   *EnergyOrder
		wantErr bool
	}{
		{
			name: "valid buy order",
			order: &EnergyOrder{
				Type:          OrderTypeBuy,
				EnergyType:    EnergyOrderSpot,
				Quantity:      1000,
				Price:         0.5,
				DeliveryRegion: "cn-east",
				DeliveryStart: time.Now().Add(1 * time.Hour),
				DeliveryEnd:   time.Now().Add(2 * time.Hour),
				TenantID:      uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "valid sell order",
			order: &EnergyOrder{
				Type:          OrderTypeSell,
				EnergyType:    EnergyOrderSpot,
				Quantity:      500,
				Price:         0.6,
				DeliveryRegion: "cn-east",
				DeliveryStart: time.Now().Add(1 * time.Hour),
				DeliveryEnd:   time.Now().Add(2 * time.Hour),
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
			engine := NewTradingEngine(repo, core)

			err := engine.CreateOrder(ctx, tt.order)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if tt.order.ID == uuid.Nil {
					t.Error("expected ID to be set")
				}
				if tt.order.OrderNo == "" {
					t.Error("expected OrderNo to be set")
				}
				if tt.order.Status == "" {
					t.Error("expected default status to be set")
				}
				if tt.order.CreatedAt.IsZero() {
					t.Error("expected CreatedAt to be set")
				}
			}
		})
	}
}

func TestGetOrder(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	engine := NewTradingEngine(repo, core)

	order := &EnergyOrder{
		Type:          OrderTypeBuy,
		EnergyType:    EnergyOrderSpot,
		Quantity:      1000,
		Price:         0.5,
		DeliveryRegion: "cn-east",
		DeliveryStart: time.Now().Add(1 * time.Hour),
		DeliveryEnd:   time.Now().Add(2 * time.Hour),
		TenantID:      uuid.New(),
	}

	if err := engine.CreateOrder(ctx, order); err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "existing order",
			id:      order.ID,
			wantErr: false,
		},
		{
			name:    "non-existing order",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := engine.GetOrder(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("GetOrder() error = %v, wantErr %v", err, tt.wantErr)
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

func TestCancelOrder(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	engine := NewTradingEngine(repo, core)

	pendingOrder := &EnergyOrder{
		Type:          OrderTypeBuy,
		EnergyType:    EnergyOrderSpot,
		Quantity:      1000,
		Price:         0.5,
		DeliveryRegion: "cn-east",
		Status:        OrderStatusPending,
		DeliveryStart: time.Now().Add(1 * time.Hour),
		DeliveryEnd:   time.Now().Add(2 * time.Hour),
		TenantID:      uuid.New(),
	}

	completedOrder := &EnergyOrder{
		Type:          OrderTypeBuy,
		EnergyType:    EnergyOrderSpot,
		Quantity:      1000,
		Price:         0.5,
		DeliveryRegion: "cn-east",
		Status:        OrderStatusFilled,
		DeliveryStart: time.Now().Add(1 * time.Hour),
		DeliveryEnd:   time.Now().Add(2 * time.Hour),
		TenantID:      uuid.New(),
	}

	if err := engine.CreateOrder(ctx, pendingOrder); err != nil {
		t.Fatalf("failed to create pending order: %v", err)
	}
	if err := engine.CreateOrder(ctx, completedOrder); err != nil {
		t.Fatalf("failed to create completed order: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		reason  string
		wantErr bool
	}{
		{
			name:    "cancel pending order",
			id:      pendingOrder.ID,
			reason:  "user request",
			wantErr: false,
		},
		{
			name:    "cancel completed order",
			id:      completedOrder.ID,
			reason:  "user request",
			wantErr: true,
		},
		{
			name:    "cancel non-existing order",
			id:      uuid.New(),
			reason:  "user request",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.CancelOrder(ctx, tt.id, tt.reason)

			if (err != nil) != tt.wantErr {
				t.Errorf("CancelOrder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGetMarketPrice(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	engine := NewTradingEngine(repo, core)

	tests := []struct {
		name       string
		region     string
		energyType EnergyOrderType
	}{
		{
			name:       "spot price cn-east",
			region:     "cn-east",
			energyType: EnergyOrderSpot,
		},
		{
			name:       "forward price cn-east",
			region:     "cn-east",
			energyType: EnergyOrderForward,
		},
		{
			name:       "spot price cn-west",
			region:     "cn-west",
			energyType: EnergyOrderSpot,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			price, err := engine.GetMarketPrice(ctx, tt.region, tt.energyType)

			if err != nil {
				t.Errorf("GetMarketPrice() error = %v", err)
				return
			}

			if price < 0 {
				t.Errorf("expected non-negative price, got %f", price)
			}
		})
	}
}

func TestGetOrderBook(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	engine := NewTradingEngine(repo, core)

	buyOrder1 := &EnergyOrder{
		Type:          OrderTypeBuy,
		EnergyType:    EnergyOrderSpot,
		Quantity:      1000,
		Price:         0.55,
		DeliveryRegion: "cn-east",
		Status:        OrderStatusPending,
		DeliveryStart: time.Now().Add(1 * time.Hour),
		DeliveryEnd:   time.Now().Add(2 * time.Hour),
		TenantID:      uuid.New(),
	}
	buyOrder2 := &EnergyOrder{
		Type:          OrderTypeBuy,
		EnergyType:    EnergyOrderSpot,
		Quantity:      500,
		Price:         0.52,
		DeliveryRegion: "cn-east",
		Status:        OrderStatusPending,
		DeliveryStart: time.Now().Add(1 * time.Hour),
		DeliveryEnd:   time.Now().Add(2 * time.Hour),
		TenantID:      uuid.New(),
	}
	sellOrder := &EnergyOrder{
		Type:          OrderTypeSell,
		EnergyType:    EnergyOrderSpot,
		Quantity:      800,
		Price:         0.58,
		DeliveryRegion: "cn-east",
		Status:        OrderStatusPending,
		DeliveryStart: time.Now().Add(1 * time.Hour),
		DeliveryEnd:   time.Now().Add(2 * time.Hour),
		TenantID:      uuid.New(),
	}

	engine.CreateOrder(ctx, buyOrder1)
	engine.CreateOrder(ctx, buyOrder2)
	engine.CreateOrder(ctx, sellOrder)

	tests := []struct {
		name       string
		region     string
		energyType EnergyOrderType
	}{
		{
			name:       "cn-east spot order book",
			region:     "cn-east",
			energyType: EnergyOrderSpot,
		},
		{
			name:       "cn-west spot order book",
			region:     "cn-west",
			energyType: EnergyOrderSpot,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			book, err := engine.GetOrderBook(ctx, tt.region, tt.energyType)

			if err != nil {
				t.Errorf("GetOrderBook() error = %v", err)
				return
			}

			if book == nil {
				t.Error("expected non-nil order book")
				return
			}

			if book.Region != tt.region {
				t.Errorf("expected region %s, got %s", tt.region, book.Region)
			}
		})
	}
}

func TestListOrders(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	engine := NewTradingEngine(repo, core)

	order1 := &EnergyOrder{
		Type:          OrderTypeBuy,
		EnergyType:    EnergyOrderSpot,
		Quantity:      1000,
		Price:         0.5,
		DeliveryRegion: "cn-east",
		Status:        OrderStatusPending,
		TenantID:      uuid.New(),
	}
	order2 := &EnergyOrder{
		Type:          OrderTypeSell,
		EnergyType:    EnergyOrderSpot,
		Quantity:      500,
		Price:         0.55,
		DeliveryRegion: "cn-east",
		Status:        OrderStatusFilled,
		TenantID:      uuid.New(),
	}

	engine.CreateOrder(ctx, order1)
	engine.CreateOrder(ctx, order2)

	tests := []struct {
		name   string
		filter *OrderFilter
		count  int
	}{
		{
			name:   "list all orders",
			filter: nil,
			count:  2,
		},
		{
			name:   "filter by pending status",
			filter: &OrderFilter{Status: OrderStatusPending},
			count:  1,
		},
		{
			name:   "filter by filled status",
			filter: &OrderFilter{Status: OrderStatusFilled},
			count:  1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orders, err := engine.ListOrders(ctx, tt.filter)

			if err != nil {
				t.Errorf("ListOrders() error = %v", err)
				return
			}

			if len(orders) != tt.count {
				t.Errorf("expected %d orders, got %d", tt.count, len(orders))
			}
		})
	}
}

func TestGetTradingStatistics(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	engine := NewTradingEngine(repo, core)

	order1 := &EnergyOrder{
		Type:          OrderTypeBuy,
		EnergyType:    EnergyOrderSpot,
		Quantity:      1000,
		Price:         0.5,
		DeliveryRegion: "cn-east",
		Status:        OrderStatusFilled,
		TenantID:      uuid.New(),
	}
	order2 := &EnergyOrder{
		Type:          OrderTypeSell,
		EnergyType:    EnergyOrderSpot,
		Quantity:      500,
		Price:         0.55,
		DeliveryRegion: "cn-east",
		Status:        OrderStatusFilled,
		TenantID:      uuid.New(),
	}

	engine.CreateOrder(ctx, order1)
	engine.CreateOrder(ctx, order2)

	tests := []struct {
		name   string
		region string
		period string
	}{
		{
			name:   "cn-east daily statistics",
			region: "cn-east",
			period: "daily",
		},
		{
			name:   "all regions daily statistics",
			region: "",
			period: "daily",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stats, err := engine.GetTradingStatistics(ctx, tt.region, tt.period)

			if err != nil {
				t.Errorf("GetTradingStatistics() error = %v", err)
				return
			}

			if stats == nil {
				t.Error("expected non-nil statistics")
				return
			}

			if stats.Region != tt.region {
				t.Errorf("expected region %s, got %s", tt.region, stats.Region)
			}
		})
	}
}

func TestGetPriceQuote(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	engine := NewTradingEngine(repo, core)

	tests := []struct {
		name       string
		region     string
		energyType EnergyOrderType
	}{
		{
			name:       "spot price cn-east",
			region:     "cn-east",
			energyType: EnergyOrderSpot,
		},
		{
			name:       "forward price cn-east",
			region:     "cn-east",
			energyType: EnergyOrderForward,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			quote, err := engine.GetPriceQuote(ctx, tt.region, tt.energyType)

			if err != nil {
				t.Errorf("GetPriceQuote() error = %v", err)
				return
			}

			if quote == nil {
				t.Error("expected non-nil quote")
				return
			}

			if quote.Region != tt.region {
				t.Errorf("expected region %s, got %s", tt.region, quote.Region)
			}
		})
	}
}

func TestGetPriceHistory(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	engine := NewTradingEngine(repo, core)

	tests := []struct {
		name       string
		region     string
		energyType EnergyOrderType
		period     string
	}{
		{
			name:       "daily history cn-east",
			region:     "cn-east",
			energyType: EnergyOrderSpot,
			period:     "daily",
		},
		{
			name:       "weekly history cn-east",
			region:     "cn-east",
			energyType: EnergyOrderSpot,
			period:     "weekly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			history, err := engine.GetPriceHistory(ctx, tt.region, tt.energyType, tt.period)

			if err != nil {
				t.Errorf("GetPriceHistory() error = %v", err)
				return
			}

			if history == nil {
				t.Error("expected non-nil history")
			}
		})
	}
}

func TestMatchOrder(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	engine := NewTradingEngine(repo, core)

	buyOrder := &EnergyOrder{
		Type:          OrderTypeBuy,
		EnergyType:    EnergyOrderSpot,
		Quantity:      1000,
		Price:         0.55,
		DeliveryRegion: "cn-east",
		Status:        OrderStatusPending,
		DeliveryStart: time.Now().Add(1 * time.Hour),
		DeliveryEnd:   time.Now().Add(2 * time.Hour),
		TenantID:      uuid.New(),
	}

	sellOrder := &EnergyOrder{
		Type:          OrderTypeSell,
		EnergyType:    EnergyOrderSpot,
		Quantity:      1000,
		Price:         0.50,
		DeliveryRegion: "cn-east",
		Status:        OrderStatusPending,
		DeliveryStart: time.Now().Add(1 * time.Hour),
		DeliveryEnd:   time.Now().Add(2 * time.Hour),
		TenantID:      uuid.New(),
	}

	if err := engine.CreateOrder(ctx, buyOrder); err != nil {
		t.Fatalf("failed to create buy order: %v", err)
	}
	if err := engine.CreateOrder(ctx, sellOrder); err != nil {
		t.Fatalf("failed to create sell order: %v", err)
	}

	tests := []struct {
		name    string
		order   *EnergyOrder
		wantErr bool
	}{
		{
			name:    "match buy order",
			order:   buyOrder,
			wantErr: false,
		},
		{
			name:    "match sell order",
			order:   sellOrder,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches, err := engine.MatchOrder(ctx, tt.order)

			if (err != nil) != tt.wantErr {
				t.Errorf("MatchOrder() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr && matches == nil {
				t.Error("expected non-nil matches")
			}
		})
	}
}

func TestSubmitOrder(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	engine := NewTradingEngine(repo, core)

	order := &EnergyOrder{
		Type:          OrderTypeBuy,
		EnergyType:    EnergyOrderSpot,
		Quantity:      1000,
		Price:         0.5,
		DeliveryRegion: "cn-east",
		Status:        OrderStatusPending,
		DeliveryStart: time.Now().Add(1 * time.Hour),
		DeliveryEnd:   time.Now().Add(2 * time.Hour),
		TenantID:      uuid.New(),
	}

	if err := engine.CreateOrder(ctx, order); err != nil {
		t.Fatalf("failed to create order: %v", err)
	}

	tests := []struct {
		name    string
		id      uuid.UUID
		wantErr bool
	}{
		{
			name:    "submit valid order",
			id:      order.ID,
			wantErr: false,
		},
		{
			name:    "submit non-existing order",
			id:      uuid.New(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.SubmitOrder(ctx, tt.id)

			if (err != nil) != tt.wantErr {
				t.Errorf("SubmitOrder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCalculateOrderValue(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	engine := NewTradingEngine(repo, core)

	tests := []struct {
		name     string
		order    *EnergyOrder
		expected float64
	}{
		{
			name: "calculate value",
			order: &EnergyOrder{
				Quantity: 1000,
				Price:    0.5,
			},
			expected: 500,
		},
		{
			name: "zero quantity",
			order: &EnergyOrder{
				Quantity: 0,
				Price:    0.5,
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			value, err := engine.CalculateOrderValue(ctx, tt.order)

			if err != nil {
				t.Errorf("CalculateOrderValue() error = %v", err)
				return
			}

			if value != tt.expected {
				t.Errorf("expected %f, got %f", tt.expected, value)
			}
		})
	}
}

func TestValidateOrder(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	engine := NewTradingEngine(repo, core)

	tests := []struct {
		name    string
		order   *EnergyOrder
		wantErr bool
	}{
		{
			name: "valid order",
			order: &EnergyOrder{
				Type:          OrderTypeBuy,
				EnergyType:    EnergyOrderSpot,
				Quantity:      1000,
				Price:         0.5,
				DeliveryRegion: "cn-east",
				DeliveryStart: time.Now().Add(1 * time.Hour),
				DeliveryEnd:   time.Now().Add(2 * time.Hour),
				TenantID:      uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "zero quantity",
			order: &EnergyOrder{
				Type:          OrderTypeBuy,
				EnergyType:    EnergyOrderSpot,
				Quantity:      0,
				Price:         0.5,
				DeliveryRegion: "cn-east",
				TenantID:      uuid.New(),
			},
			wantErr: true,
		},
		{
			name: "zero price",
			order: &EnergyOrder{
				Type:          OrderTypeBuy,
				EnergyType:    EnergyOrderSpot,
				Quantity:      1000,
				Price:         0,
				DeliveryRegion: "cn-east",
				TenantID:      uuid.New(),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateOrder(ctx, tt.order)

			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateOrder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestOrderQuantityBoundaryConditions(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	engine := NewTradingEngine(repo, core)

	tests := []struct {
		name    string
		order   *EnergyOrder
		wantErr bool
	}{
		{
			name: "quantity at minimum",
			order: &EnergyOrder{
				Type:          OrderTypeBuy,
				EnergyType:    EnergyOrderSpot,
				Quantity:      10,
				Price:         0.5,
				DeliveryRegion: "cn-east",
				DeliveryStart: time.Now().Add(1 * time.Hour),
				DeliveryEnd:   time.Now().Add(2 * time.Hour),
				TenantID:      uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "quantity at maximum",
			order: &EnergyOrder{
				Type:          OrderTypeBuy,
				EnergyType:    EnergyOrderSpot,
				Quantity:      100000,
				Price:         0.5,
				DeliveryRegion: "cn-east",
				DeliveryStart: time.Now().Add(1 * time.Hour),
				DeliveryEnd:   time.Now().Add(2 * time.Hour),
				TenantID:      uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "zero quantity",
			order: &EnergyOrder{
				Type:          OrderTypeBuy,
				EnergyType:    EnergyOrderSpot,
				Quantity:      0,
				Price:         0.5,
				DeliveryRegion: "cn-east",
				DeliveryStart: time.Now().Add(1 * time.Hour),
				DeliveryEnd:   time.Now().Add(2 * time.Hour),
				TenantID:      uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "zero price",
			order: &EnergyOrder{
				Type:          OrderTypeBuy,
				EnergyType:    EnergyOrderSpot,
				Quantity:      1000,
				Price:         0,
				DeliveryRegion: "cn-east",
				DeliveryStart: time.Now().Add(1 * time.Hour),
				DeliveryEnd:   time.Now().Add(2 * time.Hour),
				TenantID:      uuid.New(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.CreateOrder(ctx, tt.order)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateOrder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGreenEnergyOrder(t *testing.T) {
	ctx := context.Background()
	repo := newMockEnergyRepository()
	core := NewEnergyMarketCore(repo, nil)
	engine := NewTradingEngine(repo, core)

	tests := []struct {
		name    string
		order   *EnergyOrder
		wantErr bool
	}{
		{
			name: "green energy order",
			order: &EnergyOrder{
				Type:          OrderTypeBuy,
				EnergyType:    EnergyOrderSpot,
				Quantity:      1000,
				Price:         0.6,
				IsGreen:       true,
				DeliveryRegion: "cn-east",
				DeliveryStart: time.Now().Add(1 * time.Hour),
				DeliveryEnd:   time.Now().Add(2 * time.Hour),
				TenantID:      uuid.New(),
			},
			wantErr: false,
		},
		{
			name: "green certificate order",
			order: &EnergyOrder{
				Type:          OrderTypeBuy,
				EnergyType:    EnergyOrderGreenCert,
				Quantity:      100,
				Price:         0.1,
				DeliveryRegion: "cn-east",
				DeliveryStart: time.Now().Add(1 * time.Hour),
				DeliveryEnd:   time.Now().Add(2 * time.Hour),
				TenantID:      uuid.New(),
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.CreateOrder(ctx, tt.order)

			if (err != nil) != tt.wantErr {
				t.Errorf("CreateOrder() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
