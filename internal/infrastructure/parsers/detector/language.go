package detector

import (
	"os"
	"path/filepath"

	"github.com/felixgeelhaar/coverctl/internal/application"
)

// LanguageMarker represents a file that indicates a specific language.
type LanguageMarker struct {
	Filename string
	Language application.Language
	Priority int // Higher priority wins when multiple markers exist
}

// DefaultLanguageMarkers defines the default project file markers for language detection.
var DefaultLanguageMarkers = []LanguageMarker{
	// Go
	{Filename: "go.mod", Language: application.LanguageGo, Priority: 100},
	{Filename: "go.sum", Language: application.LanguageGo, Priority: 90},

	// Python
	{Filename: "pyproject.toml", Language: application.LanguagePython, Priority: 100},
	{Filename: "setup.py", Language: application.LanguagePython, Priority: 90},
	{Filename: "requirements.txt", Language: application.LanguagePython, Priority: 80},
	{Filename: "Pipfile", Language: application.LanguagePython, Priority: 85},
	{Filename: "poetry.lock", Language: application.LanguagePython, Priority: 85},

	// JavaScript/TypeScript
	{Filename: "package.json", Language: application.LanguageJavaScript, Priority: 90},
	{Filename: "tsconfig.json", Language: application.LanguageTypeScript, Priority: 100},
	{Filename: "yarn.lock", Language: application.LanguageJavaScript, Priority: 80},
	{Filename: "pnpm-lock.yaml", Language: application.LanguageJavaScript, Priority: 80},
	{Filename: "package-lock.json", Language: application.LanguageJavaScript, Priority: 80},

	// Java
	{Filename: "pom.xml", Language: application.LanguageJava, Priority: 100},
	{Filename: "build.gradle", Language: application.LanguageJava, Priority: 100},
	{Filename: "build.gradle.kts", Language: application.LanguageJava, Priority: 100},
	{Filename: "settings.gradle", Language: application.LanguageJava, Priority: 90},
	{Filename: "settings.gradle.kts", Language: application.LanguageJava, Priority: 90},

	// Rust
	{Filename: "Cargo.toml", Language: application.LanguageRust, Priority: 100},
	{Filename: "Cargo.lock", Language: application.LanguageRust, Priority: 90},
}

// DetectLanguage detects the primary programming language of a project.
// It searches for language-specific project files starting from the given directory.
func (d *Detector) DetectLanguage(projectDir string) (application.Language, error) {
	return d.DetectLanguageWithMarkers(projectDir, DefaultLanguageMarkers)
}

// DetectLanguageWithMarkers detects language using custom markers.
func (d *Detector) DetectLanguageWithMarkers(projectDir string, markers []LanguageMarker) (application.Language, error) {
	var bestMatch application.Language
	var bestPriority int

	// Search current directory and walk upward to find project root markers
	searchDirs := []string{projectDir}

	// Add parent directories up to a reasonable limit
	currentDir := projectDir
	for i := 0; i < 5; i++ {
		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break // Reached root
		}
		searchDirs = append(searchDirs, parent)
		currentDir = parent
	}

	for _, dir := range searchDirs {
		for _, marker := range markers {
			markerPath := filepath.Join(dir, marker.Filename)
			if _, err := os.Stat(markerPath); err == nil {
				// Found a marker file
				if marker.Priority > bestPriority {
					bestMatch = marker.Language
					bestPriority = marker.Priority
				}
			}
		}
		// If we found a high-priority match in the project dir, no need to search further
		if bestPriority >= 100 {
			break
		}
	}

	if bestMatch == "" {
		return application.LanguageAuto, nil
	}

	return bestMatch, nil
}

// GetDefaultProfilePaths returns common coverage profile paths for a language.
func (d *Detector) GetDefaultProfilePaths(lang application.Language) []string {
	switch lang {
	case application.LanguageGo:
		return []string{
			"coverage.out",
			"cover.out",
			"c.out",
		}
	case application.LanguagePython:
		return []string{
			"coverage.xml",    // pytest-cov with --cov-report=xml
			".coverage",       // coverage.py default
			"coverage.info",   // lcov format
			"htmlcov/",        // HTML report directory
			"coverage-report", // Common directory name
		}
	case application.LanguageJavaScript, application.LanguageTypeScript:
		return []string{
			"coverage/lcov.info",     // nyc/c8/jest default
			"coverage/coverage.json", // JSON format
			"coverage/cobertura.xml", // Cobertura format
			".nyc_output/",           // nyc intermediate files
		}
	case application.LanguageJava:
		return []string{
			"target/site/jacoco/jacoco.xml",                  // Maven JaCoCo
			"build/reports/jacoco/test/jacocoTestReport.xml", // Gradle JaCoCo
			"target/site/cobertura/coverage.xml",             // Maven Cobertura
			"build/reports/cobertura/coverage.xml",           // Gradle Cobertura
		}
	case application.LanguageRust:
		return []string{
			"target/coverage/lcov.info", // cargo-llvm-cov
			"target/coverage/cobertura.xml",
			"coverage/lcov.info",
		}
	default:
		return []string{}
	}
}

// GetDefaultFormat returns the default coverage format for a language.
func (d *Detector) GetDefaultFormat(lang application.Language) application.Format {
	switch lang {
	case application.LanguageGo:
		return application.FormatGo
	case application.LanguagePython:
		return application.FormatCobertura // coverage.py --xml is most common
	case application.LanguageJavaScript, application.LanguageTypeScript:
		return application.FormatLCOV // nyc/c8 default
	case application.LanguageJava:
		return application.FormatJaCoCo // JaCoCo is most common
	case application.LanguageRust:
		return application.FormatLCOV // cargo-llvm-cov default
	default:
		return application.FormatAuto
	}
}
