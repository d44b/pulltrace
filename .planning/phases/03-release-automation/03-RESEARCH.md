# Phase 3: Release Automation - Research

**Researched:** 2026-02-23
**Domain:** GitHub Actions CI — Helm classic repository indexing, GHCR OCI push, GitHub Release creation
**Confidence:** HIGH

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| HELM-01 | `helm repo add pulltrace https://d44b.github.io/pulltrace/charts` succeeds | Helm `repo index --url` flag sets download URLs; index.yaml at `/charts/index.yaml` must be served via gh-pages |
| HELM-02 | `helm install pulltrace pulltrace/pulltrace` installs from classic Helm repo | Depends on HELM-01 and HELM-03 being correct; no extra work once index.yaml is valid |
| HELM-03 | `https://d44b.github.io/pulltrace/charts/index.yaml` is publicly reachable | helm-pages job must push index.yaml to gh-pages `/charts/` path using peaceiris/actions-gh-pages with `destination_dir: charts` |
| HELM-04 | CI on semver tag packages `.tgz`, runs `helm repo index --merge`, pushes to gh-pages `/charts/` without overwriting docs | `keep_files: true` + `destination_dir: charts` is the established safe co-deployment pattern; concurrency group prevents race with docs.yml |
| REL-01 | `ci.yml` has `contents: write` permission (currently `contents: read`) | Trivial one-line change; required by both softprops/action-gh-release and the gh-pages push |
| REL-02 | Pushing `git tag v0.1.0` triggers CI to create a GitHub Release with title, body, and changelog link | softprops/action-gh-release@v2 is the standard action; body_path or inline body; no PAT needed with contents: write |
</phase_requirements>

---

## Summary

Phase 3 adds two CI jobs to an already partially-implemented `ci.yml`. The existing file has a `helm-release` job that only pushes to GHCR OCI; it is missing the classic `helm repo index` step and the `github-release` job. Additionally, the global `permissions: contents: read` blocks both GitHub Release creation and any push to the gh-pages branch — this must be flipped to `contents: write` as the first change.

The helm-pages publishing pattern is well-established: package the chart, run `helm repo index --merge` against an existing index (if present), then use `peaceiris/actions-gh-pages@v4` with `destination_dir: charts` and `keep_files: true` to deploy only the `/charts/` subdirectory while leaving the MkDocs docs site at the root of `gh-pages` intact. The `docs.yml` workflow already uses this action correctly for the docs root; both jobs need a shared concurrency group (`deploy-gh-pages`) with `cancel-in-progress: false` to prevent them racing to push the gh-pages branch when a semver tag is pushed on main.

The GitHub Release job uses `softprops/action-gh-release@v2` (current version as of late 2025), which requires `contents: write` and no PAT. The release body can be an inline heredoc or point to a file. Since `body_path` delivers the entire file verbatim, the recommended pattern for this project is an inline `body` string that summarizes what's new and links to the CHANGELOG `[0.1.0]` section — this avoids the action printing the entire CHANGELOG header boilerplate.

**Primary recommendation:** Extend the existing `helm-release` job with a `helm repo index --merge` → `peaceiris/actions-gh-pages` push step, add a separate `github-release` job using `softprops/action-gh-release@v2`, fix `contents: write`, and add a shared `deploy-gh-pages` concurrency group to both `ci.yml` and `docs.yml`.

---

## Standard Stack

### Core

| Tool/Action | Version | Purpose | Why Standard |
|-------------|---------|---------|--------------|
| `azure/setup-helm` | v4 | Install Helm CLI in runner | Already in ci.yml; official Microsoft action |
| `peaceiris/actions-gh-pages` | v4 | Push content to gh-pages branch | Already in docs.yml; only safe co-deployment option with `destination_dir` + `keep_files` |
| `softprops/action-gh-release` | v2 | Create GitHub Release on semver tag | Project decision (STATE.md); most widely used release action; no PAT required |
| `helm repo index` | (Helm CLI) | Generate/merge `index.yaml` for classic Helm repo | Built-in Helm command; `--merge` flag preserves existing entries |
| `helm package` | (Helm CLI) | Produce `.tgz` chart artifact | Already used in existing `helm-release` job |

### Supporting

| Tool | Purpose | When to Use |
|------|---------|-------------|
| `actions/checkout@v4` | Fetch source code | Every job |
| `github.ref_name` | Tag name (e.g., `v0.1.0`) in expressions | Extracting semver from tag trigger |
| `GITHUB_REF_NAME#v` bash substitution | Strip `v` prefix to get bare version | Needed for `helm package --version` and Chart.yaml sed |

### Alternatives Considered (and rejected)

| Instead of | Could Use | Why Rejected |
|------------|-----------|--------------|
| peaceiris/actions-gh-pages | helm/chart-releaser-action | Creates duplicate GitHub Releases, forces `index.yaml` to gh-pages root (conflicts with docs site) — documented in STATE.md |
| softprops/action-gh-release | gh CLI (`gh release create`) | gh CLI works but requires shell scripting for file attachment and body; action is cleaner and already decided in STATE.md |
| inline `body` in softprops | `body_path: CHANGELOG.md` | `body_path` delivers the entire file verbatim — CHANGELOG header boilerplate and `[Unreleased]` section appear in the release body. Inline body or a pre-extracted snippet is cleaner |
| Manual `git checkout gh-pages && git push` | peaceiris/actions-gh-pages | Manual git approach requires PAT or token setup, error-prone; peaceiris handles auth with `GITHUB_TOKEN` automatically |

---

## Architecture Patterns

### Pattern 1: Extend Existing `helm-release` Job (helm-pages step)

**What:** Append steps to the existing `helm-release` job (after OCI push) that checkout the gh-pages branch into a temp dir, run `helm repo index --merge`, then deploy via peaceiris.

**When to use:** Single job handles all chart publishing — OCI push + classic index — keeping needs chain simple.

**Approach — using peaceiris `destination_dir`:**

```yaml
# Append to helm-release job after the OCI push step

      - name: Checkout gh-pages for index merge
        uses: actions/checkout@v4
        with:
          ref: gh-pages
          path: gh-pages-branch
          # Only fetch the charts/ subdirectory to avoid pulling full docs
          sparse-checkout: charts
          sparse-checkout-cone-mode: true

      - name: Generate/merge Helm repo index
        run: |
          VERSION=${{ steps.version.outputs.version }}
          mkdir -p gh-pages-branch/charts
          # Copy packaged chart into the charts dir
          cp /tmp/charts/pulltrace-${VERSION}.tgz gh-pages-branch/charts/
          # Merge with existing index (if any) so previous versions are preserved
          helm repo index gh-pages-branch/charts \
            --url https://d44b.github.io/pulltrace/charts \
            --merge gh-pages-branch/charts/index.yaml 2>/dev/null || \
          helm repo index gh-pages-branch/charts \
            --url https://d44b.github.io/pulltrace/charts

      - name: Deploy charts/ to gh-pages
        uses: peaceiris/actions-gh-pages@v4
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: ./gh-pages-branch/charts
          publish_branch: gh-pages
          destination_dir: charts
          keep_files: true
          commit_message: "chore: Helm chart ${{ steps.version.outputs.version }}"
```

**Alternative approach — no sparse checkout (simpler):**

```yaml
      - name: Create chart staging dir
        run: mkdir -p /tmp/helm-pages

      - name: Copy packaged chart
        run: cp /tmp/charts/pulltrace-${{ steps.version.outputs.version }}.tgz /tmp/helm-pages/

      - name: Generate Helm index
        run: |
          helm repo index /tmp/helm-pages \
            --url https://d44b.github.io/pulltrace/charts

      - name: Deploy charts to gh-pages
        uses: peaceiris/actions-gh-pages@v4
        with:
          github_token: ${{ secrets.GITHUB_TOKEN }}
          publish_dir: /tmp/helm-pages
          publish_branch: gh-pages
          destination_dir: charts
          keep_files: true
          commit_message: "chore: Helm chart ${{ steps.version.outputs.version }}"
```

**IMPORTANT NOTE on --merge vs fresh index:** The `--merge` approach preserves old chart versions in `index.yaml` when new versions are published. The simple approach generates a fresh `index.yaml` containing only the current `.tgz` in `/tmp/helm-pages`. Since v0.1.0 is the first release, both approaches produce identical output. Use `--merge` for correctness in future releases.

The recommended pattern for this project is the **simpler no-sparse-checkout approach** for v0.1.0 (one chart, one version). When publishing v0.2.0+, add the sparse-checkout + merge logic.

### Pattern 2: Separate `github-release` Job

**What:** A dedicated job that runs after `helm-release` (needs: [helm-release]) on semver tags only, creates the GitHub Release using softprops/action-gh-release@v2, uploads the Helm chart `.tgz` as a release asset.

**Why separate from helm-release:** Clean job dependency graph; release body can reference a prior artifact; semantically separate concerns.

```yaml
  github-release:
    name: GitHub Release
    needs: [helm-release]
    runs-on: ubuntu-latest
    if: startsWith(github.ref, 'refs/tags/v')
    steps:
      - uses: actions/checkout@v4

      - name: Extract version
        id: version
        run: echo "version=${GITHUB_REF_NAME#v}" >> "$GITHUB_OUTPUT"

      - uses: azure/setup-helm@v4

      - name: Package chart for release asset
        run: |
          VERSION=${{ steps.version.outputs.version }}
          sed -i "s/^version:.*/version: ${VERSION}/" charts/pulltrace/Chart.yaml
          sed -i "s/^appVersion:.*/appVersion: \"${VERSION}\"/" charts/pulltrace/Chart.yaml
          helm package charts/pulltrace --destination /tmp/charts

      - name: Create GitHub Release
        uses: softprops/action-gh-release@v2
        with:
          name: "v${{ steps.version.outputs.version }}"
          body: |
            ## Pulltrace v${{ steps.version.outputs.version }}

            Real-time Kubernetes image pull progress monitor.

            ### Install

            **Helm (classic repo):**
            ```bash
            helm repo add pulltrace https://d44b.github.io/pulltrace/charts
            helm repo update
            helm install pulltrace pulltrace/pulltrace --version ${{ steps.version.outputs.version }}
            ```

            **Helm (OCI):**
            ```bash
            helm install pulltrace oci://ghcr.io/d44b/charts/pulltrace --version ${{ steps.version.outputs.version }}
            ```

            ### What's New

            See [CHANGELOG.md](https://github.com/d44b/pulltrace/blob/main/CHANGELOG.md#${{ steps.version.outputs.version }}) for full details.
          files: |
            /tmp/charts/pulltrace-${{ steps.version.outputs.version }}.tgz
          make_latest: "true"
```

### Pattern 3: Permissions Fix and Concurrency

**The permissions fix** (task 03-01, required for REL-01):

```yaml
# Change at top of ci.yml
permissions:
  contents: write   # was: read — must be write for GitHub Releases and gh-pages push
  packages: write
```

**The concurrency race problem:** When a semver tag is pushed on main, both `ci.yml` (tag trigger) and `docs.yml` (main trigger, if the tag push also updates main) could push to gh-pages simultaneously. However, based on ci.yml triggers: `docs.yml` only triggers on `branches: ["main"]` — a tag push does NOT trigger a branch push event. So the race only occurs if someone pushes a tag AND a commit to main simultaneously.

For safety, add a shared concurrency group to the helm-pages step only (not the whole ci.yml workflow):

```yaml
# In ci.yml helm-release job — add a concurrency group scoped to the job
# This prevents the github-release job from overlapping with helm-release on retries
# Cross-workflow: docs.yml must use the same group name

# Add to helm-release job:
concurrency:
  group: deploy-gh-pages
  cancel-in-progress: false
```

And update `docs.yml`:

```yaml
# docs.yml — change existing concurrency block to use shared group name
concurrency:
  group: deploy-gh-pages      # was: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: false   # was: true — never cancel a deployment in flight
```

### Anti-Patterns to Avoid

- **`helm repo index . --url URL`** without `--merge` on an existing index: Destroys previous chart versions from the index. Always use `--merge` when an `index.yaml` already exists.
- **`cancel-in-progress: true` on the deploy-gh-pages concurrency group:** Can abort a gh-pages push mid-commit, leaving the branch in a broken state. Always `false` for deployments.
- **Using `helm/chart-releaser-action`:** Creates its own GitHub Releases (conflicts with softprops job), and puts `index.yaml` at gh-pages root (conflicts with docs site). Documented as rejected in STATE.md.
- **Setting `contents: write` at job level only:** If the workflow-level permission is `read`, job-level cannot escalate beyond it. The fix must be at the workflow `permissions:` block.
- **`body_path: CHANGELOG.md`:** Dumps the entire 23-line CHANGELOG into the release body, including the `[Unreleased]` heading and comparison links. Write an inline body instead.
- **Forgetting `if: startsWith(github.ref, 'refs/tags/v')` on release jobs:** Without this guard, the release job fires on every branch push, failing because there is no tag to release.

---

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Push content to gh-pages branch | Custom git checkout/commit/push shell script | peaceiris/actions-gh-pages@v4 | Handles auth with GITHUB_TOKEN, concurrency guards, sparse push, keep_files semantics |
| Create GitHub Release | `gh release create` shell script | softprops/action-gh-release@v2 | Handles asset upload, idempotency, draft/prerelease, body templating cleanly |
| Generate Helm repo index | Write index.yaml manually | `helm repo index --merge` | Index format has digest, timestamps, URLs — all auto-computed by helm |

**Key insight:** All three problems have subtle edge cases (auth token scope, merge semantics, asset overwrite behavior) that the established actions handle correctly.

---

## Common Pitfalls

### Pitfall 1: `contents: read` Blocks Release Creation AND gh-pages Push

**What goes wrong:** The CI job attempting `softprops/action-gh-release@v2` or `peaceiris/actions-gh-pages@v4` fails with a 403 "Resource not accessible by integration" error.

**Why it happens:** The current `ci.yml` has `permissions: contents: read` at workflow level. Job-level permissions cannot escalate beyond the workflow level.

**How to avoid:** Change workflow-level to `permissions: contents: write` (and keep `packages: write`).

**Warning signs:** "403" or "Resource not accessible" in the release or pages deploy step logs.

### Pitfall 2: `helm repo index --url` Must Match the Served Path

**What goes wrong:** `helm repo add` succeeds, `helm repo update` succeeds, but `helm install` fails to download the `.tgz` with a 404.

**Why it happens:** The `--url` passed to `helm repo index` is baked into the `urls:` field of `index.yaml`. If it doesn't match where the `.tgz` files are actually served, downloads fail.

**How to avoid:** `--url https://d44b.github.io/pulltrace/charts` — must exactly match the `helm repo add` URL. The `.tgz` file must be deployed to the same `charts/` subdirectory on gh-pages.

**Warning signs:** `helm install` download step 404 even after successful `helm repo add`.

### Pitfall 3: First `helm repo index --merge` Fails if `index.yaml` Doesn't Exist Yet

**What goes wrong:** `helm repo index /tmp/helm-pages --merge /tmp/helm-pages/index.yaml` exits non-zero when the merge target doesn't exist (first publish).

**Why it happens:** The `--merge` flag expects the merge target to already exist.

**How to avoid:** Either (a) omit `--merge` on first publish (safe for v0.1.0 since there's no prior index), or (b) use the shell `||` fallback shown in the Pattern 1 example: try merge, fall back to fresh index on failure.

**Warning signs:** Step fails with "file not found" on the merge target path.

### Pitfall 4: Two Workflows Racing to Push gh-pages

**What goes wrong:** Both `docs.yml` and `ci.yml` attempt to `git push` to the `gh-pages` branch simultaneously; one push is rejected as non-fast-forward.

**Why it happens:** Without a shared concurrency group, GitHub can run both workflows at the same time. Specifically: if someone pushes a semver tag from main, the tag push could trigger `ci.yml` (tag trigger) while a recent commit on main triggers `docs.yml`.

**How to avoid:** Both workflows must share `concurrency: group: deploy-gh-pages` with `cancel-in-progress: false`.

**Warning signs:** `! [rejected] gh-pages -> gh-pages (non-fast-forward)` in the peaceiris step logs.

### Pitfall 5: Chart Version in Chart.yaml vs Tag Must Match

**What goes wrong:** `helm install pulltrace pulltrace/pulltrace --version 0.1.0` fails to find the chart, or installs the wrong version.

**Why it happens:** The `index.yaml` entry's `version:` field comes from `Chart.yaml` at package time. If the `sed -i` that updates `Chart.yaml` runs before `helm package`, versions match. If package runs first (e.g., steps reordered), the packaged `.tgz` has version `0.1.0` from the static Chart.yaml but the tag is `v0.2.0`.

**How to avoid:** Always run the `sed -i` version update step BEFORE `helm package`. Check: `helm show chart /tmp/charts/pulltrace-X.Y.Z.tgz | grep version`.

**Warning signs:** `helm install` resolves a chart with a different appVersion than the pushed tag.

### Pitfall 6: `github-release` Job Packages Chart Again (Duplicate sed)

**What goes wrong:** If both `helm-release` and `github-release` run `sed -i` on `Chart.yaml` in separate jobs, both start from a fresh checkout. No duplication issue by default — each job has a separate workspace. But if they run concurrently and share an artifact, the second sed could produce a corrupt file.

**How to avoid:** Use `needs: [helm-release]` to ensure `github-release` runs AFTER `helm-release`. Use the `upload-artifact` / `download-artifact` pattern to share the packaged `.tgz` between jobs, avoiding re-packaging.

**Better pattern (share artifact):**
```yaml
# In helm-release — upload the .tgz
      - uses: actions/upload-artifact@v4
        with:
          name: helm-chart
          path: /tmp/charts/pulltrace-*.tgz
          retention-days: 1

# In github-release — download and use it
      - uses: actions/download-artifact@v4
        with:
          name: helm-chart
          path: /tmp/charts
```

---

## Code Examples

Verified patterns from official sources and established CI patterns:

### Complete `permissions:` block fix (REL-01)

```yaml
# Source: GitHub Actions docs — permissions must be write at workflow level for release creation
permissions:
  contents: write   # Required by softprops/action-gh-release and peaceiris/actions-gh-pages push
  packages: write   # Required by docker/build-push-action and helm push OCI
```

### `helm repo index` with --url and --merge (HELM-03, HELM-04)

```yaml
# Source: https://helm.sh/docs/helm/helm_repo_index/
# Generate fresh index (v0.1.0 — no prior index exists)
- name: Generate Helm repo index
  run: |
    helm repo index /tmp/helm-pages \
      --url https://d44b.github.io/pulltrace/charts

# Generate with merge (v0.2.0+ — preserves prior versions in index)
- name: Generate/merge Helm repo index
  run: |
    helm repo index /tmp/helm-pages \
      --url https://d44b.github.io/pulltrace/charts \
      --merge /path/to/existing/charts/index.yaml
```

### `peaceiris/actions-gh-pages@v4` subdirectory deploy (HELM-04)

```yaml
# Source: https://github.com/peaceiris/actions-gh-pages
# destination_dir: charts — deploys ONLY to /charts/ subdirectory
# keep_files: true — preserves everything else on gh-pages (the docs site)
- uses: peaceiris/actions-gh-pages@v4
  with:
    github_token: ${{ secrets.GITHUB_TOKEN }}
    publish_dir: /tmp/helm-pages          # local dir with .tgz + index.yaml
    publish_branch: gh-pages
    destination_dir: charts               # target: gh-pages/charts/
    keep_files: true                      # CRITICAL: preserves docs site at root
    commit_message: "chore: Helm chart ${{ github.ref_name }}"
```

### `softprops/action-gh-release@v2` with inline body (REL-02)

```yaml
# Source: https://github.com/softprops/action-gh-release (v2.5.0 as of Dec 2025)
- uses: softprops/action-gh-release@v2
  with:
    name: "${{ github.ref_name }}"
    body: |
      ## Pulltrace ${{ github.ref_name }}

      ### Install via Helm

      ```bash
      helm repo add pulltrace https://d44b.github.io/pulltrace/charts
      helm repo update
      helm install pulltrace pulltrace/pulltrace --version ${{ steps.version.outputs.version }}
      ```

      See [CHANGELOG](https://github.com/d44b/pulltrace/blob/main/CHANGELOG.md) for full details.
    files: /tmp/charts/pulltrace-*.tgz
    make_latest: "true"
```

### Shared concurrency group (prevents docs/helm race)

```yaml
# In ci.yml helm-release job AND docs.yml deploy job — use SAME group name
concurrency:
  group: deploy-gh-pages
  cancel-in-progress: false    # never abort an in-flight gh-pages deployment
```

### Tag-only guard pattern

```yaml
# Both helm-release and github-release jobs use this guard
if: startsWith(github.ref, 'refs/tags/v')
```

---

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| `helm/chart-releaser-action` | Manual helm package + peaceiris + softprops | ~2022-2023 | chart-releaser creates duplicate releases and forces index.yaml to root; manual gives full control |
| PAT required for gh-pages push | GITHUB_TOKEN with `contents: write` | ~2021 (GH Actions fix) | No secrets required; safer |
| `body_path: CHANGELOG.md` | Inline `body:` with targeted content | N/A | Avoids dumping entire changelog into release |
| `softprops/action-gh-release@v1` | `@v2` | 2024 | Cross-platform support, updated Node runtime |

**Deprecated/outdated:**
- `helm/chart-releaser-action`: Still maintained but opinionated about release structure; incompatible with co-deployed docs site. Do not use.
- `peaceiris/actions-gh-pages@v3`: Superseded by v4; v4 supports Node 20 runtime.

---

## Open Questions

1. **Should `github-release` depend on `helm-release` or run in parallel?**
   - What we know: The release body references chart install commands that depend on HELM-01 being live. The Helm chart `.tgz` asset upload in the release requires the packaged file.
   - What's unclear: Whether it's acceptable for the GitHub Release to appear before the gh-pages Helm index is live (brief window where release exists but `helm repo add` would fail).
   - Recommendation: Use `needs: [helm-release]` to ensure the GitHub Release appears only after the chart is indexed. This also allows sharing the chart artifact via `upload-artifact`/`download-artifact`, avoiding re-packaging.

2. **Does `docs.yml` trigger on a tag push to main?**
   - What we know: `docs.yml` triggers on `push: branches: ["main"]`. A tag push is NOT a branch push event in GitHub Actions.
   - What's unclear: Whether the project workflow ever simultaneously tags and pushes to main (unlikely but possible).
   - Recommendation: Add the shared concurrency group defensively. It costs nothing and prevents the failure mode entirely.

3. **What happens to `index.yaml` when `keep_files: true` + `destination_dir: charts` is used and charts/ already exists?**
   - What we know: peaceiris `keep_files: true` preserves existing files in `destination_dir` that are NOT in the new `publish_dir`. So the existing docs at root are preserved. Files in `charts/` that are in the new deploy overwrite; files not in the new deploy are preserved (old `.tgz` files stay).
   - What's unclear: Whether old `.tgz` files accumulating in `charts/` (from prior releases) will be served correctly. Answer: Yes — they remain in the branch and are preserved by `keep_files`, which is the correct behavior for a multi-version Helm repo.
   - Recommendation: No action needed. The behavior is correct and desirable.

---

## Sources

### Primary (HIGH confidence)

- [softprops/action-gh-release README](https://github.com/softprops/action-gh-release) — v2.5.0 current version, all input parameters, `contents: write` requirement verified
- [peaceiris/actions-gh-pages](https://github.com/peaceiris/actions-gh-pages) — v4 current, `destination_dir`, `keep_files` semantics verified
- [helm.sh/docs/helm/helm_repo_index](https://helm.sh/docs/helm/helm_repo_index/) — `--merge`, `--url` flags, index format verified
- [helm.sh/docs/topics/chart_repository](https://helm.sh/docs/topics/chart_repository/) — `index.yaml` structure, URL scheme for subdirectory repos verified
- [docs.github.com — Control workflow concurrency](https://docs.github.com/en/actions/how-tos/write-workflows/choose-when-workflows-run/control-workflow-concurrency) — shared concurrency group pattern, `cancel-in-progress: false` for deployments verified

### Secondary (MEDIUM confidence)

- [codingtricks.io — Automating Helm chart packaging](https://codingtricks.io/automating-helm-chart-packaging-using-github-actions/index.html) — helm repo index --merge + git push pattern; cross-verified with helm official docs
- [dev.to/jamiemagee — How to host Helm chart on GitHub](https://dev.to/jamiemagee/how-to-host-your-helm-chart-repository-on-github-3kd) — GITHUB_TOKEN (no PAT) confirmed; cross-verified with GH Actions token docs
- Project STATE.md — key decisions: chart-releaser-action rejected, softprops chosen, peaceiris keep_files pattern confirmed

### Tertiary (LOW confidence)

- None — all key claims verified with official docs.

---

## Metadata

**Confidence breakdown:**
- Standard stack: HIGH — softprops v2.5.0, peaceiris v4, helm CLI verified against official docs and GitHub repos
- Architecture: HIGH — ci.yml and docs.yml read directly; gap analysis is concrete
- Pitfalls: HIGH — contents: read vs write is directly observable in ci.yml; --merge behavior from official helm docs; race condition from docs.yml trigger analysis

**Research date:** 2026-02-23
**Valid until:** 2026-05-23 (90 days — GitHub Actions action versions are stable; helm repo index behavior is stable)
