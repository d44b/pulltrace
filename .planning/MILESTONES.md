# Milestones

## v0.1 Open Source Release (Shipped: 2026-02-23)

**Phases completed:** 4 phases, 9 plans
**Files changed:** 29 files, ~3,100 lines added
**Timeline:** 2026-02-23 (single session, ~44min execution)

**Key accomplishments:**
1. CONTRIBUTING.md with Go 1.22 Docker workaround + CHANGELOG.md in keep-a-changelog format with [0.1.0] entry
2. Fixed `pulltrace_pull_errors_total` Prometheus counter and layer `bytesPerSec`/`mediaType` population in server
3. GitHub repository topics (5) and description set for discoverability
4. MkDocs Material documentation site live at `https://d44b.github.io/pulltrace/` with installation, configuration, architecture, and Prometheus reference pages
5. Classic Helm repository via `helm repo add pulltrace https://d44b.github.io/pulltrace/charts` — co-deployed with docs on gh-pages using shared concurrency group
6. Automated GitHub Release creation on semver tag push via `softprops/action-gh-release@v2` with install commands and CHANGELOG link
7. All three GHCR packages (pulltrace-agent, pulltrace-server, charts/pulltrace) made public
8. v0.1.0 tag pushed; CI all 7 jobs green; GitHub Release, Docker images, Helm classic + OCI repos all publicly reachable

**Tech debt carried forward:**
- CONTRIBUTING.md dead link to removed CODE_OF_CONDUCT.md (line 147)
- Social preview image not uploaded (no GitHub API; requires browser Settings)
- No Go unit or integration tests (pre-existing)
- ci.yml OCI release body URL hardcodes `d44b` owner (breaks forks)

**Archive:** `.planning/milestones/v0.1-ROADMAP.md` | `.planning/milestones/v0.1-REQUIREMENTS.md`

---

