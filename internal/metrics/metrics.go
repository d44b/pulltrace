// Package metrics provides Prometheus metrics for Pulltrace.
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	PullsActive = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "pulltrace",
		Name:      "pulls_active",
		Help:      "Number of currently active image pulls.",
	})

	PullsTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "pulltrace",
		Name:      "pulls_total",
		Help:      "Total number of image pulls observed.",
	})

	PullDurationSeconds = promauto.NewHistogram(prometheus.HistogramOpts{
		Namespace: "pulltrace",
		Name:      "pull_duration_seconds",
		Help:      "Duration of image pulls in seconds.",
		Buckets:   []float64{1, 5, 10, 30, 60, 120, 300, 600},
	})

	PullBytesTotal = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "pulltrace",
		Name:      "pull_bytes_total",
		Help:      "Total bytes downloaded across all pulls.",
	})

	PullErrors = promauto.NewCounter(prometheus.CounterOpts{
		Namespace: "pulltrace",
		Name:      "pull_errors_total",
		Help:      "Total number of pull errors.",
	})

	AgentReports = promauto.NewCounterVec(prometheus.CounterOpts{
		Namespace: "pulltrace",
		Name:      "agent_reports_total",
		Help:      "Total agent reports received, by node.",
	}, []string{"node"})

	SSEClients = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "pulltrace",
		Name:      "sse_clients_active",
		Help:      "Number of active SSE client connections.",
	})
)
