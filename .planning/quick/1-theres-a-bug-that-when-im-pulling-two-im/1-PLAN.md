---
phase: quick
plan: 1
type: execute
wave: 1
depends_on: []
files_modified:
  - internal/server/server.go
autonomous: true
requirements: [BUG-speed-drop]

must_haves:
  truths:
    - "When two images are pulling concurrently, the speed of the first pull continues showing correctly after the second completes"
    - "Total throughput does not drop to 0 when one of two concurrent pulls finishes"
    - "Speed recovers immediately (not after ~10 seconds) when a concurrent pull completes"
  artifacts:
    - path: "internal/server/server.go"
      provides: "Fixed rate calculation using per-report byte deltas"
      contains: "lastBytes map"
  key_links:
    - from: "internal/server/server.go processReport"
      to: "model.RateCalculator.Add"
      via: "delta bytes, not cumulative"
      pattern: "delta.*downloadedBytes|downloadedBytes.*delta"
---

<objective>
Fix the speed display bug that causes per-pull speed and total throughput to drop to 0 for several seconds when one of two concurrent image pulls completes.

Purpose: The RateCalculator receives cumulative downloadedBytes. When two images pull concurrently as content-digest entries merged into a single "__pulling__" key, the merged downloadedBytes = sum of all active layers from both images. When image B finishes and disappears from the agent report, the next merged downloadedBytes only contains image A's layers — a drop. The rate calculator sees last.bytes < first.bytes → total < 0 → Rate() = 0. Speed shows as 0 until the 10-second window purges the old high-value samples.

Output: server.go with a lastBytes map tracking cumulative bytes per pull key. processReport feeds (currentBytes - lastBytes) deltas to rc.Add() instead of the raw cumulative total.
</objective>

<execution_context>
@/Users/jmaciejewski/.claude/get-shit-done/workflows/execute-plan.md
@/Users/jmaciejewski/.claude/get-shit-done/templates/summary.md
</execution_context>

<context>
@.planning/STATE.md
@internal/server/server.go
@internal/model/rate.go
</context>

<tasks>

<task type="auto">
  <name>Task 1: Fix rate calculator to use byte deltas instead of cumulative totals</name>
  <files>internal/server/server.go</files>
  <action>
Add a `lastBytes map[string]int64` field to the `Server` struct. Initialize it in `New()` as `make(map[string]int64)`.

In `processReport`, BEFORE calling `rc.Add(downloadedBytes)` (around line 435), compute the delta:

```go
prev := s.lastBytes[key]
delta := downloadedBytes - prev
if delta < 0 {
    delta = 0
}
s.lastBytes[key] = downloadedBytes
rc.Add(prev + delta)  // or just rc.Add(downloadedBytes) with delta-aware Add
```

Wait — the rate calculator uses samples as absolute values and computes rate as (last - first) / elapsed. The fix is to feed DELTAS as increments so the rate calculator accumulates a running total that only grows. Change the approach: instead of storing cumulative bytes in each sample, use the delta as an absolute increment and track a running sum.

The cleanest fix: change `rc.Add(downloadedBytes)` to `rc.Add(downloadedBytes - s.lastBytes[key])` ONLY if `downloadedBytes >= s.lastBytes[key]`, then set `s.lastBytes[key] = downloadedBytes`. But this changes the semantics of what Add receives — the rate calculator uses `last.bytes - first.bytes` which would then be `sum_of_deltas_in_window` — wrong.

Correct approach: keep rate calculator as-is (cumulative semantics), but maintain a MONOTONIC cumulative counter per pull key in the server. Track `s.lastBytes[key]` as the running total bytes fed to the rate calculator. Each report, compute `delta = max(0, downloadedBytes - s.lastBytes[key])` then call `rc.Add(s.lastBytes[key] + delta)` and update `s.lastBytes[key] += delta`. This ensures the value fed to rc.Add() never decreases.

Implementation steps:
1. Add `lastBytes map[string]int64` to the `Server` struct (after `rates` field)
2. In `New()`, initialize: `lastBytes: make(map[string]int64)`
3. In `processReport`, for the per-pull rate calculation block (lines ~430-436), replace:
   ```go
   rc.Add(downloadedBytes)
   ```
   with:
   ```go
   prev := s.lastBytes[key]
   delta := downloadedBytes - prev
   if delta < 0 {
       delta = 0
   }
   s.lastBytes[key] = prev + delta
   rc.Add(prev + delta)
   ```
4. In the `cleanup()` function, alongside `delete(s.rates, key)` and `delete(s.lastSeen, key)`, also add `delete(s.lastBytes, key)` to prevent memory leaks
5. Also clean up `lastBytes` for layer rate keys when a pull is deleted (same pattern as the layer rates cleanup loop)

Note: Do NOT change the layer-level rate calculators (layerKey lines ~398-405) — layer bytes from a single image's layers are monotonic within a single pull. Only the per-pull rc.Add on the merged __pulling__ key is affected. But applying the same delta fix to all keys (not just __pulling__) is safe and consistent. Apply the fix uniformly for all keys.

Also apply the same monotonic fix for layer rate calculators: track `s.lastBytes[layerKey]` and feed monotonic values to the layer rc.Add too (same pattern).
  </action>
  <verify>
    <automated>cd /Users/jmaciejewski/workspace/pulltrace && docker build -t pulltrace-test:fix . 2>&1 | tail -5</automated>
    <manual>To simulate the bug: start pulltrace server, have agent report two concurrent pulls, then remove one from the report — verify speed of remaining pull and total throughput stay non-zero immediately after the removal.</manual>
  </verify>
  <done>
    - server.go compiles with no errors (Docker build passes)
    - lastBytes map exists on Server struct, initialized in New(), cleaned up in cleanup()
    - rc.Add() receives a value that never decreases for a given key across reports
    - When one of two concurrent pulls completes, the remaining pull's bytesPerSec and total throughput remain non-zero without a ~10-second recovery period
  </done>
</task>

</tasks>

<verification>
Build verification: `docker build -t pulltrace-test:fix .` must exit 0.

Logic verification: After the fix, `s.lastBytes[key]` holds the highest cumulative downloadedBytes ever fed to rc.Add for that key. If the agent report shows lower bytes for any reason (e.g., digest-merged pull losing one image's layers), the delta is clamped to 0 and the rate calculator's running total stays flat rather than going negative. Rate() returns (last - first) / elapsed where both are from the monotonic series — still positive.
</verification>

<success_criteria>
- docker build exits 0 (Go compiles)
- Server struct has lastBytes field initialized and cleaned up
- rc.Add() call uses a monotonically non-decreasing value derived from lastBytes + max(0, delta)
- No speed-drops-to-zero regression when concurrent pulls complete
</success_criteria>

<output>
After completion, create `.planning/quick/1-theres-a-bug-that-when-im-pulling-two-im/1-SUMMARY.md` following the summary template.
</output>
