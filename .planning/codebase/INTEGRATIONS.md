# External Integrations

**Analysis Date:** 2025-02-23

## APIs & External Services

**Agent-to-Server Communication:**
- Endpoint: `POST /api/v1/report` - AgentReport JSON submission
  - Client: `github.com/d44b/pulltrace/internal/agent/agent.go`
  - Auth: Optional Bearer token via `Authorization: Bearer $PULLTRACE_AGENT_TOKEN`
  - Body: JSON-encoded `model.AgentReport` (nodeName, timestamp, pulls array)

**Server Event Stream:**
- Endpoint: `GET /api/v1/events` - Server-Sent Events (SSE) stream
  - Consumed by: `web/src/hooks.js` usePulls() via EventSource API
  - Events: PullEvent JSON objects with pull and layer status updates
  - Max concurrent clients: 256 (enforced by server)

**REST API Endpoints:**
- `GET /api/v1/pulls` - Initial state fetch of all active pulls (web/src/hooks.js)
- `GET /metrics` - Prometheus metrics endpoint (port 9090)

## Data Storage

**Databases:**
- None - Pulltrace is stateless and does not persist data to external databases
- In-memory state only: Active pulls stored in server RAM with TTL-based expiration
- History TTL: 30 minutes (configurable via PULLTRACE_HISTORY_TTL)

**File Storage:**
- Local filesystem only - Embedded UI assets served from `web/dist/` (bundled in Dockerfile.server)
- No external object storage (S3, GCS, etc.)

**Caching:**
- In-memory caching of Kubernetes pod metadata by PodWatcher (internal/k8s/podwatcher.go)
- Pod-to-image correlation cached per node
- Pulling images tracked with 10-minute TTL (const pullingImageTTL in podwatcher.go)

## Authentication & Identity

**Auth Provider:**
- Custom token-based authentication (optional)
- Mechanism: Shared secret via `PULLTRACE_AGENT_TOKEN` environment variable
- Implementation: Bearer token in Authorization header, constant-time comparison (crypto/subtle.ConstantTimeCompare)
- Scope: Agent-to-server API authentication only (internal cluster communication)
- No external OAuth2/OIDC provider integration

**Kubernetes In-Cluster Auth:**
- Uses `k8s.io/client-go/rest.InClusterConfig()` - Kubernetes ServiceAccount token mounting
- Requires ClusterRole with permissions for:
  - `pods` list, watch
  - `events` watch
  - Core API access

## Monitoring & Observability

**Prometheus Metrics (port 9090):**
- `pulltrace_pulls_active` (Gauge) - Currently active image pulls
- `pulltrace_pulls_total` (Counter) - Total pulls observed
- `pulltrace_pull_duration_seconds` (Histogram) - Pull duration distribution (buckets: 1s, 5s, 10s, 30s, 60s, 120s, 300s, 600s)
- `pulltrace_pull_bytes_total` (Counter) - Total bytes downloaded
- `pulltrace_pull_errors_total` (Counter) - Pull error count (defined but not currently incremented)
- `pulltrace_agent_reports_total` (Counter) - Reports received from agents
- `pulltrace_sse_clients_active` (Gauge) - Active SSE connections
- Client: `github.com/prometheus/client_golang/prometheus` with promauto registration
- Endpoint format: `/metrics` via promhttp.Handler()

**Logging:**
- Framework: Go 1.21+ structured logging (log/slog)
- Levels: debug, info, warn, error
- Configuration: PULLTRACE_LOG_LEVEL environment variable
- Outputs: stdout (structured JSON format)
- No external logging service integration (Datadog, Stackdriver, etc.)

**Error Tracking:**
- No external error tracking service (Sentry, Bugsnag, etc.)
- Errors logged to stdout via slog

**Distributed Tracing:**
- OpenTelemetry framework available (go.opentelemetry.io/otel v1.31.0) but not actively used
- No active tracing integration or exporter configured

## CI/CD & Deployment

**Hosting:**
- Kubernetes - Target deployment platform (Helm chart: charts/pulltrace/)
- Container Registry: GitHub Container Registry (ghcr.io/d44b/pulltrace-agent, ghcr.io/d44b/pulltrace-server)

**Deployment:**
- Helm 3 chart for Kubernetes deployment
- DaemonSet for agents (charts/pulltrace/templates/agent-daemonset.yaml)
- Deployment for server (charts/pulltrace/templates/server-deployment.yaml)
- ConfigMap for configuration (templates/configmap.yaml)
- Secret for authentication token (templates/secret.yaml)
- RBAC: ServiceAccount, ClusterRole, ClusterRoleBinding (templates/rbac.yaml, templates/serviceaccount.yaml)
- NetworkPolicy template available (templates/networkpolicy.yaml)
- Ingress template available (templates/ingress.yaml, disabled by default)

**CI Pipeline:**
- No external CI system detected (no .github/workflows, .gitlab-ci.yml, etc. in scope)
- Build target: Multi-stage Docker Dockerfile.server and Dockerfile.agent
- Kubernetes API version: v0.31.4 (supports current stable K8s APIs)

## Runtime Container Integration

**containerd Integration:**
- Socket communication: `/run/containerd/containerd.sock` or `/var/run/containerd/containerd.sock`
- Protocol: containerd v2 gRPC API (github.com/containerd/containerd/v2)
- What's accessed:
  - Content store for layer tracking (github.com/containerd/containerd/v2/core/content)
  - Pull state and metadata per image
- Permissions: Requires root UID to access socket (agent DaemonSet runs as root)
- Failure mode: Agent cannot start without containerd socket access

**Kubernetes API Integration:**
- Client library: k8s.io/client-go v0.31.4
- In-cluster auth: ServiceAccount token (pod.spec.serviceAccount)
- API groups used:
  - core/v1 - Pod and Event resources
  - meta/v1 - Standard Kubernetes metadata
- Operations:
  - List and Watch pods across specified namespaces (PULLTRACE_WATCH_NAMESPACES)
  - Watch Events for "Pulling" image pull events from kubelet
  - Metadata correlation: image reference → node + namespace
- Failure mode: Server degrades gracefully (optional pod-to-image correlation feature disabled)

## Webhooks & Callbacks

**Incoming:**
- `POST /api/v1/report` - Agent reports (described under APIs & External Services)
- No external webhook consumers (no outbound POST to external systems)

**Outgoing:**
- None - Pulltrace does not send webhooks or callbacks to external systems
- No integrations with Slack, PagerDuty, external APIs, etc.

## Network Communication

**Inbound (Server):**
- HTTP API: Port 8080 (default)
- Prometheus metrics: Port 9090 (default)
- SSE connections from frontend (same HTTP port 8080)
- Agent reports (same HTTP port 8080)

**Outbound (Agent):**
- HTTP POST to server at PULLTRACE_SERVER_URL (configurable, cluster-internal)
- containerd socket access (local UNIX socket, not network)

**In-Cluster Networking:**
- Service: `pulltrace-server` ClusterIP (default port 8080)
- Agent-to-server traffic: Internal to cluster only (no external egress)
- Kubernetes API server access: Standard in-cluster API endpoint (kubernetes.default.svc)

---

*Integration audit: 2025-02-23*
