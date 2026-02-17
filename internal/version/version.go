// Package version holds build-time version info injected via ldflags.
//
// Build with:
//
//	go build -ldflags "-X github.com/NodeSpy/vop/internal/version.Version=1.0.0 \
//	  -X github.com/NodeSpy/vop/internal/version.Commit=abc1234 \
//	  -X github.com/NodeSpy/vop/internal/version.Date=2026-02-17"
package version

// These variables are set at build time via -ldflags.
var (
	Version = "dev"
	Commit  = "none"
	Date    = "unknown"
)

// String returns a formatted version string.
func String() string {
	return Version
}

// Full returns version with commit and date info.
func Full() string {
	return Version + " (commit " + Commit + ", built " + Date + ")"
}
