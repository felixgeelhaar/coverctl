📐 Technical Design Document (TDD)

1. Architecture Overview
   ┌──────────────┐
   │ coverctl │ CLI
   └──────┬───────┘
   │
   ┌──────▼────────────────────────┐
   │ Core Engine │
   ├───────────┬───────────────────┤
   │ Config │ Domain Resolver │
   │ Loader │ (go list) │
   ├───────────┼───────────────────┤
   │ Coverage │ Policy Evaluator │
   │ Runner │ │
   ├───────────┼───────────────────┤
   │ Reporters │ Autodetector │
   └───────────┴───────────────────┘

2. High-Level Flow (check)

Locate module root (go env GOMOD)

Load config or run autodetection

Resolve domain package patterns

Run go test with:

-covermode=atomic

-coverpkg=<union of domains>

Parse coverage profile

Apply excludes

Aggregate coverage per domain

Evaluate policy

Emit report + exit code

3. Module Layout
   coverctl/
   ├── cmd/coverctl/            # legacy entrypoint (thin wrapper)
   │   └── main.go
   ├── main.go                  # root entrypoint for go install
   ├── internal/
   │   ├── domain/              # core domain types, policy, checker
   │   ├── application/         # services: check, run, report, detect, badge, trend, debt
   │   ├── infrastructure/      # adapters
   │   │   ├── config/          # YAML config loader
   │   │   ├── gotool/          # go test runner
   │   │   ├── profile/         # coverage profile parser
   │   │   ├── report/          # text/JSON/HTML reporters
   │   │   ├── badge/           # SVG badge generator
   │   │   ├── history/         # trend history JSON store
   │   │   ├── watcher/         # file system watcher (fsnotify)
   │   │   └── diff/            # git diff provider
   │   └── cli/                 # CLI parsing, output, wiring
   ├── templates/               # default config template
   └── schemas/
       └── coverctl.schema.json

4. Coverage Execution
   Unit Mode
   go test \
    -covermode=atomic \
    -coverpkg=./... \
    -coverprofile=.cover/coverage.out \
    ./...

Integration Mode (future)

go build -cover

GOCOVERDIR

go tool covdata merge

go tool covdata textfmt

5. Coverage Parsing

Native Go coverage text format

Parsed into:

file

block

statements covered / total

Normalized to module-relative paths.

6. Domain Resolution
   Input

domain.match → package patterns

Process

go list -json

map import path → dir → files

attach files to domains

allow multi-domain overlap (warn)

7. Policy Evaluation
   type PolicyResult struct {
   Domain string
   Covered int
   Total int
   Percent float64
   Required float64
   Status Pass|Fail
   }

Evaluation:

apply excludes first

compute % per domain

compare against domain.min or default.min

8. Reporting
   Text
   Domain Coverage Required Status

---

core 83.2% 85.0% FAIL
api 81.1% 80.0% PASS

JSON
{
"domains": [...],
"summary": { "pass": false }
}

HTML
- Styled HTML report with coverage percentages
- Color-coded status indicators (PASS/WARN/FAIL)
- Domain and file-level breakdown

SVG Badge
- Coverage percentage badge for README embedding
- Configurable label, style (flat, flat-square)
- Color thresholds: green (80+), yellow (60-79), red (<60)

9. Error Handling & Exit Codes
   Code Meaning
   0 Pass
   1 Policy violation
   2 Config error
   3 Tooling error
10. Performance & Safety

Always -covermode=atomic

Single go test invocation

No reflection or runtime hooks

No mutation of source code

11. Security & Trust

No network calls

No telemetry

Deterministic output

CI-safe

12. OSS Considerations

Apache-2.0 or MIT

Extensive README examples

Golden-file tests for coverage parsing

Test repos for autodetection validation

13. Advanced Features

Watch Mode
- File system watcher using fsnotify
- Debounced re-runs on .go file changes
- Skips hidden directories and vendor/
- Signal handling for graceful shutdown (Ctrl+C)

Coverage Trends
- JSON history file (.coverctl-history.json)
- Records coverage per domain over time
- Trend analysis with configurable day range
- Integration with CI via commit/branch metadata

Threshold Suggestions
- Strategies: current, aggressive, conservative
- Analyzes actual coverage to recommend thresholds
- Optional --write-config to apply suggestions

Coverage Debt
- Calculates shortfall from required minimums
- Estimates lines of code needing tests
- Health score (0-100) for overall status
- Domain and file-level breakdown

Domain-Specific Excludes
- Per-domain exclude patterns
- Fine-grained control over coverage scope
- Separate from global excludes
