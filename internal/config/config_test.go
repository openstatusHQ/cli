package config_test

import (
	"os"
	"path/filepath"
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
		f, err := os.CreateTemp(t.TempDir(), "config*.yaml")
		if err != nil {
			t.Fatal(err)
		}

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
		f, err := os.CreateTemp(t.TempDir(), "invalid*.yaml")
		if err != nil {
			t.Fatal(err)
		}

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

func Test_ReadConfig_NoStatePollution(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "config1.yaml")
	if err := os.WriteFile(file1, []byte("tests:\n  ids:\n    - 1\n    - 2\n    - 3\n"), 0600); err != nil {
		t.Fatal(err)
	}

	file2 := filepath.Join(dir, "config2.yaml")
	if err := os.WriteFile(file2, []byte("tests:\n  ids:\n    - 4\n    - 5\n    - 6\n"), 0600); err != nil {
		t.Fatal(err)
	}

	out1, err := config.ReadConfig(file1)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(out1.Tests.Ids, []int{1, 2, 3}) {
		t.Errorf("First read: expected [1,2,3], got %v", out1.Tests.Ids)
	}

	out2, err := config.ReadConfig(file2)
	if err != nil {
		t.Fatal(err)
	}
	if !cmp.Equal(out2.Tests.Ids, []int{4, 5, 6}) {
		t.Errorf("Second read: expected [4,5,6], got %v", out2.Tests.Ids)
	}
}
