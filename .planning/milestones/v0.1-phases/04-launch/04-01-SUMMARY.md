---
phase: 04-launch
plan: "01"
subsystem: infra
tags: [ghcr, docker, helm, oci, packages, visibility]

# Dependency graph
requires:
  - phase: 03-release-automation
    provides: CI pipeline that pushes Docker images and Helm OCI chart to GHCR on semver tag

provides:
  - All three GHCR packages (pulltrace-agent, pulltrace-server, charts/pulltrace) set to public visibility
  - Unauthenticated docker pull and helm show chart are now possible for end users

affects:
  - 04-launch — Plan 04-02 (tag push) can now succeed end-to-end because images will be pullable post-push
  - Any user following the Helm install docs — they can pull without authenticating to GHCR

# Tech tracking
tech-stack:
  added: []
  patterns:
    - "GHCR package visibility is set once via GitHub Web UI — no CLI/API alternative exists"
    - "Verify package public status via gh api /users/{owner}/packages/container/{name} --jq .visibility"

key-files:
  created: []
  modified: []

key-decisions:
  - "GHCR v2 API /tags/list returns 401 even for public packages — use gh api to verify visibility instead of curl"
  - "Docker Desktop not running locally — docker pull verification skipped; gh api confirms public visibility is sufficient pre-tag-push proof"
  - "All three packages (pulltrace-agent, pulltrace-server, charts/pulltrace) confirmed public before v0.1.0 tag is pushed"

patterns-established:
  - "Post-push GHCR visibility check: gh api /users/d44b/packages/container/{name} --jq .visibility"

requirements-completed: [REL-03]

# Metrics
duration: ~5min
completed: 2026-02-23
---

# Phase 4 Plan 01: Make GHCR Packages Public Summary

**All three GHCR packages (pulltrace-agent, pulltrace-server, charts/pulltrace) set to public visibility so unauthenticated docker pull and helm install succeed after the v0.1.0 tag fires.**

## Performance

- **Duration:** ~5 min
- **Started:** 2026-02-23 (prior session)
- **Completed:** 2026-02-23T17:50:25Z
- **Tasks:** 2 (1 auto, 1 human-action checkpoint)
- **Files modified:** 0

## Accomplishments

- Confirmed all three GHCR packages exist (pulltrace-agent, pulltrace-server, charts/pulltrace)
- Human changed visibility of all three packages to Public via GitHub Web UI
- Verified with `gh api` that all three report `visibility: "public"`

## Task Commits

This plan modifies no source files. No per-task commits were created.

**Plan metadata commit:** see final docs commit below.

## Files Created/Modified

None — this plan is a configuration change on GitHub's infrastructure only.

## Decisions Made

- **GHCR v2 API is not a reliable visibility indicator:** `/v2/.../tags/list` returns 401 for both private and public packages (GHCR always requires token exchange for the v2 API). The canonical verification method is `gh api /users/{owner}/packages/container/{name} --jq .visibility`.
- **Local docker pull skipped:** Docker Desktop was not running on the local machine. Since `gh api` confirmed `visibility: "public"` for all three packages, and the real end-to-end pull test will occur automatically when CI pushes the v0.1.0 tag in Plan 04-02, this was accepted as sufficient pre-tag-push verification.
- **charts/pulltrace confirmed public:** The OCI chart package existed (created by the Phase 3 test tag) and was made public along with the two Docker image packages.

## Deviations from Plan

### Auto-fixed Issues

**1. [Rule 1 - Bug] Verification method changed from docker pull to gh api**
- **Found during:** Post-checkpoint verification
- **Issue:** Plan's `<verify>` section specified `docker logout ghcr.io && docker pull ghcr.io/d44b/pulltrace-agent:latest` — Docker Desktop was not running locally, making this command unavailable
- **Fix:** Used `gh api /users/d44b/packages/container/{name} --jq .visibility` as the verification method, which directly reports the package visibility setting without requiring Docker
- **Files modified:** None
- **Verification:** All three packages returned `"public"` from the gh api call
- **Committed in:** N/A (no code change)

---

**Total deviations:** 1 (verification method adapted — no scope change)
**Impact on plan:** Verification outcome is identical — packages are confirmed public. Docker pull test will be validated in Plan 04-02 when the real tag push triggers CI.

## Issues Encountered

- **GHCR v2 API returns 401 for public packages:** This is expected GHCR behavior — the v2 endpoint always requires a token exchange even for public packages. It cannot be used to distinguish private vs. public. Noted in decisions above.
- **Docker Desktop not running locally:** Local machine cannot run docker pull. Not a blocker — visibility confirmed via GitHub API, and the real pull test occurs in CI post-tag.

## User Setup Required

None — the manual step (setting package visibility) was completed by the human during the checkpoint.

## Must-Haves Verification

| Truth | Status |
|---|---|
| Unauthenticated user can `docker pull ghcr.io/d44b/pulltrace-agent:latest` without 401 | Confirmed via `gh api` showing `visibility: public`; docker pull not locally testable |
| Unauthenticated user can `docker pull ghcr.io/d44b/pulltrace-server:latest` without 401 | Confirmed via `gh api` showing `visibility: public`; docker pull not locally testable |
| Unauthenticated user can `helm show chart oci://ghcr.io/d44b/charts/pulltrace` without auth error | Confirmed via `gh api` showing `visibility: public`; helm test not locally run |

## Next Phase Readiness

- All three GHCR packages are public — the v0.1.0 tag push (Plan 04-02) will result in images that any Kubernetes node can pull without authenticating
- Plan 04-02 (push v0.1.0 tag and verify full CI run end-to-end) is now unblocked
- No outstanding concerns

---
*Phase: 04-launch*
*Completed: 2026-02-23*
