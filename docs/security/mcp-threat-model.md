# MCP Security Architecture

## Purpose

This document describes the threat model for `coverctl mcp serve`, the trust boundaries around MCP inputs, and the hardening controls in place to reduce prompt-injection-to-code-execution risk.

## System boundary

- Entry point: MCP tool/resource requests over stdio.
- Primary component: `internal/mcp/server.go`.
- Security control surface: `internal/mcp/sanitize.go` (input validation/sanitization for untrusted MCP fields).

## Threat model

### Assets

- Local developer machine and CI runner integrity.
- Repository contents and config files.
- Coverage artifacts and generated reports.

### Trust boundaries

1. **Untrusted**: MCP arguments (may be derived from LLM output and external content).
2. **Trusted with validation**: resolved file paths and build flag fields after validation/sanitization.
3. **Trusted operator path**: direct CLI usage by a human in terminal (not MCP-mediated).

### Primary attack path

Prompt injection in upstream text (PR description, issue body, fetched page) influences an agent to call MCP tools with dangerous test-runner flags intended to load arbitrary code or pivot execution scope.

## Controls in place

### 1) Path scoping and validation

- Scoped path validation is applied to MCP path inputs before use.
- Rejected inputs return a structured rejection response (`passed=false`, explicit error, safe summary).

### 2) Build-flag sanitization

`internal/mcp/sanitize.go` blocks dangerous argument classes for MCP-originated inputs, including:

- Dangerous long flags (examples): `--rootdir`, `--cov-config`, `--init-script`, `--require`, `--node-options`, `--manifest-path`.
- Dangerous short prefixes (examples): `-D`, `-I`, `-P`.
- Shell metacharacters and control characters in free-form arg inputs.
- Invalid tag and timeout formats.

### 3) Fail-closed behavior

- Any failed sanitization returns a rejection; tool execution does not proceed.
- Rejection responses are deterministic and machine-readable for CI/agent handling.

## Explicit boundaries / non-goals

- CLI calls made by a human operator are not sanitized as MCP inputs are; the operator is the trust boundary.
- coverctl does not sandbox downstream language toolchains; it reduces attack surface by constraining MCP-supplied arguments.

## Operational guidance

- Use MCP mode for agent workflows: `coverctl mcp serve`.
- Prefer local-first execution in trusted repos.
- Keep toolchain dependencies updated.
- Treat repeated MCP rejection spikes as an indicator of prompt-injection attempts or malformed agent prompts.

## Residual risk

- New or unknown dangerous flags in third-party runners may emerge over time.
- Mitigation: maintain denylist updates in `sanitize.go`, keep tests current (`internal/mcp/sanitize_test.go`), and monitor rejection telemetry/logs.

## Code references

- `internal/mcp/server.go`
- `internal/mcp/sanitize.go`
- `internal/mcp/sanitize_test.go`
