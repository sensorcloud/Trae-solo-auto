package billing

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"k8s.io/klog/v2"
)

type BillStatus string
type PaymentMethod string

const (
	BillStatusPending   BillStatus = "pending"
	BillStatusPaid     BillStatus = "paid"
	BillStatusOverdue  BillStatus = "overdue"
	BillStatusCancelled BillStatus = "cancelled"

	PaymentMethodCreditCard PaymentMethod = "credit_card"
	PaymentMethodAlipay     PaymentMethod = "alipay"
	PaymentMethodWeChat     PaymentMethod = "wechat"
	PaymentMethodBank       PaymentMethod = "bank_transfer"
)

type Bill struct {
	ID          uuid.UUID      `json:"id"`
	TenantID    uuid.UUID      `json:"tenant_id"`
	JobID       uuid.UUID      `json:"job_id,omitempty"`
	OrderID     uuid.UUID      `json:"order_id,omitempty"`
	Description string         `json:"description"`
	Usage       ResourceUsage `json:"resource_usage"`
	UnitPrice   PriceUnit     `json:"unit_price"`
	Quantity    float64        `json:"quantity"`
	TotalAmount float64       `json:"total_amount"`
	Currency    string        `json:"currency"`
	Discount    float64       `json:"discount"`
	Status      BillStatus    `json:"status"`
	DueDate     time.Time     `json:"due_date"`
	PaidAt      *time.Time    `json:"paid_at,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`
}

type ResourceUsage struct {
	CPUHours      float64 `json:"cpu_hours"`
	MemoryGBHours float64 `json:"memory_gb_hours"`
	GPUHours      float64 `json:"gpu_hours"`
	StorageGBHours float64 `json:"storage_gb_hours"`
	NetworkGB     float64 `json:"network_gb"`
}

type PriceUnit struct {
	CPU      float64 `json:"cpu"`
	Memory   float64 `json:"memory"`
	GPU      float64 `json:"gpu"`
	Storage  float64 `json:"storage"`
	Network  float64 `json:"network"`
	Currency string  `json:"currency"`
}

type BillingService struct {
	bills     map[uuid.UUID]*Bill
	summaries map[uuid.UUID]*BillingSummary
	prices    *PriceUnit
	mu        sync.RWMutex
}

func NewBillingService() *BillingService {
	return &BillingService{
		bills:     make(map[uuid.UUID]*Bill),
		summaries: make(map[uuid.UUID]*BillingSummary),
		prices:    DefaultPriceUnits(),
	}
}

func DefaultPriceUnits() *PriceUnit {
	return &PriceUnit{
		CPU:      0.05,
		Memory:   0.01,
		GPU:      2.00,
		Storage:  0.001,
		Network:  0.05,
		Currency: "CNY",
	}
}

func (bs *BillingService) CreateBill(ctx context.Context, bill *Bill) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	if bill.ID == uuid.Nil {
		bill.ID = uuid.New()
	}
	bill.Status = BillStatusPending
	bill.Currency = bs.prices.Currency
	bill.UnitPrice = *bs.prices
	bill.TotalAmount = bs.CalculateTotal(bill)
	bill.CreatedAt = time.Now()
	bill.UpdatedAt = time.Now()

	if bill.DueDate.IsZero() {
		bill.DueDate = bill.CreatedAt.Add(30 * 24 * time.Hour)
	}

	bs.bills[bill.ID] = bill
	klog.Infof("Created bill %s for tenant %s, amount: %.2f", bill.ID, bill.TenantID, bill.TotalAmount)
	return nil
}

func (bs *BillingService) CalculateTotal(bill *Bill) float64 {
	usage := bill.Usage
	price := bill.UnitPrice

	cpuCost := usage.CPUHours * price.CPU
	memCost := usage.MemoryGBHours * price.Memory
	gpuCost := usage.GPUHours * price.GPU
	storageCost := usage.StorageGBHours * price.Storage
	networkCost := usage.NetworkGB * price.Network

	total := cpuCost + memCost + gpuCost + storageCost + networkCost

	if bill.Discount > 0 {
		total = total * (1 - bill.Discount/100)
	}

	return total
}

func (bs *BillingService) GetBill(ctx context.Context, billID uuid.UUID) (*Bill, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	bill, exists := bs.bills[billID]
	if !exists {
		return nil, fmt.Errorf("bill %s not found", billID)
	}
	return bill, nil
}

func (bs *BillingService) ListBills(ctx context.Context, tenantID uuid.UUID, filter *BillFilter) ([]*Bill, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	var result []*Bill
	for _, bill := range bs.bills {
		if tenantID != uuid.Nil && bill.TenantID != tenantID {
			continue
		}
		if filter != nil {
			if filter.Status != "" && bill.Status != filter.Status {
				continue
			}
			if !filter.StartDate.IsZero() && bill.CreatedAt.Before(filter.StartDate) {
				continue
			}
			if !filter.EndDate.IsZero() && bill.CreatedAt.After(filter.EndDate) {
				continue
			}
		}
		result = append(result, bill)
	}
	return result, nil
}

func (bs *BillingService) UpdateBill(ctx context.Context, billID uuid.UUID, updates *BillUpdate) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	bill, exists := bs.bills[billID]
	if !exists {
		return fmt.Errorf("bill %s not found", billID)
	}

	if updates.ResourceUsage != nil {
		bill.Usage = *updates.ResourceUsage
		bill.TotalAmount = bs.CalculateTotal(bill)
	}
	if updates.Discount != nil {
		bill.Discount = *updates.Discount
		bill.TotalAmount = bs.CalculateTotal(bill)
	}
	if updates.Status != "" {
		bill.Status = updates.Status
	}
	if updates.DueDate != nil {
		bill.DueDate = *updates.DueDate
	}
	bill.UpdatedAt = time.Now()

	return nil
}

func (bs *BillingService) PayBill(ctx context.Context, billID uuid.UUID, method PaymentMethod) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	bill, exists := bs.bills[billID]
	if !exists {
		return fmt.Errorf("bill %s not found", billID)
	}

	if bill.Status == BillStatusPaid {
		return fmt.Errorf("bill %s already paid", billID)
	}

	now := time.Now()
	bill.Status = BillStatusPaid
	bill.PaidAt = &now
	bill.UpdatedAt = now

	klog.Infof("Bill %s paid via %s", billID, method)
	return nil
}

func (bs *BillingService) CancelBill(ctx context.Context, billID uuid.UUID) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	bill, exists := bs.bills[billID]
	if !exists {
		return fmt.Errorf("bill %s not found", billID)
	}

	if bill.Status == BillStatusPaid {
		return fmt.Errorf("cannot cancel paid bill")
	}

	bill.Status = BillStatusCancelled
	bill.UpdatedAt = time.Now()

	return nil
}

func (bs *BillingService) GetBillingSummary(ctx context.Context, tenantID uuid.UUID, period *BillingPeriod) (*BillingSummary, error) {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	bills, err := bs.filterBills(tenantID, period)
	if err != nil {
		return nil, err
	}

	summary := &BillingSummary{
		TenantID: tenantID,
		Period:   *period,
	}

	for _, bill := range bills {
		summary.TotalAmount += bill.TotalAmount
		if bill.Status == BillStatusPaid {
			summary.PaidAmount += bill.TotalAmount
		}
		if bill.Status == BillStatusPending {
			summary.PendingAmount += bill.TotalAmount
		}

		summary.ResourceUsage.CPUHours += bill.Usage.CPUHours
		summary.ResourceUsage.MemoryGBHours += bill.Usage.MemoryGBHours
		summary.ResourceUsage.GPUHours += bill.Usage.GPUHours
		summary.ResourceUsage.StorageGBHours += bill.Usage.StorageGBHours
		summary.ResourceUsage.NetworkGB += bill.Usage.NetworkGB
		summary.BillCount++
	}

	return summary, nil
}

func (bs *BillingService) filterBills(tenantID uuid.UUID, period *BillingPeriod) ([]*Bill, error) {
	var result []*Bill
	for _, bill := range bs.bills {
		if tenantID != uuid.Nil && bill.TenantID != tenantID {
			continue
		}
		if period != nil {
			if !period.StartDate.IsZero() && bill.CreatedAt.Before(period.StartDate) {
				continue
			}
			if !period.EndDate.IsZero() && bill.CreatedAt.After(period.EndDate) {
				continue
			}
		}
		result = append(result, bill)
	}
	return result, nil
}

func (bs *BillingService) ExportBills(ctx context.Context, tenantID uuid.UUID, format string) ([]byte, error) {
	bills, err := bs.ListBills(ctx, tenantID, nil)
	if err != nil {
		return nil, err
	}

	switch format {
	case "csv":
		return bs.exportCSV(bills)
	case "json":
		return bs.exportJSON(bills)
	default:
		return bs.exportCSV(bills)
	}
}

func (bs *BillingService) exportCSV(bills []*Bill) ([]byte, error) {
	var csv string
	csv = "ID,TenantID,Description,Amount,Currency,Status,DueDate,CreatedAt\n"
	for _, bill := range bills {
		csv += fmt.Sprintf("%s,%s,%s,%.2f,%s,%s,%s,%s\n",
			bill.ID, bill.TenantID, bill.Description, bill.TotalAmount,
			bill.Currency, bill.Status, bill.DueDate.Format(time.RFC3339),
			bill.CreatedAt.Format(time.RFC3339))
	}
	return []byte(csv), nil
}

func (bs *BillingService) exportJSON(bills []*Bill) ([]byte, error) {
	return []byte(fmt.Sprintf("{\"bills\": %v}", bills)), nil
}

func (bs *BillingService) UpdatePrices(ctx context.Context, prices *PriceUnit) error {
	bs.mu.Lock()
	defer bs.mu.Unlock()

	bs.prices = prices
	klog.Infof("Updated billing prices to: CPU=%.4f, Memory=%.4f, GPU=%.4f",
		prices.CPU, prices.Memory, prices.GPU)
	return nil
}

func (bs *BillingService) GetPrices(ctx context.Context) *PriceUnit {
	bs.mu.RLock()
	defer bs.mu.RUnlock()

	price := *bs.prices
	return &price
}

type BillFilter struct {
	Status    BillStatus
	StartDate time.Time
	EndDate   time.Time
}

type BillUpdate struct {
	ResourceUsage *ResourceUsage
	Discount      *float64
	Status        BillStatus
	DueDate       *time.Time
}

type BillingPeriod struct {
	StartDate time.Time
	EndDate   time.Time
}

type BillingSummary struct {
	TenantID      uuid.UUID      `json:"tenant_id"`
	Period        BillingPeriod `json:"period"`
	TotalAmount   float64       `json:"total_amount"`
	PaidAmount    float64       `json:"paid_amount"`
	PendingAmount float64       `json:"pending_amount"`
	ResourceUsage ResourceUsage `json:"resource_usage"`
	BillCount     int           `json:"bill_count"`
}

type BillWithUsage struct {
	Bill  *Bill  `json:"bill"`
	Usage string `json:"usage_summary"`
}

func CalculateResourceCost(usage *ResourceUsage, prices *PriceUnit) float64 {
	return usage.CPUHours*prices.CPU +
		usage.MemoryGBHours*prices.Memory +
		usage.GPUHours*prices.GPU +
		usage.StorageGBHours*prices.Storage +
		usage.NetworkGB*prices.Network
}
