# Changelog

All notable changes to `coverctl` will be documented here. Relicta manages this file automatically.

## [Unreleased]

### Added
- **HTML coverage reports**: Generate styled HTML reports with `-o html` flag
- **Severity levels**: Add `warn` threshold for domains (WARN status between min and warn)
- **Badge generation**: `coverctl badge` command generates SVG coverage badges
- **Coverage trend tracking**: `coverctl trend` and `coverctl record` for historical analysis
- **Threshold suggestions**: `coverctl suggest` recommends optimal thresholds
- **Coverage delta**: `--show-delta` flag displays coverage changes from history
- **Domain-specific excludes**: Per-domain `exclude` patterns for fine-grained control
- **Watch mode**: `coverctl run --watch` for continuous coverage on file changes
- **Coverage debt report**: `coverctl debt` shows coverage shortfall and remediation effort
- **Integration coverage**: Support for Go 1.20+ binary coverage with `GOCOVERDIR`
- **Profile merging**: Combine multiple coverage profiles for unified analysis
- **Diff-based checks**: Enforce coverage only on changed files with `diff.enabled`
- **File-level rules**: Per-file minimum thresholds with `files` config
- **Annotations**: `// coverctl:ignore` and `// coverctl:domain=NAME` pragmas

## [0.1.0] - 2024-??-??
- initial scaffolding of the CLI, DDD layers, report tooling, and documentation
- strict DDD/TDD guidance with coverage enforcement and Relicta release configuration
