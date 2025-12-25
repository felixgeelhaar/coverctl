# Changelog

All notable changes to `coverctl` will be documented here. Relicta manages this file automatically.

## [Unreleased]

## [1.6.0] - 2025-12-25

### Added
- **MCP Server**: Add Model Context Protocol server via `coverctl mcp serve` for AI agent integration
  - Tools: `check`, `report`, `record` for programmatic coverage operations
  - Resources: `coverctl://debt`, `coverctl://trend`, `coverctl://suggest`, `coverctl://config`
  - STDIO transport for Claude Desktop and other MCP-compatible clients

### Fixed
- Correct jsonschema tag format for MCP SDK compatibility

## [1.5.0] - 2025-12-24

### Added
- **Brief output format**: `--output brief` for single-line LLM/agent-optimized output

## [1.4.0] - 2025-12-23

### Added
- **HTML coverage reports**: Generate styled HTML reports with `-o html` flag
- **Severity levels**: Add `warn` threshold for domains (WARN status between min and warn)
- **Badge generation**: `coverctl badge` command generates SVG coverage badges
- **Coverage trend tracking**: `coverctl trend` and `coverctl record` for historical analysis
- **Threshold suggestions**: `coverctl suggest` recommends optimal thresholds
- **Coverage delta**: `--show-delta` flag displays coverage changes from history
- **Domain-specific excludes**: Per-domain `exclude` patterns for fine-grained control
- **Watch mode**: `coverctl watch` for continuous coverage on file changes
- **Coverage debt report**: `coverctl debt` shows coverage shortfall and remediation effort
- **Integration coverage**: Support for Go 1.20+ binary coverage with `GOCOVERDIR`
- **Profile merging**: Combine multiple coverage profiles for unified analysis
- **Diff-based checks**: Enforce coverage only on changed files with `diff.enabled`
- **File-level rules**: Per-file minimum thresholds with `files` config
- **Annotations**: `// coverctl:ignore` and `// coverctl:domain=NAME` pragmas

## [0.1.0] - 2024-01-01
- initial scaffolding of the CLI, DDD layers, report tooling, and documentation
- strict DDD/TDD guidance with coverage enforcement and Relicta release configuration
