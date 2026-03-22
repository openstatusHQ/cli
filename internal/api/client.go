package api

import (
	"context"

	"connectrpc.com/connect"
)

const APIBaseURL = "https://api.openstatus.dev/v1"

const ConnectBaseURL = "https://api.openstatus.dev/rpc"

func NewAuthInterceptor(apiKey string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("x-openstatus-key", apiKey)
			return next(ctx, req)
		}
	}
}
