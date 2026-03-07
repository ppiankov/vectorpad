package tui

import (
	"os/exec"
	"runtime"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestNewSpike(t *testing.T) {
	m := NewSpike()
	if m.status != statusEditing {
		t.Errorf("expected statusEditing, got %d", m.status)
	}
}

func TestCopyEmptyContent(t *testing.T) {
	m := NewSpike()

	// Simulate ctrl+y with empty content
	updated, _ := m.Update(tea.KeyMsg{Type: tea.KeyCtrlY})
	um := updated.(model)
	if um.status != statusError {
		t.Errorf("expected statusError for empty copy, got %d", um.status)
	}
	if um.message != "nothing to copy" {
		t.Errorf("expected 'nothing to copy', got %q", um.message)
	}
}

func TestCopyToClipboardRoundTrip(t *testing.T) {
	if runtime.GOOS != "darwin" && runtime.GOOS != "linux" {
		t.Skip("clipboard test only runs on darwin/linux")
	}
	if runtime.GOOS == "darwin" {
		if _, err := exec.LookPath("pbcopy"); err != nil {
			t.Skip("pbcopy not available")
		}
	}

	payload := "line 1\nline 2 with special: <>&\"'\nline 3\n"
	if err := copyToClipboard(payload); err != nil {
		t.Fatalf("copyToClipboard failed: %v", err)
	}

	// Verify round-trip
	if runtime.GOOS == "darwin" {
		out, err := exec.Command("pbpaste").Output()
		if err != nil {
			t.Fatalf("pbpaste failed: %v", err)
		}
		if string(out) != payload {
			t.Errorf("clipboard mismatch:\nwant: %q\ngot:  %q", payload, string(out))
		}
	}
}
