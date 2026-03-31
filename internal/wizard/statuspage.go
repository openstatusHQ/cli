package wizard

import (
	"context"

	status_pagev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_page/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/status_page/v1/status_pagev1connect"
	"connectrpc.com/connect"
	"github.com/openstatusHQ/cli/internal/api"
	output "github.com/openstatusHQ/cli/internal/cli"
)

func FetchStatusPages(ctx context.Context, apiKey string) ([]*status_pagev1.StatusPageSummary, error) {
	client := status_pagev1connect.NewStatusPageServiceClient(
		api.DefaultHTTPClient,
		api.ConnectBaseURL,
		connect.WithInterceptors(api.NewAuthInterceptor(apiKey)),
		connect.WithProtoJSON(),
	)
	resp, err := client.ListStatusPages(ctx, &status_pagev1.ListStatusPagesRequest{})
	if err != nil {
		return nil, output.FormatError(err, "status-page", "")
	}
	return resp.GetStatusPages(), nil
}

func FetchPageComponents(ctx context.Context, apiKey string, pageID string) ([]*status_pagev1.PageComponent, []*status_pagev1.PageComponentGroup, error) {
	client := status_pagev1connect.NewStatusPageServiceClient(
		api.DefaultHTTPClient,
		api.ConnectBaseURL,
		connect.WithInterceptors(api.NewAuthInterceptor(apiKey)),
		connect.WithProtoJSON(),
	)
	req := &status_pagev1.GetStatusPageContentRequest{}
	req.SetId(pageID)
	resp, err := client.GetStatusPageContent(ctx, req)
	if err != nil {
		return nil, nil, output.FormatError(err, "status-page", pageID)
	}
	return resp.GetComponents(), resp.GetGroups(), nil
}
