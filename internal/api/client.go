package api

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"time"

	"connectrpc.com/connect"

	output "github.com/openstatusHQ/cli/internal/cli"
)

const APIBaseURL = "https://api.openstatus.dev/v1"

const ConnectBaseURL = "https://api.openstatus.dev/rpc"

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
