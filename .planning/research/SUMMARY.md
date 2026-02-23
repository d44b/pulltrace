# Project Research Summary

**Project:** Pulltrace v0.1.0 Release Infrastructure
**Domain:** Open source Kubernetes observability tool — release wrapper (docs site, Helm repo, GitHub Releases, community files)
**Researched:** 2026-02-23
**Confidence:** HIGH

## Executive Summary

Pulltrace has a working product core (agent DaemonSet, server, React UI, Helm chart, CI pipeline). The gap is the "release wrapper" — the files, workflows, and hosted pages that make a stranger trust the project enough to install a privileged DaemonSet on their cluster. Research against cert-manager, KEDA, external-secrets, metrics-server, and flux2 shows the bar clearly: a credible v0.1.0 Kubernetes tool requires community files (CONTRIBUTING.md, CHANGELOG.md), a functional `helm repo add` install path, a GitHub Release with a useful body, and a documentation site that goes beyond the README. None of these are hard individually; the complexity is in doing them in the right order without stepping on each other. Note: CODE_OF_CONDUCT.md is intentionally omitted from this project — do not create it.

The recommended approach is Material for MkDocs 9.7.x on GitHub Pages (Python, no Node.js, three-line CI workflow, dominant standard in the Kubernetes ecosystem) for the docs site, combined with a manual `helm repo index` job that writes to a `charts/` subdirectory of the same `gh-pages` branch. The two deployments must use `peaceiris/actions-gh-pages` with `keep_files: true` — the single most important architectural constraint — so that docs deploys do not wipe the Helm index and vice versa. The existing CI's OCI push to GHCR is kept as-is; the classic `helm repo add` path is additive on top.

The primary risks are operational, not technical: GHCR packages default to private (CI push success does not mean public availability), the current `ci.yml` has `contents: read` which silently blocks release creation, and running `mkdocs gh-deploy --force` alongside any Helm repo tooling destroys the Helm index on every docs push. All three are verified against official documentation and have simple, specific fixes. If these are addressed in the right order — permissions and visibility before tagging, docs and Helm repo before announcing — the release goes smoothly.

---

## Key Findings

### Recommended Stack

The docs and release stack is intentionally minimal. Material for MkDocs 9.7.2 is the dominant choice for Kubernetes/Go tooling documentation (used by the Kubernetes project itself, standard across the ecosystem); it requires only a `docs/requirements.txt` and a `mkdocs.yml` at the repo root, and its GitHub Pages CI workflow is three lines. MkDocs is now in maintenance mode with Zensical as the successor, but Zensical is pre-release (v0.0.23 as of Feb 2026) and should not be used for production.

For the Helm repo, `helm/chart-releaser-action` is the wrong tool for this project: it creates GitHub Releases automatically (colliding with the manually-written release notes release) and places `index.yaml` at the gh-pages root (conflicting with the docs site at the same root). The correct approach is to extend the existing `helm-release` CI job with a `helm-pages` step that runs `helm repo index --merge` and pushes to a `charts/` subdir of `gh-pages`. For GitHub Release creation, `softprops/action-gh-release@v2` is the current maintained action (`actions/create-release` is deprecated).

**Core technologies:**
- **Material for MkDocs 9.7.2**: Docs site static site generator — dominant Kubernetes ecosystem standard, Python-only, GitHub Pages CI in 3 lines
- **peaceiris/actions-gh-pages@v4 with `keep_files: true`**: gh-pages deployment — the only way to coexist docs and Helm index on the same branch without data loss
- **softprops/action-gh-release@v2**: GitHub Release creation — current maintained action, supports body-from-file and asset attachment
- **Manual `helm repo index --merge`**: Classic Helm repo generation — extends existing CI without duplicating chart packaging or colliding with release notes

**What NOT to use:**
- `mkdocs gh-deploy --force` — wipes the entire gh-pages branch, destroys Helm index
- `helm/chart-releaser-action` — creates duplicate GitHub Releases, places index.yaml at gh-pages root
- Zensical — pre-release, not production-ready until ~v1.0 (12-18 months)
- Jekyll — stagnant ecosystem, Ruby dependency, no active leadership

### Expected Features

Based on analysis of cert-manager, KEDA, external-secrets, metrics-server, and flux2, the feature bar for a credible v0.1.0 Kubernetes tool is well-defined.

**Must have (table stakes) — v0.1.0:**
- **CONTRIBUTING.md** — prerequisites, Docker build workaround for Go 1.22 (critical for this project), `make` commands, PR guidelines
- **CHANGELOG.md** — keep-a-changelog format; must exist before tagging so release body has content to reference
- ~~**CODE_OF_CONDUCT.md**~~ — intentionally omitted; **do NOT create CODE_OF_CONDUCT.md**
- **Helm repo via `helm repo add`** — the README already promises this URL; it must work; OCI-only is a user confusion landmine
- **GitHub Release v0.1.0** — release body with install commands, compatibility matrix, known limitations, links; CI must create it, not just the auto-generated source zip
- **README badges** — CI status, Apache 2.0 license, Kubernetes 1.28+ compatibility; Shields.io URLs
- **GitHub repo metadata** — description + topics (`kubernetes`, `image-pull`, `containerd`, `observability`, `helm`)
- **Documentation site (GitHub Pages, Material for MkDocs)** — minimum 7 pages: index, getting-started, configuration, architecture, prometheus, known-limitations, contributing

**Should have (differentiators) — add post-launch:**
- **Artifact Hub listing** — `artifacthub-repo.yml` in gh-pages root; requires Helm repo to be live first; adds `helm search hub` discoverability
- **GitHub social preview image** — 1280x640px; displayed on GitHub and link previews
- **GitHub Discussions** — enable when Issues fill with questions rather than bugs
- **FAQ page in docs** — after first batch of Issues reveals recurring questions

**Defer (v1.0+):**
- SBOM + cosign image signing — appropriate when enterprise adoption is established
- Versioned docs (Mike) — only when breaking API changes require parallel version support
- GoReleaser integration — relevant only if a standalone `pulltrace` CLI binary is added
- ADOPTERS.md — after first organization confirms production use

### Architecture Approach

The gh-pages branch serves dual purposes: MkDocs compiled output at the root (browser docs), and a classic Helm repository under `charts/` (machine-readable index). Two independent GitHub Actions workflows write to this branch — `docs.yml` triggers on push to `main`, and a `helm-pages` job in `ci.yml` triggers only on semver tags. Both use `peaceiris/actions-gh-pages` with `keep_files: true` and different `destination_dir` values, preventing race conditions and mutual destruction. The OCI push to GHCR (already working) is unchanged; the classic Helm repo is additive.

**Major components:**
1. **`docs/` source directory + `mkdocs.yml`** — MkDocs markdown source files; `docs.yml` workflow builds and deploys to gh-pages root on every main push
2. **`gh-pages` branch `charts/` subdir** — Helm `index.yaml` + `.tgz` files; updated by `helm-pages` CI job on semver tags only via `helm repo index --merge`
3. **GitHub Releases** — created by a new `github-release` job in `ci.yml` using `softprops/action-gh-release@v2`, gated on `needs: [docker, helm-release]`; release body populated from CHANGELOG.md section
4. **GHCR OCI registry** — existing; `pulltrace-agent`, `pulltrace-server`, and `charts/pulltrace` packages must each be made public manually after first push
5. **Community files** — `CONTRIBUTING.md`, `CHANGELOG.md` at repo root; written by hand, no tooling required (CODE_OF_CONDUCT.md intentionally omitted — do NOT create it)

**Key architectural constraint:** Never run `mkdocs gh-deploy --force`. Always use `peaceiris/actions-gh-pages` with `keep_files: true`. `helm-pages` job must use `needs: [helm-release]`. `github-release` job must use `needs: [docker, helm-release]`.

### Critical Pitfalls

1. **GHCR packages private by default** — CI push success does not mean public availability. After first tag push, manually make all three packages public: `pulltrace-agent`, `pulltrace-server`, `charts/pulltrace`. Verify with `docker pull ghcr.io/d44b/pulltrace-agent:0.1.0` from an unauthenticated shell before announcing.

2. **`ci.yml` has `contents: read` — blocks release creation silently** — The existing CI permissions block will cause any `softprops/action-gh-release` job to fail with no useful error. Change to `contents: write` before pushing the v0.1.0 tag. Fix this before anything else in the release phase.

3. **`mkdocs gh-deploy --force` destroys Helm index** — This is the default MkDocs deploy command and it wipes the entire gh-pages branch. Never use it. Use `peaceiris/actions-gh-pages` with `keep_files: true` exclusively.

4. **OCI-only install confuses users** — `helm repo add` does not support OCI URLs. The README already shows a `helm repo add` command that will fail until the classic index is live. Serve both: OCI push (already done) and `charts/index.yaml` on gh-pages (new).

5. **GitHub Pages broken baseurl on project pages** — GitHub project pages are served under `/pulltrace/`, not `/`. MkDocs handles this automatically when `site_url: https://d44b.github.io/pulltrace/` is set in `mkdocs.yml`. Set this before the first deploy; do not rely on relative paths alone.

---

## Implications for Roadmap

Based on research dependencies and pitfall ordering, the work naturally groups into 4 phases. The hard constraint is that CHANGELOG.md must exist before tagging, and GHCR packages must be public before announcing. Everything else is ordered by dependency (Helm repo before Artifact Hub, docs site before release announcement).

### Phase 1: Foundation Files
**Rationale:** Community files are independent of all other work and unblock contributors from the moment the repo goes public. CHANGELOG.md is a hard prerequisite for the GitHub Release body. These are all low-effort, high-signal items that cost < 4 hours combined.
**Delivers:** CONTRIBUTING.md (with Go 1.22 Docker build workaround), CHANGELOG.md (keep-a-changelog format, `[0.1.0]` entry), GitHub repo metadata (description + topics). Do NOT create CODE_OF_CONDUCT.md — it is intentionally omitted.
**Addresses:** Table-stakes features from FEATURES.md
**Avoids:** Pitfall 8 (first contributors hit Go 1.22 wall with no guidance)

### Phase 2: Documentation Site
**Rationale:** The docs site is a dependency for the Helm repo coexistence strategy — both share the `gh-pages` branch. Setting up the `docs.yml` workflow with `peaceiris/actions-gh-pages` and `keep_files: true` establishes the correct gh-pages branch structure before the Helm repo job writes to it. Doing this first also confirms GitHub Pages is enabled in repo settings (a manual step that is easy to forget).
**Delivers:** Material for MkDocs site deployed to GitHub Pages with 7 pages (index, getting-started, configuration, architecture, prometheus, known-limitations, contributing); `docs.yml` GitHub Actions workflow; `mkdocs.yml` with correct `site_url`
**Uses:** Material for MkDocs 9.7.2, peaceiris/actions-gh-pages@v4, `docs/requirements.txt`
**Implements:** Docs site component, gh-pages branch root structure
**Avoids:** Pitfall 4 (broken baseurl — solved by `site_url` in mkdocs.yml), Pitfall 6 (Pages not enabled — must verify in Settings before proceeding to Phase 3)

### Phase 3: Release Automation
**Rationale:** With community files and docs in place, CI can be wired to create the full release package. The `helm-pages` job and `github-release` job both depend on the `helm-release` job (which already exists), so they slot into the existing workflow dependency chain. The `contents: read` → `contents: write` fix must happen in this phase, before any tag is pushed.
**Delivers:** `helm-pages` CI job (writes `charts/index.yaml` to gh-pages on semver tags via `helm repo index --merge`); `github-release` CI job (creates GitHub Release with release body from CHANGELOG, attaches `.tgz` and `checksums.txt`); `contents: write` permission fix in `ci.yml`; `helm repo add pulltrace https://d44b.github.io/pulltrace/charts` working
**Uses:** softprops/action-gh-release@v2, `helm repo index --merge`, peaceiris/actions-gh-pages@v4 (`destination_dir: charts/`)
**Implements:** Helm repo component, GitHub Release component
**Avoids:** Pitfall 2 (OCI-only confusion), Pitfall 3 (release with no artifacts), Pitfall 7 (release before images visible — gated via `needs: [docker, helm-release]`)

### Phase 4: v0.1.0 Tag and Launch
**Rationale:** This phase is verification + execution, not development. The "Looks Done But Isn't" checklist from PITFALLS.md drives the pre-flight. GHCR package visibility is the most likely point of failure for a public launch and must be explicitly verified, not assumed.
**Delivers:** `git tag v0.1.0` pushed; all 3 GHCR packages (`pulltrace-agent`, `pulltrace-server`, `charts/pulltrace`) verified public; GitHub Release page with correct body and attached assets; README badges (CI, license, Kubernetes version); announcement-ready state
**Avoids:** Pitfall 1 (GHCR packages private — verify before announcing), Pitfall 5 (Helm OCI chart private separately from Docker images)

### Phase Ordering Rationale

- **Foundation before everything** — CHANGELOG.md is a hard prerequisite for the release body; no other work can finish without it. Community files are zero-dependency and signal project seriousness from day one.
- **Docs before Helm repo** — The `peaceiris/actions-gh-pages` `keep_files: true` pattern must be established first so that the subsequent `helm-pages` job writes to the right subpath without destroying the docs. Setting up Pages in the wrong order (Helm first, docs second with `--force`) is how the gh-pages branch gets corrupted.
- **Automation before tagging** — The `contents: write` permission fix and the `github-release` job must be in place before `v0.1.0` is tagged. Re-running a failed release requires deleting and re-pushing the tag, which is friction; getting it right the first time is strongly preferable.
- **Launch as a verification phase, not a build phase** — The actual tag push should be a 15-minute checklist exercise, not a debugging session. All code and configuration should be complete and tested before Phase 4 begins.

### Research Flags

Phases with well-documented patterns (skip `/gsd:research-phase`):
- **Phase 1 (Foundation Files)**: Standard open source community file authoring; patterns are obvious and well-documented. No research needed.
- **Phase 4 (Launch)**: Operational checklist; all verification steps are defined in PITFALLS.md. No research needed.

Phases that may benefit from targeted research during planning:
- **Phase 2 (Documentation Site)**: MkDocs configuration details (`mkdocs.yml` nav structure, Material theme feature flags, Python version pinning) are well-documented, but the docs content itself requires decisions about what to write. Recommend reviewing the existing ADR files in `docs/adr/` to surface content that should be promoted to the main docs site. No external research needed, but content planning warrants its own task.
- **Phase 3 (Release Automation)**: The `helm repo index --merge` job and `peaceiris/actions-gh-pages` configuration are the most technically specific parts. ARCHITECTURE.md provides complete YAML examples for both workflows. If the implementation hits unexpected CI behavior, the workflow examples in ARCHITECTURE.md are the reference point.

---

## Confidence Assessment

| Area | Confidence | Notes |
|------|------------|-------|
| Stack | HIGH | Material for MkDocs 9.7.2 release confirmed via PyPI; chart-releaser-action v1.7.0 confirmed via GitHub releases; softprops/action-gh-release v2.5.0 confirmed; Zensical pre-release status verified |
| Features | HIGH | Verified against 5 reference projects (cert-manager, KEDA, external-secrets, metrics-server, flux2); Kubernetes template project requirements checked against official k8s-sigs source |
| Architecture | HIGH | Core mechanics verified against official Helm docs and chart-releaser-action source code; `pages-index-path` limitation confirmed via open issue #183; peaceiris/actions-gh-pages `keep_files` behavior confirmed |
| Pitfalls | HIGH | GHCR private-by-default confirmed via official GitHub Docs; `contents: read` blocker confirmed via workflow inspection; `mkdocs gh-deploy --force` destructive behavior confirmed via MkDocs issue #2796 |

**Overall confidence:** HIGH

### Gaps to Address

- **Chart `.tgz` in gh-pages vs GitHub Release assets**: ARCHITECTURE.md identifies a long-term scaling concern: storing `.tgz` files directly in the `gh-pages` branch bloats git history. For v0.1.0 with a single chart version, this is a non-issue. At 10+ versions, the preferred approach is to have `index.yaml` reference GitHub Release download URLs rather than serving `.tgz` from gh-pages. This is worth noting as a known future migration but does not affect Phase 3 implementation.

- **`helm-pages` job vs `docs-deploy` job concurrent trigger on tag**: When a semver tag is pushed, both `ci.yml` (which includes `helm-pages`) and `docs.yml` (which deploys docs) may trigger. If both run concurrently and both push to `gh-pages`, there is a git push conflict risk even with `keep_files: true`. Mitigation: add `concurrency: group: gh-pages` to both workflows, or trigger `docs.yml` only on push to `main` (not on tags). Research is inconclusive on GitHub Actions' gh-pages branch concurrency handling; this should be tested before the v0.1.0 tag push.

- **Artifact Hub OCI indexing**: FEATURES.md notes that ArtifactHub can index OCI registries directly, but recommends the classic Helm repo path for broader compatibility. The specifics of `artifacthub-repo.yml` placement and OCI vs classic indexing configuration are not fully detailed. This is a post-launch concern (Phase P2) and does not block v0.1.0.

---

## Sources

### Primary (HIGH confidence)
- MkDocs Material PyPI — version 9.7.2 release date and status: https://pypi.org/project/mkdocs-material/
- Material for MkDocs maintenance mode announcement: https://squidfunk.github.io/mkdocs-material/blog/2025/11/05/zensical/
- Helm Chart Repository Guide — subdirectory URL support: https://helm.sh/docs/topics/chart_repository/
- chart-releaser-action action.yml — confirms `pages-index-path` not exposed: https://github.com/helm/chart-releaser-action
- chart-releaser-action issue #183 — subpath unresolved: https://github.com/helm/chart-releaser-action/issues/183
- GitHub Docs: Configuring package visibility (private by default): https://docs.github.com/en/packages/learn-github-packages/configuring-a-packages-access-control-and-visibility
- GitHub Docs: Configuring GitHub Pages publishing source: https://docs.github.com/en/pages/getting-started-with-github-pages/configuring-a-publishing-source-for-your-github-pages-site
- Helm Docs: OCI registries (`helm repo add` incompatibility): https://helm.sh/docs/topics/registries/
- Kubernetes template project required files: https://github.com/kubernetes/kubernetes-template-project
- Keep a Changelog format spec: https://keepachangelog.com/en/1.1.0/
- Artifact Hub Helm chart requirements: https://artifacthub.io/docs/topics/repositories/helm-charts/

### Secondary (MEDIUM confidence)
- cert-manager repository and docs — CONTRIBUTING, release format, docs site structure: https://github.com/cert-manager/cert-manager
- KEDA repository — BUILD.md, CHANGELOG.md, release body format: https://github.com/kedacore/keda
- external-secrets repository — ADOPTERS.md pattern, docs site: https://github.com/external-secrets/external-secrets
- metrics-server repository — minimal CONTRIBUTING.md, KNOWN_ISSUES.md pattern: https://github.com/kubernetes-sigs/metrics-server
- flux2 release structure — asset patterns: https://github.com/fluxcd/flux2/releases/tag/v2.2.0
- softprops/action-gh-release — current maintained release action: https://github.com/softprops/action-gh-release
- peaceiris/actions-gh-pages — `destination_dir` + `keep_files` options: https://github.com/peaceiris/actions-gh-pages
- MkDocs issue #2796 — `gh-deploy --force` destroys branch content: https://github.com/mkdocs/mkdocs/issues/2796

---
*Research completed: 2026-02-23*
*Ready for roadmap: yes*
