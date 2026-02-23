# Roadmap: Pulltrace v0.1.0 Open Source Release

## Overview

Pulltrace has a working product core. This roadmap wraps that core in the
artifacts a stranger needs to trust and install a privileged DaemonSet on their
cluster: community files, a documentation site, a `helm repo add`-compatible
Helm repository, and a GitHub Release triggered by a version tag. Phases are
ordered so each one unblocks the next — community files before release body,
docs before Helm repo (to establish the gh-pages branch structure safely),
automation before tagging, and tagging as the final verification exercise.

## Phases

**Phase Numbering:**
- Integer phases (1, 2, 3): Planned milestone work
- Decimal phases (2.1, 2.2): Urgent insertions (marked with INSERTED)

Decimal phases appear between their surrounding integers in numeric order.

- [ ] **Phase 1: Foundation Files** - Community files, bug fixes, and repo metadata needed before anything else
- [ ] **Phase 2: Documentation Site** - MkDocs Material site on GitHub Pages covering installation, configuration, and architecture
- [ ] **Phase 3: Release Automation** - Classic Helm repo index on gh-pages and GitHub Release CI job wired and tested
- [ ] **Phase 4: Launch** - GHCR packages made public, v0.1.0 tag pushed, all artifacts verified live

## Phase Details

### Phase 1: Foundation Files
**Goal**: The repository signals credibility to first-time visitors and CHANGELOG.md exists as the hard prerequisite for the release body
**Depends on**: Nothing (first phase)
**Requirements**: COMM-01, COMM-02, FIX-01, FIX-02, META-01
**Success Criteria** (what must be TRUE):
  1. A contributor can read CONTRIBUTING.md and run a successful build using the documented Go 1.22 Docker workaround
  2. CHANGELOG.md has a `[0.1.0]` entry in keep-a-changelog format that describes what this release includes
  3. `pulltrace_pull_errors_total` counter increments when a pull completes with a non-empty Error field (verifiable via Prometheus /metrics)
  4. LayerDetail component displays non-zero bytesPerSec and a mediaType string for an active layer pull
  5. CODE_OF_CONDUCT.md does NOT exist (intentionally omitted — do not create it)
**Plans**: 3 plans

Plans:
- [ ] 01-01-PLAN.md — Author CONTRIBUTING.md (Docker workaround) and CHANGELOG.md (do NOT create CODE_OF_CONDUCT.md)
- [ ] 01-02-PLAN.md — Fix PullErrors metric increment (FIX-01) and layer bytesPerSec/mediaType population (FIX-02)
- [ ] 01-03-PLAN.md — Set GitHub repo topics via gh CLI and upload social preview image (META-01)

### Phase 2: Documentation Site
**Goal**: A stranger can navigate to `https://d44b.github.io/pulltrace/` and find enough information to install, configure, and understand Pulltrace without reading the source code
**Depends on**: Phase 1
**Requirements**: DOCS-01, DOCS-02, DOCS-03, DOCS-04, DOCS-05
**Success Criteria** (what must be TRUE):
  1. Browsing `https://d44b.github.io/pulltrace/` returns a rendered MkDocs Material site (not a 404)
  2. The Installation page shows a working `helm repo add` command and the prerequisites needed before installing
  3. The Configuration page lists every environment variable for both the server and agent with type and default value
  4. The Architecture page has a diagram showing how agent, server, and UI connect
  5. Pushing a commit to `main` automatically deploys updated docs to GitHub Pages within the CI run (no manual step)
**Plans**: 2 plans

Plans:
- [ ] 02-01-PLAN.md — Scaffold mkdocs.yml, docs/ stub pages (7 files), and docs.yml CI workflow (peaceiris/actions-gh-pages@v4, keep_files: true)
- [ ] 02-02-PLAN.md — Write full docs content (installation, configuration, architecture, prometheus, known-limitations, contributing, index pages)

### Phase 3: Release Automation
**Goal**: Pushing a semver tag causes CI to publish the Helm chart to both the classic `helm repo add` path and GHCR OCI, then create a GitHub Release with a populated body — all without manual intervention
**Depends on**: Phase 2
**Requirements**: HELM-01, HELM-02, HELM-03, HELM-04, REL-01, REL-02
**Success Criteria** (what must be TRUE):
  1. `helm repo add pulltrace https://d44b.github.io/pulltrace/charts` succeeds and returns a success message
  2. `helm install pulltrace pulltrace/pulltrace` installs from the classic Helm repo (not OCI-only)
  3. `https://d44b.github.io/pulltrace/charts/index.yaml` is publicly reachable and contains a valid chart entry
  4. A semver tag push triggers CI to produce a GitHub Release with a title, install commands, and a link to the CHANGELOG entry
  5. The docs site at the gh-pages root is intact after the `helm-pages` job completes (no mutual destruction)
**Plans**: 2 plans

Plans:
- [ ] 03-01-PLAN.md — Fix ci.yml permissions (contents: write), add helm-pages steps to helm-release job, add shared deploy-gh-pages concurrency group to ci.yml and docs.yml
- [ ] 03-02-PLAN.md — Add github-release job to ci.yml using softprops/action-gh-release@v2 with artifact sharing from helm-release

### Phase 4: Launch
**Goal**: All v0.1.0 artifacts are public and reachable by an unauthenticated user; the project is in an announcement-ready state
**Depends on**: Phase 3
**Requirements**: REL-03, REL-04
**Success Criteria** (what must be TRUE):
  1. `docker pull ghcr.io/d44b/pulltrace-agent:0.1.0` succeeds from an unauthenticated shell
  2. The GitHub Release page for v0.1.0 exists with a release body, attached Helm chart `.tgz`, and correct Docker image tags
  3. `helm install pulltrace pulltrace/pulltrace --version 0.1.0` resolves the chart from the classic Helm repo
**Plans**: TBD

Plans:
- [ ] 04-01: Make all three GHCR packages public (pulltrace-agent, pulltrace-server, charts/pulltrace)
- [ ] 04-02: Push git tag v0.1.0, run pre-flight checklist, verify all artifacts live

## Progress

**Execution Order:**
Phases execute in numeric order: 1 → 2 → 3 → 4

| Phase | Plans Complete | Status | Completed |
|-------|----------------|--------|-----------|
| 1. Foundation Files | 0/3 | Not started | - |
| 2. Documentation Site | 0/2 | Not started | - |
| 3. Release Automation | 1/2 | In Progress|  |
| 4. Launch | 0/2 | Not started | - |
