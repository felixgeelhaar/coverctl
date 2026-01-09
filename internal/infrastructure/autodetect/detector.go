package autodetect

import (
	"context"
	"os"
	"path/filepath"
	"sort"

	"github.com/felixgeelhaar/coverctl/internal/application"
	"github.com/felixgeelhaar/coverctl/internal/domain"
	"github.com/felixgeelhaar/coverctl/internal/infrastructure/gotool"
)

// Detector auto-detects project structure and generates coverage configuration.
// It supports multiple languages and uses language-specific detection patterns.
type Detector struct {
	Module   gotool.ModuleInfo
	Registry application.RunnerRegistry // Optional: for multi-language support
}

// Detect analyzes the project and generates a coverage configuration.
// For Go projects, it uses module-based detection.
// For other languages, it uses file-glob based detection.
func (d Detector) Detect() (application.Config, error) {
	// Determine project language
	lang := d.detectLanguage()

	// Use language-specific detection
	switch lang {
	case application.LanguageGo:
		return d.detectGo()
	case application.LanguagePython:
		return d.detectPython()
	case application.LanguageJavaScript, application.LanguageTypeScript:
		return d.detectJavaScript()
	case application.LanguageRust:
		return d.detectRust()
	case application.LanguageJava:
		return d.detectJava()
	default:
		// Fallback to Go detection for unknown languages
		return d.detectGo()
	}
}

// detectLanguage determines the project language.
func (d Detector) detectLanguage() application.Language {
	if d.Registry == nil {
		return application.LanguageGo
	}

	wd, err := os.Getwd()
	if err != nil {
		return application.LanguageGo
	}

	runner, err := d.Registry.DetectRunner(wd)
	if err != nil {
		return application.LanguageGo
	}

	return runner.Language()
}

// detectGo detects Go project structure.
func (d Detector) detectGo() (application.Config, error) {
	root, err := d.Module.ModuleRoot(contextBackground())
	if err != nil {
		return application.Config{}, err
	}

	domains := detectDomains(root)
	policy := domain.Policy{DefaultMin: 80, Domains: domains}
	return application.Config{Version: 1, Policy: policy, Language: application.LanguageGo}, nil
}

// detectPython detects Python project structure.
func (d Detector) detectPython() (application.Config, error) {
	wd, err := os.Getwd()
	if err != nil {
		return application.Config{}, err
	}

	domains := detectPythonDomains(wd)
	policy := domain.Policy{DefaultMin: 80, Domains: domains}
	return application.Config{Version: 1, Policy: policy, Language: application.LanguagePython}, nil
}

// detectJavaScript detects JavaScript/TypeScript project structure.
func (d Detector) detectJavaScript() (application.Config, error) {
	wd, err := os.Getwd()
	if err != nil {
		return application.Config{}, err
	}

	domains := detectJavaScriptDomains(wd)
	policy := domain.Policy{DefaultMin: 80, Domains: domains}
	return application.Config{Version: 1, Policy: policy, Language: application.LanguageJavaScript}, nil
}

// detectRust detects Rust project structure.
func (d Detector) detectRust() (application.Config, error) {
	wd, err := os.Getwd()
	if err != nil {
		return application.Config{}, err
	}

	domains := detectRustDomains(wd)
	policy := domain.Policy{DefaultMin: 80, Domains: domains}
	return application.Config{Version: 1, Policy: policy, Language: application.LanguageRust}, nil
}

// detectJava detects Java project structure.
func (d Detector) detectJava() (application.Config, error) {
	wd, err := os.Getwd()
	if err != nil {
		return application.Config{}, err
	}

	domains := detectJavaDomains(wd)
	policy := domain.Policy{DefaultMin: 80, Domains: domains}
	return application.Config{Version: 1, Policy: policy, Language: application.LanguageJava}, nil
}

func detectDomains(root string) []domain.Domain {
	var domains []domain.Domain
	top := []string{"cmd", "internal", "pkg"}
	for _, dir := range top {
		full := filepath.Join(root, dir)
		info, err := os.Stat(full)
		if err != nil || !info.IsDir() {
			continue
		}
		if dir == "internal" {
			domains = append(domains, subdomains(full)...)
			continue
		}
		domains = append(domains, domain.Domain{
			Name:  dir,
			Match: []string{"./" + dir + "/..."},
		})
	}
	if len(domains) == 0 {
		domains = append(domains, domain.Domain{Name: "module", Match: []string{"./..."}})
	}
	return domains
}

func subdomains(internalPath string) []domain.Domain {
	entries, err := os.ReadDir(internalPath)
	if err != nil {
		return []domain.Domain{{Name: "internal", Match: []string{"./internal/..."}}}
	}
	ignore := map[string]struct{}{"mocks": {}, "mock": {}, "generated": {}, "testdata": {}}
	out := make([]domain.Domain, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if _, ok := ignore[name]; ok {
			continue
		}
		out = append(out, domain.Domain{
			Name:  name,
			Match: []string{"./internal/" + name + "/..."},
		})
	}
	if len(out) == 0 {
		out = append(out, domain.Domain{Name: "internal", Match: []string{"./internal/..."}})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

func contextBackground() context.Context {
	return context.Background()
}

// detectPythonDomains detects Python project structure.
func detectPythonDomains(root string) []domain.Domain {
	var domains []domain.Domain

	// Common Python project directories
	pythonDirs := []string{"src", "lib", "app", "api", "core", "utils", "services", "models"}
	for _, dir := range pythonDirs {
		full := filepath.Join(root, dir)
		info, err := os.Stat(full)
		if err != nil || !info.IsDir() {
			continue
		}
		domains = append(domains, domain.Domain{
			Name:  dir,
			Match: []string{dir + "/**"},
		})
	}

	// Check for src layout (src/package_name)
	srcPath := filepath.Join(root, "src")
	if info, err := os.Stat(srcPath); err == nil && info.IsDir() {
		entries, _ := os.ReadDir(srcPath)
		for _, entry := range entries {
			if entry.IsDir() && !isIgnoredDir(entry.Name()) {
				domains = append(domains, domain.Domain{
					Name:  entry.Name(),
					Match: []string{"src/" + entry.Name() + "/**"},
				})
			}
		}
	}

	if len(domains) == 0 {
		// Fallback: use all Python files
		domains = append(domains, domain.Domain{Name: "project", Match: []string{"**/*.py"}})
	}

	return deduplicateDomains(domains)
}

// detectJavaScriptDomains detects JavaScript/TypeScript project structure.
func detectJavaScriptDomains(root string) []domain.Domain {
	var domains []domain.Domain

	// Common JS/TS project directories
	jsDirs := []string{"src", "lib", "app", "components", "pages", "api", "utils", "services", "hooks"}
	for _, dir := range jsDirs {
		full := filepath.Join(root, dir)
		info, err := os.Stat(full)
		if err != nil || !info.IsDir() {
			continue
		}
		domains = append(domains, domain.Domain{
			Name:  dir,
			Match: []string{dir + "/**"},
		})
	}

	if len(domains) == 0 {
		// Fallback: use all JS/TS files
		domains = append(domains, domain.Domain{Name: "project", Match: []string{"**/*.{js,jsx,ts,tsx}"}})
	}

	return domains
}

// detectRustDomains detects Rust project structure.
func detectRustDomains(root string) []domain.Domain {
	var domains []domain.Domain

	// Rust uses src directory with modules
	srcPath := filepath.Join(root, "src")
	if info, err := os.Stat(srcPath); err == nil && info.IsDir() {
		entries, _ := os.ReadDir(srcPath)
		for _, entry := range entries {
			if entry.IsDir() {
				domains = append(domains, domain.Domain{
					Name:  entry.Name(),
					Match: []string{"src/" + entry.Name() + "/**"},
				})
			}
		}
	}

	// Check for workspace members (Cargo.toml packages)
	if len(domains) == 0 {
		domains = append(domains, domain.Domain{Name: "crate", Match: []string{"src/**"}})
	}

	return domains
}

// detectJavaDomains detects Java project structure.
func detectJavaDomains(root string) []domain.Domain {
	var domains []domain.Domain

	// Maven/Gradle standard layout
	mainPath := filepath.Join(root, "src", "main", "java")
	if info, err := os.Stat(mainPath); err == nil && info.IsDir() {
		// Walk top-level packages
		entries, _ := os.ReadDir(mainPath)
		for _, entry := range entries {
			if entry.IsDir() {
				domains = append(domains, domain.Domain{
					Name:  entry.Name(),
					Match: []string{"src/main/java/" + entry.Name() + "/**"},
				})
			}
		}
	}

	// Android layout
	androidPath := filepath.Join(root, "app", "src", "main", "java")
	if info, err := os.Stat(androidPath); err == nil && info.IsDir() {
		domains = append(domains, domain.Domain{
			Name:  "app",
			Match: []string{"app/src/main/java/**"},
		})
	}

	if len(domains) == 0 {
		domains = append(domains, domain.Domain{Name: "project", Match: []string{"**/*.java"}})
	}

	return domains
}

// isIgnoredDir returns true if the directory should be ignored.
func isIgnoredDir(name string) bool {
	ignored := map[string]bool{
		"__pycache__":    true,
		".git":           true,
		"node_modules":   true,
		"venv":           true,
		".venv":          true,
		"env":            true,
		".env":           true,
		"target":         true,
		"build":          true,
		"dist":           true,
		".pytest_cache":  true,
		".mypy_cache":    true,
		"__pypackages__": true,
		".tox":           true,
		"eggs":           true,
		".eggs":          true,
	}
	return ignored[name]
}

// deduplicateDomains removes duplicate domains by name.
func deduplicateDomains(domains []domain.Domain) []domain.Domain {
	seen := make(map[string]bool)
	result := make([]domain.Domain, 0, len(domains))
	for _, d := range domains {
		if !seen[d.Name] {
			seen[d.Name] = true
			result = append(result, d)
		}
	}
	return result
}
