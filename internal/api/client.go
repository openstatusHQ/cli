package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"connectrpc.com/connect"

	output "github.com/openstatusHQ/cli/internal/cli"
)

// DefaultBaseURL is the openstatus Cloud API origin.
const DefaultBaseURL = "https://api.openstatus.dev"

// BaseURL is the API origin. Set OPENSTATUS_API_URL to target a self-hosted
// instance.
var BaseURL = resolveBaseURL(os.Getenv("OPENSTATUS_API_URL"))

var APIBaseURL = BaseURL + "/v1"

var ConnectBaseURL = BaseURL + "/rpc"

// resolveBaseURL also accepts a trailing /rpc, the form the Node SDK
// documents for the same variable.
func resolveBaseURL(v string) string {
	v = strings.TrimSuffix(strings.TrimRight(v, "/"), "/rpc")
	if v == "" {
		return DefaultBaseURL
	}
	return v
}

// PlayCheckerURL is the public Speed Checker endpoint backing the `check`
// command. The www. prefix is intentional: the bare openstatus.dev host
// returns a 308 redirect that adds latency to every call.
const PlayCheckerURL = "https://www.openstatus.dev/play/checker/api"

var DefaultHTTPClient = &http.Client{
	Timeout: 30 * time.Second,
}

func NewAuthInterceptor(apiKey string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("x-openstatus-key", apiKey)

			if output.IsDebug() {
				fmt.Fprintf(os.Stderr, "[debug] %s %s\n", req.HTTPMethod(), req.Spec().Procedure)
			}

			start := time.Now()
			resp, err := next(ctx, req)

			if output.IsDebug() {
				duration := time.Since(start)
				if err != nil {
					fmt.Fprintf(os.Stderr, "[debug] error after %s: %v\n", duration, err)
				} else {
					fmt.Fprintf(os.Stderr, "[debug] ok in %s\n", duration)
				}
			}

			return resp, err
		}
	}
}
