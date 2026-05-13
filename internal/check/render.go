package check

import (
	"fmt"
	"io"

	"github.com/fatih/color"
)

const (
	colRegion    = 28
	colLatency   = 10
	colStatus    = 7
	colState     = 20
	colTimingNum = 8
)

type Renderer struct {
	Out         io.Writer
	Timing      bool
	headerShown bool
}

func NewRenderer(out io.Writer, timing bool) *Renderer {
	return &Renderer{Out: out, Timing: timing}
}

func (r *Renderer) Row(row RegionResult) {
	if !r.headerShown {
		r.printHeader()
		r.headerShown = true
	}
	r.printRow(row)
}

func (r *Renderer) printHeader() {
	green := color.New(color.FgGreen, color.Underline).SprintfFunc()
	if r.Timing {
		fmt.Fprintln(r.Out, green(
			"%-*s  %*s  %*s  %-*s  %*s  %*s  %*s  %*s  %*s",
			colRegion, "Region",
			colLatency, "Latency",
			colStatus, "Status",
			colState, "State",
			colTimingNum, "DNS",
			colTimingNum, "Conn",
			colTimingNum, "TLS",
			colTimingNum, "TTFB",
			colTimingNum, "Transfer",
		))
		return
	}
	fmt.Fprintln(r.Out, green(
		"%-*s  %*s  %*s  %-*s",
		colRegion, "Region",
		colLatency, "Latency",
		colStatus, "Status",
		colState, "State",
	))
}

func (r *Renderer) printRow(row RegionResult) {
	region := truncate(DisplayName(row.Region), colRegion)
	latency := formatLatency(row.Latency)
	if !row.Succeeded() {
		latency = color.RedString("%*s", colLatency, dashIfZero(row.Latency, latency))
	} else {
		latency = fmt.Sprintf("%*s", colLatency, latency)
	}
	status := dashOrInt(row.Status)
	state := stateLabel(row)

	if r.Timing {
		t := row.Timing
		dns := timingCell(t, func(tt *Timing) int64 { return tt.DNS })
		conn := timingCell(t, func(tt *Timing) int64 { return tt.Connection })
		tls := timingCell(t, func(tt *Timing) int64 { return tt.TLS })
		ttfb := timingCell(t, func(tt *Timing) int64 { return tt.TTFB })
		xfer := timingCell(t, func(tt *Timing) int64 { return tt.Transfer })
		fmt.Fprintf(r.Out, "%-*s  %s  %*s  %-*s  %*s  %*s  %*s  %*s  %*s\n",
			colRegion, region,
			latency,
			colStatus, status,
			colState, truncate(state, colState),
			colTimingNum, dns,
			colTimingNum, conn,
			colTimingNum, tls,
			colTimingNum, ttfb,
			colTimingNum, xfer,
		)
		return
	}
	fmt.Fprintf(r.Out, "%-*s  %s  %*s  %-*s\n",
		colRegion, region,
		latency,
		colStatus, status,
		colState, truncate(state, colState),
	)
}

func (r *Renderer) Footer(checkedURL string, results []RegionResult, checkID string) {
	if len(results) == 0 {
		return
	}
	summary := computeSummary(results)
	fmt.Fprintln(r.Out)
	bold := color.New(color.Bold).SprintfFunc()
	if summary.Fastest != nil {
		fmt.Fprintf(r.Out, "%s %s %dms\n", bold("Fastest:"), DisplayName(summary.Fastest.Region), summary.Fastest.Latency)
	}
	if summary.Slowest != nil {
		fmt.Fprintf(r.Out, "%s %s %dms\n", bold("Slowest:"), DisplayName(summary.Slowest.Region), summary.Slowest.Latency)
	}
	fmt.Fprintf(r.Out, "%s    %dms\n", bold("Mean:"), summary.MeanLatency)
	fmt.Fprintf(r.Out, "%s %d/%d (%.0f%%)\n", bold("Success:"), summary.Successes, summary.TotalRegions, summary.SuccessRate*100)
	if checkID != "" {
		fmt.Fprintf(r.Out, "%s    %s\n", bold("View:"), shareURL(checkID))
	}
	_ = checkedURL
}

func stateLabel(row RegionResult) string {
	if row.Succeeded() {
		return "success"
	}
	if row.Message != "" {
		return row.Message
	}
	if row.State != "" {
		return row.State
	}
	return "error"
}

func formatLatency(ms int64) string {
	if ms <= 0 {
		return "—"
	}
	return fmt.Sprintf("%dms", ms)
}

func timingCell(t *Timing, get func(*Timing) int64) string {
	if t == nil {
		return "—"
	}
	v := get(t)
	if v <= 0 {
		return "0"
	}
	return fmt.Sprintf("%d", v)
}

func dashIfZero(v int64, fallback string) string {
	if v <= 0 {
		return "—"
	}
	return fallback
}

func dashOrInt(v int) string {
	if v == 0 {
		return "—"
	}
	return fmt.Sprintf("%d", v)
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	if n <= 1 {
		return s[:n]
	}
	return s[:n-1] + "…"
}

type JSONOutput struct {
	URL      string         `json:"url"`
	CheckID  string         `json:"check_id"`
	ShareURL string         `json:"share_url"`
	Results  []RegionResult `json:"results"`
	Summary  Summary        `json:"summary"`
}

type Summary struct {
	Fastest      *Endpoint `json:"fastest,omitempty"`
	Slowest      *Endpoint `json:"slowest,omitempty"`
	MeanLatency  int64     `json:"mean_latency"`
	SuccessRate  float64   `json:"success_rate"`
	TotalRegions int       `json:"total_regions"`
	Successes    int       `json:"successes"`
}

type Endpoint struct {
	Region  string `json:"region"`
	Latency int64  `json:"latency"`
}

func buildJSONOutput(checkedURL, checkID string, results []RegionResult) JSONOutput {
	return JSONOutput{
		URL:      checkedURL,
		CheckID:  checkID,
		ShareURL: shareURL(checkID),
		Results:  results,
		Summary:  computeSummary(results),
	}
}

func computeSummary(results []RegionResult) Summary {
	s := Summary{TotalRegions: len(results)}
	if len(results) == 0 {
		return s
	}
	var sumLat int64
	var latencyCount int64
	var fastestLat, slowestLat int64 = -1, -1
	var fastestRegion, slowestRegion string

	for _, r := range results {
		if r.Succeeded() {
			s.Successes++
		}
		if r.Latency > 0 {
			sumLat += r.Latency
			latencyCount++
			if fastestLat < 0 || r.Latency < fastestLat {
				fastestLat = r.Latency
				fastestRegion = r.Region
			}
			if slowestLat < 0 || r.Latency > slowestLat {
				slowestLat = r.Latency
				slowestRegion = r.Region
			}
		}
	}

	if latencyCount > 0 {
		s.MeanLatency = sumLat / latencyCount
		s.Fastest = &Endpoint{Region: fastestRegion, Latency: fastestLat}
		s.Slowest = &Endpoint{Region: slowestRegion, Latency: slowestLat}
	}
	s.SuccessRate = float64(s.Successes) / float64(len(results))
	return s
}
