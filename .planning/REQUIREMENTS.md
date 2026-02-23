# Requirements: Pulltrace v0.1.0 Open Source Release

**Defined:** 2026-02-23
**Core Value:** A DevOps engineer can install Pulltrace with a single `helm install` command, find docs explaining how it works, and trust it as a credible open source project.

## v1 Requirements

Requirements for the v0.1.0 public release. Each maps to roadmap phases.

### Community Files

- [ ] **COMM-01**: Project has `CONTRIBUTING.md` with local dev setup (Go 1.22 + Docker workaround), build instructions (`make`/`docker build`), and PR guidelines
- [ ] **COMM-02**: Project has `CHANGELOG.md` in keep-a-changelog format with a `[0.1.0]` entry summarizing what the release includes
- [ ] **COMM-03**: Project has `CODE_OF_CONDUCT.md` (Contributor Covenant v2.1)

### Documentation Site

- [ ] **DOCS-01**: User can browse a documentation site at `https://d44b.github.io/pulltrace/` (GitHub Pages, MkDocs Material theme)
- [ ] **DOCS-02**: Docs site has an Installation page with the `helm repo add` command, `helm install` command, and prerequisites listed
- [ ] **DOCS-03**: Docs site has a Configuration reference page covering all environment variables for both server and agent
- [ ] **DOCS-04**: Docs site has an Architecture page explaining the agent+server+UI data flow with a diagram
- [ ] **DOCS-05**: CI workflow automatically deploys updated docs to GitHub Pages on every push to `main`

### Helm Chart Discoverability

- [ ] **HELM-01**: `helm repo add pulltrace https://d44b.github.io/pulltrace/charts` succeeds and returns success confirmation
- [ ] **HELM-02**: `helm install pulltrace pulltrace/pulltrace` installs successfully from the classic Helm repository index
- [ ] **HELM-03**: `index.yaml` is served from `https://d44b.github.io/pulltrace/charts/index.yaml`
- [ ] **HELM-04**: CI job on semver tag packages Helm chart `.tgz`, runs `helm repo index --merge`, and pushes to `gh-pages` branch `/charts/` path without overwriting the docs

### Release Infrastructure

- [ ] **REL-01**: `ci.yml` has `contents: write` permission (currently `contents: read` — blocks GitHub Release creation)
- [ ] **REL-02**: Pushing `git tag v0.1.0` triggers CI to create a GitHub Release with title, body (what's new, install commands, compatibility), and changelog link
- [ ] **REL-03**: All three GHCR packages (`pulltrace-agent`, `pulltrace-server`, `charts/pulltrace`) are set to public visibility
- [ ] **REL-04**: v0.1.0 tag is pushed and all artifacts are live: Docker images at GHCR, Helm chart at `ghcr.io/d44b/charts/pulltrace` (OCI) and `d44b.github.io/pulltrace/charts` (classic), GitHub Release created

### Bug Fixes

- [ ] **FIX-01**: `pulltrace_pull_errors_total` Prometheus counter is incremented when a pull completes with a non-empty `PullStatus.Error` field
- [ ] **FIX-02**: Server populates `layer.bytesPerSec` and `layer.mediaType` in `PullStatus.Layers` so the LayerDetail component can display speed and content type

### Repository Metadata

- [ ] **META-01**: GitHub repository has description set, topics added (`kubernetes`, `monitoring`, `containers`, `helm`, `containerd`), and a social preview image (OG image)

## v2 Requirements

Deferred to a future release. Tracked but not in current roadmap.

### Discoverability

- **DISC-01**: Artifact Hub listing for the Helm chart (`artifacthub-repo.yml` committed)
- **DISC-02**: OCI install path prominently documented alongside classic Helm path
- **DISC-03**: Custom domain for docs site (e.g. `docs.pulltrace.dev`) with DNS setup

### Supply Chain Security

- **SC-01**: Docker image SBOMs generated and attached to GitHub Release
- **SC-02**: Docker images signed with cosign (keyless, Sigstore)
- **SC-03**: Release artifact checksums (`sha256sum`) attached to GitHub Release

### Observability Improvements

- **OBS-01**: Layer-level SSE events emitted (`layer.started`, `layer.progress`, `layer.completed`)
- **OBS-02**: OpenTelemetry tracing exporter configured and documented
- **OBS-03**: `stalePullTimeout` configurable via environment variable

## Out of Scope

| Feature | Reason |
|---------|--------|
| Persistent storage (Redis/etcd) | In-memory with 30m TTL sufficient for v0.1.0; adds operational complexity |
| UI authentication | Assumes ingress-level auth or private cluster; UI is read-only |
| Multi-cluster federation | Single cluster per deployment is the target use case for v0.1.0 |
| Slack/PagerDuty alerting | Prometheus integration covers this; users bring their own alerting stack |
| Mobile app / CLI | Web UI is the interface; no demand signal yet |
| Full frontend test suite | UI is simple enough for manual testing; low risk for v0.1.0 |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| COMM-01 | Phase 1 | Pending |
| COMM-02 | Phase 1 | Pending |
| COMM-03 | Phase 1 | Pending |
| FIX-01 | Phase 1 | Pending |
| FIX-02 | Phase 1 | Pending |
| META-01 | Phase 1 | Pending |
| DOCS-01 | Phase 2 | Pending |
| DOCS-02 | Phase 2 | Pending |
| DOCS-03 | Phase 2 | Pending |
| DOCS-04 | Phase 2 | Pending |
| DOCS-05 | Phase 2 | Pending |
| REL-01 | Phase 3 | Pending |
| HELM-01 | Phase 3 | Pending |
| HELM-02 | Phase 3 | Pending |
| HELM-03 | Phase 3 | Pending |
| HELM-04 | Phase 3 | Pending |
| REL-02 | Phase 3 | Pending |
| REL-03 | Phase 4 | Pending |
| REL-04 | Phase 4 | Pending |

**Coverage:**
- v1 requirements: 19 total
- Mapped to phases: 19
- Unmapped: 0 ✓

---
*Requirements defined: 2026-02-23*
*Last updated: 2026-02-23 after initial definition*
