# coverctl

**Coverage feedback for AI coding agents — every language, every change.**

coverctl gives Claude Code, Cursor, Cline, Aider and any MCP-capable AI coding agent inline coverage signal during the edit loop: which domains regressed, which functions are uncovered, what to test next. Domain-aware policy, fifteen languages, no SaaS account, no source upload.

![MCP](https://img.shields.io/badge/MCP-server-blueviolet) ![Languages](https://img.shields.io/badge/languages-15-blue) ![Releases](https://img.shields.io/github/v/release/felixgeelhaar/coverctl?label=release)

## Why this exists

AI coding agents write code blind to coverage. They edit, you commit, the regression surfaces in CI minutes or hours later — too late to course-correct in the same session. Existing coverage tools (Codecov, Coveralls, native `go test -cover`) target humans reading dashboards or PR comments, not agents reasoning inline.

coverctl is built for the agent loop:

- **MCP-native.** First-class Model Context Protocol server. Every coverage capability — check, report, debt, suggest, compare — is an agent-callable tool.
- **Domain-aware.** Enforce stricter coverage on critical paths (`auth/`, `payment/`) than on utility code, declared once in `.coverctl.yaml`. Agent gets per-domain pass/fail, not just an overall percentage that hides gaps.
- **Multi-language by design.** Agents touch any language; coverage tooling must too. 15 languages: Go, Python, TS/JS, Java, Rust, C#, C/C++, PHP, Ruby, Swift, Dart, Scala, Elixir, Shell.
- **Local-first.** No SaaS account, no source-coverage upload, no third-party dependency in the agent's reach.
- **Hardened MCP surface.** Untrusted-input sanitization on every agent-supplied test argument blocks the prompt-injection → arbitrary-code-execution pivot through pytest/gradle/mvn/npm test runners.

## Quickstart for AI agents

### Claude Code

```bash
brew install felixgeelhaar/tap/coverctl
```

Add to `~/.config/claude-code/mcp.json`:

```json
{
  "mcpServers": {
    "coverctl": {
      "command": "coverctl",
      "args": ["mcp", "serve"]
    }
  }
}
```

Ask the agent: *"Run coverctl check and tell me which domains regressed."*

### Claude Desktop

`~/.config/claude/claude_desktop_config.json` (macOS/Linux) or `%APPDATA%\Claude\claude_desktop_config.json` (Windows):

```json
{
  "mcpServers": {
    "coverctl": {
      "command": "coverctl",
      "args": ["mcp", "serve"],
      "cwd": "/path/to/your/project"
    }
  }
}
```

### Cursor / Cline / other MCP clients

Any MCP-capable client works. Point it at `coverctl mcp serve` over stdio.

## MCP tool reference

| Tool | Purpose |
| --- | --- |
| `init` | Auto-detect project structure and create `.coverctl.yaml` with domain policies. |
| `check` | Run tests with coverage and enforce policy. Returns per-domain pass/fail, files, warnings. |
| `report` | Analyze an existing coverage profile without running tests. |
| `record` | Record current coverage to history for trend tracking. |
| `compare` | Compare two coverage profiles. Returns delta, improved/regressed files, domain changes. |
| `debt` | Coverage gap per domain — where to spend effort, ranked. |
| `suggest` | Recommend thresholds (`current` / `aggressive` / `conservative`). |
| `badge` | Generate SVG coverage badge. |
| `pr-comment` | Post coverage report to GitHub / GitLab / Bitbucket PR. |

### MCP resources (read-only context)

| URI | Content |
| --- | --- |
| `coverctl://debt` | Coverage debt as JSON. |
| `coverctl://trend` | Trend over recorded history. |
| `coverctl://suggest` | Threshold suggestions. |
| `coverctl://config` | Detected project config. |

### Security note for MCP users

MCP input is downstream of LLM output, which can be downstream of untrusted text (PR descriptions, fetched pages). coverctl rejects test-runner flags that allow arbitrary code loading (`--rootdir`, `--cov-config`, `-D`, `-I`, `--require`, `--init-script`, `--node-options`, etc.) when they come from MCP. CLI invocations from a human terminal are not sanitized — the human is the trust boundary there.

## Quickstart for humans

```bash
brew install felixgeelhaar/tap/coverctl

cd your-project
coverctl init      # auto-detect language + domains, write .coverctl.yaml
coverctl check     # enforce policy; exit 1 on violation
```

## CLI reference

The CLI is the substrate behind the MCP server; humans can use it directly.

| Command | Purpose |
| --- | --- |
| `init` / `i` | Interactive wizard, auto-detects language and domains. `--no-interactive` for CI. |
| `check` / `c` | Run coverage and enforce policy. `-o json` for machine output, `--fail-under N`, `--ratchet`, `--from-profile`. |
| `run` / `r` | Produce coverage artifacts without policy evaluation. |
| `watch` / `w` | Re-run coverage on file change during development. |
| `report` | Evaluate an existing profile. `-o html`, `--uncovered`, `--diff <ref>`, `--merge <profile>`. |
| `detect` | Auto-detect domains and write config. `--dry-run` to preview. |
| `badge` | SVG coverage badge. `--style flat-square`. |
| `compare` | Diff two profiles. |
| `debt` | Coverage debt report. |
| `trend` | Coverage trend from recorded history. |
| `record` | Append current coverage to history. `--commit`, `--branch` for CI. |
| `suggest` | Threshold suggestions. `--write-config` to apply. |
| `pr-comment` | Post coverage to GitHub/GitLab/Bitbucket PR. |
| `ignore` | Show configured excludes and tracked domains. |
| `mcp serve` | Start MCP server (stdio). |

Global flags: `-q/--quiet`, `--no-color`, `--ci` (combines quiet + GitHub Actions annotations).

### Test-execution flags

`check`, `run`, `record` accept toolchain flags forwarded to the underlying test runner:

| Flag | Example |
| --- | --- |
| `--tags` | `--tags integration,e2e` |
| `--race` | (Go race detector) |
| `--short` | Skip long-running tests |
| `-v` | Verbose test output |
| `--run` | `--run TestFoo` |
| `--timeout` | `--timeout 30m` |
| `--test-arg` | Repeatable: `--test-arg=-count=1 --test-arg=-parallel=4` |
| `--language` / `-l` | Override autodetection: `go`, `python`, `nodejs`, `rust`, `java`, ... |

## Configuration

`.coverctl.yaml` (schema: [`schemas/coverctl.schema.json`](schemas/coverctl.schema.json)):

```yaml
version: 1
policy:
  default:
    min: 75
  domains:
    - name: auth
      match: ["./internal/auth/..."]
      min: 90       # critical path — stricter
    - name: api
      match: ["./internal/api/..."]
      min: 80
    - name: utils
      match: ["./internal/utils/..."]
      # falls back to default min: 75
exclude:
  - internal/generated/*
```

Domain-aware enforcement is the point: overall coverage hides regressions in critical paths. coverctl evaluates each domain against its own minimum and fails the build if any domain falls below.

### Advanced

```yaml
files:
  - match: ["internal/core/*.go"]
    min: 90                          # per-file overrides
diff:
  enabled: true
  base: origin/main                  # only enforce on changed files
integration:
  enabled: true                      # Go 1.20+ GOCOVERDIR integration tests
  packages: ["./internal/integration/..."]
  cover_dir: ".cover/integration"
  profile: ".cover/integration.out"
merge:
  profiles: [".cover/unit.out", ".cover/integration.out"]
annotations:
  enabled: true                      # // coverctl:ignore, // coverctl:domain=NAME
```

Multi-package monorepo? Use `extends:` for inherited policies.

Starting point: copy `templates/coverctl.yaml`.

## Supported languages

| Language | Format | Detection markers |
| --- | --- | --- |
| Go | Native cover profile | `go.mod`, `go.sum` |
| Python | Cobertura, LCOV | `pyproject.toml`, `setup.py`, `requirements.txt` |
| TypeScript / JavaScript | LCOV | `tsconfig.json`, `package.json` |
| Java | JaCoCo, Cobertura | `pom.xml`, `build.gradle` |
| Rust | LCOV (cargo-llvm-cov) | `Cargo.toml` |
| C# / .NET | Cobertura (coverlet) | `*.csproj`, `*.sln` |
| C / C++ | LCOV (gcov/lcov) | `CMakeLists.txt`, `meson.build` |
| PHP | Cobertura (PHPUnit) | `composer.json`, `phpunit.xml` |
| Ruby | LCOV (SimpleCov) | `Gemfile`, `Rakefile` |
| Swift | LCOV (llvm-cov) | `Package.swift` |
| Dart | LCOV (dart test) | `pubspec.yaml` |
| Scala | Cobertura (scoverage) | `build.sbt` |
| Elixir | LCOV (mix test) | `mix.exs` |
| Shell | Cobertura (kcov) | `*.bats` |

## GitHub Action

```yaml
jobs:
  coverage:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - uses: actions/setup-go@v6
        with:
          go-version: "1.25"
      - uses: ./.github/actions/coverctl
        with:
          command: check
          config: .coverctl.yaml
          output: text
```

## Architecture

- `internal/domain` — coverage stats, policy evaluation, value objects.
- `internal/application` — orchestration: check, run, report, detect, record, compare, debt, suggest.
- `internal/infrastructure` — runners (15 languages), parsers (Go/LCOV/Cobertura/JaCoCo), config, history, PR clients (GitHub/GitLab/Bitbucket).
- `internal/cli` — CLI parsing, output formatters.
- `internal/mcp` — MCP server, input sanitization, tool/resource handlers.

Strict DDD: dependencies point inward. Domain knows nothing of CLI, MCP, or infrastructure.

## Contributing

- TDD: tests before behavior changes.
- Coverage ≥80% (`go test ./... -cover`).
- Conventional Commits (`feat:`, `fix:`, `chore:`, ...) for Relicta version-bump logic.
- `main` is protected; merge via PR after CI green (`.github/workflows/go.yml`).
- Run `gofmt -w` and `golangci-lint v2` before pushing.

## Releases

Managed by [Relicta](https://github.com/felixgeelhaar/relicta). Do not push `v*` tags manually.

## Security

See [SECURITY.md](SECURITY.md) for disclosure policy. MCP-input sanitization (`internal/mcp/sanitize.go`) is the primary defense against prompt-injection-driven argument attacks; report bypasses privately.
