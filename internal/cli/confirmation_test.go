package cli_test

import (
	"os"
	"testing"

	"github.com/openstatusHQ/cli/internal/cli"
)

func Test_AskForConfirmation(t *testing.T) {
	t.Run("Returns true for 'y' input", func(t *testing.T) {
		// Create a pipe to simulate stdin
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}

		// Save original stdin
		oldStdin := os.Stdin
		os.Stdin = r

		// Write the input
		go func() {
			w.WriteString("y\n")
			w.Close()
		}()

		// Restore stdin after test
		t.Cleanup(func() {
			os.Stdin = oldStdin
		})

		result, err := cli.AskForConfirmation("Test prompt")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !result {
			t.Error("Expected true for 'y' input, got false")
		}
	})

	t.Run("Returns true for 'yes' input", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}

		oldStdin := os.Stdin
		os.Stdin = r

		go func() {
			w.WriteString("yes\n")
			w.Close()
		}()

		t.Cleanup(func() {
			os.Stdin = oldStdin
		})

		result, err := cli.AskForConfirmation("Test prompt")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !result {
			t.Error("Expected true for 'yes' input, got false")
		}
	})

	t.Run("Returns true for 'Y' input (case insensitive)", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}

		oldStdin := os.Stdin
		os.Stdin = r

		go func() {
			w.WriteString("Y\n")
			w.Close()
		}()

		t.Cleanup(func() {
			os.Stdin = oldStdin
		})

		result, err := cli.AskForConfirmation("Test prompt")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !result {
			t.Error("Expected true for 'Y' input, got false")
		}
	})

	t.Run("Returns true for 'YES' input (case insensitive)", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}

		oldStdin := os.Stdin
		os.Stdin = r

		go func() {
			w.WriteString("YES\n")
			w.Close()
		}()

		t.Cleanup(func() {
			os.Stdin = oldStdin
		})

		result, err := cli.AskForConfirmation("Test prompt")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if !result {
			t.Error("Expected true for 'YES' input, got false")
		}
	})

	t.Run("Returns false for 'n' input", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}

		oldStdin := os.Stdin
		os.Stdin = r

		go func() {
			w.WriteString("n\n")
			w.Close()
		}()

		t.Cleanup(func() {
			os.Stdin = oldStdin
		})

		result, err := cli.AskForConfirmation("Test prompt")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result {
			t.Error("Expected false for 'n' input, got true")
		}
	})

	t.Run("Returns false for 'no' input", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}

		oldStdin := os.Stdin
		os.Stdin = r

		go func() {
			w.WriteString("no\n")
			w.Close()
		}()

		t.Cleanup(func() {
			os.Stdin = oldStdin
		})

		result, err := cli.AskForConfirmation("Test prompt")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result {
			t.Error("Expected false for 'no' input, got true")
		}
	})

	t.Run("Returns false for empty input", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}

		oldStdin := os.Stdin
		os.Stdin = r

		go func() {
			w.WriteString("\n")
			w.Close()
		}()

		t.Cleanup(func() {
			os.Stdin = oldStdin
		})

		result, err := cli.AskForConfirmation("Test prompt")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result {
			t.Error("Expected false for empty input, got true")
		}
	})

	t.Run("Returns false for random input", func(t *testing.T) {
		r, w, err := os.Pipe()
		if err != nil {
			t.Fatal(err)
		}

		oldStdin := os.Stdin
		os.Stdin = r

		go func() {
			w.WriteString("maybe\n")
			w.Close()
		}()

		t.Cleanup(func() {
			os.Stdin = oldStdin
		})

		result, err := cli.AskForConfirmation("Test prompt")
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if result {
			t.Error("Expected false for random input, got true")
		}
	})
}
