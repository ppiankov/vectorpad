package sidecar

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestEncodeProjectPath(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"/Users/pashah/dev/myproject", "-Users-pashah-dev-myproject"},
		{"/home/user/code", "-home-user-code"},
	}
	for _, tt := range tests {
		got := encodeProjectPath(tt.input)
		if got != tt.want {
			t.Errorf("encodeProjectPath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestReadStats(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	entries := []Entry{
		{
			Type:      "user",
			UUID:      "uuid-1",
			Timestamp: time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC),
			SessionID: "sess-1",
			Message:   &Message{Role: "user", Content: json.RawMessage(`"Hello"`)},
		},
		{
			Type:       "assistant",
			UUID:       "uuid-2",
			ParentUUID: "uuid-1",
			Timestamp:  time.Date(2026, 3, 28, 10, 0, 1, 0, time.UTC),
			SessionID:  "sess-1",
			Message:    &Message{Role: "assistant", Content: json.RawMessage(`"Hi there"`)},
		},
		{
			Type:       "user",
			UUID:       "uuid-3",
			ParentUUID: "uuid-2",
			Timestamp:  time.Date(2026, 3, 28, 10, 0, 2, 0, time.UTC),
			SessionID:  "sess-1",
			Message:    &Message{Role: "user", Content: json.RawMessage(`"How are you?"`)},
		},
	}

	writeTestJSONL(t, path, entries)

	stats, err := ReadStats(path)
	if err != nil {
		t.Fatalf("ReadStats: %v", err)
	}

	if stats.TurnCount != 3 {
		t.Errorf("TurnCount = %d, want 3", stats.TurnCount)
	}
	if stats.UserTurns != 2 {
		t.Errorf("UserTurns = %d, want 2", stats.UserTurns)
	}
	if stats.AssistTurns != 1 {
		t.Errorf("AssistTurns = %d, want 1", stats.AssistTurns)
	}
	if stats.LastUUID != "uuid-3" {
		t.Errorf("LastUUID = %q, want uuid-3", stats.LastUUID)
	}
	if stats.SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want sess-1", stats.SessionID)
	}
}

func TestLastEntryUUID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	entries := []Entry{
		{Type: "user", UUID: "uuid-1", Timestamp: time.Now()},
		{Type: "assistant", UUID: "uuid-2", ParentUUID: "uuid-1", Timestamp: time.Now()},
		{Type: "user", UUID: "uuid-3", ParentUUID: "uuid-2", Timestamp: time.Now()},
	}

	writeTestJSONL(t, path, entries)

	got, err := LastEntryUUID(path)
	if err != nil {
		t.Fatalf("LastEntryUUID: %v", err)
	}
	if got != "uuid-3" {
		t.Errorf("got %q, want uuid-3", got)
	}
}

func TestLastEntryUUIDEmpty(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := LastEntryUUID(path)
	if err == nil {
		t.Fatal("expected error for empty file")
	}
}

func writeTestJSONL(t *testing.T, path string, entries []Entry) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	for _, e := range entries {
		line, err := json.Marshal(e)
		if err != nil {
			t.Fatal(err)
		}
		line = append(line, '\n')
		if _, err := f.Write(line); err != nil {
			t.Fatal(err)
		}
	}
}
