package privatelocation_test

import (
	"io"
	"net/http"
	"strings"
	"testing"

	private_locationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/private_location/v1"

	"github.com/openstatusHQ/cli/internal/privatelocation"
)

type interceptorHTTPClient struct {
	f func(req *http.Request) (*http.Response, error)
}

func (i *interceptorHTTPClient) RoundTrip(req *http.Request) (*http.Response, error) {
	return i.f(req)
}

func (i *interceptorHTTPClient) GetHTTPClient() *http.Client {
	return &http.Client{
		Transport: i,
	}
}

// jsonResponder replies to every request with the given body, recording the procedures called.
func jsonResponder(bodies map[string]string, calls *[]string) *interceptorHTTPClient {
	return &interceptorHTTPClient{
		f: func(req *http.Request) (*http.Response, error) {
			if calls != nil {
				*calls = append(*calls, req.URL.Path)
			}
			body := "{}"
			for suffix, b := range bodies {
				if strings.HasSuffix(req.URL.Path, suffix) {
					body = b
					break
				}
			}
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		},
	}
}

// errorResponder replies with a Connect error envelope.
func errorResponder(status int, body string) *interceptorHTTPClient {
	return &interceptorHTTPClient{
		f: func(req *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: status,
				Body:       io.NopCloser(strings.NewReader(body)),
				Header:     http.Header{"Content-Type": []string{"application/json"}},
			}, nil
		},
	}
}

func Test_PrivateLocationsCmd(t *testing.T) {
	t.Parallel()

	t.Run("Returns valid command", func(t *testing.T) {
		cmd := privatelocation.PrivateLocationsCmd()

		if cmd == nil {
			t.Fatal("Expected non-nil command")
		}
		if cmd.Name != "private-locations" {
			t.Errorf("Expected command name 'private-locations', got %s", cmd.Name)
		}
	})

	t.Run("Has pl alias and no others", func(t *testing.T) {
		cmd := privatelocation.PrivateLocationsCmd()

		if len(cmd.Aliases) != 1 || cmd.Aliases[0] != "pl" {
			t.Errorf("Expected exactly the 'pl' alias, got %v", cmd.Aliases)
		}
	})

	t.Run("Has expected subcommands", func(t *testing.T) {
		cmd := privatelocation.PrivateLocationsCmd()

		if len(cmd.Commands) != 3 {
			t.Errorf("Expected 3 subcommands, got %d", len(cmd.Commands))
		}

		expected := map[string]bool{"list": false, "info": false, "create": false}
		for _, sub := range cmd.Commands {
			if _, ok := expected[sub.Name]; ok {
				expected[sub.Name] = true
			}
		}
		for name, found := range expected {
			if !found {
				t.Errorf("Expected subcommand '%s' not found", name)
			}
		}
	})

	t.Run("info has no view alias", func(t *testing.T) {
		cmd := privatelocation.PrivateLocationsCmd()

		for _, sub := range cmd.Commands {
			if sub.Name != "info" {
				continue
			}
			for _, alias := range sub.Aliases {
				if alias == "view" {
					t.Error("Expected no 'view' alias on info")
				}
			}
		}
	})
}

func Test_MaskToken(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		token string
		want  string
	}{
		{"empty stays empty", "", ""},
		{"short token fully masked", "abc", "•••"},
		{"exactly four fully masked", "abcd", "••••"},
		{"long token keeps last four", "os_pl_live_9f3a2b1c8d4e6f", "••••••••••••4e6f"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := privatelocation.MaskToken(tt.token); got != tt.want {
				t.Errorf("MaskToken(%q) = %q, want %q", tt.token, got, tt.want)
			}
		})
	}
}

func Test_statusToString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status private_locationv1.PrivateLocationStatus
		want   string
	}{
		{private_locationv1.PrivateLocationStatus_PRIVATE_LOCATION_STATUS_ACTIVE, "active"},
		{private_locationv1.PrivateLocationStatus_PRIVATE_LOCATION_STATUS_ERROR, "error"},
		{private_locationv1.PrivateLocationStatus_PRIVATE_LOCATION_STATUS_UNSPECIFIED, "unknown"},
	}

	for _, tt := range tests {
		if got := privatelocation.StatusToString(tt.status); got != tt.want {
			t.Errorf("StatusToString(%v) = %q, want %q", tt.status, got, tt.want)
		}
	}
}

func Test_formatLastSeen(t *testing.T) {
	t.Parallel()

	if got := privatelocation.FormatLastSeen(""); got != "never" {
		t.Errorf("Expected 'never' for empty timestamp, got %q", got)
	}

	if got := privatelocation.FormatLastSeen("2026-07-24T14:02:11Z"); got == "never" || got == "" {
		t.Errorf("Expected a formatted timestamp, got %q", got)
	}

	if got := privatelocation.FormatLastSeen("not-a-date"); got != "not-a-date" {
		t.Errorf("Expected unparseable input to pass through, got %q", got)
	}
}

func Test_formatMetadata(t *testing.T) {
	t.Parallel()

	if got := privatelocation.FormatMetadata(nil); got != "none" {
		t.Errorf("Expected 'none' for empty metadata, got %q", got)
	}

	// Keys are sorted so the output is stable across runs.
	got := privatelocation.FormatMetadata(map[string]string{"team": "infra", "env": "prod"})
	if got != "env=prod, team=infra" {
		t.Errorf("Expected sorted metadata, got %q", got)
	}
}

func Test_formatMonitors(t *testing.T) {
	t.Parallel()

	if got := privatelocation.FormatMonitors(nil); got != "none" {
		t.Errorf("Expected 'none' for no monitors, got %q", got)
	}

	got := privatelocation.FormatMonitors([]privatelocation.MonitorRef{
		{ID: "123", Name: "api-prod"},
		{ID: "456"},
	})
	if got != "2: api-prod (123), 456" {
		t.Errorf("Expected resolved name then bare ID fallback, got %q", got)
	}
}

func Test_parseMetadataFlag(t *testing.T) {
	t.Parallel()

	t.Run("nil for no values", func(t *testing.T) {
		got, err := privatelocation.ParseMetadataFlag(nil)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if got != nil {
			t.Errorf("Expected nil map, got %v", got)
		}
	})

	t.Run("parses key=value pairs", func(t *testing.T) {
		got, err := privatelocation.ParseMetadataFlag([]string{"env=prod", "team=infra"})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if got["env"] != "prod" || got["team"] != "infra" {
			t.Errorf("Unexpected metadata: %v", got)
		}
	})

	t.Run("keeps = inside the value", func(t *testing.T) {
		got, err := privatelocation.ParseMetadataFlag([]string{"query=a=b"})
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if got["query"] != "a=b" {
			t.Errorf("Expected value 'a=b', got %q", got["query"])
		}
	})

	t.Run("rejects malformed input", func(t *testing.T) {
		for _, bad := range []string{"noequals", "=novalue"} {
			if _, err := privatelocation.ParseMetadataFlag([]string{bad}); err == nil {
				t.Errorf("Expected an error for %q", bad)
			}
		}
	})
}

func Test_Refs(t *testing.T) {
	t.Parallel()

	refs := map[string]privatelocation.Ref{
		"pl_1": {ID: "pl_1", Name: "office-paris", Status: "active"},
	}

	got := privatelocation.Refs([]string{"pl_1", "pl_missing"}, refs)
	if len(got) != 2 {
		t.Fatalf("Expected 2 refs, got %d", len(got))
	}
	if got[0].Name != "office-paris" {
		t.Errorf("Expected resolved name, got %q", got[0].Name)
	}
	// Unresolved IDs still produce an entry so callers can fall back to the raw ID.
	if got[1].ID != "pl_missing" || got[1].Name != "" {
		t.Errorf("Expected bare ID for unresolved entry, got %+v", got[1])
	}
}

func Test_Labels(t *testing.T) {
	t.Parallel()

	refs := map[string]privatelocation.Ref{
		"pl_1": {ID: "pl_1", Name: "office-paris", Status: "active"},
	}

	labels := privatelocation.Labels([]string{"pl_1", "pl_missing"}, refs, false)
	if labels[0] != "office-paris" {
		t.Errorf("Expected name, got %q", labels[0])
	}
	if labels[1] != "pl_missing" {
		t.Errorf("Expected raw ID fallback, got %q", labels[1])
	}

	withStatus := privatelocation.Labels([]string{"pl_1"}, refs, true)
	if !strings.Contains(withStatus[0], "office-paris") || !strings.Contains(withStatus[0], "active") {
		t.Errorf("Expected name and status, got %q", withStatus[0])
	}
}
