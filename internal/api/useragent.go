package api

import (
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"net/url"
	"os"
	"runtime"
	"strconv"
	"strings"
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

// isSet treats empty and falsy values (0, false, no, off, any case) as unset,
// and anything else as an opt-out.
func isSet(v string) bool {
	v = strings.TrimSpace(v)
	if v == "" || strings.EqualFold(v, "no") || strings.EqualFold(v, "off") {
		return false
	}
	if b, err := strconv.ParseBool(v); err == nil {
		return b
	}
	return true
}

// sendsUsage reports whether a request to u may carry the command and
// invocation ID: only requests to the configured API, plus the hosted speed
// checker when the CLI targets openstatus Cloud. A self-hosted setup never
// sends them to openstatus.dev.
func sendsUsage(u *url.URL) bool {
	if UsageOptOut() {
		return false
	}
	if u.Host == hostOf(BaseURL) {
		return true
	}
	return BaseURL == DefaultBaseURL && u.Host == hostOf(PlayCheckerURL)
}

func hostOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return u.Host
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
// carries the CLI user agent and, when sendsUsage allows it, the command and
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

	if sendsUsage(req.URL) {
		req.Header.Set(HeaderCLIInvocation, invocationID)
		if name := command.Load(); name != nil && *name != "" {
			req.Header.Set(HeaderCLICommand, *name)
		}
	}
	return t.base.RoundTrip(req)
}
