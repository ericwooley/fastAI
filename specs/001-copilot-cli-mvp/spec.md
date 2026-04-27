# Feature Specification: Copilot CLI MVP

**Feature Branch**: `001-copilot-cli-mvp`  
**Created**: 2026-04-27  
**Status**: Draft  
**Input**: User description: "Lets get an MVP going with full support for file editing, command execution, and login with github copilot. There must also be a required flag to specify a model."

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Start Authenticated Agent Run (Priority: P1)

As a developer, I want to sign in with my GitHub Copilot account and start a non-interactive
agent run with a required model selection so I can delegate a coding task from the CLI.

**Why this priority**: Without authenticated agent execution, the product cannot deliver its
core value as a Copilot-style autonomous CLI tool.

**Independent Test**: A user signs in, runs one command with a task prompt and explicit model
selection, and receives a clear success or failure result without any interactive follow-up.

**Acceptance Scenarios**:

1. **Given** a user has not signed in yet, **When** they complete GitHub Copilot login and
   run the CLI with a task prompt and model flag, **Then** the system starts an autonomous
   agent run and reports the outcome through standard command output.
2. **Given** a user is already signed in, **When** they run the CLI with a task prompt and
   model flag, **Then** the system reuses the authenticated account and starts the agent run
   without asking for inline decisions.

---

### User Story 2 - Apply File Changes (Priority: P2)

As a developer, I want the agent to edit files in the current workspace so it can complete
coding tasks end to end.

**Why this priority**: File editing is required for the MVP to produce useful coding results
instead of only returning suggestions.

**Independent Test**: A signed-in user runs a task that requires creating or updating files,
and the workspace reflects the requested changes while the CLI reports what changed.

**Acceptance Scenarios**:

1. **Given** a signed-in user runs a task that requires file changes, **When** the agent
   finishes, **Then** the relevant files are created, updated, or removed in the workspace
   and the CLI summarizes the changes.
2. **Given** a task would affect files outside the allowed workspace, **When** the agent
   attempts the change, **Then** the system blocks the action and reports a failure clearly.

---

### User Story 3 - Run Repository Commands (Priority: P3)

As a developer, I want the agent to run commands in the repository and resume follow-up work
with a saved session so it can verify results and continue tasks across multiple invocations.

**Why this priority**: Command execution and follow-up continuity are necessary for practical
coding workflows, but they can build on the authenticated run from User Story 1.

**Independent Test**: A signed-in user starts one run that executes repository commands, then
starts a second run with `--session` and receives follow-up behavior tied to the earlier run.

**Acceptance Scenarios**:

1. **Given** a signed-in user starts a task that requires repository commands, **When** the
   agent executes those commands, **Then** the CLI reports command results and whether the
   overall task succeeded or failed.
2. **Given** a prior session exists, **When** the user runs a follow-up command with the same
   session identifier and a new prompt, **Then** the agent continues from that prior session
   context instead of starting over.

### Edge Cases

- The user runs the CLI without a task prompt.
- The user runs the CLI without the required model flag.
- The user attempts to start an agent run before completing GitHub Copilot login.
- The saved GitHub Copilot login is expired, revoked, or otherwise unusable.
- The user supplies an unknown, expired, or corrupted `--session` value.
- The task requests file changes or commands outside the active repository workspace.
- A command started by the agent exits with a non-zero status.

## Requirements *(mandatory)*

### Functional Requirements

- **FR-001**: The system MUST expose the MVP through a command-line interface.
- **FR-002**: The system MUST require users to authenticate with GitHub Copilot before an
  agent run can use Copilot-backed capabilities.
- **FR-003**: The system MUST require an explicit model flag for every agent run.
- **FR-004**: The system MUST reject agent runs that omit the required model flag.
- **FR-005**: The system MUST execute requested agent workflows autonomously without
  interactive prompts during the run.
- **FR-007**: The system MUST allow the agent to create, update, and delete files within the
  active repository workspace when needed to complete a task.
- **FR-008**: The system MUST prevent file operations outside the active repository
  workspace.
- **FR-009**: The system MUST allow the agent to execute repository commands needed to
  complete a task.
- **FR-010**: The system MUST report command execution outcomes, including failures, through
  standard command output.
- **FR-011**: The system MUST support follow-up work with `--session=<identifier>`.
- **FR-012**: The system MUST persist enough session state to resume prior agent context
  safely.
- **FR-013**: The system MUST return clear command output and exit status for
  success, validation failures, authentication failures, and agent execution failures.

### Key Entities *(include if feature involves data)*

- **Authenticated Account**: Represents a local CLI login tied to a GitHub Copilot-enabled
  user identity and its current sign-in status.
- **Agent Session**: Represents a saved execution context for an autonomous run, including a
  session identifier, selected model, status, and enough history to support follow-up work.
- **Task Run**: Represents one non-interactive execution request, including the user prompt,
  requested capabilities, resulting actions, and final outcome.
- **Workspace Change**: Represents a file creation, modification, or deletion produced by a
  task run inside the active repository.
- **Command Result**: Represents a repository command the agent executed, including the
  command requested, exit status, and summarized output.

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: 90% of signed-in users can start a first agent run with an explicit model
  selection in under 2 minutes.
- **SC-002**: 95% of valid runs that request only file editing within the workspace complete
  with a clear final outcome and without requiring interactive clarification.
- **SC-003**: 95% of valid runs that request repository command execution return a clear
  success or failure result that users can act on immediately.
- **SC-004**: 90% of follow-up runs using an existing session identifier continue prior work
  without the user needing to restate the full original task.

## Constitutional Alignment *(mandatory)*

- **Repo Scope Impact**: Expected changes are limited to the Go CLI entrypoint, internal
  agent/session/CLI packages, local authentication and session persistence behavior,
  repository-safe file editing, repository command execution, and related tests/docs.
- **Verification Strategy**: Pure-function tests cover flag validation, workspace safety
  rules, session validation, and result classification; integration tests cover login state,
  session persistence, file-edit orchestration, command execution orchestration, and agent
  backend boundaries; CLI end-to-end tests cover authenticated runs, required `--model`
  enforcement, file-edit tasks, command-execution tasks, and resumed-session flows.
- **Integration Parity Impact**: N/A for this feature spec; downstream workflow/template
  parity is already governed by the repository constitution.
- **CLI Contract Impact**: Users invoke the CLI with a free-form prompt argument, MUST
  provide a required model flag for each run, MAY provide `--session=<identifier>` for
  follow-up work, and receive non-interactive command output plus exit codes. Repository
  implementation MUST still use `github.com/google/adk-go` for AI-driven agent execution as
  required by the constitution.
- **Complexity Justification**: Session persistence, authenticated account state, file-edit
  controls, and command execution boundaries are required to deliver the MVP safely.

## Assumptions

- Users run the CLI from inside the repository they want the agent to work on.
- Interactive prompts during an active agent run are out of scope for the MVP.
- GitHub Copilot login can be established once and reused across later CLI invocations until
  it expires or is revoked.
- File editing and command execution are limited to the active repository workspace rather
  than arbitrary system-wide access.
- Users know which model they want to use and will provide it explicitly on every run.
- Session state is stored locally and is available between CLI runs.
