// Package version holds the CLI release version, shared by the command tree
// and the API client's User-Agent.
package version

// Version is overridden at release time via
// -ldflags "-X github.com/openstatusHQ/cli/internal/version.Version=<tag>".
var Version = "v1.3.2"
