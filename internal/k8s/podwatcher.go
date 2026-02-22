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

const pullingImageTTL = 10 * time.Minute

// PodWatcher watches pods and kubelet events to correlate image pulls with pods.
type PodWatcher struct {
	client     kubernetes.Interface
	namespaces []string
	mu         sync.RWMutex
	// podsByImage maps "node:normalizedImage" -> []PodCorrelation
	podsByImage map[string][]model.PodCorrelation
	// pullingByNode tracks images currently being pulled per node,
	// based on kubelet "Pulling" events. Values are insertion timestamps for TTL.
	pullingByNode map[string]map[string]time.Time
	logger        *slog.Logger
	stopCh        chan struct{}
}

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
		client:        clientset,
		namespaces:    namespaces,
		podsByImage:   make(map[string][]model.PodCorrelation),
		pullingByNode: make(map[string]map[string]time.Time),
		logger:        logger,
		stopCh:        make(chan struct{}),
	}, nil
}

func (pw *PodWatcher) Run(ctx context.Context) error {
	pw.logger.Info("starting pod watcher", "namespaces", pw.namespaces)

	ns := ""
	if len(pw.namespaces) == 1 {
		ns = pw.namespaces[0]
	}

	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-pw.stopCh:
				return
			default:
			}
			if err := pw.watchEvents(ctx, ns); err != nil {
				pw.logger.Error("event watch error, retrying", "error", err)
				time.Sleep(5 * time.Second)
			}
		}
	}()

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-pw.stopCh:
				return
			case <-ticker.C:
				pw.cleanupStalePulling()
			}
		}
	}()

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

func (pw *PodWatcher) watchEvents(ctx context.Context, namespace string) error {
	watcher, err := pw.client.CoreV1().Events(namespace).Watch(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("watching events: %w", err)
	}
	defer watcher.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case event, ok := <-watcher.ResultChan():
			if !ok {
				return fmt.Errorf("event watch channel closed")
			}
			ev, ok := event.Object.(*corev1.Event)
			if !ok {
				continue
			}
			if ev.InvolvedObject.Kind != "Pod" {
				continue
			}
			switch ev.Reason {
			case "Pulling":
				image := parseImageFromPullingMessage(ev.Message)
				node := ev.Source.Host
				if image != "" && node != "" {
					pw.addPullingImage(node, image)
					pw.logger.Debug("pulling event", "node", node, "image", image)
				}
			case "Pulled":
				image := parseImageFromPulledMessage(ev.Message)
				node := ev.Source.Host
				if image != "" && node != "" {
					pw.removePullingImage(node, image)
					pw.logger.Debug("pulled event", "node", node, "image", image)
				}
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
			for _, c := range pod.Spec.Containers {
				if c.Name == cs.Name {
					key := nodeName + ":" + normalizeImageRef(c.Image)
					pw.addCorrelation(key, model.PodCorrelation{
						Namespace: pod.Namespace,
						PodName:   pod.Name,
						Container: c.Name,
						Image:     c.Image,
					})
				}
			}
		}
	}

	for _, cs := range pod.Status.InitContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason == "ContainerCreating" {
			for _, c := range pod.Spec.InitContainers {
				if c.Name == cs.Name {
					key := nodeName + ":" + normalizeImageRef(c.Image)
					pw.addCorrelation(key, model.PodCorrelation{
						Namespace: pod.Namespace,
						PodName:   pod.Name,
						Container: c.Name,
						Image:     c.Image,
					})
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

func (pw *PodWatcher) addPullingImage(nodeName, image string) {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	if pw.pullingByNode[nodeName] == nil {
		pw.pullingByNode[nodeName] = make(map[string]time.Time)
	}
	pw.pullingByNode[nodeName][image] = time.Now()
}

func (pw *PodWatcher) removePullingImage(nodeName, image string) {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	if images, ok := pw.pullingByNode[nodeName]; ok {
		delete(images, image)
		if len(images) == 0 {
			delete(pw.pullingByNode, nodeName)
		}
	}
}

// cleanupStalePulling removes entries from pullingByNode that have not received
// a "Pulled" event within pullingImageTTL. This prevents unbounded growth when
// kubelet events are missed (e.g., due to watcher restarts).
func (pw *PodWatcher) cleanupStalePulling() {
	pw.mu.Lock()
	defer pw.mu.Unlock()
	cutoff := time.Now().Add(-pullingImageTTL)
	for node, images := range pw.pullingByNode {
		for img, ts := range images {
			if ts.Before(cutoff) {
				delete(images, img)
			}
		}
		if len(images) == 0 {
			delete(pw.pullingByNode, node)
		}
	}
}

func (pw *PodWatcher) inNamespaces(ns string) bool {
	for _, n := range pw.namespaces {
		if n == ns {
			return true
		}
	}
	return false
}

func (pw *PodWatcher) GetPodsForImage(nodeName, imageRef string) []model.PodCorrelation {
	pw.mu.RLock()
	defer pw.mu.RUnlock()
	return pw.podsByImage[nodeName+":"+normalizeImageRef(imageRef)]
}

func (pw *PodWatcher) GetPullingImagesForNode(nodeName string) []string {
	pw.mu.RLock()
	defer pw.mu.RUnlock()
	if images, ok := pw.pullingByNode[nodeName]; ok {
		result := make([]string, 0, len(images))
		for img := range images {
			result = append(result, img)
		}
		return result
	}
	return nil
}

func (pw *PodWatcher) GetWaitingImagesForNode(nodeName string) []string {
	pw.mu.RLock()
	defer pw.mu.RUnlock()
	prefix := nodeName + ":"
	seen := make(map[string]bool)
	var images []string
	for key := range pw.podsByImage {
		if strings.HasPrefix(key, prefix) {
			img := key[len(prefix):]
			if !seen[img] {
				seen[img] = true
				images = append(images, img)
			}
		}
	}
	return images
}

func (pw *PodWatcher) Stop() {
	close(pw.stopCh)
}

func normalizeImageRef(ref string) string {
	if !strings.Contains(ref, "/") {
		ref = "docker.io/library/" + ref
	} else {
		// Check whether the first path segment looks like a registry hostname
		// (contains "." or ":"). Only inspect the first segment to avoid false
		// matches on version tags like "1.27" in "library/nginx:1.27".
		firstSeg := ref[:strings.Index(ref, "/")]
		if !strings.ContainsAny(firstSeg, ".:") {
			ref = "docker.io/" + ref
		}
	}
	if !strings.Contains(ref, ":") && !strings.Contains(ref, "@") {
		ref = ref + ":latest"
	}
	return ref
}

// parseImageFromPullingMessage extracts the image from a kubelet Pulling event.
// Message format: Pulling image "nginx:latest"
func parseImageFromPullingMessage(msg string) string {
	const prefix = "Pulling image \""
	idx := strings.Index(msg, prefix)
	if idx == -1 {
		return ""
	}
	rest := msg[idx+len(prefix):]
	end := strings.Index(rest, "\"")
	if end == -1 {
		return ""
	}
	return rest[:end]
}

// parseImageFromPulledMessage extracts the image from a kubelet Pulled event.
// Message format: Successfully pulled image "nginx:latest" in 5.2s
func parseImageFromPulledMessage(msg string) string {
	const prefix = "pulled image \""
	idx := strings.Index(msg, prefix)
	if idx == -1 {
		return ""
	}
	rest := msg[idx+len(prefix):]
	end := strings.Index(rest, "\"")
	if end == -1 {
		return ""
	}
	return rest[:end]
}
