# Pulltrace

[![License](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go](https://img.shields.io/badge/Go-1.22+-00ADD8.svg)](https://go.dev)
[![Kubernetes](https://img.shields.io/badge/Kubernetes-1.28+-326CE5.svg)](https://kubernetes.io)

Real-time container image pull progress for Kubernetes.

Pulltrace connects to the containerd runtime on each node to track layer-by-layer download progress, correlates pulls with waiting pods, and provides a live web UI, structured JSON logs, and Prometheus metrics.

## Architecture

```
+---------------------+         +---------------------+
|   pulltrace-agent   |  HTTP   |  pulltrace-server   |
|   (DaemonSet)       | ------> |  (Deployment)       |
|                     | POST    |                     |
| - containerd socket |/report  | - Aggregates agents |
| - Monitors ingests  |         | - K8s pod watcher   |
| - Per-node tracking |         | - REST API           |
+---------------------+         | - SSE streaming     |
      one per node              | - Prometheus metrics |
                                | - Embedded web UI   |
                                +----------+----------+
                                           |
                                     SSE /events
                                           |
                                +----------v----------+
                                |      Web UI         |
                                |   (React + Vite)    |
                                |                     |
                                | - Live progress bars|
                                | - Pod correlation   |
                                | - Layer details     |
                                | - Filtering         |
                                +---------------------+
```

**Data flow:**

1. **Agent** reads containerd content store ingests on each node and snapshots active pulls every 1 second (configurable).
2. **Agent** POSTs `AgentReport` JSON to the server at `/api/v1/report`.
3. **Server** merges reports from all agents, enriches with pod correlation from the Kubernetes API, and computes rates/ETAs.
4. **Server** emits `PullEvent` JSON via SSE at `/api/v1/events` and exposes pulls at `/api/v1/pulls`.
5. **Web UI** connects to the SSE stream and renders live progress.

## Quick Start

```bash
helm repo add pulltrace https://d44b.github.io/pulltrace
helm install pulltrace pulltrace/pulltrace -n pulltrace --create-namespace
```

Or install from source:

```bash
git clone https://github.com/d44b/pulltrace.git
cd pulltrace
kubectl create namespace pulltrace
# Agent requires hostPath for containerd socket - set PodSecurity to privileged:
kubectl label namespace pulltrace pod-security.kubernetes.io/enforce=privileged --overwrite
helm install pulltrace ./charts/pulltrace -n pulltrace
```

> **Note:** The agent DaemonSet needs `hostPath` access to the containerd socket. If your cluster enforces PodSecurity standards (baseline or restricted), you must set the `pulltrace` namespace to `privileged` enforcement, otherwise agent pods will be rejected. This is the only required cluster-level change.

Access the UI:

```bash
kubectl port-forward -n pulltrace svc/pulltrace-server 8080:8080
# Open http://localhost:8080
```

## Configuration

Key `values.yaml` options:

| Parameter | Default | Description |
|---|---|---|
| `agent.containerd.socketPath` | `/run/containerd/containerd.sock` | Path to the containerd socket on the host |
| `agent.auth.token` | `""` | Shared secret for agent-to-server auth (recommended for production) |
| `config.logLevel` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `config.watchNamespaces` | `""` (all) | Comma-separated namespaces to watch for pod correlation |
| `config.historyTTL` | `30m` | How long completed pulls remain visible |
| `config.reportInterval` | `2s` | How often agents report to the server |
| `server.replicas` | `1` | Server replica count |
| `server.service.port` | `8080` | Server HTTP port (API + UI) |
| `server.service.metricsPort` | `9090` | Prometheus metrics port |
| `ingress.enabled` | `false` | Enable ingress for the server |
| `namespace` | `pulltrace` | Kubernetes namespace |

See [`charts/pulltrace/values.yaml`](charts/pulltrace/values.yaml) for the full reference.

## API Endpoints

| Method | Path | Description |
|---|---|---|
| `GET` | `/api/v1/pulls` | List all active and recent image pulls |
| `GET` | `/api/v1/events` | SSE stream of real-time `PullEvent` objects |
| `POST` | `/api/v1/report` | Agent report endpoint (internal) |
| `GET` | `/metrics` | Prometheus metrics (port 9090) |
| `GET` | `/healthz` | Health check |
| `GET` | `/` | Web UI |

## JSON Log Format

Pulltrace emits structured JSON logs for every pull lifecycle event. Schema version: `v1`.

```json
{
  "schemaVersion": "v1",
  "timestamp": "2025-01-15T10:30:00Z",
  "type": "pull.progress",
  "nodeName": "worker-1",
  "pull": {
    "id": "abc123",
    "imageRef": "docker.io/library/nginx:1.27",
    "totalBytes": 67108864,
    "downloadedBytes": 33554432,
    "bytesPerSec": 5242880,
    "etaSeconds": 6.4,
    "percent": 50.0,
    "layerCount": 7,
    "layersDone": 3,
    "startedAt": "2025-01-15T10:29:50Z",
    "pods": [
      {
        "namespace": "default",
        "podName": "nginx-7d4f8b6c9-x2k9p",
        "container": "nginx"
      }
    ],
    "totalKnown": true
  }
}
```

**Event types:** `pull.started`, `pull.progress`, `pull.completed`, `pull.failed`, `layer.started`, `layer.progress`, `layer.completed`

See [`docs/schemas/pull-event-v1.json`](docs/schemas/pull-event-v1.json) for the full JSON Schema.

## Prometheus Metrics

| Metric | Type | Description |
|---|---|---|
| `pulltrace_pulls_active` | Gauge | Number of image pulls currently in progress |
| `pulltrace_pulls_total` | Counter | Total image pulls observed |
| `pulltrace_pull_duration_seconds` | Histogram | Image pull duration |
| `pulltrace_pull_bytes_total` | Counter | Total bytes downloaded |
| `pulltrace_agents_connected` | Gauge | Number of connected agents |

## Security

Pulltrace has no built-in authentication. The API exposes cluster inventory data (node names, pod names, image references).

- **Agent token** — set `agent.auth.token` in `values.yaml` to require agents to authenticate with the server. Both agent and server must use the same token.
- **Network isolation** — enable `networkPolicy.enabled=true` to restrict who can reach the server (recommended for production).
- **Ingress auth** — if exposing via ingress, front it with an authenticating proxy (e.g., `oauth2-proxy`).
- **Agent socket** — the agent requires `runtimeSocket.enabled=true` and `runtimeSocket.risksAcknowledged=true`. Helm will fail if the socket is enabled without the acknowledgment.

See [SECURITY.md](SECURITY.md) for the full threat model.

## Known Limitations

- **containerd v2 only.** Pulltrace uses the containerd v2 content store API. Other runtimes (CRI-O, Docker) are not currently supported.
- **Total size is best-effort.** Layer total sizes may not be known until the manifest is fully resolved. The `totalKnown` field indicates whether the reported total is authoritative.
- **Single-cluster.** Pulltrace is designed for a single Kubernetes cluster. Multi-cluster aggregation is not built in.
- **Cached layers are invisible.** If a layer is already present on the node, containerd does not create an ingest and Pulltrace will not track it. Pulls that are fully cached will not appear.
- **No authentication.** The API and UI do not include authentication. Use network policies or ingress auth if needed.

## License

Apache License 2.0. See [LICENSE](LICENSE) for details.

## Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines on setting up a development environment, running tests, and submitting pull requests.
