package autodetect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/coverctl/internal/infrastructure/gotool"
)

func TestDetectDomains(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "internal", "policy"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, "cmd"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	domains := detectDomains(root)
	if len(domains) < 2 {
		t.Fatalf("expected multiple domains, got %d", len(domains))
	}
}

func TestSubdomainsFallback(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "internal"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	domains := subdomains(filepath.Join(root, "internal"))
	if len(domains) != 1 || domains[0].Name != "internal" {
		t.Fatalf("expected internal domain fallback")
	}
}

func TestSubdomainsIgnore(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"mocks", "generated", "policy"} {
		if err := os.MkdirAll(filepath.Join(root, "internal", name), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
	}
	domains := subdomains(filepath.Join(root, "internal"))
	if len(domains) != 1 || domains[0].Name != "policy" {
		t.Fatalf("expected policy to be the only domain")
	}
}

func TestDetectDomainsFallback(t *testing.T) {
	root := t.TempDir()
	domains := detectDomains(root)
	if len(domains) != 1 || domains[0].Name != "module" {
		t.Fatalf("expected module fallback")
	}
}

func TestDetectorDetect(t *testing.T) {
	cfg, err := Detector{Module: gotool.ModuleResolver{}}.Detect()
	if err != nil {
		t.Fatalf("detect: %v", err)
	}
	if len(cfg.Policy.Domains) == 0 {
		t.Fatalf("expected domains")
	}
}
