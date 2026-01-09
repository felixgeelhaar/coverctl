package application

import (
	"context"
	"fmt"
)

// WatchHandler handles watch mode operations.
type WatchHandler struct {
	ConfigLoader   ConfigLoader
	Autodetector   Autodetector
	DomainResolver DomainResolver
	CoverageRunner CoverageRunner
	RunnerRegistry RunnerRegistry
}

// Note: WatchCallback type is defined in service.go

// Watch runs coverage tests in a loop, re-running when source files change.
func (h *WatchHandler) Watch(ctx context.Context, opts WatchOptions, watcher FileWatcher, callback WatchCallback) error {
	moduleRoot, err := h.DomainResolver.ModuleRoot(ctx)
	if err != nil {
		return err
	}

	if err := watcher.WatchDir(moduleRoot); err != nil {
		return fmt.Errorf("failed to watch directory: %w", err)
	}

	checkHandler := &CheckHandler{
		ConfigLoader:   h.ConfigLoader,
		Autodetector:   h.Autodetector,
		DomainResolver: h.DomainResolver,
		CoverageRunner: h.CoverageRunner,
		RunnerRegistry: h.RunnerRegistry,
	}

	runOpts := RunOnlyOptions{
		ConfigPath: opts.ConfigPath,
		Profile:    opts.Profile,
		Domains:    opts.Domains,
		BuildFlags: opts.BuildFlags,
	}

	runNumber := 1
	runErr := checkHandler.RunOnly(ctx, runOpts)
	if callback != nil {
		callback(runNumber, runErr)
	}

	events := watcher.Events(ctx)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case _, ok := <-events:
			if !ok {
				return nil
			}
			runNumber++
			runErr := checkHandler.RunOnly(ctx, runOpts)
			if callback != nil {
				callback(runNumber, runErr)
			}
		}
	}
}
