package market

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestMarketService_CreateOffer(t *testing.T) {
	service := NewMarketService()
	ctx := context.Background()

	offer := &MarketOffer{
		ProviderID: uuid.New(),
		ClusterID:  uuid.New(),
		ResourceSpec: ResourceSpec{
			CPU:     8,
			Memory:  16384,
			GPU:     2,
			Storage: 100,
		},
		PricePerUnit: 10.0,
		Currency:     "CNY",
		Available:    5,
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}

	err := service.CreateOffer(ctx, offer)
	if err != nil {
		t.Fatalf("Failed to create offer: %v", err)
	}

	if offer.ID == uuid.Nil {
		t.Error("Offer ID should not be nil")
	}

	if offer.Status != OfferStatusActive {
		t.Errorf("Expected status %s, got %s", OfferStatusActive, offer.Status)
	}
}

func TestMarketService_ListOffers(t *testing.T) {
	service := NewMarketService()
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		offer := &MarketOffer{
			ProviderID: uuid.New(),
			ClusterID:  uuid.New(),
			ResourceSpec: ResourceSpec{
				CPU:  float64(4 + i*2),
				GPU:  i,
			},
			PricePerUnit: 10.0 + float64(i),
			Currency:     "CNY",
			Available:    5,
			ValidUntil:   time.Now().Add(24 * time.Hour),
		}
		if err := service.CreateOffer(ctx, offer); err != nil {
			t.Fatalf("Failed to create offer: %v", err)
		}
	}

	offers, err := service.ListOffers(ctx, nil)
	if err != nil {
		t.Fatalf("Failed to list offers: %v", err)
	}

	if len(offers) != 5 {
		t.Errorf("Expected 5 offers, got %d", len(offers))
	}
}

func TestMarketService_FilterOffers(t *testing.T) {
	service := NewMarketService()
	ctx := context.Background()

	offer := &MarketOffer{
		ProviderID: uuid.New(),
		ClusterID:  uuid.New(),
		ResourceSpec: ResourceSpec{
			CPU: 8,
			GPU: 2,
		},
		PricePerUnit: 10.0,
		Currency:     "CNY",
		Available:    5,
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	service.CreateOffer(ctx, offer)

	filter := &OfferFilter{
		MinCPU: 4,
		MinGPU: 1,
		MaxPrice: 15.0,
	}

	offers, err := service.ListOffers(ctx, filter)
	if err != nil {
		t.Fatalf("Failed to filter offers: %v", err)
	}

	if len(offers) != 1 {
		t.Errorf("Expected 1 offer, got %d", len(offers))
	}
}

func TestMarketService_CreateOrder(t *testing.T) {
	service := NewMarketService()
	ctx := context.Background()

	offer := &MarketOffer{
		ProviderID: uuid.New(),
		ClusterID:  uuid.New(),
		ResourceSpec: ResourceSpec{
			CPU: 8,
			GPU: 2,
		},
		PricePerUnit: 10.0,
		Currency:     "CNY",
		Available:    5,
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	service.CreateOffer(ctx, offer)

	order := &MarketOrder{
		OfferID:    offer.ID,
		ConsumerID: uuid.New(),
		Type:       OrderTypeBuy,
		Quantity:   2,
	}

	err := service.CreateOrder(ctx, order)
	if err != nil {
		t.Fatalf("Failed to create order: %v", err)
	}

	if order.TotalAmount != 20.0 {
		t.Errorf("Expected total amount 20.0, got %.2f", order.TotalAmount)
	}

	updatedOffer, _ := service.GetOffer(ctx, offer.ID)
	if updatedOffer.Available != 3 {
		t.Errorf("Expected available 3, got %d", updatedOffer.Available)
	}
}

func TestMarketService_GetPrices(t *testing.T) {
	service := NewMarketService()
	ctx := context.Background()

	spec := &ResourceSpec{
		CPU: 8,
		GPU: 2,
	}

	info, err := service.GetPrices(ctx, spec)
	if err != nil {
		t.Fatalf("Failed to get prices: %v", err)
	}

	if info == nil {
		t.Error("PriceInfo should not be nil")
	}
}

func TestMarketService_GetMarketStats(t *testing.T) {
	service := NewMarketService()
	ctx := context.Background()

	offer := &MarketOffer{
		ProviderID: uuid.New(),
		ClusterID:  uuid.New(),
		ResourceSpec: ResourceSpec{
			CPU: 8,
			GPU: 2,
		},
		PricePerUnit: 10.0,
		Currency:     "CNY",
		Available:    5,
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	service.CreateOffer(ctx, offer)

	order := &MarketOrder{
		OfferID:    offer.ID,
		ConsumerID: uuid.New(),
		Type:       OrderTypeBuy,
		Quantity:   1,
	}
	service.CreateOrder(ctx, order)

	stats, err := service.GetMarketStats(ctx)
	if err != nil {
		t.Fatalf("Failed to get market stats: %v", err)
	}

	if stats.TotalOffers != 1 {
		t.Errorf("Expected 1 total offer, got %d", stats.TotalOffers)
	}

	if stats.TotalOrders != 1 {
		t.Errorf("Expected 1 total order, got %d", stats.TotalOrders)
	}
}
