
## Product Messaging & Docs Alignment

Align CLI help text, README, and PRD with current reality (15 languages, MCP-native, domain-aware positioning). Remove all Go-only wording from user-facing copy unless contextually Go-specific.

---

## AI/Agent Success Metrics Baseline

Define and instrument MCP tool-call success metrics: success rate, rejection rate, time-to-success, pre-commit regression catch rate. Enable data-driven decisions on which 3 MCP tools drive retained value.

---

## Architecture Drift Guardrails

Add explicit extraction plan and enforcement for acknowledged large files (service.go, cli.go, server.go). New capabilities must land in dedicated handler files per the existing ceiling-test contract.

---

## Golden Path UX for First-Run

Tighten docs and CLI flow around a single golden path: init -> check -> suggest -> record. Include actionable failure-snippet guidance per step. New user goal: install to first fix in <10 minutes.

---

## CI Product Metrics Dogfooding

Extend CI workflow (go.yml) to preserve machine-readable check/report outputs for trend analysis. Current dogfooding only asserts pass/fail; add structured artifact collection.

---

## GTM & Enterprise Readiness Package

Build ICP-focused targeting for polyglot AI-assisted teams with compliance governance needs. Publish security architecture doc (MCP threat model + sanitization boundaries).

---

## Coverage Quality Hotspot Uplift

Add scenario tests for weaker coverage surfaces: internal/mcp, internal/cli, runner edge cases. Focus on failure-path handling and parser/runner boundary conditions.

---
