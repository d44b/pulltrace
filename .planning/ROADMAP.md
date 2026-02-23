# Roadmap: Pulltrace

## Milestones

- ✅ **v0.1.0 Open Source Release** — Phases 1-4 (shipped 2026-02-23)
- 🚧 **v0.2 Housekeeping** — Phases 5-6 (in progress)

## Phases

<details>
<summary>✅ v0.1.0 Open Source Release (Phases 1-4) — SHIPPED 2026-02-23</summary>

- [x] Phase 1: Foundation Files (3/3 plans) — completed 2026-02-23
- [x] Phase 2: Documentation Site (2/2 plans) — completed 2026-02-23
- [x] Phase 3: Release Automation (2/2 plans) — completed 2026-02-23
- [x] Phase 4: Launch (2/2 plans) — completed 2026-02-23

Full details: `.planning/milestones/v0.1-ROADMAP.md`

</details>

### 🚧 v0.2 Housekeeping (In Progress)

**Milestone Goal:** Clean up post-launch debt, fix CI for future releases, and fully validate pulltrace on a live cluster.

- [ ] **Phase 5: Housekeeping** - Fix dead links, CI merge flag, document social preview
- [ ] **Phase 6: d4b Cluster Validation** - Deploy and verify all features on live cluster

## Phase Details

### Phase 5: Housekeeping
**Goal**: Post-launch debt is cleared and the repo is clean for future contributors and releases
**Depends on**: Phase 4 (v0.1.0 shipped)
**Requirements**: MAINT-01, MAINT-02, COMM-01
**Success Criteria** (what must be TRUE):
  1. CONTRIBUTING.md line 147 no longer references CODE_OF_CONDUCT.md (link removed or replaced with a valid target)
  2. CI helm repo index step uses `--merge` flag so a future v0.2.0 tag push preserves the v0.1.0 chart entry in index.yaml
  3. The manual step for uploading a social preview image is documented in CONTRIBUTING.md or a release checklist so it is not forgotten again
**Plans**: 1 plan

Plans:
- [ ] 05-01-PLAN.md — Fix CI --merge flag, document social preview step, close all Phase 5 requirements

### Phase 6: d4b Cluster Validation
**Goal**: pulltrace is confirmed working end-to-end on the d4b cluster — UI, metrics, pod correlation, and layer drill-down all verified against live traffic
**Depends on**: Phase 5
**Requirements**: VALID-01, VALID-02, VALID-03, VALID-04, VALID-05
**Success Criteria** (what must be TRUE):
  1. `helm install pulltrace pulltrace/pulltrace -n pulltrace --create-namespace` completes and all pods reach Running state on d4b
  2. The React UI is reachable in a browser and shows at least one live pull row with progress bar during an active image pull
  3. `curl http://<server>:9090/metrics` responds with 200 and output contains `pulltrace_pulls_active` and `pulltrace_pull_duration_seconds`
  4. An in-flight pull row in the UI shows at least one associated pod name in the waiting-pods column
  5. Clicking a pull row in the UI expands the layer drill-down and shows per-layer status for an active pull
**Plans**: TBD

## Progress

| Phase | Milestone | Plans Complete | Status | Completed |
|-------|-----------|----------------|--------|-----------|
| 1. Foundation Files | v0.1 | 3/3 | Complete | 2026-02-23 |
| 2. Documentation Site | v0.1 | 2/2 | Complete | 2026-02-23 |
| 3. Release Automation | v0.1 | 2/2 | Complete | 2026-02-23 |
| 4. Launch | v0.1 | 2/2 | Complete | 2026-02-23 |
| 5. Housekeeping | v0.2 | 0/1 | Not started | - |
| 6. d4b Cluster Validation | v0.2 | 0/? | Not started | - |
