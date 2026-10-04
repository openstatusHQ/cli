package api

import "testing"

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
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := resolveBaseURL(tt.in); got != tt.want {
				t.Errorf("resolveBaseURL(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestDefaultEndpointsUnchanged(t *testing.T) {
	base := resolveBaseURL("")
	if got := base + "/v1"; got != "https://api.openstatus.dev/v1" {
		t.Errorf("REST endpoint = %q", got)
	}
	if got := base + "/rpc"; got != "https://api.openstatus.dev/rpc" {
		t.Errorf("ConnectRPC endpoint = %q", got)
	}
}
