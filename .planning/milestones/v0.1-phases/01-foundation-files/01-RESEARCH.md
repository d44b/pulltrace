# Phase 1: Foundation Files - Research

**Researched:** 2026-02-23
**Domain:** Open-source community files, Go Prometheus metrics, React frontend data binding, GitHub repository metadata
**Confidence:** HIGH

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| COMM-01 | CONTRIBUTING.md with local dev setup (Go 1.22 + Docker workaround), build instructions, and PR guidelines | File exists but is missing the Go 1.22 Docker workaround — needs a new section added |
| COMM-02 | CHANGELOG.md in keep-a-changelog format with a [0.1.0] entry | File does not exist — must be created from scratch at repo root |
| ~~COMM-03~~ | ~~CODE_OF_CONDUCT.md~~ | **Removed — do NOT create CODE_OF_CONDUCT.md** |
| FIX-01 | `pulltrace_pull_errors_total` counter incremented when pull completes with non-empty Error field | Counter defined in metrics.go but never called — one-line fix in server.go processReport |
| FIX-02 | Server populates `layer.bytesPerSec` and `layer.mediaType` so LayerDetail can display them | LayerStatus struct has both fields; server never populates them — requires layer-level rate tracking and MediaType copy |
| META-01 | GitHub repo has description, topics (kubernetes, monitoring, containers, helm, containerd), and social preview image | Description already set; topics empty; social preview is UI-only (no API) |
</phase_requirements>

---

## Summary

Phase 1 delivers the credibility signals that distinguish an active open-source project from an abandoned repo. Five of the six original requirements are pure file operations or small Go/metadata changes. One requirement (FIX-02: layer bytesPerSec) is a moderate Go change that requires per-layer rate tracking, but the data model is already in place.

COMM-03 (CODE_OF_CONDUCT.md) has been intentionally removed. Do NOT create CODE_OF_CONDUCT.md.

The GitHub social preview image (META-01) is the only requirement that cannot be automated — the GitHub API and `gh` CLI do not expose social preview upload. The plan must document this as a manual step with a generated placeholder image.

**Primary recommendation:** Fix files in order of simplest to most complex — COMM-02 (create CHANGELOG.md), META-01 (gh repo edit + manual UI step), COMM-01 (add Docker workaround section to CONTRIBUTING.md), FIX-01 (one-line metrics increment), FIX-02 (layer rate calculator + MediaType copy).

---

## Current State Audit

This section documents the actual state of each requirement before work begins. The planner MUST use this to scope tasks correctly.

| Req | File/Code | Current State | Gap |
|-----|-----------|---------------|-----|
| COMM-01 | `CONTRIBUTING.md` (3.9K, 132 lines) | Has build steps, running locally, PR guidelines | Missing: Go 1.22 + Docker workaround for devs on older Go |
| COMM-02 | `CHANGELOG.md` | Does not exist | Create from scratch |
| ~~COMM-03~~ | `CODE_OF_CONDUCT.md` | **Removed — do NOT create this file** | N/A |
| FIX-01 | `internal/metrics/metrics.go` + `internal/server/server.go` | `PullErrors` counter defined, never incremented | Add `metrics.PullErrors.Inc()` in `processReport` at pull completion when `pull.Error != ""` |
| FIX-02 | `internal/server/server.go` `processReport()` | `LayerStatus.BytesPerSec` and `LayerStatus.MediaType` never populated | Populate MediaType from agent report; add per-layer rate calculator |
| META-01 | GitHub repo | Description set; topics: empty; social preview: not set | Add 5 topics via `gh repo edit`; social preview requires GitHub UI (not automatable) |

---

## Standard Stack

### Core (no new dependencies required)

| Library/Tool | Version | Purpose | Notes |
|---|---|---|---|
| `github.com/prometheus/client_golang` | v1.20.5 (already in go.mod) | Prometheus counter increment | Already imported — `metrics.PullErrors.Inc()` is the only change |
| `github.com/d44b/pulltrace/internal/model` | — | `RateCalculator` already exists | Already used for pull-level rate — reuse same type for per-layer |
| `gh` CLI | system | Set repo topics, description | `gh repo edit --add-topic` |

### RateCalculator — existing pattern to reuse for FIX-02

The server already maintains per-pull `*model.RateCalculator` in `s.rates map[string]*model.RateCalculator`. The same type must be used per-layer. The layer key is `pullKey + ":" + layer.Digest`.

```go
// Source: internal/server/server.go — existing pull-level pattern
rc, ok := s.rates[key]
if !ok {
    rc = model.NewRateCalculator(10 * time.Second)
    s.rates[key] = rc
}
rc.Add(downloadedBytes)
existing.BytesPerSec = rc.Rate()
```

For layers, mirror this pattern with layer-keyed entries in `s.rates`.

---

## Architecture Patterns

### Pattern 1: Prometheus Counter Increment (FIX-01)

**What:** Call `metrics.PullErrors.Inc()` at the point where a completed pull is detected with a non-empty Error field.

**Where in the code:** In `processReport()` in `internal/server/server.go`, inside the "pulls absent from report" loop that marks pulls as completed. The `pull.Error` field is set by the agent (via `PullState` — actually stored in `PullStatus.Error`). The increment must happen when `pull.CompletedAt` is being set AND `pull.Error != ""`.

**Example — correct location:**
```go
// Source: internal/server/server.go, cleanupLoop / processReport completion path
// Existing code at line ~461-465:
pull.CompletedAt = &now
pull.Percent = 100
metrics.PullsActive.Dec()
metrics.PullDurationSeconds.Observe(now.Sub(pull.StartedAt).Seconds())
metrics.PullBytesTotal.Add(float64(pull.TotalBytes))
// ADD HERE:
if pull.Error != "" {
    metrics.PullErrors.Inc()
}
```

**Important:** The `PullStatus.Error` field is populated from `PullState.Error` — but checking the `PullState` struct in model/event.go reveals it does NOT have an Error field. The Error field is only on `PullStatus`. The agent must set it somehow, or the server sets it on stale/failed pulls. For v0.1.0, increment when the pull completes with Error != "" — the most defensible behavior.

**Verification:** Start server, POST a report that generates a pull, then POST a report with `pull.Error` set (or simulate a stale pull with error). Hit `:9090/metrics` and grep for `pulltrace_pull_errors_total`.

### Pattern 2: Per-Layer Rate Calculator (FIX-02)

**What:** For each layer in `processReport()`, maintain a rate calculator keyed by `pullKey + ":layer:" + layer.Digest` and populate `LayerStatus.BytesPerSec`. Also copy `MediaType` from `LayerState` to `LayerStatus`.

**The data flow:**
- Agent sends `PullState.Layers []LayerState` — `LayerState` has `MediaType` field
- Server builds `LayerStatus` in `processReport()` — currently does NOT copy MediaType or compute BytesPerSec
- Frontend reads `pull.layers[i].bytesPerSec` and `pull.layers[i].mediaType` in `LayerDetail.jsx`

**Current gap in processReport (lines ~385-405):**
```go
ls := model.LayerStatus{
    PullID:          key,
    Digest:          layer.Digest,
    TotalBytes:      layer.TotalBytes,
    DownloadedBytes: layer.DownloadedBytes,
    TotalKnown:      layer.TotalKnown,
}
// MISSING: MediaType copy and BytesPerSec calculation
```

**Fix:**
```go
ls := model.LayerStatus{
    PullID:          key,
    Digest:          layer.Digest,
    MediaType:       layer.MediaType,  // ADD: copy from agent report
    TotalBytes:      layer.TotalBytes,
    DownloadedBytes: layer.DownloadedBytes,
    TotalKnown:      layer.TotalKnown,
}

// ADD: per-layer rate calculation
layerKey := key + ":layer:" + layer.Digest
lrc, ok := s.rates[layerKey]
if !ok {
    lrc = model.NewRateCalculator(10 * time.Second)
    s.rates[layerKey] = lrc
}
lrc.Add(layer.DownloadedBytes)
ls.BytesPerSec = lrc.Rate()
```

**Cleanup consideration:** Layer rate entries in `s.rates` must be cleaned up when their parent pull is removed. The `cleanup()` function deletes `s.rates[key]` — it must also delete `layerKey` entries. Use a prefix scan: `key + ":layer:"`.

### Pattern 3: Keep-a-Changelog Format (COMM-02)

**What:** Standard format for CHANGELOG.md. Latest version first, sections only included if non-empty.

**Correct structure for this project:**
```markdown
# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-02-23

### Added
- Real-time container image pull progress monitoring via containerd socket
- Layer-by-layer download progress with bytesPerSec and ETA
- Kubernetes pod correlation (matches pulls to waiting pods)
- Prometheus metrics: pulls_active, pulls_total, pull_duration_seconds, pull_bytes_total, pull_errors_total
- SSE streaming for live UI updates
- React + Vite web UI with per-layer expandable detail
- Helm chart for Kubernetes deployment

[Unreleased]: https://github.com/d44b/pulltrace/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/d44b/pulltrace/releases/tag/v0.1.0
```

**Source:** https://keepachangelog.com/en/1.1.0/ (verified 2026-02-23)

### Pattern 4: GitHub Repo Topics (META-01)

**CLI command:**
```bash
gh repo edit d44b/pulltrace \
  --add-topic kubernetes \
  --add-topic monitoring \
  --add-topic containers \
  --add-topic helm \
  --add-topic containerd
```

**Verify:**
```bash
gh api repos/d44b/pulltrace --jq '.topics'
```

**Social preview:** Cannot be set programmatically. GitHub API discussion confirms no endpoint exists. The plan must include a manual step: go to GitHub → Repository Settings → Social preview → Upload image. A 1280×640px PNG is the recommended size.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Layer download speed | Custom sliding window | `model.RateCalculator` (already in codebase) | Already used for pull-level rates; same type works for layers |
| Changelog generation | Script/automation | Hand-authored CHANGELOG.md | v0.1.0 is first release; automation adds complexity for a one-time event |
| Repo topic setting | GitHub UI clicks | `gh repo edit --add-topic` | Repeatable, scriptable, verifiable |

---

## Common Pitfalls

### Pitfall 1: CONTRIBUTING.md "Go 1.22 workaround" placement
**What goes wrong:** Adding the Docker workaround at the end of the file where contributors with older Go won't see it until they've already tried `make build` and failed.
**Why it happens:** Tacking additions to the end is the path of least resistance.
**How to avoid:** Insert the workaround in the "Development Setup" section, immediately before the `make build` command, as a named subsection "Building with Docker (Go 1.22 required locally)."
**Warning signs:** If the workaround appears after the "Running locally" section, it will be missed.

### Pitfall 2: Creating CODE_OF_CONDUCT.md
**What goes wrong:** An executor sees COMM-03 references in old notes and creates CODE_OF_CONDUCT.md anyway.
**Why it happens:** COMM-03 appears in research files and old summaries.
**How to avoid:** COMM-03 is removed. CODE_OF_CONDUCT.md must NOT be created. If it exists, delete it.
**Warning signs:** Any task or step that writes CODE_OF_CONDUCT.md.

### Pitfall 3: FIX-01 — where is pull.Error set?
**What goes wrong:** `PullState` (agent report) has no `Error` field. `PullStatus.Error` exists but the agent never sets it.
**Why it happens:** The Error field was added to PullStatus for future use.
**How to avoid:** For v0.1.0, a reasonable approach is: increment `PullErrors` when a stale pull (timed out in cleanup) is force-completed — these are de-facto failed pulls. Alternatively, increment when `pull.CompletedAt` is set AND `pull.Error != ""` — which requires the agent or server to populate `pull.Error` when an error condition is detected. Check `internal/agent/agent.go` and `internal/containerd/watcher.go` for where errors are surfaced.
**Warning signs:** If `pulltrace_pull_errors_total` never increments in normal operation, the implementation is correct but the counter is untestable without error injection.

### Pitfall 4: FIX-02 — layer rate cleanup leak
**What goes wrong:** Per-layer rate calculators in `s.rates` accumulate forever because `cleanup()` only deletes the pull key, not the `pullKey:layer:digest` keys.
**Why it happens:** The cleanup loop doesn't know about layer-keyed entries.
**How to avoid:** When deleting a pull from `s.rates` in `cleanup()`, also iterate over `s.rates` and delete all keys with the prefix `key + ":layer:"`.
**Warning signs:** Memory growth in long-running servers with many layer events.

### Pitfall 5: META-01 — social preview is UI-only
**What goes wrong:** Planning task assumes social preview can be set via API or `gh` CLI.
**Why it happens:** Other settings (description, topics) are automatable.
**How to avoid:** The planner must mark social preview as a manual step with explicit instructions. A task can prepare the image file and generate instructions, but execution requires a human to upload through the GitHub UI.
**Warning signs:** Any plan step that says "gh api ... social_preview" — this endpoint does not exist.

### Pitfall 6: CHANGELOG.md comparison links
**What goes wrong:** Including `[0.1.0]: https://github.com/...` links that 404 before the tag is pushed.
**Why it happens:** Copy from examples that assume the tag exists.
**How to avoid:** Include the comparison links at the bottom — they will 404 until Phase 4 tag push, which is acceptable. The file content is what matters for Phase 1.

---

## Code Examples

### FIX-01: PullErrors increment in processReport

```go
// Source: internal/server/server.go, processReport() — pull completion block (~line 460)
// Location: inside the "for key, pull := range s.pulls" loop, where pull.CompletedAt is set

pull.CompletedAt = &now
pull.Percent = 100
metrics.PullsActive.Dec()
metrics.PullDurationSeconds.Observe(now.Sub(pull.StartedAt).Seconds())
metrics.PullBytesTotal.Add(float64(pull.TotalBytes))
if pull.Error != "" {
    metrics.PullErrors.Inc()
}
```

### FIX-02: Layer MediaType + BytesPerSec in processReport

```go
// Source: internal/server/server.go, processReport() — layer loop (~line 385)
// Changes: add MediaType copy + per-layer RateCalculator

for _, layer := range pull.Layers {
    totalBytes += layer.TotalBytes
    downloadedBytes += layer.DownloadedBytes

    ls := model.LayerStatus{
        PullID:          key,
        Digest:          layer.Digest,
        MediaType:       layer.MediaType,       // ADD: copy from LayerState
        TotalBytes:      layer.TotalBytes,
        DownloadedBytes: layer.DownloadedBytes,
        TotalKnown:      layer.TotalKnown,
    }

    // ADD: per-layer rate calculation
    layerKey := key + ":layer:" + layer.Digest
    lrc, ok := s.rates[layerKey]
    if !ok {
        lrc = model.NewRateCalculator(10 * time.Second)
        s.rates[layerKey] = lrc
    }
    lrc.Add(layer.DownloadedBytes)
    ls.BytesPerSec = lrc.Rate()

    if layer.TotalKnown && layer.TotalBytes > 0 {
        ls.Percent = float64(layer.DownloadedBytes) / float64(layer.TotalBytes) * 100
    }
    if layer.TotalKnown && layer.DownloadedBytes >= layer.TotalBytes {
        ls.Percent = 100
        layersDone++
        completedAt := now
        ls.CompletedAt = &completedAt
    }
    layerStatuses = append(layerStatuses, ls)
}
```

### FIX-02: Layer rate cleanup in cleanup()

```go
// Source: internal/server/server.go, cleanup() — add after deleting pull key
// Location: inside "if pull.CompletedAt != nil && pull.CompletedAt.Before(ttlCutoff)"

delete(s.pulls, key)
delete(s.rates, key)
delete(s.lastSeen, key)

// ADD: clean up per-layer rate calculators
layerPrefix := key + ":layer:"
for rateKey := range s.rates {
    if strings.HasPrefix(rateKey, layerPrefix) {
        delete(s.rates, rateKey)
    }
}
```

### META-01: Set repo topics via gh CLI

```bash
# Source: gh CLI docs (verified 2026-02-23)
gh repo edit d44b/pulltrace \
  --add-topic kubernetes \
  --add-topic monitoring \
  --add-topic containers \
  --add-topic helm \
  --add-topic containerd

# Verify
gh api repos/d44b/pulltrace --jq '.topics'
# Expected: ["kubernetes","monitoring","containers","helm","containerd"]
```

---

## State of the Art

| Topic | Approach for This Project |
|-------|--------------------------|
| CHANGELOG format | Keep a Changelog v1.1.0 — industry standard, human-readable |
| Contributor Covenant | v2.1 — current version as of 2026 |
| Prometheus counter increment | `counter.Inc()` — one-line call, no new deps |
| GitHub topics | `gh repo edit --add-topic` — idempotent CLI command |
| GitHub social preview | Manual UI upload — no API exists (confirmed 2026-02-23) |

---

## Open Questions

1. **FIX-01: Where does `PullStatus.Error` get populated?**
   - What we know: `PullState` (agent-side) has no Error field; `PullStatus.Error` field exists but is never set in current code
   - What's unclear: Does the agent ever surface a pull error to the server? Is `Error` populated only by stale-pull detection?
   - Recommendation: Check `internal/agent/agent.go` and `internal/containerd/watcher.go` before implementing. If Error is never populated, the counter will technically satisfy the requirement if incremented when stale pulls are force-completed (those are effectively errors). Document this choice.

2. **COMM-01: Full Docker build workaround details**
   - What we know: Local Go is 1.18, go.mod requires 1.22.0, `make build` fails locally
   - What's unclear: Is there a `docker run` invocation that mounts the source and runs `go build`? Does the Makefile have a Docker-based build target?
   - Recommendation: Check Dockerfile.server/Dockerfile.agent for the Go version used, then write the Docker workaround accordingly. The standard pattern is `docker run --rm -v $(pwd):/app -w /app golang:1.22 go build ./...`.

3. **COMM-03: Contact method placeholder**
   - What we know: The full Contributor Covenant v2.1 requires `[INSERT CONTACT METHOD]`
   - What's unclear: Does the project have a maintainer email to use?
   - Recommendation: Use `conduct@d44b.github.io` or leave the `[INSERT CONTACT METHOD]` placeholder per the official template guidance. Do not invent an email address.

---

## Sources

### Primary (HIGH confidence)
- Direct code inspection: `internal/metrics/metrics.go` — PullErrors counter defined, never called
- Direct code inspection: `internal/server/server.go` — processReport() layer loop missing MediaType/BytesPerSec
- ~~Direct code inspection: `CODE_OF_CONDUCT.md`~~ — file removed; COMM-03 dropped
- Direct code inspection: `CONTRIBUTING.md` — missing Go 1.22 Docker workaround
- `gh api repos/d44b/pulltrace --jq '.topics'` — confirmed topics: [] (empty)
- `gh repo edit --help` — confirmed `--add-topic` flag exists

### Secondary (MEDIUM confidence)
- https://keepachangelog.com/en/1.1.0/ — keep-a-changelog format verified (WebFetch 2026-02-23)
- https://www.contributor-covenant.org/version/2/1/code_of_conduct/ — v2.1 structure verified (WebFetch 2026-02-23); full text is ~145 lines
- GitHub community discussion — social preview not settable via API (WebSearch, multiple sources agreeing 2026-02-23)

### Tertiary (LOW confidence)
- None

---

## Metadata

**Confidence breakdown:**
- COMM-01 (CONTRIBUTING.md gap): HIGH — direct file read confirms missing Docker workaround
- COMM-02 (CHANGELOG.md): HIGH — file confirmed absent (`ls` output)
- ~~COMM-03 (CODE_OF_CONDUCT.md)~~: Removed — do NOT create this file
- FIX-01 (metrics increment): HIGH — code read confirms PullErrors never called; one-line fix location identified
- FIX-02 (layer rates): HIGH — code read confirms LayerStatus fields unpopulated; fix location and pattern identified
- META-01 (GitHub metadata): HIGH — gh CLI confirmed, topics confirmed empty, social preview UI-only confirmed
- FIX-01 Error field population: LOW — unclear how/whether Error is ever set; needs agent code investigation

**Research date:** 2026-02-23
**Valid until:** 2026-03-25 (stable domain — community files and Prometheus API don't change)
