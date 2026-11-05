package terraform

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	monitorv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/monitor/v1"
	private_locationv1 "buf.build/gen/go/openstatus/api/protocolbuffers/go/openstatus/private_location/v1"
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

func TestCheckExistingFiles_RefusesExistingPrivateLocations(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "private_locations.tf"), []byte("existing"), 0o644); err != nil {
		t.Fatalf("seeding fixture: %v", err)
	}

	err := checkExistingFiles(dir, false)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "private_locations.tf") {
		t.Errorf("expected error to mention filename, got: %v", err)
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
		printSummary("/tmp/out", NewGenerator(&WorkspaceData{}))
	})
	if !strings.Contains(out, "terraform init -upgrade") {
		t.Errorf("expected init-upgrade hint, got:\n%s", out)
	}
	if !strings.Contains(out, "~> 0.6") {
		t.Errorf("expected version mention in hint, got:\n%s", out)
	}
}

func TestPrintSummary_CountsPrivateLocations(t *testing.T) {
	l := &private_locationv1.PrivateLocation{}
	l.SetId("pl_1")
	l.SetName("office-paris")

	out := captureStdout(t, func() {
		printSummary("/tmp/out", NewGenerator(&WorkspaceData{
			PrivateLocations: []*private_locationv1.PrivateLocation{l},
		}))
	})

	if !strings.Contains(out, "1 private locations") {
		t.Errorf("expected private location count, got:\n%s", out)
	}
	if !strings.Contains(out, "1 import blocks") {
		t.Errorf("expected private location to count toward imports, got:\n%s", out)
	}
}

func TestPrintSummary_CountsICMPMonitors(t *testing.T) {
	m := &monitorv1.ICMPMonitor{}
	m.SetId("777")
	m.SetName("Gateway Ping")

	out := captureStdout(t, func() {
		printSummary("/tmp/out", NewGenerator(&WorkspaceData{
			ICMPMonitors: []*monitorv1.ICMPMonitor{m},
		}))
	})

	if !strings.Contains(out, "1 monitors (0 HTTP, 0 TCP, 0 DNS, 1 ICMP, 0 gRPC, 0 heartbeat)") {
		t.Errorf("expected ICMP monitor count, got:\n%s", out)
	}
	if !strings.Contains(out, "1 import blocks") {
		t.Errorf("expected ICMP monitor to count toward imports, got:\n%s", out)
	}
}

func TestPrintSummary_CountsHeartbeatMonitors(t *testing.T) {
	m := &monitorv1.HeartbeatMonitor{}
	m.SetId("888")
	m.SetName("Nightly backup")

	out := captureStdout(t, func() {
		printSummary("/tmp/out", NewGenerator(&WorkspaceData{
			HeartbeatMonitors: []*monitorv1.HeartbeatMonitor{m},
		}))
	})

	if !strings.Contains(out, "1 monitors (0 HTTP, 0 TCP, 0 DNS, 0 ICMP, 0 gRPC, 1 heartbeat)") {
		t.Errorf("expected heartbeat monitor count, got:\n%s", out)
	}
	if !strings.Contains(out, "1 import blocks") {
		t.Errorf("expected heartbeat monitor to count toward imports, got:\n%s", out)
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
