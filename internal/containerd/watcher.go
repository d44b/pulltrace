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

type Watcher struct {
	socketPath string
	namespace  string
	client     *containerd.Client
	mu         sync.RWMutex
	pulls      map[string]*pullTracker
	closeOnce  sync.Once
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

func (w *Watcher) Connect(ctx context.Context) error {
	c, err := containerd.New(w.socketPath, containerd.WithDefaultNamespace(w.namespace))
	if err != nil {
		return fmt.Errorf("connecting to containerd at %s: %w", w.socketPath, err)
	}
	w.client = c
	return nil
}

func (w *Watcher) Close() error {
	var err error
	w.closeOnce.Do(func() {
		close(w.stopCh)
		if w.client != nil {
			err = w.client.Close()
		}
	})
	return err
}

func (w *Watcher) Poll(ctx context.Context) ([]model.PullState, error) {
	if w.client == nil {
		return nil, fmt.Errorf("not connected")
	}

	store := w.client.ContentStore()
	statuses, err := store.ListStatuses(ctx, "")
	if err != nil {
		return nil, fmt.Errorf("listing content statuses: %w", err)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	activeRefs := make(map[string]bool)
	for _, status := range statuses {
		activeRefs[status.Ref] = true
		w.updateLayerFromStatus(status)
	}

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
		// Keep the entry briefly so the server sees the completed state.
		if allDone && time.Since(pt.startedAt) > 30*time.Second {
			delete(w.pulls, key)
		}
	}
}

// extractImageRef groups containerd ingest refs by image. Ingest refs for K8s
// image pulls typically follow "<image>@<digest>" or bare digest patterns.
// This is best-effort; the server correlates with K8s events for accuracy.
func extractImageRef(ref string) string {
	parts := strings.SplitN(ref, "@", 2)
	if len(parts) == 2 {
		return parts[0]
	}
	return ref
}
