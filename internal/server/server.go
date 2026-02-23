package server

import (
	"context"
	"crypto/subtle"
	"encoding/json"
	"fmt"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/d44b/pulltrace/internal/k8s"
	"github.com/d44b/pulltrace/internal/metrics"
	"github.com/d44b/pulltrace/internal/model"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	maxReportBodyBytes = 1 << 20 // 1 MiB

	rateLimitWindow = 500 * time.Millisecond

	// maxRateLimitEntries prevents memory exhaustion from reports with arbitrary node names.
	maxRateLimitEntries = 1024

	// maxActivePulls prevents unbounded memory growth; new pulls are dropped at capacity.
	maxActivePulls = 10000

	// maxSSEClients prevents resource exhaustion from SSE connections.
	maxSSEClients = 256

	// stalePullTimeout force-completes pulls that stop sending updates.
	stalePullTimeout = 10 * time.Minute

	mergedPullSuffix = ":__merged__"
)

type Config struct {
	HTTPAddr        string
	MetricsAddr     string
	LogLevel        string
	WatchNamespaces []string
	HistoryTTL      time.Duration
	AgentToken      string // if non-empty, agents must present a matching Bearer token
}

func ConfigFromEnv() Config {
	c := Config{
		HTTPAddr:    envOrDefault("PULLTRACE_HTTP_ADDR", ":8080"),
		MetricsAddr: envOrDefault("PULLTRACE_METRICS_ADDR", ":9090"),
		LogLevel:    envOrDefault("PULLTRACE_LOG_LEVEL", "info"),
		AgentToken:  os.Getenv("PULLTRACE_AGENT_TOKEN"),
	}

	if ns := os.Getenv("PULLTRACE_WATCH_NAMESPACES"); ns != "" {
		c.WatchNamespaces = strings.Split(ns, ",")
	}

	if ttl := os.Getenv("PULLTRACE_HISTORY_TTL"); ttl != "" {
		if d, err := time.ParseDuration(ttl); err == nil {
			c.HistoryTTL = d
		}
	}
	if c.HistoryTTL == 0 {
		c.HistoryTTL = 30 * time.Minute
	}

	return c
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

type rateLimiter struct {
	mu    sync.Mutex
	nodes map[string]time.Time
}

func newRateLimiter() *rateLimiter {
	return &rateLimiter{nodes: make(map[string]time.Time)}
}

func (rl *rateLimiter) allow(node string) bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	last, ok := rl.nodes[node]
	if ok && now.Sub(last) < rateLimitWindow {
		return false
	}
	if !ok && len(rl.nodes) >= maxRateLimitEntries {
		return false
	}
	rl.nodes[node] = now
	return true
}

func (rl *rateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	cutoff := time.Now().Add(-1 * time.Minute)
	for node, ts := range rl.nodes {
		if ts.Before(cutoff) {
			delete(rl.nodes, node)
		}
	}
}

type Server struct {
	config      Config
	logger      *slog.Logger
	podWatcher  *k8s.PodWatcher
	mu          sync.RWMutex
	pulls       map[string]*model.PullStatus
	rates       map[string]*model.RateCalculator
	lastSeen    map[string]time.Time
	// lastBytes tracks the highest cumulative downloadedBytes ever fed to each
	// RateCalculator. This ensures rc.Add always receives a monotonically
	// non-decreasing value so that Rate() never goes negative when a concurrent
	// pull finishes and the merged byte total drops.
	lastBytes   map[string]int64
	sseClients  map[chan []byte]struct{}
	sseMu       sync.Mutex
	webFS       fs.FS
	rateLimiter *rateLimiter
}

func New(cfg Config, webFS fs.FS) *Server {
	level := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	return &Server{
		config:      cfg,
		logger:      slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})),
		pulls:       make(map[string]*model.PullStatus),
		rates:       make(map[string]*model.RateCalculator),
		lastSeen:    make(map[string]time.Time),
		lastBytes:   make(map[string]int64),
		sseClients:  make(map[chan []byte]struct{}),
		webFS:       webFS,
		rateLimiter: newRateLimiter(),
	}
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
		w.Header().Set("Content-Security-Policy",
			"default-src 'self'; "+
				"script-src 'self'; "+
				"style-src 'self' https://fonts.googleapis.com; "+
				"font-src 'self' https://fonts.gstatic.com; "+
				"img-src 'self' data:; "+
				"connect-src 'self'; "+
				"frame-ancestors 'none'")
		next.ServeHTTP(w, r)
	})
}

func (s *Server) Run(ctx context.Context) error {
	s.logger.Info("starting pulltrace server",
		"httpAddr", s.config.HTTPAddr,
		"metricsAddr", s.config.MetricsAddr,
		"tokenAuth", s.config.AgentToken != "",
	)

	pw, err := k8s.NewPodWatcher(s.config.WatchNamespaces, s.logger)
	if err != nil {
		s.logger.Warn("pod watcher unavailable, running without pod correlation", "error", err)
	} else {
		s.podWatcher = pw
		go func() {
			if err := pw.Run(ctx); err != nil {
				s.logger.Error("pod watcher failed", "error", err)
			}
		}()
	}

	go s.cleanupLoop(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/report", s.handleReport)
	mux.HandleFunc("/api/v1/pulls", s.handlePulls)
	mux.HandleFunc("/api/v1/events", s.handleSSE)
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)

	if s.webFS != nil {
		mux.Handle("/", http.FileServer(http.FS(s.webFS)))
	}

	httpServer := &http.Server{
		Addr:              s.config.HTTPAddr,
		Handler:           securityHeaders(mux),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		// WriteTimeout is 0 because SSE connections are long-lived.
		// Slow-client protection comes from the 256-client cap and non-blocking channel sends.
		IdleTimeout:    120 * time.Second,
		MaxHeaderBytes: 1 << 16,
	}

	// Metrics on a separate port — restrict access via NetworkPolicy.
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsServer := &http.Server{
		Addr:              s.config.MetricsAddr,
		Handler:           metricsMux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	errCh := make(chan error, 2)
	go func() { errCh <- httpServer.ListenAndServe() }()
	go func() { errCh <- metricsServer.ListenAndServe() }()

	s.logger.Info("server started", "http", s.config.HTTPAddr, "metrics", s.config.MetricsAddr)

	select {
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		httpServer.Shutdown(shutdownCtx)   //nolint:errcheck
		metricsServer.Shutdown(shutdownCtx) //nolint:errcheck
		return nil
	case err := <-errCh:
		return err
	}
}

func (s *Server) handleReport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if s.config.AgentToken != "" {
		provided := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		if subtle.ConstantTimeCompare([]byte(provided), []byte(s.config.AgentToken)) != 1 {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}
	}

	r.Body = http.MaxBytesReader(w, r.Body, maxReportBodyBytes)

	var report model.AgentReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	if report.NodeName == "" || len(report.NodeName) > 253 {
		http.Error(w, "invalid nodeName", http.StatusBadRequest)
		return
	}

	if !s.rateLimiter.allow(report.NodeName) {
		http.Error(w, "rate limited", http.StatusTooManyRequests)
		return
	}

	metrics.AgentReports.Inc()
	s.processReport(report)
	w.WriteHeader(http.StatusOK)
}

// isContentDigest reports whether ref is a raw containerd content digest rather
// than a human-readable image name. These refs are merged server-side into a
// single logical pull keyed by "__pulling__" until the image name is resolved.
func isContentDigest(ref string) bool {
	for _, prefix := range []string{"sha256:", "layer-sha256:", "config-sha256:", "manifest-sha256:", "index-sha256:"} {
		if strings.HasPrefix(ref, prefix) {
			return true
		}
	}
	return false
}

// mergeDigestPulls consolidates raw content-digest entries from an agent report
// into a single synthetic pull. Containerd tracks individual content objects
// (manifest, config, layers) as separate ingests; grouping them here gives a
// coherent per-image view before the image name is resolved from K8s events.
func mergeDigestPulls(pulls []model.PullState) []model.PullState {
	var normal []model.PullState
	var digestLayers []model.LayerState
	var earliestStart time.Time
	allKnown := true

	for _, pull := range pulls {
		if isContentDigest(pull.ImageRef) {
			digestLayers = append(digestLayers, pull.Layers...)
			if earliestStart.IsZero() || pull.StartedAt.Before(earliestStart) {
				earliestStart = pull.StartedAt
			}
			if !pull.TotalKnown {
				allKnown = false
			}
		} else {
			normal = append(normal, pull)
		}
	}

	if len(digestLayers) == 0 {
		return normal
	}

	merged := model.PullState{
		ImageRef:   "__pulling__",
		Layers:     digestLayers,
		StartedAt:  earliestStart,
		TotalKnown: allKnown,
	}
	return append(normal, merged)
}

func (s *Server) processReport(report model.AgentReport) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	mergedPulls := mergeDigestPulls(report.Pulls)
	updatedKeys := make(map[string]bool)

	for _, pull := range mergedPulls {
		key := report.NodeName + ":" + pull.ImageRef
		if pull.ImageRef == "__pulling__" {
			key = report.NodeName + mergedPullSuffix
		}
		updatedKeys[key] = true

		existing, ok := s.pulls[key]
		if !ok || existing.CompletedAt != nil {
			if !ok && len(s.pulls) >= maxActivePulls {
				s.logger.Warn("pulls map at capacity, dropping new pull",
					"node", report.NodeName,
					"image", pull.ImageRef,
					"limit", maxActivePulls,
				)
				continue
			}
			// Use a unique ID per pull so the frontend creates a fresh row
			// instead of reusing the previous completed entry (same slot key).
			uid := fmt.Sprintf("%s@%d", key, now.UnixNano())
			existing = &model.PullStatus{
				ID:        uid,
				NodeName:  report.NodeName,
				ImageRef:  pull.ImageRef,
				StartedAt: pull.StartedAt,
			}
			s.pulls[key] = existing
			metrics.PullsTotal.Inc()
			metrics.PullsActive.Inc()
		}

		if strings.HasSuffix(key, mergedPullSuffix) && existing.ImageRef == "__pulling__" && s.podWatcher != nil {
			if images := s.podWatcher.GetPullingImagesForNode(report.NodeName); len(images) > 0 {
				existing.ImageRef = images[0]
			} else if images := s.podWatcher.GetWaitingImagesForNode(report.NodeName); len(images) > 0 {
				existing.ImageRef = images[0]
			}
		}

		s.lastSeen[key] = now

		var totalBytes, downloadedBytes int64
		layersDone := 0
		layerStatuses := make([]model.LayerStatus, 0, len(pull.Layers))

		for _, layer := range pull.Layers {
			totalBytes += layer.TotalBytes
			downloadedBytes += layer.DownloadedBytes

			ls := model.LayerStatus{
				PullID:          key,
				Digest:          layer.Digest,
				MediaType:       layer.MediaType,
				TotalBytes:      layer.TotalBytes,
				DownloadedBytes: layer.DownloadedBytes,
				TotalKnown:      layer.TotalKnown,
			}

			layerKey := key + ":layer:" + layer.Digest
			lrc, ok := s.rates[layerKey]
			if !ok {
				lrc = model.NewRateCalculator(10 * time.Second)
				s.rates[layerKey] = lrc
			}
			layerPrev := s.lastBytes[layerKey]
			layerDelta := layer.DownloadedBytes - layerPrev
			if layerDelta < 0 {
				layerDelta = 0
			}
			s.lastBytes[layerKey] = layerPrev + layerDelta
			lrc.Add(layerPrev + layerDelta)
			ls.BytesPerSec = lrc.Rate()

			if layer.TotalKnown && layer.TotalBytes > 0 {
				ls.Percent = float64(layer.DownloadedBytes) / float64(layer.TotalBytes) * 100
			}
			if layer.TotalKnown && layer.DownloadedBytes >= layer.TotalBytes {
				ls.Percent = 100
				layersDone++
				completedAt := now
				ls.CompletedAt = &completedAt
			}
			layerStatuses = append(layerStatuses, ls)
		}

		existing.TotalBytes = totalBytes
		existing.DownloadedBytes = downloadedBytes
		existing.LayerCount = len(pull.Layers)
		existing.LayersDone = layersDone
		existing.TotalKnown = pull.TotalKnown
		existing.Layers = layerStatuses

		if totalBytes > 0 {
			existing.Percent = float64(downloadedBytes) / float64(totalBytes) * 100
		}

		rc, ok := s.rates[key]
		if !ok {
			rc = model.NewRateCalculator(10 * time.Second)
			s.rates[key] = rc
		}
		prev := s.lastBytes[key]
		delta := downloadedBytes - prev
		if delta < 0 {
			delta = 0
		}
		s.lastBytes[key] = prev + delta
		rc.Add(prev + delta)
		existing.BytesPerSec = rc.Rate()
		if existing.TotalKnown && existing.TotalBytes > existing.DownloadedBytes {
			existing.ETASeconds = rc.ETA(existing.TotalBytes - existing.DownloadedBytes)
		}

		if s.podWatcher != nil {
			existing.Pods = s.podWatcher.GetPodsForImage(report.NodeName, existing.ImageRef)
		}

		event := model.PullEvent{
			SchemaVersion: model.SchemaVersion,
			Timestamp:     report.Timestamp,
			Type:          model.EventPullProgress,
			NodeName:      report.NodeName,
			Pull:          existing,
		}
		if data, err := json.Marshal(event); err == nil {
			s.logger.Debug("pull.progress",
				"node", report.NodeName,
				"image", existing.ImageRef,
				"percent", existing.Percent,
			)
			s.broadcastSSE(data)
		}
	}

	// Pulls absent from the report have completed on the node.
	nodePrefix := report.NodeName + ":"
	for key, pull := range s.pulls {
		if !strings.HasPrefix(key, nodePrefix) {
			continue
		}
		if updatedKeys[key] || pull.CompletedAt != nil {
			continue
		}

		pull.CompletedAt = &now
		pull.Percent = 100
		metrics.PullsActive.Dec()
		metrics.PullDurationSeconds.Observe(now.Sub(pull.StartedAt).Seconds())
		metrics.PullBytesTotal.Add(float64(pull.TotalBytes))
		if pull.Error != "" {
			metrics.PullErrors.Inc()
		}

		event := model.PullEvent{
			SchemaVersion: model.SchemaVersion,
			Timestamp:     now,
			Type:          model.EventPullCompleted,
			NodeName:      report.NodeName,
			Pull:          pull,
		}
		if data, err := json.Marshal(event); err == nil {
			s.logger.Info("pull.completed",
				"node", report.NodeName,
				"image", pull.ImageRef,
				"duration", now.Sub(pull.StartedAt).String(),
			)
			s.broadcastSSE(data)
		}
	}
}

func (s *Server) handlePulls(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.mu.RLock()
	pulls := make([]model.PullStatus, 0, len(s.pulls))
	for _, p := range s.pulls {
		pulls = append(pulls, *p)
	}
	s.mu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(model.APIResponse{Pulls: pulls}) //nolint:errcheck
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	// Register client atomically to avoid TOCTOU between capacity check and insertion.
	ch := make(chan []byte, 64)
	s.sseMu.Lock()
	if len(s.sseClients) >= maxSSEClients {
		s.sseMu.Unlock()
		http.Error(w, "too many connections", http.StatusServiceUnavailable)
		return
	}
	s.sseClients[ch] = struct{}{}
	s.sseMu.Unlock()
	metrics.SSEClients.Inc()

	defer func() {
		s.sseMu.Lock()
		delete(s.sseClients, ch)
		s.sseMu.Unlock()
		metrics.SSEClients.Dec()
	}()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// SSE comment flushes headers through buffering proxies.
	w.Write([]byte(": connected\n\n")) //nolint:errcheck

	s.mu.RLock()
	now := time.Now()
	for _, p := range s.pulls {
		event := model.PullEvent{
			SchemaVersion: model.SchemaVersion,
			Timestamp:     now,
			Type:          model.EventPullProgress,
			NodeName:      p.NodeName,
			Pull:          p,
		}
		if data, err := json.Marshal(event); err == nil {
			w.Write([]byte("data: ")) //nolint:errcheck
			w.Write(data)             //nolint:errcheck
			w.Write([]byte("\n\n"))   //nolint:errcheck
		}
	}
	s.mu.RUnlock()
	flusher.Flush()

	for {
		select {
		case <-r.Context().Done():
			return
		case data, ok := <-ch:
			if !ok {
				return
			}
			w.Write([]byte("data: ")) //nolint:errcheck
			w.Write(data)             //nolint:errcheck
			w.Write([]byte("\n\n"))   //nolint:errcheck
			flusher.Flush()
		}
	}
}

func (s *Server) broadcastSSE(data []byte) {
	s.sseMu.Lock()
	defer s.sseMu.Unlock()
	for ch := range s.sseClients {
		select {
		case ch <- data:
		default:
			// Drop if client is slow.
		}
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok")) //nolint:errcheck
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok")) //nolint:errcheck
}

func (s *Server) cleanupLoop(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.cleanup()
			s.rateLimiter.cleanup()
		}
	}
}

func (s *Server) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	ttlCutoff := now.Add(-s.config.HistoryTTL)

	for key, pull := range s.pulls {
		if pull.CompletedAt != nil && pull.CompletedAt.Before(ttlCutoff) {
			delete(s.pulls, key)
			delete(s.rates, key)
			delete(s.lastSeen, key)
			delete(s.lastBytes, key)
			layerPrefix := key + ":layer:"
			for rateKey := range s.rates {
				if strings.HasPrefix(rateKey, layerPrefix) {
					delete(s.rates, rateKey)
					delete(s.lastBytes, rateKey)
				}
			}
			continue
		}
		if pull.CompletedAt == nil {
			if lastSeen, ok := s.lastSeen[key]; ok && now.Sub(lastSeen) > stalePullTimeout {
				completedAt := lastSeen
				pull.CompletedAt = &completedAt
				metrics.PullsActive.Dec()
				s.logger.Warn("force-completing stale pull", "key", key, "lastSeen", lastSeen)
			}
		}
	}
}
