# Technology Stack

**Analysis Date:** 2025-02-23

## Languages

**Primary:**
- Go 1.22.0 - Backend server and agent applications (cmd/pulltrace-server, cmd/pulltrace-agent)
- JavaScript (ES Module) - React frontend UI components

**Secondary:**
- TypeScript - Type annotations (no tsconfig detected, using JSDoc in some files)
- YAML - Kubernetes Helm templates and configuration

## Runtime

**Environment:**
- Go 1.22 (required in go.mod) - Note: Local development requires Go 1.22+; Go 1.18 cannot parse 1.22.0 directive
- Node.js 22 (Alpine) - Frontend build stage in Dockerfile.server
- Container runtime: distroless/static-debian12 (non-root, minimal attack surface)

**Package Manager:**
- Go modules (go.mod, go.sum) - Lockfile present with 95+ dependencies
- npm (web/package.json, web/package-lock.json) - Lockfile present

## Frameworks

**Core Backend:**
- Standard Go net/http - HTTP server for API endpoints and SSE
- Prometheus client_golang v1.20.5 - Metrics collection and export

**Frontend:**
- React 18.3.1 - UI component library and state management (web/src/hooks.js, web/src/components/)
- Vite 6.0.0 - Build tool and dev server (web/vite.config.js) with dev proxy to http://localhost:8080

**Build:**
- Docker multi-stage builds - Dockerfile.server (Node + Go), Dockerfile.agent (Go only)
- Helm v2 charts - Kubernetes deployment templates (charts/pulltrace/)

## Key Dependencies

**Critical:**
- containerd/containerd v2.0.4 - Runtime API for monitoring image pulls on nodes
- k8s.io/client-go v0.31.4 - Kubernetes API client for pod/event watching
- k8s.io/api v0.31.4 - Kubernetes API types
- k8s.io/apimachinery v0.31.4 - Kubernetes common types and utilities

**Infrastructure:**
- prometheus/client_golang v1.20.5 - Metrics export (gauge, counter, histogram types)
- google.golang.org/grpc v1.68.1 - gRPC transport for containerd client communication
- golang.org/x/oauth2 v0.23.0 - OAuth2 support (pulled in by k8s.io/client-go)

**Observability:**
- go.opentelemetry.io/otel v1.31.0 - OpenTelemetry tracing framework (pulled in by containerd)
- sirupsen/logrus v1.9.3 - Structured logging (used by containerd dependencies)

**Go Standard Library Usage:**
- context - Cancellation and deadline propagation
- encoding/json - JSON marshaling for API responses
- log/slog - Structured logging (Go 1.21+)
- net/http - HTTP server and client
- sync - Mutexes and synchronization primitives
- time - Duration and ticker operations

## Configuration

**Environment:**

Backend server configuration via environment variables:
- `PULLTRACE_HTTP_ADDR` - Server HTTP listen address (default: ":8080")
- `PULLTRACE_METRICS_ADDR` - Prometheus metrics listen address (default: ":9090")
- `PULLTRACE_LOG_LEVEL` - Logging level: debug, info, warn, error (default: "info")
- `PULLTRACE_WATCH_NAMESPACES` - Comma-separated Kubernetes namespaces to watch (default: all)
- `PULLTRACE_HISTORY_TTL` - Pull history retention duration (default: "30m")
- `PULLTRACE_AGENT_TOKEN` - Shared secret for agent-to-server authentication (optional)

Agent configuration via environment variables:
- `PULLTRACE_NODE_NAME` - Node identifier (required, typically pod.spec.nodeName)
- `PULLTRACE_SERVER_URL` - Server endpoint URL (required, e.g., http://pulltrace-server:8080)
- `PULLTRACE_CONTAINERD_SOCKET` - containerd socket path (default: "/run/containerd/containerd.sock")
- `PULLTRACE_LOG_LEVEL` - Logging level: debug, info, warn, error (default: "info")
- `PULLTRACE_REPORT_INTERVAL` - Pull state report interval (default: "1s")
- `PULLTRACE_AGENT_TOKEN` - Shared secret for authentication (must match server if set)

Frontend configuration (Vite):
- Dev proxy target: `/api` → `http://localhost:8080`
- Build output: `web/dist/`
- Server port (dev): 3030

**Build:**
- Dockerfile.server - Multi-stage: Node 22 Alpine (UI build) → Go 1.22 Alpine (binary build) → distroless/static-debian12 (runtime)
- Dockerfile.agent - Go 1.22 Alpine build → distroless/static-debian12 runtime (no UI dependency)
- Both built with CGO_ENABLED=0, -ldflags="-s -w" (fully static, stripped binaries)

## Platform Requirements

**Development:**
- Go 1.22.0 or later (cannot build locally with Go 1.18 due to go.mod version directive)
- Node.js 22+ (for npm in web/)
- Docker (for cross-platform builds)
- containerd socket access (local testing of agent)

**Production:**
- Kubernetes 1.20+ (tested with api v0.31.4, supports standard K8s APIs)
- containerd runtime (agent must access /run/containerd/containerd.sock or /var/run/containerd/containerd.sock)
- RBAC authorization enabled (Helm chart creates ServiceAccount and required roles)
- Linux x86_64 (amd64) or ARM64 capable hosts (distroless images support both)

**Deployment:**
- Helm 3.x (Chart apiVersion: v2)
- kubectl (for Helm installation)

---

*Stack analysis: 2025-02-23*
