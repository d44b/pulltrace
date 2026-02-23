# Codebase Concerns

**Analysis Date:** 2025-02-23

## Tech Debt

**Go 1.22 Local Build Incompatibility:**
- Issue: `go.mod` specifies `go 1.22.0`, but local development environment has Go 1.18 installed; local builds will fail with version parsing errors
- Files: `go.mod`
- Impact: Developers cannot build the project locally on systems with Go < 1.22; must use Docker/CI builds
- Fix approach: Either upgrade local Go version or use Docker for development builds. CI is not affected.

**Layer Event Definitions Without Emission:**
- Issue: Data model defines `PullEvent` with `EventPullProgress` and `EventPullCompleted` types, but layer-level events (`layer.started`, `layer.progress`, `layer.completed`) are never emitted
- Files: `internal/model/event.go` (lines 19-21 define events); dead code kept for future use
- Impact: Frontend cannot subscribe to layer-level events; architectural groundwork exists but is incomplete
- Fix approach: When implementing layer event streaming, populate event types and emit from `internal/server/server.go` during `processReport()` (lines 332-483)

**Prometheus Metric Never Incremented:**
- Issue: `PullErrors` counter metric is defined but never incremented anywhere in the codebase
- Files: `internal/metrics/metrics.go` (line 35); no calls to `metrics.PullErrors.Inc()`
- Impact: Metric is always zero; error tracking via Prometheus is non-functional
- Fix approach: Add `metrics.PullErrors.Inc()` when handling pull completion with errors. Currently, errors are stored in `PullStatus.Error` but not tracked in Prometheus.

## Missing Test Coverage

**No Agent Tests:**
- What's not tested: `internal/agent/agent.go` - the entire agent loop, containerd socket validation, HTTP reporting, rate limiting, token authentication
- Files: `internal/agent/agent.go` (191 lines)
- Risk: Agent can silently fail or have bugs in containerd polling, socket validation, or reporting without detection
- Priority: High - Agent is critical path; must connect to containerd and send reports reliably

**No Containerd Watcher Tests:**
- What's not tested: `internal/containerd/watcher.go` - layer tracking, image ref extraction, rate calculation, cleanup logic
- Files: `internal/containerd/watcher.go` (187 lines)
- Risk: Changes to layer aggregation or cleanup can break pull tracking without detection
- Priority: High - Core data collection logic; used by agent on every poll cycle

**No K8s Pod Watcher Integration Tests:**
- What's not tested: `internal/k8s/podwatcher.go` has unit tests for helpers (`normalizeImageRef`, `parseImageFromPullingMessage`, `parseImageFromPulledMessage`), but **no tests for pod/event watching loops, correlation logic, or race conditions**
- Files: `internal/k8s/podwatcher_test.go` (121 lines) - covers only parsing and helpers
- Risk: Pod-to-pull correlation can silently break; event watch reconnection logic untested; state mutation during concurrent reads/writes can fail
- Priority: Medium - K8s integration is optional (agent degrades gracefully without it)

**No Server Concurrency Tests:**
- What's not tested: `internal/server/server.go` has 351 lines of tests but **all are unit tests without concurrent SSE client scenarios, race conditions under high load, or cleanup correctness under load**
- Files: `internal/server/server_test.go` (351 lines)
- Risk: Race conditions in SSE broadcast (`broadcastSSE`), cleanup logic, and rate limiter can surface only in production
- Priority: Medium - Server is generally well-tested but concurrent scenarios need coverage

**No API Contract Tests:**
- What's not tested: Happy-path integration between agent and server; e.g., agent sends well-formed reports, server parses and correlates them, SSE events propagate
- Risk: Breaking changes to API contract (AgentReport schema, event structure) can go undetected
- Priority: Medium

**No Frontend Tests:**
- What's not tested: All React components (`App.jsx`, `PullRow.jsx`, `LayerDetail.jsx`, etc.) and hooks (`usePulls`, `useFilters`)
- Files: `web/src/**/*.jsx`, `web/src/hooks.js`
- Risk: UI bugs (incorrect formatting, race conditions with SSE state updates, layer expand/collapse) go undetected
- Priority: Low - Frontend is user-facing but relatively simple and issues are visible in manual testing

## Known Issues

**Layer Data Missing MediaType and BytesPerSec:**
- Symptoms: LayerDetail component attempts to render `layer.bytesPerSec` and `layer.mediaType`, but these fields are never populated by the server
- Files: `internal/server/server.go` (lines 389-405); `web/src/components/LayerDetail.jsx` (line 18)
- Trigger: Any pull with layers; check network tab for layer JSON
- Workaround: Frontend gracefully handles missing fields; MediaType and BytesPerSec are not critical for display

**SSE Clients Cap at 256 Connections:**
- Symptoms: 257th simultaneous SSE client receives "too many connections" (HTTP 503)
- Files: `internal/server/server.go` (lines 34, 512-515)
- Trigger: Large clusters or multiple concurrent dashboard users
- Workaround: Reconnect; limit dashboard connections or scale server replicas

**Rate Limiter Can Reject New Nodes at Capacity:**
- Symptoms: After 1024 unique nodes have reported, new node names are rejected with 429 (Too Many Requests)
- Files: `internal/server/server.go` (lines 28, 100-102)
- Trigger: Cluster > 1024 nodes; applies per server instance (not global)
- Workaround: Restart server or scale horizontally; old node entries clean up after 1 minute

## Fragile Areas

**Image Reference Normalization:**
- Files: `internal/k8s/podwatcher.go` (lines 354-370); critical for pod-to-pull correlation
- Why fragile: Heuristic-based approach (checking for `/` and `:`) can fail on unusual image refs. For example, malformed refs or multi-architecture image selectors may not normalize correctly
- Safe modification: Add comprehensive test cases before changing normalization logic; ensure round-trip: image → normalized → parsed back
- Test coverage: `normalizeImageRef()` has test cases but edge cases for unusual image formats not covered

**Message Parsing for Kubelet Events:**
- Files: `internal/k8s/podwatcher.go` (lines 372-402)
- Why fragile: Parsing "Pulling image \"%s\"" and "pulled image \"%s\"" patterns relies on exact kubelet message format; kubelet version changes may alter message format
- Safe modification: Add test cases for kubelet version-specific message formats before changing; consider fallback patterns
- Test coverage: Test cases cover standard formats but kubelet version variance not tested

**Merged Pull Key Generation:**
- Files: `internal/server/server.go` (lines 39, 342-344)
- Why fragile: `mergedPullSuffix = ":__merged__"` is used to group digest-based pulls, but if an actual image name happens to end with `:__merged__`, it could collide
- Safe modification: Use a more collision-resistant sentinel (UUID or constant not user-reachable)
- Test coverage: Test covers the merge logic but collision cases not tested

**Time-Based Stale Pull Detection:**
- Files: `internal/server/server.go` (lines 36-37, 622-627)
- Why fragile: Pulls are force-completed if no reports arrive for 10 minutes; in high-latency networks or temporary outages, legitimate pulls may be marked stale
- Safe modification: Make `stalePullTimeout` configurable via environment variable; add alerting when stale pulls are detected
- Test coverage: No tests for stale detection; can only verify via manual cluster testing

## Scaling Limits

**In-Memory Pull State:**
- Current capacity: 10,000 active pulls (`maxActivePulls` in `internal/server/server.go` line 31)
- Limit: Server will reject new pulls once 10k are in-flight; each pull occupies ~1KB (rough estimate)
- Scaling path: For larger clusters, increase `maxActivePulls` or implement persistent store (Redis/etcd) for pull state; distribute server across multiple instances

**SSE Client Connections:**
- Current capacity: 256 concurrent SSE clients (`maxSSEClients` in `internal/server/server.go` line 34)
- Limit: 257th client gets HTTP 503; applies per server instance
- Scaling path: Increase `maxSSEClients` if memory permits (~64KB per client in buffer); scale horizontally with multiple server replicas behind load balancer

**Rate Limit Tracking Map:**
- Current capacity: 1,024 unique node entries (`maxRateLimitEntries` in `internal/server/server.go` line 28)
- Limit: New node from cluster > 1024 nodes rejected until stale entries clean up (1 minute)
- Scaling path: For very large clusters, implement distributed rate limiting via shared cache (Redis) or increase entry limit; note that cleanup runs every 1 minute

**Frontend Pull History:**
- Current limit: Browser memory; no explicit cap on number of pulls stored in frontend state
- Limit: Very large clusters (1000+ nodes × 100+ pulls each) may degrade UI responsiveness
- Scaling path: Implement server-side pull history pagination; frontend should request chunks rather than holding all history

## Performance Bottlenecks

**Rate Calculation on Every Report:**
- Problem: `RateCalculator.Add()` and `Rate()` called during every `processReport()` for every pull; uses linear scan of samples within window
- Files: `internal/model/rate.go` (lines 24-50, 61-71); `internal/server/server.go` (lines 424-425)
- Cause: Rate window pruning is O(n) for each rate calculation; no caching of rate between reports
- Improvement path: Cache rate value; only recalculate if samples changed. For very high-frequency reports, consider fixed-size ring buffer instead of vector

**No Rate Limiting in Frontend SSE Subscription:**
- Problem: Frontend may process all SSE events without throttling; if server floods events at high frequency, frontend re-renders every event
- Files: `web/src/hooks.js` - usePulls() hook processes each SSE event
- Cause: No debounce or rate limit on state updates
- Improvement path: Add debounce/throttle to pull state updates in usePulls() hook; batch updates (e.g., max 10 updates/sec)

**Lock Contention in Server State:**
- Problem: `processReport()` holds `s.mu` lock for entire duration (lines 333-483 in `internal/server/server.go`); blocks reads and other writes
- Files: `internal/server/server.go` (lines 118-122 for lock definition, 333 for lock acquisition)
- Cause: Single large critical section instead of finer-grained locking
- Improvement path: Split state into per-node or per-pull locks; use read-write locks more granularly

## Security Considerations

**Token Auth Uses Constant-Time Comparison (Good):**
- Risk: None - constant-time comparison in place to prevent timing attacks
- Files: `internal/server/server.go` (line 254 uses `subtle.ConstantTimeCompare`)
- Current mitigation: Correctly implemented
- Recommendations: Ensure token is passed via secure channel (HTTPS); document token storage best practices in Helm chart

**Socket Path Validation for Containerd Connection:**
- Risk: Agent could be tricked into connecting to non-containerd UNIX sockets (e.g., Docker socket) if environment misconfigured
- Files: `internal/agent/agent.go` (lines 18-23, 181-191)
- Current mitigation: Whitelist of allowed prefixes (`/run/containerd/`, `/var/run/containerd/`)
- Recommendations: Sufficient for typical Kubernetes deployments; consider additional validation (socket ownership check) for hostile environments

**CSP Headers Configured (Good):**
- Risk: None - Content Security Policy restrictive
- Files: `internal/server/server.go` (lines 161-168)
- Current mitigation: Correctly configured CSP (`default-src 'self'`, no `unsafe-inline`)
- Recommendations: Ensure all frontend assets are served with correct MIME types; test CSP with real-world usage

**No Encryption in Transit for Agent Reports:**
- Risk: If HTTPS not used, agent reports (including container image details) transmitted in plaintext over network
- Files: `internal/agent/agent.go` (line 158); URL scheme depends on `PULLTRACE_SERVER_URL` env var
- Current mitigation: None in code; relies on operator to use HTTPS URL
- Recommendations: Document HTTPS requirement in deployment guide; consider enforcing HTTPS in code or Helm chart defaults

## Dependencies at Risk

**Containerd v2.0.4:**
- Risk: Major version; consider upgrading path if security patches lag
- Impact: Agent cannot connect if containerd API changes
- Migration plan: Monitor containerd releases; pin to 2.1.x or later once stable and tested

**Kubernetes Client v0.31.4:**
- Risk: Trailing edge; k8s 1.31 will EOL November 2025
- Impact: May miss features in newer K8s clusters or encounter API compatibility issues
- Migration plan: Test upgrade to latest client-go quarterly; consider v0.32.x or later in next release

**Prometheus Client v1.20.5:**
- Risk: Stable, widely used; low risk
- Impact: None expected
- Migration plan: No action needed; routine updates

## Test Coverage Gaps (High Priority)

**Agent Poll & Report Cycle:**
- What's not tested: `Agent.pollAndReport()`, `Agent.sendReport()`, error handling and retry logic
- Files: `internal/agent/agent.go` (lines 125-178)
- Risk: Agent can fail silently; broken reporting goes undetected
- Priority: High

**Containerd Content Store Polling:**
- What's not tested: `Watcher.Poll()`, layer state transitions, rate calculation for layers
- Files: `internal/containerd/watcher.go` (lines 74-130)
- Risk: Data collection failures undetected; corrupted or inconsistent layer state can propagate
- Priority: High

**Server Pull Completion Logic:**
- What's not tested: Pull completion when absent from report; metrics increments; SSE event broadcasting during high concurrency
- Files: `internal/server/server.go` (lines 451-482)
- Risk: Completed pulls not tracked accurately; metrics undercount; race conditions in SSE broadcast
- Priority: Medium

**Pod Correlation Edge Cases:**
- What's not tested: Pod deletion during correlation; namespace filtering with nil slice; concurrent pod updates
- Files: `internal/k8s/podwatcher.go` (lines 190-253)
- Risk: Pods can be orphaned in correlation map; concurrent mutations can cause data races
- Priority: Medium

---

*Concerns audit: 2025-02-23*
