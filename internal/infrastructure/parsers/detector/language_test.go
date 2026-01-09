package detector

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/felixgeelhaar/coverctl/internal/application"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDetector_DetectLanguage_Go(t *testing.T) {
	tmpdir := t.TempDir()
	createFile(t, tmpdir, "go.mod")

	detector := New()
	lang, err := detector.DetectLanguage(tmpdir)

	require.NoError(t, err)
	assert.Equal(t, application.LanguageGo, lang)
}

func TestDetector_DetectLanguage_Python_Pyproject(t *testing.T) {
	tmpdir := t.TempDir()
	createFile(t, tmpdir, "pyproject.toml")

	detector := New()
	lang, err := detector.DetectLanguage(tmpdir)

	require.NoError(t, err)
	assert.Equal(t, application.LanguagePython, lang)
}

func TestDetector_DetectLanguage_Python_Requirements(t *testing.T) {
	tmpdir := t.TempDir()
	createFile(t, tmpdir, "requirements.txt")

	detector := New()
	lang, err := detector.DetectLanguage(tmpdir)

	require.NoError(t, err)
	assert.Equal(t, application.LanguagePython, lang)
}

func TestDetector_DetectLanguage_JavaScript(t *testing.T) {
	tmpdir := t.TempDir()
	createFile(t, tmpdir, "package.json")

	detector := New()
	lang, err := detector.DetectLanguage(tmpdir)

	require.NoError(t, err)
	assert.Equal(t, application.LanguageJavaScript, lang)
}

func TestDetector_DetectLanguage_TypeScript(t *testing.T) {
	tmpdir := t.TempDir()
	createFile(t, tmpdir, "package.json")
	createFile(t, tmpdir, "tsconfig.json")

	detector := New()
	lang, err := detector.DetectLanguage(tmpdir)

	require.NoError(t, err)
	// TypeScript has higher priority than JavaScript
	assert.Equal(t, application.LanguageTypeScript, lang)
}

func TestDetector_DetectLanguage_Java_Maven(t *testing.T) {
	tmpdir := t.TempDir()
	createFile(t, tmpdir, "pom.xml")

	detector := New()
	lang, err := detector.DetectLanguage(tmpdir)

	require.NoError(t, err)
	assert.Equal(t, application.LanguageJava, lang)
}

func TestDetector_DetectLanguage_Java_Gradle(t *testing.T) {
	tmpdir := t.TempDir()
	createFile(t, tmpdir, "build.gradle")

	detector := New()
	lang, err := detector.DetectLanguage(tmpdir)

	require.NoError(t, err)
	assert.Equal(t, application.LanguageJava, lang)
}

func TestDetector_DetectLanguage_Java_GradleKts(t *testing.T) {
	tmpdir := t.TempDir()
	createFile(t, tmpdir, "build.gradle.kts")

	detector := New()
	lang, err := detector.DetectLanguage(tmpdir)

	require.NoError(t, err)
	assert.Equal(t, application.LanguageJava, lang)
}

func TestDetector_DetectLanguage_Rust(t *testing.T) {
	tmpdir := t.TempDir()
	createFile(t, tmpdir, "Cargo.toml")

	detector := New()
	lang, err := detector.DetectLanguage(tmpdir)

	require.NoError(t, err)
	assert.Equal(t, application.LanguageRust, lang)
}

func TestDetector_DetectLanguage_Unknown(t *testing.T) {
	tmpdir := t.TempDir()
	// No language markers

	detector := New()
	lang, err := detector.DetectLanguage(tmpdir)

	require.NoError(t, err)
	assert.Equal(t, application.LanguageAuto, lang)
}

func TestDetector_DetectLanguage_InParentDir(t *testing.T) {
	// Create parent with go.mod, child without
	tmpdir := t.TempDir()
	createFile(t, tmpdir, "go.mod")
	childDir := filepath.Join(tmpdir, "cmd", "myapp")
	require.NoError(t, os.MkdirAll(childDir, 0o755))

	detector := New()
	lang, err := detector.DetectLanguage(childDir)

	require.NoError(t, err)
	assert.Equal(t, application.LanguageGo, lang)
}

func TestDetector_DetectLanguage_PriorityWins(t *testing.T) {
	// Both go.sum (priority 90) and go.mod (priority 100)
	tmpdir := t.TempDir()
	createFile(t, tmpdir, "go.sum")
	createFile(t, tmpdir, "go.mod")

	detector := New()
	lang, err := detector.DetectLanguage(tmpdir)

	require.NoError(t, err)
	assert.Equal(t, application.LanguageGo, lang)
}

func TestDetector_GetDefaultProfilePaths_Go(t *testing.T) {
	detector := New()
	paths := detector.GetDefaultProfilePaths(application.LanguageGo)

	assert.Contains(t, paths, "coverage.out")
	assert.Contains(t, paths, "cover.out")
}

func TestDetector_GetDefaultProfilePaths_Python(t *testing.T) {
	detector := New()
	paths := detector.GetDefaultProfilePaths(application.LanguagePython)

	assert.Contains(t, paths, "coverage.xml")
	assert.Contains(t, paths, ".coverage")
}

func TestDetector_GetDefaultProfilePaths_JavaScript(t *testing.T) {
	detector := New()
	paths := detector.GetDefaultProfilePaths(application.LanguageJavaScript)

	assert.Contains(t, paths, "coverage/lcov.info")
}

func TestDetector_GetDefaultProfilePaths_Java(t *testing.T) {
	detector := New()
	paths := detector.GetDefaultProfilePaths(application.LanguageJava)

	assert.Contains(t, paths, "target/site/jacoco/jacoco.xml")
}

func TestDetector_GetDefaultProfilePaths_Rust(t *testing.T) {
	detector := New()
	paths := detector.GetDefaultProfilePaths(application.LanguageRust)

	assert.Contains(t, paths, "target/coverage/lcov.info")
}

func TestDetector_GetDefaultFormat_Go(t *testing.T) {
	detector := New()
	format := detector.GetDefaultFormat(application.LanguageGo)
	assert.Equal(t, application.FormatGo, format)
}

func TestDetector_GetDefaultFormat_Python(t *testing.T) {
	detector := New()
	format := detector.GetDefaultFormat(application.LanguagePython)
	assert.Equal(t, application.FormatCobertura, format)
}

func TestDetector_GetDefaultFormat_JavaScript(t *testing.T) {
	detector := New()
	format := detector.GetDefaultFormat(application.LanguageJavaScript)
	assert.Equal(t, application.FormatLCOV, format)
}

func TestDetector_GetDefaultFormat_TypeScript(t *testing.T) {
	detector := New()
	format := detector.GetDefaultFormat(application.LanguageTypeScript)
	assert.Equal(t, application.FormatLCOV, format)
}

func TestDetector_GetDefaultFormat_Java(t *testing.T) {
	detector := New()
	format := detector.GetDefaultFormat(application.LanguageJava)
	assert.Equal(t, application.FormatJaCoCo, format)
}

func TestDetector_GetDefaultFormat_Rust(t *testing.T) {
	detector := New()
	format := detector.GetDefaultFormat(application.LanguageRust)
	assert.Equal(t, application.FormatLCOV, format)
}

// createFile creates an empty file with the given name.
func createFile(t *testing.T, dir, name string) {
	t.Helper()
	path := filepath.Join(dir, name)
	err := os.WriteFile(path, []byte{}, 0o644)
	require.NoError(t, err)
}
