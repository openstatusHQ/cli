package config_test

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openstatusHQ/cli/internal/config"
)

func Test_ConvertAssertionTargets(t *testing.T) {
	t.Run("Convert StatusCode with int target", func(t *testing.T) {
		assertions := []config.Assertion{
			{
				Kind:    config.StatusCode,
				Compare: config.Eq,
				Target:  200,
			},
		}

		config.ConvertAssertionTargets(assertions)

		if assertions[0].Target != 200 {
			t.Errorf("Expected target to be 200, got %v", assertions[0].Target)
		}
	})

	t.Run("Convert StatusCode with int64 target", func(t *testing.T) {
		assertions := []config.Assertion{
			{
				Kind:    config.StatusCode,
				Compare: config.Eq,
				Target:  int64(200),
			},
		}

		config.ConvertAssertionTargets(assertions)

		if assertions[0].Target != 200 {
			t.Errorf("Expected target to be 200, got %v", assertions[0].Target)
		}
	})

	t.Run("Convert StatusCode with float64 target", func(t *testing.T) {
		assertions := []config.Assertion{
			{
				Kind:    config.StatusCode,
				Compare: config.Eq,
				Target:  float64(200),
			},
		}

		config.ConvertAssertionTargets(assertions)

		if assertions[0].Target != 200 {
			t.Errorf("Expected target to be 200, got %v", assertions[0].Target)
		}
	})

	t.Run("Convert Header with string target", func(t *testing.T) {
		assertions := []config.Assertion{
			{
				Kind:    config.Header,
				Compare: config.Eq,
				Target:  "application/json",
				Key:     "Content-Type",
			},
		}

		config.ConvertAssertionTargets(assertions)

		if assertions[0].Target != "application/json" {
			t.Errorf("Expected target to be 'application/json', got %v", assertions[0].Target)
		}
	})

	t.Run("Convert TextBody with string target", func(t *testing.T) {
		assertions := []config.Assertion{
			{
				Kind:    config.TextBody,
				Compare: config.Contains,
				Target:  "success",
			},
		}

		config.ConvertAssertionTargets(assertions)

		if assertions[0].Target != "success" {
			t.Errorf("Expected target to be 'success', got %v", assertions[0].Target)
		}
	})

	t.Run("Convert multiple assertions", func(t *testing.T) {
		assertions := []config.Assertion{
			{
				Kind:    config.StatusCode,
				Compare: config.Eq,
				Target:  float64(200),
			},
			{
				Kind:    config.Header,
				Compare: config.Eq,
				Target:  "application/json",
				Key:     "Content-Type",
			},
			{
				Kind:    config.TextBody,
				Compare: config.Contains,
				Target:  "OK",
			},
		}

		config.ConvertAssertionTargets(assertions)

		expected := []config.Assertion{
			{
				Kind:    config.StatusCode,
				Compare: config.Eq,
				Target:  200,
			},
			{
				Kind:    config.Header,
				Compare: config.Eq,
				Target:  "application/json",
				Key:     "Content-Type",
			},
			{
				Kind:    config.TextBody,
				Compare: config.Contains,
				Target:  "OK",
			},
		}

		if !cmp.Equal(expected, assertions) {
			t.Errorf("Expected %v, got %v", expected, assertions)
		}
	})

	t.Run("Empty assertions slice", func(t *testing.T) {
		assertions := []config.Assertion{}
		config.ConvertAssertionTargets(assertions)
		if len(assertions) != 0 {
			t.Errorf("Expected empty slice, got %v", assertions)
		}
	})
}
