package scheduler

import (
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"
)

type QueuePriority int

const (
	PriorityHigh   QueuePriority = 100
	PriorityMedium QueuePriority = 50
	PriorityLow    QueuePriority = 10
)

type Queue struct {
	Name       string         `json:"name"`
	Priority   QueuePriority `json:"priority"`
	Weight     int           `json:"weight"`
	MaxLen     int           `json:"max_len"`
	MaxTime    time.Duration `json:"max_time"`
	IsFair     bool          `json:"is_fair"`
	Jobs       []*QueuedJob  `json:"jobs"`
	mu         sync.Mutex
}

type QueuedJob struct {
	Job           *Job
	QueuedAt      time.Time
	Priority      int
	Attempts      int
	ReadyInCluster []string
}

type Job struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Namespace string    `json:"namespace"`
	Type      JobType   `json:"type"`
	Status    JobStatus `json:"status"`
	Spec      JobSpec   `json:"spec"`
	Queue     string    `json:"queue"`
	Priority  int       `json:"priority"`
	CreatedAt time.Time `json:"created_at"`
}

type JobType string

const (
	JobTypePod       JobType = "pod"
	JobTypeBatch     JobType = "batch"
	JobTypeTraining  JobType = "training"
	JobTypeInference JobType = "inference"
	JobTypeRay       JobType = "ray"
	JobTypeSpark     JobType = "spark"
)

type JobStatus string

const (
	JobStatusPending    JobStatus = "pending"
	JobStatusQueued     JobStatus = "queued"
	JobStatusScheduling JobStatus = "scheduling"
	JobStatusRunning    JobStatus = "running"
	JobStatusSucceeded  JobStatus = "succeeded"
	JobStatusFailed     JobStatus = "failed"
	JobStatusCancelled  JobStatus = "cancelled"
)

type JobSpec struct {
	Image         string            `json:"image"`
	Command       []string          `json:"command"`
	Args          []string          `json:"args"`
	Env           []corev1.EnvVar   `json:"env"`
	CPU           string            `json:"cpu"`
	Memory        string            `json:"memory"`
	GPU           int64             `json:"gpu"`
	GPUMemory     int64             `json:"gpu_memory"`
	GPUCores      int32             `json:"gpu_cores"`
	Volumes       []VolumeSpec      `json:"volumes"`
	RestartPolicy string            `json:"restart_policy"`
	NodeSelector  map[string]string `json:"node_selector"`
	Tolerations   []corev1.Toleration `json:"tolerations"`
}

type VolumeSpec struct {
	Name      string `json:"name"`
	MountPath string `json:"mount_path"`
	Type      string `json:"type"`
	Size      string `json:"size"`
}

type QueueManager struct {
	queues map[string]*Queue
	mu     sync.Mutex
}

func NewQueueManager() *QueueManager {
	return &QueueManager{
		queues: make(map[string]*Queue),
	}
}

func (qm *QueueManager) CreateQueue(name string, priority QueuePriority, maxLen int, isFair bool) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if _, exists := qm.queues[name]; exists {
		return fmt.Errorf("queue %s already exists", name)
	}

	qm.queues[name] = &Queue{
		Name:     name,
		Priority: priority,
		MaxLen:   maxLen,
		IsFair:   isFair,
		Jobs:     make([]*QueuedJob, 0),
	}

	klog.Infof("Queue %s created with priority %d", name, priority)
	return nil
}

func (qm *QueueManager) DeleteQueue(name string) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	if _, exists := qm.queues[name]; !exists {
		return fmt.Errorf("queue %s not found", name)
	}

	delete(qm.queues, name)
	klog.Infof("Queue %s deleted", name)
	return nil
}

func (qm *QueueManager) GetQueue(name string) (*Queue, error) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	queue, exists := qm.queues[name]
	if !exists {
		return nil, fmt.Errorf("queue %s not found", name)
	}

	return queue, nil
}

func (qm *QueueManager) ListQueues() []*Queue {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	queues := make([]*Queue, 0, len(qm.queues))
	for _, q := range qm.queues {
		queues = append(queues, q)
	}

	return queues
}

func (qm *QueueManager) Enqueue(job *Job) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	queue, exists := qm.queues[job.Queue]
	if !exists {
		return fmt.Errorf("queue %s not found", job.Queue)
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	if queue.MaxLen > 0 && len(queue.Jobs) >= queue.MaxLen {
		return fmt.Errorf("queue %s is full", job.Queue)
	}

	queuedJob := &QueuedJob{
		Job:      job,
		QueuedAt: time.Now(),
		Priority: job.Priority,
		Attempts: 0,
	}

	queue.Jobs = append(queue.Jobs, queuedJob)
	klog.Infof("Job %s enqueued to queue %s", job.ID, queue.Name)
	return nil
}

func (qm *QueueManager) Dequeue() (*QueuedJob, string, error) {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	sortedQueues := qm.getSortedQueues()
	if len(sortedQueues) == 0 {
		return nil, "", fmt.Errorf("no queues available")
	}

	for _, queue := range sortedQueues {
		queue.mu.Lock()
		if len(queue.Jobs) == 0 {
			queue.mu.Unlock()
			continue
		}

		job := qm.selectNextJob(queue)
		if job == nil {
			queue.mu.Unlock()
			continue
		}

		queue.mu.Unlock()
		return job, queue.Name, nil
	}

	return nil, "", fmt.Errorf("no jobs available in any queue")
}

func (qm *QueueManager) getSortedQueues() []*Queue {
	queues := make([]*Queue, 0, len(qm.queues))
	for _, q := range qm.queues {
		queues = append(queues, q)
	}

	for i := 0; i < len(queues)-1; i++ {
		for j := i + 1; j < len(queues); j++ {
			if queues[i].Priority < queues[j].Priority {
				queues[i], queues[j] = queues[j], queues[i]
			}
		}
	}

	return queues
}

func (qm *QueueManager) selectNextJob(queue *Queue) *QueuedJob {
	if !queue.IsFair || len(queue.Jobs) == 0 {
		if len(queue.Jobs) == 0 {
			return nil
		}
		job := queue.Jobs[0]
		queue.Jobs = queue.Jobs[1:]
		return job
	}

	var selected *QueuedJob
	var selectedIndex int

	for i, qj := range queue.Jobs {
		if selected == nil || qj.Priority > selected.Priority {
			selected = qj
			selectedIndex = i
		}
	}

	if selected != nil {
		queue.Jobs = append(queue.Jobs[:selectedIndex], queue.Jobs[selectedIndex+1:]...)
		return selected
	}

	return nil
}

func (qm *QueueManager) Requeue(job *Job) error {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	queue, exists := qm.queues[job.Queue]
	if !exists {
		return fmt.Errorf("queue %s not found", job.Queue)
	}

	queue.mu.Lock()
	defer queue.mu.Unlock()

	queuedJob := &QueuedJob{
		Job:      job,
		QueuedAt: time.Now(),
		Priority: job.Priority,
		Attempts: 0,
	}

	queue.Jobs = append(queue.Jobs, queuedJob)
	klog.Infof("Job %s requeued to queue %s", job.ID, queue.Name)
	return nil
}

func (qm *QueueManager) GetQueueMetrics() map[string]int {
	qm.mu.Lock()
	defer qm.mu.Unlock()

	metrics := make(map[string]int)
	for name, queue := range qm.queues {
		queue.mu.Lock()
		metrics[name] = len(queue.Jobs)
		queue.mu.Unlock()
	}

	return metrics
}

type GangScheduler struct {
	gangs map[string]*Gang
	mu    sync.Mutex
}

type Gang struct {
	Name     string     `json:"name"`
	MinMember int       `json:"min_member"`
	MaxMember int       `json:"max_member"`
	Scheduled int       `json:"scheduled"`
	Jobs     []*Job     `json:"jobs"`
	Status   GangStatus `json:"status"`
}

type GangStatus string

const (
	GangStatusPending   GangStatus = "pending"
	GangStatusScheduling GangStatus = "scheduling"
	GangStatusReady     GangStatus = "ready"
	GangStatusBound     GangStatus = "bound"
	GangStatusFailed    GangStatus = "failed"
)

func NewGangScheduler() *GangScheduler {
	return &GangScheduler{
		gangs: make(map[string]*Gang),
	}
}

func (gs *GangScheduler) RegisterGang(name string, minMember, maxMember int) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if _, exists := gs.gangs[name]; exists {
		return fmt.Errorf("gang %s already exists", name)
	}

	gs.gangs[name] = &Gang{
		Name:      name,
		MinMember: minMember,
		MaxMember: maxMember,
		Status:    GangStatusPending,
		Jobs:      make([]*Job, 0),
	}

	klog.Infof("Gang %s registered with minMember=%d, maxMember=%d", name, minMember, maxMember)
	return nil
}

func (gs *GangScheduler) AddJobToGang(gangName string, job *Job) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	gang, exists := gs.gangs[gangName]
	if !exists {
		return fmt.Errorf("gang %s not found", gangName)
	}

	if len(gang.Jobs) >= gang.MaxMember {
		return fmt.Errorf("gang %s is full", gangName)
	}

	gang.Jobs = append(gang.Jobs, job)
	klog.Infof("Job %s added to gang %s", job.ID, gangName)

	if len(gang.Jobs) >= gang.MinMember {
		gang.Status = GangStatusReady
	}

	return nil
}

func (gs *GangScheduler) IsGangReady(gangName string) bool {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	gang, exists := gs.gangs[gangName]
	if !exists {
		return false
	}

	return gang.Status == GangStatusReady
}

func (gs *GangScheduler) GetGang(gangName string) (*Gang, error) {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	gang, exists := gs.gangs[gangName]
	if !exists {
		return nil, fmt.Errorf("gang %s not found", gangName)
	}

	return gang, nil
}

func (gs *GangScheduler) DeleteGang(gangName string) error {
	gs.mu.Lock()
	defer gs.mu.Unlock()

	if _, exists := gs.gangs[gangName]; !exists {
		return fmt.Errorf("gang %s not found", gangName)
	}

	delete(gs.gangs, gangName)
	klog.Infof("Gang %s deleted", gangName)
	return nil
}

func (gs *GangScheduler) ScheduleGang(gangName string) ([]*corev1.Pod, error) {
	gs.mu.Lock()
	gang, exists := gs.gangs[gangName]
	if !exists {
		gs.mu.Unlock()
		return nil, fmt.Errorf("gang %s not found", gangName)
	}

	if gang.Status != GangStatusReady {
		gs.mu.Unlock()
		return nil, fmt.Errorf("gang %s is not ready", gangName)
	}

	gang.Status = GangStatusScheduling
	gs.mu.Unlock()

	pods := make([]*corev1.Pod, 0, len(gang.Jobs))
	for _, job := range gang.Jobs {
		pod := gs.buildGangPod(job, gangName)
		pods = append(pods, pod)
	}

	gang.Status = GangStatusBound
	klog.Infof("Gang %s scheduled with %d pods", gangName, len(pods))

	return pods, nil
}

func (gs *GangScheduler) buildGangPod(job *Job, gangName string) *corev1.Pod {
	gpuResource := corev1.ResourceList{}
	if job.Spec.GPU > 0 {
		gpuResource[corev1.ResourceName("nvidia.com/gpu")] = *resource.NewQuantity(job.Spec.GPU, resource.DecimalSI)
	}

	if job.Spec.GPUMemory > 0 {
		gpuResource[corev1.ResourceName("nvidia.com/gpu_memory")] = *resource.NewQuantity(job.Spec.GPUMemory, resource.DecimalSI)
	}

	if job.Spec.GPUCores > 0 {
		gpuResource[corev1.ResourceName("nvidia.com/gpu_cores")] = *resource.NewQuantity(int64(job.Spec.GPUCores), resource.DecimalSI)
	}

	resources := corev1.ResourceRequirements{
		Requests: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(job.Spec.CPU),
			corev1.ResourceMemory: resource.MustParse(job.Spec.Memory),
		},
		Limits: corev1.ResourceList{
			corev1.ResourceCPU:    resource.MustParse(job.Spec.CPU),
			corev1.ResourceMemory: resource.MustParse(job.Spec.Memory),
		},
	}

	for resName, resQty := range gpuResource {
		resources.Requests[resName] = resQty
		resources.Limits[resName] = resQty
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      job.Name,
			Namespace: job.Namespace,
			Labels: map[string]string{
				"app":          "edge-job",
				"edgehub.io/gang": gangName,
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicy(job.Spec.RestartPolicy),
			SchedulerName: "volcano",
			Containers: []corev1.Container{
				{
					Name:    "job",
					Image:   job.Spec.Image,
					Command: job.Spec.Command,
					Args:    job.Spec.Args,
					Env:     job.Spec.Env,
					Resources: resources,
					VolumeMounts: func() []corev1.VolumeMount {
						mounts := make([]corev1.VolumeMount, 0)
						for _, vol := range job.Spec.Volumes {
							mounts = append(mounts, corev1.VolumeMount{
								Name:      vol.Name,
								MountPath: vol.MountPath,
							})
						}
						return mounts
					}(),
				},
			},
			NodeSelector: job.Spec.NodeSelector,
			Tolerations:  job.Spec.Tolerations,
			Volumes: func() []corev1.Volume {
				volumes := make([]corev1.Volume, 0)
				for _, vol := range job.Spec.Volumes {
					volumes = append(volumes, corev1.Volume{
						Name: vol.Name,
						VolumeSource: corev1.VolumeSource{
							EmptyDir: &corev1.EmptyDirVolumeSource{},
						},
					})
				}
				return volumes
			}(),
		},
	}
}

func NewPodSpec(job *Job) *corev1.Pod {
	gs := NewGangScheduler()
	return gs.buildGangPod(job, "default")
}
