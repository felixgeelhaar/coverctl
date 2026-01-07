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
	if gomod == "" || gomod == os.DevNull {
		return "", errors.New("module root not found")
	}
	return filepath.Dir(gomod), nil
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
	modulePath := strings.TrimSpace(out.String())
	if modulePath == "" {
		return "", errors.New("module path not found")
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
