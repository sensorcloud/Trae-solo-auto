package billing

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBillingService_CreateBill(t *testing.T) {
	service := NewBillingService()
	ctx := context.Background()

	bill := &Bill{
		TenantID: uuid.New(),
		Description: "Test Bill",
		Usage: ResourceUsage{
			CPUHours:      100,
			MemoryGBHours: 200,
			GPUHours:      50,
		},
	}

	err := service.CreateBill(ctx, bill)
	if err != nil {
		t.Fatalf("Failed to create bill: %v", err)
	}

	if bill.ID == uuid.Nil {
		t.Error("Bill ID should not be nil")
	}

	if bill.Status != BillStatusPending {
		t.Errorf("Expected status %s, got %s", BillStatusPending, bill.Status)
	}

	expectedAmount := 100*0.05 + 200*0.01 + 50*2.0
	if bill.TotalAmount != expectedAmount {
		t.Errorf("Expected amount %.2f, got %.2f", expectedAmount, bill.TotalAmount)
	}
}

func TestBillingService_GetBill(t *testing.T) {
	service := NewBillingService()
	ctx := context.Background()

	bill := &Bill{
		TenantID:    uuid.New(),
		Description: "Test Bill",
		Usage: ResourceUsage{
			CPUHours: 100,
		},
	}
	service.CreateBill(ctx, bill)

	retrieved, err := service.GetBill(ctx, bill.ID)
	if err != nil {
		t.Fatalf("Failed to get bill: %v", err)
	}

	if retrieved.ID != bill.ID {
		t.Errorf("Expected ID %s, got %s", bill.ID, retrieved.ID)
	}
}

func TestBillingService_ListBills(t *testing.T) {
	service := NewBillingService()
	ctx := context.Background()
	tenantID := uuid.New()

	for i := 0; i < 3; i++ {
		bill := &Bill{
			TenantID:    tenantID,
			Description: "Test Bill",
			Usage: ResourceUsage{
				CPUHours: 100,
			},
		}
		service.CreateBill(ctx, bill)
	}

	bills, err := service.ListBills(ctx, tenantID, nil)
	if err != nil {
		t.Fatalf("Failed to list bills: %v", err)
	}

	if len(bills) != 3 {
		t.Errorf("Expected 3 bills, got %d", len(bills))
	}
}

func TestBillingService_PayBill(t *testing.T) {
	service := NewBillingService()
	ctx := context.Background()

	bill := &Bill{
		TenantID:    uuid.New(),
		Description: "Test Bill",
		Usage: ResourceUsage{
			CPUHours: 100,
		},
	}
	service.CreateBill(ctx, bill)

	err := service.PayBill(ctx, bill.ID, PaymentMethodAlipay)
	if err != nil {
		t.Fatalf("Failed to pay bill: %v", err)
	}

	paidBill, _ := service.GetBill(ctx, bill.ID)
	if paidBill.Status != BillStatusPaid {
		t.Errorf("Expected status %s, got %s", BillStatusPaid, paidBill.Status)
	}

	if paidBill.PaidAt == nil {
		t.Error("PaidAt should not be nil")
	}
}

func TestBillingService_GetBillingSummary(t *testing.T) {
	service := NewBillingService()
	ctx := context.Background()
	tenantID := uuid.New()

	for i := 0; i < 3; i++ {
		bill := &Bill{
			TenantID:    tenantID,
			Description: "Test Bill",
			Usage: ResourceUsage{
				CPUHours: 100,
				GPUHours: 10,
			},
		}
		service.CreateBill(ctx, bill)
	}

	period := &BillingPeriod{
		StartDate: time.Now().Add(-24 * time.Hour),
		EndDate:   time.Now().Add(24 * time.Hour),
	}

	summary, err := service.GetBillingSummary(ctx, tenantID, period)
	if err != nil {
		t.Fatalf("Failed to get billing summary: %v", err)
	}

	if summary.BillCount != 3 {
		t.Errorf("Expected 3 bill count, got %d", summary.BillCount)
	}

	if summary.TotalAmount <= 0 {
		t.Error("Total amount should be greater than 0")
	}
}

func TestBillingService_UpdatePrices(t *testing.T) {
	service := NewBillingService()
	ctx := context.Background()

	prices := &PriceUnit{
		CPU:      0.10,
		Memory:   0.02,
		GPU:      4.00,
		Storage:  0.002,
		Network:  0.10,
		Currency: "CNY",
	}

	err := service.UpdatePrices(ctx, prices)
	if err != nil {
		t.Fatalf("Failed to update prices: %v", err)
	}

	retrieved := service.GetPrices(ctx)
	if retrieved.CPU != 0.10 {
		t.Errorf("Expected CPU price 0.10, got %.4f", retrieved.CPU)
	}
}

func TestCalculateResourceCost(t *testing.T) {
	usage := &ResourceUsage{
		CPUHours:      100,
		MemoryGBHours: 200,
		GPUHours:      50,
		StorageGBHours: 1000,
		NetworkGB:     50,
	}

	prices := &PriceUnit{
		CPU:     0.05,
		Memory:  0.01,
		GPU:     2.00,
		Storage: 0.001,
		Network: 0.05,
	}

	cost := CalculateResourceCost(usage, prices)
	expected := 100*0.05 + 200*0.01 + 50*2.0 + 1000*0.001 + 50*0.05

	if cost != expected {
		t.Errorf("Expected cost %.2f, got %.2f", expected, cost)
	}
}
