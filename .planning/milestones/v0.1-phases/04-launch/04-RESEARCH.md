# Phase 4: Launch - Research

**Researched:** 2026-02-23
**Domain:** GHCR package visibility, git tag push, post-release artifact verification
**Confidence:** HIGH

## Summary

Phase 4 is the launch gate for v0.1.0. All CI automation was built and verified in Phase 3. This phase has exactly two things to do: (1) make the three GHCR packages public before the tag fires, and (2) push `git tag v0.1.0` to trigger the CI pipeline and then verify every success criterion is live. There is no code to write. The work is human-action-dependent for the visibility change (no CLI/API path exists) and a single git command for the tag push.

The critical dependency order is non-negotiable: **GHCR packages must be public BEFORE the tag is pushed.** If images are pushed to private packages first, unauthenticated `docker pull` will fail even after the CI run succeeds. Package visibility must be set once and cannot be reversed — making a package public is permanent on GHCR.

The three GHCR packages that must become public are `pulltrace-agent`, `pulltrace-server`, and `charts/pulltrace` (the OCI Helm chart package). All three will appear under `https://github.com/users/d44b/packages` once the CI workflow has pushed them at least once (any earlier branch push, or the test tag from Phase 3 verification, will have created them). The visibility change UI is under each package's "Package settings" → "Danger Zone" → "Change visibility".

**Primary recommendation:** Make all three GHCR packages public first, then push the annotated tag `git tag -a v0.1.0 -m "Release v0.1.0" && git push origin v0.1.0`, then run the verification checklist against live endpoints.

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| REL-03 | All three GHCR packages (`pulltrace-agent`, `pulltrace-server`, `charts/pulltrace`) are set to public visibility | GHCR visibility must be changed via UI per-package; no CLI/API path exists; must happen before tag push |
| REL-04 | v0.1.0 tag is pushed and all artifacts are live: Docker images at GHCR, Helm chart at OCI and classic repo, GitHub Release created | One `git push origin v0.1.0` triggers the full CI pipeline already wired in Phase 3; verification confirms each artifact endpoint |
</phase_requirements>

## Standard Stack

### Core
| Tool | Version | Purpose | Why Standard |
|------|---------|---------|--------------|
| GitHub Web UI | N/A | Change GHCR package visibility to public | Only supported method — no REST API or gh CLI for visibility mutation |
| git (annotated tag) | any | Create and push semver tag to trigger CI | Annotated tags carry a message; lightweight tags work but annotated is best practice for releases |
| curl / docker | system | Verify unauthenticated artifact pull post-launch | Fastest smoke test that needs no auth setup |
| gh CLI | 2.x | Query release status, verify GitHub Release page | `gh release view v0.1.0` confirms release body, attached assets, and latest flag |

### Supporting
| Tool | Version | Purpose | When to Use |
|------|---------|---------|-------------|
| helm | 3.x | Verify `helm repo add` and `helm install` resolve v0.1.0 | Post-launch smoke test; confirms index.yaml entry for 0.1.0 is reachable |
| docker logout ghcr.io | N/A | Ensure test environment is unauthenticated before pulling | Without this, cached credentials mask private-package errors |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| Manual UI for visibility | REST API `PATCH /user/packages/{type}/{name}` | API does not support changing visibility; UI is the only path (confirmed with docs) |
| Annotated tag | Lightweight tag `git tag v0.1.0` | Both trigger CI; annotated is convention for releases but not required |

## Architecture Patterns

### Execution Order (Non-Negotiable)

```
1. Verify packages exist in GHCR (confirm CI has run at least once on main)
2. Make pulltrace-agent public
3. Make pulltrace-server public
4. Make charts/pulltrace public
5. Push git tag v0.1.0 → triggers CI pipeline
6. Monitor CI: lint-test → build-ui → helm-lint → docker (×2) → helm-release → github-release
7. Verify each success criterion live
```

**Why this order matters:** If the tag fires before packages are public, the images exist at private URLs. Making them public afterward does not retroactively allow unauthenticated pulls for clients that cached the 401 response, and more importantly, any K8s cluster attempting `docker pull` will receive a 401 until visibility is public.

### CI Pipeline Flow (from Phase 3, verified)

```
push tag v*.*.*
    └── lint-test (Go vet + test)
    └── build-ui (npm ci + build)
    └── helm-lint
            └── docker (agent) ─┐
            └── docker (server) ─┤─ both push to ghcr.io/d44b/pulltrace-{agent,server}
                                └── helm-release
                                        packages chart → OCI ghcr.io/d44b/charts/pulltrace
                                        generates index.yaml
                                        deploys to gh-pages /charts/
                                        └── github-release
                                                creates GitHub Release
                                                attaches .tgz artifact
```

### Package Settings URL Pattern

Direct links to the package settings pages for `d44b`:

```
https://github.com/users/d44b/packages/container/pulltrace-agent/settings
https://github.com/users/d44b/packages/container/pulltrace-server/settings
https://github.com/users/d44b/packages/container/charts%2Fpulltrace/settings
```

The `charts/pulltrace` package name contains a slash — it is URL-encoded as `charts%2Fpulltrace` in the path. If the direct URL does not resolve, navigate via: GitHub profile → Packages tab → select each package → Package settings (right sidebar gear icon).

### Visibility Change UI Steps (per package)

1. Navigate to the package settings URL above
2. Scroll to "Danger Zone" section
3. Click "Change visibility"
4. Select "Public"
5. Type the exact package name to confirm (e.g., `pulltrace-agent`)
6. Click "I understand the consequences, change package visibility"

**Warning:** This action is irreversible. Once public, GHCR packages cannot be made private again.

### Tag Push Commands

```bash
# From main branch, clean working tree:
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

Do NOT use `git push --tags` (pushes all tags including any test/scratch tags). Push the specific tag by name.

### Verification Commands (post-launch)

```bash
# 1. Unauthenticated Docker pull (must docker logout first)
docker logout ghcr.io
docker pull ghcr.io/d44b/pulltrace-agent:0.1.0

# 2. GitHub Release
gh release view v0.1.0 --repo d44b/pulltrace

# 3. Helm classic repo
helm repo add pulltrace https://d44b.github.io/pulltrace/charts
helm repo update
helm search repo pulltrace --version 0.1.0

# 4. index.yaml entry
curl -s https://d44b.github.io/pulltrace/charts/index.yaml | grep "version: 0.1.0"
```

### Anti-Patterns to Avoid
- **Push tag before making packages public:** CI will push images to private packages; the images will exist but be unreachable unauthenticated. Must flip visibility first.
- **Use `git push --tags`:** Pushes any stale or test tags in addition to v0.1.0. Push the specific tag.
- **Test with authenticated shell:** `docker pull` will succeed even for private packages when credentials are cached. Must `docker logout ghcr.io` before testing.
- **Verify GitHub Release before CI completes:** `github-release` job runs last in the pipeline (after `helm-release`). Wait for the full pipeline green before verifying.

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Making packages public | Script or API call | GitHub Web UI | No API endpoint supports visibility mutation on GHCR packages |
| Creating the release | Manual `gh release create` | Let CI do it | `github-release` job is wired and verified — running it manually risks duplicating or clobbering the CI-created release |
| Updating index.yaml manually | Hand-editing the file | Let CI `helm repo index` do it | CI already targets correct URL; manual edits can break the digest or introduce format errors |

**Key insight:** Phase 4 is intentionally thin. All automation was built in Phase 3. The only manual action that has no automated substitute is the GHCR visibility change via UI.

## Common Pitfalls

### Pitfall 1: GHCR Package Not Yet Created
**What goes wrong:** The package settings URL returns 404 because no CI run has ever pushed to that package name.
**Why it happens:** GHCR packages are created lazily on first push. If no tag or branch push has run the `docker` job yet, the package doesn't exist.
**How to avoid:** Confirm packages exist before attempting visibility change. The `main` branch CI runs `docker` jobs on every push — at least one `main` push should have created both `pulltrace-agent` and `pulltrace-server`. The `charts/pulltrace` OCI package is only created on a semver tag (the `helm-release` job). If Phase 3 was validated with a test tag, the OCI chart package may already exist.
**Warning signs:** Package settings URL 404; package not listed at `https://github.com/users/d44b/packages`.

### Pitfall 2: OCI Chart Package Name Contains Slash
**What goes wrong:** Navigating to the `charts/pulltrace` package settings fails because the slash is not URL-encoded.
**Why it happens:** GHCR OCI push uses `oci://ghcr.io/d44b/charts` as the registry root, making the package name `charts/pulltrace` (with a slash). This is different from `pulltrace-agent` and `pulltrace-server`.
**How to avoid:** Use `charts%2Fpulltrace` in the URL, or navigate via the Packages tab UI.
**Warning signs:** 404 on the direct settings URL.

### Pitfall 3: docker pull Tests Against Authenticated Shell
**What goes wrong:** Verification passes locally because Docker credentials are cached, but external unauthenticated users still get 401.
**Why it happens:** `docker pull` transparently uses stored credentials from `docker login` without showing it.
**How to avoid:** Always run `docker logout ghcr.io` before testing public image accessibility. Alternatively, test from a fresh environment (CI job, docker-in-docker, or a machine that has never logged into GHCR).
**Warning signs:** `docker pull` succeeds but `curl -H "Authorization: " https://ghcr.io/v2/d44b/pulltrace-agent/tags/list` returns 401.

### Pitfall 4: CI Pipeline Times Out or Partial Failure
**What goes wrong:** One matrix job (e.g., `docker (server)`) fails, causing `helm-release` and `github-release` to not run.
**Why it happens:** Docker multi-arch builds (linux/amd64,linux/arm64) can be slow or hit QEMU issues.
**How to avoid:** Watch the full CI run in GitHub Actions after the tag push. If any job fails, check logs before re-tagging.
**Warning signs:** `github-release` job not present in the run, or tagged release not appearing on the Releases page.

### Pitfall 5: Re-tagging After Failed Run
**What goes wrong:** If the CI run fails and you delete and re-push the tag, the GitHub Release job may try to create a duplicate release.
**Why it happens:** `softprops/action-gh-release@v2` defaults to `draft: false, prerelease: false` — re-running creates or updates the release, which is safe. But the Helm index may have an older entry.
**How to avoid:** If re-running is needed, delete the tag locally and remotely, fix the issue, and re-push. The `helm repo index` step overwrites `index.yaml` in gh-pages, so the index will be consistent. The GitHub Release will be overwritten/created fresh.

## Code Examples

### Tag Create and Push
```bash
# Source: git documentation, standard practice
git tag -a v0.1.0 -m "Release v0.1.0"
git push origin v0.1.0
```

### Unauthenticated Smoke Test
```bash
# Source: GHCR docs — test true anonymous access
docker logout ghcr.io
docker pull ghcr.io/d44b/pulltrace-agent:0.1.0
docker pull ghcr.io/d44b/pulltrace-server:0.1.0
```

### Verify GitHub Release via CLI
```bash
# Source: gh CLI docs
gh release view v0.1.0 --repo d44b/pulltrace
# Should show: title, body with helm install commands, attached .tgz asset
```

### Verify Helm Classic Repo
```bash
# Source: Helm docs
helm repo add pulltrace https://d44b.github.io/pulltrace/charts
helm repo update
helm search repo pulltrace
# Should show: pulltrace/pulltrace   0.1.0   0.1.0
```

### Verify OCI Helm Chart
```bash
helm show chart oci://ghcr.io/d44b/charts/pulltrace --version 0.1.0
# Should return Chart.yaml contents without error
```

### Check index.yaml Directly
```bash
curl -s https://d44b.github.io/pulltrace/charts/index.yaml
# Should contain: version: 0.1.0, appVersion: "0.1.0"
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| chart-releaser-action for Helm + GitHub Releases | Custom ci.yml with helm package + peaceiris/actions-gh-pages + softprops/action-gh-release | Phase 3 decision | chart-releaser creates duplicate releases and forces index.yaml to repo root; custom approach separates concerns correctly |
| `provenance: true` on docker/build-push-action | `provenance: false` | Phase 3 (FIX applied) | Provenance:true on private GHCR packages triggers 403 on attestation blob check; disabled until packages are public |

**Note on provenance:** The ci.yml has `provenance: false` as a workaround for the GHCR 403 on private packages. Once the packages are public, provenance could theoretically be re-enabled. However, for v0.1.0 this is out of scope (SC-01/SC-02 are v2 requirements). Leave `provenance: false` as-is.

## Open Questions

1. **Does `charts/pulltrace` OCI package already exist?**
   - What we know: The `helm-release` job only runs on semver tags. Phase 3 verification notes mention a test tag was used to verify the workflow. If a test tag was pushed during Phase 3 development, the `charts/pulltrace` package will exist in GHCR.
   - What's unclear: Whether a test tag was actually pushed or if Phase 3 was verified statically/with stub CI runs.
   - Recommendation: Plan 04-01 should include a step to confirm all three packages exist before attempting visibility change. If `charts/pulltrace` does not exist yet, the OCI push step will run for the first time on the v0.1.0 tag — meaning the visibility change for `charts/pulltrace` can only happen after the tag push CI completes (or after a test tag creates it first).

2. **Index.yaml --merge behavior for v0.1.0**
   - What we know: STATE.md records the decision "No --merge flag for v0.1.0 helm repo index (no prior index.yaml exists; use --merge for v0.2.0+)". The ci.yml helm-release job does NOT use `--merge`. Phase 3 verification confirmed `index.yaml` is already live at the URL with a pulltrace 0.1.0 entry (pushed during Phase 3 testing).
   - What's unclear: The existing index.yaml was generated during Phase 3 test. When v0.1.0 tag fires, `helm repo index` without `--merge` will regenerate a fresh index.yaml from scratch with only the v0.1.0 chart. This is correct for v0.1.0 — there's only one version. No concern here.
   - Recommendation: No action needed.

## Validation Architecture

> Skipped — `workflow.nyquist_validation` is not present in `.planning/config.json` (the config only has `mode`, `depth`, `parallelization`, `commit_docs`, `model_profile`, and `workflow` keys for `research`, `plan_check`, `verifier`, `auto_advance`). Nyquist validation is not enabled.

## Sources

### Primary (HIGH confidence)
- [GitHub Docs: Configuring package access control and visibility](https://docs.github.com/en/packages/learn-github-packages/configuring-a-packages-access-control-and-visibility) — UI steps for making GHCR package public; confirmed no API path
- [GitHub Docs: REST API for packages](https://docs.github.com/en/rest/packages) — confirmed no visibility mutation endpoint
- `/Users/jmaciejewski/workspace/pulltrace/.github/workflows/ci.yml` — verified CI pipeline structure, tag trigger, job order
- `/Users/jmaciejewski/workspace/pulltrace/.planning/phases/03-release-automation/03-VERIFICATION.md` — confirmed all Phase 3 artifacts verified live; charts/index.yaml HTTP 200; docs site HTTP 200
- `/Users/jmaciejewski/workspace/pulltrace/charts/pulltrace/Chart.yaml` — confirmed version: 0.1.0, appVersion: "0.1.0" already set

### Secondary (MEDIUM confidence)
- [gh CLI discussion: no package visibility command](https://github.com/cli/cli/discussions/6003) — confirmed `gh` has no package visibility mutation; only `gh repo edit --visibility` for repos
- `curl ghcr.io/v2/d44b/pulltrace-agent/tags/list` returned HTTP 401 — confirms pulltrace-agent package exists but is currently private

### Tertiary (LOW confidence)
- WebSearch: GHCR package settings URL pattern `https://github.com/users/{owner}/packages/container/{name}/settings` — consistent across multiple community sources but not officially documented; treat as best-known navigation path

## Metadata

**Confidence breakdown:**
- GHCR visibility change (UI steps, irreversibility, no API): HIGH — confirmed in official docs
- Package settings URL pattern: MEDIUM — from community sources, not officially documented
- Tag push and CI trigger: HIGH — verified from ci.yml on:push.tags config
- Verification commands: HIGH — derived from live-verified Phase 3 endpoints
- OCI chart package existence: LOW — depends on whether a test tag was pushed in Phase 3

**Research date:** 2026-02-23
**Valid until:** 2026-04-23 (GitHub UI rarely changes for this workflow; stable)
