package monitors_test

import (
	"testing"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	"github.com/openstatusHQ/cli/internal/config"
	"github.com/openstatusHQ/cli/internal/monitors"
)

func Test_RegionRoundTrip(t *testing.T) {
	for i := 1; i <= 28; i++ {
		r := monitorv1.Region(i)
		t.Run(r.String(), func(t *testing.T) {
			code := monitors.RegionToString(r)
			back := monitors.StringToRegion(config.Region(code))
			if back != r {
				t.Errorf("round-trip failed: %v -> %q -> %v", r, code, back)
			}
		})
	}
}

func Test_RegionToString_NoCollisions(t *testing.T) {
	seen := make(map[string]monitorv1.Region)
	for i := 1; i <= 28; i++ {
		r := monitorv1.Region(i)
		code := monitors.RegionToString(r)
		if prev, exists := seen[code]; exists {
			t.Errorf("collision: %v and %v both map to %q", prev, r, code)
		}
		seen[code] = r
	}
}

func Test_RegionToString_Koyeb(t *testing.T) {
	tests := []struct {
		region   monitorv1.Region
		expected string
	}{
		{monitorv1.Region_REGION_KOYEB_FRA, "koyeb_fra"},
		{monitorv1.Region_REGION_KOYEB_PAR, "koyeb_par"},
		{monitorv1.Region_REGION_KOYEB_SFO, "koyeb_sfo"},
		{monitorv1.Region_REGION_KOYEB_SIN, "koyeb_sin"},
		{monitorv1.Region_REGION_KOYEB_TYO, "koyeb_tyo"},
		{monitorv1.Region_REGION_KOYEB_WAS, "koyeb_was"},
	}
	for _, tt := range tests {
		t.Run(tt.region.String(), func(t *testing.T) {
			got := monitors.RegionToString(tt.region)
			if got != tt.expected {
				t.Errorf("RegionToString(%v) = %q, want %q", tt.region, got, tt.expected)
			}
		})
	}
}

func Test_RegionToString_Railway(t *testing.T) {
	tests := []struct {
		region   monitorv1.Region
		expected string
	}{
		{monitorv1.Region_REGION_RAILWAY_US_WEST2, "railway_us-west2"},
		{monitorv1.Region_REGION_RAILWAY_US_EAST4, "railway_us-east4-eqdc4a"},
		{monitorv1.Region_REGION_RAILWAY_EUROPE_WEST4, "railway_europe-west4-drams3a"},
		{monitorv1.Region_REGION_RAILWAY_ASIA_SOUTHEAST1, "railway_asia-southeast1-eqsg3a"},
	}
	for _, tt := range tests {
		t.Run(tt.region.String(), func(t *testing.T) {
			got := monitors.RegionToString(tt.region)
			if got != tt.expected {
				t.Errorf("RegionToString(%v) = %q, want %q", tt.region, got, tt.expected)
			}
		})
	}
}
