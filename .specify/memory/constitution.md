<!--
Sync Impact Report
Version change: 2.0.0 -> 2.1.0
Modified principles:
- II. Autonomous CLI Agent Contract -> II. Autonomous CLI Agent Contract
- IV. Minimal, Explicit Go Change -> IV. Minimal, Explicit Go Change
Added sections:
- None
Removed sections:
- None
Templates requiring updates:
- ✅ updated: .specify/templates/plan-template.md
- ✅ updated: .specify/templates/spec-template.md
- ✅ updated: .specify/templates/tasks-template.md
- ✅ updated: .opencode/command/speckit.plan.md
- ✅ updated: .opencode/command/speckit.tasks.md
- ✅ updated: .opencode/command/speckit.implement.md
- ✅ updated: .github/agents/speckit.plan.agent.md
- ✅ updated: .github/agents/speckit.tasks.agent.md
- ✅ updated: .github/agents/speckit.implement.agent.md
- ✅ updated: .specify/integrations/opencode.manifest.json
- ✅ updated: .specify/integrations/copilot.manifest.json
Follow-up TODOs:
- TODO(RATIFICATION_DATE): Original adoption date is unknown because the prior file was still an unratified template.
-->

# fastAI Constitution

## Core Principles

### I. Go CLI Product Authority
`fastAI` MUST be designed and implemented as a Go CLI tool. All planning, specifications,
tasks, and implementation guidance MUST assume the primary deliverable is a command-line
binary, not a chat UI, daemon-first service, or interactive wizard. The product MUST
optimize for clear command contracts, deterministic execution, portable builds, and
maintainable Go code that follows current industry standards. Rationale: product shape is
an architectural constraint; treating a CLI as a generic app produces the wrong APIs,
tests, and user experience.

### II. Autonomous CLI Agent Contract
The CLI's primary behavior MUST create and run a basic Copilot-style coding agent that
acts autonomously and is not interactive. User intent MUST be passed as a command
argument, for example `fastAI 'Do x y or z'`, and follow-up work MUST support explicit
session continuation through a `--session=<name>` flag, for example
`fastAI --session=whatever-i-want 'Do x y or z'`. Plans and specs MUST define how a new
session is created, how an existing session is resumed, what state is persisted, and how
non-interactive execution reports results without prompting for inline decisions. All AI
interactions MUST use `github.com/google/adk-go` as the integration layer for model,
agent, and tool orchestration unless the constitution is formally amended.
Rationale: autonomous agent behavior is the defining product contract, and session
semantics must be deliberate rather than incidental.

### III. Verification-First Go Architecture
All code MUST follow the engineering standard from `AGENTS.md`: pure functions first,
glue code second, end-to-end behavior last. Roughly 85% of test effort MUST target pure,
deterministic logic; about 10% MUST validate composition and integration boundaries; and
about 5% MUST cover end-to-end CLI behavior. TDD is the default when behavior changes.
All tests MUST be fast, and new architecture MUST improve testability rather than hide
logic behind side effects. Rationale: autonomous agents need reliable, fast feedback, and
Go code is easiest to maintain when core logic stays deterministic and decoupled.

### IV. Minimal, Explicit Go Change
Changes MUST use the smallest correct diff, keep code explicit, and favor standard
library or already-approved dependencies unless a new dependency is clearly justified.
Contributors MUST prefer simple packages, clear interfaces, and concrete types over
speculative abstractions. `github.com/google/adk-go` is the approved AI dependency and
MUST be used instead of ad hoc provider SDK combinations unless a documented limitation
requires a constitutional amendment. Backward-compatibility layers, hidden magic, and
unnecessary frameworks MUST not be added without a documented need. Rationale: Go code
remains maintainable when behavior is obvious from the source and complexity is
introduced only when demanded by the problem.

### V. Integration Parity and Session Continuity
Shared workflow behavior MUST remain aligned across supported integrations in this
repository, especially `.opencode/command/*`, `.github/agents/*`, shared templates, and
integration manifests. Any shared change that affects CLI semantics, autonomous agent
behavior, session handling, testing policy, or delivery workflow MUST update both
OpenCode and Copilot artifacts in the same change, including checksum manifests when
applicable. Rationale: parity failures create hidden product disagreements, especially
around session reuse and non-interactive execution.

## Go Engineering Standards

- All work MUST remain inside the repository root. Temporary artifacts MUST stay under
  `./tmp`, and ignored temporary paths MUST be added to `.gitignore` if needed.
- The `opencode` folder MUST be treated as reference-only and MUST NOT become a source of
  committed implementation changes.
- Go packages MUST have a single, clear responsibility and expose behavior that is easy
  to test in isolation.
- CLI entrypoints MUST keep orchestration separate from pure decision logic.
- Errors MUST be explicit, contextual, and returned up the call chain rather than hidden
  through panics except for truly unrecoverable startup faults.
- Configuration, filesystem access, process execution, and agent backends MUST be placed
  behind seams that allow deterministic tests.
- Any code that invokes AI models, tools, or agent workflows MUST wrap
  `github.com/google/adk-go` behind repository-local interfaces so pure logic remains
  testable and provider behavior can be stubbed deterministically.
- Manual file changes MUST use targeted patch-style edits so reviewers can inspect the
  exact intent.
- Edits MUST prefer ASCII unless the target file already requires other characters.

## CLI Product Constraints

- The default UX MUST be non-interactive. If required input is missing, the command MUST
  fail with a clear error and usage guidance instead of opening a prompt.
- The CLI MUST accept a free-form task prompt as an argument and MUST support
  `--session=<identifier>` for follow-up execution against prior agent context.
- Session identifiers, storage format, lifecycle rules, and failure recovery behavior
  MUST be documented in specs and validated in tests before implementation is complete.
- Plans MUST define the command surface, expected stdout/stderr behavior, and exit code
  contract for success, validation failures, and agent execution failures.
- Plans and specs MUST identify how `github.com/google/adk-go` is used for agent
  execution, tool calling, session continuity, and failure handling.
- Tasks MUST include direct verification of new-session flow, resumed-session flow, and
  non-interactive failure behavior.

## Governance

- This constitution supersedes conflicting defaults in templates, prompts, commands, and
  local habits inside this repository.
- Amendments MUST update `.specify/memory/constitution.md` together with any dependent
  templates, command files, agent files, and integration manifests that the amendment
  affects.
- Every amendment MUST include a Sync Impact Report at the top of this file that lists
  the version change, affected principles or sections, synced files, and deferred
  follow-up items.
- Versioning follows semantic versioning for governance changes:
  - MAJOR: removes or redefines a principle in a backward-incompatible way.
  - MINOR: adds a principle, section, or materially stronger requirement.
  - PATCH: clarifies wording or fixes non-semantic inconsistencies.
- Compliance review is REQUIRED during planning, task generation, implementation, and any
  human review of generated artifacts. Constitution violations are blocking until the
  artifacts are corrected or the constitution is formally amended.

**Version**: 2.1.0 | **Ratified**: TODO(RATIFICATION_DATE): Original adoption date unknown; template had not been previously ratified. | **Last Amended**: 2026-04-27
