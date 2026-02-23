# Pulltrace

Real-time Kubernetes image pull progress monitor.

Pulltrace gives you live visibility into image pulls happening across your cluster nodes — layers, download speed, and ETA — in a single web UI.

## What It Does

When you deploy a new image to your cluster, Kubernetes schedules pods and nodes begin pulling image layers from the registry. This process is invisible by default: `kubectl get pods` shows `ContainerCreating` with no detail.

Pulltrace fills that gap. It runs a lightweight DaemonSet agent on each node that reads from the containerd socket, and a central server that aggregates the data and streams it to a browser UI.

## Key Features

- **Per-layer progress** — see which layers are downloading, their size, and download speed
- **ETA** — estimated time remaining based on current download rate
- **Multi-node** — all nodes in the cluster visible in one view
- **Pod correlation** — image references linked to pod names via the Kubernetes API
- **Prometheus metrics** — pull counts, durations, bytes, and error rates
- **Single `helm install`** — deploys as a DaemonSet + Deployment with a standard Helm chart

## Quick Start

See the [Installation](installation.md) page for the full install guide.

```bash
helm repo add pulltrace https://d44b.github.io/pulltrace/charts
helm repo update
helm install pulltrace pulltrace/pulltrace -n pulltrace --create-namespace
```

## Requirements

- Kubernetes 1.28+ with containerd runtime
- Helm 3
