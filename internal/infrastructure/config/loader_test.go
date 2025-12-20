package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/felixgeelhaar/coverctl/internal/application"
	"github.com/felixgeelhaar/coverctl/internal/domain"
)

func TestLoadConfig(t *testing.T) {
	content := "policy:\n  default:\n    min: 75\n  domains:\n    - name: core\n      match: [\"./internal/core/...\"]\n      min: 85\nexclude:\n  - internal/generated/*\n"
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".coverctl.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg, err := Loader{}.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Policy.DefaultMin != 75 {
		t.Fatalf("expected default min 75")
	}
	if len(cfg.Policy.Domains) != 1 {
		t.Fatalf("expected 1 domain")
	}
}

func TestWriteConfig(t *testing.T) {
	cfg := dummyConfig()
	var buf bytes.Buffer
	if err := Write(&buf, cfg); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !strings.Contains(buf.String(), "policy:") {
		t.Fatalf("expected policy block")
	}
}

func dummyConfig() application.Config {
	min := 85.0
	return application.Config{
		Policy: domain.Policy{
			DefaultMin: 80,
			Domains: []domain.Domain{{
				Name:  "core",
				Match: []string{"./internal/core/..."},
				Min:   &min,
			}},
		},
		Exclude: []string{"internal/generated/*"},
	}
}

func TestExistsMissing(t *testing.T) {
	ok, err := (Loader{}).Exists(filepath.Join(t.TempDir(), "missing.yaml"))
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if ok {
		t.Fatalf("expected missing to be false")
	}
}

func TestExistsPresent(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "config.yaml")
	if err := os.WriteFile(path, []byte("policy:\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	ok, err := (Loader{}).Exists(path)
	if err != nil {
		t.Fatalf("exists: %v", err)
	}
	if !ok {
		t.Fatalf("expected exists to be true")
	}
}

func TestLoadInvalidYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".coverctl.yaml")
	if err := os.WriteFile(path, []byte(":bad"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := (Loader{}).Load(path); err == nil {
		t.Fatalf("expected error")
	}
}
