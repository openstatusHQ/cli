package api

import (
	"strings"
	"testing"
)

func TestResolveBaseURL(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"unset", "", DefaultBaseURL},
		{"origin", "https://openstatus.example.com", "https://openstatus.example.com"},
		{"trailing slash", "https://openstatus.example.com/", "https://openstatus.example.com"},
		{"node sdk form", "https://openstatus.example.com/rpc", "https://openstatus.example.com"},
		{"node sdk form with slash", "https://openstatus.example.com/rpc/", "https://openstatus.example.com"},
		{"path prefix", "http://localhost:3001/api", "http://localhost:3001/api"},
		{"surrounding whitespace", "  https://openstatus.example.com\n", "https://openstatus.example.com"},
		{"crlf", "https://openstatus.example.com\r\n", "https://openstatus.example.com"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := resolveBaseURL(tt.in)
			if err != nil {
				t.Fatalf("resolveBaseURL(%q) error: %v", tt.in, err)
			}
			if got != tt.want {
				t.Errorf("resolveBaseURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestResolveBaseURL_Invalid(t *testing.T) {
	for _, in := range []string{
		"openstatus.example.com",
		"ftp://openstatus.example.com",
		"https://",
		"://openstatus.example.com",
	} {
		t.Run(in, func(t *testing.T) {
			_, err := resolveBaseURL(in)
			if err == nil {
				t.Fatalf("resolveBaseURL(%q) = nil error, want one", in)
			}
			if !strings.Contains(err.Error(), "OPENSTATUS_API_URL") {
				t.Errorf("error %q does not name OPENSTATUS_API_URL", err)
			}
		})
	}
}

func TestDefaultEndpointsUnchanged(t *testing.T) {
	if APIBaseURL != "https://api.openstatus.dev/v1" {
		t.Errorf("APIBaseURL = %q", APIBaseURL)
	}
	if ConnectBaseURL != "https://api.openstatus.dev/rpc" {
		t.Errorf("ConnectBaseURL = %q", ConnectBaseURL)
	}
}

// The variable is read when LoadBaseURL runs, not at package init, so a value
// from the .env file (loaded in main) is honoured.
func TestLoadBaseURL_ReadsEnvAtCallTime(t *testing.T) {
	t.Cleanup(func() {
		BaseURL, APIBaseURL, ConnectBaseURL = DefaultBaseURL, DefaultBaseURL+"/v1", DefaultBaseURL+"/rpc"
	})
	t.Setenv("OPENSTATUS_API_URL", "https://openstatus.example.com")

	if err := LoadBaseURL(); err != nil {
		t.Fatalf("LoadBaseURL: %v", err)
	}
	if BaseURL != "https://openstatus.example.com" ||
		APIBaseURL != "https://openstatus.example.com/v1" ||
		ConnectBaseURL != "https://openstatus.example.com/rpc" {
		t.Errorf("got %q, %q, %q", BaseURL, APIBaseURL, ConnectBaseURL)
	}
}

func TestLoadBaseURL_InvalidKeepsDefaults(t *testing.T) {
	t.Setenv("OPENSTATUS_API_URL", "openstatus.example.com")

	if err := LoadBaseURL(); err == nil {
		t.Fatal("LoadBaseURL = nil error, want one")
	}
	if BaseURL != DefaultBaseURL {
		t.Errorf("BaseURL = %q, want default", BaseURL)
	}
}
