# Data Model: Copilot CLI MVP

## AuthenticatedAccount

**Purpose**: Represents the locally stored GitHub-backed identity used for Copilot-authenticated
agent runs.

**Fields**:
- `provider`: constant identifier for the GitHub/Copilot auth source
- `user_id`: stable GitHub account identifier when available
- `login`: human-readable GitHub login name
- `access_token_ref`: reference to the stored token material
- `scopes`: granted capabilities needed for model access
- `expires_at`: optional token expiry timestamp
- `last_validated_at`: timestamp of the last successful authenticated operation

**Validation rules**:
- Token material must exist before a run can start.
- Expired or revoked credentials must force a re-login path.
- Only one active account is needed for the MVP.

**Relationships**:
- One `AuthenticatedAccount` can authorize many `AgentSession` records.

## AgentSession

**Purpose**: Stores the minimum cross-run context needed to resume work safely with
`--session=<identifier>`.

**Fields**:
- `session_id`: user-provided or generated identifier
- `repo_key`: stable key derived from the canonical repository root
- `model`: required model selected for the run
- `status`: `active`, `completed`, or `failed`
- `created_at`: creation timestamp
- `updated_at`: last mutation timestamp
- `last_prompt`: most recent task prompt
- `history_ref`: pointer to persisted turn/run history
- `last_run_id`: identifier of the most recent `TaskRun`

**Validation rules**:
- `session_id` must be non-empty and filesystem-safe.
- A resumed session must match the current repository key.
- A run cannot resume a missing or corrupted session record.

**State transitions**:
- `active` -> `completed` when a run finishes successfully
- `active` -> `failed` when execution aborts irrecoverably
- `completed` -> `active` when a valid follow-up run resumes the session

## TaskRun

**Purpose**: Captures one autonomous CLI execution from prompt receipt through final outcome.

**Fields**:
- `run_id`: unique execution identifier
- `session_id`: linked session identifier
- `prompt`: free-form task input from the CLI
- `model`: required model flag value
- `requested_capabilities`: file editing and/or command execution
- `started_at`: run start timestamp
- `finished_at`: run completion timestamp
- `outcome`: `succeeded`, `failed`, `validation_failed`, or `auth_failed`
- `summary`: user-facing completion summary

**Validation rules**:
- `prompt` must be present.
- `model` must be present for every run, including resumed sessions.
- Outcome must map to a documented exit status.

**Relationships**:
- One `TaskRun` belongs to one `AgentSession`.
- One `TaskRun` may produce many `WorkspaceChange` and `CommandResult` records.

## WorkspaceChange

**Purpose**: Represents a filesystem mutation requested or produced during a task run.

**Fields**:
- `path`: repository-relative path
- `operation`: `create`, `update`, or `delete`
- `status`: `applied` or `blocked`
- `reason`: explanation for blocked or failed work
- `bytes_changed`: optional size delta for reporting

**Validation rules**:
- Path must remain under the canonical repository root.
- Operations outside the repo or through unsafe path traversal are blocked.

## CommandResult

**Purpose**: Represents one repository command executed on behalf of a task run.

**Fields**:
- `command_line`: requested command string or argv form
- `working_directory`: canonical repository root or approved subdirectory
- `exit_code`: integer process result
- `stdout_summary`: truncated or summarized standard output
- `stderr_summary`: truncated or summarized standard error
- `duration_ms`: execution duration
- `status`: `succeeded`, `failed`, or `blocked`

**Validation rules**:
- Commands must execute from within the active repository boundary.
- Non-zero exit status must be preserved and surfaced in final reporting.

## Relationships Summary

- `AuthenticatedAccount` authorizes `AgentSession` records.
- `AgentSession` groups `TaskRun` history per repository and session identifier.
- `TaskRun` aggregates `WorkspaceChange` and `CommandResult` entries for reporting.
