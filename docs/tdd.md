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
   ├── cmd/coverctl/
   │ └── main.go
   ├── internal/
   │ ├── config/
   │ ├── autodetect/
   │ ├── runner/
   │ ├── coverprofile/
   │ ├── domains/
   │ ├── policy/
   │ ├── report/
   │ └── util/
   ├── pkg/ (optional public API)
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
