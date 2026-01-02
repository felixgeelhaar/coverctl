package pathutil

import "errors"

// Sentinel errors for path validation.
var (
	// ErrEmptyPath is returned when an empty path is provided.
	ErrEmptyPath = errors.New("path cannot be empty")

	// ErrNullBytes is returned when a path contains null bytes.
	ErrNullBytes = errors.New("path contains null bytes")

	// ErrPathEscapesBase is returned when a path escapes the base directory.
	ErrPathEscapesBase = errors.New("path escapes base directory")
)
