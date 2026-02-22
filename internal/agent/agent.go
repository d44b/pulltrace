// Package agent implements the per-node Pulltrace agent.
package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"time"

	ctrd "github.com/d44b/pulltrace/internal/containerd"
	"github.com/d44b/pulltrace/internal/model"
)

// Config holds agent configuration.
type Config struct {
	NodeName        string
	ServerURL       string
	ContainerdSocket string
	ReportInterval  time.Duration
	LogLevel        string
}

// ConfigFromEnv loads agent config from environment variables.
func ConfigFromEnv() Config {
	c := Config{
		NodeName:        os.Getenv("PULLTRACE_NODE_NAME"),
		ServerURL:       os.Getenv("PULLTRACE_SERVER_URL"),
		ContainerdSocket: envOrDefault("PULLTRACE_CONTAINERD_SOCKET", "/run/containerd/containerd.sock"),
		LogLevel:        envOrDefault("PULLTRACE_LOG_LEVEL", "info"),
	}

	if interval := os.Getenv("PULLTRACE_REPORT_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			c.ReportInterval = d
		}
	}
	if c.ReportInterval == 0 {
		c.ReportInterval = 2 * time.Second
	}

	return c
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

// Agent is the per-node Pulltrace agent.
type Agent struct {
	config  Config
	watcher *ctrd.Watcher
	client  *http.Client
	logger  *slog.Logger
}

// New creates a new agent.
func New(cfg Config) *Agent {
	level := slog.LevelInfo
	switch cfg.LogLevel {
	case "debug":
		level = slog.LevelDebug
	case "warn":
		level = slog.LevelWarn
	case "error":
		level = slog.LevelError
	}

	return &Agent{
		config:  cfg,
		watcher: ctrd.NewWatcher(cfg.ContainerdSocket, "k8s.io"),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})),
	}
}

// Run starts the agent main loop.
func (a *Agent) Run(ctx context.Context) error {
	a.logger.Info("starting pulltrace agent",
		"node", a.config.NodeName,
		"server", a.config.ServerURL,
		"socket", a.config.ContainerdSocket,
		"interval", a.config.ReportInterval,
	)

	if err := a.watcher.Connect(ctx); err != nil {
		return fmt.Errorf("connecting to containerd: %w", err)
	}
	defer a.watcher.Close()

	a.logger.Info("connected to containerd")

	ticker := time.NewTicker(a.config.ReportInterval)
	defer ticker.Stop()

	// Retry loop with jitter for initial connection
	for {
		select {
		case <-ctx.Done():
			a.logger.Info("agent shutting down")
			return nil
		case <-ticker.C:
			if err := a.pollAndReport(ctx); err != nil {
				a.logger.Error("poll/report failed", "error", err)
			}
		}
	}
}

func (a *Agent) pollAndReport(ctx context.Context) error {
	states, err := a.watcher.Poll(ctx)
	if err != nil {
		return fmt.Errorf("polling containerd: %w", err)
	}

	report := model.AgentReport{
		NodeName:  a.config.NodeName,
		Timestamp: time.Now(),
		Pulls:     states,
	}

	// Log each active pull as JSON
	for _, p := range states {
		a.logger.Info("pull.progress",
			"imageRef", p.ImageRef,
			"layers", len(p.Layers),
			"totalKnown", p.TotalKnown,
		)
	}

	if a.config.ServerURL == "" {
		return nil
	}

	return a.sendReport(ctx, report)
}

func (a *Agent) sendReport(ctx context.Context, report model.AgentReport) error {
	data, err := json.Marshal(report)
	if err != nil {
		return fmt.Errorf("marshaling report: %w", err)
	}

	url := a.config.ServerURL + "/api/v1/report"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.client.Do(req)
	if err != nil {
		return fmt.Errorf("sending report to %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %d", resp.StatusCode)
	}

	return nil
}
