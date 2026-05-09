
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

## Wedge Re-anchoring: PRD and ICP

Reframe PRD around agent-loop wedge (in-loop coverage feedback before commit). Prune personas to Taylor primary + Jordan secondary buyer. Replace vanity success metrics (5+ langs shipped, 1000 installs, GitHub stars) with North Star (Weekly Protected Agent Loops) plus input metrics (activation, MCP tool-call success, pre-commit hook adoption, regressions caught per session). Reframe ICP brief competitive alternatives to lead with red-CI-agent-loop status quo (not Codecov). Pull compliance-sensitive paths from ECP gate to expansion accelerator. Replace front-page positioning with single Initiative Hypothesis.

---

## Agent-Mode Onboarding Path

Add parallel Terminal vs AI Agent quick-start path. Agent-mode page shows install, enabling MCP server in Claude Code/Cursor/Cline, first agent-initiated check, agent UI transparency (tool-call visible), approval gate example, structured rejection example, override capability. Cross-link from terminal quick-start. Closes wedge-invisible gap for primary ICP at first contact.

---

## Golden-Path Failure-Mode Snippets

Add per-step caution blocks to quick-start.mdx covering predictable first-run failures: no language markers detected, missing language toolchain (pytest-cov, nyc, cargo-tarpaulin), profile path mismatch, threshold-too-high first FAIL, no tests detected. Each block names the failure, gives the exact recovery command. Closes the original T-5 requirement properly.

---

## Realistic CLI Output with Inline Next-Action Hints

Redesign coverctl check terminal output: realistic mixed PASS/FAIL rows with shortfall delta, summary line, and inline next-action footer (run coverctl suggest DOMAIN / coverctl debt). Update quick-start sample to mirror real output. Adds designed Peak-End moment on first passing check (subtle success line + next-step nudge). Touches internal/cli/check.go print path and docs sample.

---

## MCP Agent-Loop Eval Harness

Build internal/eval/ skeleton: 50-100 synthetic regression scenarios across supported languages (known coverage drop in known domain), scripted headless MCP agent replay, LLM-as-judge for output-interpretation accuracy, tool-selection accuracy and recall metrics, adversarial prompt-injection eval set. Wire into CI as gate. Establishes denominator for North Star regression catch rate that telemetry alone cannot measure.

---

## Mode-Aware MCP Tool Surface

Add coverctl mcp serve --mode=agent|ci flag. Agent mode advertises pruned 3-tool surface (check, suggest, debt) for reliable agent tool selection within context budget. CI mode advertises full 8 tools (adds report, compare, record, badge, pr-comment). Auto-detect mode by MCP client-id where possible. Validates the check/suggest/debt value-driver hypothesis from metrics spec.

---

## MCP Output Boundary Hardening

Canonicalize and escape user-controlled strings in MCP tool outputs (file paths, test names, profile contents, PR descriptions in pr-comment) before return to agent. Closes Lethal Trifecta exposure where untrusted content flows from coverage profiles back into agent context as a new prompt-injection vector. Add 50+ adversarial output-injection tests under internal/mcp/. Update docs/security/mcp-threat-model.md with output-boundary controls section.

---

## Structured Rejection Schema and Output Budgets

Stable JSON schema for all MCP rejection responses with required fields: passed=false, error_code, summary, remediation (agent-actionable next step). Add per-tool output token budget (default 2K), pagination cursors for overflow, verbosity flag (brief|normal|verbose) so agents can request minimal default. Auto-truncate verbose outputs (e.g., report) to top-N failing domains with summary. Reduces context pollution and prevents agent-stuck-on-rejection failures.

---

## Pricing and Monetization Wedge Decision

Two-page strategy decision doc evaluating monetization options for coverctl: open-core (paid hosted coverage history, team dashboards, cloud MCP relay), paid SLA support contracts, enterprise security feature gate (audit logging, SSO, compliance exports), or remain pure OSS with sponsorship. Decide before scaling community-led GTM. Decision artifact only, no implementation in this task. Output: docs/strategy/monetization-decision.md.

---

## Category Point-of-View Doc

Two-page Lochhead Point-of-View document defining the agent-loop coverage category. Sections: world today (red-CI agent loops, polyglot tool sprawl, governance gap), world we are describing (in-loop coverage governance, MCP-callable, polyglot-uniform), why now (3 bullets: Claude Code adoption inflection, MCP standard emerging, polyglot pain), the category name, who benefits most (pruned ECP). Drives README hero, content calendar seed articles, conference pitch language. Output: docs/strategy/category-pov.md.

---

## Activation Funnel and GTM Metrics

Distinct GTM funnel metrics layer separate from tool-execution telemetry: activation rate (init users reaching first passing check), 30-day usage retention (repos calling check weekly), advocate-mention count (unprompted Claude Code/Discord/X mentions), plugin marketplace install velocity, enterprise inbound (procurement document requests), Sean Ellis 40% PMF survey infrastructure. Opt-in trace donation pipeline for real-world data growing eval corpus.

---

## 5-User Polyglot Usability Test

Krug-style observed usability test: recruit 5 polyglot devs actively using Claude Code or Cursor, watch them install coverctl from scratch and reach first fix. Two using Python+TS, two using Go+Rust, one using Java or Shell. Measure: did they discover MCP/agent integration unprompted, did first failed check produce a clear next action, time-to-first-fix, abandonment points. Single-day spend. Validates onboarding fixes from features F2/F3/F4 before broader rollout.

---

## Module-Root Failure Remediation

Surface a clearer error when coverctl check cannot resolve the Go module root. Close the runtime boundary gap evidenced in issue #20 ('module root not found' even with valid go.mod), where the user explicitly asked for an actionable hint. Add a new operational rejection code (e.g., OP_MODULE_ROOT_MISSING) plus matching remediation copy: list searched paths, suggest passing --language explicitly, suggest running from repo root, suggest checking for nested submodules. Apply at the application service boundary so both CLI and MCP handlers emit the same structured response. Output: extended sanitize.go op-codes, service.go diagnostic, mcp handler integration, regression test.

---

## Rust Quick-Start Tab

Add a Rust example to the quick-start tabs in both terminal and agent-mode quick-starts. Closes the residual discoverability gap after PR #48 fixed Rust LCOV parsing: a Rust user landing today still sees only Go/Python/TS samples and may assume Rust is unsupported. Show the canonical Rust .coverctl.yaml block, the cargo-llvm-cov invocation, and the LCOV path coverctl reads by default. Output: docs/src/content/docs/quick-start.mdx Tabs item; equivalent agent-mode example in quick-start-agent.mdx; cross-link from troubleshooting if relevant.

---

## coverctl mcp doctor First-Run Validation

Add a coverctl mcp doctor subcommand that runs the same initialize handshake an MCP client issues, then prints a structured success/failure report to stderr. Closes the opacity pattern surfaced in issues #8 (server EOF on initialize) and #19 (cwd context confusion) where setup failures had no in-product diagnostic path and forced GitHub-issue-driven debugging. Doctor checks: binary on PATH, working directory has expected markers, .coverctl.yaml resolvable, MCP initialize roundtrip OK, sample tool dispatch (check --validate) OK, mode auto-detect signal. Outputs each step as PASS/FAIL with remediation. Output: internal/cli/cmd_mcp.go new subcommand, command help text, mcp.mdx doctor section, regression test.

---

## Public Surface Wedge Reframe

Re-anchor README and Astro landing page body to the agent-loop wedge. Hero already aligned but body content (Problem/Solution/Cards/Designed for Teams) still argues domain-coverage features inside the Codecov category frame. Scope: rewrite The Problem + The Solution sections around agent-loop struggling moment (not 'one number for whole codebase' Codecov critique); replace generic Cards with agent-loop-anchored value props; cut 'Designed for Teams' section; add named category phrase 'agent-loop coverage governance' to hero + first paragraph and repeat through copy; remove copy leaks ('first-class', 'domain-driven design principles', uncontextualized 'local-first'); compress over-feature-densified sub-line; trim landing hero CTAs from 3 to 2 (drop GitHub from hero, available in nav); drop Languages badge (claim already in tagline). Output: rewritten docs/src/content/docs/index.mdx body + README copy edits.

---

## Transparency-Moment Artifact

Add the missing proof artifact that shows what coverctl looks like inside an agent session. Three components: (1) 6-line agent-session transcript using ai-expert's concrete copy (user prompt → agent edits → coverctl check tool call → structured rejection → coverctl suggest chained call → agent's natural-language plan with explicit failing domain + uncovered lines); (2) anti-pattern card with three concrete hallucination markers (lowering thresholds as 'fix', claiming coverage rose without a new check call, ignoring error_code on retry); (3) compressed agent-calibration block ('does well' / 'watch for' / 'contract is in docs') promoted above the install line so trust is calibrated before install (Mollick informed-use vs misinformed-use). Land in both README (after hero) and landing (between hero and feature cards). Validate: run prompt through real Claude Code session before publishing to ensure transcript matches actual client rendering.

---

## Security and Privacy Public Posture

Surface coverctl's full AI-infrastructure maturity on public copy. Today README documents only the input boundary (sanitization rejecting dangerous flags); the output-boundary canonicalization that ships in internal/mcp/sanitize_output.go is invisible. Procurement-grade buyers (Jordan) read security copy line-by-line and downgrade trust on incomplete threat models. Scope: (1) replace 'Security note' with two-bullet Security Boundaries section covering input + output + Lethal Trifecta framing; (2) reframe 'MCP-native, first-class MCP server' to 'agent-loop native via MCP' with explicit multi-vendor acknowledgment (Anthropic/OpenAI/Google/MS/AWS, Linux Foundation co-governance) — addresses procurement vendor-lock-in question; (3) add three-sentence privacy-first + eval-gated framing (local-first default, opt-in telemetry, 50+ adversarial eval scenarios); (4) compatibility surface single source (test against Claude Code, Claude Desktop, Cursor, Cline, Aider, Continue, OpenCode) — eliminates README/landing inconsistency. Mirror to landing where appropriate.

---

## Community and Platform-Teams Surface Area

ICP brief names community-led as PRIMARY motion and content-led as secondary. Today neither README nor landing exposes any community surface area: no Claude Code marketplace link, no MCP Registry link, no Discussions CTA, no GitHub Sponsors button, no contributor recognition. Strategy without distribution. Scope: (1) add Community section to README between Quickstart and CLI reference (marketplace + MCP registry + Discussions + Sponsor + maintainer credit); (2) add For Platform & DevEx Teams section linking threat-model + rejection-schema + GTM-metrics-spec artifacts plus low-friction inbound path via 'platform-evaluation' GitHub issue label — opens Stage 4 monetization trigger detection; (3) one-line PMF survey nudge ('Used coverctl for a few weeks? Run coverctl survey to share PMF feedback') under Community — zero-cost activation of existing instrumentation; (4) pull-quote the ICP brief's customer struggling-moment line above install commands (Gerhardt audience-first); (5) pre-emptive monetization framing in 'Why this exists' (free CLI/MCP forever, hosted history additive not paywall) — defuses bait-and-switch perception when Stage 2 launches.

---

## README and Landing Information Architecture

UX-layer fixes to scan/hierarchy/IA on both surfaces. Today README hides primary action (install) below 22 lines of prose; landing CardGrid equalizes agent and terminal paths the strategy says are not equal; trust-calibration content lives only in quick-start.mdx (post-install, too late). Scope: (1) Reorder README first viewport — install line + MCP JSON config + agent-prompt example as first scannable block (hero → install → transcript → why → tool reference → CLI reference → configuration → community → security → contributing); (2) add scannable nav strip near top of README ([Get started] · [MCP tools] · [CLI reference] · [Configuration] · [Why this exists]); (3) collapse 'Quickstart for humans' and 'Golden path' into CLI reference (they're how-to fragments, not first-touch); (4) move detailed Architecture section to dedicated ARCHITECTURE.md (or below Contributing); (5) replace symmetric Quick Start CardGrid on landing with asymmetric primary agent card + inline secondary terminal link (Von Restorff + Hick's); (6) add trust-calibration one-liner immediately after wedge claim on both surfaces ('Works best on standard Go/Python/JS/Java/Rust... Mock-heavy code or exotic monorepos may need explicit domains: block').

---
