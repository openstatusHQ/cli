package monitors

import (
	"encoding/json"

	"github.com/urfave/cli/v3"
)

type Monitor struct {
	ID            int               `json:"id"`
	Name          string            `json:"name"`
	URL           string            `json:"url"`
	Periodicity   string            `json:"periodicity"`
	Description   string            `json:"description"`
	Method        string            `json:"method"`
	Regions       []string          `json:"regions"`
	Active        bool              `json:"active"`
	Public        bool              `json:"public"`
	Timeout       int               `json:"timeout"`
	DegradedAfter int               `json:"degraded_after,omitempty"`
	Body          string            `json:"body"`
	Headers       []Header          `json:"headers,omitempty"`
	Assertions    []Assertion       `json:"assertions,omitempty"`
}

type Header struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}

type Assertion struct {
	Type    string `json:"type"`
	Compare string `json:"compare"`
	Value   string `json:"value"`
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
			GetMonitorCreateCmd(),
			GetMonitorDeleteCmd(),
			GetMonitorInfoCmd(),
			GetMonitorsListCmd(),
			GetMonitorsTriggerCmd(),
		},
	}
	return &monitorsCmd
}
