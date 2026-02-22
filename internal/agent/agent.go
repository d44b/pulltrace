package agent

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"strings"
	"time"

	ctrd "github.com/d44b/pulltrace/internal/containerd"
	"github.com/d44b/pulltrace/internal/model"
)

// allowedSocketPrefixes restricts the agent to containerd sockets, preventing
// accidental or malicious redirection to other UNIX sockets on the host.
var allowedSocketPrefixes = []string{
	"/run/containerd/",
	"/var/run/containerd/",
}

type Config struct {
	NodeName         string
	ServerURL        string
	ContainerdSocket string
	ReportInterval   time.Duration
	LogLevel         string
	AgentToken       string
}

func ConfigFromEnv() Config {
	c := Config{
		NodeName:         os.Getenv("PULLTRACE_NODE_NAME"),
		ServerURL:        os.Getenv("PULLTRACE_SERVER_URL"),
		ContainerdSocket: envOrDefault("PULLTRACE_CONTAINERD_SOCKET", "/run/containerd/containerd.sock"),
		LogLevel:         envOrDefault("PULLTRACE_LOG_LEVEL", "info"),
		AgentToken:       os.Getenv("PULLTRACE_AGENT_TOKEN"),
	}

	if interval := os.Getenv("PULLTRACE_REPORT_INTERVAL"); interval != "" {
		if d, err := time.ParseDuration(interval); err == nil {
			c.ReportInterval = d
		}
	}
	if c.ReportInterval == 0 {
		c.ReportInterval = 1 * time.Second
	}

	return c
}

func envOrDefault(key, defaultVal string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultVal
}

type Agent struct {
	config  Config
	watcher *ctrd.Watcher
	client  *http.Client
	logger  *slog.Logger
}

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
		client:  &http.Client{Timeout: 10 * time.Second},
		logger:  slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: level})),
	}
}

func (a *Agent) Run(ctx context.Context) error {
	a.logger.Info("starting pulltrace agent",
		"node", a.config.NodeName,
		"server", a.config.ServerURL,
		"socket", a.config.ContainerdSocket,
		"interval", a.config.ReportInterval,
		"tokenAuth", a.config.AgentToken != "",
	)

	// Validate socket path to prevent connecting to non-containerd sockets.
	if err := validateSocketPath(a.config.ContainerdSocket); err != nil {
		return fmt.Errorf("socket path validation: %w", err)
	}

	if err := a.watcher.Connect(ctx); err != nil {
		return fmt.Errorf("connecting to containerd: %w", err)
	}
	defer a.watcher.Close()

	a.logger.Info("connected to containerd")

	ticker := time.NewTicker(a.config.ReportInterval)
	defer ticker.Stop()

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
	if a.config.AgentToken != "" {
		req.Header.Set("Authorization", "Bearer "+a.config.AgentToken)
	}

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

func validateSocketPath(path string) error {
	for _, prefix := range allowedSocketPrefixes {
		if strings.HasPrefix(path, prefix) {
			return nil
		}
	}
	return fmt.Errorf(
		"socket path %q is not under an allowed prefix (%v); only containerd sockets are supported",
		path, allowedSocketPrefixes,
	)
}
