package sidecar

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestInjectUserMessage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	// Seed with one assistant entry.
	seed := []Entry{
		{
			Type:      "assistant",
			UUID:      "parent-uuid",
			Timestamp: time.Date(2026, 3, 28, 10, 0, 0, 0, time.UTC),
			Message:   &Message{Role: "assistant", Content: json.RawMessage(`"I can help with that."`)},
		},
	}
	writeTestJSONL(t, path, seed)

	// Inject a user message.
	err := InjectUserMessage(path, "What is the status of the deployment?")
	if err != nil {
		t.Fatalf("InjectUserMessage: %v", err)
	}

	// Read back and verify.
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var entries []Entry
	for scanner.Scan() {
		var e Entry
		if json.Unmarshal(scanner.Bytes(), &e) == nil {
			entries = append(entries, e)
		}
	}

	if len(entries) != 2 {
		t.Fatalf("entry count = %d, want 2", len(entries))
	}

	injected := entries[1]
	if injected.Type != "user" {
		t.Errorf("type = %q, want user", injected.Type)
	}
	if injected.ParentUUID != "parent-uuid" {
		t.Errorf("parentUuid = %q, want parent-uuid", injected.ParentUUID)
	}
	if injected.Message == nil {
		t.Fatal("message is nil")
	}
	if injected.Message.Role != "user" {
		t.Errorf("role = %q, want user", injected.Message.Role)
	}

	// Content should be a JSON string.
	var text string
	if err := json.Unmarshal(injected.Message.Content, &text); err != nil {
		t.Fatalf("unmarshal content: %v", err)
	}
	if text != "What is the status of the deployment?" {
		t.Errorf("content = %q", text)
	}
}

func TestInjectUserMessageWithParent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	// Create empty file.
	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	err := InjectUserMessageWithParent(path, "precise question here", "specific-parent-uuid")
	if err != nil {
		t.Fatalf("InjectUserMessageWithParent: %v", err)
	}

	// Read back.
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	var e Entry
	if scanner.Scan() {
		if err := json.Unmarshal(scanner.Bytes(), &e); err != nil {
			t.Fatal(err)
		}
	}

	if e.ParentUUID != "specific-parent-uuid" {
		t.Errorf("parentUuid = %q, want specific-parent-uuid", e.ParentUUID)
	}
	if e.Type != "user" {
		t.Errorf("type = %q, want user", e.Type)
	}
}

func TestInjectPreservesExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	// Seed with three entries.
	seed := []Entry{
		{Type: "user", UUID: "u1", Timestamp: time.Now(), Message: &Message{Role: "user", Content: json.RawMessage(`"q1"`)}},
		{Type: "assistant", UUID: "a1", ParentUUID: "u1", Timestamp: time.Now(), Message: &Message{Role: "assistant", Content: json.RawMessage(`"r1"`)}},
		{Type: "user", UUID: "u2", ParentUUID: "a1", Timestamp: time.Now(), Message: &Message{Role: "user", Content: json.RawMessage(`"q2"`)}},
	}
	writeTestJSONL(t, path, seed)

	// Inject.
	if err := InjectUserMessage(path, "injected"); err != nil {
		t.Fatal(err)
	}

	// Count lines.
	f, err := os.Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)
	count := 0
	for scanner.Scan() {
		if len(scanner.Bytes()) > 0 {
			count++
		}
	}

	if count != 4 {
		t.Errorf("line count = %d, want 4", count)
	}
}
