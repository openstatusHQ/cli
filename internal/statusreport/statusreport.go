package statusreport

import (
	"fmt"
	"net/http"

	status_reportv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_report/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/status_report/v1/status_reportv1connect"
	"connectrpc.com/connect"
	"github.com/fatih/color"
	"github.com/openstatusHQ/cli/internal/api"
	"github.com/urfave/cli/v3"
)

func NewStatusReportClient(apiKey string) status_reportv1connect.StatusReportServiceClient {
	return status_reportv1connect.NewStatusReportServiceClient(
		http.DefaultClient,
		api.ConnectBaseURL,
		connect.WithInterceptors(api.NewAuthInterceptor(apiKey)),
		connect.WithProtoJSON(),
	)
}

func NewStatusReportClientWithHTTPClient(httpClient *http.Client, apiKey string) status_reportv1connect.StatusReportServiceClient {
	return status_reportv1connect.NewStatusReportServiceClient(
		httpClient,
		api.ConnectBaseURL,
		connect.WithInterceptors(api.NewAuthInterceptor(apiKey)),
		connect.WithProtoJSON(),
	)
}

func statusToSDK(s string) (status_reportv1.StatusReportStatus, error) {
	switch s {
	case "investigating":
		return status_reportv1.StatusReportStatus_STATUS_REPORT_STATUS_INVESTIGATING, nil
	case "identified":
		return status_reportv1.StatusReportStatus_STATUS_REPORT_STATUS_IDENTIFIED, nil
	case "monitoring":
		return status_reportv1.StatusReportStatus_STATUS_REPORT_STATUS_MONITORING, nil
	case "resolved":
		return status_reportv1.StatusReportStatus_STATUS_REPORT_STATUS_RESOLVED, nil
	default:
		return status_reportv1.StatusReportStatus_STATUS_REPORT_STATUS_UNSPECIFIED,
			fmt.Errorf("invalid status %q: must be one of investigating, identified, monitoring, resolved", s)
	}
}

func statusToString(s status_reportv1.StatusReportStatus) string {
	switch s {
	case status_reportv1.StatusReportStatus_STATUS_REPORT_STATUS_INVESTIGATING:
		return "investigating"
	case status_reportv1.StatusReportStatus_STATUS_REPORT_STATUS_IDENTIFIED:
		return "identified"
	case status_reportv1.StatusReportStatus_STATUS_REPORT_STATUS_MONITORING:
		return "monitoring"
	case status_reportv1.StatusReportStatus_STATUS_REPORT_STATUS_RESOLVED:
		return "resolved"
	default:
		return "unknown"
	}
}

func statusColor(s string) string {
	switch s {
	case "investigating":
		return color.RedString(s)
	case "identified":
		return color.YellowString(s)
	case "monitoring":
		return color.BlueString(s)
	case "resolved":
		return color.GreenString(s)
	default:
		return s
	}
}

func StatusReportCmd() *cli.Command {
	return &cli.Command{
		Name:    "status-report",
		Aliases: []string{"sr"},
		Usage:   "Manage status reports",
		Commands: []*cli.Command{
			GetStatusReportListCmd(),
			GetStatusReportInfoCmd(),
			GetStatusReportCreateCmd(),
			GetStatusReportUpdateCmd(),
			GetStatusReportDeleteCmd(),
			GetStatusReportAddUpdateCmd(),
		},
	}
}
