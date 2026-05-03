package mcp

import (
	"fmt"
	"regexp"
	"strings"
)

// SanitizationError indicates an MCP-supplied build flag was rejected.
//
// MCP input is downstream of LLM output, which is downstream of arbitrary
// untrusted text (PR descriptions, issue bodies, fetched web pages). A prompt
// injection attacker who controls any of those can ask an AI agent to call
// coverctl with malicious build flags. Many language test runners support
// flags that load arbitrary code: pytest --rootdir / --import-mode, gradle
// -I init.gradle, mvn -Dexec.executable=, npm --require, cargo --target-dir,
// etc. We allow-list-by-shape what the agent can hand to those tools.
type SanitizationError struct {
	Field  string
	Value  string
	Reason string
}

func (e *SanitizationError) Error() string {
	return fmt.Sprintf("rejected MCP input %s=%q: %s", e.Field, e.Value, e.Reason)
}

// dangerousLongFlags are long-form flag names that, across one or more
// supported language toolchains, allow loading arbitrary code or pivoting
// filesystem scope. Matched exactly, or with a `=value` suffix.
//
// Conservative-by-default: we reject these even if benign in some toolchains,
// because MCP input must be safe across every runner the registry might pick.
var dangerousLongFlags = []string{
	// pytest / coverage.py: load conftest/plugins from attacker path
	"--rootdir",
	"--import-mode",
	"--cov-config",
	"--cov-source",
	"--plugin",
	"--confcutdir",

	// generic config-path overrides used by many tools
	"--config",
	"--config-file",
	"--configfile",

	// gradle: init scripts
	"--init-script",
	"--define",

	// cargo: pivot manifest / target dir
	"--manifest-path",
	"--target-dir",

	// node: arbitrary module require / debugger attach / option injection
	"--require",
	"--node-options",
	"--inspect",
	"--inspect-brk",
	"--experimental-loader",
	"--loader",

	// generic exec / eval surfaces
	"--eval",
	"--exec",
	"--command",

	// dotnet / xunit equivalents
	"--runsettings",
	"--diag",
	"--blame-hang-dump-path",
	"--results-directory",

	// ruby / bundler
	"--gemfile",

	// file include/require styles
	"--include",
	"--require-from",
}

// dangerousShortFlagPrefixes are single-dash short flags whose value is
// concatenated directly (e.g. `-Dexec.executable=...` for mvn/java,
// `-Iscript.gradle` for gradle, `-Pkey=value` for gradle). These three
// short flags are exclusively associated with the JVM toolchain's code-exec
// surfaces; we don't extend this list to letters that collide with common
// Go/pytest test flags (-c, -p, -r, -e all have benign meanings that would
// produce false positives).
var dangerousShortFlagPrefixes = []string{
	"-D", // mvn / java / gradle system property — `-Dexec.executable=/bin/sh`
	"-I", // gradle init script — `-Iscript.gradle`
	"-P", // gradle project property — `-Pkey=value`
}

// tagsPattern accepts comma-separated identifiers used as Go build tags.
// Mirrors `go/build` constraint syntax: letters, digits, underscore.
var tagsPattern = regexp.MustCompile(`^[A-Za-z0-9_,]*$`)

// timeoutPattern accepts Go time.Duration syntax (e.g. "10m", "1h30s", "500ms").
var timeoutPattern = regexp.MustCompile(`^[0-9]+(ns|us|µs|ms|s|m|h)?([0-9]+(ns|us|µs|ms|s|m|h))*$`)

// shellMetaPattern matches shell metacharacters that signal an injection
// attempt in arg-shaped strings (testArgs). Args reach exec without a shell,
// but these characters have no legitimate purpose in argv and their presence
// is itself a signal.
var shellMetaPattern = regexp.MustCompile("[`$;|&><\n\r]")

// runShellMetaPattern is a permissive variant for test-name regex patterns,
// which legitimately contain `|` (alternation) and `<`/`>` (PCRE lookarounds).
// Still rejects unambiguous injection markers.
var runShellMetaPattern = regexp.MustCompile("[`;&\n\r]|\\$\\(")

// rejectionResponse builds the standard handler response for an input
// validation failure (sanitization or scope check). Centralised here so the
// shape is consistent across every MCP handler.
func rejectionResponse(err error) map[string]any {
	return map[string]any{
		"passed":  false,
		"error":   err.Error(),
		"summary": "Rejected unsafe MCP input",
	}
}

// SanitizeTestArgs validates a list of additional test runner arguments
// supplied via MCP input. Returns an error on the first dangerous arg.
//
// Caller is expected to discard the entire BuildFlags on error rather than
// passing through partially sanitized args.
func SanitizeTestArgs(args []string) error {
	for i, raw := range args {
		field := fmt.Sprintf("testArgs[%d]", i)

		if raw == "" {
			continue
		}
		if strings.ContainsAny(raw, "\x00\n\r") {
			return &SanitizationError{Field: field, Value: raw, Reason: "contains control characters"}
		}
		if shellMetaPattern.MatchString(raw) {
			return &SanitizationError{Field: field, Value: raw, Reason: "contains shell metacharacter"}
		}

		// Only inspect tokens that look like flags; positional args (e.g. test
		// pattern / package selector) pass through unchanged after the
		// metachar check above.
		if !strings.HasPrefix(raw, "-") {
			continue
		}

		// Long flags: split `--flag=value` into prefix for matching; covers
		// both `--flag value` (separate args) and `--flag=value`.
		flag := raw
		if eq := strings.IndexByte(raw, '='); eq != -1 {
			flag = raw[:eq]
		}

		if strings.HasPrefix(flag, "--") {
			for _, bad := range dangerousLongFlags {
				if flag == bad {
					return &SanitizationError{
						Field:  field,
						Value:  raw,
						Reason: fmt.Sprintf("flag %q can load arbitrary code via the underlying test runner; not allowed from MCP input", bad),
					}
				}
			}
			continue
		}

		// Short flags: value may be concatenated (e.g. `-Dexec.executable=…`,
		// `-Iscript.gradle`, `-rmodule`). Use prefix match on the raw arg.
		for _, bad := range dangerousShortFlagPrefixes {
			if strings.HasPrefix(raw, bad) {
				return &SanitizationError{
					Field:  field,
					Value:  raw,
					Reason: fmt.Sprintf("flag prefix %q can load arbitrary code via the underlying test runner; not allowed from MCP input", bad),
				}
			}
		}
	}
	return nil
}

// SanitizeTags validates a Go-style build tag string.
func SanitizeTags(tags string) error {
	if tags == "" {
		return nil
	}
	if !tagsPattern.MatchString(tags) {
		return &SanitizationError{Field: "tags", Value: tags, Reason: "build tags must be alphanumeric, underscore, comma"}
	}
	return nil
}

// SanitizeRunPattern validates a -run pattern (Go) / -k expression (pytest) /
// equivalent test-name filter. Allows regex syntax including `|` (alternation)
// and `<`/`>` (PCRE lookarounds); rejects unambiguous shell injection markers.
func SanitizeRunPattern(pattern string) error {
	if pattern == "" {
		return nil
	}
	if strings.ContainsAny(pattern, "\x00\n\r") {
		return &SanitizationError{Field: "run", Value: pattern, Reason: "contains control characters"}
	}
	if runShellMetaPattern.MatchString(pattern) {
		return &SanitizationError{Field: "run", Value: pattern, Reason: "contains shell metacharacter"}
	}
	return nil
}

// SanitizeTimeout validates a Go time.Duration string.
func SanitizeTimeout(timeout string) error {
	if timeout == "" {
		return nil
	}
	if !timeoutPattern.MatchString(timeout) {
		return &SanitizationError{Field: "timeout", Value: timeout, Reason: "must be Go duration syntax (e.g. 10m, 1h, 500ms)"}
	}
	return nil
}

// SanitizeBuildFlagsInput validates every untrusted build-flag field in one
// shot. Returns the first SanitizationError encountered.
func SanitizeBuildFlagsInput(tags, run, timeout string, testArgs []string) error {
	if err := SanitizeTags(tags); err != nil {
		return err
	}
	if err := SanitizeRunPattern(run); err != nil {
		return err
	}
	if err := SanitizeTimeout(timeout); err != nil {
		return err
	}
	if err := SanitizeTestArgs(testArgs); err != nil {
		return err
	}
	return nil
}
