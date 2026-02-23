---
phase: quick
plan: 1
subsystem: server
tags: [go, rate-calculator, concurrency, speed-display, bug-fix]

# Dependency graph
requires: []
provides:
  - Monotonic byte tracking for RateCalculator preventing speed drop to zero on concurrent pull completion
affects: [server, rate-display, ui-speed]

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "Monotonic counter pattern: track prev bytes per key, feed max(prev, current) to rate calculator to prevent negative deltas"

key-files:
  created: []
  modified:
    - internal/server/server.go

key-decisions:
  - "Use per-key lastBytes map on Server rather than modifying RateCalculator semantics — keeps RateCalculator as a pure cumulative-to-rate converter and isolates the monotonic enforcement at the call site"
  - "Apply the monotonic fix uniformly to both per-pull and per-layer rate keys — consistent and prevents any future similar issue with layer bytes"

patterns-established:
  - "Monotonic accumulator: whenever feeding cumulative bytes from an external source to a rate calculator, clamp to max(prev, current) to guard against drops caused by source discontinuities"

requirements-completed: [BUG-speed-drop]

# Metrics
duration: 2min
completed: 2026-02-23
---

# Quick Fix 1: Speed Drop Bug Summary

**Monotonic lastBytes map added to Server that clamps cumulative byte deltas to zero-floor, preventing RateCalculator from returning 0 when a concurrent __pulling__ pull finishes and the merged byte total drops**

## Performance

- **Duration:** 2 min
- **Started:** 2026-02-23T21:43:50Z
- **Completed:** 2026-02-23T21:45:36Z
- **Tasks:** 1
- **Files modified:** 1

## Accomplishments
- Root cause identified: `mergeDigestPulls` sums bytes from all active digest-pull layers; when one image finishes the sum drops; `RateCalculator.Rate()` computed `last.bytes - first.bytes < 0` and clamped to 0
- Added `lastBytes map[string]int64` to `Server` struct, initialized in `New()`, cleaned up in `cleanup()`
- All `rc.Add()` calls now receive `prev + max(0, current - prev)` — a value that never decreases for a given key
- Fix applied uniformly to both per-pull rate keys and per-layer rate keys
- Docker build passes, Go compiles cleanly

## Task Commits

Each task was committed atomically:

1. **Task 1: Fix rate calculator to use byte deltas instead of cumulative totals** - `09985e0` (fix)

## Files Created/Modified
- `internal/server/server.go` - Added `lastBytes map[string]int64` field, monotonic clamping before `rc.Add()` calls for both pull-level and layer-level rate calculators, cleanup in `cleanup()`

## Decisions Made
- Keep `RateCalculator` unchanged (cumulative semantics intact) — the fix lives entirely in `processReport` at the call sites, not inside the calculator
- Apply uniformly to all keys (not just `__pulling__`) — consistent behavior, no special-casing required

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered
- Dockerfile not named `Dockerfile` — used `Dockerfile.server` instead. Build command adjusted to `-f Dockerfile.server`.

## User Setup Required
None - no external service configuration required.

## Next Phase Readiness
- Bug fix is live on main branch
- Speed display will remain stable when concurrent pulls complete
- No further changes required for this issue

---
*Phase: quick*
*Completed: 2026-02-23*
