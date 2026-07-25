# fastAI

A non-interactive, autonomous CLI coding agent. Invoke it from your terminal inside any Git repository to delegate coding tasks—file edits, command execution, and session-persistent follow-up work.

## Install

### Homebrew (recommended)

Install the release binary directly from the `ericwooley/apps` tap:

```bash
brew install ericwooley/apps/fastai
fastAI --version
```

The fully qualified install command adds the tap automatically. Upgrade later releases with:

```bash
brew upgrade ericwooley/apps/fastai
```

### Build and install with make

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

### Build and move the binary manually

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

### Go install

```bash
go install github.com/ericwooley/fastAI/cmd/fastAI@latest
```

Requires Go 1.25.x. The binary is placed in `$GOPATH/bin` or `~/go/bin`.

---

## Prerequisites

- Run from inside a Git repository
- One configured AI provider: GitHub Copilot login or a supported provider API key
- Go 1.25.x when building or installing from source

## Dev Quickstart

```bash
make help                     # Show available targets
make build                    # Build tmp/bin/fastAI
make install                  # Install fastAI into your user bin directory
make install-hooks            # Install pre-commit and commit-msg checks
make test                     # Run all tests
make check                    # Verify formatting, vet, and run all tests
make run ARGS='--provider github-copilot --model github:gpt-4.1 --permissions all "Refactor the auth module and add tests"'
```

Common focused test scopes:

```bash
make test-unit
make test-integration
make test-e2e
```

Authenticate the local build when you need GitHub Copilot-backed runs:

```bash
make login
```

## Features

- **Explicit provider selection** via `--provider <provider>` or `FASTAI_DEFAULT_PROVIDER`
- **GitHub Copilot login** via OAuth device flow (`fastAI login copilot`)
- **API-key providers** via environment variables for OpenAI-compatible endpoints
- **Autonomous, non-interactive execution** — no prompts during a run
- **File operations** — create, update, patch, and delete files within the repo boundary
- **Command execution** — run shell commands scoped to the repository
- **Session persistence** — resume prior work as a conversation with `--session <id>`
- **Global session** — reuse or reset a repository-wide conversation with `--globalSession` and `--newGlobalSession`
- **Session history** — inspect recent saved chat inputs and outputs with `--history`
- **Hierarchical repository instructions** — load `AGENTS.md` files from the repository root through the current working directory
- **Model selection** via `--model <model>` or `FASTAI_DEFAULT_MODEL`
- **Tool permissions** via `--permissions <list>` or `FASTAI_DEFAULT_PERMISSIONS`
- **Repository safety** — all file and command operations are confined to the repo root

## Usage

### Providers And Credentials

| Provider ID | Name | Credential | Example model usage |
|-------------|------|------------|---------------------|
| `github-copilot` | GitHub Copilot | `fastAI login copilot` OAuth device flow. No API key is required for normal CLI runs. | `--provider github-copilot --model github:gpt-4.1` |
| `openai` | OpenAI | Set `OPENAI_API_KEY` | `--provider openai --model gpt-4.1` |
| `openrouter` | OpenRouter | Set `OPENROUTER_API_KEY` | `--provider openrouter --model deepseek/deepseek-chat` |
| `deepseek` | DeepSeek | Set `DEEPSEEK_API_KEY` | `--provider deepseek --model deepseek-chat` |

For non-Copilot providers, export the key before running fastAI:

```bash
export OPENAI_API_KEY="..."
export OPENROUTER_API_KEY="..."
export DEEPSEEK_API_KEY="..."
```

You can also set CLI defaults with environment variables:

```bash
export FASTAI_DEFAULT_PROVIDER="github-copilot"
export FASTAI_DEFAULT_MODEL="github:gpt-5-mini"
export FASTAI_DEFAULT_PERMISSIONS="all"
```

Flag values override their matching `FASTAI_DEFAULT_*` values.

### Login (one-time Copilot setup)

```bash
fastAI login copilot
```

Opens a browser for GitHub device-flow authentication. Copilot credentials are stored locally.

### Run a task

```bash
fastAI --provider github-copilot --model github:gpt-4.1 --permissions all "Refactor the auth module and add tests"
```

If you omit the prompt or pass an empty prompt, fastAI opens `$VISUAL`, then `$EDITOR`, then `vi`
so you can write the request in a temporary file. Saving and closing the editor submits that text.

Each run must resolve `provider`, `model`, and `permissions` from either flags or matching defaults:

- `--provider` or `FASTAI_DEFAULT_PROVIDER`
- `--model` or `FASTAI_DEFAULT_MODEL`
- `--permissions` or `FASTAI_DEFAULT_PERMISSIONS`

If any of those are missing after resolution, the CLI exits with a validation error.

Before sending a request, fastAI loads each existing `AGENTS.md` along the path from the repository
root to the current working directory. It appends those files to the system instruction in
root-to-leaf order. When instructions conflict, a later file closer to the current working directory
takes priority.

### Permissions

Use `--permissions` as a comma-separated list of allowed tool groups:

- `all`: allow read, write, and execute
- `read`: allow file reads only
- `write`: allow file writes only
- `execute`: allow shell commands only
- `read,write`: allow reads and writes but not command execution
- `none`: disable all tools

`all` and `none` must be used alone.

### Continue a prior session

```bash
fastAI --provider github-copilot --model github:gpt-4.1 --permissions all --session my-task "Now add error handling"
```

Named sessions include prior prompts and agent summaries in follow-up model requests. The raw
session history remains on disk so the agent can grep older details by timestamp or run id when
needed.

### Continue the global session

```bash
fastAI --provider github-copilot --model github:gpt-4.1 --permissions all --globalSession "Keep going"
fastAI --provider github-copilot --model github:gpt-4.1 --permissions all --newGlobalSession "Start fresh"
```

`--globalSession` uses a repository-wide session named `global`. `--newGlobalSession` deletes that
stored conversation before running the new prompt.

### Inspect session history

```bash
fastAI --history
fastAI --history 20
fastAI --session my-task --history 10
```

`--history` prints saved chat inputs and outputs for the selected session without starting an agent
run. It defaults to the repository global session and shows the last 5 conversations unless you pass
a count.

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
make check                 # Formatting, vet, tests, and release tooling
```

Test scopes:

```bash
make test-unit             # Unit tests (pure logic)
make test-integration      # Integration tests (glue code)
make test-e2e              # End-to-end CLI tests
```

## Conventional Commits

Install the repository hooks once after cloning:

```bash
make install-hooks
```

The `pre-commit` hook runs the quick formatting, internal-unit-test, and staged-whitespace checks. The `commit-msg` hook enforces [Conventional Commits](https://www.conventionalcommits.org/) because the commit message is not available during `pre-commit`. CI runs the complete suite.

Use `fix:` or `perf:` for a patch release, `feat:` for a minor release, and add `!` after the type or a `BREAKING CHANGE:` footer for a major release. Scoped forms such as `feat(cli): ...` are supported. The other allowed types (`build`, `chore`, `ci`, `docs`, `refactor`, `revert`, `style`, and `test`) do not publish a release by themselves. CI applies the same policy to every non-merge commit in a pull request or push.

## Releases

Every push to `main` plans a release from the Conventional Commits since the latest Go-compatible `vX.Y.Z` tag. When a release is required, GitHub Actions:

1. runs the complete validation suite;
2. builds versioned release archives for macOS and Linux on ARM64 and AMD64, plus Windows AMD64;
3. publishes the plain semantic-version tag and GitHub release with generated notes and SHA-256 checksums;
4. renders, installs, and tests `Formula/fastai.rb`, then updates `ericwooley/homebrew-apps`.

Publishing GitHub releases and updating Homebrew run in the `release` environment. In `ericwooley/fastAI`, create that environment under **Settings → Environments**, then add `TAP_GITHUB_TOKEN` as an environment secret. Use a fine-grained personal access token limited to `ericwooley/homebrew-apps` with **Contents: Read and write**. If only the tap update needs to be retried, run **Update Homebrew** with the existing plain version such as `0.1.0`. To rebuild or repair a partial GitHub release, run **Release** with that existing plain version; the workflow rebuilds the tagged source, replaces the release assets, and retries Homebrew.

## Project Structure

```
cmd/fastAI/          CLI entrypoint
.github/workflows/   CI, semantic releases, and Homebrew publishing
internal/
  agent/             Agent runner, ADK tools, GitHub Copilot adapter
  auth/              OAuth device flow, local token storage
  cli/               Cobra commands, validation, output formatting
  commandexec/       Repo-scoped shell command execution
  session/           Session lifecycle, persistence, ID management
  workspace/         File operations, repo root discovery, path safety
packaging/homebrew/  Generated formula template
scripts/             Local install, hooks, and release tooling
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
