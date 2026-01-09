package pathutil

import (
	"os"
	"path/filepath"
	"testing"
)

func TestValidatePath(t *testing.T) {
	// Create a temporary directory for testing symlinks
	tmpDir := t.TempDir()
	realFile := filepath.Join(tmpDir, "realfile.txt")
	if err := os.WriteFile(realFile, []byte("test"), 0o600); err != nil {
		t.Fatalf("create test file: %v", err)
	}

	symlinkPath := filepath.Join(tmpDir, "symlink.txt")
	if err := os.Symlink(realFile, symlinkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	tests := []struct {
		name    string
		path    string
		wantErr error
	}{
		{
			name:    "empty path returns ErrEmptyPath",
			path:    "",
			wantErr: ErrEmptyPath,
		},
		{
			name:    "path with null byte returns ErrNullBytes",
			path:    "some\x00path",
			wantErr: ErrNullBytes,
		},
		{
			name:    "valid relative path succeeds",
			path:    "some/valid/path.txt",
			wantErr: nil,
		},
		{
			name:    "valid absolute path succeeds",
			path:    "/some/valid/path.txt",
			wantErr: nil,
		},
		{
			name:    "path with dot-dot is cleaned",
			path:    "some/../valid/path.txt",
			wantErr: nil,
		},
		{
			name:    "path with single dot is cleaned",
			path:    "./some/./path.txt",
			wantErr: nil,
		},
		{
			name:    "symlink is resolved",
			path:    symlinkPath,
			wantErr: nil,
		},
		{
			name:    "non-existent path returns cleaned path",
			path:    filepath.Join(tmpDir, "nonexistent", "file.txt"),
			wantErr: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ValidatePath(tt.path)

			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Errorf("ValidatePath(%q) error = %v, wantErr %v", tt.path, err, tt.wantErr)
				}
				return
			}

			if err != nil {
				t.Errorf("ValidatePath(%q) unexpected error: %v", tt.path, err)
				return
			}

			if result == "" {
				t.Errorf("ValidatePath(%q) returned empty string", tt.path)
			}
		})
	}
}

func TestValidatePath_SymlinkResolution(t *testing.T) {
	tmpDir := t.TempDir()
	realFile := filepath.Join(tmpDir, "realfile.txt")
	if err := os.WriteFile(realFile, []byte("test"), 0o600); err != nil {
		t.Fatalf("create test file: %v", err)
	}

	symlinkPath := filepath.Join(tmpDir, "symlink.txt")
	if err := os.Symlink(realFile, symlinkPath); err != nil {
		t.Fatalf("create symlink: %v", err)
	}

	result, err := ValidatePath(symlinkPath)
	if err != nil {
		t.Fatalf("ValidatePath(%q) error: %v", symlinkPath, err)
	}

	// Result should be the resolved real path, not the symlink
	// Use EvalSymlinks on the expected path too to handle OS-level symlinks (e.g., /var -> /private/var on macOS)
	expectedPath, err := filepath.EvalSymlinks(realFile)
	if err != nil {
		t.Fatalf("EvalSymlinks(%q) error: %v", realFile, err)
	}

	if result != expectedPath {
		t.Errorf("ValidatePath(%q) = %q, want %q (resolved symlink)", symlinkPath, result, expectedPath)
	}
}

func TestValidatePath_NullByteVariants(t *testing.T) {
	nullPaths := []string{
		"\x00",
		"path\x00",
		"\x00path",
		"pa\x00th",
		"/some/\x00/path",
		"some/path\x00.txt",
	}

	for _, path := range nullPaths {
		t.Run("null_in_path", func(t *testing.T) {
			_, err := ValidatePath(path)
			if err != ErrNullBytes {
				t.Errorf("ValidatePath(%q) error = %v, want ErrNullBytes", path, err)
			}
		})
	}
}

func TestIsPathSafe(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "empty path is not safe",
			path: "",
			want: false,
		},
		{
			name: "null byte is not safe",
			path: "some\x00path",
			want: false,
		},
		{
			name: "valid relative path is safe",
			path: "some/valid/path.txt",
			want: true,
		},
		{
			name: "valid absolute path is safe",
			path: "/some/valid/path.txt",
			want: true,
		},
		{
			name: "parent traversal at start is not safe",
			path: "../some/path",
			want: false,
		},
		{
			name: "parent traversal after clean is not safe",
			path: "a/../../path",
			want: false,
		},
		{
			name: "contained parent traversal is safe",
			path: "some/../valid/path",
			want: true,
		},
		{
			name: "single dot path is safe",
			path: "./path",
			want: true,
		},
		{
			name: "current directory is safe",
			path: ".",
			want: true,
		},
		{
			name: "double slash path is safe",
			path: "some//path",
			want: true,
		},
		{
			name: "path with spaces is safe",
			path: "some path/with spaces.txt",
			want: true,
		},
		{
			name: "hidden file is safe",
			path: ".hidden/file.txt",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsPathSafe(tt.path); got != tt.want {
				t.Errorf("IsPathSafe(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestIsPathSafe_NullByteVariants(t *testing.T) {
	nullPaths := []string{
		"\x00",
		"path\x00",
		"\x00path",
		"pa\x00th",
		"/some/\x00/path",
		"some/path\x00.txt",
	}

	for _, path := range nullPaths {
		t.Run("null_in_path", func(t *testing.T) {
			if IsPathSafe(path) {
				t.Errorf("IsPathSafe(%q) = true, want false (contains null byte)", path)
			}
		})
	}
}

func TestValidatePath_CleanedPath(t *testing.T) {
	tests := []struct {
		input    string
		contains string // The cleaned path should contain this substring
	}{
		{
			input:    "some//path.txt",
			contains: "some",
		},
		{
			input:    "some/./path.txt",
			contains: "path.txt",
		},
		{
			input:    "some/other/../path.txt",
			contains: "path.txt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ValidatePath(tt.input)
			if err != nil {
				t.Fatalf("ValidatePath(%q) error: %v", tt.input, err)
			}

			// The path should not contain // or /./ after cleaning
			if filepath.Clean(result) != result {
				t.Errorf("ValidatePath(%q) returned uncleaned path: %q", tt.input, result)
			}
		})
	}
}
