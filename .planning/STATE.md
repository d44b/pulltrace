# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-23)

**Core value:** A DevOps engineer can install Pulltrace with a single `helm install` command, find docs explaining how it works, and trust it as a credible open source project.
**Current focus:** Phase 4 - Launch

## Current Position

Phase: 4 of 4 (Launch)
Plan: 1 of 2 completed in current phase
Status: Plan 04-01 COMPLETE — GHCR packages public, Plan 04-02 ready
Last activity: 2026-02-23 — All three GHCR packages set to public; Plan 04-02 (push v0.1.0 tag) unblocked

Progress: [████████░░] 80% (Phase 4 plan 1/2 complete)

## Performance Metrics

**Velocity:**
- Total plans completed: 6
- Average duration: 6min
- Total execution time: 36min

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01-foundation-files | 3 | 17min | 6min |
| 02-documentation-site | 2 | 18min | 9min |
| 03-release-automation | 2 | 2min | 1min |

**Recent Trend:**
- Last 5 plans: 02-01 (8min), 02-02 (10min), 03-01 (1min), 03-02 (1min)
- Trend: stable

*Updated after each plan completion*
| Phase 01-foundation-files P01 | 5 | 2 tasks | 2 files |
| Phase 01-foundation-files P03 | 1min | 2 tasks | 0 files |
| Phase 02-documentation-site P01 | 8min | 2 tasks | 9 files |
| Phase 02-documentation-site P02 | 10min | 3 tasks | 7 files |
| Phase 03-release-automation P01 | 1min | 2 tasks | 2 files |
| Phase 03-release-automation P02 | 1min | 1 tasks | 1 files |
| Phase 04-launch P01 | 5min | 2 tasks | 0 files |

## Accumulated Context

### Decisions

Decisions are logged in PROJECT.md Key Decisions table.
Recent decisions affecting current work:

- Material for MkDocs 9.7.2 chosen for docs site (dominant K8s ecosystem standard, Python-only)
- peaceiris/actions-gh-pages with keep_files: true is the only safe co-deployment pattern for docs + Helm index on same gh-pages branch
- helm/chart-releaser-action NOT used (creates duplicate releases, forces index.yaml to gh-pages root)
- softprops/action-gh-release@v2 for GitHub Release creation (chart-releaser-action replacement)
- PullErrors.Inc() placed in processReport() completion path only (not cleanup()) — agent is authoritative error source, cleanup is a timeout safety net
- Layer rate keys use key+":layer:"+digest compound format enabling HasPrefix bulk-delete on pull eviction
- [Phase 01-foundation-files]: golang:1.22-alpine used in Docker workaround command to match Dockerfile.server FROM line exactly
- [Phase 01-foundation-files]: CHANGELOG.md comparison links acceptable as 404 until v0.1.0 tag pushed in Phase 4
- [Phase 01-foundation-files]: GitHub topics set via gh CLI (scriptable, idempotent); social preview deferred to manual upload via Settings UI
- [Phase 03-release-automation]: No --merge flag for v0.1.0 helm repo index (no prior index.yaml exists; use --merge for v0.2.0+)
- [Phase 03-release-automation]: cancel-in-progress:false on deploy-gh-pages concurrency — never abort an in-flight gh-pages push
- [Phase 03-release-automation]: needs: [helm-release] not needs: [docker] for github-release job — ensures release appears only after Helm index is live
- [Phase 04-launch]: GHCR v2 /tags/list always returns 401 regardless of visibility — use gh api /users/{owner}/packages/container/{name} to verify package visibility
- [Phase 04-launch]: All three GHCR packages (pulltrace-agent, pulltrace-server, charts/pulltrace) set to public before v0.1.0 tag push — Plan 04-02 unblocked

### Pending Todos

None yet.

### Blockers/Concerns

None — GHCR packages blocker resolved (all three packages set to public in Plan 04-01).

## Session Continuity

Last session: 2026-02-23
Stopped at: Completed 04-01-PLAN.md — GHCR packages public, Plan 04-02 ready
Resume file: None
