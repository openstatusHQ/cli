package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"os"
	"runtime"
	"sync/atomic"

	"github.com/openstatusHQ/cli/internal/version"
)

// Headers sent on every request so the API can attribute usage to CLI
// commands without the CLI making any extra call.
const (
	HeaderCLICommand    = "x-openstatus-cli-command"
	HeaderCLIInvocation = "x-openstatus-cli-invocation"
)

// userAgent identifies the CLI, e.g. "openstatus-cli/v1.3.2 (darwin; arm64)".
var userAgent = "openstatus-cli/" + version.Version + " (" + runtime.GOOS + "; " + runtime.GOARCH + ")"

var (
	command atomic.Pointer[string]
	// invocationID is random per process: it lets the API count one command
	// run once even when it makes several requests.
	invocationID = newInvocationID()
)

// SetCommand records the command being run, e.g. "monitors apply". Call it
// before the command makes any request.
func SetCommand(name string) { command.Store(&name) }

// UsageOptOut reports whether the user asked not to share command usage, via
// DO_NOT_TRACK or OPENSTATUS_NO_TELEMETRY.
func UsageOptOut() bool {
	return isSet(os.Getenv("DO_NOT_TRACK")) || isSet(os.Getenv("OPENSTATUS_NO_TELEMETRY"))
}

func isSet(v string) bool { return v != "" && v != "0" && v != "false" }

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
// carries the CLI user agent and, unless the user opted out, the command and
// invocation ID.
func NewTransport(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return cliTransport{base: base}
}

func (t cliTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// A RoundTripper must not modify the caller's request.
	req = req.Clone(req.Context())

	// Keep any existing agent (e.g. connect-go's) after ours.
	ua := userAgent
	if existing := req.Header.Get("User-Agent"); existing != "" {
		ua += " " + existing
	}
	req.Header.Set("User-Agent", ua)

	if !UsageOptOut() {
		req.Header.Set(HeaderCLIInvocation, invocationID)
		if name := command.Load(); name != nil && *name != "" {
			req.Header.Set(HeaderCLICommand, *name)
		}
	}
	return t.base.RoundTrip(req)
}
