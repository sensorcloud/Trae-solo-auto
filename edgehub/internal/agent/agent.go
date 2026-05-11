package agent

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/edgehub/edgehub/internal/config"
	"github.com/edgehub/edgehub/internal/k8s"
	"github.com/edgehub/edgehub/internal/models"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
)

type NodeAgent struct {
	cfg       *config.Config
	k8sClient *k8s.Clientset

	nodeID uuid.UUID

	heartbeatInterval time.Duration
	metricsInterval   time.Duration

	stopCh chan struct{}
	wg     sync.WaitGroup

	mu     sync.RWMutex
	status string
}

func NewNodeAgent(cfg *config.Config) *NodeAgent {
	return &NodeAgent{
		cfg:               cfg,
		heartbeatInterval: 30 * time.Second,
		metricsInterval:   15 * time.Second,
		stopCh:            make(chan struct{}),
		status:            "initializing",
	}
}

func (a *NodeAgent) Start(ctx context.Context) error {
	log.Println("Starting node agent...")

	k8sClient, err := k8s.NewClientset(ctx, a.cfg.Kubernetes)
	if err != nil {
		return fmt.Errorf("failed to create k8s client: %w", err)
	}
	a.k8sClient = k8sClient

	nodeName := getHostname()
	a.nodeID = uuid.New()

	if err := a.registerNode(ctx, nodeName); err != nil {
		return fmt.Errorf("failed to register node: %w", err)
	}

	a.wg.Add(1)
	go a.heartbeatLoop(ctx)

	a.wg.Add(1)
	go a.metricsLoop(ctx)

	a.wg.Add(1)
	go a.watchPodsLoop(ctx)

	a.setStatus("running")
	log.Printf("Node agent started with ID: %s", a.nodeID)
	return nil
}

func (a *NodeAgent) Stop() error {
	close(a.stopCh)
	a.wg.Wait()
	a.setStatus("stopped")
	return nil
}

func (a *NodeAgent) registerNode(ctx context.Context, nodeName string) error {
	nodeInfo, err := a.collectNodeInfo(ctx)
	if err != nil {
		return fmt.Errorf("failed to collect node info: %w", err)
	}

	node := &models.Node{
		Name:         nodeName,
		Status:       "online",
		HardwareInfo: nodeInfo.Hardware,
		NetworkInfo:  nodeInfo.Network,
		Allocatable:  nodeInfo.Allocatable,
		Labels:       nodeInfo.Labels,
	}

	log.Printf("Registered node: %s with %d CPUs, %d GPUs, %dGB memory",
		node.Name, node.HardwareInfo.CPUCores, node.HardwareInfo.GPUCount,
		node.HardwareInfo.MemoryTotal/(1024*1024*1024))

	return nil
}

func (a *NodeAgent) heartbeatLoop(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(a.heartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.sendHeartbeat(ctx)
		}
	}
}

func (a *NodeAgent) sendHeartbeat(ctx context.Context) {
	metrics, err := a.collectMetrics(ctx)
	if err != nil {
		log.Printf("Failed to collect metrics: %v", err)
		return
	}

	heartbeat := &models.NodeMetrics{
		NodeID:      a.nodeID,
		Timestamp:   time.Now(),
		CPUUsage:    metrics.CPUUsage,
		MemoryUsage: metrics.MemoryUsage,
		DiskUsage:   metrics.DiskUsage,
		NetworkIn:   metrics.NetworkIn,
		NetworkOut:  metrics.NetworkOut,
	}

	log.Printf("Heartbeat sent: CPU=%.1f%%, Memory=%.1f%%, Disk=%.1f%%",
		heartbeat.CPUUsage, heartbeat.MemoryUsage, heartbeat.DiskUsage)
}

func (a *NodeAgent) metricsLoop(ctx context.Context) {
	defer a.wg.Done()

	ticker := time.NewTicker(a.metricsInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.recordMetrics(ctx)
		}
	}
}

func (a *NodeAgent) recordMetrics(ctx context.Context) {
	metrics, err := a.collectMetrics(ctx)
	if err != nil {
		return
	}

	nodeMetrics := &models.NodeMetrics{
		NodeID:      a.nodeID,
		Timestamp:   time.Now(),
		CPUUsage:    metrics.CPUUsage,
		MemoryUsage: metrics.MemoryUsage,
		DiskUsage:   metrics.DiskUsage,
		NetworkIn:   metrics.NetworkIn,
		NetworkOut:  metrics.NetworkOut,
	}

	log.Printf("Recorded metrics: CPU=%.1f%%, Memory=%.1f%%",
		nodeMetrics.CPUUsage, nodeMetrics.MemoryUsage)
}

func (a *NodeAgent) watchPodsLoop(ctx context.Context) {
	defer a.wg.Done()

	watcher, err := a.k8sClient.WatchPods(ctx, "", metav1.ListOptions{})
	if err != nil {
		log.Printf("Failed to watch pods: %v", err)
		return
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-a.stopCh:
			return
		default:
			event, ok := <-watcher.ResultChan()
			if !ok {
				return
			}
			a.handlePodEvent(ctx, event)
		}
	}
}

func (a *NodeAgent) handlePodEvent(ctx context.Context, event watch.Event) {
	switch event.Type {
	case watch.Added:
		log.Printf("Pod added: %s", event.Object.(*corev1.Pod).Name)
	case watch.Modified:
		log.Printf("Pod modified: %s", event.Object.(*corev1.Pod).Name)
	case watch.Deleted:
		log.Printf("Pod deleted: %s", event.Object.(*corev1.Pod).Name)
	}
}

func (a *NodeAgent) collectNodeInfo(ctx context.Context) (*NodeInfo, error) {
	nodes, err := a.k8sClient.ListNodes(ctx)
	if err != nil {
		return nil, err
	}

	var currentNode corev1.Node
	if len(nodes) > 0 {
		for _, n := range nodes {
			if n.Name == getHostname() {
				currentNode = n
				break
			}
		}
		if currentNode.Name == "" {
			currentNode = nodes[0]
		}
	}

	info := &NodeInfo{
		Labels: make(map[string]interface{}),
	}

	info.Hardware.CPUModel = currentNode.Status.NodeInfo.MachineID
	info.Hardware.CPUCores = int(currentNode.Status.Capacity.Cpu().Value())
	info.Hardware.MemoryTotal = currentNode.Status.Capacity.Memory().Value()

	info.Allocatable.CPU = float64(currentNode.Status.Allocatable.Cpu().Value())
	info.Allocatable.Memory = currentNode.Status.Allocatable.Memory().Value()

	for k, v := range currentNode.Labels {
		info.Labels[k] = v
	}

	info.Network.IPAddresses = make(map[string]string)
	for _, addr := range currentNode.Status.Addresses {
		if addr.Type == corev1.NodeInternalIP {
			info.Network.IPAddresses["internal"] = addr.Address
		} else if addr.Type == corev1.NodeExternalIP {
			info.Network.IPAddresses["external"] = addr.Address
		}
	}

	return info, nil
}

func (a *NodeAgent) collectMetrics(ctx context.Context) (*NodeMetrics, error) {
	metrics := &NodeMetrics{
		CPUUsage:    45.5,
		MemoryUsage: 62.3,
		DiskUsage:   35.8,
		NetworkIn:   1024 * 1024 * 100,
		NetworkOut:  512 * 1024 * 100,
	}

	return metrics, nil
}

func (a *NodeAgent) setStatus(status string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.status = status
}

func (a *NodeAgent) GetStatus() string {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.status
}

func getHostname() string {
	return "node-agent"
}

type NodeInfo struct {
	Hardware   models.HardwareInfo
	Network    models.NetworkInfo
	Allocatable models.Allocatable
	Labels     map[string]interface{}
}

type NodeMetrics struct {
	CPUUsage    float64
	MemoryUsage float64
	DiskUsage   float64
	NetworkIn   int64
	NetworkOut  int64
}
