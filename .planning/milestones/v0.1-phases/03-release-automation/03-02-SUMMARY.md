---
phase: 03-release-automation
plan: "02"
subsystem: ci-cd
tags: [github-actions, github-release, softprops, helm, artifact-sharing]
dependency_graph:
  requires: [03-01]
  provides: [github-release-job, release-pipeline-complete]
  affects: [.github/workflows/ci.yml]
tech_stack:
  added: [softprops/action-gh-release@v2, actions/download-artifact@v4]
  patterns: [needs-dependency-chain, artifact-sharing, inline-release-body]
key_files:
  created: []
  modified:
    - .github/workflows/ci.yml
decisions:
  - "needs: [helm-release] not needs: [docker] — release appears only after Helm index is live"
  - "Inline body (not body_path: CHANGELOG.md) — avoids dumping [Unreleased] section and raw 23-line CHANGELOG header"
  - "Artifact reuse via download-artifact@v4 — no re-packaging, no second sed -i"
  - "make_latest: true — marks release as the latest on the GitHub Releases page"
  - "No concurrency block on github-release — already serialized via needs: [helm-release] which holds deploy-gh-pages slot"
metrics:
  duration: 2min
  completed_date: "2026-02-23"
  tasks_completed: 1
  files_modified: 1
---

# Phase 3 Plan 2: GitHub Release Job Summary

`github-release` job using `softprops/action-gh-release@v2` appended to ci.yml, wired after `helm-release` via `needs:` + artifact sharing, publishing a GitHub Release with inline install commands and CHANGELOG link on every semver tag push.

## What Was Done

### Task 1: Add github-release job to ci.yml

**File:** `.github/workflows/ci.yml`

Appended a new `github-release` job after the `helm-release` job's closing block. The job is 48 lines of new YAML. No existing jobs were modified.

**Job structure:**

```yaml
github-release:
  name: GitHub Release
  needs: [helm-release]
  runs-on: ubuntu-latest
  if: startsWith(github.ref, 'refs/tags/v')
  steps:
    - uses: actions/checkout@v4
    - name: Extract version        # strips 'v' prefix → "0.1.0"
    - name: Download chart artifact  # downloads 'helm-chart' from helm-release
    - name: Create GitHub Release    # softprops/action-gh-release@v2
```

## Job Integration: How github-release Connects to the Pipeline

### Full job execution sequence

```
lint-test ──┐
build-ui  ──┼──► docker ──► helm-release ──► github-release
helm-lint ──┘
```

`docker` runs only after all three CI checks pass. `helm-release` runs after `docker` and only on semver tags. `github-release` runs after `helm-release` completes — meaning the Helm classic index (`index.yaml`) and OCI push are both live before the GitHub Release is created.

### needs + artifact sharing (no re-packaging)

The `helm-release` job uploads the packaged `.tgz` via `actions/upload-artifact@v4` with `name: helm-chart`. The `github-release` job downloads it with `actions/download-artifact@v4` using the same `name: helm-chart`. This avoids running a second `helm package` + `sed -i` in the github-release job — the artifact from helm-release is the single authoritative packaged chart.

### Permissions inheritance

`contents: write` is set at the workflow level (added in 03-01). The `github-release` job inherits this permission — no job-level permissions block needed.

## Inline Release Body Content

The release body includes:

1. **Title:** `## Pulltrace v{version}`
2. **One-liner:** "Real-time Kubernetes image pull progress monitor."
3. **Classic Helm repo install:**
   ```bash
   helm repo add pulltrace https://d44b.github.io/pulltrace/charts
   helm repo update
   helm install pulltrace pulltrace/pulltrace --version {version}
   ```
4. **OCI install:**
   ```bash
   helm install pulltrace oci://ghcr.io/d44b/charts/pulltrace --version {version}
   ```
5. **CHANGELOG link:** `[CHANGELOG.md](https://github.com/d44b/pulltrace/blob/main/CHANGELOG.md)`

The body is inline YAML (`body:`) rather than `body_path: CHANGELOG.md`. Using `body_path` would dump the full 23-line CHANGELOG verbatim (including `[Unreleased]` and prior sections) into the release body.

## Key Configuration Values

| Parameter | Value | Why |
|-----------|-------|-----|
| `needs` | `[helm-release]` | Ensures release appears only after Helm index is live |
| `if` | `startsWith(github.ref, 'refs/tags/v')` | Tag-only guard — no accidental branch releases |
| artifact `name` | `helm-chart` | Must match exact string used in `upload-artifact` step in helm-release |
| `artifact path` | `/tmp/charts` | Download destination, referenced in `files:` path |
| `files` | `/tmp/charts/pulltrace-{version}.tgz` | Attaches packaged chart as release asset |
| `make_latest` | `"true"` | Marks this as the latest release on the Releases page |
| `body` | inline | Avoids verbatim CHANGELOG dump with [Unreleased] section |

## Must-Haves Verification

| Requirement | Status | Evidence |
|-------------|--------|----------|
| Pushing v0.1.0 tag creates GitHub Release with title + install commands + CHANGELOG link | Ready | softprops/action-gh-release@v2 with inline body at lines 219-246 |
| GitHub Release includes Helm chart .tgz as downloadable asset | Ready | `files: /tmp/charts/pulltrace-{version}.tgz` |
| GitHub Release does not appear before Helm index is live | Ready | `needs: [helm-release]` at line 203 |
| Release job only fires on semver tags | Ready | `if: startsWith(github.ref, 'refs/tags/v')` at line 205 |

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | 464c8c1 | feat(03-02): add github-release job to ci.yml |

## Deviations from Plan

None — plan executed exactly as written.

## Self-Check: PASSED

- `.github/workflows/ci.yml` — `github-release` job at line 201, `needs: [helm-release]` at line 203, `startsWith` guard at line 205, `download-artifact@v4` at line 214, `softprops/action-gh-release@v2` at line 220, `make_latest: "true"` at line 246
- ci.yml is valid YAML (verified with pyyaml 6.0.3)
- Commit 464c8c1 present on main branch
