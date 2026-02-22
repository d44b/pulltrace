// Package server implements the Pulltrace aggregator server.
package server

import (
	"context"
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

// Config holds server configuration.
type Config struct {
	HTTPAddr        string
	MetricsAddr     string
	LogLevel        string
	WatchNamespaces []string
	HistoryTTL      time.Duration
}

// ConfigFromEnv loads server config from environment variables.
func ConfigFromEnv() Config {
	c := Config{
		HTTPAddr:    envOrDefault("PULLTRACE_HTTP_ADDR", ":8080"),
		MetricsAddr: envOrDefault("PULLTRACE_METRICS_ADDR", ":9090"),
		LogLevel:    envOrDefault("PULLTRACE_LOG_LEVEL", "info"),
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

// Server is the Pulltrace aggregator.
type Server struct {
	config     Config
	logger     *slog.Logger
	podWatcher *k8s.PodWatcher
	mu         sync.RWMutex
	pulls      map[string]*model.PullStatus // keyed by "node:imageRef"
	sseClients map[chan []byte]struct{}
	sseMu      sync.Mutex
	webFS      fs.FS
}

// New creates a new server.
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
		config:     cfg,
		logger:     slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})),
		pulls:      make(map[string]*model.PullStatus),
		sseClients: make(map[chan []byte]struct{}),
		webFS:      webFS,
	}
}

// Run starts the server.
func (s *Server) Run(ctx context.Context) error {
	s.logger.Info("starting pulltrace server",
		"httpAddr", s.config.HTTPAddr,
		"metricsAddr", s.config.MetricsAddr,
	)

	// Start pod watcher
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

	// Start TTL cleanup
	go s.cleanupLoop(ctx)

	// HTTP API
	mux := http.NewServeMux()
	mux.HandleFunc("/api/v1/report", s.handleReport)
	mux.HandleFunc("/api/v1/pulls", s.handlePulls)
	mux.HandleFunc("/api/v1/events", s.handleSSE)
	mux.HandleFunc("/healthz", s.handleHealthz)
	mux.HandleFunc("/readyz", s.handleReadyz)

	// Serve embedded UI
	if s.webFS != nil {
		mux.Handle("/", http.FileServer(http.FS(s.webFS)))
	}

	httpServer := &http.Server{
		Addr:    s.config.HTTPAddr,
		Handler: mux,
	}

	// Metrics server
	metricsMux := http.NewServeMux()
	metricsMux.Handle("/metrics", promhttp.Handler())
	metricsServer := &http.Server{
		Addr:    s.config.MetricsAddr,
		Handler: metricsMux,
	}

	errCh := make(chan error, 2)
	go func() { errCh <- httpServer.ListenAndServe() }()
	go func() { errCh <- metricsServer.ListenAndServe() }()

	s.logger.Info("server started", "http", s.config.HTTPAddr, "metrics", s.config.MetricsAddr)

	select {
	case <-ctx.Done():
		httpServer.Shutdown(context.Background())
		metricsServer.Shutdown(context.Background())
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

	var report model.AgentReport
	if err := json.NewDecoder(r.Body).Decode(&report); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}

	metrics.AgentReports.WithLabelValues(report.NodeName).Inc()

	s.processReport(report)

	w.WriteHeader(http.StatusOK)
}

func (s *Server) processReport(report model.AgentReport) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, pull := range report.Pulls {
		key := report.NodeName + ":" + pull.ImageRef
		existing, ok := s.pulls[key]
		if !ok {
			existing = &model.PullStatus{
				ID:         key,
				ImageRef:   pull.ImageRef,
				StartedAt:  pull.StartedAt,
				TotalKnown: pull.TotalKnown,
			}
			s.pulls[key] = existing
			metrics.PullsTotal.Inc()
			metrics.PullsActive.Inc()
		}

		// Aggregate layer data
		var totalBytes, downloadedBytes int64
		layersDone := 0
		for _, layer := range pull.Layers {
			totalBytes += layer.TotalBytes
			downloadedBytes += layer.DownloadedBytes
			if layer.TotalKnown && layer.DownloadedBytes >= layer.TotalBytes {
				layersDone++
			}
		}

		existing.TotalBytes = totalBytes
		existing.DownloadedBytes = downloadedBytes
		existing.LayerCount = len(pull.Layers)
		existing.LayersDone = layersDone
		existing.TotalKnown = pull.TotalKnown

		if totalBytes > 0 {
			existing.Percent = float64(downloadedBytes) / float64(totalBytes) * 100
		}

		// Add pod correlation
		if s.podWatcher != nil {
			existing.Pods = s.podWatcher.GetPodsForImage(report.NodeName, pull.ImageRef)
		}

		// Emit SSE event
		event := model.PullEvent{
			SchemaVersion: model.SchemaVersion,
			Timestamp:     report.Timestamp,
			Type:          model.EventPullProgress,
			NodeName:      report.NodeName,
			Pull:          existing,
		}

		// Log as JSON
		if data, err := json.Marshal(event); err == nil {
			fmt.Println(string(data))
			s.broadcastSSE(data)
		}
	}

	// Check for completed pulls (pulls present in state but not in report)
	for key, pull := range s.pulls {
		if !strings.HasPrefix(key, report.NodeName+":") {
			continue
		}
		found := false
		for _, rp := range report.Pulls {
			if report.NodeName+":"+rp.ImageRef == key {
				found = true
				break
			}
		}
		if !found && pull.CompletedAt == nil {
			now := time.Now()
			pull.CompletedAt = &now
			pull.Percent = 100
			metrics.PullsActive.Dec()
			metrics.PullDurationSeconds.Observe(now.Sub(pull.StartedAt).Seconds())
			metrics.PullBytesTotal.Add(float64(pull.TotalBytes))

			event := model.PullEvent{
				SchemaVersion: model.SchemaVersion,
				Timestamp:     now,
				Type:          model.EventPullCompleted,
				NodeName:      report.NodeName,
				Pull:          pull,
			}
			if data, err := json.Marshal(event); err == nil {
				fmt.Println(string(data))
				s.broadcastSSE(data)
			}
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
	json.NewEncoder(w).Encode(model.APIResponse{Pulls: pulls})
}

func (s *Server) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	ch := make(chan []byte, 64)
	s.sseMu.Lock()
	s.sseClients[ch] = struct{}{}
	s.sseMu.Unlock()
	metrics.SSEClients.Inc()

	defer func() {
		s.sseMu.Lock()
		delete(s.sseClients, ch)
		s.sseMu.Unlock()
		metrics.SSEClients.Dec()
	}()

	// Send current state as initial data
	s.mu.RLock()
	for _, p := range s.pulls {
		event := model.PullEvent{
			SchemaVersion: model.SchemaVersion,
			Timestamp:     time.Now(),
			Type:          model.EventPullProgress,
			Pull:          p,
		}
		if data, err := json.Marshal(event); err == nil {
			fmt.Fprintf(w, "data: %s\n\n", data)
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
			fmt.Fprintf(w, "data: %s\n\n", data)
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
			// Drop if client is slow
		}
	}
}

func (s *Server) handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

func (s *Server) handleReadyz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
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
		}
	}
}

func (s *Server) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-s.config.HistoryTTL)
	for key, pull := range s.pulls {
		if pull.CompletedAt != nil && pull.CompletedAt.Before(cutoff) {
			delete(s.pulls, key)
		}
	}
}
