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
	content := "version: 1\npolicy:\n  default:\n    min: 75\n  domains:\n    - name: core\n      match: [\"./internal/core/...\"]\n      min: 85\nexclude:\n  - internal/generated/*\n"
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".coverctl.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	cfg, err := Loader{}.Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Version != 1 {
		t.Fatalf("expected version 1")
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
	if !strings.Contains(buf.String(), "version: 1") {
		t.Fatalf("expected version in output")
	}
	if !strings.Contains(buf.String(), "policy:") {
		t.Fatalf("expected policy block")
	}
}

func dummyConfig() application.Config {
	min := 85.0
	return application.Config{
		Version: 1,
		Policy: domain.Policy{
			DefaultMin: 80,
			Domains: []domain.Domain{{
				Name:  "core",
				Match: []string{"./internal/core/..."},
				Min:   &min,
			}},
		},
		Exclude: []string{"internal/generated/*"},
		Files:   []domain.FileRule{{Match: []string{"internal/core/*.go"}, Min: 85}},
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

func TestLoadUnsupportedVersion(t *testing.T) {
	content := "version: 2\npolicy:\n  default:\n    min: 75\n  domains:\n    - name: core\n      match: [\"./internal/core/...\"]\n"
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".coverctl.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if _, err := (Loader{}).Load(path); err == nil {
		t.Fatalf("expected version error")
	}
}

func TestLoadVersionZeroDefaultsToOne(t *testing.T) {
	// No version field should default to 1
	content := "policy:\n  default:\n    min: 75\n"
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".coverctl.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := (Loader{}).Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Version != 1 {
		t.Fatalf("expected version 1, got %d", cfg.Version)
	}
}

func TestLoadDiffBaseDefault(t *testing.T) {
	// When diff is enabled but base is empty, it should default to origin/main
	content := "version: 1\npolicy:\n  default:\n    min: 75\ndiff:\n  enabled: true\n"
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".coverctl.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := (Loader{}).Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !cfg.Diff.Enabled {
		t.Fatal("expected diff to be enabled")
	}
	if cfg.Diff.Base != "origin/main" {
		t.Fatalf("expected default base 'origin/main', got %q", cfg.Diff.Base)
	}
}

func TestLoadDiffBaseExplicit(t *testing.T) {
	// When diff base is explicitly set, it should be preserved
	content := "version: 1\npolicy:\n  default:\n    min: 75\ndiff:\n  enabled: true\n  base: origin/develop\n"
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".coverctl.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := (Loader{}).Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Diff.Base != "origin/develop" {
		t.Fatalf("expected base 'origin/develop', got %q", cfg.Diff.Base)
	}
}

func TestLoadDiffDisabledNoDefault(t *testing.T) {
	// When diff is disabled, base should not get a default
	content := "version: 1\npolicy:\n  default:\n    min: 75\ndiff:\n  enabled: false\n"
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".coverctl.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := (Loader{}).Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Diff.Base != "" {
		t.Fatalf("expected empty base when disabled, got %q", cfg.Diff.Base)
	}
}

func TestLoadIntegrationDefaults(t *testing.T) {
	// When integration is enabled, CoverDir and Profile should get defaults
	content := "version: 1\npolicy:\n  default:\n    min: 75\nintegration:\n  enabled: true\n  packages:\n    - ./cmd/...\n"
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".coverctl.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := (Loader{}).Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !cfg.Integration.Enabled {
		t.Fatal("expected integration to be enabled")
	}
	if cfg.Integration.CoverDir != filepath.Join(".cover", "integration") {
		t.Fatalf("expected default CoverDir '.cover/integration', got %q", cfg.Integration.CoverDir)
	}
	if cfg.Integration.Profile != filepath.Join(".cover", "integration.out") {
		t.Fatalf("expected default Profile '.cover/integration.out', got %q", cfg.Integration.Profile)
	}
}

func TestLoadIntegrationExplicitPaths(t *testing.T) {
	// When integration has explicit paths, they should be preserved
	content := "version: 1\npolicy:\n  default:\n    min: 75\nintegration:\n  enabled: true\n  packages:\n    - ./cmd/...\n  cover_dir: /tmp/cover\n  profile: /tmp/int.out\n"
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".coverctl.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := (Loader{}).Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Integration.CoverDir != "/tmp/cover" {
		t.Fatalf("expected CoverDir '/tmp/cover', got %q", cfg.Integration.CoverDir)
	}
	if cfg.Integration.Profile != "/tmp/int.out" {
		t.Fatalf("expected Profile '/tmp/int.out', got %q", cfg.Integration.Profile)
	}
}

func TestLoadIntegrationDisabledNoDefaults(t *testing.T) {
	// When integration is disabled, no defaults should be applied
	content := "version: 1\npolicy:\n  default:\n    min: 75\nintegration:\n  enabled: false\n"
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".coverctl.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := (Loader{}).Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if cfg.Integration.CoverDir != "" {
		t.Fatalf("expected empty CoverDir when disabled, got %q", cfg.Integration.CoverDir)
	}
	if cfg.Integration.Profile != "" {
		t.Fatalf("expected empty Profile when disabled, got %q", cfg.Integration.Profile)
	}
}

func TestLoadFileNotFound(t *testing.T) {
	_, err := (Loader{}).Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestLoadWithMergeProfiles(t *testing.T) {
	content := "version: 1\npolicy:\n  default:\n    min: 75\nmerge:\n  profiles:\n    - unit.out\n    - integration.out\n"
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".coverctl.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := (Loader{}).Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if len(cfg.Merge.Profiles) != 2 {
		t.Fatalf("expected 2 merge profiles, got %d", len(cfg.Merge.Profiles))
	}
	if cfg.Merge.Profiles[0] != "unit.out" {
		t.Fatalf("expected first profile 'unit.out', got %q", cfg.Merge.Profiles[0])
	}
}

func TestLoadWithAnnotations(t *testing.T) {
	content := "version: 1\npolicy:\n  default:\n    min: 75\nannotations:\n  enabled: true\n"
	tmp := t.TempDir()
	path := filepath.Join(tmp, ".coverctl.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := (Loader{}).Load(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if !cfg.Annotations.Enabled {
		t.Fatal("expected annotations to be enabled")
	}
}

func TestWriteWithVersion0DefaultsTo1(t *testing.T) {
	cfg := application.Config{
		Version: 0, // Should be written as version 1
		Policy: domain.Policy{
			DefaultMin: 80,
		},
	}
	var buf bytes.Buffer
	if err := Write(&buf, cfg); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !strings.Contains(buf.String(), "version: 1") {
		t.Fatalf("expected 'version: 1' in output, got:\n%s", buf.String())
	}
}

func TestWriteWithIntegration(t *testing.T) {
	cfg := application.Config{
		Version: 1,
		Policy: domain.Policy{
			DefaultMin: 80,
		},
		Integration: application.IntegrationConfig{
			Enabled:  true,
			Packages: []string{"./cmd/..."},
			RunArgs:  []string{"-v"},
			CoverDir: ".cover/int",
			Profile:  ".cover/int.out",
		},
	}
	var buf bytes.Buffer
	if err := Write(&buf, cfg); err != nil {
		t.Fatalf("write: %v", err)
	}
	content := buf.String()
	if !strings.Contains(content, "enabled: true") {
		t.Fatal("expected integration enabled in output")
	}
	if !strings.Contains(content, "cover_dir:") {
		t.Fatal("expected cover_dir in output")
	}
}
