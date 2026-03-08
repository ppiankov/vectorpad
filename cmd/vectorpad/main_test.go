package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVersionCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(
		[]string{"vectorpad", "version"},
		strings.NewReader(""),
		&stdout,
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "vectorpad") {
		t.Fatalf("expected version output, got %q", stdout.String())
	}
}

func TestRunReadsStdinAndRendersVectorBlock(t *testing.T) {
	input := "Do not change file permissions. We will ship this today. Maybe we can batch writes."
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(
		[]string{"vectorpad"},
		strings.NewReader(input),
		&stdout,
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}

	output := stdout.String()
	expected := []string{
		"VECTOR",
		"Constraints:",
		"[CONSTRAINT][LOCKED] Do not change file permissions.",
		"Decisions:",
		"[DECISION][LOCKED] We will ship this today.",
		"Tentatives:",
		"[TENTATIVE][LOCKED:Maybe] Maybe we can batch writes.",
		"PREFLIGHT",
		"AMBIGUITY",
		"PREFLIGHT_JSON",
		"AMBIGUITY_JSON",
		"\"token_weight\"",
		"\"vector_integrity\"",
		"\"blast_radius\"",
	}

	for _, fragment := range expected {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected output to contain %q, got:\n%s", fragment, output)
		}
	}

	preflightJSON := extractJSONSection(t, output, "PREFLIGHT_JSON", "AMBIGUITY_JSON")
	var decoded map[string]any
	if err := json.Unmarshal([]byte(preflightJSON), &decoded); err != nil {
		t.Fatalf("expected parseable preflight json, got %v", err)
	}

	ambiguityJSON := extractJSONSection(t, output, "AMBIGUITY_JSON", "PREFLIGHT_JSON")
	var ambiguityDecoded map[string]any
	if err := json.Unmarshal([]byte(ambiguityJSON), &ambiguityDecoded); err != nil {
		t.Fatalf("expected parseable ambiguity json, got %v", err)
	}
}

func TestRunWithEmptyStdin(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(
		[]string{"vectorpad"},
		strings.NewReader("  \n  "),
		&stdout,
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "VECTOR") {
		t.Fatalf("expected vector block output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "PREFLIGHT") {
		t.Fatalf("expected preflight human output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "AMBIGUITY") {
		t.Fatalf("expected ambiguity human output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "PREFLIGHT_JSON") {
		t.Fatalf("expected preflight json output, got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "AMBIGUITY_JSON") {
		t.Fatalf("expected ambiguity json output, got %q", stdout.String())
	}
}

func TestRunAddCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("VECTORPAD_HOME", home)

	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(
		[]string{"vectorpad", "add", "context dilution attack vector"},
		strings.NewReader("this should not be read"),
		&stdout,
		&stderr,
	)
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stderr.Len() != 0 {
		t.Fatalf("expected no stderr output, got %q", stderr.String())
	}
	if !strings.Contains(stdout.String(), "stashed ") {
		t.Fatalf("expected stashed output, got %q", stdout.String())
	}

	// Verify item was persisted (SQLite DB or JSON).
	dbPath := filepath.Join(home, "stash.db")
	jsonPath := filepath.Join(home, "stash", "stacks.json")
	_, dbErr := os.Stat(dbPath)
	_, jsonErr := os.Stat(jsonPath)
	if dbErr != nil && jsonErr != nil {
		t.Fatalf("expected stash db or json file to be written, db: %v, json: %v", dbErr, jsonErr)
	}
}

func TestRunAddCommandRequiresText(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	exitCode := run(
		[]string{"vectorpad", "add"},
		strings.NewReader(""),
		&stdout,
		&stderr,
	)
	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if stdout.Len() != 0 {
		t.Fatalf("expected no stdout output, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "add requires an idea string") {
		t.Fatalf("expected usage error on stderr, got %q", stderr.String())
	}
}

func extractJSONSection(t *testing.T, output string, marker string, nextMarkers ...string) string {
	t.Helper()

	markerWithNewline := marker + "\n"
	index := strings.Index(output, markerWithNewline)
	if index == -1 {
		t.Fatalf("expected output to contain %q marker", markerWithNewline)
	}

	section := output[index+len(markerWithNewline):]
	end := len(section)
	for _, nextMarker := range nextMarkers {
		needle := "\n" + nextMarker + "\n"
		nextIndex := strings.Index(section, needle)
		if nextIndex != -1 && nextIndex < end {
			end = nextIndex
		}
	}

	return strings.TrimSpace(section[:end])
}
