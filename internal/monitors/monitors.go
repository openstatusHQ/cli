package monitors

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	"buf.build/gen/go/openstatus/api/connectrpc/gosimple/openstatus/monitor/v1/monitorv1connect"
	"connectrpc.com/connect"
	"github.com/openstatusHQ/cli/internal/config"
	"github.com/urfave/cli/v3"
)

// APIBaseURL is the base URL for the OpenStatus API
const APIBaseURL = "https://api.openstatus.dev/v1"

// ConnectBaseURL is the base URL for the Connect RPC API
const ConnectBaseURL = "https://api.openstatus.dev/rpc"

// NewAuthInterceptor creates an interceptor that adds the API key to all requests
func NewAuthInterceptor(apiKey string) connect.UnaryInterceptorFunc {
	return func(next connect.UnaryFunc) connect.UnaryFunc {
		return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
			req.Header().Set("x-openstatus-key", apiKey)
			return next(ctx, req)
		}
	}
}

// NewMonitorClient creates a new Monitor service client with authentication
func NewMonitorClient(apiKey string) monitorv1connect.MonitorServiceClient {
	return monitorv1connect.NewMonitorServiceClient(
		http.DefaultClient,
		ConnectBaseURL,
		connect.WithInterceptors(NewAuthInterceptor(apiKey)),
		connect.WithProtoJSON(),
	)
}

// NewMonitorClientWithHTTPClient creates a new Monitor service client with a custom HTTP client
func NewMonitorClientWithHTTPClient(httpClient *http.Client, apiKey string) monitorv1connect.MonitorServiceClient {
	return monitorv1connect.NewMonitorServiceClient(
		httpClient,
		ConnectBaseURL,
		connect.WithInterceptors(NewAuthInterceptor(apiKey)),
		connect.WithProtoJSON(),
	)
}

// Helper functions to convert SDK types to CLI display types

// periodicityToString converts SDK Periodicity enum to string
func periodicityToString(p monitorv1.Periodicity) string {
	switch p {
	case monitorv1.Periodicity_PERIODICITY_30S:
		return "30s"
	case monitorv1.Periodicity_PERIODICITY_1M:
		return "1m"
	case monitorv1.Periodicity_PERIODICITY_5M:
		return "5m"
	case monitorv1.Periodicity_PERIODICITY_10M:
		return "10m"
	case monitorv1.Periodicity_PERIODICITY_30M:
		return "30m"
	case monitorv1.Periodicity_PERIODICITY_1H:
		return "1h"
	default:
		return "unknown"
	}
}

// httpMethodToString converts SDK HTTPMethod enum to string
func httpMethodToString(m monitorv1.HTTPMethod) string {
	switch m {
	case monitorv1.HTTPMethod_HTTP_METHOD_GET:
		return "GET"
	case monitorv1.HTTPMethod_HTTP_METHOD_POST:
		return "POST"
	case monitorv1.HTTPMethod_HTTP_METHOD_HEAD:
		return "HEAD"
	case monitorv1.HTTPMethod_HTTP_METHOD_PUT:
		return "PUT"
	case monitorv1.HTTPMethod_HTTP_METHOD_PATCH:
		return "PATCH"
	case monitorv1.HTTPMethod_HTTP_METHOD_DELETE:
		return "DELETE"
	case monitorv1.HTTPMethod_HTTP_METHOD_TRACE:
		return "TRACE"
	case monitorv1.HTTPMethod_HTTP_METHOD_CONNECT:
		return "CONNECT"
	case monitorv1.HTTPMethod_HTTP_METHOD_OPTIONS:
		return "OPTIONS"
	default:
		return ""
	}
}

// regionToString converts SDK Region enum to string
func regionToString(r monitorv1.Region) string {
	switch r {
	case monitorv1.Region_REGION_FLY_AMS:
		return "ams"
	case monitorv1.Region_REGION_FLY_ARN:
		return "arn"
	case monitorv1.Region_REGION_FLY_BOM:
		return "bom"
	case monitorv1.Region_REGION_FLY_CDG:
		return "cdg"
	case monitorv1.Region_REGION_FLY_DFW:
		return "dfw"
	case monitorv1.Region_REGION_FLY_EWR:
		return "ewr"
	case monitorv1.Region_REGION_FLY_FRA:
		return "fra"
	case monitorv1.Region_REGION_FLY_GRU:
		return "gru"
	case monitorv1.Region_REGION_FLY_IAD:
		return "iad"
	case monitorv1.Region_REGION_FLY_JNB:
		return "jnb"
	case monitorv1.Region_REGION_FLY_LAX:
		return "lax"
	case monitorv1.Region_REGION_FLY_LHR:
		return "lhr"
	case monitorv1.Region_REGION_FLY_NRT:
		return "nrt"
	case monitorv1.Region_REGION_FLY_ORD:
		return "ord"
	case monitorv1.Region_REGION_FLY_SJC:
		return "sjc"
	case monitorv1.Region_REGION_FLY_SIN:
		return "sin"
	case monitorv1.Region_REGION_FLY_SYD:
		return "syd"
	case monitorv1.Region_REGION_FLY_YYZ:
		return "yyz"
	default:
		return r.String()
	}
}

// regionsToStrings converts a slice of SDK Region enums to strings
func regionsToStrings(regions []monitorv1.Region) []string {
	result := make([]string, len(regions))
	for i, r := range regions {
		result[i] = r.String()
	}
	return result
}

// Inverse converter functions (config types → SDK types)

// stringToPeriodicity converts config.Frequency to SDK Periodicity
func stringToPeriodicity(f config.Frequency) monitorv1.Periodicity {
	switch f {
	case config.The30S:
		return monitorv1.Periodicity_PERIODICITY_30S
	case config.The1M:
		return monitorv1.Periodicity_PERIODICITY_1M
	case config.The5M:
		return monitorv1.Periodicity_PERIODICITY_5M
	case config.The10M:
		return monitorv1.Periodicity_PERIODICITY_10M
	case config.The30M:
		return monitorv1.Periodicity_PERIODICITY_30M
	case config.The1H:
		return monitorv1.Periodicity_PERIODICITY_1H
	default:
		return monitorv1.Periodicity_PERIODICITY_10M
	}
}

// stringToHTTPMethod converts config.Method to SDK HTTPMethod
func stringToHTTPMethod(m config.Method) monitorv1.HTTPMethod {
	switch m {
	case config.Get:
		return monitorv1.HTTPMethod_HTTP_METHOD_GET
	case config.Post:
		return monitorv1.HTTPMethod_HTTP_METHOD_POST
	case config.Put:
		return monitorv1.HTTPMethod_HTTP_METHOD_PUT
	case config.Patch:
		return monitorv1.HTTPMethod_HTTP_METHOD_PATCH
	case config.Delete:
		return monitorv1.HTTPMethod_HTTP_METHOD_DELETE
	case config.Head:
		return monitorv1.HTTPMethod_HTTP_METHOD_HEAD
	case config.Options:
		return monitorv1.HTTPMethod_HTTP_METHOD_OPTIONS
	default:
		return monitorv1.HTTPMethod_HTTP_METHOD_GET
	}
}

// stringToRegion converts config.Region to SDK Region
func stringToRegion(r config.Region) monitorv1.Region {
	switch r {
	case config.Ams:
		return monitorv1.Region_REGION_FLY_AMS
	case config.Arn:
		return monitorv1.Region_REGION_FLY_ARN
	case config.BOM:
		return monitorv1.Region_REGION_FLY_BOM
	case config.Cdg:
		return monitorv1.Region_REGION_FLY_CDG
	case config.Dfw:
		return monitorv1.Region_REGION_FLY_DFW
	case config.Ewr:
		return monitorv1.Region_REGION_FLY_EWR
	case config.Fra:
		return monitorv1.Region_REGION_FLY_FRA
	case config.Gru:
		return monitorv1.Region_REGION_FLY_GRU
	case config.Iad:
		return monitorv1.Region_REGION_FLY_IAD
	case config.Jnb:
		return monitorv1.Region_REGION_FLY_JNB
	case config.Lax:
		return monitorv1.Region_REGION_FLY_LAX
	case config.Lhr:
		return monitorv1.Region_REGION_FLY_LHR
	case config.Nrt:
		return monitorv1.Region_REGION_FLY_NRT
	case config.Ord:
		return monitorv1.Region_REGION_FLY_ORD
	case config.Sin:
		return monitorv1.Region_REGION_FLY_SIN
	case config.Sjc:
		return monitorv1.Region_REGION_FLY_SJC
	case config.Syd:
		return monitorv1.Region_REGION_FLY_SYD
	case config.Yyz:
		return monitorv1.Region_REGION_FLY_YYZ
	default:
		return monitorv1.Region_REGION_UNSPECIFIED
	}
}

// stringsToRegions converts []config.Region to []monitorv1.Region
func stringsToRegions(regions []config.Region) []monitorv1.Region {
	result := make([]monitorv1.Region, len(regions))
	for i, r := range regions {
		result[i] = stringToRegion(r)
	}
	return result
}

// configCompareToNumberComparator converts config.Compare to NumberComparator
func configCompareToNumberComparator(c config.Compare) monitorv1.NumberComparator {
	switch c {
	case config.Eq:
		return monitorv1.NumberComparator_NUMBER_COMPARATOR_EQUAL
	case config.NotEq:
		return monitorv1.NumberComparator_NUMBER_COMPARATOR_NOT_EQUAL
	case config.Gt:
		return monitorv1.NumberComparator_NUMBER_COMPARATOR_GREATER_THAN
	case config.Gte:
		return monitorv1.NumberComparator_NUMBER_COMPARATOR_GREATER_THAN_OR_EQUAL
	case config.Lt:
		return monitorv1.NumberComparator_NUMBER_COMPARATOR_LESS_THAN
	case config.LTE:
		return monitorv1.NumberComparator_NUMBER_COMPARATOR_LESS_THAN_OR_EQUAL
	default:
		return monitorv1.NumberComparator_NUMBER_COMPARATOR_EQUAL
	}
}

// configCompareToStringComparator converts config.Compare to StringComparator
func configCompareToStringComparator(c config.Compare) monitorv1.StringComparator {
	switch c {
	case config.Eq:
		return monitorv1.StringComparator_STRING_COMPARATOR_EQUAL
	case config.NotEq:
		return monitorv1.StringComparator_STRING_COMPARATOR_NOT_EQUAL
	case config.Contains:
		return monitorv1.StringComparator_STRING_COMPARATOR_CONTAINS
	case config.NotContains:
		return monitorv1.StringComparator_STRING_COMPARATOR_NOT_CONTAINS
	case config.Empty:
		return monitorv1.StringComparator_STRING_COMPARATOR_EMPTY
	case config.NotEmpty:
		return monitorv1.StringComparator_STRING_COMPARATOR_NOT_EMPTY
	case config.Gt:
		return monitorv1.StringComparator_STRING_COMPARATOR_GREATER_THAN
	case config.Gte:
		return monitorv1.StringComparator_STRING_COMPARATOR_GREATER_THAN_OR_EQUAL
	case config.Lt:
		return monitorv1.StringComparator_STRING_COMPARATOR_LESS_THAN
	case config.LTE:
		return monitorv1.StringComparator_STRING_COMPARATOR_LESS_THAN_OR_EQUAL
	default:
		return monitorv1.StringComparator_STRING_COMPARATOR_EQUAL
	}
}

// Builder functions (config.Monitor → SDK monitor types)

// configToHTTPMonitor converts config.Monitor to SDK HTTPMonitor
func configToHTTPMonitor(m config.Monitor) *monitorv1.HTTPMonitor {
	// Convert headers
	headers := make([]*monitorv1.Headers, 0, len(m.Request.Headers))
	for k, v := range m.Request.Headers {
		headers = append(headers, &monitorv1.Headers{Key: k, Value: v})
	}

	// Convert assertions to separate types
	var statusCodeAssertions []*monitorv1.StatusCodeAssertion
	var bodyAssertions []*monitorv1.BodyAssertion
	var headerAssertions []*monitorv1.HeaderAssertion

	for _, a := range m.Assertions {
		switch a.Kind {
		case config.StatusCode:
			var target int64
			switch v := a.Target.(type) {
			case int:
				target = int64(v)
			case int64:
				target = v
			case float64:
				target = int64(v)
			}
			statusCodeAssertions = append(statusCodeAssertions, &monitorv1.StatusCodeAssertion{
				Target:     target,
				Comparator: configCompareToNumberComparator(a.Compare),
			})
		case config.TextBody:
			target, _ := a.Target.(string)
			bodyAssertions = append(bodyAssertions, &monitorv1.BodyAssertion{
				Target:     target,
				Comparator: configCompareToStringComparator(a.Compare),
			})
		case config.Header:
			target, _ := a.Target.(string)
			headerAssertions = append(headerAssertions, &monitorv1.HeaderAssertion{
				Key:        a.Key,
				Target:     target,
				Comparator: configCompareToStringComparator(a.Compare),
			})
		}
	}

	monitor := &monitorv1.HTTPMonitor{
		Name:                 m.Name,
		Description:          m.Description,
		Url:                  m.Request.URL,
		Method:               stringToHTTPMethod(m.Request.Method),
		Body:                 m.Request.Body,
		Periodicity:          stringToPeriodicity(m.Frequency),
		Active:               m.Active,
		Public:               m.Public,
		Regions:              stringsToRegions(m.Regions),
		Timeout:              m.Timeout,
		Retry:                m.Retry,
		Headers:              headers,
		StatusCodeAssertions: statusCodeAssertions,
		BodyAssertions:       bodyAssertions,
		HeaderAssertions:     headerAssertions,
	}

	if m.DegradedAfter > 0 {
		monitor.DegradedAt = &m.DegradedAfter
	}

	return monitor
}

// configToTCPMonitor converts config.Monitor to SDK TCPMonitor
func configToTCPMonitor(m config.Monitor) *monitorv1.TCPMonitor {
	monitor := &monitorv1.TCPMonitor{
		Name:        m.Name,
		Description: m.Description,
		Uri:         fmt.Sprintf("%s:%d", m.Request.Host, m.Request.Port),
		Periodicity: stringToPeriodicity(m.Frequency),
		Active:      m.Active,
		Public:      m.Public,
		Regions:     stringsToRegions(m.Regions),
		Timeout:     m.Timeout,
		Retry:       m.Retry,
	}

	if m.DegradedAfter > 0 {
		monitor.DegradedAt = &m.DegradedAfter
	}

	return monitor
}

type Monitor struct {
	ID            int         `json:"id"`
	Name          string      `json:"name"`
	URL           string      `json:"url"`
	Periodicity   string      `json:"periodicity"`
	Description   string      `json:"description"`
	Method        string      `json:"method"`
	Regions       []string    `json:"regions"`
	Active        bool        `json:"active"`
	Public        bool        `json:"public"`
	Timeout       int         `json:"timeout"`
	DegradedAfter int         `json:"degraded_after,omitempty"`
	Body          string      `json:"body"`
	Headers       []Header    `json:"headers,omitempty"`
	Assertions    []Assertion `json:"assertions,omitempty"`
	Retry         int         `json:"retry"`
	JobType       string      `json:"jobType"`
}

type Header struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Assertion struct {
	Type    string `json:"type"`
	Compare string `json:"compare"`
	Key     string `json:"key"`
	Target  any    `json:"target"`
}

type Timing struct {
	DnsStart          int64 `json:"dnsStart"`
	DnsDone           int64 `json:"dnsDone"`
	ConnectStart      int64 `json:"connectStart"`
	ConnectDone       int64 `json:"connectDone"`
	TlsHandshakeStart int64 `json:"tlsHandshakeStart"`
	TlsHandshakeDone  int64 `json:"tlsHandshakeDone"`
	FirstByteStart    int64 `json:"firstByteStart"`
	FirstByteDone     int64 `json:"firstByteDone"`
	TransferStart     int64 `json:"transferStart"`
	TransferDone      int64 `json:"transferDone"`
}

type MonitorTriggerResponse struct {
	ResultId int `json:"resultId"`
}

type TriggerResultResponse struct {
	MonitorId  string `json:"monitorId"`
	Url        string `json:"url"`
	Error      bool   `json:"error"`
	Region     string `json:"region"`
	Timestamp  int    `json:"timestamp"`
	Latency    int    `json:"latency"`
	StatusCode int    `json:"statusCode"`
	Timing     Timing `json:"timing"`
}

type RunResult struct {
	JobType   string `json:"jobType"`
	Region    string `json:"region"`
	Message   json.RawMessage
	Timestamp int64 `json:"timestamp"`
	Latency   int64 `json:"latency"`
}

type HTTPRunResult struct {
	Headers map[string]string `json:"headers,omitempty"`
	Body    string            `json:"body,omitempty"`
	Error   string            `json:"error,omitempty"`
	Status  int               `json:"status,omitempty"`
	Timing  Timing            `json:"timing"`
}

type TCPRunResult struct {
	ErrorMessage string `json:"errorMessage"`
	Timing       struct {
		TCPStart int64 `json:"tcpStart"`
		TCPDone  int64 `json:"tcpDone"`
	} `json:"timing"`
}

func MonitorsCmd() *cli.Command {
	monitorsCmd := cli.Command{
		Name:  "monitors",
		Usage: "Manage your monitors",

		Commands: []*cli.Command{
			GetMonitorsApplyCmd(),
			GetMonitorCreateCmd(),
			GetMonitorDeleteCmd(),
			GetMonitorImportCmd(),
			GetMonitorInfoCmd(),
			GetMonitorsListCmd(),
			GetMonitorsTriggerCmd(),
		},
	}
	return &monitorsCmd
}
