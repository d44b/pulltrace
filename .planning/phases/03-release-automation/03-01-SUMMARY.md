---
phase: 03-release-automation
plan: "01"
subsystem: ci-cd
tags: [github-actions, helm, gh-pages, permissions, concurrency]
dependency_graph:
  requires: []
  provides: [helm-classic-repo-pipeline, gh-pages-permissions, deploy-concurrency]
  affects: [.github/workflows/ci.yml, .github/workflows/docs.yml]
tech_stack:
  added: [peaceiris/actions-gh-pages@v4, actions/upload-artifact@v4]
  patterns: [helm-repo-index, gh-pages-subdirectory-deploy, shared-concurrency-group]
key_files:
  created: []
  modified:
    - .github/workflows/ci.yml
    - .github/workflows/docs.yml
decisions:
  - "contents:write at workflow level (not job level) — job-level cannot escalate beyond workflow level"
  - "No --merge flag for v0.1.0 helm repo index (no prior index.yaml exists)"
  - "destination_dir:charts + keep_files:true preserves docs site at gh-pages root"
  - "cancel-in-progress:false on deploy-gh-pages concurrency — never abort an in-flight gh-pages push"
  - "Upload chart artifact for github-release job reuse, avoiding re-packaging"
metrics:
  duration: 1min
  completed_date: "2026-02-23"
  tasks_completed: 2
  files_modified: 2
---

# Phase 3 Plan 1: CI Permissions, Helm Pages Pipeline, and Concurrency Summary

Classic Helm repo pipeline via `helm repo index --url` + `peaceiris/actions-gh-pages@v4` subdirectory deploy, with `contents:write` permission fix and shared `deploy-gh-pages` concurrency group preventing race conditions.

## What Was Done

### Task 1: Fix ci.yml permissions and add deploy-gh-pages concurrency group

**Changes to `.github/workflows/ci.yml`:**
- Changed workflow-level `permissions.contents` from `read` to `write`
  - Required by both `softprops/action-gh-release` and `peaceiris/actions-gh-pages` push
  - Must be at workflow level — job-level permissions cannot escalate beyond workflow level

**Changes to `.github/workflows/docs.yml`:**
- Added `concurrency` block to the `deploy` job:
  ```yaml
  concurrency:
    group: deploy-gh-pages
    cancel-in-progress: false   # never abort an in-flight gh-pages deployment
  ```
- This serializes docs deploys with the helm-release job's gh-pages push

### Task 2: Add helm-pages steps to helm-release job

**Changes to `.github/workflows/ci.yml` — `helm-release` job:**

Added job-level `concurrency` block (serializes with docs.yml pushes):
```yaml
concurrency:
  group: deploy-gh-pages
  cancel-in-progress: false
```

Appended four new steps after the existing "Push chart to GHCR OCI" step:

1. **Upload chart artifact** — shares packaged `.tgz` with the future `github-release` job via `actions/upload-artifact@v4`, avoiding re-packaging with a separate sed+helm-package run.

2. **Stage Helm repo files** — copies packaged `.tgz` into `/tmp/helm-pages/` staging directory.

3. **Generate Helm repo index** — runs:
   ```
   helm repo index /tmp/helm-pages --url https://d44b.github.io/pulltrace/charts
   ```
   No `--merge` flag for v0.1.0 (no prior `index.yaml` exists). When publishing v0.2.0+, add sparse-checkout + `--merge` to preserve prior chart versions.

4. **Deploy charts to gh-pages** — deploys `/tmp/helm-pages` (contains `.tgz` + `index.yaml`) to the `charts/` subdirectory of gh-pages:
   ```yaml
   uses: peaceiris/actions-gh-pages@v4
   with:
     destination_dir: charts
     keep_files: true
   ```
   The `destination_dir: charts` + `keep_files: true` combination deploys ONLY to `/charts/` on gh-pages, leaving the MkDocs docs site at the root intact.

## Key Configuration Values

| Parameter | Value | Why |
|-----------|-------|-----|
| `--url` for `helm repo index` | `https://d44b.github.io/pulltrace/charts` | Must exactly match the `helm repo add` URL — baked into `index.yaml` download URLs |
| `destination_dir` | `charts` | Deploys only to `/charts/` subdirectory of gh-pages |
| `keep_files` | `true` | Preserves the docs site at gh-pages root |
| `cancel-in-progress` | `false` | Never abort an in-flight gh-pages push (would leave branch in broken state) |
| Artifact `retention-days` | `1` | Short-lived; only needed by `github-release` job in same run |

## Must-Haves Verification

| Requirement | Status | Evidence |
|-------------|--------|----------|
| `helm repo add pulltrace https://d44b.github.io/pulltrace/charts` succeeds after semver tag | Ready | `helm repo index --url https://d44b.github.io/pulltrace/charts` bakes correct URL into `index.yaml` |
| `https://d44b.github.io/pulltrace/charts/index.yaml` publicly reachable | Ready | peaceiris deploys to `destination_dir: charts` on gh-pages |
| Docs site at gh-pages root intact after helm-pages job | Ready | `keep_files: true` preserves root |
| CI does not fail with 403 on gh-pages push or Release creation | Ready | `contents: write` at workflow level |
| Concurrent gh-pages pushes serialized (not non-fast-forward rejected) | Ready | Both ci.yml + docs.yml share `group: deploy-gh-pages` with `cancel-in-progress: false` |

## Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | c4f69b4 | fix(03-01): set contents:write permission and add shared deploy-gh-pages concurrency |
| 2 | 165c773 | feat(03-01): extend helm-release job with helm-pages steps and concurrency group |

## Deviations from Plan

None — plan executed exactly as written.

## Self-Check: PASSED

- `.github/workflows/ci.yml` — exists, `contents: write` at line 11, `deploy-gh-pages` concurrency at line 145, all 4 new steps at lines 173-198
- `.github/workflows/docs.yml` — exists, `deploy-gh-pages` concurrency at line 14
- Commits c4f69b4 and 165c773 present on main branch
