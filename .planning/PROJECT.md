# Pulltrace

## What This Is

Pulltrace is a real-time Kubernetes image pull monitor. A DaemonSet agent on each node reads containerd's content store, reports layer-by-layer pull progress to a central server, which aggregates multi-node state and streams live updates to a React dashboard via SSE. The project is now publicly released at v0.1.0 with a full open-source wrapper: documentation site, Helm repository, GitHub Release, and community files.

## Core Value

A DevOps engineer deploying to Kubernetes can see exactly which images are pulling, how fast, which pods are waiting, and when they'll be ready — without kubectl exec or log digging.

## Current State (v0.1.0)

**Shipped:** 2026-02-23
**Repo:** https://github.com/d44b/pulltrace
**Docs:** https://d44b.github.io/pulltrace/
**Install:** `helm repo add pulltrace https://d44b.github.io/pulltrace/charts && helm install pulltrace pulltrace/pulltrace -n pulltrace --create-namespace`

**Tech stack:**
- Go 1.22 (backend: agent + server)
- React + Vite (frontend, embedded in server binary)
- Helm 3 chart (DaemonSet + Deployment + RBAC)
- MkDocs Material 9.7.2 (documentation site)
- GitHub Actions CI/CD (lint/test/docker push/helm release/github release)
- GHCR (Docker images + OCI Helm chart), GitHub Pages (docs + classic Helm repo)

**Approximate LOC:** ~3,100 lines added in v0.1.0 release work (docs, CI, community files, bug fixes)

## Requirements

### Validated

<!-- Shipped and confirmed — v0.1.0 -->

- ✓ Agent DaemonSet reads containerd socket and reports pull state every 1s — existing (pre-v0.1)
- ✓ Server aggregates multi-node AgentReports into unified PullStatus map — existing (pre-v0.1)
- ✓ SSE stream broadcasts PullEvent updates to all connected clients — existing (pre-v0.1)
- ✓ React UI renders live pull rows with progress bars, speed gauge, filters — existing (pre-v0.1)
- ✓ Layer-level drill-down in UI (LayerDetail component) — existing (pre-v0.1)
- ✓ Pod correlation via Kubernetes API (identifies waiting pods for each pull) — existing (pre-v0.1)
- ✓ Prometheus metrics on port 9090 (pulls active, total, duration, bytes, SSE clients) — existing (pre-v0.1)
- ✓ Optional Bearer token auth for agent → server communication — existing (pre-v0.1)
- ✓ Helm chart for full Kubernetes deployment (DaemonSet + Deployment + RBAC) — existing (pre-v0.1)
- ✓ Docker images (agent + server) published to GHCR on tag via CI — existing (pre-v0.1)
- ✓ Helm chart published to GHCR OCI registry on tag via CI — existing (pre-v0.1)
- ✓ Apache 2.0 license — existing (pre-v0.1)
- ✓ SECURITY.md threat model — existing (pre-v0.1)
- ✓ Architecture Decision Records (ADRs) — existing (pre-v0.1)
- ✓ CONTRIBUTING.md with Go 1.22 Docker workaround, build instructions, PR guidelines — v0.1
- ✓ CHANGELOG.md in keep-a-changelog format with [0.1.0] entry — v0.1
- ✓ `pulltrace_pull_errors_total` counter incremented when pull completes with non-empty Error — v0.1
- ✓ Layer `bytesPerSec` and `mediaType` populated server-side in PullStatus.Layers — v0.1
- ✓ GitHub repository topics (kubernetes, monitoring, containers, helm, containerd) and description — v0.1
- ✓ MkDocs Material docs site at https://d44b.github.io/pulltrace/ with installation, configuration, architecture pages — v0.1
- ✓ Classic Helm repository: `helm repo add pulltrace https://d44b.github.io/pulltrace/charts` — v0.1
- ✓ CI auto-deploys docs to GitHub Pages on push to main — v0.1
- ✓ CI creates GitHub Release with body, install commands, and .tgz on semver tag push — v0.1
- ✓ All GHCR packages (pulltrace-agent, pulltrace-server, charts/pulltrace) public — v0.1
- ✓ v0.1.0 released: Docker images, Helm chart (classic + OCI), GitHub Release all live — v0.1

### Active

<!-- v0.2.0 candidates — not yet started -->

- [ ] Artifact Hub listing (`artifacthub-repo.yml`) for Helm chart discoverability
- [ ] Docker image SBOMs and cosign signatures attached to GitHub Release
- [ ] Layer-level SSE events emitted (`layer.started`, `layer.progress`, `layer.completed`)
- [ ] Social preview image uploaded to GitHub repository Settings (no API; manual)
- [ ] Fix CONTRIBUTING.md dead link (line 147 references removed CODE_OF_CONDUCT.md)

### Out of Scope

- Persistent storage (Redis/etcd for pull history) — in-memory with 30m TTL is sufficient
- UI authentication — assumes ingress-level auth or private cluster network
- Mobile app / CLI — web UI is the interface
- Multi-cluster federation — single cluster per deployment
- Slack/PagerDuty alerting — Prometheus integration covers this use case
- OpenTelemetry tracing — adds operational complexity without clear demand signal
- `stalePullTimeout` via env var — current 30m TTL acceptable for v0.1.0 use cases

## Constraints

- **Tech stack:** Go 1.22, React + Vite, Helm 3, GitHub Actions — no new languages or platforms
- **Hosting:** GitHub free tier only — GitHub Pages for docs + GHCR for images/charts
- **No external services:** No paid CI, no external doc hosting (Netlify, Vercel, etc.)
- **Go version:** Local machines with Go < 1.22 must build via Docker (`golang:1.22-alpine`)

## Key Decisions

| Decision | Rationale | Outcome |
|----------|-----------|---------|
| OCI helm chart registry (GHCR) | Already wired in CI; free, authenticated | ✓ Good — shipping OCI + classic |
| GitHub Pages for docs | Free, co-located with repo | ✓ Good — MkDocs Material live |
| Apache 2.0 license | Permissive, compatible with containerd and client-go deps | ✓ Good |
| MkDocs Material over Hugo/Jekyll | Dominant K8s ecosystem standard; Python-only, simpler setup | ✓ Good |
| `peaceiris/actions-gh-pages` with `keep_files: true` | Only safe co-deployment pattern for docs + Helm index on same gh-pages branch | ✓ Good — no mutual destruction observed |
| `helm/chart-releaser-action` NOT used | Creates duplicate releases, forces `index.yaml` to gh-pages root | ✓ Good — avoided |
| `softprops/action-gh-release@v2` for GitHub Release | `chart-releaser-action` replacement; inline body avoids CHANGELOG dump | ✓ Good |
| Inline release body (not `body_path`) | Avoids verbatim [Unreleased] section and 23-line CHANGELOG header in release body | ✓ Good |
| PullErrors.Inc() in processReport() only (not cleanup()) | Agent is authoritative error source; cleanup() is a timeout safety net | ✓ Good |
| Layer rate keys: `key+":layer:"+digest` compound format | Enables HasPrefix bulk-delete on pull eviction in cleanup() | ✓ Good |
| `cancel-in-progress: false` on deploy-gh-pages concurrency | Never abort an in-flight gh-pages push (leaves branch in broken state) | ✓ Good |
| `needs: [helm-release]` for github-release job | Release appears only after Helm index is live on gh-pages | ✓ Good |
| v0.1.0 as first public release | Feature-complete core; stabilize and document before adding features | ✓ Good — shipped |
| No `--merge` for v0.1.0 helm repo index | No prior index.yaml; use `--merge` for v0.2.0+ to preserve prior chart versions | — Pending: must remember for v0.2.0 |

---
*Last updated: 2026-02-23 after v0.1 milestone*
