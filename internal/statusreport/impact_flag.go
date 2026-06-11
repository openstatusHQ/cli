package statusreport

import (
	"fmt"
	"strings"

	status_reportv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_report/v1"
)

func parseImpactsFlag(raw []string) ([]*status_reportv1.ComponentImpact, error) {
	if len(raw) == 0 {
		return nil, nil
	}

	var out []*status_reportv1.ComponentImpact
	seen := make(map[string]struct{})

	for _, entry := range raw {
		entry = strings.TrimSpace(entry)
		if entry == "" {
			return nil, fmt.Errorf("invalid --impact value: empty entry")
		}

		key, value, ok := strings.Cut(entry, "=")
		if !ok {
			return nil, fmt.Errorf("invalid --impact value %q: expected <component_id>=<level>", entry)
		}
		key = strings.TrimSpace(key)
		value = strings.TrimSpace(value)
		if key == "" {
			return nil, fmt.Errorf("invalid --impact value %q: empty component id", entry)
		}
		if value == "" {
			return nil, fmt.Errorf("invalid --impact value %q: empty impact level", entry)
		}

		level, err := impactToSDK(value)
		if err != nil {
			return nil, fmt.Errorf("invalid --impact value %q: %w", entry, err)
		}

		if _, dup := seen[key]; dup {
			return nil, fmt.Errorf("invalid --impact: component %q specified more than once", key)
		}
		seen[key] = struct{}{}

		ci := &status_reportv1.ComponentImpact{}
		ci.SetPageComponentId(key)
		ci.SetImpact(level)
		out = append(out, ci)
	}

	return out, nil
}
