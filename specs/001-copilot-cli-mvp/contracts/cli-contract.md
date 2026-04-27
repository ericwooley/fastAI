# CLI Contract: Copilot CLI MVP

## Command Surface

## `fastAI login`

Starts GitHub device-flow authentication and persists a reusable Copilot-compatible token for
later non-interactive runs.

**Usage**:

```text
fastAI login
```

**Behavior**:
- Prints the GitHub device-flow URL and user code.
- Waits for authorization completion.
- Stores authenticated account metadata and token reference locally.
- Returns success without starting an agent run.

## `fastAI [flags] <prompt>`

Starts one autonomous agent run against the active repository.

**Usage**:

```text
fastAI --model <model> [--session <identifier>] <prompt>
```

**Required flags**:
- `--model <model>`: required on every run, including resumed sessions.

**Optional flags**:
- `--session <identifier>`: resumes or continues work for an existing session when valid.

**Arguments**:
- `<prompt>`: required free-form task instruction.

## Validation Rules

- Missing `<prompt>` returns a validation error.
- Missing `--model` returns a validation error.
- A run without valid GitHub authentication returns an authentication error and points users to
  `fastAI login`.
- A provided `--session` must exist, be readable, and belong to the current repository.
- Requested file edits and commands must stay within the active repository safety boundary.

## Exit Status Contract

- `0`: run completed successfully
- `1`: agent execution failed after a valid start
- `2`: CLI validation failed, including missing prompt, missing `--model`, or invalid session
- `3`: authentication missing, expired, or rejected
- `4`: blocked unsafe operation, including repo-boundary violations for file or command work

## Output Contract

### Success output

- Short execution summary
- Session identifier used or created
- Model used for the run
- File change summary, if any
- Command result summary, if any

### Failure output

- Clear one-line failure classification
- Relevant validation, auth, safety, or execution message
- Session identifier when available
- Recommended recovery action when the failure is user-correctable

## ADK Integration Boundary

- CLI code passes validated prompt, model, repository capabilities, and optional session state
  into a repository-local agent runner interface.
- The agent runner uses `google.golang.org/adk` for orchestration.
- GitHub-backed model calls are performed through a repository-local adapter that converts the
  selected `--model` and authenticated token into ADK-compatible model execution.
- File editing and command execution are exposed to the runner as repository-safe tools only.
