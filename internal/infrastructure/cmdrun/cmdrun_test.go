package cmdrun

import (
	"bytes"
	"context"
	"log/slog"
	"strings"
	"testing"
)

func TestExec_SuccessEmitsStartAndEnd(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	r := Runner{Logger: logger}

	if err := r.Exec(context.Background(), "", "true", nil); err != nil {
		t.Fatalf("expected success, got %v", err)
	}

	out := buf.String()
	if !strings.Contains(out, `"msg":"cmd start"`) {
		t.Errorf("missing start event: %s", out)
	}
	if !strings.Contains(out, `"msg":"cmd end"`) {
		t.Errorf("missing end event: %s", out)
	}
	if !strings.Contains(out, `"binary":"true"`) {
		t.Errorf("missing binary attr: %s", out)
	}
	if !strings.Contains(out, `"exit":0`) {
		t.Errorf("expected exit=0: %s", out)
	}
}

func TestExec_FailurePropagatesAndLogsExit(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	r := Runner{Logger: logger}

	err := r.Exec(context.Background(), "", "false", nil)
	if err == nil {
		t.Fatal("expected non-nil error from `false`")
	}
	if !strings.Contains(buf.String(), `"exit":1`) {
		t.Errorf("expected exit=1 in log, got %s", buf.String())
	}
}

func TestExec_UnresolvedBinaryFallsThroughExec(t *testing.T) {
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, &slog.HandlerOptions{Level: slog.LevelDebug}))
	r := Runner{Logger: logger}

	err := r.Exec(context.Background(), "", "definitely-not-a-real-binary-xyz123", nil)
	if err == nil {
		t.Fatal("expected error for nonexistent binary")
	}
}

func TestFingerprint_DeterministicAndShort(t *testing.T) {
	a := fingerprint([]string{"-v", "./..."})
	b := fingerprint([]string{"-v", "./..."})
	if a != b {
		t.Error("fingerprint must be deterministic")
	}
	if len(a) != 8 {
		t.Errorf("expected 8-char fingerprint, got %d", len(a))
	}
	c := fingerprint([]string{"-v", "./..", "extra"})
	if a == c {
		t.Error("fingerprint must change with different args")
	}
}

func TestJoinFingerprint(t *testing.T) {
	got := JoinFingerprint("python", []string{"-c", "x"})
	if !strings.HasPrefix(got, "python:") {
		t.Errorf("expected prefix 'python:', got %q", got)
	}
}
