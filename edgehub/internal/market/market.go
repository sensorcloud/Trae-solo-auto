package market

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"k8s.io/klog/v2"
)

type OrderType string
type OrderStatus string
type OfferStatus string

const (
	OrderTypeBuy  OrderType = "buy"
	OrderTypeSell OrderType = "sell"

	OrderStatusPending   OrderStatus = "pending"
	OrderStatusMatching  OrderStatus = "matching"
	OrderStatusFulfilled OrderStatus = "fulfilled"
	OrderStatusCancelled OrderStatus = "cancelled"
	OrderStatusExpired   OrderStatus = "expired"

	OfferStatusActive   OfferStatus = "active"
	OfferStatusReserved OfferStatus = "reserved"
	OfferStatusSold    OfferStatus = "sold"
	OfferStatusExpired OfferStatus = "expired"
)

type ResourceSpec struct {
	CPU      float64 `json:"cpu"`
	Memory   int64   `json:"memory"`
	GPU      int     `json:"gpu"`
	Storage  int64   `json:"storage"`
	Duration int     `json:"duration"`
}

type MarketOffer struct {
	ID            uuid.UUID      `json:"id"`
	ProviderID    uuid.UUID      `json:"provider_id"`
	ClusterID     uuid.UUID      `json:"cluster_id"`
	ResourceSpec  ResourceSpec   `json:"resource_spec"`
	PricePerUnit  float64        `json:"price_per_unit"`
	Currency      string         `json:"currency"`
	MinDuration   int            `json:"min_duration"`
	MaxDuration   int            `json:"max_duration"`
	Status        OfferStatus    `json:"status"`
	Available     int            `json:"available"`
	ValidFrom     time.Time      `json:"valid_from"`
	ValidUntil    time.Time      `json:"valid_until"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
}

type MarketOrder struct {
	ID           uuid.UUID   `json:"id"`
	OfferID      uuid.UUID   `json:"offer_id"`
	ConsumerID   uuid.UUID   `json:"consumer_id"`
	Type         OrderType   `json:"type"`
	Status       OrderStatus `json:"status"`
	Quantity     int         `json:"quantity"`
	Price        float64     `json:"price"`
	TotalAmount  float64     `json:"total_amount"`
	Currency     string      `json:"currency"`
	FulfilledAt  *time.Time  `json:"fulfilled_at,omitempty"`
	CreatedAt    time.Time   `json:"created_at"`
	UpdatedAt    time.Time   `json:"updated_at"`
}

type PriceRecord struct {
	Timestamp   time.Time    `json:"timestamp"`
	ResourceSpec ResourceSpec `json:"resource_spec"`
	AvgPrice    float64      `json:"avg_price"`
	MinPrice    float64      `json:"min_price"`
	MaxPrice    float64      `json:"max_price"`
	Volume      int          `json:"volume"`
}

type MarketService struct {
	offers    map[uuid.UUID]*MarketOffer
	orders    map[uuid.UUID]*MarketOrder
	prices    []*PriceRecord
	mu        sync.RWMutex
	priceAgg  *PriceAggregator
}

func NewMarketService() *MarketService {
	return &MarketService{
		offers:   make(map[uuid.UUID]*MarketOffer),
		orders:   make(map[uuid.UUID]*MarketOrder),
		prices:  make([]*PriceRecord, 0),
		priceAgg: NewPriceAggregator(),
	}
}

func (ms *MarketService) CreateOffer(ctx context.Context, offer *MarketOffer) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	offer.ID = uuid.New()
	offer.Status = OfferStatusActive
	offer.CreatedAt = time.Now()
	offer.UpdatedAt = time.Now()

	if offer.ValidUntil.IsZero() {
		offer.ValidUntil = offer.CreatedAt.Add(7 * 24 * time.Hour)
	}

	ms.offers[offer.ID] = offer
	klog.Infof("Created market offer %s by provider %s", offer.ID, offer.ProviderID)
	return nil
}

func (ms *MarketService) GetOffer(ctx context.Context, offerID uuid.UUID) (*MarketOffer, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	offer, exists := ms.offers[offerID]
	if !exists {
		return nil, fmt.Errorf("offer %s not found", offerID)
	}
	return offer, nil
}

func (ms *MarketService) ListOffers(ctx context.Context, filter *OfferFilter) ([]*MarketOffer, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var result []*MarketOffer
	now := time.Now()

	for _, offer := range ms.offers {
		if filter != nil {
			if offer.Status != OfferStatusActive {
				continue
			}
			if offer.Available <= 0 {
				continue
			}
			if now.After(offer.ValidUntil) {
				continue
			}
			if filter.MinCPU > 0 && offer.ResourceSpec.CPU < filter.MinCPU {
				continue
			}
			if filter.MinGPU > 0 && offer.ResourceSpec.GPU < filter.MinGPU {
				continue
			}
			if filter.MaxPrice > 0 && offer.PricePerUnit > filter.MaxPrice {
				continue
			}
		}
		result = append(result, offer)
	}

	return result, nil
}

func (ms *MarketService) UpdateOffer(ctx context.Context, offerID uuid.UUID, updates *OfferUpdate) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	offer, exists := ms.offers[offerID]
	if !exists {
		return fmt.Errorf("offer %s not found", offerID)
	}

	if updates.PricePerUnit != nil {
		offer.PricePerUnit = *updates.PricePerUnit
	}
	if updates.Available != nil {
		offer.Available = *updates.Available
	}
	if updates.ValidUntil != nil {
		offer.ValidUntil = *updates.ValidUntil
	}
	offer.UpdatedAt = time.Now()

	return nil
}

func (ms *MarketService) DeleteOffer(ctx context.Context, offerID uuid.UUID) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	if _, exists := ms.offers[offerID]; !exists {
		return fmt.Errorf("offer %s not found", offerID)
	}

	delete(ms.offers, offerID)
	klog.Infof("Deleted offer %s", offerID)
	return nil
}

func (ms *MarketService) CreateOrder(ctx context.Context, order *MarketOrder) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	offer, exists := ms.offers[order.OfferID]
	if !exists {
		return fmt.Errorf("offer %s not found", order.OfferID)
	}

	if offer.Status != OfferStatusActive {
		return fmt.Errorf("offer %s is not active", order.OfferID)
	}
	if offer.Available < order.Quantity {
		return fmt.Errorf("insufficient available resources: requested %d, available %d", order.Quantity, offer.Available)
	}

	order.ID = uuid.New()
	order.Status = OrderStatusMatching
	order.Price = offer.PricePerUnit
	order.TotalAmount = offer.PricePerUnit * float64(order.Quantity)
	order.Currency = offer.Currency
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()

	ms.orders[order.ID] = order

	if err := ms.matchOrder(ctx, order); err != nil {
		order.Status = OrderStatusPending
		return err
	}

	klog.Infof("Created and matched order %s for offer %s", order.ID, order.OfferID)
	return nil
}

func (ms *MarketService) GetOrder(ctx context.Context, orderID uuid.UUID) (*MarketOrder, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	order, exists := ms.orders[orderID]
	if !exists {
		return nil, fmt.Errorf("order %s not found", orderID)
	}
	return order, nil
}

func (ms *MarketService) ListOrders(ctx context.Context, consumerID uuid.UUID) ([]*MarketOrder, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var result []*MarketOrder
	for _, order := range ms.orders {
		if consumerID != uuid.Nil && order.ConsumerID != consumerID {
			continue
		}
		result = append(result, order)
	}
	return result, nil
}

func (ms *MarketService) CancelOrder(ctx context.Context, orderID uuid.UUID) error {
	ms.mu.Lock()
	defer ms.mu.Unlock()

	order, exists := ms.orders[orderID]
	if !exists {
		return fmt.Errorf("order %s not found", orderID)
	}

	if order.Status == OrderStatusFulfilled {
		return fmt.Errorf("cannot cancel fulfilled order")
	}

	order.Status = OrderStatusCancelled
	order.UpdatedAt = time.Now()

	klog.Infof("Cancelled order %s", orderID)
	return nil
}

func (ms *MarketService) matchOrder(ctx context.Context, order *MarketOrder) error {
	offer := ms.offers[order.OfferID]
	if offer == nil {
		return fmt.Errorf("offer not found")
	}

	offer.Available -= order.Quantity
	if offer.Available <= 0 {
		offer.Status = OfferStatusSold
	}

	now := time.Now()
	order.Status = OrderStatusFulfilled
	order.FulfilledAt = &now

	priceRecord := &PriceRecord{
		Timestamp:   now,
		ResourceSpec: offer.ResourceSpec,
		AvgPrice:    offer.PricePerUnit,
		MinPrice:    offer.PricePerUnit,
		MaxPrice:    offer.PricePerUnit,
		Volume:      order.Quantity,
	}
	ms.prices = append(ms.prices, priceRecord)
	ms.priceAgg.AddRecord(priceRecord)

	return nil
}

func (ms *MarketService) GetPrices(ctx context.Context, spec *ResourceSpec) (*PriceInfo, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	info := ms.priceAgg.GetPriceInfo(spec)
	return info, nil
}

func (ms *MarketService) GetPriceRecommendation(ctx context.Context, spec *ResourceSpec) (*PriceRecommendation, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	recommendation := ms.priceAgg.GetRecommendation(spec)

	return &PriceRecommendation{
		RecommendedPrice: recommendation.RecommendedPrice,
		Confidence:       recommendation.Confidence,
		Trend:            recommendation.Trend,
		Factors:          recommendation.Factors,
	}, nil
}

func (ms *MarketService) GetMarketStats(ctx context.Context) (*MarketStats, error) {
	ms.mu.RLock()
	defer ms.mu.RUnlock()

	var totalOffers, activeOffers int
	var totalOrders, fulfilledOrders int
	var totalVolume float64

	for _, offer := range ms.offers {
		totalOffers++
		if offer.Status == OfferStatusActive {
			activeOffers++
		}
	}

	for _, order := range ms.orders {
		totalOrders++
		if order.Status == OrderStatusFulfilled {
			fulfilledOrders++
			totalVolume += order.TotalAmount
		}
	}

	return &MarketStats{
		TotalOffers:     totalOffers,
		ActiveOffers:    activeOffers,
		TotalOrders:     totalOrders,
		FulfilledOrders: fulfilledOrders,
		TotalVolume:     totalVolume,
	}, nil
}

type OfferFilter struct {
	MinCPU    float64
	MinGPU    int
	MinMemory int64
	MaxPrice  float64
	Region    string
}

type OfferUpdate struct {
	PricePerUnit *float64
	Available    *int
	ValidUntil   *time.Time
}
