# Kopilot Rearchitecture Implementation Plan

## Purpose

This document is the single source of truth for implementing the current rearchitecture work on branch feat/rearchitecture.

It is designed for deterministic execution later, with explicit sequencing, file-level scope, acceptance criteria, and stop conditions.

## Scope

In scope:

- Runtime correctness fixes for LLM abstraction integration
- Security and reliability hardening for kubectl execution path
- Multi-provider correctness and guardrails
- Documentation alignment with actual code and dependencies
- CI hardening and repository hygiene

Out of scope:

- New user-facing product areas unrelated to current rearchitecture
- Wide refactors of k8s collectors not needed for listed work packages
- Non-kopilot repositories in this workspace

## Execution Rules

1. Apply work packages in numeric order unless a package explicitly allows parallel execution.
2. Do not change public behavior outside each package scope.
3. Keep each package in its own commit.
4. After each package, run its verification commands before moving forward.
5. If a package fails verification after three fix attempts, stop and log blocker in the Blocker Log section.
6. Never force-clean unrelated branch changes.

## Branch and Baseline

Target repository:

- owner: e9169
- repo: kopilot
- working branch: feat/rearchitecture
- default branch: main

Baseline verification command set:

```bash
go test ./...
```

## Work Package Index

- WP-01 Event Type Alignment
- WP-02 MCP Server Config Propagation
- WP-03 Kubectl Validation Enforcement
- WP-04 OpenAI Streaming Tool Call Panic Guard
- WP-05 Attachment Safety Limits
- WP-06 Timeout Configurability
- WP-07 Provider and Session Contract Tests
- WP-08 Documentation Synchronization
- WP-09 CI and Security Workflow Hardening
- WP-10 Repository Hygiene Cleanup

## WP-01 Event Type Alignment

Objective:

- Ensure agent session event handling consumes normalized llm.EventType values emitted by all providers.

Primary files:

- pkg/agent/agent.go
- pkg/llm/types.go
- pkg/llm/copilot/provider.go
- pkg/llm/openai/provider.go
- pkg/llm/gemini/provider.go

Required changes:

1. Update session event switch in agent loop to match normalized constants and not provider-native string literals.
2. Replace string literal matching with llm.EventType constant matching.
3. Ensure message, delta, idle, error, usage pathways remain behaviorally identical to current intended flow.

Acceptance criteria:

- Streaming text renders correctly.
- Session idle state flips true when completion finishes.
- Usage metrics continue to populate when provider supplies usage events.

Verification commands:

```bash
go test ./pkg/agent ./pkg/llm
```

Commit message:

- fix(agent): align session handler with normalized llm event types

## WP-02 MCP Server Config Propagation

Objective:

- Ensure MCP servers loaded from config are passed correctly into Copilot provider session creation.

Primary files:

- pkg/agent/agent.go
- pkg/llm/copilot/provider.go
- pkg/llm/types.go

Required changes:

1. Standardize MCP server extra config type across abstraction boundary.
2. Remove failing type assertion path that can drop MCP servers silently.
3. Add safe conversion path for provider-specific expected structure.

Acceptance criteria:

- MCP server list from config is present in created Copilot session config.
- No panic or silent drop when MCP config exists.

Verification commands:

```bash
go test ./pkg/agent ./pkg/llm
```

Commit message:

- fix(llm): preserve MCP server config through provider abstraction

## WP-03 Kubectl Validation Enforcement

Objective:

- Enforce existing kubectl argument validation and sanitization before command execution.

Primary files:

- pkg/agent/tools.go
- pkg/agent/validation.go
- pkg/agent/validation_test.go

Required changes:

1. In handleKubectlExec flow, validate params.Args using validateKubectlCommand before execution.
2. Apply sanitizeKubectlArgs to execution argument set before building final command.
3. Ensure error messaging remains user-readable in both text and json output modes.

Acceptance criteria:

- Injection patterns are blocked before process spawn.
- Dangerous disallowed patterns return deterministic errors.
- Existing tests remain green and are expanded only if needed.

Verification commands:

```bash
go test ./pkg/agent
```

Commit message:

- fix(security): enforce kubectl validation and arg sanitization before exec

## WP-04 OpenAI Streaming Tool Call Panic Guard

Objective:

- Remove nil-pointer panic risk in streamed tool call chunk merge logic.

Primary files:

- pkg/llm/openai/provider.go
- pkg/llm/provider_smoke_test.go

Required changes:

1. Guard access to tc.Index when merging streamed tool call chunks.
2. Ignore or safely buffer malformed chunks without crashing session.
3. Preserve correct behavior for valid streamed tool chunks.

Acceptance criteria:

- No panic possible from nil tc.Index.
- Normal streaming tool calls still reconstruct correctly.

Verification commands:

```bash
go test ./pkg/llm
```

Commit message:

- fix(openai): guard streamed tool call chunk index to prevent panic

## WP-05 Attachment Safety Limits

Objective:

- Prevent prompt blowups and unsafe file ingestion from attachment expansion.

Primary files:

- pkg/agent/agent.go
- pkg/agent/agent_test.go

Required changes:

1. Define max attachment file size limit.
2. Define max cumulative attachment bytes per prompt.
3. Skip binary-like content for inline prompt injection.
4. Return explicit user-facing reason when attachment is rejected.

Acceptance criteria:

- Large files are rejected with clear message.
- Binary files are not blindly inlined.
- Small text files still work.

Verification commands:

```bash
go test ./pkg/agent
```

Commit message:

- fix(agent): add attachment size and content safety limits

## WP-06 Timeout Configurability

Objective:

- Replace hardcoded kubectl command timeout with configurable value.

Primary files:

- pkg/agent/tools.go
- pkg/agent/agent.go
- README.md

Required changes:

1. Introduce env-based timeout configuration, with sane default equal to current behavior.
2. Validate timeout value parsing and fallback safely.
3. Document variable in README usage and environment sections.

Acceptance criteria:

- Default timeout remains unchanged when env var absent.
- Custom timeout is applied when env var valid.
- Invalid env values fallback safely.

Verification commands:

```bash
go test ./pkg/agent ./...
```

Commit message:

- feat(agent): make kubectl timeout configurable via environment

## WP-07 Provider and Session Contract Tests

Objective:

- Add regression coverage for provider abstraction contracts and session lifecycle expectations.

Primary files:

- pkg/llm/provider_smoke_test.go
- pkg/agent/agent_test.go
- pkg/agent/tools_test.go

Required changes:

1. Add tests for normalized event lifecycle assumptions.
2. Add tests for provider switching session reset behavior where possible.
3. Add tests for MCP config pass-through behavior at abstraction boundaries.

Acceptance criteria:

- New tests fail on pre-fix behavior and pass after fixes.
- Test suite remains deterministic and does not require external credentials by default.

Verification commands:

```bash
go test ./pkg/agent ./pkg/llm
```

Commit message:

- test(agent): add provider abstraction lifecycle and regression coverage

## WP-08 Documentation Synchronization

Objective:

- Align docs with current dependency versions and multi-provider architecture.

Primary files:

- README.md
- docs/INSTALLATION.md
- docs/MODEL_SELECTION.md

Required changes:

1. Update dependency versions to match go.mod.
2. Remove Copilot-only required wording when openai and gemini provider modes are valid.
3. Update install and auth matrix by provider.
4. Verify markdownlint-related style constraints in modified docs.

Acceptance criteria:

- No stale versions for copilot sdk and k8s modules.
- Setup instructions are accurate for copilot, openai, and gemini modes.

Verification commands:

```bash
go test ./...
```

Commit message:

- docs: sync provider setup and dependency versions with rearchitecture

## WP-09 CI and Security Workflow Hardening

Objective:

- Make security workflow outputs actionable and reduce silent failure paths.

Primary files:

- .github/workflows/gosec.yml
- .github/workflows/ci.yml

Required changes:

1. Remove unconditional pass behavior for gosec and adopt explicit severity policy.
2. Keep workflow practical for contributor velocity, but fail on meaningful findings.
3. Optionally add docs lint workflow or equivalent docs validation path.

Acceptance criteria:

- Security checks no longer always pass by construction.
- CI behavior is explicit and documented.

Verification commands:

```bash
# local structural checks
go test ./...
```

Commit message:

- ci(security): enforce actionable gosec policy and tighten workflow behavior

## WP-10 Repository Hygiene Cleanup

Objective:

- Remove temporary scratch scripts from repository root and keep development helpers organized.

Primary files:

- refactor.py
- scratch_fix_tests.py
- scratch_fix_json_test.py

Required changes:

1. Delete root scratch scripts if no longer needed.
2. If needed for reference, move to a clearly named internal tooling directory excluded from release artifacts.
3. Ensure no build or test path depends on these scripts.

Acceptance criteria:

- Repository root contains only intentional project files.
- No tests or build tasks reference removed scratch scripts.

Verification commands:

```bash
go test ./...
```

Commit message:

- chore(repo): remove transient scratch scripts from project root

## Parallelization Plan

Safe parallel groups:

- Group A: WP-01 and WP-02 can be developed together but must be merged in order WP-01 then WP-02.
- Group B: WP-05 and WP-06 can run in parallel after WP-03.
- Group C: WP-08 and WP-09 can run in parallel after WP-07.

Non-parallel constraints:

- WP-07 depends on completion of WP-01 to WP-06.
- WP-10 must be last.

## Quality Gates

Per package gate:

- Package acceptance criteria satisfied
- Verification commands pass
- No unrelated file churn
- Commit message matches planned scope

Final integration gate:

```bash
go test ./...
```

Recommended optional checks:

```bash
make fmt
make vet
make lint
```

## Rollback Protocol

If a package introduces regressions:

1. Revert the package commit only.
2. Record reason in Blocker Log with failing command and error snippet.
3. Create follow-up patch under same package id with suffix A, B, or C.

## Blocker Log

- None yet.

## Implementation Journal Template

Use this template after each package:

- Package id:
- Date:
- Commit sha:
- Files changed:
- Acceptance criteria status:
- Verification commands run:
- Notes:
- Follow-up required:

## Implementation Journal

- Package id: WP-01 Event Type Alignment
- Date: 2026-04-28
- Commit sha: pending
- Files changed:
	- pkg/agent/agent.go
- Acceptance criteria status: passed
- Verification commands run:
	- go test ./pkg/agent ./pkg/llm
- Notes:
	- Replaced provider-native event string matching with normalized llm.EventType constants in setupSessionEventHandler.
	- Preserved existing behavior for message, delta, idle, error, and usage handlers.
- Follow-up required:
	- None

- Package id: WP-02 MCP Server Config Propagation
- Date: 2026-04-28
- Commit sha: pending
- Files changed:
	- pkg/llm/copilot/provider.go
	- pkg/llm/copilot/provider_test.go
- Acceptance criteria status: passed
- Verification commands run:
	- go test ./pkg/agent ./pkg/llm ./pkg/llm/copilot
- Notes:
	- Added robust MCP server parsing to support both typed and generic map payloads from ExtraConfig.
	- Removed the brittle type assertion path that could silently drop MCPServers.
	- Added unit tests for typed map, generic map, and invalid shape inputs.
- Follow-up required:
	- None

- Package id: WP-03 Kubectl Validation Enforcement
- Date: 2026-05-09
- Commit sha: pending
- Files changed:
	- pkg/agent/tools.go
	- pkg/agent/validation_test.go
- Acceptance criteria status: passed
- Verification commands run:
	- go test ./pkg/agent
- Notes:
	- Added validateKubectlCommand call in handleKubectlExec before cluster lookup, so injection patterns are blocked before any process spawn.
	- Added sanitizeKubectlArgs call; sanitized args used for both buildKubectlCommand and isReadOnlyCommand.
	- Error path returns user-readable message in both text and JSON output modes via existing buildKubectlTextResult/buildKubectlJSONResult helpers.
	- Introduced runKubectlCommandFunc indirection variable to enable unit test injection without starting real kubectl.
	- Added three integration tests: validation blocks before provider lookup, JSON error format, and sanitization of --token/-w before exec.
- Follow-up required:
	- None

## Ready To Execute Checklist

- [ ] Baseline tests pass
- [x] WP-01 complete
- [x] WP-02 complete
- [x] WP-03 complete
- [ ] WP-04 complete
- [ ] WP-05 complete
- [ ] WP-06 complete
- [ ] WP-07 complete
- [ ] WP-08 complete
- [ ] WP-09 complete
- [ ] WP-10 complete
- [ ] Final integration gate pass
