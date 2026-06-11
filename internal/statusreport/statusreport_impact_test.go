package statusreport

import (
	"strings"
	"testing"

	status_reportv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_report/v1"
)

func Test_parseImpactsFlag(t *testing.T) {
	t.Parallel()

	t.Run("Empty input returns nil", func(t *testing.T) {
		got, err := parseImpactsFlag(nil)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got != nil {
			t.Errorf("expected nil slice, got %v", got)
		}
	})

	t.Run("Single pair", func(t *testing.T) {
		got, err := parseImpactsFlag([]string{"comp_api=major_outage"})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 entry, got %d", len(got))
		}
		if got[0].GetPageComponentId() != "comp_api" {
			t.Errorf("expected comp_api, got %s", got[0].GetPageComponentId())
		}
		if got[0].GetImpact() != status_reportv1.PageComponentImpact_PAGE_COMPONENT_IMPACT_MAJOR_OUTAGE {
			t.Errorf("expected major_outage, got %v", got[0].GetImpact())
		}
	})

	t.Run("Multiple repeated pairs", func(t *testing.T) {
		got, err := parseImpactsFlag([]string{"comp_api=major_outage", "comp_db=degraded"})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(got) != 2 {
			t.Fatalf("expected 2 entries, got %d", len(got))
		}
	})

	t.Run("Whitespace tolerated around key, =, and value", func(t *testing.T) {
		got, err := parseImpactsFlag([]string{"  comp_api = major_outage  "})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if len(got) != 1 || got[0].GetPageComponentId() != "comp_api" {
			t.Errorf("expected comp_api, got %v", got)
		}
		if got[0].GetImpact() != status_reportv1.PageComponentImpact_PAGE_COMPONENT_IMPACT_MAJOR_OUTAGE {
			t.Errorf("expected major_outage, got %v", got[0].GetImpact())
		}
	})

	t.Run("degraded_performance alias accepted", func(t *testing.T) {
		got, err := parseImpactsFlag([]string{"comp_x=degraded_performance"})
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if got[0].GetImpact() != status_reportv1.PageComponentImpact_PAGE_COMPONENT_IMPACT_DEGRADED_PERFORMANCE {
			t.Errorf("expected degraded_performance enum, got %v", got[0].GetImpact())
		}
	})

	t.Run("All four canonical tokens parse", func(t *testing.T) {
		cases := map[string]status_reportv1.PageComponentImpact{
			"operational":    status_reportv1.PageComponentImpact_PAGE_COMPONENT_IMPACT_OPERATIONAL,
			"degraded":       status_reportv1.PageComponentImpact_PAGE_COMPONENT_IMPACT_DEGRADED_PERFORMANCE,
			"partial_outage": status_reportv1.PageComponentImpact_PAGE_COMPONENT_IMPACT_PARTIAL_OUTAGE,
			"major_outage":   status_reportv1.PageComponentImpact_PAGE_COMPONENT_IMPACT_MAJOR_OUTAGE,
		}
		for token, want := range cases {
			got, err := parseImpactsFlag([]string{"c=" + token})
			if err != nil {
				t.Errorf("%s: unexpected error %v", token, err)
				continue
			}
			if got[0].GetImpact() != want {
				t.Errorf("%s: expected %v, got %v", token, want, got[0].GetImpact())
			}
		}
	})

	t.Run("Missing = errors", func(t *testing.T) {
		_, err := parseImpactsFlag([]string{"comp_api"})
		if err == nil {
			t.Fatal("expected error for missing =")
		}
		if !strings.Contains(err.Error(), "expected <component_id>=<level>") {
			t.Errorf("expected helpful error, got %v", err)
		}
	})

	t.Run("Empty key errors", func(t *testing.T) {
		_, err := parseImpactsFlag([]string{"=major_outage"})
		if err == nil {
			t.Fatal("expected error for empty key")
		}
	})

	t.Run("Empty value errors", func(t *testing.T) {
		_, err := parseImpactsFlag([]string{"comp_api="})
		if err == nil {
			t.Fatal("expected error for empty value")
		}
	})

	t.Run("Empty entry errors", func(t *testing.T) {
		_, err := parseImpactsFlag([]string{"   "})
		if err == nil {
			t.Fatal("expected error for empty entry")
		}
	})

	t.Run("Unknown impact level errors", func(t *testing.T) {
		_, err := parseImpactsFlag([]string{"comp_api=catastrophic"})
		if err == nil {
			t.Fatal("expected error for unknown impact")
		}
		if !strings.Contains(err.Error(), "operational, degraded, partial_outage, major_outage") {
			t.Errorf("expected allowed-values list in error, got %v", err)
		}
	})

	t.Run("Duplicate component_id errors", func(t *testing.T) {
		_, err := parseImpactsFlag([]string{"comp_api=major_outage", "comp_api=degraded"})
		if err == nil {
			t.Fatal("expected error for duplicate component")
		}
		if !strings.Contains(err.Error(), "specified more than once") {
			t.Errorf("expected duplicate-key error, got %v", err)
		}
	})
}

func Test_impactToString_roundtrip(t *testing.T) {
	t.Parallel()

	tokens := []string{"operational", "degraded", "partial_outage", "major_outage"}
	for _, tok := range tokens {
		sdk, err := impactToSDK(tok)
		if err != nil {
			t.Errorf("%s: impactToSDK error %v", tok, err)
			continue
		}
		got := impactToString(sdk)
		if got != tok {
			t.Errorf("round-trip %s -> %v -> %s mismatch", tok, sdk, got)
		}
	}
}

func Test_impactToString_unspecified(t *testing.T) {
	t.Parallel()

	got := impactToString(status_reportv1.PageComponentImpact_PAGE_COMPONENT_IMPACT_UNSPECIFIED)
	if got != "" {
		t.Errorf("UNSPECIFIED should map to empty string, got %q", got)
	}
}

func Test_impactSelectOptions(t *testing.T) {
	t.Parallel()

	opts := impactSelectOptions()
	if len(opts) != 4 {
		t.Errorf("expected 4 options, got %d", len(opts))
	}
}
