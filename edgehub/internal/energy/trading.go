package energy

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"k8s.io/klog/v2"
)

type TradingEngine struct {
	repo       EnergyRepository
	energyCore *EnergyMarketCore
	
	orders       map[uuid.UUID]*EnergyOrder
	orderBooks   map[string]*OrderBook
	transactions map[uuid.UUID]*EnergyTransaction
	
	mu           sync.RWMutex
	
	matchingConfig *MatchingConfig
}

type MatchingConfig struct {
	PriceTickSize    float64 `json:"price_tick_size"`
	MinOrderQuantity float64 `json:"min_order_quantity"`
	MaxOrderQuantity float64 `json:"max_order_quantity"`
	MatchingInterval int     `json:"matching_interval_ms"`
	EnableAutoMatch  bool    `json:"enable_auto_match"`
}

func DefaultMatchingConfig() *MatchingConfig {
	return &MatchingConfig{
		PriceTickSize:    0.001,
		MinOrderQuantity: 1,
		MaxOrderQuantity: 1000000,
		MatchingInterval: 1000,
		EnableAutoMatch:  true,
	}
}

type OrderBook struct {
	Region     string          `json:"region"`
	EnergyType EnergyOrderType `json:"energy_type"`
	BuyOrders  []*OrderEntry   `json:"buy_orders"`
	SellOrders []*OrderEntry   `json:"sell_orders"`
	LastPrice  float64         `json:"last_price"`
	LastVolume float64         `json:"last_volume"`
	UpdatedAt  time.Time       `json:"updated_at"`
}

type OrderEntry struct {
	OrderID   uuid.UUID `json:"order_id"`
	Price     float64   `json:"price"`
	Quantity  float64   `json:"quantity"`
	FilledQty float64   `json:"filled_qty"`
	UserID    uuid.UUID `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
}

func NewTradingEngine(repo EnergyRepository, energyCore *EnergyMarketCore) *TradingEngine {
	return &TradingEngine{
		repo:            repo,
		energyCore:      energyCore,
		orders:          make(map[uuid.UUID]*EnergyOrder),
		orderBooks:      make(map[string]*OrderBook),
		transactions:    make(map[uuid.UUID]*EnergyTransaction),
		matchingConfig:  DefaultMatchingConfig(),
	}
}

func (te *TradingEngine) SetMatchingConfig(config *MatchingConfig) {
	te.matchingConfig = config
}

func (te *TradingEngine) CreateOrder(ctx context.Context, order *EnergyOrder) error {
	if order.ID == uuid.Nil {
		order.ID = uuid.New()
	}
	
	if order.OrderNo == "" {
		order.OrderNo = te.generateOrderNo()
	}
	
	if order.Quantity < te.matchingConfig.MinOrderQuantity {
		return fmt.Errorf("订单数量 %.2f 小于最小下单量 %.2f", order.Quantity, te.matchingConfig.MinOrderQuantity)
	}
	if order.Quantity > te.matchingConfig.MaxOrderQuantity {
		return fmt.Errorf("订单数量 %.2f 超过最大下单量 %.2f", order.Quantity, te.matchingConfig.MaxOrderQuantity)
	}
	
	order.Status = OrderStatusPending
	order.CreatedAt = time.Now()
	order.UpdatedAt = time.Now()
	
	if order.DeliveryStart.IsZero() {
		order.DeliveryStart = time.Now().Add(15 * time.Minute)
	}
	if order.DeliveryEnd.IsZero() {
		order.DeliveryEnd = order.DeliveryStart.Add(time.Hour)
	}
	
	if err := te.repo.CreateOrder(ctx, order); err != nil {
		return fmt.Errorf("创建订单失败: %w", err)
	}
	
	te.mu.Lock()
	te.orders[order.ID] = order
	te.addToOrderBook(order)
	te.mu.Unlock()
	
	klog.Infof("创建订单成功: %s, 类型: %s, 能源类型: %s, 数量: %.2f, 价格: %.3f",
		order.OrderNo, order.Type, order.EnergyType, order.Quantity, order.Price)
	
	return nil
}

func (te *TradingEngine) generateOrderNo() string {
	return fmt.Sprintf("EN%s%04d", time.Now().Format("20060102150405"), time.Now().Nanosecond()/100000)
}

func (te *TradingEngine) addToOrderBook(order *EnergyOrder) {
	key := te.getOrderBookKey(order.DeliveryRegion, order.EnergyType)
	
	book, exists := te.orderBooks[key]
	if !exists {
		book = &OrderBook{
			Region:     order.DeliveryRegion,
			EnergyType: order.EnergyType,
			BuyOrders:  make([]*OrderEntry, 0),
			SellOrders: make([]*OrderEntry, 0),
			UpdatedAt:  time.Now(),
		}
		te.orderBooks[key] = book
	}
	
	entry := &OrderEntry{
		OrderID:   order.ID,
		Price:     order.Price,
		Quantity:  order.Quantity,
		FilledQty: 0,
		UserID:    order.BuyerID,
		CreatedAt: order.CreatedAt,
	}
	
	if order.Type == OrderTypeBuy {
		book.BuyOrders = append(book.BuyOrders, entry)
		sort.Slice(book.BuyOrders, func(i, j int) bool {
			if book.BuyOrders[i].Price != book.BuyOrders[j].Price {
				return book.BuyOrders[i].Price > book.BuyOrders[j].Price
			}
			return book.BuyOrders[i].CreatedAt.Before(book.BuyOrders[j].CreatedAt)
		})
	} else {
		book.SellOrders = append(book.SellOrders, entry)
		sort.Slice(book.SellOrders, func(i, j int) bool {
			if book.SellOrders[i].Price != book.SellOrders[j].Price {
				return book.SellOrders[i].Price < book.SellOrders[j].Price
			}
			return book.SellOrders[i].CreatedAt.Before(book.SellOrders[j].CreatedAt)
		})
	}
	
	book.UpdatedAt = time.Now()
}

func (te *TradingEngine) getOrderBookKey(region string, energyType EnergyOrderType) string {
	return fmt.Sprintf("%s_%s", region, energyType)
}

func (te *TradingEngine) GetOrder(ctx context.Context, id uuid.UUID) (*EnergyOrder, error) {
	te.mu.RLock()
	if order, ok := te.orders[id]; ok {
		te.mu.RUnlock()
		return order, nil
	}
	te.mu.RUnlock()
	
	order, err := te.repo.GetOrder(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("获取订单失败: %w", err)
	}
	
	te.mu.Lock()
	te.orders[id] = order
	te.mu.Unlock()
	
	return order, nil
}

func (te *TradingEngine) GetOrderByNo(ctx context.Context, orderNo string) (*EnergyOrder, error) {
	return te.repo.GetOrderByNo(ctx, orderNo)
}

func (te *TradingEngine) ListOrders(ctx context.Context, filter *OrderFilter) ([]*EnergyOrder, error) {
	return te.repo.ListOrders(ctx, filter)
}

func (te *TradingEngine) CancelOrder(ctx context.Context, id uuid.UUID, reason string) error {
	order, err := te.GetOrder(ctx, id)
	if err != nil {
		return err
	}
	
	if order.Status == OrderStatusFilled {
		return fmt.Errorf("已成交订单无法取消")
	}
	if order.Status == OrderStatusCancelled {
		return fmt.Errorf("订单已取消")
	}
	
	now := time.Now()
	order.Status = OrderStatusCancelled
	order.CancelledAt = &now
	order.CancelReason = reason
	order.UpdatedAt = now
	
	if err := te.repo.UpdateOrder(ctx, order); err != nil {
		return fmt.Errorf("取消订单失败: %w", err)
	}
	
	te.mu.Lock()
	te.orders[id] = order
	te.removeFromOrderBook(order)
	te.mu.Unlock()
	
	klog.Infof("取消订单成功: %s, 原因: %s", order.OrderNo, reason)
	return nil
}

func (te *TradingEngine) removeFromOrderBook(order *EnergyOrder) {
	key := te.getOrderBookKey(order.DeliveryRegion, order.EnergyType)
	book, exists := te.orderBooks[key]
	if !exists {
		return
	}
	
	if order.Type == OrderTypeBuy {
		for i, entry := range book.BuyOrders {
			if entry.OrderID == order.ID {
				book.BuyOrders = append(book.BuyOrders[:i], book.BuyOrders[i+1:]...)
				break
			}
		}
	} else {
		for i, entry := range book.SellOrders {
			if entry.OrderID == order.ID {
				book.SellOrders = append(book.SellOrders[:i], book.SellOrders[i+1:]...)
				break
			}
		}
	}
	
	book.UpdatedAt = time.Now()
}

func (te *TradingEngine) SubmitOrder(ctx context.Context, id uuid.UUID) error {
	order, err := te.GetOrder(ctx, id)
	if err != nil {
		return err
	}
	
	if order.Status != OrderStatusPending {
		return fmt.Errorf("订单状态不允许提交: %s", order.Status)
	}
	
	order.Status = OrderStatusSubmitted
	order.UpdatedAt = time.Now()
	
	if err := te.repo.UpdateOrder(ctx, order); err != nil {
		return fmt.Errorf("提交订单失败: %w", err)
	}
	
	te.mu.Lock()
	te.orders[id] = order
	te.mu.Unlock()
	
	if te.matchingConfig.EnableAutoMatch {
		go func() {
			if _, err := te.MatchOrder(context.Background(), order); err != nil {
				klog.Warningf("订单撮合失败: %v", err)
			}
		}()
	}
	
	klog.Infof("提交订单成功: %s", order.OrderNo)
	return nil
}

func (te *TradingEngine) MatchOrder(ctx context.Context, order *EnergyOrder) ([]*EnergyOrder, error) {
	te.mu.Lock()
	defer te.mu.Unlock()
	
	if order.Status != OrderStatusSubmitted && order.Status != OrderStatusPartial {
		return nil, fmt.Errorf("订单状态不允许撮合: %s", order.Status)
	}
	
	key := te.getOrderBookKey(order.DeliveryRegion, order.EnergyType)
	book, exists := te.orderBooks[key]
	if !exists {
		return nil, fmt.Errorf("订单簿不存在: %s", key)
	}
	
	var matchedOrders []*EnergyOrder
	var oppositeOrders []*OrderEntry
	
	if order.Type == OrderTypeBuy {
		oppositeOrders = book.SellOrders
	} else {
		oppositeOrders = book.BuyOrders
	}
	
	remainingQty := order.Quantity
	
	for _, entry := range oppositeOrders {
		if remainingQty <= 0 {
			break
		}
		
		var canMatch bool
		if order.Type == OrderTypeBuy {
			canMatch = order.Price >= entry.Price
		} else {
			canMatch = order.Price <= entry.Price
		}
		
		if !canMatch {
			continue
		}
		
		matchedOrder, err := te.repo.GetOrder(ctx, entry.OrderID)
		if err != nil {
			continue
		}
		
		if matchedOrder.Status != OrderStatusSubmitted && matchedOrder.Status != OrderStatusPartial {
			continue
		}
		
		matchPrice := entry.Price
		availableQty := entry.Quantity - entry.FilledQty
		matchQty := math.Min(remainingQty, availableQty)
		
		if matchQty <= 0 {
			continue
		}
		
		transaction := &EnergyTransaction{
			OrderID:         order.ID,
			BuyerID:         order.BuyerID,
			SellerID:        matchedOrder.SellerID,
			Quantity:        matchQty,
			Unit:            order.Unit,
			Price:           matchPrice,
			TotalAmount:     matchQty * matchPrice,
			Currency:        order.Currency,
			TransactionTime: time.Now(),
			SettlementStatus: "pending",
			TenantID:        order.TenantID,
		}
		
		if order.Type == OrderTypeSell {
			transaction.BuyerID = matchedOrder.BuyerID
			transaction.SellerID = order.SellerID
		}
		
		if order.IsGreen || matchedOrder.IsGreen {
			transaction.CarbonSaved = matchQty * 0.5
		}
		
		if err := te.repo.CreateTransaction(ctx, transaction); err != nil {
			klog.Warningf("创建交易记录失败: %v", err)
			continue
		}
		
		te.transactions[transaction.ID] = transaction
		
		entry.FilledQty += matchQty
		remainingQty -= matchQty
		
		if entry.FilledQty >= entry.Quantity {
			matchedOrder.Status = OrderStatusFilled
			now := time.Now()
			matchedOrder.FilledAt = &now
		} else {
			matchedOrder.Status = OrderStatusPartial
		}
		matchedOrder.UpdatedAt = time.Now()
		
		if err := te.repo.UpdateOrder(ctx, matchedOrder); err != nil {
			klog.Warningf("更新匹配订单失败: %v", err)
		}
		
		te.orders[matchedOrder.ID] = matchedOrder
		matchedOrders = append(matchedOrders, matchedOrder)
		
		book.LastPrice = matchPrice
		book.LastVolume += matchQty
		
		klog.Infof("订单撮合成功: 买单 %s, 卖单 %s, 数量: %.2f, 价格: %.3f",
			order.OrderNo, matchedOrder.OrderNo, matchQty, matchPrice)
	}
	
	if remainingQty <= 0 {
		order.Status = OrderStatusFilled
		now := time.Now()
		order.FilledAt = &now
	} else if remainingQty < order.Quantity {
		order.Status = OrderStatusPartial
	}
	order.UpdatedAt = time.Now()
	
	if err := te.repo.UpdateOrder(ctx, order); err != nil {
		return nil, fmt.Errorf("更新订单失败: %w", err)
	}
	
	te.orders[order.ID] = order
	book.UpdatedAt = time.Now()
	
	return matchedOrders, nil
}

func (te *TradingEngine) GetPriceQuote(ctx context.Context, region string, energyType EnergyOrderType) (*PriceQuote, error) {
	return te.energyCore.GetLatestPriceQuote(ctx, region, energyType)
}

func (te *TradingEngine) GetPriceHistory(ctx context.Context, region string, energyType EnergyOrderType, period string) ([]*PriceQuote, error) {
	limit := 100
	switch period {
	case "day":
		limit = 96
	case "week":
		limit = 672
	case "month":
		limit = 2880
	}
	
	return te.repo.ListPriceQuotes(ctx, region, energyType, limit)
}

func (te *TradingEngine) GetOrderBook(ctx context.Context, region string, energyType EnergyOrderType) (*OrderBook, error) {
	te.mu.RLock()
	defer te.mu.RUnlock()
	
	key := te.getOrderBookKey(region, energyType)
	book, exists := te.orderBooks[key]
	if !exists {
		return &OrderBook{
			Region:     region,
			EnergyType: energyType,
			BuyOrders:  make([]*OrderEntry, 0),
			SellOrders: make([]*OrderEntry, 0),
			UpdatedAt:  time.Now(),
		}, nil
	}
	
	return book, nil
}

func (te *TradingEngine) CreateGreenCertificate(ctx context.Context, cert *GreenCertificate) error {
	if cert.ID == uuid.Nil {
		cert.ID = uuid.New()
	}
	
	if cert.CertNo == "" {
		cert.CertNo = te.generateCertNo()
	}
	
	cert.Status = "available"
	cert.IssuedAt = time.Now()
	
	if cert.ExpiresAt.IsZero() {
		cert.ExpiresAt = cert.IssuedAt.AddDate(1, 0, 0)
	}
	
	if err := te.repo.CreateGreenCertificate(ctx, cert); err != nil {
		return fmt.Errorf("创建绿证失败: %w", err)
	}
	
	klog.Infof("创建绿证成功: %s, 能源类型: %s, 数量: %.2f %s",
		cert.CertNo, cert.SourceType, cert.EnergyAmount, cert.Unit)
	
	return nil
}

func (te *TradingEngine) generateCertNo() string {
	return fmt.Sprintf("GC%s%03d", time.Now().Format("20060102150405"), time.Now().Nanosecond()/1000000)
}

func (te *TradingEngine) TransferGreenCertificate(ctx context.Context, certID, toOwnerID uuid.UUID) error {
	cert, err := te.repo.GetGreenCertificate(ctx, certID)
	if err != nil {
		return fmt.Errorf("获取绿证失败: %w", err)
	}
	
	if cert.Status != "available" {
		return fmt.Errorf("绿证状态不允许转让: %s", cert.Status)
	}
	
	now := time.Now()
	cert.OwnerID = toOwnerID
	cert.Status = "transferred"
	cert.TransferredAt = &now
	
	if err := te.repo.UpdateGreenCertificate(ctx, cert); err != nil {
		return fmt.Errorf("转让绿证失败: %w", err)
	}
	
	klog.Infof("绿证转让成功: %s, 新所有者: %s", cert.CertNo, toOwnerID)
	return nil
}

func (te *TradingEngine) ListGreenCertificates(ctx context.Context, filter *GreenCertFilter) ([]*GreenCertificate, error) {
	return te.repo.ListGreenCertificates(ctx, filter)
}

func (te *TradingEngine) CreateForwardContract(ctx context.Context, contract *ForwardContract) (*EnergyOrder, error) {
	if contract.StartDate.IsZero() || contract.EndDate.IsZero() {
		return nil, fmt.Errorf("合约日期不能为空")
	}
	
	if contract.StartDate.Before(time.Now()) {
		return nil, fmt.Errorf("合约开始日期不能早于当前时间")
	}
	
	order := &EnergyOrder{
		Type:          contract.Type,
		EnergyType:    EnergyOrderForward,
		Quantity:      contract.Quantity,
		Unit:          contract.Unit,
		Price:         contract.Price,
		TotalAmount:   contract.Quantity * contract.Price,
		Currency:      contract.Currency,
		BuyerID:       contract.BuyerID,
		SellerID:      contract.SellerID,
		TenantID:      contract.TenantID,
		DeliveryStart: contract.StartDate,
		DeliveryEnd:   contract.EndDate,
		DeliveryRegion: contract.Region,
		IsGreen:       contract.IsGreen,
	}
	
	if err := te.CreateOrder(ctx, order); err != nil {
		return nil, fmt.Errorf("创建远期合约失败: %w", err)
	}
	
	klog.Infof("创建远期合约成功: 数量 %.2f, 价格 %.3f, 交割期 %s ~ %s",
		contract.Quantity, contract.Price,
		contract.StartDate.Format("2006-01-02"), contract.EndDate.Format("2006-01-02"))
	
	return order, nil
}

type ForwardContract struct {
	Type        OrderType `json:"type"`
	Quantity    float64   `json:"quantity"`
	Unit        string    `json:"unit"`
	Price       float64   `json:"price"`
	Currency    string    `json:"currency"`
	BuyerID     uuid.UUID `json:"buyer_id"`
	SellerID    uuid.UUID `json:"seller_id"`
	TenantID    uuid.UUID `json:"tenant_id"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	Region      string    `json:"region"`
	IsGreen     bool      `json:"is_green"`
}

func (te *TradingEngine) CreateGreenCertOrder(ctx context.Context, request *GreenCertOrderRequest) (*EnergyOrder, error) {
	certs, err := te.repo.ListGreenCertificates(ctx, &GreenCertFilter{
		SourceType: request.SourceType,
		Status:     "available",
	})
	if err != nil {
		return nil, fmt.Errorf("查询绿证失败: %w", err)
	}
	
	var availableEnergy float64
	var matchedCerts []*GreenCertificate
	for _, cert := range certs {
		if availableEnergy >= request.Quantity {
			break
		}
		availableEnergy += cert.EnergyAmount
		matchedCerts = append(matchedCerts, cert)
	}
	
	if availableEnergy < request.Quantity {
		return nil, fmt.Errorf("绿证数量不足: 需要 %.2f, 可用 %.2f", request.Quantity, availableEnergy)
	}
	
	avgPrice := 0.0
	for _, cert := range matchedCerts {
		avgPrice += cert.Price * cert.EnergyAmount
	}
	avgPrice /= availableEnergy
	
	order := &EnergyOrder{
		Type:           OrderTypeBuy,
		EnergyType:     EnergyOrderGreenCert,
		Quantity:       request.Quantity,
		Unit:           "MWh",
		Price:          avgPrice,
		TotalAmount:    request.Quantity * avgPrice,
		Currency:       "CNY",
		BuyerID:        request.BuyerID,
		TenantID:       request.TenantID,
		DeliveryStart:  time.Now(),
		DeliveryEnd:    time.Now().AddDate(0, 0, 30),
		DeliveryRegion: request.Region,
		IsGreen:        true,
	}
	
	if err := te.CreateOrder(ctx, order); err != nil {
		return nil, fmt.Errorf("创建绿证订单失败: %w", err)
	}
	
	now := time.Now()
	for _, cert := range matchedCerts {
		remaining := cert.EnergyAmount
		if request.Quantity <= 0 {
			break
		}
		
		transferQty := math.Min(remaining, request.Quantity)
		request.Quantity -= transferQty
		
		cert.Status = "transferred"
		cert.OwnerID = request.BuyerID
		cert.TransferredAt = &now
		
		if err := te.repo.UpdateGreenCertificate(ctx, cert); err != nil {
			klog.Warningf("更新绿证状态失败: %v", err)
		}
	}
	
	order.Status = OrderStatusFilled
	order.FilledAt = &now
	order.UpdatedAt = now
	
	if err := te.repo.UpdateOrder(ctx, order); err != nil {
		return nil, fmt.Errorf("更新订单状态失败: %w", err)
	}
	
	klog.Infof("绿证交易成功: 数量 %.2f MWh, 均价 %.2f CNY/MWh", order.Quantity, avgPrice)
	
	return order, nil
}

type GreenCertOrderRequest struct {
	Quantity    float64         `json:"quantity"`
	SourceType  PowerSourceType `json:"source_type"`
	BuyerID     uuid.UUID       `json:"buyer_id"`
	TenantID    uuid.UUID       `json:"tenant_id"`
	Region      string          `json:"region"`
	MaxPrice    float64         `json:"max_price"`
}

func (te *TradingEngine) GetTradingStatistics(ctx context.Context, region string, period string) (*TradingStatistics, error) {
	filter := &OrderFilter{
		Status: OrderStatusFilled,
	}
	
	orders, err := te.repo.ListOrders(ctx, filter)
	if err != nil {
		return nil, fmt.Errorf("获取订单失败: %w", err)
	}
	
	stats := &TradingStatistics{
		Region:          region,
		Period:          period,
		TotalVolume:     0,
		TotalAmount:     0,
		TransactionCount: 0,
		GreenVolume:     0,
		AvgPrice:        0,
		MaxPrice:        0,
		MinPrice:        math.MaxFloat64,
	}
	
	now := time.Now()
	var startTime time.Time
	switch period {
	case "day":
		startTime = now.Add(-24 * time.Hour)
	case "week":
		startTime = now.Add(-7 * 24 * time.Hour)
	case "month":
		startTime = now.Add(-30 * 24 * time.Hour)
	default:
		startTime = time.Time{}
	}
	
	var totalPrice float64
	for _, order := range orders {
		if !startTime.IsZero() && order.CreatedAt.Before(startTime) {
			continue
		}
		if order.DeliveryRegion != region && region != "" {
			continue
		}
		
		stats.TotalVolume += order.Quantity
		stats.TotalAmount += order.TotalAmount
		stats.TransactionCount++
		totalPrice += order.Price
		
		if order.Price > stats.MaxPrice {
			stats.MaxPrice = order.Price
		}
		if order.Price < stats.MinPrice {
			stats.MinPrice = order.Price
		}
		
		if order.IsGreen {
			stats.GreenVolume += order.Quantity
		}
	}
	
	if stats.TransactionCount > 0 {
		stats.AvgPrice = totalPrice / float64(stats.TransactionCount)
	}
	if stats.MinPrice == math.MaxFloat64 {
		stats.MinPrice = 0
	}
	
	return stats, nil
}

type TradingStatistics struct {
	Region           string    `json:"region"`
	Period           string    `json:"period"`
	TotalVolume      float64   `json:"total_volume"`
	TotalAmount      float64   `json:"total_amount"`
	TransactionCount int       `json:"transaction_count"`
	GreenVolume      float64   `json:"green_volume"`
	AvgPrice         float64   `json:"avg_price"`
	MaxPrice         float64   `json:"max_price"`
	MinPrice         float64   `json:"min_price"`
}

func (te *TradingEngine) StartAutoMatching(ctx context.Context) error {
	if !te.matchingConfig.EnableAutoMatch {
		return nil
	}
	
	go func() {
		ticker := time.NewTicker(time.Duration(te.matchingConfig.MatchingInterval) * time.Millisecond)
		defer ticker.Stop()
		
		for {
			select {
			case <-ctx.Done():
				klog.Info("自动撮合引擎停止")
				return
			case <-ticker.C:
				te.runMatchingRound(ctx)
			}
		}
	}()
	
	klog.Info("自动撮合引擎启动")
	return nil
}

func (te *TradingEngine) runMatchingRound(ctx context.Context) {
	te.mu.RLock()
	defer te.mu.RUnlock()
	
	for _, book := range te.orderBooks {
		if len(book.BuyOrders) == 0 || len(book.SellOrders) == 0 {
			continue
		}
		
		bestBuy := book.BuyOrders[0]
		bestSell := book.SellOrders[0]
		
		if bestBuy.Price >= bestSell.Price {
			buyOrder, err := te.repo.GetOrder(ctx, bestBuy.OrderID)
			if err != nil {
				continue
			}
			
			if buyOrder.Status == OrderStatusSubmitted || buyOrder.Status == OrderStatusPartial {
				if _, err := te.MatchOrder(ctx, buyOrder); err != nil {
					klog.V(4).Infof("自动撮合失败: %v", err)
				}
			}
		}
	}
}

func (te *TradingEngine) GetMarketPrice(ctx context.Context, region string, energyType EnergyOrderType) (float64, error) {
	quote, err := te.GetPriceQuote(ctx, region, energyType)
	if err != nil {
		return 0, err
	}
	
	return quote.SpotPrice, nil
}

func (te *TradingEngine) CalculateOrderValue(ctx context.Context, order *EnergyOrder) (float64, error) {
	if order.Price > 0 {
		return order.Quantity * order.Price, nil
	}
	
	marketPrice, err := te.GetMarketPrice(ctx, order.DeliveryRegion, order.EnergyType)
	if err != nil {
		return 0, fmt.Errorf("获取市场价格失败: %w", err)
	}
	
	return order.Quantity * marketPrice, nil
}

func (te *TradingEngine) ValidateOrder(ctx context.Context, order *EnergyOrder) error {
	if order.Quantity <= 0 {
		return fmt.Errorf("订单数量必须大于0")
	}
	
	if order.Price <= 0 {
		return fmt.Errorf("订单价格必须大于0")
	}
	
	if order.Type != OrderTypeBuy && order.Type != OrderTypeSell {
		return fmt.Errorf("无效的订单类型: %s", order.Type)
	}
	
	if order.EnergyType != EnergyOrderSpot && order.EnergyType != EnergyOrderForward && order.EnergyType != EnergyOrderGreenCert {
		return fmt.Errorf("无效的能源类型: %s", order.EnergyType)
	}
	
	if order.DeliveryEnd.Before(order.DeliveryStart) {
		return fmt.Errorf("交割结束时间不能早于开始时间")
	}
	
	return nil
}

func (te *TradingEngine) GetTransactionHistory(ctx context.Context, filter *TransactionFilter) ([]*EnergyTransaction, error) {
	return te.repo.ListTransactions(ctx, filter)
}

func (te *TradingEngine) SettleTransaction(ctx context.Context, transactionID uuid.UUID) error {
	tx, err := te.repo.GetTransaction(ctx, transactionID)
	if err != nil {
		return fmt.Errorf("获取交易记录失败: %w", err)
	}
	
	if tx.SettlementStatus == "settled" {
		return fmt.Errorf("交易已结算")
	}
	
	now := time.Now()
	tx.SettlementStatus = "settled"
	tx.SettlementTime = &now
	
	if err := te.repo.UpdateTransaction(ctx, tx); err != nil {
		return fmt.Errorf("结算交易失败: %w", err)
	}
	
	klog.Infof("交易结算成功: %s, 金额: %.2f %s", transactionID, tx.TotalAmount, tx.Currency)
	return nil
}

func (te *TradingEngine) UpdateTransaction(ctx context.Context, tx *EnergyTransaction) error {
	return te.repo.UpdateTransaction(ctx, tx)
}
