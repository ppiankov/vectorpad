package tui

import (
	"strings"
	"testing"
)

func TestNewLaunchOverlay(t *testing.T) {
	o := newLaunchOverlay()
	if len(o.targets) != 6 {
		t.Errorf("expected 6 targets, got %d", len(o.targets))
	}
	// Clipboard is always available.
	if !o.targets[0].available {
		t.Error("clipboard should always be available")
	}
	// File target is always available.
	if !o.targets[4].available {
		t.Error("file target should always be available")
	}
}

func TestLaunchOverlayShowDismiss(t *testing.T) {
	o := newLaunchOverlay()
	if o.visible {
		t.Error("should start hidden")
	}
	o.show()
	if !o.visible {
		t.Error("should be visible after show")
	}
	o.dismiss()
	if o.visible {
		t.Error("should be hidden after dismiss")
	}
}

func TestLaunchOverlayNavigation(t *testing.T) {
	o := newLaunchOverlay()
	o.show()

	if o.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", o.cursor)
	}

	o.moveDown()
	if o.cursor != 1 {
		t.Errorf("expected cursor at 1, got %d", o.cursor)
	}

	o.moveUp()
	if o.cursor != 0 {
		t.Errorf("expected cursor at 0, got %d", o.cursor)
	}

	// Should not go below 0.
	o.moveUp()
	if o.cursor != 0 {
		t.Errorf("expected cursor clamped at 0, got %d", o.cursor)
	}
}

func TestLaunchOverlaySelectByKey(t *testing.T) {
	o := newLaunchOverlay()

	target := o.selectByKey("1")
	if target == nil || target.name != "Clipboard" {
		t.Error("expected Clipboard for key 1")
	}

	target = o.selectByKey("5")
	if target == nil || target.name != "File (prompt.md)" {
		t.Error("expected File for key 5")
	}

	target = o.selectByKey("6")
	if target == nil || target.name != "VectorCourt" {
		t.Error("expected VectorCourt for key 6")
	}

	target = o.selectByKey("9")
	if target != nil {
		t.Error("expected nil for unknown key")
	}
}

func TestLaunchOverlayView(t *testing.T) {
	o := newLaunchOverlay()
	o.show()

	view := o.View()
	if !strings.Contains(view, "Launch To...") {
		t.Error("expected title in view")
	}
	if !strings.Contains(view, "Clipboard") {
		t.Error("expected Clipboard in view")
	}
}

func TestLaunchOverlayViewHidden(t *testing.T) {
	o := newLaunchOverlay()
	if o.View() != "" {
		t.Error("hidden overlay should return empty string")
	}
}

func TestCliExists(t *testing.T) {
	// "go" should always exist in test environment.
	if !cliExists("go") {
		t.Error("expected 'go' to be found in PATH")
	}
	if cliExists("__nonexistent_binary_xyzzy__") {
		t.Error("expected nonexistent binary to not be found")
	}
}
