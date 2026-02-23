# Stack Research

**Domain:** Open source Kubernetes tool release infrastructure (GitHub Pages docs + Helm chart repo + GitHub Releases)
**Researched:** 2026-02-23
**Confidence:** HIGH for core tooling, MEDIUM for Zensical transition path

## Context

Pulltrace already ships Go binaries, Docker images (GHCR), and a Helm chart (GHCR OCI). This research covers the
release *wrapper*: a GitHub Pages documentation site, a traditional `helm repo add`-compatible repository hosted on
GitHub Pages, CHANGELOG automation, and GitHub Releases triggered by git tags.

The project runs on GitHub free tier only. No external services (Netlify, Vercel, ReadTheDocs). GitHub Actions is
the existing CI platform. Stack choices must integrate cleanly with the existing `ci.yml`.

---

## Recommended Stack

### Core Technologies

| Technology | Version | Purpose | Why Recommended |
|------------|---------|---------|-----------------|
| Material for MkDocs | 9.7.2 (latest stable) | Docs site static site generator | Dominant choice for Kubernetes/Go tooling docs; Markdown-native; GitHub Pages CI workflow is three lines; professional output with zero design work; used by Kubernetes project itself |
| helm/chart-releaser-action | v1.7.0 (Jan 2025) | Publish Helm index.yaml to GitHub Pages + create GitHub Releases for chart tarballs | Official Helm project action; automates `index.yaml` generation + GitHub Release attachment in one step |
| softprops/action-gh-release | v2.5.0 (Dec 2024) | Create GitHub Releases with release notes from CHANGELOG.md | Most widely used release action; supports body-from-file (reads CHANGELOG section), asset uploads, mark-as-latest; v2 is stable |
| Python | 3.12+ (use `3.x` in CI) | Runtime for MkDocs | MkDocs is Python; let CI pin to latest 3.x automatically |

### Supporting Libraries

| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| mkdocs (core) | bundled with mkdocs-material | MkDocs engine | Installed as a dependency of mkdocs-material; do not install separately |
| pymdown-extensions | latest (pinned by mkdocs-material) | Admonitions, code tabs, tasklists in Markdown | Always — Material theme requires it for note/warning callout blocks |
| mkdocs-minify-plugin | >=0.8 | Minify HTML/JS/CSS output | Use if docs are served from GitHub Pages CDN (free, reduces page size) |
| keep-a-changelog format | n/a (format not tool) | CHANGELOG.md structure | Standard format: `## [Unreleased]`, `## [0.1.0] - 2026-xx-xx`; no tooling needed to start |

### Development Tools

| Tool | Purpose | Notes |
|------|---------|-------|
| `mkdocs serve` | Local preview with hot reload | Run in `docs/` context; runs at `localhost:8000`; required for doc authoring |
| helm/chart-releaser CLI | Generate `index.yaml` locally for verification | Not required in normal flow — the action handles it; useful for debugging |
| `pip install mkdocs-material` | Install docs tooling locally | Add `docs/requirements.txt` with pinned version for reproducibility |

---

## Installation

```bash
# Create docs/requirements.txt (pin for reproducibility)
echo "mkdocs-material==9.7.2
mkdocs-minify-plugin>=0.8.0" > docs/requirements.txt

# Install locally for development
pip install -r docs/requirements.txt

# Start local preview
mkdocs serve
```

No npm or Node.js dependencies. MkDocs is pure Python.

---

## Alternatives Considered

| Recommended | Alternative | When to Use Alternative |
|-------------|-------------|-------------------------|
| Material for MkDocs | Docusaurus (React/MDX) | If you need interactive components, versioned docs, or the team is frontend-heavy and wants to extend the site with custom React. Overhead: requires Node.js toolchain in CI, more config. Not worth it for a tool like Pulltrace. |
| Material for MkDocs | Hugo | If build speed matters at scale (thousands of pages) or you need multilingual support. Hugo is what kubernetes.io uses. For a 10-20 page docs site, Hugo's template language is more complex than the benefit warrants. |
| Material for MkDocs | Jekyll | Do not use for new projects (see below). GitHub Pages has native Jekyll support but the ecosystem is stagnant (Ruby, slow builds, no active leadership). |
| Material for MkDocs | Zensical | Zensical is the future successor to Material for MkDocs by the same author, but is pre-release (v0.0.23 as of Feb 2026). Not stable for production. Migrate in 12-18 months once Zensical hits v1.0. |
| helm/chart-releaser-action | Manual `helm repo index` in CI | If you need precise control over where index.yaml lives in gh-pages (chart-releaser-action does not support `--pages-index-path`, per upstream issue #183). See gh-pages coexistence section below. |
| softprops/action-gh-release | gh CLI (`gh release create`) | If you want fewer third-party actions and are comfortable scripting release body extraction from CHANGELOG.md manually. Either works; softprops is simpler for attaching assets. |

---

## What NOT to Use

| Avoid | Why | Use Instead |
|-------|-----|-------------|
| Jekyll | No active maintainer or leadership; Ruby dependency; slow builds; GitHub's native Jekyll support is a historical artifact, not an endorsement for new projects | Material for MkDocs |
| Zensical | Pre-release (v0.0.23, no stable v1.0 as of Feb 2026); missing feature parity with Material for MkDocs; not ready for production docs site | Material for MkDocs 9.7.x (remains under critical maintenance for 12+ months) |
| `mkdocs gh-deploy --force` as the sole deploy step in CI | **This command wipes the entire gh-pages branch** on every run. If helm/chart-releaser-action writes `index.yaml` to gh-pages, the next MkDocs deploy will delete it. This is the single biggest pitfall for this setup. | Use the `peaceiris/actions-gh-pages` action with `keep_files: true`, OR deploy MkDocs to a docs subdirectory, OR use separate deployment strategies (see gh-pages coexistence below) |
| Storing chart .tgz files directly in gh-pages | Bloats the git history forever; GitHub Pages serves them fine but the repo becomes slow to clone | Use chart-releaser's default: store .tgz as GitHub Release assets, only store `index.yaml` in gh-pages |
| Automated CHANGELOG tools (git-cliff, release-drafter) at v0.1.0 | Pulltrace has no conventional commit history yet; automated tools produce empty or malformed output from mixed commit messages | Hand-write CHANGELOG.md for v0.1.0; add git-cliff config later once commit discipline is established |

---

## gh-pages Branch: Coexistence Strategy

**The problem:** helm/chart-releaser-action writes `index.yaml` to the gh-pages branch. MkDocs `gh-deploy --force`
**destroys** the entire gh-pages branch on each docs deploy, wiping `index.yaml`. This is confirmed by upstream
MkDocs issue #2796 and chart-releaser-action issue #183 (PR open but not merged as of Feb 2026).

**Recommended solution: Two-step custom deploy workflow using `peaceiris/actions-gh-pages`**

Structure the gh-pages branch as:
```
gh-pages branch:
├── index.yaml          ← Helm chart repo index (written by chart-releaser manually)
├── index.html          ← MkDocs built site root
├── assets/             ← MkDocs static assets
├── installation/       ← MkDocs docs pages
└── ...
```

Workflow strategy:
1. **Docs deploy workflow** (triggered on push to main with docs changes): Build MkDocs to `site/` directory,
   then deploy using `peaceiris/actions-gh-pages` with `keep_files: true`. This action supports preserving
   existing files in the gh-pages branch, unlike `mkdocs gh-deploy --force`.
2. **Helm release workflow** (triggered on semver tag): Run chart-releaser-action which adds/updates `index.yaml`
   in gh-pages via its own git operations. This does not touch the docs files.

**Alternative if `keep_files: true` proves complex:** Deploy MkDocs site to a `docs/` subdirectory of gh-pages
(set `site_url: https://d44b.github.io/pulltrace/docs/` in mkdocs.yml, deploy to that path). The `index.yaml`
lives at the root of gh-pages. This creates a cleaner separation but requires all internal MkDocs links to
account for the `/docs/` prefix.

**Do not use:** Running chart-releaser-action and `mkdocs gh-deploy` in the same workflow. They will race and
overwrite each other.

---

## Stack Patterns by Variant

**If you want `helm repo add` to work immediately at v0.1.0:**
- Use helm/chart-releaser-action with the `packages_with_index: true` option (stores .tgz in gh-pages
  alongside index.yaml — avoids GitHub Release asset lookups)
- Keep the OCI push in existing CI as-is; chart-releaser adds the traditional repo on top

**If you want to avoid gh-pages branch complexity entirely:**
- Skip the traditional Helm repo; document OCI-only install clearly in the docs site
- `helm install pulltrace oci://ghcr.io/d44b/charts/pulltrace --version 0.1.0` is already wired
- This is a valid choice for v0.1.0 and eliminates the coexistence problem entirely

**If docs site needs versioning later:**
- Add `mike` (MkDocs version switcher) as a Python dependency; it deploys versioned docs to gh-pages without
  destroying other files by using versioned subdirectories
- Do not add mike at v0.1.0 — single-version docs are simpler and sufficient

---

## Version Compatibility

| Package | Compatible With | Notes |
|---------|-----------------|-------|
| mkdocs-material==9.7.2 | Python 3.8-3.12+ | 9.7.x is final Material for MkDocs series; stays compatible |
| helm/chart-releaser-action@v1.7.0 | GitHub Actions ubuntu-latest | Uses chart-releaser CLI v1.7.0 internally; no breaking changes from v1.6 |
| softprops/action-gh-release@v2 | GitHub Actions ubuntu-latest | v2 uses Node.js 20; compatible with actions/checkout@v4 already in ci.yml |
| azure/setup-helm@v4 | GitHub Actions ubuntu-latest | Already in ci.yml for helm-lint; chart-releaser-action brings its own helm binary |

---

## GitHub Actions Workflow Skeleton

**Docs deploy** (`.github/workflows/docs.yml`):
```yaml
on:
  push:
    branches: [main]
    paths: ['docs/**', 'mkdocs.yml']

jobs:
  deploy:
    runs-on: ubuntu-latest
    permissions:
      contents: write
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: '3.x'
          cache: pip
          cache-dependency-path: docs/requirements.txt
      - run: pip install -r docs/requirements.txt
      - uses: peaceiris/actions-gh-pages@v4
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: ./site
          keep_files: true   # preserve index.yaml written by chart-releaser
        env:
          BUILD_CMD: mkdocs build --strict
```

**GitHub Release** (add to existing `ci.yml` helm-release job, after helm push):
```yaml
      - uses: softprops/action-gh-release@v2
        with:
          body_path: CHANGELOG.md  # or extract section with script
          files: |
            /tmp/charts/pulltrace-*.tgz
```

**Helm repo (traditional, adds to existing helm-release job):**
```yaml
      - uses: helm/chart-releaser-action@v1.7.0
        with:
          charts_dir: charts
        env:
          CR_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

---

## Sources

- MkDocs Material PyPI — version 9.7.2 confirmed as latest stable (Feb 18, 2026 release)
  https://pypi.org/project/mkdocs-material/
- Material for MkDocs maintenance mode announcement (Nov 2025)
  https://squidfunk.github.io/mkdocs-material/blog/2025/11/05/zensical/
- Zensical releases page — v0.0.23 latest, pre-release status confirmed
  https://github.com/zensical/zensical/releases
- helm/chart-releaser-action releases — v1.7.0 released Jan 20, 2025
  https://github.com/helm/chart-releaser-action/releases
- Helm official chart-releaser-action documentation
  https://helm.sh/docs/howto/chart_releaser_action/
- chart-releaser-action issue #183 — pages-index-path not supported (PR open)
  https://github.com/helm/chart-releaser-action/issues/183
- softprops/action-gh-release releases — v2.5.0 latest (Dec 1, 2024)
  https://github.com/softprops/action-gh-release/releases
- Material for MkDocs — Publishing your site (GitHub Actions workflow reference)
  https://squidfunk.github.io/mkdocs-material/publishing-your-site/
- MkDocs issue #2796 — gh-deploy overwrites all files, no preserve option
  https://github.com/mkdocs/mkdocs/issues/2796
- Helm chart repository guide (index.yaml structure)
  https://helm.sh/docs/topics/chart_repository/
- Static site generator comparison (Feb 2025)
  https://justwriteclick.com/2025/02/06/a-flight-of-static-site-generators-sampling-the-best-for-documentation/

---
*Stack research for: Pulltrace v0.1.0 release infrastructure*
*Researched: 2026-02-23*
