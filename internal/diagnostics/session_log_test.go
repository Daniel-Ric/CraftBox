package diagnostics

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSessionLogSnapshotKeepsOriginalSessionAndIncludesRotatedLogs(t *testing.T) {
	store, err := OpenStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	first, err := store.Start(Options{Version: "test", Role: "main", SegmentBytes: 512})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 8; i++ {
		first.Record("error", "connection", "failed", strings.Repeat("connection error ", 20), nil)
	}
	first.Close()
	second, err := store.Start(Options{Version: "test", Role: "main"})
	if err != nil {
		t.Fatal(err)
	}
	defer second.Close()
	path, err := store.SessionLogSnapshot(first.ID())
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)
	if !strings.HasPrefix(filepath.Base(path), first.ID()) {
		t.Fatalf("wrong log session: %s", path)
	}
	data, err := os.ReadFile(path)
	if err != nil || !strings.Contains(string(data), "connection error") {
		t.Fatalf("failed to open actual error log: %v", err)
	}
	if _, err := store.SessionLogSnapshot("../../other"); err == nil {
		t.Fatal("invalid session accepted")
	}
	if _, err := store.SessionLogSnapshot("session-20000101T000000Z-1-000000000000"); err == nil {
		t.Fatal("missing session fell back to current log")
	}
}
