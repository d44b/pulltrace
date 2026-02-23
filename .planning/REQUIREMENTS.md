# Requirements: Pulltrace

**Defined:** 2026-02-23
**Core Value:** A DevOps engineer deploying to Kubernetes can see exactly which images are pulling, how fast, which pods are waiting, and when they'll be ready — without kubectl exec or log digging.

## v0.2 Requirements

Requirements for the v0.2 Housekeeping milestone.

### Maintenance

- [x] **MAINT-01**: CONTRIBUTING.md dead link on line 147 (CODE_OF_CONDUCT.md reference) is removed or replaced
- [x] **MAINT-02**: CI helm repo index uses `--merge` flag so v0.2.0 chart release preserves the v0.1.0 chart entry

### Community

- [x] **COMM-01**: GitHub repository has a social preview image visible when shared on social media (manual upload via Settings)

### Validation

- [ ] **VALID-01**: pulltrace is deployed on the d4b cluster via Helm and all pods are running
- [ ] **VALID-02**: The React UI is accessible and shows live pull progress for image pulls happening on the cluster
- [ ] **VALID-03**: Prometheus metrics endpoint responds and contains expected metrics (`pulltrace_pulls_active`, `pulltrace_pull_duration_seconds`, etc.)
- [ ] **VALID-04**: Pod correlation is working — waiting pods for an in-flight pull are shown in the UI
- [ ] **VALID-05**: Layer drill-down in the UI expands to show per-layer status for an active pull

## Future Requirements

Deferred to v0.3+. Not in current roadmap.

### Discoverability

- **DISC-01**: Artifact Hub listing (`artifacthub-repo.yml`) for Helm chart discoverability on artifacthub.io

### Supply Chain

- **CHAIN-01**: Docker image SBOMs attached to GitHub Release
- **CHAIN-02**: cosign signatures attached to GitHub Release for image verification

### Features

- **FEAT-01**: Layer-level SSE events emitted (`layer.started`, `layer.progress`, `layer.completed`)

## Out of Scope

| Feature | Reason |
|---------|--------|
| Persistent storage (Redis/etcd) | In-memory with 30m TTL is sufficient |
| UI authentication | Assumes ingress-level auth or private cluster network |
| Mobile app / CLI | Web UI is the interface |
| Multi-cluster federation | Single cluster per deployment |
| Slack/PagerDuty alerting | Prometheus integration covers this use case |
| OpenTelemetry tracing | Adds operational complexity without clear demand signal |

## Traceability

Which phases cover which requirements. Updated during roadmap creation.

| Requirement | Phase | Status |
|-------------|-------|--------|
| MAINT-01 | Phase 5 | Complete |
| MAINT-02 | Phase 5 | Complete |
| COMM-01 | Phase 5 | Complete |
| VALID-01 | Phase 6 | Pending |
| VALID-02 | Phase 6 | Pending |
| VALID-03 | Phase 6 | Pending |
| VALID-04 | Phase 6 | Pending |
| VALID-05 | Phase 6 | Pending |

**Coverage:**
- v0.2 requirements: 8 total
- Mapped to phases: 8
- Unmapped: 0 ✓

---
*Requirements defined: 2026-02-23*
*Last updated: 2026-02-23 after Phase 5 completion*
