package scheduler

import (
	"context"
	"log"
	"sync"
	"time"

	"github.com/edgehub/edgehub/internal/config"
	"github.com/edgehub/edgehub/internal/k8s"
	"github.com/edgehub/edgehub/internal/models"
	"github.com/google/uuid"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type Scheduler struct {
	cfg       config.SchedulerConfig
	k8sClient *k8s.Clientset
	
	queues    map[string]*JobQueue
	mu        sync.RWMutex
	
	stopCh    chan struct{}
	wg        sync.WaitGroup
}

type JobQueue struct {
	jobs     []*models.Job
	priority int
	mu       sync.Mutex
}

func NewScheduler(cfg config.SchedulerConfig, k8sClient *k8s.Clientset) *Scheduler {
	return &Scheduler{
		cfg:       cfg,
		k8sClient: k8sClient,
		queues:    make(map[string]*JobQueue),
		stopCh:    make(chan struct{}),
	}
}

func (s *Scheduler) Start(ctx context.Context) error {
	log.Println("Starting scheduler...")

	s.wg.Add(1)
	go s.scheduleLoop(ctx)

	s.wg.Add(1)
	go s.cleanupLoop(ctx)

	log.Println("Scheduler started successfully")
	return nil
}

func (s *Scheduler) Stop() {
	close(s.stopCh)
	s.wg.Wait()
}

func (s *Scheduler) scheduleLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(time.Duration(s.cfg.Workers) * 100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.processQueues(ctx)
		}
	}
}

func (s *Scheduler) cleanupLoop(ctx context.Context) {
	defer s.wg.Done()

	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.cleanupOrphanedPods(ctx)
		}
	}
}

func (s *Scheduler) processQueues(ctx context.Context) {
	s.mu.RLock()
	queues := make([]*JobQueue, 0, len(s.queues))
	for _, q := range s.queues {
		queues = append(queues, q)
	}
	s.mu.RUnlock()

	for _, queue := range queues {
		queue.mu.Lock()
		for i := 0; i < len(queue.jobs) && i < s.cfg.Workers; i++ {
			job := queue.jobs[i]
			if err := s.scheduleJob(ctx, job); err != nil {
				log.Printf("Failed to schedule job %s: %v", job.ID, err)
				continue
			}
			queue.jobs = append(queue.jobs[:i], queue.jobs[i+1:]...)
			i--
		}
		queue.mu.Unlock()
	}
}

func (s *Scheduler) scheduleJob(ctx context.Context, job *models.Job) error {
	nodes, err := s.k8sClient.ListNodes(ctx)
	if err != nil {
		return err
	}

	targetNode, err := s.selectNode(nodes, job)
	if err != nil {
		return err
	}

	pod, err := s.createPodFromJob(job, targetNode.Name)
	if err != nil {
		return err
	}

	_, err = s.k8sClient.CreatePod(ctx, job.Namespace, pod)
	if err != nil {
		return err
	}

	log.Printf("Scheduled job %s on node %s", job.ID, targetNode.Name)
	return nil
}

func (s *Scheduler) selectNode(nodes []corev1.Node, job *models.Job) (*corev1.Node, error) {
	var selected *corev1.Node
	minScore := float64(0)

	for i := range nodes {
		node := &nodes[i]
		if !s.isNodeReady(node) {
			continue
		}

		if !s.matchesRequirements(node, &job.Spec) {
			continue
		}

		score := s.calculateNodeScore(node, job)
		if selected == nil || score < minScore {
			selected = node
			minScore = score
		}
	}

	if selected == nil {
		return nil, ErrNoSuitableNode
	}

	return selected, nil
}

func (s *Scheduler) isNodeReady(node *corev1.Node) bool {
	for _, condition := range node.Status.Conditions {
		if condition.Type == corev1.NodeReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func (s *Scheduler) matchesRequirements(node *corev1.Node, spec *models.JobSpec) bool {
	allocatable := node.Status.Allocatable

	if spec.Resources.Requests.CPU != "" {
		cpuReq, err := resource.ParseQuantity(spec.Resources.Requests.CPU)
		if err == nil {
			cpuAlloc := allocatable.Cpu().DeepCopy()
			cpuAlloc.Sub(cpuReq)
			if cpuAlloc.Sign() < 0 {
				return false
			}
		}
	}

	if spec.Resources.Requests.Memory != "" {
		memReq, err := resource.ParseQuantity(spec.Resources.Requests.Memory)
		if err == nil {
			memAlloc := allocatable.Memory().DeepCopy()
			memAlloc.Sub(memReq)
			if memAlloc.Sign() < 0 {
				return false
			}
		}
	}

	for _, taint := range node.Spec.Taints {
		if taint.Effect == corev1.TaintEffectNoSchedule {
			if !s.hasToleration(spec.Tolerations, taint) {
				return false
			}
		}
	}

	return true
}

func (s *Scheduler) hasToleration(tolerations []models.Taint, taint corev1.Taint) bool {
	for _, t := range tolerations {
		if t.Key == taint.Key && (t.Value == taint.Value || t.Value == "") {
			return true
		}
	}
	return false
}

func (s *Scheduler) calculateNodeScore(node *corev1.Node, job *models.Job) float64 {
	var score float64

	allocatable := node.Status.Allocatable
	
	cpuUsage := 1 - float64(allocatable.Cpu().Value())/100
	memUsage := 1 - float64(allocatable.Memory().Value())/(128*1024*1024*1024)
	score = (cpuUsage + memUsage) / 2

	for _, label := range job.Spec.NodeSelector {
		if val, ok := node.Labels[label]; ok && val != job.Spec.NodeSelector[label] {
			score += 10
		}
	}

	return score
}

func (s *Scheduler) createPodFromJob(job *models.Job, nodeName string) (*corev1.Pod, error) {
	labels := make(map[string]string)
	for k, v := range job.Labels {
		if vs, ok := v.(string); ok {
			labels[k] = vs
		}
	}
	
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "job-" + job.ID.String(),
			Namespace: job.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyNever,
			NodeName:      nodeName,
		},
	}

	if len(job.Spec.Image) > 0 {
		pod.Spec.Containers = []corev1.Container{
			{
				Name:            "main",
				Image:           job.Spec.Image,
				Command:         job.Spec.Command,
				Args:            job.Spec.Args,
				WorkingDir:      job.Spec.WorkingDir,
				Env:             s.convertEnv(job.Spec.Env),
				VolumeMounts:    s.convertVolumeMounts(job.Spec.Volumes),
				Resources: corev1.ResourceRequirements{
					Requests: s.convertResourceList(job.Spec.Resources.Requests),
					Limits:   s.convertResourceList(job.Spec.Resources.Limits),
				},
			},
		}
	}

	pod.Spec.Volumes = s.convertVolumes(job.Spec.Volumes)

	return pod, nil
}

func (s *Scheduler) convertEnv(env map[string]string) []corev1.EnvVar {
	result := make([]corev1.EnvVar, 0, len(env))
	for k, v := range env {
		result = append(result, corev1.EnvVar{Name: k, Value: v})
	}
	return result
}

func (s *Scheduler) convertResourceList(rl models.ResourceList) corev1.ResourceList {
	result := corev1.ResourceList{}
	if rl.CPU != "" {
		result[corev1.ResourceCPU] = resource.MustParse(rl.CPU)
	}
	if rl.Memory != "" {
		result[corev1.ResourceMemory] = resource.MustParse(rl.Memory)
	}
	if rl.GPU != "" {
		result[corev1.ResourceEphemeralStorage] = resource.MustParse(rl.GPU)
	}
	return result
}

func (s *Scheduler) convertVolumes(volumes []models.Volume) []corev1.Volume {
	result := make([]corev1.Volume, 0, len(volumes))
	for _, v := range volumes {
		vol := corev1.Volume{Name: v.Name}
		if v.EmptyDir != nil {
			vol.VolumeSource = corev1.VolumeSource{
				EmptyDir: &corev1.EmptyDirVolumeSource{},
			}
		} else if v.HostPath != nil {
			vol.VolumeSource = corev1.VolumeSource{
				HostPath: &corev1.HostPathVolumeSource{
					Path: v.HostPath.Path,
					Type: (*corev1.HostPathType)(&v.HostPath.Type),
				},
			}
		}
		result = append(result, vol)
	}
	return result
}

func (s *Scheduler) convertVolumeMounts(volumes []models.Volume) []corev1.VolumeMount {
	result := make([]corev1.VolumeMount, 0, len(volumes))
	for _, v := range volumes {
		result = append(result, corev1.VolumeMount{
			Name:      v.Name,
			MountPath: v.MountPath,
			ReadOnly:  v.ReadOnly,
		})
	}
	return result
}

func (s *Scheduler) cleanupOrphanedPods(ctx context.Context) {
	namespaces, err := s.k8sClient.ListNamespaces(ctx)
	if err != nil {
		log.Printf("Failed to list namespaces: %v", err)
		return
	}

	for _, ns := range namespaces {
		if ns.Status.Phase == corev1.NamespaceTerminating {
			continue
		}

		pods, err := s.k8sClient.ListPods(ctx, ns.Name, metav1.ListOptions{})
		if err != nil {
			continue
		}

		for _, pod := range pods.Items {
			if pod.DeletionTimestamp != nil {
				continue
			}

			if !isJobPod(pod.Name) {
				continue
			}

			if pod.Status.Phase == corev1.PodSucceeded || pod.Status.Phase == corev1.PodFailed {
				log.Printf("Cleaning up orphaned pod %s/%s", ns.Name, pod.Name)
				s.k8sClient.DeletePod(ctx, ns.Name, pod.Name)
			}
		}
	}
}

func isJobPod(name string) bool {
	return len(name) > 4 && name[:4] == "job-"
}

func (s *Scheduler) EnqueueJob(job *models.Job) {
	s.mu.Lock()
	defer s.mu.Unlock()

	queueName := job.Queue
	if queueName == "" {
		queueName = "default"
	}

	if _, exists := s.queues[queueName]; !exists {
		s.queues[queueName] = &JobQueue{
			jobs:     make([]*models.Job, 0),
			priority: 50,
		}
	}

	q := s.queues[queueName]
	q.mu.Lock()
	q.jobs = append(q.jobs, job)
	q.mu.Unlock()
}

var ErrNoSuitableNode = &SchedulerError{Message: "no suitable node found"}

type SchedulerError struct {
	Message string
}

func (e *SchedulerError) Error() string {
	return e.Message
}

type Config struct {
	Type string
}

func (c *Scheduler) RegisterJobHandler(jobType string, handler JobHandler) {
}

type JobHandler interface {
	Handle(ctx context.Context, job *models.Job) error
}

func GenerateJobID() uuid.UUID {
	return uuid.New()
}
