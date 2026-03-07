package attach

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRenderObjectCardTextFile(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "config.json")
	if err := os.WriteFile(f, []byte(`{"port": 8080}`), 0o600); err != nil {
		t.Fatal(err)
	}

	a := DetectPath(f)
	if a == nil {
		t.Fatal("expected attachment")
	}

	card := RenderObjectCard(a, 3, 60)
	if !strings.Contains(card, "┌") {
		t.Error("expected box border in card")
	}
	if !strings.Contains(card, "config.json") {
		t.Errorf("expected filename in card, got:\n%s", card)
	}
}

func TestRenderObjectCardNilAttachment(t *testing.T) {
	result := RenderObjectCard(nil, 3, 60)
	if result != "" {
		t.Errorf("expected empty for nil, got %q", result)
	}
}

func TestRenderObjectCardNarrowWidth(t *testing.T) {
	a := &Attachment{
		Path:  "/tmp/test.txt",
		Name:  "test.txt",
		Type:  FileTypeText,
		Label: "[text]",
		Size:  100,
		Lines: 5,
	}
	result := RenderObjectCard(a, 3, 10)
	if result != "" {
		t.Errorf("expected empty for narrow width, got %q", result)
	}
}

func TestRenderObjectCardImageNoPreview(t *testing.T) {
	a := &Attachment{
		Path:  "/tmp/photo.png",
		Name:  "photo.png",
		Type:  FileTypeImage,
		Label: "[image]",
		Size:  131072,
		Lines: -1,
	}
	card := RenderObjectCard(a, 0, 60)
	if !strings.Contains(card, "[image]") {
		t.Error("expected [image] label")
	}
	// Image cards should have only top border, metadata, bottom border — no inner separator.
	lines := strings.Split(card, "\n")
	// Top border + metadata + bottom border = 3 lines.
	if len(lines) != 3 {
		t.Errorf("expected 3 lines for image card, got %d:\n%s", len(lines), card)
	}
}
