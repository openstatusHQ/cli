package terraform

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckExistingFiles_RefusesExisting(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "monitors.tf"), []byte("existing"), 0o644); err != nil {
		t.Fatalf("seeding fixture: %v", err)
	}

	err := checkExistingFiles(dir, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "monitors.tf") {
		t.Errorf("expected error to mention filename, got: %v", err)
	}
	if !strings.Contains(err.Error(), "--force") {
		t.Errorf("expected error to mention --force, got: %v", err)
	}
}

func TestCheckExistingFiles_OverwritesWithForce(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "monitors.tf"), []byte("existing"), 0o644); err != nil {
		t.Fatalf("seeding fixture: %v", err)
	}

	if err := checkExistingFiles(dir, true); err != nil {
		t.Errorf("expected nil with force=true, got: %v", err)
	}
}

func TestCheckExistingFiles_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	if err := checkExistingFiles(dir, false); err != nil {
		t.Errorf("expected nil for empty dir, got: %v", err)
	}
}

func TestCheckExistingFiles_NonexistentDir(t *testing.T) {
	if err := checkExistingFiles(filepath.Join(t.TempDir(), "does-not-exist"), false); err != nil {
		t.Errorf("expected nil for nonexistent dir, got: %v", err)
	}
}

func TestPrintSummary_IncludesInitUpgradeHint(t *testing.T) {
	out := captureStdout(t, func() {
		printSummary("/tmp/out", &WorkspaceData{})
	})
	if !strings.Contains(out, "terraform init -upgrade") {
		t.Errorf("expected init-upgrade hint, got:\n%s", out)
	}
	if !strings.Contains(out, "~> 0.2") {
		t.Errorf("expected version mention in hint, got:\n%s", out)
	}
}

func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()

	done := make(chan string, 1)
	go func() {
		b, _ := io.ReadAll(r)
		done <- string(b)
	}()

	fn()
	w.Close()
	return <-done
}
