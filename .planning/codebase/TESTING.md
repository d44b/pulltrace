# Testing Patterns

**Analysis Date:** 2026-02-23

## Test Framework

**Runner:**
- Go: `testing` (standard library, built into `go test`)
- JavaScript: Not detected — no test framework configured or tests present

**Config (Go):**
- `go.mod` with Go 1.22.0
- Tests run via `go test ./...` (standard)
- No dedicated test config file (go.mod specifies version only)

**Assertion Library (Go):**
- None — manual assertions using standard library `if` statements
- Pattern: `if got != expected { t.Errorf("...") }`

**Run Commands:**
```bash
go test ./...                    # Run all tests
go test -v ./...                 # Verbose output
go test -race ./...              # Race detector enabled
go test ./internal/server        # Test specific package
go test -run TestName ./pkg      # Run specific test
```

## Test File Organization

**Location (Go):**
- Co-located in same package as source
- Pattern: `filename_test.go` alongside `filename.go`

**Examples:**
- `internal/server/server.go` → `internal/server/server_test.go`
- `internal/model/rate.go` → `internal/model/rate_test.go`
- `internal/k8s/podwatcher.go` → `internal/k8s/podwatcher_test.go`

**Naming:**
- Test files end with `_test.go`
- Benchmark files would use `_benchmark.go` (not present)
- Tests are in same package (not `_test` package suffix)

## Test Structure

**Suite Organization (Go):**
```go
package server  // Same package as source

import (
    "testing"
    ...
)

// Setup helper
func newTestServer() *Server {
    return New(Config{...}, nil)  // Create test instance
}

// Test function groups by area
func TestHandleReport_MethodNotAllowed(t *testing.T) { ... }
func TestHandleReport_InvalidJSON(t *testing.T) { ... }
func TestHandleReport_TokenAuth(t *testing.T) { ... }

// Helper for repeatable setup
func postReport(t *testing.T, s *Server, report model.AgentReport, token string) *httptest.ResponseRecorder {
    t.Helper()
    // ... setup code
    return w
}
```

**Patterns:**

1. **Setup helper (factory pattern):**
```go
func newTestServer() *Server {
    return New(Config{
        HTTPAddr:    ":0",
        MetricsAddr: ":0",
        LogLevel:    "error",
        HistoryTTL:  30 * time.Minute,
    }, nil)
}
```
No explicit teardown needed (Go GC handles cleanup). Used in most tests to create fresh instances.

2. **HTTP testing with httptest:**
```go
req := httptest.NewRequest(http.MethodPost, "/api/v1/report", body)
w := httptest.NewRecorder()
s.handleReport(w, req)
if w.Code != http.StatusOK {
    t.Errorf("expected 200, got %d", w.Code)
}
```
Standard Go HTTP testing pattern using `httptest` package.

3. **Assertion pattern (no assertion library):**
```go
if got != expected {
    t.Errorf("expected %v, got %v", expected, got)
}
if got {
    t.Error("should be false")
}
if err != nil {
    t.Fatalf("unexpected error: %v", err)
}
```
- `t.Error()` - test continues
- `t.Errorf()` - formatted message, test continues
- `t.Fatalf()` - formatted message, test stops immediately

## Mocking

**Framework:**
- None detected — no mock library imported
- Mocking done by dependency injection through constructors

**Patterns:**

1. **Nil dependencies for testing:**
```go
// In server_test.go
func newTestServer() *Server {
    return New(Config{...}, nil)  // nil webFS, nil podWatcher (optional)
}
```
The Server.podWatcher is optional (checked with `if s.podWatcher != nil`), so tests can pass nil to disable that subsystem.

2. **In-memory data structures:**
```go
// Tests access server state directly
s.mu.RLock()
defer s.mu.RUnlock()
pull := s.pulls["node1:nginx:latest"]
if pull == nil { t.Fatal("pull not found") }
```
No mocking library; instead inspect internal state directly.

3. **Helper functions for test input:**
```go
func postReport(t *testing.T, s *Server, report model.AgentReport, token string) *httptest.ResponseRecorder {
    t.Helper()
    body, err := json.Marshal(report)
    // ... build request
    return w
}
```
Helper encapsulates boilerplate request building; test calls it multiple times.

## Fixtures and Factories

**Test Data (Go):**
```go
// Minimal structs created inline
report := model.AgentReport{
    NodeName: "node1",
    Timestamp: time.Now(),
    Pulls: []model.PullState{
        {
            ImageRef: "nginx:latest",
            StartedAt: time.Now(),
            TotalKnown: true,
            Layers: []model.LayerState{
                {Digest: "sha256:layer1", TotalBytes: 1000, DownloadedBytes: 500, TotalKnown: true},
            },
        },
    },
}
```

**Location:**
- Test data defined inline in test functions (no separate fixtures file)
- Repeated patterns extracted into helper functions like `newTestServer()` and `postReport()`
- No factory library; Go struct literals serve as factories

**Builder pattern for complex objects:**
- Not used; tests build full struct literals or rely on zero values
- Simplicity preferred over advanced patterns

## Coverage

**Requirements:**
- Not enforced (no `.coverprofile`, no CI gates on coverage percentage)
- Coverage calculation available via `go test -cover ./...`

**View Coverage:**
```bash
go test -cover ./...                    # Per-package coverage %
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out        # Visual HTML report
```

## Test Types

**Unit Tests (Go):**
- **Scope:** Individual functions/methods in isolation
- **Approach:** Direct calls with known inputs, validate outputs
- **Examples:**
  - `TestMergeDigestPulls_*` - pure functions with deterministic inputs/outputs
  - `TestRateLimiter_*` - state machine methods
  - `TestRateCalculator_*` - rate calculation math
  - `TestHandleReport_*` - HTTP handler responses

**Integration Tests (Go):**
- **Scope:** Multiple components working together
- **Approach:** Create server, post report, verify internal state
- **Examples:**
  - `TestProcessReport_TracksNewPull` - reports → internal state update
  - `TestProcessReport_CompletesAbsentPull` - multi-report sequence
  - `TestProcessReport_MergedDigestPull` - merging logic
  - `TestIsContentDigest` and `normalizeImageRef` - image parsing

**E2E Tests:**
- **Status:** Not present
- **Would include:** Full agent → server → frontend flow
- **Missing:** No Docker Compose, no test cluster, no browser automation

## Common Patterns

**Assertion with Context Table:**
```go
func TestIsContentDigest(t *testing.T) {
    cases := []struct {
        ref    string
        expect bool
    }{
        {"sha256:abc", true},
        {"nginx:latest", false},
        {"__pulling__", false},
    }
    for _, c := range cases {
        if got := isContentDigest(c.ref); got != c.expect {
            t.Errorf("isContentDigest(%q) = %v, want %v", c.ref, got, c.expect)
        }
    }
}
```
Table-driven tests for parameterized cases. Common Go pattern.

**Time-based Testing (Timing-sensitive code):**
```go
func TestRateCalculator_Rate(t *testing.T) {
    rc := NewRateCalculator(10 * time.Second)
    rc.Add(0)
    time.Sleep(100 * time.Millisecond)
    rc.Add(1000)
    rate := rc.Rate()
    // 1000 bytes in ~100ms = ~10000 B/s; allow generous bounds for slow CI
    if rate > 200000 {
        t.Errorf("rate %f seems unreasonably high", rate)
    }
}
```
Uses `time.Sleep()` to create passage of time. Tests allow generous bounds because CI is slower than local machines.

**State Inspection:**
```go
func TestProcessReport_TracksNewPull(t *testing.T) {
    s := newTestServer()
    s.processReport(model.AgentReport{...})

    s.mu.RLock()
    defer s.mu.RUnlock()

    pull := s.pulls["node1:nginx:latest"]
    if pull == nil { t.Fatal("pull not found") }
    if pull.TotalBytes != 1000 { t.Errorf("...") }
}
```
Tests inspect internal server state directly after calling methods. No mocking; direct state access.

**Testing Error Conditions:**
```go
func TestHandleReport_InvalidJSON(t *testing.T) {
    s := newTestServer()
    req := httptest.NewRequest(http.MethodPost, "/api/v1/report", bytes.NewReader([]byte("not json")))
    w := httptest.NewRecorder()
    s.handleReport(w, req)
    if w.Code != http.StatusBadRequest {
        t.Errorf("expected 400, got %d", w.Code)
    }
}
```
Send malformed input, verify error response code.

**Testing Authorization:**
```go
func TestHandleReport_TokenAuth(t *testing.T) {
    s := newTestServer()
    s.config.AgentToken = "secret"

    if w := postReport(t, s, report, ""); w.Code != http.StatusUnauthorized {
        t.Errorf("no token: expected 401, got %d", w.Code)
    }
    if w := postReport(t, s, report, "wrong"); w.Code != http.StatusUnauthorized {
        t.Errorf("wrong token: expected 401, got %d", w.Code)
    }
    if w := postReport(t, s, report, "secret"); w.Code != http.StatusOK {
        t.Errorf("correct token: expected 200, got %d", w.Code)
    }
}
```
Test both negative cases (no token, wrong token) and positive case (correct token) in one function.

## Test Coverage Summary

**Well-tested packages:**
- `internal/server/server.go` - 351 lines of tests covering HTTP handlers, report processing, merging, rate limiting
- `internal/model/rate.go` - 84 lines of tests covering rate calculation edge cases (empty, single sample, negative deltas, ETA, window expiry)
- `internal/k8s/podwatcher.go` - 121 lines of tests covering image parsing, namespace filtering, stale cleanup

**Untested packages (No test files found):**
- `internal/agent/agent.go` - Agent loop, config, polling, and reporting functions not tested
- `internal/containerd/watcher.go` - Containerd polling and state tracking not tested
- `internal/metrics/metrics.go` - Prometheus metrics definitions not tested
- `cmd/pulltrace-server/main.go` - Server startup not tested
- `cmd/pulltrace-agent/main.go` - Agent startup not tested

**Frontend (JavaScript/React):**
- Zero tests present
- No test framework configured
- No test files in `web/src/`

## Testing Gaps and Recommendations

**Go testing gaps:**
1. Agent doesn't have tests — `pollAndReport()`, `sendReport()` not covered
2. Containerd watcher untested — `Poll()`, content store interaction not validated
3. No integration tests combining agent + server
4. No concurrency/race condition tests (despite using goroutines and mutexes)

**Frontend testing gaps:**
1. No test framework (Jest, Vitest) configured
2. No tests for `usePulls()` hook (SSE connection, state updates)
3. No tests for `useFilters()` hook (filter application)
4. No component tests for `PullRow`, `LayerDetail`, `FilterBar`
5. No tests for utility functions (`formatEta`, `formatBytes`, `parseImageRef`)

**Recommended approach for new tests:**
- **Go:** Follow existing patterns — use `testing` stdlib, table-driven tests, dependency injection via constructors
- **JavaScript:** Install Jest or Vitest, configure in `web/package.json`, follow React Testing Library patterns
- **No mocking library needed** (Go) — use dependency injection and nil interfaces instead
- **Test location:** Same directory as source, suffix with `_test.go` or `.test.js`

---

*Testing analysis: 2026-02-23*
