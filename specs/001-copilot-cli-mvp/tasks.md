# Tasks: Copilot CLI MVP

**Input**: Design documents from `/specs/001-copilot-cli-mvp/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Verification**: Include the smallest required verification tasks for each story. Use fast Go tests first, with coverage biased toward pure functions, then integration work, then CLI end-to-end validation. Add explicit manual validation only when automation is not enough.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Go CLI project**: `cmd/`, `internal/`, `test/` at repository root

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Initialize the Go CLI project and create the file layout from the implementation plan.

- [ ] T001 Initialize the Go module and required dependencies in `go.mod` and `go.sum`
- [ ] T002 Create the CLI bootstrap entrypoints in `cmd/fastAI/main.go` and `internal/cli/root.go`
- [ ] T003 Create the initial package scaffolding in `internal/auth/deviceflow.go`, `internal/agent/run.go`, `internal/session/service.go`, `internal/workspace/editor.go`, and `internal/commandexec/executor.go`

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before any user story can be implemented.

**⚠️ CRITICAL**: No user story work can begin until this phase is complete.

- [ ] T004 Implement repository-root and config-directory path resolution in `internal/workspace/paths.go` and `internal/session/store.go`
- [ ] T005 [P] Implement shared prompt/model/session validation helpers in `internal/cli/validate.go`
- [ ] T006 [P] Implement exit-code mapping and shared CLI result formatting in `internal/cli/output.go`
- [ ] T007 [P] Define file-backed authentication and session store interfaces in `internal/auth/store.go` and `internal/session/store.go`
- [ ] T008 [P] Define the ADK runner interface and GitHub Models adapter seam in `internal/agent/runner.go` and `internal/agent/githubmodels/adapter.go`
- [ ] T009 [P] Define shared session, run, and repository-key types in `internal/session/ids.go` and `internal/agent/run.go`
- [ ] T010 Implement reusable test doubles and temporary repository helpers in `test/integration/testutil_test.go` and `test/e2e/testutil_test.go`

**Checkpoint**: Foundation ready - authenticated runs, workspace safety, session persistence, and ADK boundaries can now be implemented story by story.

---

## Phase 3: User Story 1 - Start Authenticated Agent Run (Priority: P1) 🎯 MVP

**Goal**: Let a developer log in with GitHub Copilot and start a non-interactive agent run with a required `--model` flag.

**Independent Test**: A user completes `fastAI login`, then runs `fastAI --model <model> "task"` and receives a clear success or failure result without any interactive follow-up.

### Verification for User Story 1 ⚠️

- [ ] T011 [P] [US1] Add table-driven validation tests for missing prompt, missing `--model`, and invalid input handling in `internal/cli/validate_test.go`
- [ ] T012 [P] [US1] Add unit tests for persisted account loading and token-state checks in `internal/auth/store_test.go`
- [ ] T013 [P] [US1] Add integration coverage for device-flow login and authenticated run orchestration in `test/integration/login_run_test.go`
- [ ] T014 [US1] Add CLI e2e coverage for `fastAI login` and a successful `fastAI --model ...` run in `test/e2e/login_run_test.go`

### Implementation for User Story 1

- [ ] T015 [US1] Implement GitHub device-flow login and token persistence in `internal/auth/deviceflow.go`
- [ ] T016 [P] [US1] Implement authenticated account loading and login command wiring in `internal/cli/login.go`
- [ ] T017 [P] [US1] Implement baseline autonomous run orchestration and default session creation in `internal/agent/run.go`
- [ ] T018 [US1] Implement the GitHub Models-backed ADK adapter for required model selection in `internal/agent/githubmodels/adapter.go`
- [ ] T019 [US1] Wire root command execution and dependency construction in `internal/cli/root.go` and `cmd/fastAI/main.go`

**Checkpoint**: User Story 1 MUST be fully functional and independently verified.

---

## Phase 4: User Story 2 - Apply File Changes (Priority: P2)

**Goal**: Let the autonomous agent create, update, and delete files inside the active repository while blocking unsafe paths.

**Independent Test**: A signed-in user runs a file-editing task, sees in-repo changes applied, and gets a clear failure when the agent attempts an out-of-repo path.

### Verification for User Story 2 ⚠️

- [ ] T020 [P] [US2] Add table-driven tests for repository-bound path normalization and traversal blocking in `internal/workspace/paths_test.go`
- [ ] T021 [P] [US2] Add unit tests for file create, update, delete, and summary behavior in `internal/workspace/editor_test.go` and `internal/workspace/summary_test.go`
- [ ] T022 [P] [US2] Add integration coverage for workspace editing tool orchestration in `test/integration/file_edit_run_test.go`
- [ ] T023 [US2] Add CLI e2e coverage for in-repo edits and blocked out-of-repo file requests in `test/e2e/file_edit_run_test.go`

### Implementation for User Story 2

- [ ] T024 [P] [US2] Implement repository-safe file editing operations in `internal/workspace/editor.go`
- [ ] T025 [P] [US2] Implement workspace change summaries for final CLI output in `internal/workspace/summary.go`
- [ ] T026 [US2] Integrate workspace editing tools into the ADK run pipeline in `internal/agent/run.go`
- [ ] T027 [US2] Surface file-change summaries and workspace safety failures in `internal/cli/output.go`

**Checkpoint**: User Stories 1 and 2 MUST both work and be independently verified.

---

## Phase 5: User Story 3 - Run Repository Commands (Priority: P3)

**Goal**: Let the agent run repository commands and continue prior work with `--session=<identifier>`.

**Independent Test**: A signed-in user runs a task that executes repository commands, then starts a follow-up run with the same `--session` identifier and receives resumed behavior tied to that earlier session.

### Verification for User Story 3 ⚠️

- [ ] T028 [P] [US3] Add table-driven tests for session ID validation, repository-key matching, and resume rules in `internal/session/ids_test.go` and `internal/session/service_test.go`
- [ ] T029 [P] [US3] Add unit tests for command execution result classification and non-zero exit handling in `internal/commandexec/executor_test.go`
- [ ] T030 [P] [US3] Add integration coverage for resumed-session command execution flow in `test/integration/session_command_run_test.go`
- [ ] T031 [US3] Add CLI e2e coverage for `--session` continuation and command failure reporting in `test/e2e/session_command_run_test.go`

### Implementation for User Story 3

- [ ] T032 [P] [US3] Implement session lifecycle management and persisted run history in `internal/session/service.go`
- [ ] T033 [P] [US3] Implement repository-bound command execution with captured stdout, stderr, and exit status in `internal/commandexec/executor.go`
- [ ] T034 [US3] Integrate session resumption and command execution tools into `internal/agent/run.go`
- [ ] T035 [US3] Implement `--session` handling and command failure reporting in `internal/cli/root.go` and `internal/cli/output.go`

**Checkpoint**: All user stories MUST now be independently functional and verified.

---

## Phase 6: Polish & Cross-Cutting Concerns

**Purpose**: Final validation and improvements that affect multiple user stories.

- [ ] T036 [P] Document login, required `--model`, and resumed-session usage in `specs/001-copilot-cli-mvp/quickstart.md`
- [ ] T037 [P] Add automated coverage for ADK adapter failure handling in `test/integration/adk_adapter_test.go`
- [ ] T038 Run `go test ./...` and record verification notes in `specs/001-copilot-cli-mvp/quickstart.md`

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately.
- **Foundational (Phase 2)**: Depends on Setup completion - blocks all user stories.
- **User Story 1 (Phase 3)**: Depends on Foundational completion.
- **User Story 2 (Phase 4)**: Depends on Foundational completion and the authenticated run path from User Story 1.
- **User Story 3 (Phase 5)**: Depends on Foundational completion and the authenticated run path from User Story 1.
- **Polish (Phase 6)**: Depends on all desired user stories being complete.

### User Story Dependencies

- **User Story 1 (P1)**: First deliverable and MVP slice.
- **User Story 2 (P2)**: Builds on the authenticated execution path from User Story 1 but remains independently verifiable once implemented.
- **User Story 3 (P3)**: Builds on the authenticated execution path from User Story 1 and adds session continuation plus command execution.

### Within Each User Story

- Verification tasks MUST be defined before implementation.
- When behavior changes, automated tests or equivalent checks MUST fail before implementation.
- Pure logic before orchestration.
- Orchestration before CLI wiring.
- Story completion before moving to the next dependent slice.

### Parallel Opportunities

- Setup tasks are mostly sequential because they create the base layout.
- In Phase 2, T005, T006, T007, T008, and T009 can proceed in parallel after T004 starts the shared path model.
- In User Story 1, T011, T012, and T013 can proceed in parallel; T016 and T017 can proceed in parallel after verification tasks are in place.
- In User Story 2, T020, T021, and T022 can proceed in parallel; T024 and T025 can proceed in parallel before integration.
- In User Story 3, T028, T029, and T030 can proceed in parallel; T032 and T033 can proceed in parallel before final wiring.

---

## Parallel Example: User Story 1

```bash
# Launch US1 verification work together:
Task: "Add table-driven validation tests in internal/cli/validate_test.go"
Task: "Add unit tests for persisted account loading in internal/auth/store_test.go"
Task: "Add integration coverage for device-flow login in test/integration/login_run_test.go"

# Launch US1 implementation work together after verification is defined:
Task: "Implement authenticated account loading and login command wiring in internal/cli/login.go"
Task: "Implement baseline autonomous run orchestration in internal/agent/run.go"
```

## Parallel Example: User Story 2

```bash
# Launch US2 verification work together:
Task: "Add path normalization tests in internal/workspace/paths_test.go"
Task: "Add file-edit behavior tests in internal/workspace/editor_test.go and internal/workspace/summary_test.go"
Task: "Add workspace editing integration coverage in test/integration/file_edit_run_test.go"

# Launch US2 pure implementation work together:
Task: "Implement repository-safe file editing in internal/workspace/editor.go"
Task: "Implement workspace change summaries in internal/workspace/summary.go"
```

## Parallel Example: User Story 3

```bash
# Launch US3 verification work together:
Task: "Add session resume tests in internal/session/ids_test.go and internal/session/service_test.go"
Task: "Add command execution unit tests in internal/commandexec/executor_test.go"
Task: "Add resumed-session integration coverage in test/integration/session_command_run_test.go"

# Launch US3 pure implementation work together:
Task: "Implement session lifecycle management in internal/session/service.go"
Task: "Implement repository-bound command execution in internal/commandexec/executor.go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup.
2. Complete Phase 2: Foundational.
3. Complete Phase 3: User Story 1.
4. Stop and validate `fastAI login` plus `fastAI --model <model> "task"` independently.

### Incremental Delivery

1. Complete Setup + Foundational so the CLI skeleton, safety boundaries, and ADK seams are ready.
2. Deliver User Story 1 as the MVP authenticated run path.
3. Add User Story 2 for safe file editing inside the repository.
4. Add User Story 3 for command execution and resumed sessions.
5. Finish with cross-cutting validation and documentation.

### Parallel Team Strategy

1. One developer completes Setup + Foundational.
2. After User Story 1 establishes the authenticated run path:
   - Developer A extends the agent with workspace editing for User Story 2.
   - Developer B extends the agent with command execution and session continuation for User Story 3.
3. Rejoin for Polish and full-suite verification.

---

## Notes

- [P] tasks = different files, no dependencies.
- [Story] labels map each task to a specific user story for traceability.
- Each user story MUST be independently completable and verifiable.
- Verification tasks come before implementation tasks in every user story phase.
- This plan uses User Story 1 as the recommended MVP scope.
