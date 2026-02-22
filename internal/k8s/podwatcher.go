// Package k8s provides Kubernetes pod watching for image-pod correlation.
package k8s

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"github.com/d44b/pulltrace/internal/model"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// PodWatcher watches pods to correlate image pulls with waiting pods.
type PodWatcher struct {
	client     kubernetes.Interface
	namespaces []string
	mu         sync.RWMutex
	// podsByImage maps "node:image" -> []PodCorrelation
	podsByImage map[string][]model.PodCorrelation
	logger      *slog.Logger
	stopCh      chan struct{}
}

// NewPodWatcher creates a new pod watcher using in-cluster config.
func NewPodWatcher(namespaces []string, logger *slog.Logger) (*PodWatcher, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, fmt.Errorf("creating in-cluster config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("creating kubernetes client: %w", err)
	}

	return &PodWatcher{
		client:      clientset,
		namespaces:  namespaces,
		podsByImage: make(map[string][]model.PodCorrelation),
		logger:      logger,
		stopCh:      make(chan struct{}),
	}, nil
}

// Run starts watching pods.
func (pw *PodWatcher) Run(ctx context.Context) error {
	pw.logger.Info("starting pod watcher", "namespaces", pw.namespaces)

	ns := ""
	if len(pw.namespaces) == 1 {
		ns = pw.namespaces[0]
	}
	// If namespaces is empty or has multiple entries, watch all and filter

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-pw.stopCh:
			return nil
		default:
		}

		if err := pw.watchLoop(ctx, ns); err != nil {
			pw.logger.Error("watch error, retrying", "error", err)
			time.Sleep(5 * time.Second)
		}
	}
}

func (pw *PodWatcher) watchLoop(ctx context.Context, namespace string) error {
	watcher, err := pw.client.CoreV1().Pods(namespace).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("watching pods: %w", err)
	}
	defer watcher.Stop()

	// Initial list to populate cache
	pods, err := pw.client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("listing pods: %w", err)
	}
	for i := range pods.Items {
		pw.updatePod(&pods.Items[i])
	}

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("watch channel closed")
			}
			pod, ok := event.Object.(*corev1.Pod)
			if !ok {
				continue
			}

			switch event.Type {
			case watch.Added, watch.Modified:
				pw.updatePod(pod)
			case watch.Deleted:
				pw.removePod(pod)
			}
		}
	}
}

func (pw *PodWatcher) updatePod(pod *corev1.Pod) {
	if len(pw.namespaces) > 0 && !pw.inNamespaces(pod.Namespace) {
		return
	}

	pw.mu.Lock()
	defer pw.mu.Unlock()

	nodeName := pod.Spec.NodeName
	if nodeName == "" {
		return
	}

	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason == "ContainerCreating" {
			// This container might be waiting for an image pull
			for _, c := range pod.Spec.Containers {
				if c.Name == cs.Name {
					key := nodeName + ":" + normalizeImageRef(c.Image)
					corr := model.PodCorrelation{
						Namespace: pod.Namespace,
						PodName:   pod.Name,
						Container: c.Name,
					}
					pw.addCorrelation(key, corr)
				}
			}
		}
	}

	// Also check init containers
	for _, cs := range pod.Status.InitContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason == "ContainerCreating" {
			for _, c := range pod.Spec.InitContainers {
				if c.Name == cs.Name {
					key := nodeName + ":" + normalizeImageRef(c.Image)
					corr := model.PodCorrelation{
						Namespace: pod.Namespace,
						PodName:   pod.Name,
						Container: c.Name,
					}
					pw.addCorrelation(key, corr)
				}
			}
		}
	}
}

func (pw *PodWatcher) removePod(pod *corev1.Pod) {
	pw.mu.Lock()
	defer pw.mu.Unlock()

	for key, corrs := range pw.podsByImage {
		var filtered []model.PodCorrelation
		for _, c := range corrs {
			if c.PodName != pod.Name || c.Namespace != pod.Namespace {
				filtered = append(filtered, c)
			}
		}
		if len(filtered) == 0 {
			delete(pw.podsByImage, key)
		} else {
			pw.podsByImage[key] = filtered
		}
	}
}

func (pw *PodWatcher) addCorrelation(key string, corr model.PodCorrelation) {
	existing := pw.podsByImage[key]
	for _, e := range existing {
		if e.Namespace == corr.Namespace && e.PodName == corr.PodName && e.Container == corr.Container {
			return
		}
	}
	pw.podsByImage[key] = append(existing, corr)
}

func (pw *PodWatcher) inNamespaces(ns string) bool {
	for _, n := range pw.namespaces {
		if n == ns {
			return true
		}
	}
	return false
}

// GetPodsForImage returns pods waiting for a specific image on a specific node.
func (pw *PodWatcher) GetPodsForImage(nodeName, imageRef string) []model.PodCorrelation {
	pw.mu.RLock()
	defer pw.mu.RUnlock()

	key := nodeName + ":" + normalizeImageRef(imageRef)
	return pw.podsByImage[key]
}

// normalizeImageRef normalizes an image reference for matching.
func normalizeImageRef(ref string) string {
	// Add docker.io/library/ prefix for short names
	if !strings.Contains(ref, "/") {
		ref = "docker.io/library/" + ref
	} else if !strings.Contains(ref, ".") {
		ref = "docker.io/" + ref
	}
	// Add :latest if no tag
	if !strings.Contains(ref, ":") && !strings.Contains(ref, "@") {
		ref = ref + ":latest"
	}
	return ref
}

// Stop stops the watcher.
func (pw *PodWatcher) Stop() {
	close(pw.stopCh)
}
