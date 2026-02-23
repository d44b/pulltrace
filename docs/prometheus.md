# Prometheus Metrics

Pulltrace exposes Prometheus metrics on a separate port (default `:9090`, configured via `PULLTRACE_METRICS_ADDR`).

## Scrape Configuration

```yaml
scrape_configs:
  - job_name: pulltrace
    static_configs:
      - targets: ["pulltrace-server:9090"]
```

Or using Kubernetes service discovery with a `prometheus.io/scrape: "true"` annotation (the Helm chart sets this by default).

## Metrics Reference

| Metric | Type | Description |
|--------|------|-------------|
| `pulltrace_pulls_active` | Gauge | Image pulls currently in progress across all nodes |
| `pulltrace_pulls_total` | Counter | Total image pulls observed since server startup |
| `pulltrace_pull_duration_seconds` | Histogram | Pull duration in seconds (buckets: 1s, 5s, 10s, 30s, 1m, 2m, 5m, 10m) |
| `pulltrace_pull_bytes_total` | Counter | Total bytes downloaded across all pulls since server startup |
| `pulltrace_pull_errors_total` | Counter | Pulls that completed with a non-empty error field |
| `pulltrace_agent_reports_total` | Counter | Total agent report payloads received by the server |
| `pulltrace_sse_clients_active` | Gauge | Number of active SSE connections (browser UI clients) |

## Example Alert

```yaml
groups:
  - name: pulltrace
    rules:
      - alert: ImagePullErrors
        expr: increase(pulltrace_pull_errors_total[5m]) > 0
        for: 0m
        labels:
          severity: warning
        annotations:
          summary: "Image pull errors detected"
          description: "{{ $value }} pull error(s) in the last 5 minutes"
```
