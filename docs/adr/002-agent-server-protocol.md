# ADR-002: HTTP POST from Agent to Server (Not gRPC)

## Status

Accepted

## Date

2025-01-10

## Context

The Pulltrace agent runs on every node and needs to send pull progress data to the central server. We need a communication protocol that is:

- Reliable across node-to-pod networking in Kubernetes.
- Simple to implement and debug.
- Resilient to transient network failures.
- Efficient enough for periodic small payloads (a few KB every 2 seconds per node).

The two main options considered were:

1. **HTTP POST** -- Agent periodically POSTs a JSON snapshot of all active pulls to the server.
2. **gRPC streaming** -- Agent opens a bidirectional or server-streaming gRPC connection to the server.

## Decision

Use **HTTP POST** for agent-to-server communication and **Server-Sent Events (SSE)** for server-to-UI streaming.

### Agent to Server

- The agent sends an `AgentReport` JSON payload via `POST /api/v1/report` at a configurable interval (default: 2 seconds).
- Each report is a full snapshot of all active pulls on that node, not a delta.
- The server is stateless with respect to agent connections -- it merges the latest report from each node.

### Server to UI

- The server exposes `GET /api/v1/events` as an SSE endpoint.
- Each event is a `PullEvent` JSON object representing a state change.
- Clients can reconnect at any time and receive the current state via `GET /api/v1/pulls`.

## Rationale

- **Simplicity.** HTTP POST requires no proto compilation, no streaming state management, and no bidirectional connection handling. The agent is a simple loop: snapshot, serialize, POST, sleep.
- **Debuggability.** Reports can be inspected with `curl -X POST` and responses are plain JSON. No special tooling is needed.
- **Resilience.** If a POST fails, the agent simply retries on the next interval with fresh data. There is no connection state to recover. Full-snapshot semantics mean the server always has the latest state without needing to replay missed deltas.
- **Adequate performance.** An `AgentReport` for a node pulling 5 images with 10 layers each is approximately 2-3 KB of JSON. At a 2-second interval, this is negligible network overhead.
- **SSE for downstream.** SSE is natively supported by browsers (EventSource API), requires no WebSocket upgrade negotiation, and works through HTTP proxies and load balancers without special configuration.

### Why not gRPC?

- gRPC adds proto compilation to the build process and requires generated code in both agent and server.
- Streaming gRPC connections need keepalive management, reconnection logic, and buffering for backpressure.
- For payloads under 10 KB at 2-second intervals, the overhead of HTTP connection setup is negligible (and HTTP/1.1 keep-alive eliminates it entirely).
- gRPC debugging requires `grpcurl` or similar tools rather than standard `curl`.

gRPC would be reconsidered if the payload size or frequency increased significantly (e.g., sub-second reporting with hundreds of concurrent pulls per node).

## Consequences

- The server must handle potentially bursty POST traffic if many nodes report simultaneously. This is mitigated by the small payload size and the ability to scale server replicas.
- There is a latency floor equal to the report interval (default 2 seconds). Pull progress updates are not instantaneous but are sufficient for human-readable UIs.
- The full-snapshot model means some bandwidth is used for pulls whose state has not changed. This is acceptable given the small payload size.
- SSE is unidirectional. If the server needs to send commands to the agent in the future (e.g., "increase report frequency"), a separate mechanism would be needed.
