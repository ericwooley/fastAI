# Research: Copilot CLI MVP

## Decision: Use Cobra for the CLI command surface

**Rationale**: Cobra is a mature Go CLI framework with built-in required-flag support,
subcommands, help generation, and context-aware command execution, which matches the MVP's
need for a `login` subcommand plus a root task-running command with a required `--model`
flag.

**Alternatives considered**:
- Standard library `flag`: rejected because nested commands, consistent usage output, and
  required flag validation would need custom plumbing with little product value.
- Bubble Tea/Fang TUI stack: rejected because the constitution requires a non-interactive
  default UX and the MVP does not need a TUI.

## Decision: Use GitHub OAuth device flow for `fastAI login`

**Rationale**: GitHub device flow is designed for CLI tools and is the most realistic way to
let a local developer authenticate once in a browser and reuse that identity later. Using
`github.com/cli/oauth` keeps the flow small and aligned with existing GitHub CLI patterns,
while later non-interactive runs can reuse the stored token or preconfigured
`COPILOT_GITHUB_TOKEN`, `GH_TOKEN`, or `GITHUB_TOKEN` values.

**Alternatives considered**:
- Browser-only localhost callback flow: rejected because device flow is simpler for a local
  CLI and does not require running a callback server.
- Prompting for a personal access token manually: rejected because the feature explicitly
  requires login with GitHub Copilot rather than token copy/paste.

## Decision: Keep `google.golang.org/adk` as the agent runtime and add a repository-local GitHub Models adapter

**Rationale**: The constitution requires `github.com/google/adk-go` for all AI interactions,
but current ADK Go materials show strong Gemini-first support rather than a built-in GitHub
Models backend. The smallest compliant design is to keep ADK as the orchestration layer and
implement a thin repository-local model adapter that uses the authenticated GitHub token to
call the GitHub model endpoint selected by `--model`. This preserves constitutional ADK usage
without introducing a second agent framework.

**Alternatives considered**:
- Switching to a non-ADK framework with first-class GitHub Models support: rejected because
  it violates the constitution.
- Waiting for official ADK Go provider support: rejected because it blocks MVP delivery.
- Using ad hoc provider SDK combinations directly from CLI code: rejected because the
  constitution explicitly disallows bypassing ADK as the integration layer.

## Decision: Store auth and session state in the user's OS config directory, namespaced by repository root

**Rationale**: Authentication material should not live in the repository workspace. Storing
auth and session records under the OS config directory avoids accidental commits while still
supporting cross-run reuse. Namespacing sessions by canonical repository root or a stable hash
prevents follow-up context from bleeding across projects.

**Alternatives considered**:
- Storing sessions in the repo itself: rejected because it risks committing secrets and adds
  noisy implementation files to the user's workspace.
- Using a database: rejected as unnecessary operational complexity for a single-user CLI MVP.

## Decision: Enforce repository safety with canonical-path guards and explicit command execution boundaries

**Rationale**: File editing and command execution are the highest-risk capabilities in scope.
Every requested path should be resolved against the active repository root after symlink and
relative-path normalization, and commands should run only with an explicit working directory,
captured output, and bounded execution context. This keeps the safety boundary deterministic
and unit-testable.

**Alternatives considered**:
- Trusting agent-generated paths and commands directly: rejected because it violates the spec
  and constitution's safety requirements.
- Allowing arbitrary absolute paths with warnings: rejected because the MVP promises
  repository-bound behavior.

## Decision: Follow the AGENTS.md test pyramid with pure logic first

**Rationale**: Flag validation, session ID rules, repo-safety checks, output classification,
and execution planning are all deterministic logic and should dominate the test suite. Thin
integration tests should cover auth/session/file/command adapters, and a small CLI e2e layer
should confirm login, required `--model`, file editing, command execution, and resumed-session
flows.

**Alternatives considered**:
- Heavy e2e-first coverage: rejected because it is slower, harder to debug, and conflicts
  with the repository's explicit testing standard.
