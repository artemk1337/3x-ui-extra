package vkturnproxy

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func testManager(t *testing.T, script string) *Manager {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("test uses a POSIX shell fixture")
	}
	path := filepath.Join(t.TempDir(), "vk-turn-proxy")
	if err := os.WriteFile(path, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	hash := sha256.Sum256([]byte(script))
	m := &Manager{binaryPath: path, expectedHash: hex.EncodeToString(hash[:])}
	t.Cleanup(m.StopAll)
	return m
}

func TestReconcileLifecycle(t *testing.T) {
	m := testManager(t, "#!/bin/sh\nexec sleep 30\n")
	first := Spec{InboundID: 1, Listen: "0.0.0.0:40001", Connect: "127.0.0.1:51820"}
	second := Spec{InboundID: 2, Listen: "[::]:40002", Connect: "[::1]:51821"}
	if err := m.Reconcile([]Spec{first, second}); err != nil {
		t.Fatal(err)
	}
	statuses := m.Statuses()
	if len(statuses) != 2 || !statuses[0].Running || !statuses[1].Running {
		t.Fatalf("unexpected statuses after start: %+v", statuses)
	}
	firstProc := m.items[1].proc
	if err := m.Reconcile([]Spec{first, second}); err != nil {
		t.Fatal(err)
	}
	if m.items[1].proc != firstProc {
		t.Fatal("unchanged spec restarted process")
	}
	first.Listen = "0.0.0.0:40003"
	if err := m.Reconcile([]Spec{first}); err != nil {
		t.Fatal(err)
	}
	statuses = m.Statuses()
	if len(statuses) != 1 || statuses[0].Spec != first || !statuses[0].Running || m.items[1].proc == firstProc {
		t.Fatalf("unexpected statuses after replacement: %+v", statuses)
	}
	if firstProc.running() {
		t.Fatal("old process still running")
	}
	m.StopAll()
	if len(m.Statuses()) != 0 {
		t.Fatal("StopAll retained processes")
	}
}

func TestReconcileReportsBadSpecAndHash(t *testing.T) {
	m := testManager(t, "#!/bin/sh\nexec sleep 30\n")
	bad := Spec{InboundID: 1, Listen: "example.com:40001", Connect: "127.0.0.1:51820"}
	if err := m.Reconcile([]Spec{bad}); err == nil || !strings.Contains(err.Error(), "host") {
		t.Fatalf("invalid host error = %v", err)
	}
	if statuses := m.Statuses(); len(statuses) != 1 || statuses[0].Running || statuses[0].Error == "" {
		t.Fatalf("invalid spec status = %+v", statuses)
	}
	m.expectedHash = strings.Repeat("0", 64)
	valid := Spec{InboundID: 1, Listen: "0.0.0.0:40001", Connect: "127.0.0.1:51820"}
	if err := m.Reconcile([]Spec{valid}); err == nil || !strings.Contains(err.Error(), "SHA-256 mismatch") {
		t.Fatalf("hash error = %v", err)
	}
	if statuses := m.Statuses(); len(statuses) != 1 || statuses[0].Running || !strings.Contains(statuses[0].Error, "SHA-256 mismatch") {
		t.Fatalf("hash status = %+v", statuses)
	}
}

func TestReconcileRestartsExitedProcess(t *testing.T) {
	m := testManager(t, "#!/bin/sh\nexit 7\n")
	spec := Spec{InboundID: 1, Listen: "0.0.0.0:40001", Connect: "127.0.0.1:51820"}
	if err := m.Reconcile([]Spec{spec}); err != nil {
		t.Fatal(err)
	}
	old := m.items[1].proc
	select {
	case <-old.done:
	case <-time.After(15 * time.Second):
		t.Fatal("fixture process did not exit")
	}
	if statuses := m.Statuses(); len(statuses) != 1 || statuses[0].Running || statuses[0].Error == "" {
		t.Fatalf("exit status = %+v", statuses)
	}
	if err := m.Reconcile([]Spec{spec}); err != nil {
		t.Fatal(err)
	}
	if m.items[1].proc == old {
		t.Fatal("exited process was not replaced")
	}
}
