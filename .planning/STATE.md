# Project State

## Project Reference

See: .planning/PROJECT.md (updated 2026-02-23 after v0.1 milestone)

**Core value:** A DevOps engineer deploying to Kubernetes can see exactly which images are pulling, how fast, which pods are waiting, and when they'll be ready — without kubectl exec or log digging.
**Current focus:** Planning next milestone

## Current Position

Phase: Not started (defining requirements)
Plan: —
Status: Defining requirements
Last activity: 2026-02-23 — Milestone v0.2 Housekeeping started

## Performance Metrics

**Velocity:**
- Total plans completed: 8
- Average duration: ~5min
- Total execution time: ~38min

**By Phase:**

| Phase | Plans | Total | Avg/Plan |
|-------|-------|-------|----------|
| 01-foundation-files | 3 | 17min | 6min |
| 02-documentation-site | 2 | 18min | 9min |
| 03-release-automation | 2 | 2min | 1min |
| 04-launch | 2 | 7min | 4min |

**Recent Trend:**
- Last 5 plans: 02-02 (10min), 03-01 (1min), 03-02 (1min), 04-01 (5min), 04-02 (2min)
- Trend: stable

*Updated after each plan completion*
| Phase 01-foundation-files P01 | 5 | 2 tasks | 2 files |
| Phase 01-foundation-files P03 | 1min | 2 tasks | 0 files |
| Phase 02-documentation-site P01 | 8min | 2 tasks | 9 files |
| Phase 02-documentation-site P02 | 10min | 3 tasks | 7 files |
| Phase 03-release-automation P01 | 1min | 2 tasks | 2 files |
| Phase 03-release-automation P02 | 1min | 1 tasks | 1 files |
| Phase 04-launch P01 | 5min | 2 tasks | 0 files |
| Phase 04-launch P02 | 2min | 3 tasks | 0 files |

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
- [Phase 04-launch]: v0.1.0 tag was already on origin from prior session; CI run #22316078048 all 7 jobs green; all 8 post-launch artifact checks passed

### Pending Todos

- Upload social preview image to GitHub repo Settings (manual, no API)
- Fix CONTRIBUTING.md dead link to CODE_OF_CONDUCT.md (line 147)
- Add `--merge` flag to helm repo index for v0.2.0+ releases

### Blockers/Concerns

None.

## Session Continuity

Last session: 2026-02-23
Stopped at: Completed v0.1 milestone archival
Resume file: None
