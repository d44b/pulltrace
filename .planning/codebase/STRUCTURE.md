# Codebase Structure

**Analysis Date:** 2026-02-23

## Directory Layout

```
pulltrace/
├── cmd/                           # Executable entry points
│   ├── pulltrace-agent/           # Agent binary (runs on each node)
│   │   └── main.go
│   └── pulltrace-server/          # Server binary (runs once in cluster)
│       └── main.go
├── internal/                      # Private Go packages
│   ├── agent/                     # Agent polling and reporting
│   │   └── agent.go
│   ├── containerd/                # containerd API wrapper
│   │   └── watcher.go
│   ├── k8s/                       # Kubernetes integration
│   │   ├── podwatcher.go
│   │   └── podwatcher_test.go
│   ├── metrics/                   # Prometheus instrumentation
│   │   └── metrics.go
│   ├── model/                     # Shared data types
│   │   ├── event.go
│   │   ├── rate.go
│   │   └── rate_test.go
│   └── server/                    # HTTP server and aggregation logic
│       ├── server.go
│       └── server_test.go
├── web/                           # React frontend (embedded in server)
│   ├── src/
│   │   ├── App.jsx                # Main layout, pull list render
│   │   ├── hooks.js               # usePulls(), useFilters() hooks
│   │   ├── utils.js               # Formatting helpers
│   │   ├── main.jsx               # React entry point
│   │   └── components/
│   │       ├── PullRow.jsx        # Single pull row with expand
│   │       ├── LayerDetail.jsx    # Layer detail panel
│   │       ├── FilterBar.jsx      # Filter controls
│   │       ├── ProgressBar.jsx    # Progress bar component
│   │       └── SpeedGauge.jsx     # Speed indicator
│   ├── dist/                      # Built output (embedded in server)
│   ├── package.json               # npm dependencies
│   ├── vite.config.js             # Vite build config
│   └── embed.go                   # Go embed directive for dist/
├── charts/                        # Helm chart for K8s deployment
│   └── pulltrace/
│       ├── Chart.yaml
│       ├── values.yaml
│       ├── crds/                  # Custom resource definitions (if any)
│       └── templates/             # K8s manifests (StatefulSet, Service, etc.)
├── test/                          # Test fixtures and E2E tests
│   └── e2e/
├── docs/                          # Documentation
│   ├── adr/                       # Architecture Decision Records
│   └── schemas/                   # JSON schema files
├── hack/                          # Build scripts and utilities
├── Dockerfile.agent               # Container image for agent
├── Dockerfile.server              # Container image for server
├── Makefile                       # Build targets
├── go.mod, go.sum                 # Go module dependencies
└── README.md, SECURITY.md         # Project docs
```

## Directory Purposes

**cmd/:**
- Purpose: Executable entry points (two binaries)
- Contains: main() functions only; config parsing, context setup, signal handling
- Key files: `cmd/pulltrace-agent/main.go`, `cmd/pulltrace-server/main.go`

**internal/:**
- Purpose: Private Go packages; not importable by external consumers
- Contains: Core business logic, HTTP handlers, data models
- Key packages: `agent/`, `server/`, `model/`, `k8s/`, `containerd/`, `metrics/`

**internal/agent/:**
- Purpose: Agent binary logic (polling, reporting)
- Contains: Agent struct, config parsing, socket validation, poll/report loop
- Key types: `Agent`, `Config`
- Depends on: `internal/containerd/`, `internal/model/`, standard library

**internal/containerd/:**
- Purpose: Wrapper around containerd API for poll-based layer tracking
- Contains: Watcher struct, layer state machine, rate calculation
- Key types: `Watcher`, `pullTracker`, `layerTracker`
- Depends on: github.com/containerd/containerd/v2

**internal/k8s/:**
- Purpose: Kubernetes integration for pod-image correlation
- Contains: PodWatcher struct, K8s event listening, image discovery
- Key types: `PodWatcher`, K8s client init
- Depends on: k8s.io/client-go, kubernetes client library
- Key methods: `GetPullingImagesForNode()`, `GetWaitingImagesForNode()`, `GetPodsForImage()`

**internal/model/:**
- Purpose: Canonical data structures shared across agent, server, frontend
- Contains: PullStatus, PullEvent, LayerStatus, RateCalculator
- Key types: All JSON-serializable structs
- Depends on: Standard library only
- **No business logic here—only data definitions**

**internal/server/:**
- Purpose: HTTP server, aggregation logic, SSE broadcast
- Contains: Server struct, handlers, report processing, rate limiter, cleanup loop
- Key types: `Server`, `rateLimiter`, `Config`
- Depends on: `internal/k8s/`, `internal/model/`, `internal/metrics/`, standard library

**internal/metrics/:**
- Purpose: Prometheus metric registration and exported variables
- Contains: Gauge and counter definitions only
- Key variables: `PullsActive`, `PullsTotal`, `PullDurationSeconds`, `SSEClients`, etc.
- Depends on: prometheus/client_golang

**web/:**
- Purpose: React frontend source code and built output
- Contains: JSX components, hooks, build config
- Build output: `web/dist/` (embedded in server via embed.go)
- Key files: `src/App.jsx`, `src/hooks.js`, `src/components/*.jsx`

**web/src/:**
- Contains: TypeScript/JSX source
- `App.jsx`: Main component; renders header, speed panel, filters, pull list
- `hooks.js`: `usePulls()` (initial fetch + SSE), `useFilters()` (filter state + logic)
- `components/*.jsx`: Reusable components (PullRow, LayerDetail, FilterBar, ProgressBar, SpeedGauge)
- `utils.js`: Formatting functions (bytes, ETA, image parsing)

**web/dist/:**
- Purpose: Built React app (Vite output)
- Generated: Yes (from `npm run build`)
- Committed: No (in .gitignore, built in CI)
- Embedded: Yes (via `web/embed.go` into server binary)

**charts/:**
- Purpose: Helm chart for Kubernetes deployment
- Contains: Service account, RBAC, StatefulSet for server, DaemonSet for agent
- Key templates: Deployment manifests with env var injection
- CRDs: Any custom resources needed by pulltrace

**test/e2e/:**
- Purpose: End-to-end test fixtures
- Contains: Test data, integration tests (if any)

**docs/:**
- Purpose: Project documentation
- `adr/`: Architecture Decision Records explaining design choices
- `schemas/`: JSON schema files for PullEvent, AgentReport, etc.

**hack/:**
- Purpose: Development scripts and utilities
- Contains: Build helpers, linters, generators (typically Bash or Go)

## Key File Locations

**Entry Points:**
- `cmd/pulltrace-agent/main.go`: Agent binary entrypoint
- `cmd/pulltrace-server/main.go`: Server binary entrypoint
- `web/src/main.jsx`: React app entrypoint
- `web/embed.go`: Embeds web/dist into server binary

**Configuration:**
- `go.mod`: Go module definition with versions
- `web/package.json`: npm dependencies
- `web/vite.config.js`: Vite build configuration
- `Makefile`: Build targets (docker, test, fmt)
- Dockerfile.agent, Dockerfile.server: Container build configs
- `charts/pulltrace/values.yaml`: Helm defaults

**Core Logic:**
- `internal/model/event.go`: Data model definitions
- `internal/server/server.go`: Aggregation, HTTP handlers, SSE broadcast
- `internal/agent/agent.go`: Poll and report loop
- `internal/containerd/watcher.go`: Layer tracking state machine
- `internal/k8s/podwatcher.go`: Pod watcher event listener

**Testing:**
- `internal/model/rate_test.go`: RateCalculator tests
- `internal/server/server_test.go`: Server aggregation tests
- `internal/k8s/podwatcher_test.go`: PodWatcher tests
- `test/e2e/`: End-to-end test fixtures (if present)

## Naming Conventions

**Go Files:**
- Package name: Lowercase without underscores (`agent`, `server`, `containerd`)
- Filename: Lowercase, underscores only for `_test.go` suffix
- Examples: `server.go`, `agent.go`, `podwatcher.go`, `event.go`, `server_test.go`

**Go Types:**
- Exported: PascalCase (`Server`, `Agent`, `PullStatus`, `LayerStatus`)
- Unexported: camelCase (`pullTracker`, `layerTracker`, `rateLimiter`)
- Interfaces: PascalCase, typically `Reader`, `Writer`, or domain-specific

**Go Functions/Methods:**
- Exported: PascalCase (`Run`, `Poll`, `New`, `Close`)
- Unexported: camelCase (`processReport`, `pollAndReport`, `cleanup`)
- Constructors: `NewX` pattern (`NewWatcher`, `NewRateCalculator`, `NewServer`)

**Go Variables:**
- Exported constants: UPPER_SNAKE_CASE or PascalCase (`SchemaVersion`, `maxActivePulls`)
- Maps/slices: Plural or descriptive (`pulls`, `layers`, `sseClients`)
- Channels: Often suffixed with `Ch` (`errCh`, `stopCh`)

**React Components:**
- Filename: PascalCase.jsx (`App.jsx`, `PullRow.jsx`, `LayerDetail.jsx`)
- Component function: PascalCase (`App`, `PullRow`, `LayerDetail`)
- Helper functions: camelCase (`formatSpeed`, `formatBytes`, `parseImageRef`)
- Hooks: Prefixed `use` (`usePulls`, `useFilters`)

**React State:**
- State variables: camelCase (`pulls`, `connected`, `filters`, `expandedIds`)
- Setter functions: `set{State}` pattern (`setPulls`, `setConnected`, `setFilters`)

**JSON Keys:**
- camelCase throughout (Go: `json:"camelCase"` tags on structs)
- Examples: `imageRef`, `nodeName`, `bytesPerSec`, `layerCount`, `startedAt`, `completedAt`

**Environment Variables:**
- Prefix: `PULLTRACE_`
- Upper with underscores: `PULLTRACE_NODE_NAME`, `PULLTRACE_SERVER_URL`, `PULLTRACE_LOG_LEVEL`

**Directories:**
- Lowercase: `cmd`, `internal`, `web`, `charts`, `docs`, `hack`, `test`
- Package subdirs: Lowercase, hyphen-separated if needed (`pulltrace-agent`, `pulltrace-server`)

## Where to Add New Code

**New Feature:**

- **API endpoint:** Add handler to `internal/server/server.go`, register in `Run()` method
- **Server-side logic:** Create new file in `internal/` appropriate package or new package
- **Agent logic:** Add to `internal/agent/agent.go` or new `internal/agent/*.go` file
- **Data model:** Add to `internal/model/event.go` if top-level, or new file if isolated
- **Frontend component:** Create in `web/src/components/{FeatureName}.jsx`
- **Frontend hook:** Add to `web/src/hooks.js` or create `web/src/hooks/{featureName}.js`

**New Component/Module:**

- **Backend package:** Create `internal/{package}/` directory with `*.go` files
- **Follow structure:** Exactly one main type per large file; use `New{Type}()` constructor
- **Tests:** Colocate as `*_test.go` in same package
- **Frontend component:** File per component in `web/src/components/`

**Utilities:**

- **Backend helpers:** `internal/model/` for data utilities, `internal/{package}/` for domain-specific
- **Frontend helpers:** `web/src/utils.js` for formatting/parsing; component-local for render-only logic
- **Shared constants:** Backend in `internal/model/event.go` (e.g., `SchemaVersion`), frontend in `web/src/utils.js`

## Special Directories

**web/dist/:**
- Purpose: Built React application
- Generated: Yes (via `npm run build` in web/)
- Committed: No (listed in .gitignore)
- Embedded: Yes (Go embed directive in `web/embed.go` pulls this into server binary)

**internal/model/:**
- Purpose: **No business logic—only data types**
- Invariant: Should depend only on standard library
- Reason: Maximal compatibility with frontend (JSON serialization) and test code

**charts/:**
- Purpose: Production deployment via Helm
- Contains: K8s manifests as Helm templates
- Entry: `Chart.yaml` defines version and metadata
- Defaults: `values.yaml` provides all helm values

**docs/adr/:**
- Purpose: Architecture Decision Records
- Format: Markdown, typically ADR-NNNN-description.md
- Usage: Document why a design choice was made (mergeDigestPulls strategy, SSE vs WebSocket, etc.)

**test/e2e/:**
- Purpose: End-to-end tests (if present)
- Typically: Docker Compose or K8s manifests for local testing
- Execution: Often run in CI pipeline, not in unit test suite
