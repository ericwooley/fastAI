---

description: "Task list template for feature implementation"
---

# Tasks: [FEATURE NAME]

**Input**: Design documents from `/specs/[###-feature-name]/`
**Prerequisites**: plan.md (required), spec.md (required for user stories), research.md, data-model.md, contracts/

**Verification**: Include the smallest required verification tasks for each story. Use
fast Go tests first, with coverage biased toward pure functions, then integration work,
then CLI end-to-end validation. Add explicit manual validation only when automation is
not enough.

**Organization**: Tasks are grouped by user story to enable independent implementation and testing of each story.

## Format: `[ID] [P?] [Story] Description`

- **[P]**: Can run in parallel (different files, no dependencies)
- **[Story]**: Which user story this task belongs to (e.g., US1, US2, US3)
- Include exact file paths in descriptions

## Path Conventions

- **Go CLI project**: `cmd/`, `internal/`, `pkg/`, `test/` at repository root
- **Web app**: `backend/src/`, `frontend/src/`
- **Mobile**: `api/src/`, `ios/src/` or `android/src/`
- Paths shown below assume single project - adjust based on plan.md structure

<!-- 
  ============================================================================
  IMPORTANT: The tasks below are SAMPLE TASKS for illustration purposes only.
  
  The /speckit.tasks command MUST replace these with actual tasks based on:
  - User stories from spec.md (with their priorities P1, P2, P3...)
  - Feature requirements from plan.md
  - Entities from data-model.md
  - Endpoints from contracts/
  
  Tasks MUST be organized by user story so each story can be:
  - Implemented independently
  - Tested independently
  - Delivered as an MVP increment
  
  DO NOT keep these sample tasks in the generated tasks.md file.
  ============================================================================
-->

## Phase 1: Setup (Shared Infrastructure)

**Purpose**: Project initialization and basic structure

- [ ] T001 Create project structure per implementation plan
- [ ] T002 Initialize Go module, CLI entrypoint, and required dependencies
- [ ] T003 [P] Configure linting and formatting tools

---

## Phase 2: Foundational (Blocking Prerequisites)

**Purpose**: Core infrastructure that MUST be complete before ANY user story can be implemented

**⚠️ CRITICAL**: No user story work can begin until this phase is complete

Examples of foundational tasks (adjust based on your project):

- [ ] T004 Define shared CLI command parsing and config loading
- [ ] T005 [P] Implement session storage interface and persistence wiring
- [ ] T006 [P] Define agent runner interface and `github.com/google/adk-go` backend wiring
- [ ] T007 Create shared domain types and validation helpers
- [ ] T008 Configure error handling and structured output behavior
- [ ] T009 Setup environment and test fixtures for CLI execution

**Checkpoint**: Foundation ready - user story implementation can now begin in parallel

---

## Phase 3: User Story 1 - [Title] (Priority: P1) 🎯 MVP

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Verification for User Story 1 ⚠️

> **NOTE: Define verification FIRST. When behavior changes, show the failing or missing
> behavior before implementation.**

- [ ] T010 [P] [US1] Add table-driven unit tests for pure logic in internal/[domain]/[name]_test.go
- [ ] T011 [P] [US1] Add integration test for orchestration flow in test/integration/[name]_test.go
- [ ] T012 [US1] Add CLI e2e test for primary command behavior in test/e2e/[name]_test.go

### Implementation for User Story 1

- [ ] T013 [P] [US1] Implement pure domain logic in internal/[domain]/[name].go
- [ ] T014 [P] [US1] Implement supporting types or validators in internal/[domain]/[name].go
- [ ] T015 [US1] Implement orchestration layer in internal/agent/[name].go (depends on T013, T014)
- [ ] T016 [US1] Implement CLI command behavior in internal/cli/[name].go
- [ ] T017 [US1] Add validation and error handling for the CLI contract
- [ ] T018 [US1] Wire entrypoint in cmd/fastAI/main.go

**Checkpoint**: At this point, User Story 1 MUST be fully functional and independently verified

---

## Phase 4: User Story 2 - [Title] (Priority: P2)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Verification for User Story 2 ⚠️

- [ ] T019 [P] [US2] Add table-driven unit tests for session behavior in internal/session/[name]_test.go
- [ ] T020 [P] [US2] Add integration test for resumed execution in test/integration/[name]_test.go
- [ ] T021 [US2] Add CLI e2e test for `--session` continuation in test/e2e/[name]_test.go

### Implementation for User Story 2

- [ ] T022 [P] [US2] Implement session model and persistence logic in internal/session/[name].go
- [ ] T023 [US2] Implement follow-up orchestration in internal/agent/[name].go
- [ ] T024 [US2] Implement CLI `--session` handling in internal/cli/[name].go
- [ ] T025 [US2] Integrate resumed-session flow with User Story 1 behavior and `github.com/google/adk-go`

**Checkpoint**: At this point, User Stories 1 and 2 MUST both work and be independently verified

---

## Phase 5: User Story 3 - [Title] (Priority: P3)

**Goal**: [Brief description of what this story delivers]

**Independent Test**: [How to verify this story works on its own]

### Verification for User Story 3 ⚠️

- [ ] T026 [P] [US3] Add table-driven unit tests for failure formatting in internal/[domain]/[name]_test.go
- [ ] T027 [P] [US3] Add integration test for autonomous failure handling in test/integration/[name]_test.go
- [ ] T028 [US3] Add CLI e2e test for non-interactive failure behavior in test/e2e/[name]_test.go

### Implementation for User Story 3

- [ ] T029 [P] [US3] Implement failure classification logic in internal/[domain]/[name].go
- [ ] T030 [US3] Implement autonomous error reporting in internal/agent/[name].go
- [ ] T031 [US3] Implement stderr and exit-code handling in internal/cli/[name].go

**Checkpoint**: All user stories MUST now be independently functional and verified

---

[Add more user story phases as needed, following the same pattern]

---

## Phase N: Polish & Cross-Cutting Concerns

**Purpose**: Improvements that affect multiple user stories

- [ ] TXXX [P] Documentation updates for CLI usage and session behavior in docs/
- [ ] TXXX Code cleanup and refactoring
- [ ] TXXX Performance optimization across all stories
- [ ] TXXX [P] Additional unit tests in internal/.../*_test.go
- [ ] TXXX Security hardening
- [ ] TXXX [P] Verify `github.com/google/adk-go` integration behavior in automated tests
- [ ] TXXX Run quickstart.md validation

---

## Dependencies & Execution Order

### Phase Dependencies

- **Setup (Phase 1)**: No dependencies - can start immediately
- **Foundational (Phase 2)**: Depends on Setup completion - BLOCKS all user stories
- **User Stories (Phase 3+)**: All depend on Foundational phase completion
  - User stories can then proceed in parallel (if staffed)
  - Or sequentially in priority order (P1 → P2 → P3)
- **Polish (Final Phase)**: Depends on all desired user stories being complete

### User Story Dependencies

- **User Story 1 (P1)**: Can start after Foundational (Phase 2) - No dependencies on other stories
- **User Story 2 (P2)**: Can start after Foundational (Phase 2) - May integrate with US1 but MUST remain independently verifiable
- **User Story 3 (P3)**: Can start after Foundational (Phase 2) - May integrate with US1/US2 but MUST remain independently verifiable

### Within Each User Story

- Verification tasks MUST be defined before implementation
- When behavior changes, automated tests or equivalent checks MUST fail before implementation
- Pure logic before orchestration
- Orchestration before CLI wiring
- Core implementation before integration
- Story complete before moving to next priority

### Parallel Opportunities

- All Setup tasks marked [P] can run in parallel
- All Foundational tasks marked [P] can run in parallel (within Phase 2)
- Once Foundational phase completes, all user stories can start in parallel (if team capacity allows)
- All Go verification tasks for a user story marked [P] can run in parallel
- Pure-logic tasks within a story marked [P] can run in parallel
- Different user stories can be worked on in parallel by different team members

---

## Parallel Example: User Story 1

```bash
# Launch verification work for User Story 1 together:
Task: "Table-driven unit tests for pure logic in internal/[domain]/[name]_test.go"
Task: "Integration test for orchestration flow in test/integration/[name]_test.go"
Task: "CLI e2e test for primary command behavior in test/e2e/[name]_test.go"

# Launch all models for User Story 1 together:
Task: "Implement pure domain logic in internal/[domain]/[name].go"
Task: "Implement supporting types or validators in internal/[domain]/[name].go"
```

---

## Implementation Strategy

### MVP First (User Story 1 Only)

1. Complete Phase 1: Setup
2. Complete Phase 2: Foundational (CRITICAL - blocks all stories)
3. Complete Phase 3: User Story 1
4. **STOP and VALIDATE**: Test User Story 1 independently
5. Deploy/demo if ready

### Incremental Delivery

1. Complete Setup + Foundational → Foundation ready
2. Add User Story 1 → Test independently → Deploy/Demo (MVP!)
3. Add User Story 2 → Test independently → Deploy/Demo
4. Add User Story 3 → Test independently → Deploy/Demo
5. Each story adds value without breaking previous stories

### Parallel Team Strategy

With multiple developers:

1. Team completes Setup + Foundational together
2. Once Foundational is done:
   - Developer A: User Story 1
   - Developer B: User Story 2
   - Developer C: User Story 3
3. Stories complete and integrate independently

---

## Notes

- [P] tasks = different files, no dependencies
- [Story] label maps task to specific user story for traceability
- Each user story MUST be independently completable and verifiable
- Verify required checks fail, or otherwise demonstrate the missing behavior, before implementing
- Commit after each task or logical group
- Stop at any checkpoint to validate story independently
- Avoid: vague tasks, same file conflicts, cross-story dependencies that break independence
