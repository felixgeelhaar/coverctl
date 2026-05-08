# coverctl ICP GTM Brief

## Who this is for (ICP)

Primary ICP:
- Engineering teams (5-80 devs) using AI coding agents (Claude Code, Cursor, Cline) in polyglot repos.
- Operate with CI policy gates and at least one compliance-sensitive path (auth, payments, data pipelines).
- Pain today: regressions are caught late in CI, not in the agent edit loop.

Secondary ICP:
- Platform/devex teams standardizing coverage policy across languages and repos.

## Problem (customer words)

"Our AI agents can ship code fast, but they are blind to coverage policy while editing. We only see breakage in CI, after context is gone."

## Why now

- AI-assisted coding has shifted bottlenecks from code generation to quality governance.
- Polyglot codebases are common; per-language tool sprawl makes policy inconsistent.
- MCP tooling is becoming the integration layer for agent workflows; quality signals need to be agent-callable.

## Competitive alternatives (Dunford framing)

1. **Do nothing / rely on CI only**
   - Cheap today, expensive in cycle time and rework.
2. **Codecov/Coveralls/SonarQube dashboards**
   - Strong post-hoc visibility, weak in-loop agent feedback.
3. **Language-native commands only (`go test -cover`, `pytest-cov`, etc.)**
   - Fragmented policy and no unified domain-level enforcement.

## Differentiated capabilities

- MCP-native tool surface for agent-callable coverage checks.
- Domain-aware policy (`.coverctl.yaml`) with per-domain thresholds.
- Multi-language runners/parsers under one interface.
- Security hardening for MCP input (sanitization and scoped path validation).

## Differentiated value

- Catch regressions before commit while the agent still has context.
- Standardize policy across languages without switching tools.
- Reduce "red CI surprise" loops and PR rework.

## Positioning statement

For AI-assisted, polyglot engineering teams that need coverage policy confidence during coding,
coverctl is the MCP-native, domain-aware coverage enforcement tool that provides in-loop
coverage feedback before commit, unlike dashboard-only coverage products that report too late.

## Suggested motion

PLG + dev-content + enterprise proof points:
- PLG: quickstart via `coverctl init` and MCP integration.
- Dev-content: "agent-loop coverage" tutorials and failure-mode playbooks.
- Enterprise proof: security architecture note + policy examples for critical domains.

## First proof metrics

- Pre-commit regression catch rate.
- MCP `check`/`suggest` tool success rate.
- Time-to-fix after first failing coverage signal.
