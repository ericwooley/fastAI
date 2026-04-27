# Quickstart: Copilot CLI MVP

## Prerequisites

- Go 1.24.x installed
- A GitHub account with GitHub Copilot access
- Network access for GitHub device-flow login and model requests
- A local repository workspace where `fastAI` is invoked

## Build

```bash
go build ./cmd/fastAI
```

## Login

Authenticate once before starting agent-backed runs.

```bash
./fastAI login
```

Expected flow:
- The CLI prints a GitHub device-flow URL and code.
- You complete authorization in the browser.
- The CLI stores local auth state for later runs.

## Start a New Run

```bash
./fastAI --model github:gpt-4.1 "Refactor the CLI entrypoint and add tests"
```

Expected behavior:
- The run fails fast if `--model` or the prompt is missing.
- The run uses non-interactive execution only.
- Agent prompts are routed through the repository-local GitHub Copilot adapter and `google.golang.org/adk`.
- File edits and commands stay within the active repository boundary.

## Continue a Prior Session

```bash
./fastAI --model github:gpt-4.1 --session refactor-cli "Now add command execution coverage"
```

Expected behavior:
- The CLI resumes the stored session only if it belongs to the current repository.
- The run reports the reused session identifier and final outcome.

## Verification

Run the full automated test suite:

```bash
go test ./...
```

Recommended focused verification during implementation:

```bash
go test ./internal/...
go test ./test/integration/...
go test ./test/e2e/...
```

Latest verification:
- `go test ./...` passes with unit, integration, and CLI e2e coverage for login, required model validation, ADK-backed Copilot adapter behavior, safe file editing, command execution, and `--session` resume/error flows.
