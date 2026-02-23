# Contributing

Thank you for your interest in contributing to Pulltrace.

## Getting Started

See [CONTRIBUTING.md](https://github.com/d44b/pulltrace/blob/main/CONTRIBUTING.md) in the repository root for:

- Development prerequisites (Go 1.22+, the Docker workaround for local machines with older Go)
- Build instructions (`make` and `docker build`)
- PR guidelines

## Reporting Issues

Open an issue on [GitHub](https://github.com/d44b/pulltrace/issues). Include your Kubernetes version, containerd version, and the Pulltrace version (`helm list -n pulltrace`).

## Architecture Overview

Before contributing, reading the [Architecture](architecture.md) page and the [Architecture Decision Records](adr/001-runtime-containerd.md) will give you context on why key design choices were made.
