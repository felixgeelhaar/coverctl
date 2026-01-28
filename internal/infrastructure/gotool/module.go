package gotool

import (
	"bytes"
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// ModuleInfo provides Go module information.
type ModuleInfo interface {
	ModuleRoot(ctx context.Context) (string, error)
	ModulePath(ctx context.Context) (string, error)
}

// ModuleResolver resolves Go module information.
type ModuleResolver struct{}

func (m ModuleResolver) ModuleRoot(ctx context.Context) (string, error) {
	cmd := exec.CommandContext(ctx, "go", "env", "GOMOD")
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	gomod := strings.TrimSpace(out.String())
	if gomod != "" && gomod != os.DevNull {
		return filepath.Dir(gomod), nil
	}

	// Fallback: search parent directories for go.mod or go.work
	return findModuleRoot()
}

// findModuleRoot searches current and parent directories for go.mod or go.work.
// This helps in monorepo scenarios where the current directory may be a subdirectory
// that isn't directly within a Go module, or when using Go workspaces.
func findModuleRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := cwd
	for {
		// Check for go.mod first (standard Go module)
		gomodPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(gomodPath); err == nil {
			return dir, nil
		}

		// Check for go.work (Go workspace)
		goworkPath := filepath.Join(dir, "go.work")
		if _, err := os.Stat(goworkPath); err == nil {
			return dir, nil
		}

		// Move to parent directory
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding go.mod or go.work
			return "", errors.New("module root not found: no go.mod or go.work in current or parent directories")
		}
		dir = parent
	}
}

func (m ModuleResolver) ModulePath(ctx context.Context) (string, error) {
	moduleRoot, err := m.ModuleRoot(ctx)
	if err != nil {
		return "", err
	}

	cmd := exec.CommandContext(ctx, "go", "list", "-m")
	cmd.Dir = moduleRoot
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Run(); err != nil {
		return "", err
	}
	// In a Go workspace (go.work), `go list -m` returns all module paths
	// separated by newlines. We only need the first (root) module path.
	modulePath := strings.TrimSpace(out.String())
	if modulePath == "" {
		return "", errors.New("module path not found")
	}
	if idx := strings.IndexByte(modulePath, '\n'); idx >= 0 {
		modulePath = strings.TrimSpace(modulePath[:idx])
	}
	return modulePath, nil
}

// CachedModuleResolver wraps ModuleResolver with caching.
// It caches the module root and path to avoid redundant subprocess calls.
type CachedModuleResolver struct {
	inner      ModuleResolver
	mu         sync.RWMutex
	rootCache  string
	pathCache  string
	rootErr    error
	pathErr    error
	rootCached bool
	pathCached bool
}

// NewCachedModuleResolver creates a new cached module resolver.
func NewCachedModuleResolver() *CachedModuleResolver {
	return &CachedModuleResolver{inner: ModuleResolver{}}
}

func (c *CachedModuleResolver) ModuleRoot(ctx context.Context) (string, error) {
	c.mu.RLock()
	if c.rootCached {
		root, err := c.rootCache, c.rootErr
		c.mu.RUnlock()
		return root, err
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if c.rootCached {
		return c.rootCache, c.rootErr
	}

	c.rootCache, c.rootErr = c.inner.ModuleRoot(ctx)
	c.rootCached = true
	return c.rootCache, c.rootErr
}

func (c *CachedModuleResolver) ModulePath(ctx context.Context) (string, error) {
	c.mu.RLock()
	if c.pathCached {
		path, err := c.pathCache, c.pathErr
		c.mu.RUnlock()
		return path, err
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if c.pathCached {
		return c.pathCache, c.pathErr
	}

	c.pathCache, c.pathErr = c.inner.ModulePath(ctx)
	c.pathCached = true
	return c.pathCache, c.pathErr
}

// Reset clears the cache, forcing fresh resolution on next call.
func (c *CachedModuleResolver) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.rootCached = false
	c.pathCached = false
	c.rootCache = ""
	c.pathCache = ""
	c.rootErr = nil
	c.pathErr = nil
}
