# Architecture Research

**Domain:** GitHub Pages docs + Helm chart repository hosting for a Kubernetes open-source tool
**Researched:** 2026-02-23
**Confidence:** HIGH (core mechanics verified against official Helm docs and chart-releaser-action source)

---

## Standard Architecture

### System Overview

```
┌────────────────────────────────────────────────────────────────────┐
│                         Source Repository (main branch)            │
│                                                                    │
│  ┌──────────────┐  ┌──────────────┐  ┌─────────────────────────┐  │
│  │ charts/      │  │ docs/        │  │ .github/workflows/       │  │
│  │ pulltrace/   │  │ (mkdocs src) │  │ ci.yml                  │  │
│  │   Chart.yaml │  │ *.md files   │  │ docs.yml (new)          │  │
│  │   values.yaml│  │ mkdocs.yml   │  │ release.yml (new)       │  │
│  └──────┬───────┘  └──────┬───────┘  └────────────┬────────────┘  │
│         │                 │                        │               │
└─────────┼─────────────────┼────────────────────────┼───────────────┘
          │                 │                        │
          │         on push to main                  │
          │         or on tag v*.*.*                 │
          │                 │                        │
          ▼                 ▼                        │
┌──────────────────────────────────────────────────┐ │
│                 GitHub Actions CI                 │ │
│                                                  │ │
│  Job: helm-release (tag only)                    │ │
│    helm package → helm push OCI → ghcr.io        │ │
│    helm package → helm repo index → git push     │ │
│                   gh-pages branch charts/        │ │
│                                                  │ │
│  Job: docs-deploy (main push + tags)             │ │
│    pip install mkdocs-material                   │ │
│    mkdocs build → site/                          │ │
│    peaceiris/actions-gh-pages                    │ │
│      destination_dir: .  (docs at root)          │ │
│      keep_files: true    (preserves charts/)     │ │
└──────────────────────────┬───────────────────────┘ │
                           │                         │
                           ▼                         │
┌──────────────────────────────────────────────────┐ │
│              gh-pages branch                     │ │
│                                                  │ │
│  / (root)          ← docs site (MkDocs output)   │ │
│  ├── index.html                                  │ │
│  ├── installation/                               │ │
│  ├── configuration/                              │ │
│  ├── architecture/                               │ │
│  └── charts/       ← Helm repo (manual index)   │ │
│      ├── index.yaml                              │ │
│      └── pulltrace-0.1.0.tgz                    │ │
│                                                  │ │
│  Served at: username.github.io/pulltrace         │ │
└──────────────────────────────────────────────────┘
           │                    │
           ▼                    ▼
   Browser loads docs    helm repo add pulltrace
   at root URL           https://user.github.io/pulltrace/charts
```

### Component Boundaries

| Component | Responsibility | Communicates With |
|-----------|---------------|-------------------|
| `docs/` (source) | MkDocs markdown source files and `mkdocs.yml` config | CI job reads these |
| `charts/pulltrace/` (source) | Helm chart source (Chart.yaml, templates, values) | CI packages these |
| CI `docs-deploy` job | Builds MkDocs site, pushes to gh-pages root via peaceiris/actions-gh-pages | gh-pages branch |
| CI `helm-release` job | Packages chart, pushes OCI to GHCR AND updates `charts/index.yaml` on gh-pages | GHCR OCI + gh-pages branch |
| `gh-pages` branch root | Hosts compiled MkDocs static site | GitHub Pages serves it |
| `gh-pages` branch `charts/` subdir | Hosts `index.yaml` + `.tgz` files for classic Helm repo | `helm repo add` reads it |
| GHCR OCI (`ghcr.io/d44b/charts/pulltrace`) | OCI-format Helm chart (already exists) | `helm install oci://...` |

---

## Recommended gh-pages Branch Structure

```
gh-pages/
├── index.html                  # MkDocs landing page
├── 404.html                    # MkDocs 404
├── assets/                     # MkDocs theme assets (CSS, JS)
├── installation/
│   └── index.html
├── configuration/
│   └── index.html
├── architecture/
│   └── index.html
├── contributing/
│   └── index.html
└── charts/                     # Helm repository (separate from docs)
    ├── index.yaml              # helm repo add pulltrace https://.../charts
    └── pulltrace-0.1.0.tgz    # Chart artifact (optional; can point to GH Release)
```

**Why this structure:**
- `helm repo add` takes the directory URL, not the file URL. Setting it to `https://owner.github.io/pulltrace/charts` causes Helm to automatically fetch `/charts/index.yaml`. This is explicitly supported by the Helm Chart Repository Guide.
- MkDocs output (root-level `index.html`, etc.) never conflicts with the `charts/` subdirectory because they are separate path namespaces.
- `peaceiris/actions-gh-pages` with `keep_files: true` preserves `charts/` when the docs job runs, and the helm job uses `keep_files: true` preserving the docs at root when writing to `charts/`.

---

## Architectural Patterns

### Pattern 1: Dual-Purpose gh-pages Branch (Docs at Root + Helm at Subpath)

**What:** Two independent GitHub Actions jobs each write to the `gh-pages` branch — the docs job writes to root, the helm job writes to `charts/` — with `keep_files: true` preventing either job from deleting the other's output.

**When to use:** When you want a real docs site at the project's GitHub Pages URL AND a classic `helm repo add` install path on the same domain, without maintaining separate repos or branches.

**Trade-offs:**
- Pro: Single repo, single domain, no external hosting
- Pro: Both installs work: OCI (already wired) and classic `helm repo add`
- Con: Race condition risk if both jobs run concurrently (mitigate: make helm job depend on docs job, or use separate workflow triggers)
- Con: `keep_files: true` means deleted chart versions are never pruned unless you explicitly clean up

**Example `docs-deploy` job:**
```yaml
name: Deploy Docs
on:
  push:
    branches: [main]
  release:
    types: [published]

permissions:
  contents: write

jobs:
  docs:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-python@v5
        with:
          python-version: "3.x"
      - run: pip install mkdocs-material
      - run: mkdocs build --strict
      - name: Deploy to gh-pages (root)
        uses: peaceiris/actions-gh-pages@v4
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: ./site
          destination_dir: .        # docs land at gh-pages root
          keep_files: true          # CRITICAL: preserves charts/ subdir
```

**Example `helm-pages` job (runs on semver tag):**
```yaml
  helm-pages:
    runs-on: ubuntu-latest
    needs: [docker]   # keep existing dependency on docker job
    if: startsWith(github.ref, 'refs/tags/v')
    steps:
      - uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - uses: azure/setup-helm@v4

      - name: Extract version
        id: version
        run: echo "version=${GITHUB_REF_NAME#v}" >> "$GITHUB_OUTPUT"

      - name: Update chart versions
        run: |
          VERSION=${{ steps.version.outputs.version }}
          sed -i "s/^version:.*/version: ${VERSION}/" charts/pulltrace/Chart.yaml
          sed -i "s/^appVersion:.*/appVersion: \"${VERSION}\"/" charts/pulltrace/Chart.yaml

      - name: Package chart
        run: helm package charts/pulltrace --destination /tmp/charts/

      - name: Checkout gh-pages
        uses: actions/checkout@v4
        with:
          ref: gh-pages
          path: gh-pages-branch

      - name: Update Helm repo index
        run: |
          mkdir -p gh-pages-branch/charts
          cp /tmp/charts/*.tgz gh-pages-branch/charts/
          helm repo index gh-pages-branch/charts/ \
            --url https://${{ github.repository_owner }}.github.io/${{ github.event.repository.name }}/charts \
            --merge gh-pages-branch/charts/index.yaml

      - name: Commit and push to gh-pages
        run: |
          cd gh-pages-branch
          git config user.name "github-actions[bot]"
          git config user.email "41898282+github-actions[bot]@users.noreply.github.com"
          git add charts/
          git diff --staged --quiet || git commit -m "chore: publish helm chart ${{ steps.version.outputs.version }}"
          git push
```

### Pattern 2: chart-releaser-action (Alternative — Simpler but Less Flexible)

**What:** `helm/chart-releaser-action@v1.7.0` automates chart packaging, GitHub Release creation, and `index.yaml` generation. It writes `index.yaml` to the root of the `gh-pages` branch by default.

**When to use:** When you do NOT need docs at the same GitHub Pages URL, or when you're comfortable with a separate branch/repo for docs.

**Trade-offs:**
- Pro: One action handles everything (package, GH Release creation, index.yaml)
- Pro: Chart artifacts live in GitHub Releases (not in gh-pages), keeping branch lean
- Con: Places `index.yaml` at root of gh-pages by default — conflicts with docs at root
- Con: The `pages-index-path` flag exists in the underlying `cr` CLI but is NOT exposed by the action's `action.yml` inputs (confirmed in chart-releaser-action issue #183 — PRs were opened but not merged as of research date). This makes subpath placement unreliable without workarounds.
- Con: Requires `fetch-depth: 0` and creates GitHub Releases as a side effect (may conflict with manually-crafted release notes workflow)

**Verdict for Pulltrace: DO NOT use chart-releaser-action.** The existing workflow already manually packages and pushes OCI. Extending it to also update `charts/index.yaml` on gh-pages (Pattern 1 above) gives both OCI and classic installs without disrupting the release notes flow.

### Pattern 3: OCI-Only (Current State — No Classic Helm Repo)

**What:** Helm chart published only to GHCR as OCI artifact. Already implemented.

**Install:** `helm install pulltrace oci://ghcr.io/d44b/charts/pulltrace --version 0.1.0`

**When to use:** When classic `helm repo add` is not required (enterprise / internal tools). For open-source discoverability, OCI-only is a barrier: ArtifactHub indexes OCI, but new users default to `helm repo add`.

**For Pulltrace:** Keep OCI push (already working) AND add classic repo. They are additive and non-conflicting.

---

## Data Flow

### Docs Publishing Flow (on push to main)

```
Developer pushes to main
    ↓
CI: docs-deploy job
    ↓
mkdocs build → site/ (static HTML)
    ↓
peaceiris/actions-gh-pages
    publish_dir: ./site
    destination_dir: .
    keep_files: true
    ↓
gh-pages branch root updated (charts/ preserved)
    ↓
GitHub Pages CDN serves updated docs
    ↓
https://owner.github.io/pulltrace/ (live)
```

### Helm Chart Release Flow (on semver tag v*.*.*)

```
Developer pushes git tag v0.1.0
    ↓
CI: helm-release job (existing OCI push, unchanged)
    helm package → /tmp/charts/pulltrace-0.1.0.tgz
    helm push oci://ghcr.io/d44b/charts
    ↓
CI: helm-pages job (new, runs after helm-release)
    checkout gh-pages branch
    copy .tgz to gh-pages/charts/
    helm repo index gh-pages/charts/ --merge
    git push gh-pages
    ↓
gh-pages branch charts/ updated (docs at root preserved)
    ↓
helm repo add pulltrace https://owner.github.io/pulltrace/charts
helm install pulltrace/pulltrace (live)
```

### Build Order Dependencies

```
lint-test ─┐
build-ui  ─┤→ docker (agent + server) ─→ helm-release (OCI push)
helm-lint ─┘                                    ↓
                                          helm-pages (gh-pages update)
                                                ↓
                                     (independent) docs-deploy
                                    (triggered by push to main, not tag)
```

**Critical ordering rule:** `helm-pages` must run AFTER `helm-release` (or at least after `helm package`), because it needs the packaged `.tgz`. It must NOT run concurrently with `docs-deploy` to avoid gh-pages branch race conditions. Use `needs: [helm-release]` in the workflow.

---

## Recommended Project Structure (Source Repository)

```
pulltrace/
├── .github/
│   └── workflows/
│       ├── ci.yml          # existing: lint/test/docker/helm OCI (modify to add helm-pages job)
│       └── docs.yml        # new: docs-deploy job (triggers on push to main)
├── charts/
│   └── pulltrace/          # existing Helm chart source
│       ├── Chart.yaml
│       ├── values.yaml
│       └── templates/
├── docs/                   # NEW: MkDocs source directory
│   ├── index.md            # landing page (mirrors README sections)
│   ├── installation.md     # helm install steps (OCI + classic)
│   ├── configuration.md    # values.yaml reference
│   ├── architecture.md     # how agent/server work
│   ├── contributing.md     # dev setup guide
│   └── adr/                # existing ADR markdown files (symlink or copy)
├── mkdocs.yml              # NEW: MkDocs config at repo root
├── web/                    # existing React UI source
└── internal/               # existing Go packages
```

**Structure rationale:**
- `docs/` at repo root is the MkDocs convention; `mkdocs.yml` at root is required by MkDocs default config
- Existing `docs/adr/` ADR files can be referenced directly in `mkdocs.yml` via `docs_dir` pointing to `docs/`
- Separating `docs.yml` workflow from `ci.yml` avoids coupling docs rebuilds to Go/Docker builds

---

## Anti-Patterns

### Anti-Pattern 1: Deploying Docs and Helm to the Same gh-pages Path Without `keep_files: true`

**What people do:** Run `mkdocs gh-deploy --force` alongside `helm/chart-releaser-action`, both writing to gh-pages root.

**Why it's wrong:** `mkdocs gh-deploy --force` wipes the entire gh-pages branch before writing. `chart-releaser-action` also performs a full push. Whichever job runs last destroys the output of the job that ran first. This is documented behavior, not a race: `--force` flag explicitly means "replace everything."

**Do this instead:** Use `peaceiris/actions-gh-pages` with `destination_dir` + `keep_files: true` for docs, and a separate manual git push job for `charts/`. Never use `mkdocs gh-deploy --force` when other content must coexist on gh-pages.

### Anti-Pattern 2: Using chart-releaser-action When You Already Have an OCI Push Workflow

**What people do:** Add `helm/chart-releaser-action` alongside the existing OCI push, hoping to get both install methods.

**Why it's wrong:** `chart-releaser-action` creates GitHub Releases automatically (named `chart-version`), which collides with your manually-written release notes release (named `v0.1.0`). It also requires `fetch-depth: 0` and manages its own branch state, adding complexity without benefit since the chart packaging step already exists.

**Do this instead:** Extend the existing `helm-release` job with a `helm-pages` step that runs `helm repo index --merge` and pushes the `charts/` subdir to gh-pages manually. This reuses the already-packaged `.tgz` without duplicating work.

### Anti-Pattern 3: Placing index.yaml at Root When Docs Site Is Also at Root

**What people do:** `helm repo add pulltrace https://owner.github.io/pulltrace` (pointing to root), expecting it to find `index.yaml` there — while also having MkDocs output at root.

**Why it's wrong:** The MkDocs `index.html` overwrites any `index.yaml` naming expectations, and the root is dominated by the docs site. Helm's `index.yaml` at root gets overwritten every time docs are redeployed.

**Do this instead:** Place Helm repo at `https://owner.github.io/pulltrace/charts`. Helm spec explicitly supports subdirectory repos: `helm repo add` appends `/index.yaml` to whatever URL you give it. Users run `helm repo add pulltrace https://owner.github.io/pulltrace/charts` — this is standard practice.

### Anti-Pattern 4: Publishing Both Docs and Charts on Every Commit

**What people do:** Trigger the Helm repo update on every push to main alongside the docs build.

**Why it's wrong:** `helm repo index --merge` on a non-semver commit may include development `.tgz` files with unstable version numbers. Classic Helm repos are immutable-by-convention; publishing intermediate versions creates a polluted index.

**Do this instead:** Docs publish on every main push (cheap, fast, no versioning concern). Helm chart update happens only on semver tags (`if: startsWith(github.ref, 'refs/tags/v')`). This matches the existing CI behavior.

---

## Integration Points

### External Services

| Service | Integration Pattern | Notes |
|---------|---------------------|-------|
| GitHub Pages | Static site hosting via `gh-pages` branch | Free on public repos; requires repo Settings → Pages → Source: gh-pages branch |
| GHCR OCI Registry | `helm push oci://ghcr.io/...` (already implemented) | Unchanged; coexists with classic repo |
| GitHub Releases | Created manually in release workflow | Do NOT use chart-releaser-action's auto-release feature alongside this |
| ArtifactHub | Indexes OCI registries AND classic Helm repos | Adding `artifacthub-repo.yml` to gh-pages root enables ArtifactHub listing for both |

### Internal Boundaries

| Boundary | Communication | Notes |
|----------|---------------|-------|
| `ci.yml` ↔ gh-pages branch | `git push` via actions-gh-pages action | Requires `contents: write` permission |
| `docs.yml` ↔ `ci.yml` | Independent workflows; no dependency | docs.yml can be a separate file to avoid coupling |
| `helm-pages` job ↔ `helm-release` job | `needs: [helm-release]` in ci.yml | Ensures .tgz exists before index update |
| `docs-deploy` job ↔ `helm-pages` job | Must NOT run concurrently on gh-pages | Separate triggers (main push vs tag) prevent most races; if both trigger on tag, add explicit `needs` |

---

## Scaling Considerations

This is a static site + file hosting concern, not an application scaling concern.

| Scale | Architecture Adjustments |
|-------|--------------------------|
| 1 chart version | Single `.tgz` + `index.yaml` in `charts/`; no cleanup needed |
| 10+ chart versions | `charts/` accumulates `.tgz` files; stay lean by pointing `index.yaml` at GitHub Release download URLs instead of serving `.tgz` from gh-pages (chart-releaser approach — achievable manually) |
| Multiple charts | Add subdirectory per chart under `charts/` or use separate `helm repo index` run per chart; `--merge` flag handles incremental additions |

### Scaling Priorities

1. **First concern:** gh-pages branch size. If `.tgz` files accumulate in the branch, clone time grows. Mitigate by using GitHub Release asset URLs in `index.yaml` (Helm supports external download URLs) rather than serving `.tgz` from gh-pages itself.
2. **Second concern:** Build ordering complexity as more jobs are added. Keep docs workflow separate from release workflow from the start (separate `.github/workflows/docs.yml` file).

---

## Sources

- [Helm Chart Repository Guide — official docs on subdirectory URL support](https://helm.sh/docs/topics/chart_repository/) — HIGH confidence
- [chart-releaser-action action.yml — confirms `pages-index-path` NOT exposed as input](https://github.com/helm/chart-releaser-action) — HIGH confidence (source read)
- [chart-releaser-action issue #183 — pages-index-path subpath request, unresolved](https://github.com/helm/chart-releaser-action/issues/183) — HIGH confidence
- [cr CLI flags — `--pages-index-path` exists at CLI level but not in action wrapper](https://github.com/helm/chart-releaser) — HIGH confidence
- [Material for MkDocs — publishing workflow, `mkdocs gh-deploy --force` behavior](https://squidfunk.github.io/mkdocs-material/publishing-your-site/) — HIGH confidence
- [peaceiris/actions-gh-pages — `destination_dir` + `keep_files` options](https://github.com/peaceiris/actions-gh-pages) — HIGH confidence
- [stefanprodan/helm-gh-pages — `target_dir` + `index_dir` inputs for subpath placement](https://github.com/stefanprodan/helm-gh-pages) — MEDIUM confidence (reviewed README via WebFetch)
- [MkDocs issue #2534 — subdirectory deploy limitation and workarounds](https://github.com/mkdocs/mkdocs/issues/2534) — MEDIUM confidence (confirmed via search)
- [Fabien Lee blog (Feb 2025) — chart-releaser-action complete workflow example](https://fabianlee.org/2025/02/26/github-automated-publish-of-helm-chart-using-github-actions/) — MEDIUM confidence

---

*Architecture research for: Pulltrace GitHub Pages docs + Helm repository hosting*
*Researched: 2026-02-23*
