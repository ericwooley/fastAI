# Feature Specification: [FEATURE NAME]

**Feature Branch**: `[###-feature-name]`  
**Created**: [DATE]  
**Status**: Draft  
**Input**: User description: "$ARGUMENTS"

## User Scenarios & Testing *(mandatory)*

<!--
  IMPORTANT: User stories should be PRIORITIZED as user journeys ordered by importance.
  Each user story/journey must be INDEPENDENTLY TESTABLE - meaning if you implement just ONE of them,
  you should still have a viable MVP (Minimum Viable Product) that delivers value.
  
  Assign priorities (P1, P2, P3, etc.) to each story, where P1 is the most critical.
  Think of each story as a standalone slice of functionality that can be:
  - Developed independently
  - Tested independently
  - Deployed independently
  - Demonstrated to users independently
  - Verified with an explicit automated or manual validation method
  - Defined in terms of CLI inputs, outputs, exit codes, and persisted session behavior when applicable
-->

### User Story 1 - [Brief Title] (Priority: P1)

[Describe this user journey in plain language]

**Why this priority**: [Explain the value and why it has this priority level]

**Independent Test**: [Describe how this can be tested independently - e.g., "Can be fully tested by [specific action] and delivers [specific value]"]

**Acceptance Scenarios**:

1. **Given** [initial state], **When** [action], **Then** [expected outcome]
2. **Given** [initial state], **When** [action], **Then** [expected outcome]

---

### User Story 2 - [Brief Title] (Priority: P2)

[Describe this user journey in plain language]

**Why this priority**: [Explain the value and why it has this priority level]

**Independent Test**: [Describe how this can be tested independently]

**Acceptance Scenarios**:

1. **Given** [initial state], **When** [action], **Then** [expected outcome]

---

### User Story 3 - [Brief Title] (Priority: P3)

[Describe this user journey in plain language]

**Why this priority**: [Explain the value and why it has this priority level]

**Independent Test**: [Describe how this can be tested independently]

**Acceptance Scenarios**:

1. **Given** [initial state], **When** [action], **Then** [expected outcome]

---

[Add more user stories as needed, each with an assigned priority]

### Edge Cases

<!--
  ACTION REQUIRED: The content in this section represents placeholders.
  Fill them out with the right edge cases.
-->

- What happens when the user runs the CLI without a prompt argument?
- How does the system handle an unknown, expired, or corrupted `--session` value?

## Requirements *(mandatory)*

<!--
  ACTION REQUIRED: The content in this section represents placeholders.
  Fill them out with the right functional requirements.
-->

### Functional Requirements

- **FR-001**: System MUST expose the feature through a Go CLI command contract.
- **FR-002**: System MUST execute the requested agent workflow autonomously without interactive prompts.
- **FR-003**: Users MUST be able to resume follow-up work with `--session=<identifier>`.
- **FR-004**: System MUST persist enough session state to resume prior agent context safely.
- **FR-005**: System MUST return explicit stdout/stderr output and exit codes for success and failure paths.
- **FR-006**: System MUST use `github.com/google/adk-go` for all AI interactions.

*Example of marking unclear requirements:*

- **FR-007**: System MUST authenticate users via [NEEDS CLARIFICATION: auth method not specified - email/password, SSO, OAuth?]
- **FR-008**: System MUST retain user data for [NEEDS CLARIFICATION: retention period not specified]

### Key Entities *(include if feature involves data)*

- **[Entity 1]**: [What it represents, key attributes without implementation]
- **[Entity 2]**: [What it represents, relationships to other entities]

## Success Criteria *(mandatory)*

<!--
  ACTION REQUIRED: Define measurable success criteria.
  These must be technology-agnostic and measurable.
-->

### Measurable Outcomes

- **SC-001**: [Measurable metric, e.g., "Users can complete account creation in under 2 minutes"]
- **SC-002**: [Measurable metric, e.g., "System handles 1000 concurrent users without degradation"]
- **SC-003**: [User satisfaction metric, e.g., "90% of users successfully complete primary task on first attempt"]
- **SC-004**: [Business metric, e.g., "Reduce support tickets related to [X] by 50%"]

## Constitutional Alignment *(mandatory)*

<!--
  ACTION REQUIRED: Capture the repo-level constraints that downstream plan/tasks work
  must satisfy.
-->

- **Repo Scope Impact**: [Which repository areas are expected to change, or state N/A]
- **Verification Strategy**: [Describe pure-function tests, glue/integration tests, and CLI e2e coverage for each story]
- **Integration Parity Impact**: [List OpenCode/Copilot/shared template updates required, or state N/A]
- **CLI Contract Impact**: [Document prompt argument format, `--session` behavior, stdout/stderr contract, exit codes, and how `github.com/google/adk-go` is used]
- **Complexity Justification**: [Note any required abstraction, compatibility layer, or state N/A]

## Assumptions

<!--
  ACTION REQUIRED: The content in this section represents placeholders.
  Fill them out with the right assumptions based on reasonable defaults
  chosen when the feature description did not specify certain details.
-->

- [Assumption about target users, e.g., "Users have stable internet connectivity"]
- [Assumption about scope boundaries, e.g., "Interactive prompts are out of scope for v1"]
- [Assumption about data/environment, e.g., "Session state is stored locally and is available between CLI runs"]
- [Dependency on existing system/service, e.g., "Requires access to `github.com/google/adk-go` and the configured model provider"]
