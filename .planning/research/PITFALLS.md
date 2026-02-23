# Pitfalls Research

**Domain:** Open source Kubernetes tool first release (Helm + GHCR + GitHub Pages + GitHub Releases)
**Researched:** 2026-02-23
**Confidence:** HIGH (most pitfalls verified against official GitHub and Helm documentation)

---

## Critical Pitfalls

### Pitfall 1: GHCR Packages Are Private by Default — Users Can't Pull Your Images

**What goes wrong:**
You push `pulltrace-agent` and `pulltrace-server` to GHCR via CI and the workflow succeeds. Then users try to `helm install` or `docker pull ghcr.io/d44b/pulltrace-agent:0.1.0` without authenticating and get a 401 Unauthorized or "not found" error. The Helm chart install fails at the image pull step inside Kubernetes with `ImagePullBackOff`.

**Why it happens:**
GitHub's official documentation confirms: "When you first publish a package that is scoped to your personal account, the default visibility is private and only you can see the package." This applies to every image and the Helm OCI chart artifact. The CI workflow pushes successfully (it authenticates via `GITHUB_TOKEN`), so the push gives no hint the result is private.

Additionally, for organization-owned repos, the organization's "Package creation" setting in member privileges must have "Public packages" enabled before any individual package can be made public. If that org setting is off, the "Make public" button doesn't appear on the package settings page.

**How to avoid:**
After the first tag push completes CI:
1. For each package (`pulltrace-agent`, `pulltrace-server`, `charts/pulltrace`): go to `github.com/users/d44b/packages` → select package → Settings → "Change visibility" → Public.
2. One-time action only — GitHub warns this is irreversible, which is fine for an open source project.
3. Verify anonymously: `docker pull ghcr.io/d44b/pulltrace-agent:0.1.0` from a machine not logged in to GHCR, or use `curl -s https://ghcr.io/v2/d44b/pulltrace-agent/tags/list` (no auth header).
4. Optional: Automate in CI using `gh` CLI after push: `gh api --method PATCH /user/packages/container/pulltrace-agent -f visibility=public`.

**Warning signs:**
- `helm install` works in your own environment (where you're already `docker login`'d to GHCR) but fails for a colleague who hasn't logged in.
- `helm pull oci://ghcr.io/d44b/charts/pulltrace --version 0.1.0` returns "not found" or 401 for anonymous users.
- Package appears in your GitHub profile Packages tab but with a lock icon.

**Phase to address:**
Release phase (v0.1.0 tag + GitHub Release). Verify package visibility as an explicit step in the release checklist before announcing.

---

### Pitfall 2: OCI Helm Install Is Non-Obvious — Users Expect `helm repo add`

**What goes wrong:**
The README says "install via Helm" and users reflexively try:
```
helm repo add pulltrace <some-url>
helm install pulltrace pulltrace/pulltrace
```
Neither command works with OCI-only distribution. `helm repo add` does not support `oci://` URLs — it only works with classic HTTP index-based repos. Users get no useful error message and conclude the chart is broken or the project is unmaintained.

**Why it happens:**
`helm repo add` and `helm search repo` are incompatible with OCI registries. Helm 3.8+ supports OCI natively via `helm install oci://...`, but the mental model gap between classic Helm repos and OCI registries is significant. Most Kubernetes operators and tutorials still show `helm repo add` workflows. Community discussions on external-secrets, apollographql/router, and other projects confirm this is a recurring user confusion point.

**How to avoid:**
Two complementary approaches (pick one or both):

**Option A — Document OCI install clearly and prominently:**
Put the OCI install command on line 1 of the Installation section, with exact syntax:
```bash
helm install pulltrace oci://ghcr.io/d44b/charts/pulltrace \
  --version 0.1.0 \
  --namespace pulltrace --create-namespace
```
Explicitly state that `helm repo add` is not used.

**Option B — Also serve a classic Helm repo via GitHub Pages:**
Run `helm repo index` on the packaged chart and publish `index.yaml` to GitHub Pages. This enables `helm repo add https://d44b.github.io/pulltrace && helm install pulltrace pulltrace/pulltrace`. The OCI push and the Pages index can coexist — CI already packages the chart; add a step to also write an `index.yaml`. The `helm/chart-releaser-action` automates this pattern.

For v0.1.0, Option A is faster. Option B is better UX for the long run. The PROJECT.md already captures this decision.

**Warning signs:**
- Issue tracker fills with "helm repo add doesn't work" questions within the first week.
- Users open issues asking "what's the repo URL?"

**Phase to address:**
Documentation phase (GitHub Pages + install guide). Address in the Installation page, not just README.

---

### Pitfall 3: GitHub Release Exists But Has No Attached Artifacts or Checksums

**What goes wrong:**
You push `git tag v0.1.0` and GitHub auto-creates a release page with just the auto-generated source tarball (the `.tar.gz` GitHub generates from the tag). There are no binary builds, no container image digest list, no checksums. Users who find the release page have nothing downloadable except source code — the images only live in GHCR with no reference from the release.

The existing `ci.yml` has no `create-release` job. The workflow pushes Docker images and Helm chart to GHCR on tag, but the GitHub Release itself is never created by CI.

**Why it happens:**
The `ci.yml` workflow has no job that calls `softprops/action-gh-release` or `gh release create`. The `helm-release` job pushes to GHCR OCI but writes nothing to the GitHub Releases page. Without a workflow job that runs on `refs/tags/v*`, GitHub creates only the bare auto-generated release.

**How to avoid:**
Add a `github-release` job to `ci.yml` that:
1. Runs `if: startsWith(github.ref, 'refs/tags/v')` after `docker` and `helm-release` complete.
2. Uses `softprops/action-gh-release@v2` (the current maintained action; `actions/create-release` is deprecated and unmaintained).
3. Attaches: `pulltrace-$VERSION-linux-amd64.tar.gz`, `pulltrace-$VERSION-linux-arm64.tar.gz`, and `checksums.txt` generated with `sha256sum`.
4. The `contents: write` permission must be present in the workflow's permissions block — the current `ci.yml` only has `contents: read`. This will silently fail release creation.

```yaml
permissions:
  contents: write   # required for release creation
  packages: write   # already present
```

**Warning signs:**
- `git tag v0.1.0 && git push origin v0.1.0` completes but the Releases page shows "No releases published" or only has the auto-generated source zip.
- CI passes green but no release job appears in the Actions tab.
- Current `ci.yml` permissions block: `contents: read` — this will block release creation.

**Phase to address:**
Release automation phase. Fix `contents: read` → `contents: write` in `ci.yml` before pushing the first tag.

---

### Pitfall 4: GitHub Pages Deploys But Docs Links Are All Broken (baseurl)

**What goes wrong:**
You build a docs site and deploy it. The Pages URL is `https://d44b.github.io/pulltrace/` (project page, not user page). Every internal link, CSS reference, and image resolves to `https://d44b.github.io/` instead of `https://d44b.github.io/pulltrace/` — every page except the root 404s and the site looks broken.

**Why it happens:**
GitHub project pages (not user/org pages) are served under a path prefix matching the repo name (`/pulltrace`). Jekyll's `baseurl` must be set to `/pulltrace` in `_config.yml`. If left blank or set to `/`, all `relative_url` and `absolute_url` filter calls generate wrong paths. This is one of the oldest and most persistent Jekyll gotchas — the issue tracker has entries going back to 2012 that are still referenced today.

If using plain HTML/Vite docs (not Jekyll), the same issue applies: any relative path like `./assets/main.js` works, but absolute paths like `/assets/main.js` break because the root is `/pulltrace/`, not `/`.

**How to avoid:**
- **Jekyll:** Set `baseurl: "/pulltrace"` in `_config.yml`. Use `{{ page.url | prepend: site.baseurl }}` for all internal links, or use `relative_url` filter consistently. Test locally with `bundle exec jekyll serve --baseurl /pulltrace`.
- **Plain HTML/Vite:** Set `base: '/pulltrace/'` in `vite.config.js` for the docs site build. All asset references will then be prefixed correctly.
- **MkDocs:** Set `site_url: https://d44b.github.io/pulltrace/` in `mkdocs.yml` — MkDocs handles the rest automatically.
- Test the deployed site by clicking every navigation link before announcing.

**Warning signs:**
- CSS/JS loads on the homepage but inner pages return 404.
- Browser network tab shows 404s for `/assets/...` instead of `/pulltrace/assets/...`.
- Works perfectly at `localhost:4000` but broken at `d44b.github.io/pulltrace/`.

**Phase to address:**
Documentation phase (GitHub Pages setup). Verify correct baseurl before the v0.1.0 announcement.

---

## Moderate Pitfalls

### Pitfall 5: Helm OCI Chart Is Private Even Though Docker Images Are Public

**What goes wrong:**
You make `pulltrace-agent` and `pulltrace-server` images public in GHCR, but forget that the Helm chart OCI artifact (`ghcr.io/d44b/charts/pulltrace`) is a separate package with its own visibility setting. Users can pull images but `helm install oci://ghcr.io/d44b/charts/pulltrace` still fails with 401.

**Why it happens:**
Each pushed OCI artifact is its own package in GHCR. Publishing to `oci://ghcr.io/d44b/charts/pulltrace` creates a new package entry at `github.com/users/d44b/packages/container/charts%2Fpulltrace` — different URL from the Docker image packages. It starts private like all GHCR packages. The three packages that need to be made public are independent: `pulltrace-agent`, `pulltrace-server`, and `charts/pulltrace`.

**How to avoid:**
After first tag push, explicitly visit the packages page and make all three public. Include in the release checklist. Verify with `helm pull oci://ghcr.io/d44b/charts/pulltrace --version 0.1.0` from an unauthenticated shell.

---

### Pitfall 6: GitHub Pages Source Branch Not Configured — Site Never Deploys

**What goes wrong:**
You push a `gh-pages` branch (or configure a GitHub Actions workflow to deploy Pages), but the Pages site URL returns 404 for weeks. The workflow runs green but nothing is served.

**Why it happens:**
GitHub Pages must be explicitly enabled in the repository Settings → Pages. The source (branch or GitHub Actions) must be selected. It is NOT enabled by default. For GitHub Actions deployment, you also need `pages: write` and `id-token: write` permissions in the deploy workflow. Without these, `actions/deploy-pages` silently fails or produces a confusing auth error.

**How to avoid:**
- Go to Settings → Pages → Source → select "GitHub Actions" (or specific branch).
- Ensure the deploy workflow has:
  ```yaml
  permissions:
    pages: write
    id-token: write
    contents: read
  ```
- Check Actions tab for a "pages-build-and-deployment" workflow run after the first push.

---

### Pitfall 7: Release Workflow Runs Before Docker Push Is Visible — Helm Chart References Missing Images

**What goes wrong:**
The CI creates a GitHub Release with release notes that say "install version 0.1.0" before the Docker images are fully visible (either still processing or still private). Users who install within the first few minutes of a release get `ImagePullBackOff` because the images aren't available yet.

**Why it happens:**
GHCR image publication is near-instant once the push completes, but there is a propagation window. More critically: if the GitHub Release creation job runs in parallel with or before the Docker push job, the images may genuinely not exist yet when users install.

The existing `ci.yml` correctly gates `helm-release` on `needs: [docker]`. The pattern to follow for the GitHub Release job is the same: `needs: [docker, helm-release]` so it only fires after all artifacts are live.

**How to avoid:**
Gate the `github-release` job on `needs: [docker, helm-release]`. This means the GitHub Release is only created (and notifications sent) after GHCR has all three artifacts.

---

### Pitfall 8: CONTRIBUTING.md Missing — First Contributors Have No Path In

**What goes wrong:**
A user files an issue, wants to contribute a fix, clones the repo, and immediately hits the Go 1.22 build requirement that their local machine (Go 1.18) can't satisfy. There's no CONTRIBUTING.md explaining how to build via Docker, what the test command is, or how to run the full stack locally. The user gives up. The issue stays open.

**Why it happens:**
CONTRIBUTING.md is treated as non-critical documentation that can be added "later." But the first contributor experience happens at the first public release — there's no "later" before someone needs it.

For Pulltrace specifically, the Go 1.22 requirement is a real barrier: the MEMORY.md explicitly documents "local machine has Go 1.18 — cannot build locally, use Docker/CI." This is non-obvious and must be documented.

**How to avoid:**
CONTRIBUTING.md must include:
1. "You need Go 1.22. If your local Go is older, build via `docker buildx build`" — with the exact command.
2. "Run tests: `go test ./... -v -race`" — requires a k8s cluster? No? Document it.
3. PR template: small, focused, one issue per PR.
4. How to run the full stack locally (kind/k3d cluster steps or `docker compose` if one exists).

---

## Technical Debt Patterns

| Shortcut | Immediate Benefit | Long-term Cost | When Acceptable |
|----------|-------------------|----------------|-----------------|
| Skip `checksums.txt` in GitHub Release | Faster release workflow | Security-conscious users won't install; downstream automation that verifies checksums breaks | Never for a binary distribution; skip for source-only |
| OCI-only Helm (no classic repo) | Zero extra work — already wired | Users who muscle-memory `helm repo add` get confused; some GitOps tools still prefer classic repos | Acceptable if docs prominently explain OCI install |
| Auto-generated GitHub release notes only | Zero effort | Doesn't capture "what changed", doesn't link to Helm chart version | Acceptable for v0.1.0 if CHANGELOG.md exists and is linked |
| Skip ArtifactHub registration | Zero setup work | Chart is not discoverable via `helm search hub`; reduces organic discovery | Acceptable for v0.1.0 but register before any public announcement |
| Hardcode `contents: read` in CI permissions | Follows least-privilege pattern | Blocks release creation job; silent fail | Never — fix before first tag |

---

## Integration Gotchas

| Integration | Common Mistake | Correct Approach |
|-------------|----------------|------------------|
| GHCR + Helm OCI | Assume chart inherits image package visibility | Chart is a separate package; make it public independently at `github.com/users/<owner>/packages` |
| GitHub Pages + Jekyll | Leave `baseurl` unset on a project page | Set `baseurl: "/pulltrace"` in `_config.yml`; test with `--baseurl` flag locally |
| GitHub Actions release job | Forget `contents: write` permission | Explicitly set `permissions: contents: write` in the job or top-level workflow |
| GitHub Actions release job | Create release in parallel with artifact build | Always use `needs: [docker, helm-release]` to gate release creation |
| `softprops/action-gh-release` | Not providing `tag_name` | Action auto-detects from `github.ref` on tag push; explicit is clearer |
| Helm chart OCI push | Version in `Chart.yaml` doesn't match git tag | The existing `ci.yml` correctly `sed`s the version — verify this runs before `helm package` |

---

## "Looks Done But Isn't" Checklist

- [ ] **GHCR packages public:** CI pushed images green does NOT mean they're publicly pullable. Verify `docker pull ghcr.io/d44b/pulltrace-agent:0.1.0` without GHCR login.
- [ ] **Helm OCI chart public:** Separate from Docker images. Verify `helm pull oci://ghcr.io/d44b/charts/pulltrace --version 0.1.0` without login.
- [ ] **GitHub Release has attached assets:** Auto-generated source zip is NOT the release. Verify binaries/checksums are attached to the GitHub Release page.
- [ ] **CI has `contents: write`:** Current `ci.yml` has `contents: read` — a release creation job will fail silently. Verify before pushing v0.1.0 tag.
- [ ] **GitHub Pages enabled in repo settings:** The workflow alone doesn't serve a site. Verify Settings → Pages shows the source and "Your site is live at..." message.
- [ ] **Docs links work on deployed site, not just locally:** Click through every page at `d44b.github.io/pulltrace/` — broken baseurl only manifests on the live domain.
- [ ] **CONTRIBUTING.md mentions Go 1.22 Docker workaround:** Unique to this project — the local build limitation is critical to document.

---

## Recovery Strategies

| Pitfall | Recovery Cost | Recovery Steps |
|---------|---------------|----------------|
| GHCR packages left private | LOW | Make packages public in GHCR settings. No tag re-push needed. |
| GitHub Release missing artifacts | MEDIUM | Edit release via UI to attach files, or delete + re-run CI workflow via `gh release delete v0.1.0 && git push origin v0.1.0`. |
| Broken Pages baseurl | LOW | Fix `_config.yml` (or `vite.config.js`), push to Pages branch — redeploys in ~1 minute. |
| Release job failed (wrong permissions) | LOW | Fix `ci.yml` permissions, delete the bad tag, re-tag: `git tag -d v0.1.0 && git push --delete origin v0.1.0 && git tag v0.1.0 && git push origin v0.1.0`. |
| No classic Helm repo — user confusion | MEDIUM | Add `index.yaml` to GitHub Pages and update docs. Can be done post-release without breaking existing OCI installs. |

---

## Pitfall-to-Phase Mapping

| Pitfall | Prevention Phase | Verification |
|---------|------------------|--------------|
| GHCR packages private | Release phase (pre-announcement checklist) | `docker pull ghcr.io/d44b/pulltrace-agent:0.1.0` without auth succeeds |
| Helm OCI chart private | Release phase (pre-announcement checklist) | `helm pull oci://ghcr.io/d44b/charts/pulltrace --version 0.1.0` without auth succeeds |
| OCI-only Helm confusion | Documentation phase | Install guide prominently shows `helm install oci://...` syntax; no `helm repo add` dead end |
| GitHub Release missing artifacts | Release automation phase | Release page shows binaries + checksums attached, not just auto-generated zip |
| `contents: read` blocks release | Release automation phase | CI workflow permissions updated before v0.1.0 tag is pushed |
| GitHub Pages broken baseurl | Documentation phase | All nav links on live `d44b.github.io/pulltrace/` resolve correctly |
| Pages not enabled in settings | Documentation phase | Settings → Pages shows "Your site is live" before docs are announced |
| CONTRIBUTING.md missing Go workaround | Documentation phase | Dev setup section in CONTRIBUTING.md includes Docker build command |
| Release created before images visible | Release automation phase | `github-release` job uses `needs: [docker, helm-release]` |

---

## Sources

- [GitHub Docs: Configuring a package's access control and visibility](https://docs.github.com/en/packages/learn-github-packages/configuring-a-packages-access-control-and-visibility) — HIGH confidence; confirms packages are private by default, irreversible public change
- [GitHub Docs: Working with the Container registry](https://docs.github.com/en/packages/working-with-a-github-packages-registry/working-with-the-container-registry) — HIGH confidence
- [Helm Docs: Use OCI-based registries](https://helm.sh/docs/topics/registries/) — HIGH confidence; confirms `helm repo add` incompatibility with OCI
- [GitHub Docs: Configuring a publishing source for GitHub Pages](https://docs.github.com/en/pages/getting-started-with-github-pages/configuring-a-publishing-source-for-your-github-pages-site) — HIGH confidence; confirms Pages must be explicitly enabled, permissions required
- [Made Mistakes: Jekyll's site.url and baseurl](https://mademistakes.com/mastering-jekyll/site-url-baseurl/) — MEDIUM confidence; long-running authoritative reference on baseurl issue
- [Jekyll issue #332: baseurl relative links fail](https://github.com/jekyll/jekyll/issues/332) — MEDIUM confidence; confirms this is a known long-standing issue
- [softprops/action-gh-release](https://github.com/softprops/action-gh-release) — MEDIUM confidence; confirmed as the current maintained release action
- [GitHub community discussion: Release checksums](https://github.com/orgs/community/discussions/23512) — MEDIUM confidence; community confirms expected practice
- [Helm chart-releaser-action issues: non-fast-forward errors](https://github.com/helm/chart-releaser-action/issues/180) — MEDIUM confidence; real reported mistake
- [Classic Helm Repo vs OCI Helm Package confusion](https://medium.com/@prayag-sangode/classic-helm-repo-vs-oci-helm-package-understanding-helm-chart-packaging-3ff54bb16b00) — MEDIUM confidence
- [GHCR org visibility setting required for public packages](https://github.com/orgs/community/discussions/26014) — MEDIUM confidence; org-level setting blocks individual package visibility change

---
*Pitfalls research for: Open source Kubernetes tool first release (Helm + GHCR + GitHub Pages + GitHub Releases)*
*Researched: 2026-02-23*
