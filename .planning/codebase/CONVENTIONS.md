# Coding Conventions

**Analysis Date:** 2026-02-23

## Naming Patterns

**Files (Go):**
- Package files use snake_case: `podwatcher.go`, `podwatcher_test.go`, `watcher.go`
- Main entry points: `cmd/pulltrace-{service}/main.go`
- Tests suffix with `_test.go` and belong in same package

**Files (JavaScript/JSX):**
- Components use PascalCase: `PullRow.jsx`, `LayerDetail.jsx`, `FilterBar.jsx`
- Utilities and hooks use camelCase: `hooks.js`, `utils.js`, `main.jsx`
- Index files: `index.css`

**Functions (Go):**
- Exported functions use PascalCase: `New()`, `NewRateCalculator()`, `ConfigFromEnv()`, `Poll()`
- Unexported functions use camelCase: `processReport()`, `mergeDigestPulls()`, `isContentDigest()`, `handleReport()`
- Receiver methods are descriptive: `(rl *rateLimiter) allow()`, `(pw *PodWatcher) Run()`, `(s *Server) handlePulls()`
- HTTP handlers follow pattern: `handleXxx` — e.g., `handleReport()`, `handleSSE()`, `handleHealthz()`

**Functions (JavaScript):**
- React components: PascalCase, arrow functions or function declarations
- Hooks: `usePulls()`, `useFilters()` — follow React hook naming convention
- Utility functions: camelCase — `formatBytes()`, `formatEta()`, `parseImageRef()`, `getPullStatus()`
- Helper functions within components: camelCase — `formatSpeed()`, `splitSpeed()`, `speedToBarPct()`

**Variables (Go):**
- Package-level constants are UPPER_SNAKE_CASE: `EventPullProgress`, `SchemaVersion`, `maxReportBodyBytes`, `rateLimitWindow`
- Local variables are camelCase: `now`, `report`, `pull`, `key`, `existing`
- Receiver shorthand uses single/double letters: `s *Server`, `pw *PodWatcher`, `rl *rateLimiter`, `rc *RateCalculator`

**Variables (JavaScript):**
- State variables: camelCase — `pulls`, `connected`, `filters`, `expandedIds`
- Ref variables: camelCase with Ref suffix — `eventSourceRef`, `maxPctRef`, `prevStartRef`
- Constants: UPPER_SNAKE_CASE for magic numbers only — `units = ['B', 'KB', ...]` is array, not constant
- Object keys: camelCase — `pull.imageRef`, `pull.startedAt`, `pull.completedAt`

**Types (Go):**
- Struct types are PascalCase: `PullStatus`, `LayerStatus`, `AgentReport`, `PullEvent`, `Config`, `Server`, `RateCalculator`
- Interface/enum-like types: `EventType` (string-based)
- Private structs: `rateLimiter`, `pullTracker`, `layerTracker`, `rateSample`

**JSON Tags (Go):**
- Use camelCase in JSON: `json:"imageRef"`, `json:"startedAt"`, `json:"completedAt"`
- Optional fields with omitempty: `json:"pull,omitempty"`, `json:"pods,omitempty"`

## Code Style

**Formatting:**
- Go: Standard gofmt (enforced by Go toolchain)
- JavaScript: No formal formatter detected, but consistent 2-space indentation observed
- Line length: Go typically follows 80-100 char guide via gofmt; JavaScript varies

**Linting:**
- Go: `//nolint:errcheck` directives used sparingly for intentional error ignoring (HTTP writes to response writer where error is not critical)
- No .eslintrc or .prettierrc found for JavaScript/React; formatting appears ad-hoc but consistent

## Import Organization

**Order (Go):**
1. Standard library imports (`context`, `encoding/json`, `fmt`, `log/slog`, etc.)
2. Third-party imports (github.com/..., k8s.io/...)
3. Local imports (github.com/d44b/pulltrace/...)

Example from `internal/server/server.go`:
```go
import (
	"context"
	"crypto/subtle"
	"encoding/json"
	...
	"github.com/d44b/pulltrace/internal/k8s"
	"github.com/d44b/pulltrace/internal/metrics"
	"github.com/d44b/pulltrace/internal/model"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)
```

**Order (JavaScript):**
1. React and third-party libraries (`import React, ...`, `import ReactDOM from ...`)
2. Local components (relative paths: `./components/`, `./hooks`, `./utils`)
3. Stylesheets last (`.css` imports)

Example from `web/src/App.jsx`:
```jsx
import React, { useState, useMemo } from 'react';
import FilterBar from './components/FilterBar';
import PullRow from './components/PullRow';
import { usePulls, useFilters } from './hooks';
```

**Path Aliases:**
- No aliases detected; all imports use direct paths
- JavaScript uses relative paths (`./`, `../`)
- Go uses full module paths

## Error Handling

**Patterns (Go):**

1. **Wrapping errors with context:**
```go
if err != nil {
    return fmt.Errorf("operation name: %w", err)
}
```
Examples from codebase:
- `fmt.Errorf("connecting to containerd: %w", err)`
- `fmt.Errorf("socket path validation: %w", err)`
- `fmt.Errorf("polling containerd: %w", err)`

2. **HTTP error responses:**
```go
http.Error(w, "error message", http.StatusBadRequest)
return  // Important: always return after http.Error
```
Used in `handleReport()`, `handleSSE()`, `handlePulls()`

3. **Silent error suppression (intentional):**
```go
//nolint:errcheck
```
Used only for non-critical operations like HTTP writes to ResponseWriter where lost writes are acceptable (SSE streaming, health checks). Also in test setup where errors are acceptable.

4. **Panics:**
- Not used. All errors are returned and handled at call site.

**Patterns (JavaScript):**

1. **Try-catch for JSON parsing:**
```jsx
try {
    const evt = JSON.parse(event.data);
    // ... process
} catch {
    // ignore parse errors
}
```

2. **Fetch error handling:**
```jsx
.catch(() => {})  // Silently ignore fetch failures
```
Acceptable because SSE has exponential backoff retry in `connect()` function.

3. **Optional chaining and nullish coalescing:**
```jsx
pull.pods?.some(...)  // Optional chaining
pull.imageRef?.toLowerCase()  // Safe access
filters.image || ''  // Fallback
```

## Logging

**Framework (Go):**
- `log/slog` with JSON handler (standard library, Go 1.21+)
- Levels: `LevelInfo`, `LevelDebug`, `LevelWarn`, `LevelError`
- Configured via `PULLTRACE_LOG_LEVEL` env var (default: "info")

**Patterns:**
```go
s.logger.Info("starting pulltrace server",
    "httpAddr", s.config.HTTPAddr,
    "metricsAddr", s.config.MetricsAddr,
)
s.logger.Warn("pulls map at capacity, dropping new pull",
    "node", report.NodeName,
    "image", pull.ImageRef,
)
s.logger.Error("pod watcher failed", "error", err)
```

**Framework (JavaScript):**
- `console` (standard; no library detected)
- Minimal logging in frontend; error handling prefers silent fails with retry
- No structured logging

## Comments

**When to Comment (Go):**
- Package-level comments on exported types: `// PullStatus describes the current state...`
- Complex logic with multiple conditions: Brief explanation of what/why
- Non-obvious uses of sync primitives: `// Cleanup old entries to prevent memory leak`
- Special handling: `// Keep one sample before the cutoff as an anchor for rate calculation.`

**Examples:**
```go
// isContentDigest reports whether ref is a raw containerd content digest
// rather than a human-readable image name.
func isContentDigest(ref string) bool { ... }

// Pulls absent from the report have completed on the node.
for key, pull := range s.pulls { ... }
```

**When NOT to Comment:**
- Self-explanatory variable assignments
- Standard error checking
- Obvious loop/conditional logic

**JSDoc/TSDoc:**
- Not used in Go (no type annotations needed; types are in code)
- Not used in JavaScript/React
- Function signatures are self-documenting via type system

## Comments in React/JavaScript

**Minimal but present:**
```jsx
// High-water mark: bar never goes backwards (prevents flicker...)
// Use high-water mark for display...
// For errors, show the real value...
```
Used to explain non-obvious UI logic and performance optimizations.

## Function Design

**Size (Go):**
- Functions typically 15-50 lines
- Longest functions: `processReport()` (~150 lines), `handleSSE()` (~70 lines)
- These are kept long because they manage significant state transitions or protocol handling
- Most helper functions are <30 lines

**Parameters:**
- Receivers on methods use pointer types: `(s *Server)`, `(pw *PodWatcher)`
- Config passed as struct: `cfg Config`
- HTTP handlers: `(w http.ResponseWriter, r *http.Request)`
- Avoid >3 non-receiver parameters; use structs for multiple related parameters

**Return Values:**
- Simple functions return single values: `Rate() float64`
- Error-returning functions: `(string, error)` or `error` only
- Constructor functions return pointer: `func New(...) *Server`
- Go convention: error is always last return value

**Size (JavaScript/React):**
- Component functions: 40-150 lines (including JSX markup)
- Utility functions: <20 lines
- Hooks: 20-60 lines
- Inline helpers within components for tight coupling: <10 lines

**Parameters:**
- React components use destructuring: `function PullRow({ pull, expanded, onToggle })`
- Utility functions: 1-2 parameters typical
- Callbacks passed as props for React: `onClick`, `onToggle`, `setFilter(key, value)`

**Return Values:**
- React components return JSX
- Hooks return objects: `{ pulls, connected }`, `{ filters, setFilter, filterPulls }`
- Utilities return single values or objects
- Callbacks invoke state setters

## Module Design

**Exports (Go):**
- Package-level functions and types are exported: `New()`, `PullStatus`, `Run()`, `ConfigFromEnv()`
- Helper functions not exported: `isContentDigest()`, `mergeDigestPulls()`, `prune()` (lowercase)
- Constants exported if part of API: `EventPullProgress`, `SchemaVersion`

**Internal organization:**
```
internal/model/      — Data types (event.go), utilities (rate.go)
internal/server/     — HTTP server and request handlers
internal/agent/      — Agent polling and reporting
internal/k8s/        — Kubernetes watcher
internal/containerd/ — Containerd polling
internal/metrics/    — Prometheus metric definitions
```

**Exports (JavaScript):**
```jsx
export function usePulls() { ... }      // Named export for hook
export function useFilters() { ... }    // Named export for hook
export default function App() { ... }   // Default export for main component
export default function PullRow(...) {} // Default export for component
export function formatBytes(...) {}     // Named export for utility
```

**Barrel Files:**
- Not used. Each file is imported directly.
- `web/src/` imports from specific files: `import { usePulls } from './hooks'`

## Constants and Magic Numbers

**Go:**
- Named constants for tuning: `maxActivePulls = 10000`, `maxSSEClients = 256`, `stalePullTimeout = 10 * time.Minute`
- Located near top of package or in Config struct
- Not inlined; always extracted for readability and maintainability

**JavaScript:**
- Arrays of units repeated but not extracted: `['B', 'KB', 'MB', 'GB']` appears in multiple places
- Magic numbers for UI: `Math.min(3, ...)`, `(Math.log10(bps) - 3) / 6 * 100` are local to function
- No global constants file; magic numbers stay close to usage

## Concurrency Patterns

**Go:**
- Mutexes protect shared state: `sync.RWMutex` on Server.pulls, PodWatcher.podsByImage
- Read locks for queries: `s.mu.RLock()` / `s.mu.RUnlock()` in `handlePulls()`
- Write locks for updates: `s.mu.Lock()` / `s.mu.Unlock()` in `processReport()`
- Goroutine-safe error channels: `make(chan error, buffered)` in `Run()`
- Context cancellation for graceful shutdown

**JavaScript:**
- No manual concurrency; React state management handles updates
- Fetch requests are naturally async (Promises)
- EventSource (SSE) is single-threaded event loop

## Naming Edge Cases

**Special identifiers:**
- Go: Receiver shorthand single letters acceptable (`s`, `pw`, `rc`, `rl`) when type is obvious from context
- JavaScript: Abbreviated refs acceptable (`eventSourceRef` not `eventSourceReference`)
- HTTP handler names always start with `handle`: `handleReport`, `handleSSE` (not `reportHandler`)

---

*Convention analysis: 2026-02-23*
