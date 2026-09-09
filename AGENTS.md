# Codex — AI-DLC v1

## Load the repository workflow

Before working on this repository, read these files in order:

1. `CONVENTIONS.md` — notation, language, and commit conventions.
2. `CLAUDE.md` — shared repository policy, especially document ownership and
   Construction branches. Read the referenced files explicitly; do not treat
   Claude's `@path` notation as an automatic import in Codex.
3. `.aidlc/aidlc-rules/VERSION` — the installed version is **1.0.1**.
4. `.aidlc/aidlc-rules/aws-aidlc-rules/core-workflow.md` — the canonical workflow.

Use the vendored v1 rules directly. Do not download a newer version, copy the
workflow into another rules directory, or restart Inception merely to set up Codex.
Rule details resolve to `.aidlc/aidlc-rules/aws-aidlc-rule-details/`. Load the
common rules required by the core workflow and the relevant stage file before
executing that stage.

## Restore the owner and current stage

The user's stated handle takes precedence. If no handle is stated, read
`local/aidlc-context.md` when it exists. This is an optional, ignored file for this
checkout's owner and source paths; it is not a second workflow or progress log.
Do not infer the owner from Git author settings or another contributor's branch.

Read `aidlc-docs/construction-roster.md`, then the owner's
`aidlc-docs/<handle>/aidlc-state.md` and `audit.md` if present. Restore the existing
stage and decisions before producing artifacts. Setup and context restoration
do not count as completing or approving a development stage.

For the current Construction assignment, load the shared inputs as needed:

- Inception state and execution plan: `aidlc-docs/v1-run-dhseo/aidlc-state.md`
  and `aidlc-docs/v1-run-dhseo/inception/plans/execution-plan.md`.
- Unit definitions, dependency/file matrix, and gate mapping:
  `aidlc-docs/v1-run-dhseo/inception/application-design/`.
- Requirements and accepted decisions: `requirements/`.
- Shared reverse engineering: `aidlc-docs/inception/reverse-engineering/`.
- Protocol and ADR authority: the pinned `enode-design/` submodule, following
  `requirements/canon.md`.

Some older artifacts still mention pre-layering paths or completed stages as
pending. Resolve their paths through `CLAUDE.md` and the construction roster;
use the owner's current state, recorded approvals, and actual code to resume.
Do not repeat approved stages because a historical summary is stale.

## Document ownership and inherited decisions

- Construction artifacts, plans, state, and audit belong under
  `aidlc-docs/<handle>/`. Inception artifacts belong to their run directory.
- Interpret generic `aidlc-docs/construction/`, `aidlc-docs/aidlc-state.md`, and
  `aidlc-docs/audit.md` paths in the upstream rules relative to the owner's
  document root. Reverse engineering remains shared at its existing path.
- Update only the owner's state. Append timestamped user input and decisions to
  the owner's audit log; preserve the original input and existing entries.
- The facilitator owns shared/run state and `design/*.pen`. Follow the roster's
  branch-to-PR-to-main policy where older documents describe a previous policy.
- Inherit extension choices from the run state and `requirements/decisions.md`:
  security-baseline enabled, resiliency-baseline and property-based-testing
  disabled. Apply the accepted security scope and exceptions in decisions §3
  and §8. Do not ask for these choices again on resume.
- Preserve the approved execution plan's skipped stages. Follow its Functional
  Design, Code Generation, and Build and Test stages for each assigned unit,
  including their existing review gates.

## Work and verification

Check the current branch and changes before editing. Use an owner/unit topic
branch and preserve unrelated work. Check prerequisite code and gate evidence
before starting dependent units; distinguish unavailable prerequisites from
passing gates. Follow `requirements/scene-gates.md` for any deferred gate.

Application code belongs in the repository root, never in `aidlc-docs/`. Respect
the unit file matrix and import boundaries. In particular, `internal/api/ui`
must not import `internal/store`; new API handlers belong in separate files.

Use `requirements/scene-gates.md` and `.github/workflows/ci.yml` for the checks
appropriate to the change. DB-backed tests require `scripts/testdb.sh`; a run
with missing prerequisites or skipped DB tests is not a passing scene gate.
