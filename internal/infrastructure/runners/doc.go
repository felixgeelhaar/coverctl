// Package runners provides language-specific coverage test runners.
//
// The runners package implements the CoverageRunner interface for multiple
// programming languages, enabling coverctl to run coverage tests in any
// supported language ecosystem.
//
// Supported Languages:
//   - Go: Uses `go test -cover` (original functionality)
//   - Python: Uses pytest-cov or coverage.py
//   - Node.js/TypeScript: Uses nyc, c8, or Jest
//   - Rust: Uses cargo-tarpaulin or cargo-llvm-cov
//   - Java: Uses Maven/Gradle with JaCoCo
//
// The Registry provides auto-detection of project language and selects
// the appropriate runner automatically.
//
// Usage:
//
//	module := gotool.NewCachedModuleResolver()
//	registry := runners.NewRegistry(module)
//	runner, err := registry.DetectRunner(projectDir)
//	if err != nil {
//	    return err
//	}
//	profilePath, err := runner.Run(ctx, opts)
package runners
