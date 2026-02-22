# Security Policy

## Reporting Vulnerabilities

If you discover a security vulnerability in Pulltrace, please report it responsibly.

**Do not open a public GitHub issue for security vulnerabilities.**

Instead, please email **security@d44b.dev** with:

- A description of the vulnerability.
- Steps to reproduce it.
- The potential impact.
- Any suggested fixes (optional).

We will acknowledge receipt within 48 hours and aim to release a fix within 7 days for critical issues.

## Security Model

### Component Privileges

#### pulltrace-agent (DaemonSet)

The agent runs on every node and requires the following privileges:

| Privilege | Reason |
|---|---|
| Host path volume: containerd socket | Required to call `content.ListStatuses` for ingest progress |
| `hostPID: false` | Not required |
| `hostNetwork: false` | Not required |
| `privileged: false` | Not required |
| `readOnlyRootFilesystem: true` | Agent writes no files |
| No Kubernetes RBAC | Agent does not call the Kubernetes API |

The agent runs as a non-root user with a read-only root filesystem. It requires only a single volume mount for the containerd socket.

#### pulltrace-server (Deployment)

The server runs as a standard deployment and requires the following privileges:

| Privilege | Reason |
|---|---|
| RBAC: `get`, `list`, `watch` on `pods` | Correlate image pulls with waiting pods |
| No host paths | Server does not access the host filesystem |
| `readOnlyRootFilesystem: true` | Server writes no files |
| No privileged mode | Not required |

The server runs as a non-root user. Its RBAC is limited to read-only pod access for pod correlation. If `config.watchNamespaces` is set, RBAC can be scoped to specific namespaces.

### Containerd Socket Access

**Risk:** Access to the containerd socket grants significant node-level capabilities. A process with write access to the containerd socket can create, modify, and delete containers on the node.

**Mitigations:**

1. **Read-only operations.** The agent only calls `content.ListStatuses` and `content.Info` on the containerd API. It does not create containers, modify images, or perform any write operations.
2. **Non-root user.** The agent container runs as a non-root UID. The containerd socket permissions must allow this user to connect (typically group `root` or a dedicated `containerd` group).
3. **No privilege escalation.** The agent pod spec sets `allowPrivilegeEscalation: false`.
4. **Security context constraints.** On OpenShift or clusters with PodSecurityAdmission, the agent requires only the `baseline` profile, not `privileged`, with the exception of the host path volume mount.

**Recommendation:** Review your cluster's security policies before deploying. If your threat model does not allow host path mounts to the containerd socket, Pulltrace cannot be used.

### Network Security

- Agent-to-server communication uses cluster-internal HTTP. If your cluster does not enforce mTLS between pods (e.g., via a service mesh), this traffic is unencrypted.
- The server API and UI do not include authentication or authorization. If exposed outside the cluster via ingress, apply your own authentication (e.g., OAuth2 proxy, ingress auth annotations, or network policies).
- The Prometheus metrics endpoint (`/metrics`) is unauthenticated. Restrict access using network policies if needed.

### Supply Chain

- Container images are built via GitHub Actions and published to `ghcr.io/d44b/`.
- Images are built from official Go and Node.js base images.
- Go dependencies are managed via `go.mod` with checksums in `go.sum`.
- Node.js dependencies are managed via `package.json` with a lockfile.

## Supported Versions

| Version | Supported |
|---|---|
| 0.1.x | Yes |
