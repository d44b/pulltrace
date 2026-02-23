# Installation

## Prerequisites

- **Kubernetes 1.28+** with **containerd** runtime (CRI-O is not supported — Pulltrace reads the containerd gRPC socket directly)
- **Helm 3**
- A namespace with the `pod-security.kubernetes.io/enforce=privileged` label — the agent DaemonSet mounts the host containerd socket via `hostPath`, which requires a privileged pod security profile

## Install

```bash
# 1. Add the Pulltrace Helm repository
helm repo add pulltrace https://d44b.github.io/pulltrace/charts
helm repo update

# 2. Create the namespace with the required pod security label
kubectl create namespace pulltrace
kubectl label namespace pulltrace \
  pod-security.kubernetes.io/enforce=privileged --overwrite

# 3. Install Pulltrace
helm install pulltrace pulltrace/pulltrace \
  -n pulltrace
```

## Access the UI

```bash
kubectl port-forward -n pulltrace svc/pulltrace-server 8080:8080
```

Open [http://localhost:8080](http://localhost:8080) in your browser.

## Upgrade

```bash
helm repo update
helm upgrade pulltrace pulltrace/pulltrace -n pulltrace
```

## Uninstall

```bash
helm uninstall pulltrace -n pulltrace
```

## See Also

- [Configuration](configuration.md) — environment variables for server and agent
- [Architecture](architecture.md) — how Pulltrace works
