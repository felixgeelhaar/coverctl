package cli

// Version information, set at build time via ldflags
var (
	// Version is the semantic version (e.g., "1.2.3")
	Version = "dev"
	// Commit is the git commit SHA
	Commit = "unknown"
	// Date is the build date
	Date = "unknown"
)
