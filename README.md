# fastAI

A non-interactive, autonomous CLI coding agent backed by GitHub Copilot. Invoke it from your terminal inside any Git repository to delegate coding tasks—file edits, command execution, and session-persistent follow-up work.

## Features

- **GitHub Copilot authentication** via OAuth device flow (`fastAI login`)
- **Autonomous, non-interactive execution** — no prompts during a run
- **File operations** — create, update, patch, and delete files within the repo boundary
- **Command execution** — run shell commands scoped to the repository
- **Session persistence** — resume prior work with `--session <id>`
- **Required model flag** — every run must specify `--model` (e.g., `github:gpt-4.1`)
- **Repository safety** — all file and command operations are confined to the repo root

## Prerequisites

- **Go 1.24.x**
- A GitHub account with **GitHub Copilot** access
- Run from inside a Git repository

## Install

```bash
go install github.com/ericwooley/fastAI/cmd/fastAI@latest
```

Or build from source:

```bash
git clone https://github.com/ericwooley/fastAI.git
cd fastAI
go build ./cmd/fastAI
```

## Usage

### Login (one-time setup)

```bash
fastAI login
```

Opens a browser for GitHub device-flow authentication. Credentials are stored locally.

### Run a task

```bash
fastAI --model github:gpt-4.1 "Refactor the auth module and add tests"
```

`--model` is required on every invocation. Without it, the CLI exits with a validation error.

### Continue a prior session

```bash
fastAI --model github:gpt-4.1 --session my-task "Now add error handling"
```

## Exit Codes

| Code | Meaning                     |
|------|-----------------------------|
| 0    | Success                     |
| 1    | Agent execution failed      |
| 2    | CLI validation failed       |
| 3    | Authentication required/failed |
| 4    | Unsafe operation blocked    |

## Build & Test

```bash
go build ./cmd/fastAI      # Build the binary
go test ./...              # Run all tests (unit + integration + e2e)
go vet ./...               # Static analysis
```

Test scopes:

```bash
go test ./internal/...     # Unit tests (pure logic)
go test ./test/integration/...  # Integration tests (glue code)
go test ./test/e2e/...     # End-to-end CLI tests
```

## Project Structure

```
cmd/fastAI/          CLI entrypoint
internal/
  agent/             Agent runner, ADK tools, GitHub Copilot adapter
  auth/              OAuth device flow, local token storage
  cli/               Cobra commands, validation, output formatting
  commandexec/       Repo-scoped shell command execution
  session/           Session lifecycle, persistence, ID management
  workspace/         File operations, repo root discovery, path safety
specs/               Feature specifications and planning
test/
  integration/       Integration tests
  e2e/               End-to-end CLI tests
```

## Architecture

fastAI follows a three-layer testing pyramid:

1. **Pure functions** (~85% of tests) — deterministic, no side effects. Validation, path safety, session IDs, result classification.
2. **Glue code** (~10% of tests) — composes pure functions. Login state, session persistence, file-edit orchestration, agent backend boundaries.
3. **E2E** (~5% of tests) — full CLI flows with dependency injection and fake backends.

The agent runner uses [Google ADK](https://github.com/google/adk-go) to manage LLM interactions, with a custom adapter that translates between the ADK's content model and GitHub Copilot's chat completion API.
