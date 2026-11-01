package run

import "testing"

func TestRunMonitorResultFailed(t *testing.T) {
	t.Parallel()
	for name, tc := range map[string]struct {
		res  runMonitorResult
		want bool
	}{
		"all pass":      {runMonitorResult{Results: []runRegionResult{{Status: "pass"}, {Status: "pass"}}}, false},
		"one fail":      {runMonitorResult{Results: []runRegionResult{{Status: "pass"}, {Status: "fail"}}}, true},
		"unknown":       {runMonitorResult{Results: []runRegionResult{{Status: "unknown"}}}, true},
		"trigger error": {runMonitorResult{Error: "boom"}, true},
	} {
		if got := tc.res.failed(); got != tc.want {
			t.Errorf("%s: failed() = %v, want %v", name, got, tc.want)
		}
	}
}
