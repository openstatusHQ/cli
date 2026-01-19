package config_test

import (
	"os"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/openstatusHQ/cli/internal/config"
)

var configFile = `
tests:
  ids:
    - 1
    - 2
    - 3
`

func Test_ReadConfig(t *testing.T) {
	t.Run("Read valid config file", func(t *testing.T) {
		f, err := os.CreateTemp(".", "config*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())

		if _, err := f.Write([]byte(configFile)); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}

		out, err := config.ReadConfig(f.Name())
		if err != nil {
			t.Fatal(err)
		}

		expect := &config.Config{
			Tests: config.TestsConfig{
				Ids: []int{1, 2, 3},
			},
		}

		if !cmp.Equal(expect, out) {
			t.Errorf("Expected %v, got %v", expect, out)
		}
	})

	t.Run("File does not exist", func(t *testing.T) {
		_, err := config.ReadConfig("nonexistent.yaml")
		if err == nil {
			t.Error("Expected error for non-existent file, got nil")
		}
	})

	t.Run("Invalid YAML content", func(t *testing.T) {
		f, err := os.CreateTemp(".", "invalid*.yaml")
		if err != nil {
			t.Fatal(err)
		}
		defer os.Remove(f.Name())

		if _, err := f.Write([]byte("invalid: yaml: content: [")); err != nil {
			t.Fatal(err)
		}
		if err := f.Close(); err != nil {
			t.Fatal(err)
		}

		_, err = config.ReadConfig(f.Name())
		if err == nil {
			t.Error("Expected error for invalid YAML, got nil")
		}
	})
}
