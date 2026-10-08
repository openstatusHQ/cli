package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"runtime"
	"sync"
)

// Headers sent on every request so the API can attribute usage to CLI
// versions and commands without the CLI making any extra call.
const (
	HeaderCLICommand    = "x-openstatus-cli-command"
	HeaderCLIInvocation = "x-openstatus-cli-invocation"
)

var (
	invocationMu sync.RWMutex
	cliVersion   = "dev"
	cliCommand   string
	// invocationID is random per process: it lets the API count one command
	// run once even when it makes several requests.
	invocationID = newInvocationID()
)

// SetInvocation records the CLI version and the command being run, e.g.
// "monitors apply". Call it before the command makes any request.
func SetInvocation(version, command string) {
	invocationMu.Lock()
	defer invocationMu.Unlock()
	cliVersion, cliCommand = version, command
}

// UserAgent identifies the CLI, e.g. "openstatus-cli/v1.3.2 (darwin; arm64)".
func UserAgent() string {
	invocationMu.RLock()
	defer invocationMu.RUnlock()
	return userAgent(cliVersion)
}

func userAgent(version string) string {
	return "openstatus-cli/" + version + " (" + runtime.GOOS + "; " + runtime.GOARCH + ")"
}

func newInvocationID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// cliTransport tags outgoing requests with the CLI headers.
type cliTransport struct {
	base http.RoundTripper
}

// NewTransport wraps base (http.DefaultTransport when nil) so every request
// carries the CLI user agent, command and invocation ID.
func NewTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return cliTransport{base: base}
}

func (t cliTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	invocationMu.RLock()
	version, command := cliVersion, cliCommand
	invocationMu.RUnlock()

	// A RoundTripper must not modify the caller's request.
	req = req.Clone(req.Context())
	req.Header.Set("User-Agent", userAgent(version))
	req.Header.Set(HeaderCLIInvocation, invocationID)
	if command != "" {
		req.Header.Set(HeaderCLICommand, command)
	}
	return t.base.RoundTrip(req)
}
