# Repository Guidelines

## Project Structure & Module Organization
This repository contains product/design documentation in `docs/` and an initial Go module scaffold aligned with strict DDD:
- `cmd/coverctl/` for the CLI entrypoint (`main.go`).
- `internal/domain/` for entities/value objects and policy evaluation.
- `internal/application/` for use cases and orchestration.
- `internal/infrastructure/` for adapters (config, go tool integration, profile parsing, reporting, autodetect).
- `schemas/` for the configuration JSON schema and related assets.

## Build, Test, and Development Commands
Use standard Go tooling at the module root:
- `go test ./...` runs the full test suite.
- `go test -covermode=atomic -coverpkg=./... -coverprofile=.cover/coverage.out ./...` generates coverage artifacts the tool is expected to consume.
- `go test ./cmd/coverctl -run TestCLI` runs CLI-focused tests (if present).

## Coding Style & Naming Conventions
- Strict DDD: separate `domain` (entities, value objects, domain services), `application` (use cases), and `infrastructure` (CLI, IO, Go tool calls); never let infrastructure depend on application/domain.
- Model behavior in domain types (methods) instead of utility functions; keep invariants inside aggregates.
- Interfaces live in the domain or application layer; infrastructure provides implementations via dependency injection.
- Name packages after domain concepts (e.g., `policy`, `domains`, `report`) and keep cross-domain references explicit.
- Go formatting: run `gofmt -w` on all `.go` files; keep imports grouped by standard/third-party/internal.
- Package names should be short, lowercase, and match folder names.
- Filenames: tests must use `*_test.go`; golden files should live near tests (e.g., `testdata/`).

## Testing Guidelines
- Follow TDD: write or update tests before implementation changes.
- Use the standard `testing` package.
- Prefer table-driven tests for policy evaluation and domain resolution.
- Add golden-file tests for coverage parsing and report formatting (see `docs/tdd.md`).
- Maintain overall test coverage > 80%; add targeted tests for new logic or edge cases.

## Commit & Pull Request Guidelines
- No commit history is available in this repository, so no existing convention can be inferred.
- Suggested convention: Conventional Commits (e.g., `feat: add domain autodetection`).
- PRs should include: a clear description, linked issue (if any), and example output for CLI/report changes.

## Security & Configuration Notes
- The tool is expected to be deterministic and offline: no network calls and no telemetry.
- Configuration should be loaded from `.coverctl.yaml`; keep schema changes in `schemas/` aligned with code.
