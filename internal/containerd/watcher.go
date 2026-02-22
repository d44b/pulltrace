// Package containerd provides integration with the containerd runtime
// to monitor image pull progress via the content store.
package containerd

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/d44b/pulltrace/internal/model"

	containerd "github.com/containerd/containerd/v2/client"
	"github.com/containerd/containerd/v2/core/content"
)

// Watcher monitors containerd for active image pulls.
type Watcher struct {
	socketPath string
	namespace  string
	client     *containerd.Client
	mu         sync.RWMutex
	pulls      map[string]*pullTracker
	stopCh     chan struct{}
}

type pullTracker struct {
	imageRef  string
	layers    map[string]*layerTracker
	startedAt time.Time
}

type layerTracker struct {
	digest          string
	totalBytes      int64
	downloadedBytes int64
	totalKnown      bool
	startedAt       time.Time
	completedAt     *time.Time
	rate            *model.RateCalculator
}

// NewWatcher creates a new containerd watcher.
func NewWatcher(socketPath, namespace string) *Watcher {
	if namespace == "" {
		namespace = "k8s.io"
	}
	return &Watcher{
		socketPath: socketPath,
		namespace:  namespace,
		pulls:      make(map[string]*pullTracker),
		stopCh:     make(chan struct{}),
	}
}

// Connect establishes connection to containerd.
func (w *Watcher) Connect(ctx context.Context) error {
	c, err := containerd.New(w.socketPath, containerd.WithDefaultNamespace(w.namespace))
	if err != nil {
		return fmt.Errorf("connecting to containerd at %s: %w", w.socketPath, err)
	}
	w.client = c
	return nil
}

// Close shuts down the watcher.
func (w *Watcher) Close() error {
	close(w.stopCh)
	if w.client != nil {
		return w.client.Close()
	}
	return nil
}

// Poll queries containerd for active ingests and returns current pull states.
func (w *Watcher) Poll(ctx context.Context) ([]model.PullState, error) {
	if w.client == nil {
		return nil, fmt.Errorf("not connected")
	}

	store := w.client.ContentStore()

	// List active ingests (layers being downloaded)
	statuses, err := store.ListStatuses(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("listing content statuses: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	// Track which ingests are still active
	activeRefs := make(map[string]bool)

	for _, status := range statuses {
		activeRefs[status.Ref] = true
		w.updateLayerFromStatus(status)
	}

	// Mark completed layers
	for _, pt := range w.pulls {
		for ref, lt := range pt.layers {
			if !activeRefs[ref] && lt.completedAt == nil {
				now := time.Now()
				lt.completedAt = &now
				if lt.totalKnown {
					lt.downloadedBytes = lt.totalBytes
				}
			}
		}
	}

	// Build output
	var states []model.PullState
	for _, pt := range w.pulls {
		ps := model.PullState{
			ImageRef:   pt.imageRef,
			StartedAt:  pt.startedAt,
			TotalKnown: true,
		}
		for _, lt := range pt.layers {
			ls := model.LayerState{
				Digest:          lt.digest,
				TotalBytes:      lt.totalBytes,
				DownloadedBytes: lt.downloadedBytes,
				TotalKnown:      lt.totalKnown,
			}
			if !lt.totalKnown {
				ps.TotalKnown = false
			}
			ps.Layers = append(ps.Layers, ls)
		}
		states = append(states, ps)
	}

	// Clean up completed pulls (all layers done)
	w.cleanCompleted()

	return states, nil
}

func (w *Watcher) updateLayerFromStatus(status content.Status) {
	ref := status.Ref
	imageRef := extractImageRef(ref)

	pt, ok := w.pulls[imageRef]
	if !ok {
		pt = &pullTracker{
			imageRef:  imageRef,
			layers:    make(map[string]*layerTracker),
			startedAt: status.StartedAt,
		}
		w.pulls[imageRef] = pt
	}

	lt, ok := pt.layers[ref]
	if !ok {
		lt = &layerTracker{
			digest:    ref,
			startedAt: status.StartedAt,
			rate:      model.NewRateCalculator(10 * time.Second),
		}
		pt.layers[ref] = lt
	}

	lt.downloadedBytes = status.Offset
	lt.totalBytes = status.Total
	lt.totalKnown = status.Total > 0
	lt.rate.Add(status.Offset)
}

func (w *Watcher) cleanCompleted() {
	for key, pt := range w.pulls {
		allDone := true
		for _, lt := range pt.layers {
			if lt.completedAt == nil {
				allDone = false
				break
			}
		}
		if allDone && time.Since(pt.startedAt) > 30*time.Second {
			delete(w.pulls, key)
		}
	}
}

// extractImageRef tries to parse an image reference from a containerd ingest ref.
// Containerd CRI ingest refs for image pulls typically follow patterns like:
// "content-<hash>-<digest>" or "<image>@<digest>"
// This is best-effort; the server correlates with K8s pod events for accuracy.
func extractImageRef(ref string) string {
	// Containerd ingest refs for K8s image pulls often look like:
	// "sha256:<hex>" or contain the image reference
	// Group layers by the prefix before any sha256 digest
	parts := strings.SplitN(ref, "@", 2)
	if len(parts) == 2 {
		return parts[0]
	}
	return ref
}
