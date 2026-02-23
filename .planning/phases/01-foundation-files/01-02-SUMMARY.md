---
phase: 01-foundation-files
plan: "02"
subsystem: api
tags: [go, prometheus, metrics, sse, rate-calculator]

# Dependency graph
requires:
  - phase: 01-foundation-files
    provides: existing server.go with processReport() and cleanup() logic
provides:
  - PullErrors Prometheus counter incremented on pull completion with non-empty Error field
  - LayerStatus.MediaType populated from agent LayerState.MediaType
  - LayerStatus.BytesPerSec computed via per-layer RateCalculator (key pattern: key+":layer:"+digest)
  - Layer rate entries cleaned up in cleanup() to prevent unbounded rates map growth
affects: [metrics, monitoring, frontend-layer-detail]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Per-layer rate tracking uses compound key: pullKey + ':layer:' + digest — consistent with pull-level key pattern"
    - "Cleanup iterates s.rates with HasPrefix to delete all child keys when parent pull is evicted"

key-files:
  created: []
  modified:
    - internal/server/server.go

key-decisions:
  - "PullErrors.Inc() placed in processReport() completion path only (not cleanup()) so it fires when agent confirms pull gone, not on TTL eviction"
  - "Layer rate keys scoped as key+':layer:'+digest to namespace cleanly under pull key and allow prefix-based bulk delete"

patterns-established:
  - "Metric increment follows data availability: PullBytesTotal.Add then PullErrors.Inc to keep error accounting paired with completion"
  - "Compound rate keys (parent:child:id) enable hierarchical cleanup via HasPrefix scan"

requirements-completed: []

# Metrics
duration: 8min
completed: 2026-02-23
---

# Phase 1 Plan 02: Foundation Files — Bug Fixes Summary

**Wired `metrics.PullErrors` counter on pull completion and added per-layer `BytesPerSec`/`MediaType` population in `processReport()` with coordinated cleanup in `cleanup()`**

## Performance

- **Duration:** ~8 min
- **Started:** 2026-02-23T14:09:44Z
- **Completed:** 2026-02-23T14:18:00Z
- **Tasks:** 2
- **Files modified:** 1

## Accomplishments

- `metrics.PullErrors.Inc()` now fires in `processReport()` when a pull is marked complete and `pull.Error != ""` — the counter was defined but never incremented
- `LayerStatus.MediaType` is now copied from the agent's `LayerState.MediaType` field during the layer loop
- `LayerStatus.BytesPerSec` is now computed per-layer via a `model.RateCalculator` stored at `key+":layer:"+digest` in `s.rates`
- `cleanup()` now iterates `s.rates` and deletes all layer-level rate entries (`key+":layer:*"`) when a completed pull is evicted, preventing unbounded map growth

## Task Commits

Each task was committed atomically:

1. **Task 1 + Task 2: Wire PullErrors and layer BytesPerSec/MediaType** - `415caf2` (fix)

**Plan metadata:** (docs commit below)

## Files Created/Modified

- `internal/server/server.go` - Three targeted changes: PullErrors.Inc() in completion path, MediaType+BytesPerSec in layer loop, HasPrefix cleanup in cleanup()

## Decisions Made

- PullErrors is incremented only in the `processReport()` completion path (absent-from-report pulls), not in `cleanup()`'s stale-pull force-completion path. Rationale: the agent is the authoritative source of error state; cleanup is a timeout safety net, not an error signal.
- Layer rate keys use `key + ":layer:" + digest` compound format. This scopes them under the pull key naturally and enables a single `HasPrefix` scan to bulk-delete all layer rates when a pull is evicted.

## Deviations from Plan

None - plan executed exactly as written. Both tasks applied directly without complications. The `strings` package was already imported (confirmed at line 12).

## Issues Encountered

None. Docker build (`golang:1.22 go build ./internal/server/...`) returned clean with no output.

## Next Phase Readiness

- `internal/server/server.go` is production-ready for both metric fixes
- Frontend will now receive `mediaType` and `bytesPerSec` per layer in SSE events and `/api/v1/pulls` responses
- Prometheus scrape of `:9090/metrics` will show `pulltrace_pull_errors_total` incrementing correctly

---
*Phase: 01-foundation-files*
*Completed: 2026-02-23*

## Self-Check: PASSED

- FOUND: `.planning/phases/01-foundation-files/01-02-SUMMARY.md`
- FOUND: commit `415caf2` (fix(01-02): wire PullErrors metric and populate layer BytesPerSec/MediaType)
