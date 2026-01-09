# Product Requirements Document: Language-Agnostic Coverage Enforcement

## Overview

This document describes the product requirements for extending coverctl from a Go-only tool to a language-agnostic coverage enforcement platform.

**Vision:** Make coverctl the universal domain-aware coverage enforcement tool for any language ecosystem, enabling teams to apply consistent coverage policies across polyglot codebases.

---

## Problem Statement

### Current State

coverctl is tightly coupled to Go:
- Uses `go test -cover` for coverage generation
- Parses Go-specific coverage profile format
- Relies on `go.mod` and `go list` for project/package discovery
- Domain detection assumes Go package structure

### Market Opportunity

| Metric | Go-Only | Language-Agnostic |
|--------|---------|-------------------|
| Target Developers | ~2M Go developers | ~30M+ developers |
| Claude Code Plugin Market | ~5% | ~80% |
| Enterprise Appeal | Single-language teams | Polyglot organizations |

### User Pain Points

1. **Polyglot Teams:** Organizations using Go, Python, TypeScript need separate coverage tools per language
2. **Inconsistent Policies:** Each language ecosystem has different coverage tool conventions
3. **AI Integration Gap:** No universal coverage tool for AI-assisted development workflows
4. **Domain-Level Enforcement:** Most tools only support project-level thresholds, not domain/module-level

---

## Goals & Success Metrics

### Primary Goals

1. **G1:** Support coverage analysis for the top 5 language ecosystems (Go, Python, TypeScript/JavaScript, Java, Rust)
2. **G2:** Maintain 100% backward compatibility with existing Go workflows
3. **G3:** Enable one-command setup for any supported language
4. **G4:** Publish as a Claude Code plugin with universal appeal

### Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Languages Supported | 5+ | Format parsers + runners |
| Plugin Installs | 1,000+ in first 3 months | Marketplace analytics |
| Go User Retention | 100% | No breaking changes |
| Setup Time | < 2 minutes | Time from install to first check |
| GitHub Stars | +500 | Community adoption signal |

---

## User Personas

### Persona 1: Go Developer (Existing User)
- **Name:** Alex
- **Role:** Backend Engineer
- **Current Usage:** Uses coverctl for Go projects
- **Expectation:** Everything keeps working exactly as before
- **New Value:** Can now use same tool for Python scripts in repo

### Persona 2: Python/TypeScript Developer (New User)
- **Name:** Sam
- **Role:** Full-Stack Developer
- **Pain Point:** No domain-aware coverage tool for Python/TS
- **Expectation:** Simple setup, understands project structure
- **Value Prop:** Enforces coverage at module/domain level, not just overall

### Persona 3: Platform Engineer (Enterprise)
- **Name:** Jordan
- **Role:** DevOps/Platform Team Lead
- **Pain Point:** Different coverage tools per language, inconsistent policies
- **Expectation:** One tool, one config format, works everywhere
- **Value Prop:** Standardize coverage policies across all repos

### Persona 4: AI-Assisted Developer (Claude Code User)
- **Name:** Taylor
- **Role:** Developer using Claude Code
- **Pain Point:** Wants AI to help enforce coverage during development
- **Expectation:** Plugin works for any language project
- **Value Prop:** Universal coverage assistant in Claude Code

---

## Feature Requirements

### Phase 1: Multi-Format Profile Analysis (MVP)

**Priority:** P0 (Must Have)
**Effort:** ~5 days

#### FR1.1: LCOV Format Parser
Support parsing LCOV format (`lcov.info`, `coverage.lcov`):
- Used by: pytest-cov, nyc/c8, Jest, Ruby, PHP, GCC/LLVM
- Parse `SF:`, `DA:`, `LF:`, `LH:` directives
- Map to internal `domain.CoverageStat` structure

#### FR1.2: Cobertura XML Parser
Support parsing Cobertura XML format:
- Used by: Java (Maven/Gradle), Python (coverage.py), .NET, many CI tools
- Parse `<package>`, `<class>`, `<line>` elements
- Handle both DTD versions (coverage.py vs Java)

#### FR1.3: Format Auto-Detection
Automatically detect coverage format from file:
- Sniff file headers (mode: for Go, `<?xml` for XML, `TN:` for LCOV)
- Use file extension as hint (`.out`, `.info`, `.xml`)
- Fall back to explicit `format:` config field

#### FR1.4: Configuration Extension
Extend `.coverctl.yaml` schema:
```yaml
version: 2
language: auto  # or: go, python, typescript, java, rust
profile:
  format: auto  # or: go, lcov, cobertura, jacoco
  path: coverage.out
```

#### FR1.5: Language Auto-Detection
Detect project language from markers:
- Go: `go.mod`
- Python: `pyproject.toml`, `setup.py`, `requirements.txt`
- TypeScript/JS: `package.json`, `tsconfig.json`
- Java: `pom.xml`, `build.gradle`
- Rust: `Cargo.toml`

### Phase 2: Language-Specific Runners (Optional)

**Priority:** P1 (Should Have)
**Effort:** ~10 days

#### FR2.1: Runner Interface
Define abstract runner interface:
```go
type CoverageRunner interface {
    Run(ctx context.Context, opts RunOptions) (profilePath string, err error)
    Name() string
    Detect() bool
}
```

#### FR2.2: Python Runner
Execute `pytest --cov` with appropriate flags:
- Auto-detect pytest-cov or coverage.py
- Generate LCOV or XML output
- Pass through test patterns and markers

#### FR2.3: Node.js Runner
Execute coverage tools:
- Support nyc, c8, Jest --coverage
- Auto-detect from package.json scripts
- Generate LCOV output

#### FR2.4: Rust Runner
Execute `cargo tarpaulin` or `cargo llvm-cov`:
- Generate LCOV output
- Handle workspace configurations

#### FR2.5: Java Runner
Execute Maven/Gradle with JaCoCo:
- `mvn jacoco:report` or `gradle jacocoTestReport`
- Parse JaCoCo XML output

### Phase 3: Claude Code Plugin

**Priority:** P0 (Must Have)
**Effort:** ~3 days

#### FR3.1: Plugin Manifest
Create `.claude-plugin/plugin.json`:
```json
{
  "name": "coverctl",
  "description": "Universal domain-aware coverage enforcement",
  "keywords": ["coverage", "testing", "go", "python", "typescript", "java", "rust"]
}
```

#### FR3.2: Slash Commands
- `/coverctl:check` - Run coverage check
- `/coverctl:report` - Analyze existing profile
- `/coverctl:suggest` - Get threshold recommendations

#### FR3.3: Skills
- `coverage-enforcement` - Auto-invoke during TDD
- `coverage-review` - Activate during PR reviews

#### FR3.4: MCP Integration
Bundle existing MCP server within plugin.

---

## Non-Functional Requirements

### NFR1: Backward Compatibility
- All existing Go workflows MUST continue working unchanged
- Existing `.coverctl.yaml` files (version 1) MUST be supported
- CLI commands and flags MUST remain stable

### NFR2: Performance
- Profile parsing: < 100ms for 10,000 files
- Format detection: < 10ms
- Language detection: < 50ms

### NFR3: Error Messages
- Clear error when unsupported format detected
- Helpful suggestions for missing dependencies
- Language-specific troubleshooting guidance

### NFR4: Documentation
- README with examples for each language
- Language-specific quick-start guides
- Migration guide for existing users

---

## User Stories

### Epic 1: Profile Analysis (Phase 1)

```
US1.1: As a Python developer, I want to analyze my pytest-cov LCOV output
       so that I can enforce domain-level coverage policies.

US1.2: As a Java developer, I want to analyze my JaCoCo XML report
       so that I can enforce coverage thresholds per package.

US1.3: As a polyglot developer, I want coverctl to auto-detect my coverage format
       so that I don't need to specify it manually.

US1.4: As an existing Go user, I want my workflow to remain unchanged
       so that I don't need to update any scripts or configs.
```

### Epic 2: Test Runners (Phase 2)

```
US2.1: As a Python developer, I want coverctl to run pytest with coverage
       so that I have a single command for enforcement.

US2.2: As a TypeScript developer, I want coverctl to run my coverage tool
       so that I don't need to manage multiple commands.

US2.3: As a CI engineer, I want coverctl to work with any language in my monorepo
       so that I can standardize my pipeline.
```

### Epic 3: Plugin Distribution (Phase 3)

```
US3.1: As a Claude Code user, I want to install coverctl with one command
       so that I can quickly add coverage enforcement.

US3.2: As a developer, I want coverctl to activate automatically during TDD
       so that I get continuous coverage feedback.

US3.3: As a team lead, I want to share coverctl configuration via plugin
       so that my team has consistent coverage policies.
```

---

## Acceptance Criteria

### Phase 1 MVP

- [ ] `coverctl report --profile lcov.info` works for LCOV files
- [ ] `coverctl report --profile coverage.xml` works for Cobertura XML
- [ ] `coverctl init` detects Python/TypeScript/Java/Rust projects
- [ ] Existing Go workflows pass all regression tests
- [ ] Config schema supports `language` and `profile.format` fields
- [ ] Error messages guide users to correct format/configuration

### Phase 2 Runners

- [ ] `coverctl check` runs `pytest --cov` for Python projects
- [ ] `coverctl check` runs `npm test -- --coverage` for Node.js
- [ ] `coverctl check` runs `cargo tarpaulin` for Rust
- [ ] Runner auto-detection works based on project markers
- [ ] All runners produce analyzable coverage profiles

### Phase 3 Plugin

- [ ] Plugin installable via `/plugin install coverctl`
- [ ] Slash commands work for all supported languages
- [ ] Skills activate appropriately based on context
- [ ] Plugin listed in Claude Code marketplace

---

## Risks & Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Breaking Go compatibility | High | Low | Extensive regression testing |
| Coverage format variations | Medium | High | Test with real-world samples |
| Runner dependency issues | Medium | Medium | Document prerequisites clearly |
| Plugin rejection | Medium | Low | Follow Anthropic guidelines |
| Scope creep to more languages | Low | High | Strict phase boundaries |

---

## Timeline

| Phase | Duration | Deliverables |
|-------|----------|--------------|
| Phase 1: Profile Analysis | 1 week | LCOV + Cobertura parsers, auto-detection |
| Phase 2: Runners | 2 weeks | Python, Node.js, Rust, Java runners |
| Phase 3: Plugin | 1 week | Claude Code plugin, marketplace submission |

**Total:** ~4 weeks for full implementation

---

## Out of Scope

1. **IDE Plugins:** VS Code, JetBrains extensions (future consideration)
2. **Custom Format Support:** User-defined format parsers
3. **Remote Coverage:** Fetching coverage from CI systems
4. **Coverage Visualization:** Web UI dashboard (use existing HTML report)
5. **Proprietary Formats:** Coveralls, Codecov native formats

---

## Appendix

### Supported Coverage Formats

| Format | File Extensions | Languages |
|--------|-----------------|-----------|
| Go Coverage | `.out` | Go |
| LCOV | `.info`, `.lcov` | Python, JS/TS, Ruby, PHP, C/C++ |
| Cobertura XML | `.xml` | Java, Python, .NET |
| JaCoCo XML | `.xml` | Java, Kotlin |
| LLVM-cov JSON | `.json` | Rust, C/C++ |

### Competitive Analysis

| Tool | Languages | Domain-Aware | AI Integration | Plugin System |
|------|-----------|--------------|----------------|---------------|
| coverctl (current) | Go only | Yes | MCP | No |
| Codecov | Many | No | No | No |
| Coveralls | Many | No | No | No |
| SonarQube | Many | Partial | No | Yes |
| **coverctl (proposed)** | **Many** | **Yes** | **MCP + Plugin** | **Claude Code** |
