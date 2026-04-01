package terraform

import "testing"

func TestSanitizeName(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"API Health Check", "api_health_check"},
		{"3xx Error Monitor", "resource_3xx_error_monitor"},
		{"hello---world", "hello_world"},
		{"", "unnamed"},
		{"   ", "unnamed"},
		{"café / über", "cafe_uber"},
		{"Simple", "simple"},
		{"already_valid", "already_valid"},
		{"___leading_trailing___", "leading_trailing"},
		{"UPPER CASE", "upper_case"},
		{"special!@#$%chars", "special_chars"},
		{"123", "resource_123"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := sanitizeName(tt.input)
			if got != tt.want {
				t.Errorf("sanitizeName(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNameRegistryDeduplication(t *testing.T) {
	reg := NewNameRegistry()

	got1 := reg.Name("openstatus_http_monitor", "API")
	got2 := reg.Name("openstatus_http_monitor", "API")
	got3 := reg.Name("openstatus_http_monitor", "API")

	if got1 != "api" {
		t.Errorf("first = %q, want %q", got1, "api")
	}
	if got2 != "api_2" {
		t.Errorf("second = %q, want %q", got2, "api_2")
	}
	if got3 != "api_3" {
		t.Errorf("third = %q, want %q", got3, "api_3")
	}
}

func TestNameRegistryPerTypeScopee(t *testing.T) {
	reg := NewNameRegistry()

	httpName := reg.Name("openstatus_http_monitor", "API")
	tcpName := reg.Name("openstatus_tcp_monitor", "API")

	if httpName != "api" {
		t.Errorf("http = %q, want %q", httpName, "api")
	}
	if tcpName != "api" {
		t.Errorf("tcp = %q, want %q", tcpName, "api")
	}
}
