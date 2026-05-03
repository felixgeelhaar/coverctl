# ECP Discovery Interviews — coverctl

**Goal:** Validate or kill the wedge that coverctl is built around — coverage feedback for AI coding agents — by hearing struggling moments unprompted from the people coverctl was built for.

**Target:** 10 interviews in 30 days (one per ~2-3 days).

**Decision rule:**
- ≥6 of 10 surface coverage-blind-spot pain in their AI-coding loop unprompted → **commit to the wedge.** Build agent-specific tools (suggest_tests, explain_uncovered, coverage_diff_for_changed_files). Plan launch.
- 3-5 of 10 → **wedge is real but narrow.** Reposition coverctl as agent-native CLI for OSS maintainers using AI agents heavily. Skip SaaS expansion.
- ≤2 of 10 → **wedge is intellectually interesting but commercially weak.** Decide: keep as craft project, or pivot to "domain-aware coverage policy" without the agent angle.

---

## Recruiting

### Who
Engineers using AI coding agents heavily (≥1 hour/day) on multi-file or multi-language codebases. Recommended titles: senior, staff, lead, principal, tech-lead, EM-with-IC-time. Company size 50-500.

### Where to find them
- **r/cursor** — DM users who have posted detailed workflows in the last 60 days
- **r/ChatGPTCoding** — same
- **Anthropic Claude Code Discord** — `#showcase` and `#help` channel actives
- **Cline / Aider Discord servers** — same
- **HN comments** — search for "I use Cursor" / "I use Claude Code" with technical depth
- **Twitter/X** — search for `Cursor "test coverage"` and similar
- **Personal network** — engineers you know who mention agents in their workflow

### Outreach script (≤500 chars, one paragraph)

> Hi — I'm researching how AI coding agents handle test coverage. Saw your [post / comment / repo]. Would you be up for a 25-minute call about your workflow — what works, what's frustrating? No pitch, no recording without consent. I'm building a tool in this space and trying to make sure I'm solving a real problem before I keep building. Available [3 specific time windows]. Calendly: [link].

### Compensation
$50 Amazon gift card or open-source maintainer thank-you (their pick). Mention upfront.

---

## Interview rules (non-negotiable)

1. **Story-based, never hypothetical.** Always "Tell me about the last time…" never "Would you use a tool that…"
2. **Listen for struggling moments.** A struggling moment = a specific past event where something broke down and progress stalled. Pain expressed as a story, not as a complaint.
3. **No pitching coverctl.** Not in the call, not in the calendar invite. If they ask what you're building, say: "A coverage tool for AI agents. I'd rather show it to you after I've heard your workflow than colour your answers."
4. **Silence is a tool.** When they trail off, count to five. They'll often add the most honest detail in that pause.
5. **Capture exact quotes.** Verbatim. Their words become your copy.
6. **Same script every time.** Comparable data > flexibility.
7. **Recap at end.** "Here's what I heard. Did I miss anything?" — surfaces what they meant vs what they said.

---

## Pre-interview prep

- 5 min skim their GitHub / public posts. One specific reference to acknowledge in the call.
- Calendar block: 30 min (25 + 5 buffer).
- Quiet room. Headphones. Camera on (theirs optional).
- Notion/markdown doc open with the script below pre-filled.

---

## The script

### 0. Set the frame (2 min)

> "Thanks for the time. I want to learn how you work with AI coding agents — specifically around testing and code quality. I'm not selling anything in this call. I might ask if you'd look at something later, but only if it's relevant. I'll take notes. Anything you want off the record, just say. Sound good?"

### 1. Workflow & agent setup (3 min)

> "Walk me through your last full coding day. What were you working on? What tools were open?"

Listen for: which agent (Cursor / Claude Code / Cline / Aider / Copilot / multiple), language(s), repo size, solo vs team.

> "How did the agent fit in? What part did it own, what did you do?"

### 2. The struggling moment (8 min — the core)

> "Tell me about the last time the agent shipped you broken or untested code that bit you later."

Wait. Don't fill silence. If nothing, try:

> "Tell me about the last time you were unsure whether a function the agent wrote was actually tested."

Or:

> "Tell me about the last time you reviewed an agent's PR and noticed a coverage gap."

Follow-up probes (use only what fits):
- "What did you do at that moment?"
- "What happened next?"
- "Why was that frustrating? What were you hoping would be different?"
- "Did you mention it to the agent? What did you say?"
- "What did you do to make sure it didn't happen again?"
- "How often does this happen — once a week? Once a day? Once an hour?"

### 3. Current coping mechanisms (5 min)

> "What do you do today to make sure agent-written code is tested?"

Listen for: manual review, asking the agent to write tests, running coverage tool separately, ignoring the problem, accepting reduced quality, ratcheting CI gates.

> "Has any tool helped? What did it do well? What did it not solve?"

If they mention Codecov / Coveralls / SonarQube / native cover tools: "What's the gap? What do you wish it did differently?"

### 4. Imagining the fix (3 min — only after pain established)

> "If a coverage tool sat inside the agent's loop — could tell the agent 'you just touched the auth domain, coverage dropped to 71%, here are 3 functions to test' — would that have helped in [the moment they described]?"

Listen for: enthusiasm vs polite-yes. Polite-yes = no.

> "What would make you not use it? What would make you use it once and abandon it?"

### 5. Spend & adoption (2 min)

> "Who pays for your dev tooling — you, your team, your company?"
> "If a hosted version of this kind of tool existed, who would have to approve a $X/mo spend? Where's the threshold above which it stops being self-approve?"

### 6. Close (2 min)

> "Last question — anyone in your network you'd recommend I talk to about this?" *(Snowball.)*
> "Can I share what I'm building when I have something to show? No pressure to use it."
> Recap: "Here's what I heard…" Confirm or correct.

---

## Capture template (one per interview)

```yaml
date: YYYY-MM-DD
who:
  name: ...
  role: ...
  company_size: ...
  agent_stack: [Cursor, Claude Code, ...]
  primary_languages: [Go, Python, ...]
  recruited_from: ...
  compensation: $50 / OSS thank-you / declined

unprompted_pain_struggling_moment:  # quote them verbatim
  quote: |
    "..."
  frequency: <once an hour | once a day | once a week | rarer>
  blast_radius: <self only | team | downstream users>

current_workaround:
  what_they_do: ...
  what_tool_if_any: ...
  gap_with_existing_tool: ...

reaction_to_wedge_pitch:  # only after section 2-3
  enthusiasm: <strong | mild | polite-yes | indifferent | negative>
  surprises_us: ...
  blockers_to_adoption: ...

monetization_signal:
  who_pays: ...
  self_approve_ceiling_per_month: $...

snowball_referrals: [...]
follow_up_consent: <yes | no>

scoring:
  unprompted_pain: <1-5>  # 1 = no pain mentioned, 5 = vivid recurring story
  fit_with_wedge: <1-5>   # 1 = wrong target, 5 = ICP bullseye
  conversion_signal: <1-5>  # would they actually use + pay
```

---

## After every 3 interviews

- 30-min review block. Read all 3 capture docs side by side.
- Update positioning brief based on language they used (their words → README copy).
- Note repeated quotes. Three people saying the same thing in different words = a strong signal.
- Update this script if a probe is consistently dead-air. Replace with one that worked.

## After all 10

Decision meeting (you, alone or with one trusted advisor):
1. Score each interview (1-5 across the three axes).
2. Apply decision rule above.
3. Write 1-page memo. Preserve the loudest quotes.
4. If "commit": draft launch plan (Show HN + r/cursor + devtools newsletters).
5. If "narrow": write the new positioning explicitly.
6. If "weak": decide kill or keep-as-craft. Make the call.

---

## Anti-patterns

- **Hypothetical questions.** "Would you use…" is poison. Always past, always specific.
- **Leading the witness.** "Don't you find it frustrating when…" — they'll agree to be polite.
- **Pitching mid-interview.** Even one sentence corrupts the rest.
- **Asking for features.** "What features would you want?" produces a wishlist of things they will never use. Ask about the problem, not the solution.
- **Trusting "yes I'd pay."** Self-reported willingness-to-pay is ~30% of actual. Discount accordingly.
- **Stopping after 3 enthusiastic interviews.** You're calibrating against the loudest, not the average. 10 is the floor.
