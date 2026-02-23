# Architecture

**Analysis Date:** 2026-02-23

## Pattern Overview

**Overall:** Agent-Server-UI streaming architecture with real-time pull aggregation.

**Key Characteristics:**
- Distributed agents push containerd pull state to central server via HTTP POST
- Server aggregates state from multiple nodes and broadcasts updates via Server-Sent Events (SSE)
- React frontend consumes initial state snapshot and subscribes to real-time SSE stream
- Pod watcher correlates image pulls with waiting pods via Kubernetes API
- Sentinel merge pattern consolidates raw content digests into logical per-image pulls

## Layers

**Agent Layer:**
- Purpose: Poll containerd for active image pulls on a node, report state to server
- Location: `cmd/pulltrace-agent/main.go`, `internal/agent/agent.go`, `internal/containerd/watcher.go`
- Contains: Containerd socket polling, HTTP reporting, socket path validation
- Depends on: containerd API (v2), model data types
- Used by: Deployed as Kubernetes DaemonSet, runs on each node

**Server Aggregation Layer:**
- Purpose: Receive agent reports, aggregate multi-node pull state, manage SSE broadcast
- Location: `internal/server/server.go`, `cmd/pulltrace-server/main.go`
- Contains: Pull storage, rate limiting, image resolution, pod correlation, metrics
- Depends on: Kubernetes API (via PodWatcher), model types, HTTP handlers
- Used by: Frontend via REST and SSE, agents via POST

**Data Model Layer:**
- Purpose: Define canonical data structures for pulls, layers, events
- Location: `internal/model/event.go`, `internal/model/rate.go`
- Contains: PullStatus, PullEvent, LayerStatus, RateCalculator
- Depends on: Standard library only
- Used by: All layers (agent, server, frontend via JSON)

**Pod Watcher Layer:**
- Purpose: Watch Kubernetes pods and kubelet events to correlate images with pulling pods
- Location: `internal/k8s/podwatcher.go`
- Contains: K8s client, event watching, pod-image caching with TTL
- Depends on: client-go (Kubernetes client library)
- Used by: Server's image resolution logic

**Metrics Layer:**
- Purpose: Export Prometheus metrics for observability
- Location: `internal/metrics/metrics.go`
- Contains: Pull counters, gauges, histograms, SSE client tracking
- Depends on: prometheus client_golang
- Used by: Server during report processing and SSE management

**Frontend UI Layer:**
- Purpose: Display real-time pull progress with filtering and drill-down
- Location: `web/src/App.jsx`, `web/src/hooks.js`, `web/src/components/`
- Contains: React components, SSE subscription hook, filter logic
- Depends on: React, Vite, model JSON structures
- Used by: Browsers accessing server HTTP root

## Data Flow

**Agent Report Flow:**

1. Agent connects to containerd socket (validated for security)
2. Ticker triggers every `ReportInterval` (default 1s)
3. `Poll()` queries containerd ContentStore for active ingests (layer downloads)
4. Extracts layer metadata: digest, bytes downloaded, total size
5. Groups layers by image reference (imageRef) into PullState structs
6. Builds AgentReport with NodeName, timestamp, pulls array
7. POST to `{serverURL}/api/v1/report` with optional Bearer token
8. Server receives, validates, rate-limits by node
9. Returns HTTP 200 or error status

**Server Aggregation Flow:**

1. `handleReport()` receives AgentReport JSON
2. Validates nodeName length and format
3. Rate limiter checks node hasn't reported within 500ms window
4. `mergeDigestPulls()` consolidates raw content digests (sha256:*, layer-sha256:*, config-sha256:*) into synthetic `__pulling__` pull
5. For each merged pull:
   - Creates/updates PullStatus keyed by `node:imageRef`
   - Calculates bytes/percent from layer totals
   - Updates RateCalculator with current downloadedBytes
   - Queries PodWatcher for pod correlations
   - Resolves `__pulling__` to actual image name from pod watching
   - Builds LayerStatus array with per-layer progress
6. Detects completed pulls (missing from report but previously active)
7. Broadcasts PullEvent over SSE to all connected clients
8. Updates Prometheus metrics

**Pod Watcher Resolution:**

1. Watches kubelet "Pulling" and "Pulled" events in cluster
2. Maintains `pullingByNode[node][image]` with insertion timestamp for 10min TTL
3. When merging pulls, queries `GetPullingImagesForNode()` to resolve `__pulling__` sentinel
4. Falls back to `GetWaitingImagesForNode()` if no actively-pulling image
5. Updates PullStatus.ImageRef with discovered image name

**State Management:**

- Server holds all pull state in `pulls: map[string]*PullStatus`
- Key format: `node:imageRef` (or `node:__merged__` for unresolved merges)
- Completed pulls retained for HistoryTTL (default 30 min)
- Stale uncompleted pulls force-completed after stalePullTimeout (10 min no reports)
- Cleanup loop runs every 1 minute to evict expired entries

**SSE Client Flow:**

1. Frontend calls `fetch('/api/v1/pulls')` for initial state snapshot
2. Frontend opens EventSource to `/api/v1/events`
3. Server sends current pull state snapshot on SSE handshake (100+ events possible)
4. Each PullEvent broadcast includes full PullStatus with all layers
5. Frontend merges updates by pull.id (creates new row or replaces existing)
6. Handles disconnection with 3-second reconnect backoff

## Key Abstractions

**PullStatus:**
- Purpose: Canonical representation of one image pull's aggregated state
- Examples: `internal/model/event.go` lines 24-41
- Pattern: Includes layer array, pod correlations, progress metrics; JSON serializable

**LayerStatus:**
- Purpose: Per-layer download progress within a pull
- Examples: `internal/model/event.go` lines 44-55
- Pattern: Mirrors containerd's per-content-object tracking; aggregated by PullStatus

**RateCalculator:**
- Purpose: Sliding-window download speed calculation
- Examples: `internal/model/rate.go`
- Pattern: Keeps 2+ samples per window to calculate bytes/second; ETA derived from rate

**mergeDigestPulls():**
- Purpose: Transform raw containerd content digests into logical image pulls
- Examples: `internal/server/server.go` lines 299-330
- Pattern: Pure function; groups digest-type refs into synthetic `__pulling__` pull; preserves layer data

**Sentinel `__pulling__`:**
- Purpose: Placeholder for pulls downloading under content digests before image name resolved
- Examples: Used throughout `internal/server/server.go` for key matching
- Pattern: Replaced with actual imageRef once PodWatcher discovers image name

## Entry Points

**Agent Entry Point:**
- Location: `cmd/pulltrace-agent/main.go`
- Triggers: Pod creation (DaemonSet ensures one per node)
- Responsibilities: Read config from env, connect to containerd, start polling loop, handle shutdown signals

**Server Entry Point:**
- Location: `cmd/pulltrace-server/main.go`
- Triggers: Pod creation, typically one replica
- Responsibilities: Read config from env, init PodWatcher, embed web UI, start HTTP + metrics servers

**Frontend Entry Point:**
- Location: `web/src/main.jsx`
- Triggers: Browser navigation to server HTTP root
- Responsibilities: Hydrate React tree from HTML template, mount App component

**HTTP Handlers:**
- `/api/v1/report`: Receive AgentReport, validate, aggregate, broadcast
- `/api/v1/pulls`: Snapshot endpoint, return current state (initial fetch)
- `/api/v1/events`: SSE stream, send updates continuously
- `/healthz`, `/readyz`: Liveness/readiness probes
- `/`: Serve React app (Vite build output embedded in binary)

## Error Handling

**Strategy:** Graceful degradation; errors are logged but don't stop the system.

**Patterns:**

- **Agent → Server:** If POST fails (network error, auth failure, server down), log and retry on next tick
- **Server → PodWatcher:** If K8s API unavailable, pod correlation is skipped; pulls still tracked, just without pod metadata
- **Server → SSE:** If SSE client is slow (doesn't read), message is dropped (non-blocking channel send with `default:` clause)
- **Rate Limiting:** Nodes reporting too frequently (>500ms window) receive 429; agent retries on next tick
- **Capacity Limits:** At maxActivePulls (10k) or maxSSEClients (256), new entries are rejected with warn logs

## Cross-Cutting Concerns

**Logging:**
- Structured JSON logs via `slog` with explicit levels (info, debug, warn, error)
- Agent logs: poll progress, report transmission
- Server logs: aggregation, completion, pod resolution, SSE client tracking
- All logs written to stdout for container log aggregation

**Validation:**
- AgentReport: NodeName must be non-empty and ≤253 chars (DNS constraint)
- Socket path: Agent validates containerd socket prefix to prevent TOCTOU attacks
- Bearer token: Optional per-agent authentication via `PULLTRACE_AGENT_TOKEN` env var
- Rate limiting: Per-node window-based to prevent report flooding

**Authentication:**
- Bearer token authentication for agent → server POST (opt-in via env var)
- No authentication for frontend (assumes behind ingress auth or private network)
- Kubernetes in-cluster auth for server → K8s API (uses ServiceAccount mounted in pod)

**Security Headers:**
- CSP, X-Frame-Options, X-Content-Type-Options set on all HTTP responses
- Metrics endpoint exposed on separate port (9090) for network isolation via NetworkPolicy
