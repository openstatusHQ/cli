package statusreport

import (
	"testing"
)

func Test_statusSelectOptions(t *testing.T) {
	t.Parallel()

	opts := statusSelectOptions()
	if len(opts) != 4 {
		t.Errorf("expected 4 options, got %d", len(opts))
	}
}
