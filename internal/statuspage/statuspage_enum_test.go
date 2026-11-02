package statuspage_test

import (
	"testing"

	status_pagev1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/status_page/v1"

	"github.com/openstatusHQ/cli/internal/statuspage"
)

func Test_EnumMappings(t *testing.T) {
	t.Parallel()

	if got := statuspage.AccessTypeToString(status_pagev1.PageAccessType_PAGE_ACCESS_TYPE_IP_RESTRICTED); got != "ip-restricted" {
		t.Errorf("Expected 'ip-restricted', got %q", got)
	}

	locales := map[status_pagev1.Locale]string{
		status_pagev1.Locale_LOCALE_TR: "tr",
		status_pagev1.Locale_LOCALE_HI: "hi",
		status_pagev1.Locale_LOCALE_KO: "ko",
		status_pagev1.Locale_LOCALE_JA: "ja",
	}
	for l, want := range locales {
		if got := statuspage.LocaleToString(l); got != want {
			t.Errorf("LocaleToString(%v) = %q, want %q", l, got, want)
		}
	}
}
