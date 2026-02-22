# ADR-001: Use containerd Content Store API for Progress Tracking

## Status

Accepted

## Date

2025-01-10

## Context

Kubernetes does not expose image pull progress through any standard API. The CRI (Container Runtime Interface) provides `PullImage` as a synchronous RPC that returns only when the pull completes -- there is no streaming progress callback. To provide real-time pull visibility, we need to go below the CRI layer and interact with the container runtime directly.

The two dominant container runtimes in production Kubernetes clusters are:

- **containerd** -- used by default in EKS, GKE, AKS, k3s, kind, and most managed Kubernetes distributions.
- **CRI-O** -- used primarily in OpenShift and some bare-metal deployments.

Each runtime has different internal APIs for observing download progress.

### containerd approach

containerd v2 exposes a content store API via gRPC. When an image is being pulled, each layer is downloaded as an "ingest" in the content store. The `content.ListStatuses` API returns all active ingests with their `Offset` (bytes downloaded) and `Total` (expected size) fields, updated in real time.

### CRI-O approach

CRI-O uses the `containers/image` library internally. Progress tracking would require either parsing CRI-O's log output or using CRI-O-specific extension APIs, which are less stable and less documented.

## Decision

Pulltrace will use the **containerd v2 content store API** (`content.ListStatuses`) as its primary mechanism for tracking image pull progress.

The agent binary will:

1. Connect to the containerd gRPC socket (default: `/run/containerd/containerd.sock`).
2. Periodically call `content.ListStatuses` to enumerate active ingests.
3. Correlate ingests with image references using the `content.Info` metadata.
4. Compute per-layer and per-image progress, rates, and ETAs.

## Rationale

- **Market share.** containerd is the most widely deployed Kubernetes runtime. Supporting it first maximizes reach.
- **API quality.** The content store `ListStatuses` API provides exactly the data we need: bytes downloaded and total size per layer, updated in real time.
- **Stability.** The containerd v2 content API is a published, stable gRPC service with protobuf definitions.
- **Simplicity.** A single gRPC call returns all active downloads on the node. No log parsing or filesystem watching is required.
- **Extensibility.** The agent architecture is pluggable. A CRI-O backend can be added later behind the same internal interface without changing the server or UI.

## Consequences

- Pulltrace will only work on nodes running containerd v2. Nodes using CRI-O, Docker (dockershim), or other runtimes will not report pull progress.
- The agent requires access to the containerd socket, which means it needs a host path volume mount. This has security implications documented in SECURITY.md.
- Total layer sizes may not be known until the image manifest is resolved. The agent must handle the case where `Total` is zero and mark the size as unknown (`totalKnown: false`).
- If containerd changes its content store API in a future major version, the agent will need to be updated.
