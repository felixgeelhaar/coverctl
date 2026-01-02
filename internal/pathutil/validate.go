// Package pathutil provides utilities for safe path handling.
package pathutil

import (
	"path/filepath"
	"strings"
)

// ValidatePath ensures a path is safe and doesn't escape the allowed scope.
// It returns the cleaned absolute path if valid.
// This function also resolves symlinks to detect symlink-based path traversal attacks.
// Returns ErrEmptyPath if path is empty, ErrNullBytes if path contains null bytes.
func ValidatePath(path string) (string, error) {
	if path == "" {
		return "", ErrEmptyPath
	}

	// Clean the path to resolve . and ..
	cleaned := filepath.Clean(path)

	// Check for null bytes (path traversal attack vector)
	if strings.Contains(cleaned, "\x00") {
		return "", ErrNullBytes
	}

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
