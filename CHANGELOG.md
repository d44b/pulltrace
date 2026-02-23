# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-02-23

### Added
- Real-time container image pull progress monitoring via containerd socket
- Layer-by-layer download progress with bytesPerSec and estimated time remaining
- Kubernetes pod correlation — matches active pulls to waiting pods by image name
- Prometheus metrics: `pulltrace_pulls_active`, `pulltrace_pulls_total`, `pulltrace_pull_duration_seconds`, `pulltrace_pull_bytes_total`, `pulltrace_pull_errors_total`
- Server-Sent Events (SSE) streaming for live browser updates without polling
- React + Vite web UI with per-pull expandable layer detail view
- Helm chart for Kubernetes DaemonSet + server deployment

[Unreleased]: https://github.com/d44b/pulltrace/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/d44b/pulltrace/releases/tag/v0.1.0
