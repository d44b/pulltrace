# Contributing to Pulltrace

Thank you for your interest in contributing to Pulltrace. This document explains how to set up a development environment, run tests, and submit changes.

## Prerequisites

- [Go 1.22+](https://go.dev/dl/)
- [Node.js 20+](https://nodejs.org/) and npm
- [Docker](https://docs.docker.com/get-docker/) (for building images)
- [Helm 3](https://helm.sh/docs/intro/install/) (for chart development)
- [kubectl](https://kubernetes.io/docs/tasks/tools/) (for testing against a cluster)
- A Kubernetes cluster with containerd (e.g., [kind](https://kind.sigs.k8s.io/), [k3s](https://k3s.io/), or [minikube](https://minikube.sigs.k8s.io/))

## Repository Structure

```
pulltrace/
  cmd/
    agent/          # Agent entrypoint
    server/         # Server entrypoint
  internal/
    agent/          # Agent logic (containerd client, reporter)
    server/         # Server logic (API, SSE, aggregator)
    watcher/        # Kubernetes pod watcher
    model/          # Shared data types
  web/              # React frontend (Vite)
  charts/pulltrace/ # Helm chart
  docs/             # Documentation and ADRs
  hack/             # Development scripts
  test/             # Integration tests
```

## Development Setup

### Clone and build

```bash
git clone https://github.com/d44b/pulltrace.git
cd pulltrace
```

### Building with Docker (Go 1.22 required locally)

The project's `go.mod` requires Go 1.22.0. If your local Go installation is older (e.g., Go 1.18), `make build` will fail with a toolchain version error. Use Docker to build instead:

```bash
# Build Go binaries via Docker
docker run --rm -v $(pwd):/app -w /app golang:1.22-alpine go build ./...

# Run tests via Docker
docker run --rm -v $(pwd):/app -w /app golang:1.22-alpine go test ./...
```

CI uses `golang:1.22-alpine` via the Dockerfile, so pushed code will build correctly in CI even if your local build fails.

```bash
# Build Go binaries
make build

# Build the frontend
cd web && npm install && npm run build && cd ..

# Run tests
make test
```

### Running locally

For development, run the server and frontend separately:

```bash
# Terminal 1: Start the server (without a real agent)
go run ./cmd/server --log-level=debug

# Terminal 2: Start the frontend dev server with hot reload
cd web && npm run dev
```

The Vite dev server proxies `/api` to `localhost:8080` (the Go server). Open `http://localhost:5173` to see the UI.

### Running against a cluster

```bash
# Build images and load into kind
make docker-build
kind load docker-image ghcr.io/d44b/pulltrace-agent:dev
kind load docker-image ghcr.io/d44b/pulltrace-server:dev

# Install with Helm
helm install pulltrace ./charts/pulltrace \
  -n pulltrace --create-namespace \
  --set agent.image.tag=dev \
  --set server.image.tag=dev

# Port-forward to access the UI
kubectl port-forward -n pulltrace svc/pulltrace-server 8080:8080
```

## Running Tests

```bash
# Unit tests
make test

# Unit tests with coverage
make test-cover

# Lint
make lint

# All checks (lint + test)
make check
```

## Code Style

- **Go:** Format with `gofmt`. Lint with `golangci-lint`. Follow standard Go conventions.
- **JavaScript/JSX:** No specific formatter is configured. Keep it consistent with existing code.
- **Commits:** Write clear commit messages. Use the imperative mood (e.g., "Add layer progress tracking" not "Added layer progress tracking").

## Submitting Changes

1. **Fork** the repository and create a feature branch from `main`.
2. **Make your changes.** Keep commits focused and atomic.
3. **Add tests** for new functionality.
4. **Run `make check`** to verify lint and tests pass.
5. **Open a pull request** against `main` with a clear description of what the change does and why.

### PR Guidelines

- Keep PRs focused. One logical change per PR.
- Update documentation if your change affects the API, configuration, or user-facing behavior.
- Add or update tests. PRs that decrease test coverage without justification will be asked for revisions.
- If your PR addresses an open issue, reference it in the description (e.g., "Fixes #42").

## Reporting Issues

Open an issue at [github.com/d44b/pulltrace/issues](https://github.com/d44b/pulltrace/issues) with:

- A clear title describing the problem or feature request.
- Steps to reproduce (for bugs).
- Expected vs. actual behavior (for bugs).
- Your environment: Kubernetes version, containerd version, Pulltrace version.

