# Timich MCP Project Journal

## Standard
- Canonical file: `.codex/Journal.md` at the project root.
- Always update the same project-root journal file when working from any worktree.
- Keep entries grouped by worktree.
- Use one compact Journey Log entry per session.
- Prefer references to features, components, or spec areas instead of file lists.
- Do not include machine-specific paths, personal identifiers, or secrets.
- Mark deprecated content as `(deprecated by <changes reference>)` or with strikethrough.
- Content older than one month can be removed when no longer useful.
- When the journal becomes long, compact it by removing deprecated logs and summarizing older entries.

## Worktree: main
Workspace: timich-mcp
Branch: codex/add-ci-release-workflows
Last Updated: 2026-05-14

### Architecture & Design Decisions
- Decision: Protect the default branch using the same lightweight rule profile as the companion agent repository.
  - Context: The repository previously had no protection on `main`.
  - Choice: Require pull requests for changes, apply the rule to admins, keep required approvals and status checks disabled, and forbid force pushes and deletions.
  - Alternatives: Require approval reviews or status checks immediately.
  - Tradeoffs: The selected profile prevents direct pushes while avoiding new CI or review gates.
  - References: Repository branch protection policy.
- Decision: Mirror the companion repository's GitHub Actions shape for PR verification and releases.
  - Context: The repository had no workflow automation, while install docs already referred to GitHub Releases.
  - Choice: Add a PR workflow that runs the shared verify script, plus a tag/manual release workflow that builds platform bundles and checksum files using the source-defined Makefile release target.
  - Alternatives: Inline all CI commands in workflow YAML, keep a public-repo-only release builder script, or publish a single host-platform binary.
  - Tradeoffs: Calling `make dist` keeps the public workflow aligned with the monorepo source-of-truth artifact shape, while the workflow still owns the platform matrix and GitHub Release publishing.
  - References: Pull request verification and release publishing.

### Journey Log
- 2026-05-13 (Session: main branch protection): Mirrored the companion repository's `main` protection profile onto this repository.
  - Outcome: `main` now requires pull requests, applies to admins, and disallows force pushes and deletion.
  - References: GitHub branch protection settings.
- 2026-05-13 (Session: PR and release workflows): Added GitHub Actions automation for pull request verification and GitHub Release publishing.
  - Outcome: Pull requests to `main` run Go tests and a build; releases can be created from `v*` tags or manual workflow dispatch with version input.
  - References: GitHub Actions workflows and release artifact builder.
- 2026-05-14 (Session: PR handoff): Prepared the CI and release workflow changes for pull request review.
  - Outcome: Verification passed locally before branch push and PR creation.
  - References: Pull request branch for workflow automation.
- 2026-05-14 (Session: release target alignment): Updated the release workflow branch to use the Makefile `dist` target instead of a separate public-repo release builder script.
  - Outcome: The public workflow now builds each release platform by calling `make dist`, matching the monorepo source-defined release packaging.
  - References: Release workflow, Makefile release target.
- 2026-05-14 (Session: publish artifact review): Scoped the Makefile publish target to the artifact created by the current publish invocation.
  - Outcome: `publish-dist` no longer uploads every same-version artifact already present in the output directory.
  - References: Makefile release target, release publishing.

### Lessons Learned
- Pattern to repeat: Compare repository policy through GitHub API before changing branch protection.
- Pattern to repeat: Keep CI logic in scripts that can run both locally and in GitHub Actions.
- Pattern to repeat: Keep artifact layout in Make targets that can be source-synced from the Timich monorepo.
- Pitfall to avoid: Adding approval or status-check requirements when the request only asks to block direct pushes.
- Pitfall to avoid: Copying Linux-only release targets from the agent repository when the MCP binary is installed on client machines.
- Pitfall to avoid: Letting public repository-only release scripts drift from the monorepo source of truth.
- Pitfall to avoid: Uploading stale same-version release artifacts from the output directory without manifest-style validation.
- Prevention checklist: Verify the resulting protection JSON after updating.
- Prevention checklist: Run the PR verify script and at least one release artifact build before handoff.

### Prompt Summary (User Requests)
- 2026-05-13:
  - Request: Protect this repository's `main` branch like the companion agent repository.
  - Scope: GitHub branch protection only.
  - Constraints: Direct pushes should be blocked and pull requests should be required.
- 2026-05-13:
  - Request: Make pull requests run tests like the companion agent repository and add a release workflow.
  - Scope: GitHub Actions workflows and local CI/release helper scripts.
  - Constraints: Match the companion repository pattern while adapting release platforms for MCP client installs.

### Prompt Retrospective (Improve Requests Next Time)
- Clarify target behavior: State whether approvals, status checks, or conversation resolution should also be required.
- Define constraints up front: Mention whether admins should be included or exempted.
- Call out non-goals/detours to avoid: Avoid adding CI gates unless explicitly requested.

### Implementation Retrospective & TODOs
- [ ] Consider adding required status checks to branch protection after the first PR workflow run establishes the check name.
