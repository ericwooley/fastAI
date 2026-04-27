# Implementation Plan: Copilot CLI MVP

**Branch**: `001-copilot-cli-mvp` | **Date**: 2026-04-27 | **Spec**: `/home/ericwooley/fastAI/specs/001-copilot-cli-mvp/spec.md`
**Input**: Feature specification from `/home/ericwooley/fastAI/specs/001-copilot-cli-mvp/spec.md`

**Note**: This template is filled in by the `/speckit.plan` command. See `.specify/templates/plan-template.md` for the execution workflow.

## Summary

Build a Go CLI that logs into GitHub Copilot, requires an explicit `--model` flag for each
non-interactive agent run, and safely supports workspace-limited file editing, repository
command execution, and follow-up work via `--session=<identifier>`. The smallest viable
architecture uses Cobra for the CLI surface, `github.com/cli/oauth` for GitHub device-flow
login, repository-local orchestration around `google.golang.org/adk`, and a local GitHub
Models adapter plus filesystem-backed auth/session stores that keep core decision logic pure
and testable.

## Technical Context

**Language/Version**: Go 1.24.x  
**Primary Dependencies**: `github.com/spf13/cobra`, `github.com/cli/oauth`, `google.golang.org/adk`, Go standard library `net/http`/`os`/`os/exec`/`filepath`  
**Storage**: Local filesystem state under the user's OS config directory for auth/session metadata, plus temporary runtime work under repository-safe temp directories  
**Testing**: `go test ./...`, table-driven unit tests for pure logic, integration tests for auth/session/workspace adapters, CLI e2e tests with temporary repositories  
**Target Platform**: Linux, macOS, and Windows CLI environments
**Project Type**: Go CLI agent  
**Performance Goals**: First authenticated run can be started in under 2 minutes, subsequent validated CLI invocations complete local preflight checks in under 100 ms, and non-network startup stays under 2 seconds  
**Constraints**: Non-interactive by default, `--model` required for every run, file edits and commands restricted to the active repository root, GitHub Copilot-compatible authentication required before agent-backed runs, `github.com/google/adk-go` must remain the AI integration layer  
**Scale/Scope**: Single-user local developer workflow, one active repository per invocation, local persistence for dozens to hundreds of sessions, and task runs that may edit multiple files and execute multiple repository commands

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

- [x] Go CLI product authority: plan targets a Go CLI binary and not an interactive app
- [x] Autonomous CLI agent contract: command surface, non-interactive flow, `--session` semantics, and `github.com/google/adk-go` usage are defined
- [x] Verification-first Go architecture: pure logic, glue code, and CLI e2e coverage are planned with fast tests
- [x] Minimal explicit Go change: design uses the smallest viable approach and justifies every new dependency or abstraction
- [x] Integration parity and session continuity: this feature changes product code only and does not require shared workflow/template sync
- [x] Repo-scope compliance: all planned paths, tools, and edits remain inside this repository

Post-design review: all constitutional gates still pass after research and Phase 1 design.

## Project Structure

### Documentation (this feature)

```text
specs/001-copilot-cli-mvp/
├── plan.md              # This file (/speckit.plan command output)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)
```text
cmd/
└── fastAI/
    └── main.go

internal/
├── agent/
│   ├── run.go
│   ├── runner.go
│   └── githubmodels/
├── auth/
│   ├── deviceflow.go
│   └── store.go
├── cli/
│   ├── root.go
│   ├── login.go
│   └── output.go
├── commandexec/
│   └── executor.go
├── session/
│   ├── service.go
│   ├── store.go
│   └── ids.go
└── workspace/
    ├── editor.go
    ├── paths.go
    └── summary.go

test/
├── integration/
└── e2e/
```

**Structure Decision**: `cmd/fastAI/main.go` wires the Cobra command tree and delegates into
`internal/cli`. Pure validation and safety logic lives in `internal/session`, `internal/workspace`,
and any small helper packages; orchestration around ADK runs, login, and command execution lives
in `internal/agent`, `internal/auth`, and `internal/commandexec`. Auth tokens and session records
are stored outside the repository under the user's OS config directory, keyed by repository root
so follow-up work stays isolated per workspace while repository edits and commands remain confined
to the active repo.

## Complexity Tracking

> **Fill ONLY if Constitution Check has violations that must be justified**

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| [e.g., 4th project] | [current need] | [why 3 projects insufficient] |
| [e.g., Repository pattern] | [specific problem] | [why direct DB access insufficient] |
