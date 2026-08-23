package audit

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestWrite(t *testing.T) {
	directory := t.TempDir()
	logger, err := New(directory)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 8, 24, 1, 2, 3, 0, time.UTC)
	logger.Write(Event{Time: now, User: "admin", IP: "127.0.0.1", Method: "GET", Path: "/api/v1/files", Status: 200})
	data, err := os.ReadFile(filepath.Join(directory, "2026-08-24.jsonl"))
	if err != nil || !strings.Contains(string(data), `"user":"admin"`) {
		t.Fatalf("log = %q, %v", data, err)
	}
}
