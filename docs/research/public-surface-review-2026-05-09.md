# Four-Expert Review of README and Landing Page (2026-05-09)

Review of `README.md` (277 lines) and `docs/src/content/docs/index.mdx`
(164 lines) against the agent-loop coverage governance wedge defined
in `docs/strategy/category-pov.md`.

Reviewers: product-expert, gtm-expert, ux-expert, ai-expert.

## Common verdict

Wedge re-anchoring landed at the *strategy* level (POV doc, PRD,
ICP brief). It did **not** propagate to the public surfaces. The
hero on each surface is wedge-aligned; the body falls back to a
domain-coverage feature catalogue inside the Codecov category frame
the strategy explicitly retired. A first-time visitor with weak
attention reads the body and concludes "another coverage tool with
a polyglot story" — exactly the visitor coverctl decided not to
fight for.

Closeable in one editing pass; no new product capability required.

## Findings by reviewer

### product-expert

- Hero good; sub-line over-feature-densifies.
- Body content (Problem / Solution / Cards / Designed for Teams)
  reinforces the wrong category frame.
- Quickstart-for-humans peer section dilutes Taylor-first intent;
  demote to one line under CLI reference.
- **Highest-leverage missing artifact:** the transparency moment.
  No transcript, no screenshot, nothing showing what coverctl looks
  like inside an agent session. Add a 6-line transcript immediately
  after the hero on both surfaces.
- "Designed for Teams" → "domain-driven design principles" leaks
  internal architecture and collides with the coverage-domain concept.
  Cut.
- Single "Get Started" CTA fails both personas. Pair Taylor-CTA with
  a Jordan-CTA.

### gtm-expert

- Category is named in `category-pov.md`, named **nowhere** on the
  public page. Lochhead test fails: visitor cannot repeat the phrase.
  Promote "agent-loop coverage governance" to hero + first paragraph.
- Competitive alternatives never named explicitly. Add a 3-row block
  (red-CI agent loop / dashboards / native commands) under the hero.
- Community-led is the **primary** motion per ICP brief; surface area
  on the README for that motion is **zero**. Add Community section
  (marketplace + MCP registry + Discussions + Sponsors + maintainer
  credit).
- Jordan-path absent from both surfaces. Add `For Platform & DevEx
  Teams` section linking threat-model + rejection schema +
  gtm-metrics-spec; create `platform-evaluation` issue label as the
  Stage-4-trigger inbound path.
- PMF survey (`coverctl survey`) ships in the binary; no surface
  mentions it. One-line nudge under Community.
- Pre-emptive monetization framing missing — risks bait-and-switch
  perception when Stage 2 launches.

### ux-expert

- README first viewport hides the primary action (install line at
  ~line 26, below 22 lines of prose). Reorder so install + MCP JSON +
  agent prompt are the first scannable block.
- Landing's symmetric two-LinkCard Quick Start equalizes paths the
  strategy says are not equal. Replace with asymmetric primary agent
  card + inline secondary terminal link.
- Diataxis collapse on README (tutorial + how-to + reference +
  explanation interleaved with no signposting). Add scannable nav
  strip; reorder by IA.
- Trust-calibration content lives only in quick-start.mdx
  (post-install). Promote a one-liner to both first-touch surfaces.
- Hero density: 3 CTAs on landing → 2; drop Languages badge.
- "Designed for Teams" cards convey nothing; cut.

### ai-expert

- Public copy is AI-Enhanced-shaped; product is AI-Native. Mismatch
  reads as feature catalogue when it should read as edit-loop
  integration.
- "MCP-native, first-class MCP server" reads as 18-month-old framing.
  MCP is now multi-vendor table stakes. Reframe to "agent-loop native
  via MCP" with explicit Anthropic/OpenAI/Google/MS/AWS
  acknowledgment — addresses procurement vendor-lock-in.
- README "Security note" covers input boundary only. Output boundary
  (Lethal Trifecta defense in `internal/mcp/sanitize_output.go`) is
  shipped but undocumented. Procurement-grade buyers downgrade trust.
- Privacy-first + eval-gated framing absent from public copy. coverctl
  ships full opt-in telemetry + 50+ adversarial eval scenarios; visitors
  can't tell. Three sentences fix it.
- Trust calibration content (does well / watch for) lives only post-
  install in quick-start-agent.mdx. Promote a compressed version above
  the install line — calibrates trust *before* install.
- Compatibility surface inconsistent (README lists 4 clients, landing
  lists 3). Single-source the list.

## Concrete transcript copy (ai-expert)

For the transparency moment on README + landing:

```
You: Add the email-validation feature to the api domain.

Claude: I'll add EmailValidator + tests in internal/api/email.

  [edits: internal/api/email.go, internal/api/email_test.go]

  Tool call → coverctl check
  Result   → passed: false; failing: api (78.2%, required 80%)
             uncovered: internal/api/email.go:42-58 (error paths)

  Tool call → coverctl suggest api
  Result   → uncovered files: email.go (16 lines), retry.go (4 lines)

  Two error branches in EmailValidator.Validate are uncovered.
  I'll add test cases for the empty-input and malformed-domain
  paths so api meets the 80% threshold before we commit.
```

Anti-pattern markers (add as separate visual block):

- ✗ "Coverage looks good now, all domains passing" — when api still failing
- ✗ "Increased coverage to 95%" — number agent invented
- ✗ "Adjusting threshold to 75% in .coverctl.yaml" — lowering bar
  rather than fixing gap

## Roady decomposition

Five new features added, decomposed into 17 implementation tasks:

1. **public-surface-wedge-reframe** (T30-T33) — body copy, named
   category, alternatives block, hero density.
2. **transparency-moment-artifact** (T34-T36) — transcript validation,
   README + landing placement.
3. **security-and-privacy-public-posture** (T37-T39) — full Security
   Boundaries section, MCP multi-vendor reframe, privacy/eval framing.
4. **community-and-platform-teams-surface-area** (T40-T42) — Community
   section, For Platform Teams, struggling-moment pull-quote, PMF nudge,
   pre-emptive monetization framing.
5. **readme-and-landing-information-architecture** (T43-T46) — first-
   viewport reorder, nav strip, asymmetric CardGrid, trust-calibration
   one-liner.

## Validation moves before publishing the rewrite

- **5-second test (Krug):** show two GitHub-savvy developers the current
  README and the rewritten one for 30 seconds each. Ask: "What is this
  and what would you do first?" Better answer wins.
- **Transcript fidelity (ai-expert):** run the exact transcript prompt
  through a real Claude Code session; reconcile against idealized copy
  before publishing.
- **Category-phrase adoption (gtm-expert):** 90 days post-rewrite,
  search Hacker News / Discord / GitHub repo descriptions for unprompted
  use of "agent-loop coverage governance". If the phrase shows up,
  category design is taking. If not, re-evaluate the framing.

## Hand-off

- Roady plan now 71 tasks: 49 done preserved, 22 new pending (17
  implementation + 5 parent wrappers).
- Recommended sequencing: T34 (transcript validation) first since it
  unblocks T35/T36 and informs other body copy. Then T30-T33 wedge
  reframe in parallel. Security and Community sections last as they
  build on the reframed body.
