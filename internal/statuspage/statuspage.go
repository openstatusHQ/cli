package statuspage

import (
	"net/http"

	status_pagev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_page/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/status_page/v1/status_pagev1connect"
	"connectrpc.com/connect"
	"github.com/openstatusHQ/cli/internal/api"
	"github.com/urfave/cli/v3"
)

func NewStatusPageClient(apiKey string) status_pagev1connect.StatusPageServiceClient {
	return status_pagev1connect.NewStatusPageServiceClient(
		api.DefaultHTTPClient,
		api.ConnectBaseURL,
		connect.WithInterceptors(api.NewAuthInterceptor(apiKey)),
		connect.WithProtoJSON(),
	)
}

func NewStatusPageClientWithHTTPClient(httpClient *http.Client, apiKey string) status_pagev1connect.StatusPageServiceClient {
	return status_pagev1connect.NewStatusPageServiceClient(
		httpClient,
		api.ConnectBaseURL,
		connect.WithInterceptors(api.NewAuthInterceptor(apiKey)),
		connect.WithProtoJSON(),
	)
}

func accessTypeToString(a status_pagev1.PageAccessType) string {
	switch a {
	case status_pagev1.PageAccessType_PAGE_ACCESS_TYPE_PUBLIC:
		return "public"
	case status_pagev1.PageAccessType_PAGE_ACCESS_TYPE_PASSWORD_PROTECTED:
		return "password-protected"
	case status_pagev1.PageAccessType_PAGE_ACCESS_TYPE_AUTHENTICATED:
		return "authenticated"
	default:
		return "unknown"
	}
}

func themeToString(t status_pagev1.PageTheme) string {
	switch t {
	case status_pagev1.PageTheme_PAGE_THEME_SYSTEM:
		return "system"
	case status_pagev1.PageTheme_PAGE_THEME_LIGHT:
		return "light"
	case status_pagev1.PageTheme_PAGE_THEME_DARK:
		return "dark"
	default:
		return "unknown"
	}
}

func componentTypeToString(t status_pagev1.PageComponentType) string {
	switch t {
	case status_pagev1.PageComponentType_PAGE_COMPONENT_TYPE_MONITOR:
		return "monitor"
	case status_pagev1.PageComponentType_PAGE_COMPONENT_TYPE_STATIC:
		return "static"
	default:
		return "unknown"
	}
}

func StatusPageCmd() *cli.Command {
	return &cli.Command{
		Name:    "status-page",
		Aliases: []string{"sp"},
		Usage:   "Manage status pages",
		Commands: []*cli.Command{
			GetStatusPageListCmd(),
			GetStatusPageInfoCmd(),
		},
	}
}
