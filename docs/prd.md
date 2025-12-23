📘 Product Requirements Document (PRD)
Product Name (Working)

coverctl

Declarative, domain-aware test coverage validation for Go

1. Executive Summary

Go provides excellent mechanisms for collecting test coverage but intentionally omits policy enforcement. This creates a gap in professional environments where teams must:

enforce minimum coverage,

reason about coverage in meaningful architectural units (not just global %),

prevent regressions in CI.

coverctl fills this gap.

It is an OSS, CLI-first tool that:

runs Go’s native coverage tooling correctly by default,

introduces coverage domains (logical slices of the codebase),

validates coverage against configurable policies,

autodetects domains to reduce onboarding friction,

fails CI builds when coverage policy is violated.

2. Problem Statement
   Problems with native Go coverage today

Global percentage is misleading

80% overall coverage can hide 0% coverage in critical paths.

No enforcement

go test reports coverage but never fails a build.

High setup cost

Correct usage of -coverpkg, -covermode=atomic, and covdata is non-obvious.

Integration tests are second-class

Go 1.20 enables binary coverage, but orchestration is complex.

Coverage lacks architectural context

Teams think in domains, not packages.

3. Goals & Non-Goals
   Goals

Enforce coverage as a policy, not just a metric

Make coverage domain-aware

Require zero third-party Go coverage libraries

Work out-of-the-box in CI

Support unit coverage first, integration coverage second

Be transparent, explainable, and debuggable

Non-Goals

Replacing go test

Providing IDE UI

Language-agnostic coverage (Go only)

Deep condition/path coverage analysis

4. Target Users
   Primary

Go backend teams

Platform / DevEx teams

OSS maintainers

CI/CD engineers

Secondary

Governance & quality tooling platforms

Mono-repo maintainers

SRE / reliability teams

5. Key Concepts
   5.1 Coverage Domain (Core Concept)

A domain is a named policy scope mapping to a set of Go packages.

Example:

- name: core
  match: ["./internal/core/..."]
  min: 85

Domains:

reflect architecture

have independent thresholds

surface meaningful failures

6. User Experience
   6.1 Primary CLI Commands
   coverctl check # run coverage + enforce policy
   coverctl run # run coverage only, produce artifacts
   coverctl detect # autodetect domains
   coverctl report # analyze existing profile

6.2 Typical CI Usage
coverctl check --config .coverctl.yaml

Expected behavior:

prints domain-level report

exits non-zero on violation

produces machine-readable output

7. Configuration
   7.1 Config File (.coverctl.yaml)

Key features:

versioned schema

defaults + overrides

domain-based rules

file/package exclusions

(Example abbreviated)

policy:
default:
min: 75
domains: - name: core
match: ["./internal/core/..."]
min: 85

8. Autodetection
   Goals

Zero-config onboarding

Explainable heuristics

Safe defaults

Detected Domains

cmd

internal

pkg

inferred subdomains (transport, adapters, db)

generated/mocks (excluded)

Output
coverctl detect --write-config

9. Output & Reporting
   Human-Readable

tabular summary

colored pass/fail indicators

actionable error messages

Machine-Readable

JSON (via `-o json`; text is the default format)

exit codes for CI

10. Success Metrics

< 1 minute setup time

CI-ready without scripting

Adoption in OSS Go repos

Clear failure diagnostics

No false positives from concurrency

11. Roadmap
    v1.0

Unit coverage enforcement

Domains

Autodetection

JSON + text output

v1.1

Integration coverage (Go 1.20+)

Coverage merging

Diff-based checks

v1.2

Per-file rules

Annotations / pragmas

GitHub Action

v1.3

HTML coverage reports

Severity levels (warn thresholds)

SVG badge generation

Coverage trend tracking

Threshold suggestions

Coverage delta in diff mode

Domain-specific excludes

Watch mode for continuous coverage

Coverage debt reporting
