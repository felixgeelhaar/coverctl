package runners

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/coverctl/internal/application"
)

// mockModuleInfo implements gotool.ModuleInfo for testing
type mockModuleInfo struct {
	root string
	path string
	err  error
}

func (m mockModuleInfo) ModuleRoot(ctx context.Context) (string, error) {
	return m.root, m.err
}

func (m mockModuleInfo) ModulePath(ctx context.Context) (string, error) {
	return m.path, m.err
}

func TestNewRegistry(t *testing.T) {
	module := mockModuleInfo{root: "/test", path: "example.com/test"}
	registry := NewRegistry(module)

	if registry == nil {
		t.Fatal("expected non-nil registry")
	}

	// Should have 5 runners: Go, Python, Node.js, Rust, Java
	if len(registry.runners) != 5 {
		t.Errorf("expected 5 runners, got %d", len(registry.runners))
	}
}

func TestRegistrySupportedLanguages(t *testing.T) {
	module := mockModuleInfo{root: "/test", path: "example.com/test"}
	registry := NewRegistry(module)

	langs := registry.SupportedLanguages()

	expected := map[application.Language]bool{
		application.LanguageGo:         true,
		application.LanguagePython:     true,
		application.LanguageJavaScript: true, // Node.js runner returns JavaScript
		application.LanguageRust:       true,
		application.LanguageJava:       true,
	}

	for _, lang := range langs {
		if !expected[lang] {
			t.Errorf("unexpected language: %s", lang)
		}
		delete(expected, lang)
	}

	if len(expected) > 0 {
		t.Errorf("missing languages: %v", expected)
	}
}

func TestRegistryGetRunner(t *testing.T) {
	module := mockModuleInfo{root: "/test", path: "example.com/test"}
	registry := NewRegistry(module)

	tests := []struct {
		lang     application.Language
		wantName string
		wantErr  bool
	}{
		{application.LanguageGo, "go", false},
		{application.LanguagePython, "python", false},
		{application.LanguageJavaScript, "nodejs", false},
		{application.LanguageRust, "rust", false},
		{application.LanguageJava, "java", false},
		{application.LanguageTypeScript, "", true}, // TypeScript maps to JavaScript runner
	}

	for _, tt := range tests {
		t.Run(string(tt.lang), func(t *testing.T) {
			runner, err := registry.GetRunner(tt.lang)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if runner.Name() != tt.wantName {
				t.Errorf("expected name %s, got %s", tt.wantName, runner.Name())
			}
		})
	}
}

func TestRegistryGetRunnerByName(t *testing.T) {
	module := mockModuleInfo{root: "/test", path: "example.com/test"}
	registry := NewRegistry(module)

	tests := []struct {
		name    string
		wantErr bool
	}{
		{"go", false},
		{"python", false},
		{"nodejs", false},
		{"rust", false},
		{"java", false},
		{"unknown", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner, err := registry.GetRunnerByName(tt.name)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}
			if runner.Name() != tt.name {
				t.Errorf("expected name %s, got %s", tt.name, runner.Name())
			}
		})
	}
}

func TestRegistryDetectRunner(t *testing.T) {
	// Create temp directories with language markers
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		files    []string
		wantLang application.Language
		wantName string
	}{
		{
			name:     "Go project",
			files:    []string{"go.mod"},
			wantLang: application.LanguageGo,
			wantName: "go",
		},
		{
			name:     "Python project",
			files:    []string{"pyproject.toml"},
			wantLang: application.LanguagePython,
			wantName: "python",
		},
		{
			name:     "Node.js project",
			files:    []string{"package.json"},
			wantLang: application.LanguageJavaScript,
			wantName: "nodejs",
		},
		{
			name:     "TypeScript project",
			files:    []string{"tsconfig.json", "package.json"},
			wantLang: application.LanguageJavaScript, // Node runner handles both
			wantName: "nodejs",
		},
		{
			name:     "Rust project",
			files:    []string{"Cargo.toml"},
			wantLang: application.LanguageRust,
			wantName: "rust",
		},
		{
			name:     "Maven project",
			files:    []string{"pom.xml"},
			wantLang: application.LanguageJava,
			wantName: "java",
		},
		{
			name:     "Gradle project",
			files:    []string{"build.gradle"},
			wantLang: application.LanguageJava,
			wantName: "java",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create project directory
			projectDir := filepath.Join(tmpDir, tt.name)
			if err := os.MkdirAll(projectDir, 0o755); err != nil {
				t.Fatal(err)
			}

			// Create marker files
			for _, file := range tt.files {
				path := filepath.Join(projectDir, file)
				if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
					t.Fatal(err)
				}
			}

			module := mockModuleInfo{root: projectDir, path: "example.com/test"}
			registry := NewRegistry(module)

			runner, err := registry.DetectRunner(projectDir)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if runner.Language() != tt.wantLang {
				t.Errorf("expected language %s, got %s", tt.wantLang, runner.Language())
			}
			if runner.Name() != tt.wantName {
				t.Errorf("expected name %s, got %s", tt.wantName, runner.Name())
			}
		})
	}
}

func TestRegistryDetectLanguage(t *testing.T) {
	tmpDir := t.TempDir()

	// Create a Go project
	goDir := filepath.Join(tmpDir, "go-project")
	if err := os.MkdirAll(goDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(goDir, "go.mod"), []byte("module test"), 0o644); err != nil {
		t.Fatal(err)
	}

	module := mockModuleInfo{root: goDir, path: "test"}
	registry := NewRegistry(module)

	lang := registry.DetectLanguage(goDir)
	if lang != application.LanguageGo {
		t.Errorf("expected Go, got %s", lang)
	}
}

func TestRegistryName(t *testing.T) {
	module := mockModuleInfo{root: "/test", path: "example.com/test"}
	registry := NewRegistry(module)

	if registry.Name() != "auto" {
		t.Errorf("expected name 'auto', got '%s'", registry.Name())
	}
}

func TestRegistryLanguage(t *testing.T) {
	module := mockModuleInfo{root: "/test", path: "example.com/test"}
	registry := NewRegistry(module)

	if registry.Language() != application.LanguageAuto {
		t.Errorf("expected LanguageAuto, got %s", registry.Language())
	}
}

func TestRegistryWithOptions(t *testing.T) {
	module := mockModuleInfo{root: "/test", path: "example.com/test"}

	// Test WithProjectDir option
	registry := NewRegistry(module, WithProjectDir("/custom/dir"))
	if registry.projectDir != "/custom/dir" {
		t.Errorf("expected projectDir '/custom/dir', got '%s'", registry.projectDir)
	}
}
