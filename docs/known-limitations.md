# Known Limitations

## containerd Only

Pulltrace reads the containerd gRPC socket directly. Nodes using CRI-O or Docker Engine (without containerd) are not supported.

## In-Memory State Only

Pull history is stored in-memory on the server with a configurable TTL (default 30 minutes). Restarting the server clears all pull history. There is no persistence layer.

## Single Cluster

One Pulltrace installation monitors one Kubernetes cluster. Multi-cluster federation is not supported in v0.1.0.

## No UI Authentication

The web UI is read-only and has no built-in authentication. It is designed for use behind an ingress controller with auth, within a private cluster network, or accessed via `kubectl port-forward`. Do not expose the UI directly to the public internet without an authentication proxy.

## Layer Events Not Streamed

The server maintains per-layer progress state internally, but SSE events for individual layer start/progress/completion events are not emitted in v0.1.0. Layer data is visible in the UI on each pull update cycle.
