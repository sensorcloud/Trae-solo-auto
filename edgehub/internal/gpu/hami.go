package gpu

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

type HAMiManager struct {
	clientset *kubernetes.Clientset
	config    *HAMiConfig
}

type HAMiConfig struct {
	Enabled          bool
	GPUDriverVersion string
	DefaultMemPerGPU int64
	DefaultCoresPerGPU int32
}

type GPUResource struct {
	GPUCount        int64
	MemoryPerGPU    int64
	CoresPerGPU     int32
	TotalGPUCount   int64
	AllocatedGPUCount int64
}

type GPUPod struct {
	Name      string
	Namespace string
	GPUCount  int64
	GPUCores  int32
	GPUMemory int64
	Status    GPUPodStatus
}

type GPUPodStatus string

const (
	GPUPodPending   GPUPodStatus = "Pending"
	GPUPodRunning   GPUPodStatus = "Running"
	GPUPodFailed    GPUPodStatus = "Failed"
	GPUPodSucceeded GPUPodStatus = "Succeeded"
)

func NewHAMiManager(clientset *kubernetes.Clientset, config *HAMiConfig) *HAMiManager {
	if config == nil {
		config = &HAMiConfig{
			Enabled:           true,
			GPUDriverVersion:  "535.54",
			DefaultMemPerGPU:  8192,
			DefaultCoresPerGPU: 100,
		}
	}
	return &HAMiManager{
		clientset: clientset,
		config:    config,
	}
}

func (hm *HAMiManager) GetClusterGPUInfo(ctx context.Context) (*GPUResource, error) {
	nodes, err := hm.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	var totalGPU int64
	var allocatedGPU int64

	for _, node := range nodes.Items {
		gpuQty, ok := node.Status.Capacity[ResourceNvidiaGPU]
		if !ok {
			continue
		}
		totalGPU += gpuQty.Value()

		allocQty, ok := node.Status.Allocatable[ResourceNvidiaGPU]
		if !ok {
			allocQty = gpuQty
		}
		allocatedGPU += gpuQty.Value() - allocQty.Value()
	}

	return &GPUResource{
		TotalGPUCount:    totalGPU,
		AllocatedGPUCount: allocatedGPU,
	}, nil
}

func (hm *HAMiManager) CreateGPUJob(ctx context.Context, job *GPUJobSpec) (*GPUJob, error) {
	pod := hm.buildGPUTaskPod(job)
	created, err := hm.clientset.CoreV1().Pods(job.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create GPU job pod: %w", err)
	}

	return &GPUJob{
		Name:      created.Name,
		Namespace: created.Namespace,
		Status:    string(created.Status.Phase),
	}, nil
}

func (hm *HAMiManager) GetGPUJobStatus(ctx context.Context, namespace, jobName string) (*GPUJob, error) {
	pod, err := hm.clientset.CoreV1().Pods(namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get GPU job pod: %w", err)
	}

	return &GPUJob{
		Name:      pod.Name,
		Namespace: pod.Namespace,
		Status:    string(pod.Status.Phase),
	}, nil
}

func (hm *HAMiManager) DeleteGPUJob(ctx context.Context, namespace, jobName string) error {
	return hm.clientset.CoreV1().Pods(namespace).Delete(ctx, jobName, metav1.DeleteOptions{})
}

func (hm *HAMiManager) ListGPUNodes(ctx context.Context) ([]GPUNode, error) {
	nodes, err := hm.clientset.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to list nodes: %w", err)
	}

	var gpuNodes []GPUNode
	for _, node := range nodes.Items {
		gpuQty, ok := node.Status.Capacity[ResourceNvidiaGPU]
		if !ok || gpuQty.Value() == 0 {
			continue
		}

		gpuMem := node.Status.Capacity[ResourceGPUMemory]
		gpuCores := node.Status.Capacity[ResourceGPUCores]

		gpuNodes = append(gpuNodes, GPUNode{
			Name:           node.Name,
			GPUCount:       gpuQty.Value(),
			MemoryPerGPU:   gpuMem.Value(),
			CoresPerGPU:    int32(gpuCores.Value()),
			AvailableGPUCount: func() int64 {
				allocatable, ok := node.Status.Allocatable[ResourceNvidiaGPU]
				if !ok {
					return gpuQty.Value()
				}
				return allocatable.Value()
			}(),
		})
	}

	return gpuNodes, nil
}

func (hm *HAMiManager) ScheduleGPUPod(ctx context.Context, pod *corev1.Pod) error {
	pod.Spec.SchedulerName = "hamischeduler"
	_, err := hm.clientset.CoreV1().Pods(pod.Namespace).Create(ctx, pod, metav1.CreateOptions{})
	return err
}

func (hm *HAMiManager) buildGPUTaskPod(job *GPUJobSpec) *corev1.Pod {
	gpuLimit := corev1.ResourceList{
		ResourceNvidiaGPU: *resource.NewQuantity(job.GPUCount, resource.DecimalSI),
	}

	if job.MemoryPerGPU > 0 {
		gpuLimit[ResourceGPUMemory] = *resource.NewQuantity(job.MemoryPerGPU, resource.DecimalSI)
	}
	if job.CoresPerGPU > 0 {
		gpuLimit[ResourceGPUCores] = *resource.NewQuantity(int64(job.CoresPerGPU), resource.DecimalSI)
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      job.Name,
			Namespace: job.Namespace,
			Labels: map[string]string{
				"app":                "gpu-task",
				"edgehub.io/gpu-job": job.Name,
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy:  corev1.RestartPolicyNever,
			SchedulerName:  "hamischeduler",
			Containers: []corev1.Container{
				{
					Name:  "gpu-task",
					Image: job.Image,
					Command: func() []string {
						if len(job.Command) > 0 {
							return job.Command
						}
						return []string{"sleep", "3600"}
					}(),
					Resources: corev1.ResourceRequirements{
						Limits:   gpuLimit,
						Requests: gpuLimit,
					},
				},
			},
		},
	}
}

type GPUJobSpec struct {
	Name           string
	Namespace      string
	Image          string
	Command        []string
	GPUCount       int64
	MemoryPerGPU   int64
	CoresPerGPU    int32
	Priority       int32
}

type GPUJob struct {
	Name      string
	Namespace string
	Status    string
}

type GPUNode struct {
	Name               string
	GPUCount           int64
	MemoryPerGPU       int64
	CoresPerGPU        int32
	AvailableGPUCount  int64
}

const (
	ResourceNvidiaGPU  corev1.ResourceName = "nvidia.com/gpu"
	ResourceGPUMemory  corev1.ResourceName = "nvidia.com/gpu_memory"
	ResourceGPUCores   corev1.ResourceName = "nvidia.com/gpu_cores"
)

func RegisterHAMiResources() {
	q := resource.Quantity{}
	_ = q.String()
}

func (hm *HAMiManager) GetConfig() *HAMiConfig {
	return hm.config
}

func (hm *HAMiManager) UpdateConfig(config *HAMiConfig) {
	hm.config = config
}
