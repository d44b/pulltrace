# Security Policy — Pulltrace

## Threat Model

### Overview

Pulltrace monitors container image pull progress on Kubernetes nodes. Its agent
runs as a DaemonSet with read-only access to the containerd runtime socket. The
server aggregates data and exposes it via an HTTP API and web UI.

**Pulltrace has no built-in authentication.** Access control must be enforced at
the network level (Kubernetes NetworkPolicy, ingress auth proxy, service mesh).

### Assets

| Asset | Sensitivity | Notes |
|-------|-------------|-------|
| Cluster inventory data | HIGH | Node names, pod names, namespaces, image references |
| containerd socket | CRITICAL | Read access exposes all container metadata on the node |
| Kubernetes ServiceAccount token (server) | HIGH | Cluster-wide pod list/watch |
| Agent pods | HIGH | Run as root on every node |

### Entry Points

| Entry Point | Authentication | Risk |
|-------------|---------------|------|
| `POST /api/v1/report` | None | Fake report injection, DoS |
| `GET /api/v1/pulls` | None | Cluster inventory disclosure |
| `GET /api/v1/events` (SSE) | None | Real-time inventory stream |
| `GET /` (Web UI) | None | UI access |
| `GET /metrics` (port 9090) | None | Operational metrics |
| containerd UNIX socket | Host UID 0 | Node-level container metadata |

### Trust Boundaries

```
┌─────────────────────────────────────────────────────────┐
│  External network (untrusted)                           │
│  ┌───────────────────────────────────────────────────┐  │
│  │  Kubernetes cluster network                       │  │
│  │  ┌─────────────────────────────────────────────┐  │  │
│  │  │  pulltrace namespace                        │  │  │
│  │  │  ┌──────────┐        ┌──────────────────┐   │  │  │
│  │  │  │  Agent   │──HTTP──│  Server           │   │  │  │
│  │  │  │ (root)   │        │  (non-root)       │   │  │  │
│  │  │  └────┬─────┘        └──────┬───────────┘   │  │  │
│  │  │       │ socket              │ K8s API       │  │  │
│  │  └───────┼─────────────────────┼───────────────┘  │  │
│  │          ▼                     ▼                   │  │
│  │   [containerd]          [kube-apiserver]           │  │
│  └───────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────┘
```

### Threats & Mitigations

#### 1. Public Internet Exposure (UI/API)

**Risk:** If ingress is enabled without auth, cluster inventory is publicly visible.

**Mitigations:**
- Ingress disabled by default
- Service type defaults to ClusterIP; changing to LoadBalancer/NodePort requires
  `server.service.exposureAcknowledged=true` (Helm validation)
- values.yaml documents the risk with warnings
- No CORS headers (same-origin only) — cross-origin access blocked by browsers
- CSP header restricts resource loading to same-origin only
- `X-Frame-Options: DENY` prevents clickjacking
- `X-Content-Type-Options: nosniff` prevents MIME sniffing

#### 2. Cluster-Internal RBAC Abuse

**Risk:** Any pod in the cluster can reach the server via ClusterIP and read/inject data.

**Mitigations:**
- Deploy NetworkPolicy to restrict access to the pulltrace namespace
- Server validates report payloads and applies rate limiting per node
- RBAC: server SA has read-only access to pods and events (no nodes, no secrets, no writes)
- Node name validated (max 253 chars, non-empty)

#### 3. Agent Pod Compromise (Node Escape)

**Risk:** Agent runs as root to read containerd socket. If compromised, attacker has
root on the node and access to container metadata.

**Mitigations:**
- `allowPrivilegeEscalation: false`
- All capabilities dropped (`drop: ALL`)
- `readOnlyRootFilesystem: true`
- `seccompProfile: RuntimeDefault`
- `hostNetwork: false`, `hostPID: false`, `hostIPC: false` (explicit)
- Socket mounted read-only
- Agent SA token not mounted (`automountServiceAccountToken: false`)
- Runtime socket requires explicit opt-in (`runtimeSocket.enabled` + `risksAcknowledged`)
- Agent validates socket path prefix (`/run/containerd/` or `/var/run/containerd/`)

#### 4. Data Leak (Inventory Reconnaissance)

**Risk:** API responses contain node names, pod names, namespaces, and image references.

**Mitigations:**
- ClusterIP service (not externally reachable by default)
- Ingress disabled by default with security warnings
- LoadBalancer/NodePort require explicit acknowledgment
- Recommend NetworkPolicy to restrict cluster-internal access

#### 5. DoS via Event Flood

**Risk:** Attacker floods `POST /api/v1/report` to exhaust server memory.

**Mitigations:**
- Request body limited to 1 MiB (`MaxBytesReader`)
- Per-node rate limiting (1 report/second, bounded to 1024 tracked nodes)
- Rate limiter entries auto-cleaned every 60 seconds
- Active pulls map capped at 10,000 entries; new pulls rejected at capacity
- Stale in-progress pulls force-completed after 10 minutes without update
- Completed pulls cleaned up after TTL (default 30 minutes)
- HTTP timeouts: `ReadHeaderTimeout: 10s`, `ReadTimeout: 30s`, `IdleTimeout: 120s`
- Max header size: 64 KiB
- SSE client connections capped at 256
- Prometheus metrics use no high-cardinality labels (no per-node label)
- Resource limits in Kubernetes (CPU + memory)

**WriteTimeout note:** The HTTP server has `WriteTimeout=0` because SSE connections
are long-lived. Slowloris protection relies on `ReadHeaderTimeout` and
`ReadTimeout`. SSE resource exhaustion is bounded by the 256 client cap and
non-blocking channel sends that drop slow clients.

#### 6. Supply Chain Attack

**Risk:** Malicious base image or dependency.

**Mitigations:**
- Distroless base images pinned by SHA256 digest (no tag-only references)
- `.dockerignore` excludes `.git`, `.env*`, `charts/`, and other non-build files
- Static Go binaries (CGO_ENABLED=0)
- `go.sum` committed to version control (dependency checksums verified)
- Minimal Go dependencies
- Multi-stage builds (build tools not in final image)

## Component Privileges

### pulltrace-agent (DaemonSet)

| Privilege | Value | Reason |
|---|---|---|
| Host path volume | containerd socket (read-only) | Required for `content.ListStatuses` |
| `runAsUser` | 0 (root) | Required for socket access |
| `allowPrivilegeEscalation` | false | Hardened |
| `readOnlyRootFilesystem` | true | Agent writes no files |
| `capabilities` | drop ALL | No capabilities needed |
| `seccompProfile` | RuntimeDefault | Default seccomp filter |
| `automountServiceAccountToken` | false | Agent does not call K8s API |
| `hostPID` / `hostNetwork` / `hostIPC` | false (explicit) | Not required |
| `privileged` | false | Not required |
| Socket path validation | `/run/containerd/` or `/var/run/containerd/` only | Prevents redirect to other sockets |

### pulltrace-server (Deployment)

| Privilege | Value | Reason |
|---|---|---|
| `runAsUser` | 65534 (nobody) | Non-root |
| `runAsNonRoot` | true | Enforced |
| `allowPrivilegeEscalation` | false | Hardened |
| `readOnlyRootFilesystem` | true | Server writes no files |
| `capabilities` | drop ALL | No capabilities needed |
| `seccompProfile` | RuntimeDefault | Default seccomp filter |
| `hostPID` / `hostNetwork` / `hostIPC` | false (explicit) | Not required |
| RBAC | `list`, `watch` pods and events | Pod correlation (no nodes, no secrets, no writes) |
| No host paths | — | Server does not access the host filesystem |

## Containerd Socket Access

**Risk:** Access to the containerd socket grants significant node-level capabilities.
A process with write access can create, modify, and delete containers.

**Mitigations:**

1. **Opt-in required.** Socket is NOT mounted by default. You must set both
   `agent.runtimeSocket.enabled=true` AND `agent.runtimeSocket.risksAcknowledged=true`.
   Helm will fail if you enable the socket without the acknowledgment flag.

2. **Path validation.** The agent validates that the socket path starts with
   `/run/containerd/` or `/var/run/containerd/`. This prevents misconfiguration
   that could point the agent at other UNIX sockets (e.g., Docker, kubelet).

3. **Read-only operations.** The agent only calls `content.ListStatuses` and
   `content.Info`. No write operations.

4. **Read-only mount.** The socket volume is mounted with `readOnly: true`.

5. **PodSecurity Standards.** The `pulltrace` namespace must be labeled:
   ```
   kubectl label namespace pulltrace pod-security.kubernetes.io/enforce=privileged
   ```

## Network Security

- Agent-to-server communication uses cluster-internal HTTP. For encryption,
  use a service mesh with mTLS.
- The server API and UI have no authentication. If exposed outside the cluster,
  use an authenticating proxy (e.g., oauth2-proxy, ingress auth annotations).
- The Prometheus metrics endpoint (`/metrics`) is on a separate port (9090).
  Restrict access via NetworkPolicy.

## Resource Bounds

The server enforces the following limits to prevent memory exhaustion:

| Resource | Limit | Behavior at limit |
|----------|-------|-------------------|
| Rate limiter entries | 1,024 nodes | New unknown nodes rejected (HTTP 429) |
| Active pulls | 10,000 | New pulls from reports silently dropped |
| SSE clients | 256 | New connections rejected (HTTP 503) |
| Request body | 1 MiB | Request rejected (HTTP 413) |
| Request headers | 64 KiB | Request rejected |
| Stale pull timeout | 10 minutes | Force-completed, then cleaned after TTL |
| Completed pull TTL | 30 minutes (configurable) | Deleted from memory |

Rate limiter entries are cleaned up every 60 seconds (entries older than 1 minute
are removed).

## Secure Deployment Checklist

1. **Explicitly opt in to runtime socket:**
   ```yaml
   agent:
     runtimeSocket:
       enabled: true
       risksAcknowledged: true
   ```

2. **Deploy NetworkPolicy** to restrict server access to agent pods only.

3. **If exposing via ingress**, use an authenticating proxy.

4. **If using LoadBalancer/NodePort**, acknowledge exposure:
   ```yaml
   server:
     service:
       type: LoadBalancer
       exposureAcknowledged: true
   ```

5. **Restrict `watchNamespaces`** to only the namespaces you need to monitor.

6. **Review RBAC** — the server ClusterRole grants read-only pod access
   cluster-wide. Scope with namespaced Roles if tighter control is needed.

## Supply Chain

- Container images use distroless base images pinned by SHA256 digest.
- `.dockerignore` prevents `.git`, `.env*`, and non-build files from entering
  the Docker build context.
- Go binaries are statically compiled (CGO_ENABLED=0).
- Go dependencies managed via `go.mod` with checksums in `go.sum`.
- Node.js dependencies managed via `package-lock.json` with `npm ci`.
- Multi-stage builds ensure build tools are not in the final image.

## Reporting Vulnerabilities

If you discover a security vulnerability, please report it responsibly:

1. **Do NOT open a public GitHub issue.**
2. Email **security@d44b.dev** with:
   - A description of the vulnerability
   - Steps to reproduce
   - Potential impact
   - Suggested fixes (optional)
3. We will acknowledge receipt within 48 hours.
4. Critical issues: fix within 7 days.
5. Allow 90 days for a fix before public disclosure.

## Supported Versions

| Version | Supported |
|---|---|
| 0.1.x | Yes |
