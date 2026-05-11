package k8s

import (
	"context"
	"fmt"

	"github.com/edgehub/edgehub/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

type Clientset struct {
	Client   *kubernetes.Clientset
	Config   *rest.Config
	Scheme   *runtime.Scheme
}

func NewClientset(ctx context.Context, cfg config.KubernetesConfig) (*Clientset, error) {
	var config *rest.Config
	var err error

	if cfg.KubeConfig != "" {
		config, err = clientcmd.BuildConfigFromFlags(cfg.APIServer, cfg.KubeConfig)
	} else if cfg.InCluster {
		config, err = rest.InClusterConfig()
	} else {
		config, err = clientcmd.BuildConfigFromFlags(cfg.APIServer, "")
	}

	if err != nil {
		return nil, fmt.Errorf("failed to build kubernetes config: %w", err)
	}

	config.QPS = float32(cfg.QPS)
	config.Burst = cfg.Burst
	config.Timeout = cfg.Timeout

	client, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	scheme := runtime.NewScheme()
	corev1.AddToScheme(scheme)

	return &Clientset{
		Client: client,
		Config: config,
		Scheme: scheme,
	}, nil
}

func (c *Clientset) ListNodes(ctx context.Context) ([]corev1.Node, error) {
	nodes, err := c.Client.CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return nodes.Items, nil
}

func (c *Clientset) GetNode(ctx context.Context, name string) (*corev1.Node, error) {
	return c.Client.CoreV1().Nodes().Get(ctx, name, metav1.GetOptions{})
}

func (c *Clientset) UpdateNodeStatus(ctx context.Context, node *corev1.Node) (*corev1.Node, error) {
	return c.Client.CoreV1().Nodes().UpdateStatus(ctx, node, metav1.UpdateOptions{})
}

func (c *Clientset) CreatePod(ctx context.Context, namespace string, pod *corev1.Pod) (*corev1.Pod, error) {
	return c.Client.CoreV1().Pods(namespace).Create(ctx, pod, metav1.CreateOptions{})
}

func (c *Clientset) GetPod(ctx context.Context, namespace, name string) (*corev1.Pod, error) {
	return c.Client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
}

func (c *Clientset) ListPods(ctx context.Context, namespace string, opts metav1.ListOptions) (*corev1.PodList, error) {
	return c.Client.CoreV1().Pods(namespace).List(ctx, opts)
}

func (c *Clientset) DeletePod(ctx context.Context, namespace, name string) error {
	return c.Client.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (c *Clientset) UpdatePod(ctx context.Context, namespace string, pod *corev1.Pod) (*corev1.Pod, error) {
	return c.Client.CoreV1().Pods(namespace).Update(ctx, pod, metav1.UpdateOptions{})
}

func (c *Clientset) WatchPods(ctx context.Context, namespace string, opts metav1.ListOptions) (watch.Interface, error) {
	return c.Client.CoreV1().Pods(namespace).Watch(ctx, opts)
}

func (c *Clientset) CreateNamespace(ctx context.Context, ns *corev1.Namespace) (*corev1.Namespace, error) {
	return c.Client.CoreV1().Namespaces().Create(ctx, ns, metav1.CreateOptions{})
}

func (c *Clientset) GetNamespace(ctx context.Context, name string) (*corev1.Namespace, error) {
	return c.Client.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
}

func (c *Clientset) ListNamespaces(ctx context.Context) ([]corev1.Namespace, error) {
	nsList, err := c.Client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	return nsList.Items, nil
}

func (c *Clientset) GetEvents(ctx context.Context, namespace string) (*corev1.EventList, error) {
	return c.Client.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{})
}
