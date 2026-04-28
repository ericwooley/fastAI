# fastAI

A non-interactive, autonomous CLI coding agent. Invoke it from your terminal inside any Git repository to delegate coding tasks—file edits, command execution, and session-persistent follow-up work.

## Install

### Option A — install with make (recommended)

```bash
git clone https://github.com/ericwooley/fastAI.git
cd fastAI
make build
make install
```

This builds the binary and places it into your OS-appropriate user bin directory:

| OS      | Default install dir               |
|---------|-----------------------------------|
| Linux   | `~/.local/bin`                    |
| macOS   | `/usr/local/bin` (fallback: `~/.local/bin`) |
| Windows | `%USERPROFILE%/bin` (fallback: `~/bin`)     |

Override the destination with `FASTAI_INSTALL_DIR`:

```bash
FASTAI_INSTALL_DIR="$HOME/bin" make install
```

### Option B — build and move the binary manually

```bash
git clone https://github.com/ericwooley/fastAI.git
cd fastAI
make build
```

The compiled binary is at `tmp/bin/fastAI`. Move it anywhere on your `PATH`:

```bash
# Linux / macOS
sudo mv tmp/bin/fastAI /usr/local/bin/

# Or to your user-local bin
mv tmp/bin/fastAI ~/.local/bin/

# Windows (Git Bash / WSL)
mv tmp/bin/fastAI ~/bin/
```

### Option C — go install

```bash
go install github.com/ericwooley/fastAI/cmd/fastAI@latest
```

Requires Go 1.24.x. The binary is placed in `$GOPATH/bin` or `~/go/bin`.

---

## Prerequisites

- **Go 1.24.x**
- A GitHub account with **GitHub Copilot** access
- Run from inside a Git repository

## Dev Quickstart

```bash
make help                     # Show available targets
make build                    # Build tmp/bin/fastAI
make install                  # Install fastAI into your user bin directory
make test                     # Run all tests
make check                    # Format, vet, and test
make run ARGS='--model github:gpt-4.1 "Refactor the auth module and add tests"'
```

Common focused test scopes:

```bash
make test-unit
make test-integration
make test-e2e
```

Authenticate the local build when you need real Copilot-backed runs:

```bash
make login
```

## Features

- **GitHub Copilot authentication** via OAuth device flow (`fastAI login`)
- **Autonomous, non-interactive execution** — no prompts during a run
- **File operations** — create, update, patch, and delete files within the repo boundary
- **Command execution** — run shell commands scoped to the repository
- **Session persistence** — resume prior work with `--session <id>`
- **Required model flag** — every run must specify `--model` (e.g., `github:gpt-4.1`)
- **Repository safety** — all file and command operations are confined to the repo root

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
make build                 # Build tmp/bin/fastAI
make test                  # Run all tests (unit + integration + e2e)
make vet                   # Static analysis
make check                 # Format, vet, and test
```

Test scopes:

```bash
make test-unit             # Unit tests (pure logic)
make test-integration      # Integration tests (glue code)
make test-e2e              # End-to-end CLI tests
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
