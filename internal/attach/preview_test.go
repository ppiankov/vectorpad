package attach

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestPreviewLogTailLines(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "app.log")
	content := "line1\nline2\nline3\nline4\nline5\nline6\nline7\n"
	if err := os.WriteFile(f, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	a := DetectPath(f)
	if a == nil {
		t.Fatal("expected attachment")
	}

	preview := Preview(a, 3)
	lines := strings.Split(preview, "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 tail lines, got %d: %q", len(lines), preview)
	}
	if lines[0] != "line5" {
		t.Errorf("expected tail to start at line5, got %q", lines[0])
	}
}

func TestPreviewCodeHeadLines(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "main.go")
	content := "package main\n\nimport \"fmt\"\n\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n"
	if err := os.WriteFile(f, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	a := DetectPath(f)
	if a == nil {
		t.Fatal("expected attachment")
	}

	preview := Preview(a, 3)
	lines := strings.Split(preview, "\n")
	if len(lines) != 3 {
		t.Errorf("expected 3 head lines, got %d", len(lines))
	}
	if lines[0] != "package main" {
		t.Errorf("expected head to start with 'package main', got %q", lines[0])
	}
}

func TestPreviewImageReturnsMetadata(t *testing.T) {
	a := &Attachment{
		Path:  "/tmp/photo.png",
		Name:  "photo.png",
		Type:  FileTypeImage,
		Size:  131072,
		Lines: -1,
	}
	preview := Preview(a, 0)
	if !strings.Contains(preview, "photo.png") {
		t.Errorf("expected image name in preview, got %q", preview)
	}
	if !strings.Contains(preview, "128.0KB") {
		t.Errorf("expected size in preview, got %q", preview)
	}
}

func TestPreviewNilAttachment(t *testing.T) {
	result := Preview(nil, 5)
	if result != "" {
		t.Errorf("expected empty for nil, got %q", result)
	}
}

func TestRenderCardTextFile(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "data.json")
	if err := os.WriteFile(f, []byte(`{"key": "value"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	a := DetectPath(f)
	if a == nil {
		t.Fatal("expected attachment")
	}

	card := RenderCard(a, 3)
	if !strings.Contains(card, "[json]") {
		t.Errorf("expected [json] label in card, got %q", card)
	}
	if !strings.Contains(card, "data.json") {
		t.Errorf("expected filename in card, got %q", card)
	}
}
