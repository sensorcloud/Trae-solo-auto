package federation

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/klog/v2"
)

type ClusterStatus string

const (
	ClusterStatusReady    ClusterStatus = "Ready"
	ClusterStatusOffline   ClusterStatus = "Offline"
	ClusterStatusDegraded  ClusterStatus = "Degraded"
	ClusterStatusDisabled  ClusterStatus = "Disabled"
)

type Cluster struct {
	ID           uuid.UUID       `json:"id"`
	Name         string         `json:"name"`
	Region       string         `json:"region"`
	Zone         string         `json:"zone"`
	Provider     string         `json:"provider"`
	Status       ClusterStatus   `json:"status"`
	APIEndpoint  string         `json:"api_endpoint"`
	KubeConfig   []byte         `json:"-"`
	Capacity     ClusterCapacity `json:"capacity"`
	Conditions   []metav1.Condition `json:"conditions"`
	Labels       map[string]string `json:"labels"`
	Annotations   map[string]string `json:"annotations"`
	LastHeartbeat time.Time     `json:"last_heartbeat"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
}

type ClusterCapacity struct {
	CPU         float64 `json:"cpu"`
	Memory      int64   `json:"memory"`
	GPU         int     `json:"gpu"`
	Storage     int64   `json:"storage"`
	Pods        int     `json:"pods"`
	Nodes       int     `json:"nodes"`
}

type ClusterRegistry struct {
	clusters map[string]*Cluster
	mu      sync.RWMutex
}

func NewClusterRegistry() *ClusterRegistry {
	return &ClusterRegistry{
		clusters: make(map[string]*Cluster),
	}
}

func (r *ClusterRegistry) Register(cluster *Cluster) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.clusters[cluster.Name]; exists {
		return fmt.Errorf("cluster %s already registered", cluster.Name)
	}

	if cluster.ID == uuid.Nil {
		cluster.ID = uuid.New()
	}
	cluster.CreatedAt = time.Now()
	cluster.UpdatedAt = time.Now()

	r.clusters[cluster.Name] = cluster
	klog.Infof("Cluster registered: %s in region %s", cluster.Name, cluster.Region)
	return nil
}

func (r *ClusterRegistry) Unregister(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.clusters[name]; !exists {
		return fmt.Errorf("cluster %s not found", name)
	}

	delete(r.clusters, name)
	klog.Infof("Cluster unregistered: %s", name)
	return nil
}

func (r *ClusterRegistry) Get(name string) (*Cluster, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cluster, exists := r.clusters[name]
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", name)
	}

	return cluster, nil
}

func (r *ClusterRegistry) List() []*Cluster {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Cluster, 0, len(r.clusters))
	for _, cluster := range r.clusters {
		result = append(result, cluster)
	}
	return result
}

func (r *ClusterRegistry) ListByRegion(region string) []*Cluster {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*Cluster, 0)
	for _, cluster := range r.clusters {
		if cluster.Region == region && cluster.Status == ClusterStatusReady {
			result = append(result, cluster)
		}
	}
	return result
}

func (r *ClusterRegistry) UpdateStatus(name string, status ClusterStatus) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cluster, exists := r.clusters[name]
	if !exists {
		return fmt.Errorf("cluster %s not found", name)
	}

	cluster.Status = status
	cluster.UpdatedAt = time.Now()
	klog.Infof("Cluster %s status updated to %s", name, status)
	return nil
}

func (r *ClusterRegistry) UpdateHeartbeat(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	cluster, exists := r.clusters[name]
	if !exists {
		return fmt.Errorf("cluster %s not found", name)
	}

	cluster.LastHeartbeat = time.Now()
	cluster.Status = ClusterStatusReady
	return nil
}

type PropagationPolicy struct {
	ID          uuid.UUID            `json:"id"`
	Name        string               `json:"name"`
	Propagate   bool                 `json:"propagate"`
	ClusterAffinity *ClusterAffinity `json:"cluster_affinity,omitempty"`
	Replicas    int32                `json:"replicas"`
	ClusterNames []string            `json:"cluster_names"`
	CreatedAt   time.Time            `json:"created_at"`
}

type ClusterAffinity struct {
	ClusterNames []string                   `json:"cluster_names,omitempty"`
	Region       string                     `json:"region,omitempty"`
	Zone         string                     `json:"zone,omitempty"`
	Provider     string                     `json:"provider,omitempty"`
	LabelSelector *metav1.LabelSelector     `json:"label_selector,omitempty"`
}

type OverridePolicy struct {
	ID           uuid.UUID              `json:"id"`
	Name         string                 `json:"name"`
	TargetClusters []string             `json:"target_clusters"`
	Overrides    []Override              `json:"overrides"`
	CreatedAt    time.Time              `json:"created_at"`
}

type Override struct {
	Path      string            `json:"path"`
	Operator  string            `json:"operator"`
	Value     interface{}       `json:"value"`
}

type WorkloadDistribution struct {
	PolicyName    string
	TotalReplicas int32
	Allocations   map[string]int32
}

func NewWorkloadDistribution(policyName string, replicas int32, clusters []*Cluster) *WorkloadDistribution {
	wd := &WorkloadDistribution{
		PolicyName:    policyName,
		TotalReplicas: replicas,
		Allocations:   make(map[string]int32),
	}

	if len(clusters) == 0 {
		return wd
	}

	perCluster := replicas / int32(len(clusters))
	remainder := replicas % int32(len(clusters))

	for i, cluster := range clusters {
		alloc := perCluster
		if int32(i) < remainder {
			alloc++
		}
		wd.Allocations[cluster.Name] = alloc
	}

	return wd
}

func (wd *WorkloadDistribution) GetAllocation(clusterName string) int32 {
	if alloc, ok := wd.Allocations[clusterName]; ok {
		return alloc
	}
	return 0
}

type FederatedResource struct {
	Kind       string
	APIVersion string
	Metadata   FederatedMetadata
	Spec       interface{}
}

type FederatedMetadata struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

type MultiClusterService struct {
	registry *ClusterRegistry
	services map[string]*FederatedService
	mu       sync.RWMutex
}

type FederatedService struct {
	Name        string            `json:"name"`
	Namespace   string            `json:"namespace"`
	Clusters    []string          `json:"clusters"`
	Ports       []ServicePort     `json:"ports"`
	Selector    map[string]string `json:"selector"`
	HealthCheck HealthCheckConfig `json:"health_check"`
}

type ServicePort struct {
	Name     string `json:"name"`
	Port     int32  `json:"port"`
	TargetPort int32 `json:"target_port"`
	Protocol string `json:"protocol"`
}

type HealthCheckConfig struct {
	Enabled         bool          `json:"enabled"`
	Path            string        `json:"path"`
	Interval        time.Duration `json:"interval"`
	Timeout         time.Duration `json:"timeout"`
	SuccessThreshold int         `json:"success_threshold"`
	FailureThreshold int         `json:"failure_threshold"`
}

func NewMultiClusterService(registry *ClusterRegistry) *MultiClusterService {
	return &MultiClusterService{
		registry: registry,
		services: make(map[string]*FederatedService),
	}
}

func (mcs *MultiClusterService) CreateService(svc *FederatedService) error {
	mcs.mu.Lock()
	defer mcs.mu.Unlock()

	key := fmt.Sprintf("%s/%s", svc.Namespace, svc.Name)
	if _, exists := mcs.services[key]; exists {
		return fmt.Errorf("service %s already exists", key)
	}

	mcs.services[key] = svc
	klog.Infof("Federated service created: %s across clusters %v", key, svc.Clusters)
	return nil
}

func (mcs *MultiClusterService) GetService(namespace, name string) (*FederatedService, error) {
	mcs.mu.RLock()
	defer mcs.mu.RUnlock()

	key := fmt.Sprintf("%s/%s", namespace, name)
	svc, exists := mcs.services[key]
	if !exists {
		return nil, fmt.Errorf("service %s not found", key)
	}

	return svc, nil
}

func (mcs *MultiClusterService) ListServices() []*FederatedService {
	mcs.mu.RLock()
	defer mcs.mu.RUnlock()

	result := make([]*FederatedService, 0, len(mcs.services))
	for _, svc := range mcs.services {
		result = append(result, svc)
	}
	return result
}

func (mcs *MultiClusterService) DeleteService(namespace, name string) error {
	mcs.mu.Lock()
	defer mcs.mu.Unlock()

	key := fmt.Sprintf("%s/%s", namespace, name)
	if _, exists := mcs.services[key]; !exists {
		return fmt.Errorf("service %s not found", key)
	}

	delete(mcs.services, key)
	klog.Infof("Federated service deleted: %s", key)
	return nil
}

func (mcs *MultiClusterService) GetHealthyClusters(serviceName string) []string {
	mcs.mu.RLock()
	defer mcs.mu.RUnlock()

	svc, exists := mcs.services[serviceName]
	if !exists {
		return nil
	}

	healthyClusters := make([]string, 0)
	for _, clusterName := range svc.Clusters {
		cluster, err := mcs.registry.Get(clusterName)
		if err != nil || cluster.Status != ClusterStatusReady {
			continue
		}

		if time.Since(cluster.LastHeartbeat) > 2*time.Minute {
			continue
		}

		healthyClusters = append(healthyClusters, clusterName)
	}

	return healthyClusters
}

type ResourceSync struct {
	Cluster   string
	Kind      string
	Namespace string
	Name      string
	Status    string
	SyncedAt  time.Time
}

type ClusterHealthChecker struct {
	registry      *ClusterRegistry
	healthChecker HealthChecker
	interval      time.Duration
	stopCh        chan struct{}
}

type HealthChecker interface {
	CheckClusterHealth(ctx context.Context, cluster *Cluster) error
}

type DefaultHealthChecker struct{}

func (h *DefaultHealthChecker) CheckClusterHealth(ctx context.Context, cluster *Cluster) error {
	cluster.Status = ClusterStatusReady
	cluster.LastHeartbeat = time.Now()
	return nil
}

func NewClusterHealthChecker(registry *ClusterRegistry, interval time.Duration) *ClusterHealthChecker {
	return &ClusterHealthChecker{
		registry:      registry,
		healthChecker: &DefaultHealthChecker{},
		interval:      interval,
		stopCh:        make(chan struct{}),
	}
}

func (h *ClusterHealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(h.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-h.stopCh:
			return
		case <-ticker.C:
			h.checkAllClusters(ctx)
		}
	}
}

func (h *ClusterHealthChecker) Stop() {
	close(h.stopCh)
}

func (h *ClusterHealthChecker) checkAllClusters(ctx context.Context) {
	clusters := h.registry.List()

	for _, cluster := range clusters {
		if err := h.healthChecker.CheckClusterHealth(ctx, cluster); err != nil {
			h.registry.UpdateStatus(cluster.Name, ClusterStatusDegraded)
			klog.Warningf("Cluster %s health check failed: %v", cluster.Name, err)
		} else {
			h.registry.UpdateStatus(cluster.Name, ClusterStatusReady)
		}
	}
}

type scheme struct {
	*runtime.Scheme
}

func newScheme() *scheme {
	return &scheme{runtime.NewScheme()}
}

func (s *scheme) AllKnownTypes() []runtime.Object {
	return nil
}
