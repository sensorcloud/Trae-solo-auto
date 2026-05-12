package service

import (
	"context"
	"time"

	"github.com/edgehub/edgehub/internal/models"
	"github.com/edgehub/edgehub/internal/repository"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type NodeService struct {
	db *gorm.DB
}

func NewNodeService(db *gorm.DB) *NodeService {
	return &NodeService{db: db}
}

func (s *NodeService) Register(ctx context.Context, node *models.Node) error {
	node.Status = "pending"
	node.HeartbeatAt = time.Now()
	node.LastSeenAt = time.Now()
	return s.db.WithContext(ctx).Create(node).Error
}

func (s *NodeService) GetByID(ctx context.Context, id uuid.UUID) (*models.Node, error) {
	var node models.Node
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&node).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

func (s *NodeService) List(ctx context.Context, filter *NodeFilter) ([]models.Node, int64, error) {
	var nodes []models.Node
	var total int64

	query := s.db.WithContext(ctx).Model(&models.Node{})

	if filter != nil {
		if filter.ClusterID != uuid.Nil {
			query = query.Where("cluster_id = ?", filter.ClusterID)
		}
		if filter.Status != "" {
			query = query.Where("status = ?", filter.Status)
		}
		if filter.Region != "" {
			query = query.Where("labels->>'region' = ?", filter.Region)
		}
		if filter.HasGPU {
			query = query.Where("allocatable->>'gpu' > ?", 0)
		}
		if filter.Labels != nil {
			for k, v := range filter.Labels {
				query = query.Where("labels->>? = ?", k, v)
			}
		}
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	if filter != nil {
		if filter.Offset > 0 {
			query = query.Offset(filter.Offset)
		}
		if filter.Limit > 0 {
			query = query.Limit(filter.Limit)
		}
		if filter.SortBy != "" {
			order := "desc"
			if filter.Order == "asc" {
				order = "asc"
			}
			query = query.Order(filter.SortBy + " " + order)
		}
	} else {
		query = query.Limit(100)
	}

	err = query.Find(&nodes).Error
	return nodes, total, err
}

func (s *NodeService) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&models.Node{}).Where("id = ?", id).Updates(updates).Error
}

func (s *NodeService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Delete(&models.Node{}, "id = ?", id).Error
}

func (s *NodeService) Heartbeat(ctx context.Context, id uuid.UUID, metrics *NodeHeartbeatMetrics) error {
	updates := map[string]interface{}{
		"heartbeat_at": time.Now(),
		"last_seen_at": time.Now(),
		"status":       "online",
	}

	if metrics != nil {
		updates["allocated"] = models.Allocatable{
			CPU:    metrics.CPUAllocatable,
			Memory: metrics.MemoryAllocatable,
			GPU:    metrics.GPUAllocatable,
		}
	}

	return s.db.WithContext(ctx).Model(&models.Node{}).Where("id = ?", id).Updates(updates).Error
}

func (s *NodeService) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	return s.db.WithContext(ctx).Model(&models.Node{}).Where("id = ?", id).Update("status", status).Error
}

func (s *NodeService) GetMetrics(ctx context.Context, id uuid.UUID, start, end time.Time) ([]models.NodeMetrics, error) {
	var metrics []models.NodeMetrics
	err := s.db.WithContext(ctx).
		Where("node_id = ? AND timestamp BETWEEN ? AND ?", id, start, end).
		Order("timestamp desc").
		Find(&metrics).Error
	return metrics, err
}

func (s *NodeService) RecordMetrics(ctx context.Context, m *models.NodeMetrics) error {
	return s.db.WithContext(ctx).Create(m).Error
}

type NodeFilter struct {
	ClusterID uuid.UUID
	Status    string
	Region    string
	HasGPU    bool
	Labels    map[string]string
	Offset    int
	Limit     int
	SortBy    string
	Order     string
}

type NodeHeartbeatMetrics struct {
	CPUUsage      float64
	MemoryUsage   float64
	GPUUsage      float64
	CPUAllocatable float64
	MemoryAllocatable int64
	GPUAllocatable   int
}

type JobService struct {
	db *gorm.DB
}

func NewJobService(db *gorm.DB) *JobService {
	return &JobService{db: db}
}

func (s *JobService) Submit(ctx context.Context, job *models.Job) error {
	job.Status = "pending"
	job.CreatedAt = time.Now()
	return s.db.WithContext(ctx).Create(job).Error
}

func (s *JobService) GetByID(ctx context.Context, id uuid.UUID) (*models.Job, error) {
	var job models.Job
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&job).Error
	if err != nil {
		return nil, err
	}
	return &job, nil
}

func (s *JobService) List(ctx context.Context, filter *JobFilter) ([]models.Job, int64, error) {
	var jobs []models.Job
	var total int64

	query := s.db.WithContext(ctx).Model(&models.Job{})

	if filter != nil {
		if filter.TenantID != uuid.Nil {
			query = query.Where("tenant_id = ?", filter.TenantID)
		}
		if filter.ClusterID != uuid.Nil {
			query = query.Where("cluster_id = ?", filter.ClusterID)
		}
		if filter.Status != "" {
			query = query.Where("status = ?", filter.Status)
		}
		if filter.Type != "" {
			query = query.Where("type = ?", filter.Type)
		}
		if filter.Queue != "" {
			query = query.Where("queue = ?", filter.Queue)
		}
		if filter.UserID != uuid.Nil {
			query = query.Where("user_id = ?", filter.UserID)
		}
		if filter.CreatedAfter != nil {
			query = query.Where("created_at >= ?", filter.CreatedAfter)
		}
		if filter.CreatedBefore != nil {
			query = query.Where("created_at <= ?", filter.CreatedBefore)
		}
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	if filter != nil {
		if filter.Offset > 0 {
			query = query.Offset(filter.Offset)
		}
		if filter.Limit > 0 {
			query = query.Limit(filter.Limit)
		} else {
			query = query.Limit(50)
		}
		query = query.Order("priority desc, created_at asc")
	}

	err = query.Find(&jobs).Error
	return jobs, total, err
}

func (s *JobService) Update(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&models.Job{}).Where("id = ?", id).Updates(updates).Error
}

func (s *JobService) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	updates := map[string]interface{}{"status": status}
	now := time.Now()

	switch status {
	case "running":
		updates["started_at"] = now
	case "succeeded", "failed", "cancelled":
		updates["finished_at"] = now
	}

	return s.db.WithContext(ctx).Model(&models.Job{}).Where("id = ?", id).Updates(updates).Error
}

func (s *JobService) Delete(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Delete(&models.Job{}, "id = ?", id).Error
}

func (s *JobService) Stop(ctx context.Context, id uuid.UUID) error {
	return s.UpdateStatus(ctx, id, "cancelled")
}

func (s *JobService) GetLogs(ctx context.Context, id uuid.UUID) (string, error) {
	var job models.Job
	err := s.db.WithContext(ctx).Select("id, name, namespace, status").Where("id = ?", id).First(&job).Error
	if err != nil {
		return "", err
	}
	return "", nil
}

type JobFilter struct {
	TenantID      uuid.UUID
	ClusterID      uuid.UUID
	NodeID         uuid.UUID
	Status         string
	Type           string
	Queue          string
	UserID         uuid.UUID
	CreatedAfter   *time.Time
	CreatedBefore  *time.Time
	Offset         int
	Limit          int
}

type MarketService struct {
	db *gorm.DB
}

func NewMarketService(db *gorm.DB) *MarketService {
	return &MarketService{db: db}
}

func (s *MarketService) CreateOffer(ctx context.Context, offer *models.MarketOffer) error {
	offer.Status = "active"
	offer.Available = true
	return s.db.WithContext(ctx).Create(offer).Error
}

func (s *MarketService) ListOffers(ctx context.Context, filter *OfferFilter) ([]models.MarketOffer, int64, error) {
	var offers []models.MarketOffer
	var total int64

	query := s.db.WithContext(ctx).Model(&models.MarketOffer{}).Where("available = ?", true)

	if filter != nil {
		if filter.Region != "" {
			query = query.Where("region = ?", filter.Region)
		}
		if filter.MinCPU > 0 {
			query = query.Where("(resource_spec->>'cpu')::float >= ?", filter.MinCPU)
		}
		if filter.MinGPU > 0 {
			query = query.Where("(resource_spec->>'gpu')::int >= ?", filter.MinGPU)
		}
		if filter.MaxPrice > 0 {
			query = query.Where("prices->>'on_demand' <= ?", filter.MaxPrice)
		}
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	if filter != nil {
		if filter.Limit > 0 {
			query = query.Limit(filter.Limit)
		} else {
			query = query.Limit(50)
		}
		query = query.Order("prices->>'on_demand' asc")
	}

	err = query.Find(&offers).Error
	return offers, total, err
}

func (s *MarketService) GetOffer(ctx context.Context, id uuid.UUID) (*models.MarketOffer, error) {
	var offer models.MarketOffer
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&offer).Error
	return &offer, err
}

func (s *MarketService) UpdateOffer(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&models.MarketOffer{}).Where("id = ?", id).Updates(updates).Error
}

func (s *MarketService) DeleteOffer(ctx context.Context, id uuid.UUID) error {
	return s.db.WithContext(ctx).Model(&models.MarketOffer{}).Where("id = ?", id).Updates(map[string]interface{}{
		"available": false,
		"status":   "deleted",
	}).Error
}

func (s *MarketService) CreateOrder(ctx context.Context, order *models.MarketOrder) error {
	order.Status = "pending"
	return s.db.WithContext(ctx).Create(order).Error
}

func (s *MarketService) GetOrder(ctx context.Context, id uuid.UUID) (*models.MarketOrder, error) {
	var order models.MarketOrder
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&order).Error
	return &order, err
}

func (s *MarketService) UpdateOrderStatus(ctx context.Context, id uuid.UUID, status string) error {
	updates := map[string]interface{}{"status": status}
	if status == "fulfilled" {
		now := time.Now()
		updates["fulfilled_at"] = &now
	}
	return s.db.WithContext(ctx).Model(&models.MarketOrder{}).Where("id = ?", id).Updates(updates).Error
}

func (s *MarketService) GetPrices(ctx context.Context, filter *PriceFilter) (*PriceData, error) {
	var offers []models.MarketOffer
	query := s.db.WithContext(ctx).Model(&models.MarketOffer{}).Where("available = ?", true)

	if filter != nil && filter.Region != "" {
		query = query.Where("region = ?", filter.Region)
	}

	query.Find(&offers)

	priceData := &PriceData{
		Prices: make([]PricePoint, 0),
	}

	for _, offer := range offers {
		priceData.Prices = append(priceData.Prices, PricePoint{
			OfferID:  offer.ID,
			Region:   offer.Region,
			OnDemand: offer.Prices.OnDemand,
			Spot:     offer.Prices.Spot,
			GPUType:  offer.ResourceSpec.GPUType,
		})
	}

	return priceData, nil
}

func (s *MarketService) GetRecommendation(ctx context.Context, req *RecommendationRequest) (*RecommendationResponse, error) {
	offers, _, err := s.ListOffers(ctx, &OfferFilter{
		Region: req.Region,
		MinCPU: req.MinCPU,
		MinGPU: req.MinGPU,
		Limit:  10,
	})
	if err != nil {
		return nil, err
	}

	rec := &RecommendationResponse{
		Recommendations: make([]Recommendation, 0),
	}

	for _, offer := range offers {
		price := offer.Prices.OnDemand
		if req.BillingType == "spot" {
			price = offer.Prices.Spot
		} else if req.BillingType == "reserved_1y" {
			price = offer.Prices.Reserved12M
		}

		rec.Recommendations = append(rec.Recommendations, Recommendation{
			OfferID:      offer.ID,
			Region:       offer.Region,
			Price:        price,
			ResourceSpec: offer.ResourceSpec,
			Score:        100 - float64(offer.Prices.OnDemand),
		})
	}

	return rec, nil
}

type OfferFilter struct {
	Region   string
	MinCPU   float64
	MinGPU   int
	MaxPrice float64
	Limit    int
}

type PriceFilter struct {
	Region   string
	GPUType  string
}

type PriceData struct {
	Prices []PricePoint `json:"prices"`
}

type PricePoint struct {
	OfferID  uuid.UUID `json:"offer_id"`
	Region   string    `json:"region"`
	OnDemand float64   `json:"on_demand"`
	Spot     float64   `json:"spot"`
	GPUType  string    `json:"gpu_type,omitempty"`
}

type RecommendationRequest struct {
	Region      string  `json:"region"`
	MinCPU      float64 `json:"min_cpu"`
	MinGPU      int     `json:"min_gpu"`
	MinMemory   int64   `json:"min_memory"`
	BillingType string  `json:"billing_type"`
	MaxPrice    float64 `json:"max_price"`
}

type RecommendationResponse struct {
	Recommendations []Recommendation `json:"recommendations"`
}

type Recommendation struct {
	OfferID      uuid.UUID          `json:"offer_id"`
	Region       string             `json:"region"`
	Price        float64            `json:"price"`
	ResourceSpec models.ResourceSpec `json:"resource_spec"`
	Score        float64            `json:"score"`
}

type BillingService struct {
	db    *gorm.DB
	redis *repository.RedisClient
}

func NewBillingService(db *gorm.DB, redis *repository.RedisClient) *BillingService {
	return &BillingService{db: db, redis: redis}
}

func (s *BillingService) List(ctx context.Context, filter *BillFilter) ([]models.Bill, int64, error) {
	var bills []models.Bill
	var total int64

	query := s.db.WithContext(ctx).Model(&models.Bill{})

	if filter != nil {
		if filter.TenantID != uuid.Nil {
			query = query.Where("tenant_id = ?", filter.TenantID)
		}
		if filter.Status != "" {
			query = query.Where("status = ?", filter.Status)
		}
		if filter.StartDate != nil {
			query = query.Where("period_start >= ?", filter.StartDate)
		}
		if filter.EndDate != nil {
			query = query.Where("period_end <= ?", filter.EndDate)
		}
	}

	err := query.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	if filter != nil && filter.Limit > 0 {
		query = query.Offset(filter.Offset).Limit(filter.Limit)
	}

	err = query.Order("period_start desc").Find(&bills).Error
	return bills, total, err
}

func (s *BillingService) GetByID(ctx context.Context, id uuid.UUID) (*models.Bill, error) {
	var bill models.Bill
	err := s.db.WithContext(ctx).Where("id = ?", id).First(&bill).Error
	return &bill, err
}

func (s *BillingService) GetSummary(ctx context.Context, tenantID uuid.UUID, start, end time.Time) (*BillSummary, error) {
	var bills []models.Bill
	err := s.db.WithContext(ctx).
		Where("tenant_id = ? AND period_start >= ? AND period_end <= ?", tenantID, start, end).
		Find(&bills).Error
	if err != nil {
		return nil, err
	}

	summary := &BillSummary{
		TotalAmount:    0,
		ByResourceType: make(map[string]float64),
		ByStatus:       make(map[string]float64),
		ByRegion:       make(map[string]float64),
	}

	for _, bill := range bills {
		summary.TotalAmount += bill.TotalAmount
		summary.ByResourceType[bill.ResourceType] += bill.TotalAmount
		summary.ByStatus[bill.Status] += bill.TotalAmount
	}

	return summary, nil
}

type BillFilter struct {
	TenantID  uuid.UUID
	Status    string
	StartDate *time.Time
	EndDate   *time.Time
	ProjectID *uuid.UUID
	Offset    int
	Limit     int
}

type BillSummary struct {
	TotalAmount    float64            `json:"total_amount"`
	ByResourceType map[string]float64 `json:"by_resource_type"`
	ByStatus       map[string]float64 `json:"by_status"`
	ByRegion       map[string]float64 `json:"by_region"`
}

type MonitorService struct {
	db    *gorm.DB
	redis *repository.RedisClient
}

func NewMonitorService(db *gorm.DB, redis *repository.RedisClient) *MonitorService {
	return &MonitorService{db: db, redis: redis}
}

func (s *MonitorService) GetMetrics(ctx context.Context, req *MetricsQueryRequest) (*MetricsQueryResponse, error) {
	resp := &MetricsQueryResponse{
		Metrics: make([]MetricDataPoint, 0),
	}
	return resp, nil
}

func (s *MonitorService) QueryMetrics(ctx context.Context, query string, start, end time.Time) ([]map[string]interface{}, error) {
	return []map[string]interface{}{}, nil
}

func (s *MonitorService) ListAlerts(ctx context.Context, tenantID uuid.UUID) ([]models.Alert, error) {
	var alerts []models.Alert
	err := s.db.WithContext(ctx).Where("tenant_id = ?", tenantID).Find(&alerts).Error
	return alerts, err
}

func (s *MonitorService) CreateAlert(ctx context.Context, alert *models.Alert) error {
	return s.db.WithContext(ctx).Create(alert).Error
}

func (s *MonitorService) UpdateAlert(ctx context.Context, id uuid.UUID, updates map[string]interface{}) error {
	return s.db.WithContext(ctx).Model(&models.Alert{}).Where("id = ?", id).Updates(updates).Error
}

type MetricsQueryRequest struct {
	Metrics   []string    `json:"metrics"`
	NodeIDs   []uuid.UUID `json:"node_ids,omitempty"`
	Start     time.Time   `json:"start"`
	End       time.Time   `json:"end"`
	Interval  string      `json:"interval,omitempty"`
	Aggregate string      `json:"aggregate,omitempty"`
}

type MetricsQueryResponse struct {
	Metrics []MetricDataPoint `json:"metrics"`
}

type MetricDataPoint struct {
	Metric    string                 `json:"metric"`
	NodeID    uuid.UUID              `json:"node_id"`
	Timestamp time.Time              `json:"timestamp"`
	Value     float64                `json:"value"`
	Labels    map[string]string      `json:"labels,omitempty"`
}

type AuthService struct {
	db     *gorm.DB
	jwtCfg *JWTConfig
}

func NewAuthService(db *gorm.DB, jwtCfg *JWTConfig) *AuthService {
	return &AuthService{db: db, jwtCfg: jwtCfg}
}

func (s *AuthService) Register(ctx context.Context, user *models.User, password string) error {
	user.PasswordHash = hashPassword(password)
	user.Status = "active"
	user.Role = "user"
	return s.db.WithContext(ctx).Create(user).Error
}

func (s *AuthService) Login(ctx context.Context, email, password string) (*LoginResponse, error) {
	var user models.User
	err := s.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}

	if !verifyPassword(password, user.PasswordHash) {
		return nil, ErrInvalidCredentials
	}

	token, err := generateJWT(user.ID, user.Email, user.Role, s.jwtCfg.Secret, s.jwtCfg.Expiration)
	if err != nil {
		return nil, err
	}

	refreshToken, err := generateJWT(user.ID, user.Email, user.Role, s.jwtCfg.Secret, s.jwtCfg.RefreshExp)
	if err != nil {
		return nil, err
	}

	return &LoginResponse{
		AccessToken:  token,
		RefreshToken: refreshToken,
		ExpiresIn:    int(s.jwtCfg.Expiration.Seconds()),
		User:         &user,
	}, nil
}

type LoginResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresIn    int          `json:"expires_in"`
	User         *models.User `json:"user"`
}

type JWTConfig struct {
	Secret     string
	Expiration time.Duration
	RefreshExp time.Duration
}

func hashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return ""
	}
	return string(hash)
}

func verifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func generateJWT(userID uuid.UUID, email, role, secret string, expiration time.Duration) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":    userID.String(),
		"email":  email,
		"role":   role,
		"iat":    now.Unix(),
		"exp":    now.Add(expiration).Unix(),
		"jti":    uuid.New().String(),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

var ErrInvalidCredentials = &AuthError{Message: "invalid credentials"}

type AuthError struct {
	Message string
}

func (e *AuthError) Error() string {
	return e.Message
}
