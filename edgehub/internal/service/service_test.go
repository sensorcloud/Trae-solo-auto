package service

import (
	"context"
	"testing"
	"time"

	"github.com/edgehub/edgehub/internal/models"
	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect to test database: %v", err)
	}

	if err := db.AutoMigrate(
		&models.Node{},
		&models.Job{},
		&models.MarketOffer{},
		&models.MarketOrder{},
		&models.Bill{},
	); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}

	return db
}

func TestNodeService_Register(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNodeService(db)
	ctx := context.Background()

	node := &models.Node{
		Name: "test-node-1",
		HardwareInfo: models.HardwareInfo{
			CPUModel: "Intel Xeon",
			CPUCores: 8,
		},
		Allocatable: models.Allocatable{
			CPU:    8,
			Memory: 16 * 1024 * 1024 * 1024,
		},
	}

	if err := svc.Register(ctx, node); err != nil {
		t.Fatalf("failed to register node: %v", err)
	}

	if node.ID == uuid.Nil {
		t.Error("node ID should not be nil after registration")
	}

	if node.Status != "pending" {
		t.Errorf("expected status 'pending', got '%s'", node.Status)
	}
}

func TestNodeService_List(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNodeService(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		node := &models.Node{
			Name:   "test-node-" + string(rune('a'+i)),
			Status: "online",
			Labels: map[string]interface{}{"region": "ap-southeast-1"},
		}
		if err := svc.Register(ctx, node); err != nil {
			t.Fatalf("failed to register node: %v", err)
		}
	}

	nodes, total, err := svc.List(ctx, &NodeFilter{
		Offset: 0,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("failed to list nodes: %v", err)
	}

	if total != 5 {
		t.Errorf("expected 5 nodes, got %d", total)
	}

	if len(nodes) != 5 {
		t.Errorf("expected 5 nodes in list, got %d", len(nodes))
	}
}

func TestNodeService_GetByID(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNodeService(db)
	ctx := context.Background()

	node := &models.Node{Name: "test-node"}
	if err := svc.Register(ctx, node); err != nil {
		t.Fatalf("failed to register node: %v", err)
	}

	retrieved, err := svc.GetByID(ctx, node.ID)
	if err != nil {
		t.Fatalf("failed to get node: %v", err)
	}

	if retrieved.Name != node.Name {
		t.Errorf("expected name '%s', got '%s'", node.Name, retrieved.Name)
	}
}

func TestNodeService_Heartbeat(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNodeService(db)
	ctx := context.Background()

	node := &models.Node{Name: "test-node"}
	if err := svc.Register(ctx, node); err != nil {
		t.Fatalf("failed to register node: %v", err)
	}

	metrics := &NodeHeartbeatMetrics{
		CPUUsage:      45.5,
		MemoryUsage:   62.3,
		CPUAllocatable: 4,
	}

	if err := svc.Heartbeat(ctx, node.ID, metrics); err != nil {
		t.Fatalf("failed to send heartbeat: %v", err)
	}

	updated, _ := svc.GetByID(ctx, node.ID)
	if updated.Status != "online" {
		t.Errorf("expected status 'online', got '%s'", updated.Status)
	}
}

func TestJobService_Submit(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db)
	ctx := context.Background()

	job := &models.Job{
		Name:      "test-job",
		Type:      "container",
		TenantID:  uuid.New(),
		UserID:    uuid.New(),
		Namespace: "default",
		Spec: models.JobSpec{
			Image:   "nginx:latest",
			Command: []string{"nginx"},
		},
	}

	if err := svc.Submit(ctx, job); err != nil {
		t.Fatalf("failed to submit job: %v", err)
	}

	if job.Status != "pending" {
		t.Errorf("expected status 'pending', got '%s'", job.Status)
	}
}

func TestJobService_List(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db)
	ctx := context.Background()

	tenantID := uuid.New()
	for i := 0; i < 3; i++ {
		job := &models.Job{
			Name:      "test-job-" + string(rune('0'+i)),
			Type:      "container",
			TenantID:  tenantID,
			UserID:    uuid.New(),
			Namespace: "default",
		}
		if err := svc.Submit(ctx, job); err != nil {
			t.Fatalf("failed to submit job: %v", err)
		}
	}

	jobs, total, err := svc.List(ctx, &JobFilter{
		TenantID: tenantID,
		Offset:   0,
		Limit:    10,
	})
	if err != nil {
		t.Fatalf("failed to list jobs: %v", err)
	}

	if total != 3 {
		t.Errorf("expected 3 jobs, got %d", total)
	}

	if len(jobs) != 3 {
		t.Errorf("expected 3 jobs in list, got %d", len(jobs))
	}
}

func TestJobService_UpdateStatus(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db)
	ctx := context.Background()

	job := &models.Job{
		Name:      "test-job",
		Type:      "container",
		TenantID:  uuid.New(),
		UserID:    uuid.New(),
		Namespace: "default",
	}
	if err := svc.Submit(ctx, job); err != nil {
		t.Fatalf("failed to submit job: %v", err)
	}

	if err := svc.UpdateStatus(ctx, job.ID, "running"); err != nil {
		t.Fatalf("failed to update job status: %v", err)
	}

	updated, _ := svc.GetByID(ctx, job.ID)
	if updated.Status != "running" {
		t.Errorf("expected status 'running', got '%s'", updated.Status)
	}

	if updated.StartedAt.IsZero() {
		t.Error("expected StartedAt to be set")
	}
}

func TestJobService_Stop(t *testing.T) {
	db := setupTestDB(t)
	svc := NewJobService(db)
	ctx := context.Background()

	job := &models.Job{
		Name:      "test-job",
		Type:      "container",
		TenantID:  uuid.New(),
		UserID:    uuid.New(),
		Namespace: "default",
	}
	if err := svc.Submit(ctx, job); err != nil {
		t.Fatalf("failed to submit job: %v", err)
	}

	if err := svc.Stop(ctx, job.ID); err != nil {
		t.Fatalf("failed to stop job: %v", err)
	}

	updated, _ := svc.GetByID(ctx, job.ID)
	if updated.Status != "cancelled" {
		t.Errorf("expected status 'cancelled', got '%s'", updated.Status)
	}

	if updated.FinishedAt.IsZero() {
		t.Error("expected FinishedAt to be set")
	}
}

func TestMarketService_CreateOffer(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMarketService(db)
	ctx := context.Background()

	offer := &models.MarketOffer{
		ProviderID: uuid.New(),
		Region:     "ap-southeast-1",
		ResourceSpec: models.ResourceSpec{
			CPU:    8,
			Memory: 32 * 1024 * 1024 * 1024,
			GPU:    1,
			GPUType: "NVIDIA A100",
		},
		Prices: models.Prices{
			OnDemand:   2.50,
			Spot:       1.00,
			Reserved1M: 2.00,
		},
	}

	if err := svc.CreateOffer(ctx, offer); err != nil {
		t.Fatalf("failed to create offer: %v", err)
	}

	if !offer.Available {
		t.Error("expected offer to be available")
	}
}

func TestMarketService_ListOffers(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMarketService(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		offer := &models.MarketOffer{
			ProviderID: uuid.New(),
			Region:     "ap-southeast-1",
			ResourceSpec: models.ResourceSpec{
				CPU:  8 + float64(i),
				GPU:  1,
				GPUType: "NVIDIA A100",
			},
			Prices: models.Prices{
				OnDemand: 2.0 + float64(i),
				Spot:     1.0 + float64(i),
			},
		}
		if err := svc.CreateOffer(ctx, offer); err != nil {
			t.Fatalf("failed to create offer: %v", err)
		}
	}

	offers, total, err := svc.ListOffers(ctx, &OfferFilter{
		Region: "ap-southeast-1",
		MinGPU: 1,
		Limit:  10,
	})
	if err != nil {
		t.Fatalf("failed to list offers: %v", err)
	}

	if total != 3 {
		t.Errorf("expected 3 offers, got %d", total)
	}

	if len(offers) != 3 {
		t.Errorf("expected 3 offers in list, got %d", len(offers))
	}
}

func TestMarketService_GetRecommendation(t *testing.T) {
	db := setupTestDB(t)
	svc := NewMarketService(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		offer := &models.MarketOffer{
			ProviderID: uuid.New(),
			Region:     "ap-southeast-1",
			ResourceSpec: models.ResourceSpec{
				CPU:  8,
				GPU:  1,
				GPUType: "NVIDIA A100",
			},
			Prices: models.Prices{
				OnDemand: 2.0 + float64(i)*0.5,
				Spot:     1.0 + float64(i)*0.3,
			},
		}
		svc.CreateOffer(ctx, offer)
	}

	rec, err := svc.GetRecommendation(ctx, &RecommendationRequest{
		Region:      "ap-southeast-1",
		MinGPU:      1,
		BillingType: "spot",
	})
	if err != nil {
		t.Fatalf("failed to get recommendation: %v", err)
	}

	if len(rec.Recommendations) == 0 {
		t.Error("expected at least one recommendation")
	}
}

func TestBillingService_GetSummary(t *testing.T) {
	db := setupTestDB(t)
	svc := NewBillingService(db, nil)
	ctx := context.Background()

	tenantID := uuid.New()
	start := time.Now().Add(-30 * 24 * time.Hour)
	end := time.Now()

	for i := 0; i < 3; i++ {
		bill := &models.Bill{
			TenantID:     tenantID,
			ResourceType: "gpu",
			Quantity:    1,
			UnitPrice:   2.0,
			TotalAmount: 2.0,
			PeriodStart: start,
			PeriodEnd:   end,
		}
		db.Create(bill)
	}

	summary, err := svc.GetSummary(ctx, tenantID, start, end)
	if err != nil {
		t.Fatalf("failed to get summary: %v", err)
	}

	if summary.TotalAmount != 6.0 {
		t.Errorf("expected total amount 6.0, got %f", summary.TotalAmount)
	}
}

func TestNodeFilter_Labels(t *testing.T) {
	db := setupTestDB(t)
	svc := NewNodeService(db)
	ctx := context.Background()

	labels := map[string]interface{}{
		"region": "ap-southeast-1",
		"env":    "prod",
	}
	node := &models.Node{
		Name:   "labeled-node",
		Labels: labels,
	}
	if err := svc.Register(ctx, node); err != nil {
		t.Fatalf("failed to register node: %v", err)
	}

	filter := &NodeFilter{
		Labels: map[string]string{"region": "ap-southeast-1"},
	}
	nodes, _, err := svc.List(ctx, filter)
	if err != nil {
		t.Fatalf("failed to list nodes: %v", err)
	}

	if len(nodes) != 1 {
		t.Errorf("expected 1 node, got %d", len(nodes))
	}
}
