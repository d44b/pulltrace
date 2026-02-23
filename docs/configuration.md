# Configuration

Pulltrace is configured entirely through environment variables. The Helm chart sets all required values automatically; the tables below are for reference when deploying outside Helm or overriding defaults.

## Server

The server is the central aggregator. It receives reports from agents, watches the Kubernetes API for pod correlation, and serves the web UI and SSE event stream.

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `PULLTRACE_HTTP_ADDR` | string | `:8080` | HTTP listen address for the API and UI |
| `PULLTRACE_METRICS_ADDR` | string | `:9090` | Prometheus metrics listen address |
| `PULLTRACE_LOG_LEVEL` | string | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `PULLTRACE_AGENT_TOKEN` | string | _(empty)_ | Shared token for agent authentication (optional; leave empty to disable auth) |
| `PULLTRACE_WATCH_NAMESPACES` | string | _(empty — all)_ | Comma-separated namespaces for pod/event correlation; empty means watch all namespaces |
| `PULLTRACE_HISTORY_TTL` | duration | `30m` | How long completed pulls remain visible in the UI |

## Agent

One agent DaemonSet pod runs on each node. It polls the local containerd socket and reports image pull progress to the server.

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `PULLTRACE_NODE_NAME` | string | _(required)_ | Kubernetes node name; injected automatically via `fieldRef: spec.nodeName` |
| `PULLTRACE_SERVER_URL` | string | _(required)_ | URL of the Pulltrace server (e.g. `http://pulltrace-server:8080`) |
| `PULLTRACE_CONTAINERD_SOCKET` | string | `/run/containerd/containerd.sock` | Host path to the containerd gRPC socket |
| `PULLTRACE_LOG_LEVEL` | string | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `PULLTRACE_AGENT_TOKEN` | string | _(empty)_ | Bearer token sent to the server; must match `PULLTRACE_AGENT_TOKEN` on the server if set |
| `PULLTRACE_REPORT_INTERVAL` | duration | `1s` | How often the agent polls containerd and sends a report to the server |

## Helm Values

When using the Helm chart, set environment variables via `values.yaml` overrides:

```yaml
server:
  env:
    PULLTRACE_LOG_LEVEL: debug
    PULLTRACE_HISTORY_TTL: 60m

agent:
  env:
    PULLTRACE_REPORT_INTERVAL: 2s
```

See `charts/pulltrace/values.yaml` in the repository for the full values reference.
