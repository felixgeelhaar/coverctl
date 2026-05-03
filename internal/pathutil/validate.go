// Package pathutil provides utilities for safe path handling.
package pathutil

import (
	"path/filepath"
	"strings"
)

// ValidatePath performs basic safety checks (empty, null bytes) and returns a
// cleaned path. Resolves symlinks when the path exists.
//
// SCOPE WARNING: this function does NOT enforce a containment scope. An
// absolute path or a `..`-rich relative path will be returned cleaned but
// not rejected. Callers handling untrusted input (e.g. the MCP server, where
// inputs originate from LLM output downstream of arbitrary text) MUST use
// ValidateScopedPath instead.
func ValidatePath(path string) (string, error) {
	if path == "" {
		return "", ErrEmptyPath
	}

	// Reject null bytes BEFORE Clean — Clean preserves them and downstream
	// FS calls will error opaquely otherwise.
	if strings.Contains(path, "\x00") {
		return "", ErrNullBytes
	}

	// Clean the path to resolve . and ..
	cleaned := filepath.Clean(path)

	// Resolve symlinks to prevent symlink-based path traversal
	// Note: EvalSymlinks also cleans the path and returns absolute path if input exists
	realPath, err := filepath.EvalSymlinks(cleaned)
	if err != nil {
		// If the path doesn't exist yet, return the cleaned path
		// This allows creating new files in valid locations
		return cleaned, nil
	}

	return realPath, nil
}

// ValidateScopedPath enforces that path stays within root after cleaning and
// (where the path exists) symlink resolution. Returns the cleaned, absolute
// path inside root.
//
// Use this for any path supplied by an untrusted source: MCP tool input,
// config files originating from a freshly cloned repo, etc.
//
// Rules:
//   - Empty path → ErrEmptyPath
//   - Null bytes anywhere → ErrNullBytes
//   - Path starts with `~` → ErrPathEscapesBase (no shell expansion happens;
//     literal `~` rarely intended and a common attempt at home-dir pivot)
//   - Absolute path → ErrPathEscapesBase (callers must use root-relative paths)
//   - Cleaned path resolves outside root (via `..` or symlink) → ErrPathEscapesBase
func ValidateScopedPath(path, root string) (string, error) {
	if path == "" {
		return "", ErrEmptyPath
	}
	if strings.Contains(path, "\x00") {
		return "", ErrNullBytes
	}
	if strings.HasPrefix(path, "~") {
		return "", ErrPathEscapesBase
	}
	if filepath.IsAbs(path) {
		return "", ErrPathEscapesBase
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return "", err
	}
	// Resolve symlinks in the root itself so prefix comparison is stable
	// across systems where parts of root are symlinks (macOS /var → /private/var).
	if resolved, rErr := filepath.EvalSymlinks(absRoot); rErr == nil {
		absRoot = resolved
	}

	joined := filepath.Join(absRoot, path)
	cleaned := filepath.Clean(joined)

	// Symlink resolution: if the path exists, resolve and re-check containment.
	// If it doesn't exist (creating a new file), the textual prefix check below
	// is sufficient since Clean has already collapsed any `..` segments.
	if resolved, rErr := filepath.EvalSymlinks(cleaned); rErr == nil {
		cleaned = resolved
	}

	if !pathHasPrefix(cleaned, absRoot) {
		return "", ErrPathEscapesBase
	}
	return cleaned, nil
}

// pathHasPrefix reports whether path is equal to root or lies strictly within
// root, using a separator-aware comparison that won't match
// `/foo/barbaz` against root `/foo/bar`.
func pathHasPrefix(path, root string) bool {
	if path == root {
		return true
	}
	rootWithSep := root
	if !strings.HasSuffix(rootWithSep, string(filepath.Separator)) {
		rootWithSep += string(filepath.Separator)
	}
	return strings.HasPrefix(path, rootWithSep)
}

// IsPathSafe performs basic safety checks on a path.
// Returns true if the path appears safe for file operations.
func IsPathSafe(path string) bool {
	if path == "" {
		return false
	}

	// Check for null bytes
	if strings.Contains(path, "\x00") {
		return false
	}

	// Clean and check for obvious traversal patterns
	cleaned := filepath.Clean(path)

	// After cleaning, there shouldn't be any .. remaining that goes above start
	if strings.HasPrefix(cleaned, ".."+string(filepath.Separator)) {
		return false
	}

	return true
}
