package check

import (
	"bytes"
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/fatih/color"
)

func TestMain(m *testing.M) {
	color.NoColor = true
	os.Exit(m.Run())
}

func TestDisplayName(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"fra", "Frankfurt (Fly)"},
		{"koyeb_par", "Paris (Koyeb)"},
		{"railway_us-west2", "California (Railway)"},
		{"unknown_region_xyz", "unknown_region_xyz"},
	}
	for _, c := range cases {
		if got := DisplayName(c.in); got != c.want {
			t.Errorf("DisplayName(%q) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestDisplayName_HappyFixtureCoverage(t *testing.T) {
	t.Parallel()
	codes := []string{
		"ams", "arn", "bom", "cdg", "dfw", "ewr", "fra", "gru", "iad", "jnb",
		"lax", "lhr", "nrt", "ord", "sjc", "sin", "syd", "yyz",
		"koyeb_fra", "koyeb_par", "koyeb_sfo", "koyeb_sin", "koyeb_tyo", "koyeb_was",
		"railway_us-west2", "railway_us-east4-eqdc4a", "railway_europe-west4-drams3a", "railway_asia-southeast1-eqsg3a",
	}
	for _, c := range codes {
		if DisplayName(c) == c {
			t.Errorf("region %q has no display name", c)
		}
	}
}

func TestRenderer_HumanDefault(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	r := NewRenderer(&buf, false)
	r.Row(RegionResult{Region: "fra", State: "success", Status: 200, Latency: 34, Timing: &Timing{DNS: 14, Connection: 2, TLS: 9, TTFB: 9, Transfer: 1}})
	r.Row(RegionResult{Region: "iad", State: "success", Status: 200, Latency: 67, Timing: &Timing{DNS: 53, Connection: 2, TLS: 5, TTFB: 7, Transfer: 0}})
	r.Footer("https://example.com", []RegionResult{
		{Region: "fra", State: "success", Latency: 34},
		{Region: "iad", State: "success", Latency: 67},
	}, "abc123")

	out := buf.String()
	for _, want := range []string{"Region", "Latency", "Status", "State", "Frankfurt (Fly)", "Ashburn (Fly)", "34ms", "67ms", "Fastest:", "Slowest:", "Mean:", "Success: 2/2", "View:", "abc123"} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n--- output ---\n%s", want, out)
		}
	}
	if strings.Contains(out, "DNS") {
		t.Errorf("default output should not include timing columns; got %q", out)
	}
}

func TestRenderer_TimingMode(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	r := NewRenderer(&buf, true)
	r.Row(RegionResult{Region: "fra", State: "success", Status: 200, Latency: 34, Timing: &Timing{DNS: 14, Connection: 2, TLS: 9, TTFB: 9, Transfer: 1}})
	out := buf.String()
	for _, want := range []string{"DNS", "Conn", "TLS", "TTFB", "Transfer", "Frankfurt (Fly)", "14", "9"} {
		if !strings.Contains(out, want) {
			t.Errorf("timing output missing %q\n--- output ---\n%s", want, out)
		}
	}
}

func TestRenderer_FailureRowShowsMessage(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	r := NewRenderer(&buf, false)
	r.Row(RegionResult{Region: "lhr", State: "error", Message: "url not reachable"})
	out := buf.String()
	if !strings.Contains(out, "London (Fly)") {
		t.Errorf("missing region name; got %q", out)
	}
	if !strings.Contains(out, "url not r") {
		t.Errorf("missing message (possibly truncated); got %q", out)
	}
	if !strings.Contains(out, "—") {
		t.Errorf("missing dash for unavailable latency/status; got %q", out)
	}
}

func TestComputeSummary_AllSuccess(t *testing.T) {
	t.Parallel()
	results := []RegionResult{
		{Region: "fra", State: "success", Latency: 30},
		{Region: "iad", State: "success", Latency: 90},
		{Region: "sin", State: "success", Latency: 60},
	}
	s := computeSummary(results)
	if s.TotalRegions != 3 || s.Successes != 3 {
		t.Errorf("counts = %d/%d, want 3/3", s.Successes, s.TotalRegions)
	}
	if s.SuccessRate != 1.0 {
		t.Errorf("success rate = %f, want 1.0", s.SuccessRate)
	}
	if s.MeanLatency != 60 {
		t.Errorf("mean = %d, want 60", s.MeanLatency)
	}
	if s.Fastest == nil || s.Fastest.Region != "fra" || s.Fastest.Latency != 30 {
		t.Errorf("fastest = %+v, want fra/30", s.Fastest)
	}
	if s.Slowest == nil || s.Slowest.Region != "iad" || s.Slowest.Latency != 90 {
		t.Errorf("slowest = %+v, want iad/90", s.Slowest)
	}
}

func TestComputeSummary_MixedFailures(t *testing.T) {
	t.Parallel()
	results := []RegionResult{
		{Region: "fra", State: "success", Latency: 30},
		{Region: "iad", State: "error", Message: "url not reachable"},
		{Region: "sin", State: "success", Latency: 60},
	}
	s := computeSummary(results)
	if s.Successes != 2 || s.TotalRegions != 3 {
		t.Errorf("counts = %d/%d, want 2/3", s.Successes, s.TotalRegions)
	}
	if s.SuccessRate < 0.66 || s.SuccessRate > 0.67 {
		t.Errorf("success rate = %f, want ~0.667", s.SuccessRate)
	}
	if s.MeanLatency != 45 {
		t.Errorf("mean = %d, want 45 (only count latencies > 0)", s.MeanLatency)
	}
}

func TestComputeSummary_Empty(t *testing.T) {
	t.Parallel()
	s := computeSummary(nil)
	if s.TotalRegions != 0 || s.Successes != 0 || s.MeanLatency != 0 || s.Fastest != nil || s.Slowest != nil {
		t.Errorf("non-zero summary for empty results: %+v", s)
	}
}

func TestBuildJSONOutput_Shape(t *testing.T) {
	t.Parallel()
	results := []RegionResult{
		{Region: "fra", State: "success", Status: 200, Latency: 34, Timestamp: 1, Timing: &Timing{DNS: 14, Connection: 2, TLS: 9, TTFB: 9, Transfer: 1}},
	}
	out := buildJSONOutput("https://example.com", "abc123", results)
	if out.URL != "https://example.com" {
		t.Errorf("url = %q", out.URL)
	}
	if out.CheckID != "abc123" {
		t.Errorf("check_id = %q", out.CheckID)
	}
	if out.ShareURL != "https://www.openstatus.dev/play/checker/abc123" {
		t.Errorf("share_url = %q", out.ShareURL)
	}
	if len(out.Results) != 1 || out.Results[0].Region != "fra" {
		t.Errorf("results unexpected: %+v", out.Results)
	}

	raw, err := json.Marshal(out)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	for _, want := range []string{`"timing"`, `"dns":14`, `"summary"`, `"mean_latency":34`, `"success_rate":1`, `"share_url":"https://www.openstatus.dev/play/checker/abc123"`} {
		if !bytes.Contains(raw, []byte(want)) {
			t.Errorf("json missing %q\n--- json ---\n%s", want, raw)
		}
	}
}

func TestShareURL(t *testing.T) {
	t.Parallel()
	if got := shareURL("abc"); got != "https://www.openstatus.dev/play/checker/abc" {
		t.Errorf("shareURL = %q", got)
	}
	if got := shareURL(""); got != "" {
		t.Errorf("shareURL empty = %q, want \"\"", got)
	}
}

func TestTruncate(t *testing.T) {
	t.Parallel()
	if got := truncate("short", 10); got != "short" {
		t.Errorf("truncate short = %q", got)
	}
	if got := truncate("verylongstring", 5); got != "very…" {
		t.Errorf("truncate long = %q, want \"very…\"", got)
	}
	if got := truncate("ab", 1); got != "a" {
		t.Errorf("truncate small n = %q", got)
	}
}
