package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/edgehub/edgehub/internal/models"
	"github.com/edgehub/edgehub/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type NodeHandler struct {
	svc *service.NodeService
}

func NewNodeHandler(svc *service.NodeService) *NodeHandler {
	return &NodeHandler{svc: svc}
}

func (h *NodeHandler) Register(c *gin.Context) {
	var node models.Node
	if err := c.ShouldBindJSON(&node); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.Register(c.Request.Context(), &node); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, node)
}

func (h *NodeHandler) List(c *gin.Context) {
	filter := &service.NodeFilter{
		ClusterID: parseUUID(c.Query("cluster_id")),
		Status:    c.Query("status"),
		Region:    c.Query("region"),
		HasGPU:    c.Query("has_gpu") == "true",
		Offset:    parseInt(c.Query("offset")),
		Limit:     parseInt(c.Query("limit")),
		SortBy:    c.Query("sort_by"),
		Order:     c.Query("order"),
	}

	nodes, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": nodes,
		"total": total,
		"offset": filter.Offset,
		"limit": filter.Limit,
	})
}

func (h *NodeHandler) Get(c *gin.Context) {
	id := parseUUID(c.Param("id"))
	node, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "node not found"})
		return
	}
	c.JSON(http.StatusOK, node)
}

func (h *NodeHandler) Update(c *gin.Context) {
	id := parseUUID(c.Param("id"))
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.Update(c.Request.Context(), id, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *NodeHandler) Delete(c *gin.Context) {
	id := parseUUID(c.Param("id"))
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *NodeHandler) Heartbeat(c *gin.Context) {
	id := parseUUID(c.Param("id"))
	var metrics service.NodeHeartbeatMetrics
	c.ShouldBindJSON(&metrics)

	if err := h.svc.Heartbeat(c.Request.Context(), id, &metrics); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *NodeHandler) GetMetrics(c *gin.Context) {
	id := parseUUID(c.Param("id"))
	start, _ := time.Parse(time.RFC3339, c.Query("start"))
	end, _ := time.Parse(time.RFC3339, c.Query("end"))
	if end.IsZero() {
		end = time.Now()
	}
	if start.IsZero() {
		start = end.Add(-24 * time.Hour)
	}

	metrics, err := h.svc.GetMetrics(c.Request.Context(), id, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"metrics": metrics})
}

type JobHandler struct {
	svc *service.JobService
}

func NewJobHandler(svc *service.JobService) *JobHandler {
	return &JobHandler{svc: svc}
}

func (h *JobHandler) Submit(c *gin.Context) {
	var job models.Job
	if err := c.ShouldBindJSON(&job); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	job.TenantID = parseUUID(c.GetHeader("X-Tenant-ID"))
	job.UserID = parseUUID(c.GetHeader("X-User-ID"))

	if err := h.svc.Submit(c.Request.Context(), &job); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, job)
}

func (h *JobHandler) List(c *gin.Context) {
	filter := &service.JobFilter{
		TenantID:  parseUUID(c.Query("tenant_id")),
		ClusterID: parseUUID(c.Query("cluster_id")),
		Status:    c.Query("status"),
		Type:     c.Query("type"),
		Queue:    c.Query("queue"),
		UserID:   parseUUID(c.Query("user_id")),
		Offset:   parseInt(c.Query("offset")),
		Limit:    parseInt(c.Query("limit")),
	}

	jobs, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": jobs,
		"total": total,
		"offset": filter.Offset,
		"limit": filter.Limit,
	})
}

func (h *JobHandler) Get(c *gin.Context) {
	id := parseUUID(c.Param("id"))
	job, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "job not found"})
		return
	}
	c.JSON(http.StatusOK, job)
}

func (h *JobHandler) Update(c *gin.Context) {
	id := parseUUID(c.Param("id"))
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.Update(c.Request.Context(), id, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

func (h *JobHandler) Delete(c *gin.Context) {
	id := parseUUID(c.Param("id"))
	if err := h.svc.Delete(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *JobHandler) Stop(c *gin.Context) {
	id := parseUUID(c.Param("id"))
	if err := h.svc.Stop(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "stopped"})
}

func (h *JobHandler) GetLogs(c *gin.Context) {
	id := parseUUID(c.Param("id"))
	logs, err := h.svc.GetLogs(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"logs": logs})
}

func (h *JobHandler) GetMetrics(c *gin.Context) {
	_ = parseUUID(c.Param("id"))
	c.JSON(http.StatusOK, gin.H{"metrics": []interface{}{}})
}

type MarketHandler struct {
	svc *service.MarketService
}

func NewMarketHandler(svc *service.MarketService) *MarketHandler {
	return &MarketHandler{svc: svc}
}

func (h *MarketHandler) ListOffers(c *gin.Context) {
	filter := &service.OfferFilter{
		Region:   c.Query("region"),
		MinCPU:   parseFloat(c.Query("min_cpu")),
		MinGPU:   parseInt(c.Query("min_gpu")),
		MaxPrice: parseFloat(c.Query("max_price")),
		Limit:    parseInt(c.Query("limit")),
	}

	offers, total, err := h.svc.ListOffers(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": offers,
		"total": total,
	})
}

func (h *MarketHandler) CreateOffer(c *gin.Context) {
	var offer models.MarketOffer
	if err := c.ShouldBindJSON(&offer); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	offer.ProviderID = parseUUID(c.GetHeader("X-User-ID"))

	if err := h.svc.CreateOffer(c.Request.Context(), &offer); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, offer)
}

func (h *MarketHandler) GetOffer(c *gin.Context) {
	id := parseUUID(c.Param("id"))
	offer, err := h.svc.GetOffer(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "offer not found"})
		return
	}
	c.JSON(http.StatusOK, offer)
}

func (h *MarketHandler) DeleteOffer(c *gin.Context) {
	id := parseUUID(c.Param("id"))
	if err := h.svc.DeleteOffer(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "deleted"})
}

func (h *MarketHandler) CreateOrder(c *gin.Context) {
	var order models.MarketOrder
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	order.ConsumerID = parseUUID(c.GetHeader("X-User-ID"))
	order.TenantID = parseUUID(c.GetHeader("X-Tenant-ID"))

	if err := h.svc.CreateOrder(c.Request.Context(), &order); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, order)
}

func (h *MarketHandler) GetOrder(c *gin.Context) {
	id := parseUUID(c.Param("id"))
	order, err := h.svc.GetOrder(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found"})
		return
	}
	c.JSON(http.StatusOK, order)
}

func (h *MarketHandler) GetPrices(c *gin.Context) {
	filter := &service.PriceFilter{
		Region:  c.Query("region"),
		GPUType: c.Query("gpu_type"),
	}

	prices, err := h.svc.GetPrices(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, prices)
}

func (h *MarketHandler) GetRecommendation(c *gin.Context) {
	req := &service.RecommendationRequest{
		Region:      c.Query("region"),
		MinCPU:      parseFloat(c.Query("min_cpu")),
		MinGPU:      parseInt(c.Query("min_gpu")),
		BillingType: c.Query("billing_type"),
		MaxPrice:    parseFloat(c.Query("max_price")),
	}

	rec, err := h.svc.GetRecommendation(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, rec)
}

type BillingHandler struct {
	svc *service.BillingService
}

func NewBillingHandler(svc *service.BillingService) *BillingHandler {
	return &BillingHandler{svc: svc}
}

func (h *BillingHandler) List(c *gin.Context) {
	tenantID := parseUUID(c.Query("tenant_id"))
	filter := &service.BillFilter{
		TenantID: tenantID,
		Status:   c.Query("status"),
		Offset:   parseInt(c.Query("offset")),
		Limit:    parseInt(c.Query("limit")),
	}

	bills, total, err := h.svc.List(c.Request.Context(), filter)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"items": bills,
		"total": total,
	})
}

func (h *BillingHandler) Get(c *gin.Context) {
	id := parseUUID(c.Param("id"))
	bill, err := h.svc.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "bill not found"})
		return
	}
	c.JSON(http.StatusOK, bill)
}

func (h *BillingHandler) GetSummary(c *gin.Context) {
	tenantID := parseUUID(c.Query("tenant_id"))
	start, _ := time.Parse(time.RFC3339, c.Query("start"))
	end, _ := time.Parse(time.RFC3339, c.Query("end"))
	if end.IsZero() {
		end = time.Now()
	}
	if start.IsZero() {
		start = end.Add(-30 * 24 * time.Hour)
	}

	summary, err := h.svc.GetSummary(c.Request.Context(), tenantID, start, end)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, summary)
}

func (h *BillingHandler) Export(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "export not implemented"})
}

type MonitorHandler struct {
	svc *service.MonitorService
}

func NewMonitorHandler(svc *service.MonitorService) *MonitorHandler {
	return &MonitorHandler{svc: svc}
}

func (h *MonitorHandler) GetMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"metrics": []interface{}{}})
}

func (h *MonitorHandler) QueryMetrics(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"results": []interface{}{}})
}

func (h *MonitorHandler) ListAlerts(c *gin.Context) {
	tenantID := parseUUID(c.Query("tenant_id"))
	alerts, err := h.svc.ListAlerts(c.Request.Context(), tenantID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": alerts})
}

func (h *MonitorHandler) CreateAlert(c *gin.Context) {
	var alert models.Alert
	if err := c.ShouldBindJSON(&alert); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.CreateAlert(c.Request.Context(), &alert); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, alert)
}

func (h *MonitorHandler) UpdateAlert(c *gin.Context) {
	id := parseUUID(c.Param("id"))
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.svc.UpdateAlert(c.Request.Context(), id, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "updated"})
}

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required,min=8"`
		Name     string `json:"name" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user := &models.User{Email: req.Email, Name: req.Name}
	if err := h.svc.Register(c.Request.Context(), user, req.Password); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, user)
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resp, err := h.svc.Login(c.Request.Context(), req.Email, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, resp)
}

func (h *AuthHandler) Refresh(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"token": ""})
}

func (h *AuthHandler) Logout(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"message": "logged out"})
}

func ListClusters(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"items": []interface{}{},
		"total": 0,
	})
}

func CreateCluster(c *gin.Context) {
	c.JSON(http.StatusCreated, gin.H{"message": "cluster created"})
}

func GetCluster(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"id": id})
}

func DeleteCluster(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{"message": "cluster deleted", "id": id})
}

func parseUUID(s string) uuid.UUID {
	if s == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}

func parseInt(s string) int {
	if s == "" {
		return 0
	}
	i, _ := strconv.Atoi(s)
	return i
}

func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
