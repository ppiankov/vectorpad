package attach

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDetectPathAbsolutePath(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "test.log")
	if err := os.WriteFile(f, []byte("line1\nline2\nline3\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	a := DetectPath(f)
	if a == nil {
		t.Fatal("expected attachment, got nil")
	}
	if a.Name != "test.log" {
		t.Errorf("expected name test.log, got %s", a.Name)
	}
	if a.Type != FileTypeLog {
		t.Errorf("expected log type, got %s", a.Type)
	}
	if a.Lines != 3 {
		t.Errorf("expected 3 lines, got %d", a.Lines)
	}
}

func TestDetectPathMultilineNotPath(t *testing.T) {
	a := DetectPath("/some/path\nand more text")
	if a != nil {
		t.Error("multi-line text should not be detected as path")
	}
}

func TestDetectPathNonExistentFile(t *testing.T) {
	a := DetectPath("/nonexistent/path/to/file.txt")
	if a != nil {
		t.Error("non-existent file should return nil")
	}
}

func TestDetectPathDirectory(t *testing.T) {
	a := DetectPath(t.TempDir())
	if a != nil {
		t.Error("directory should return nil")
	}
}

func TestDetectPathRegularText(t *testing.T) {
	a := DetectPath("this is a normal sentence about coding")
	if a != nil {
		t.Error("regular text should not be detected as path")
	}
}

func TestDetectPathQuotedPath(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "test file.txt")
	if err := os.WriteFile(f, []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}

	a := DetectPath("\"" + f + "\"")
	if a == nil {
		t.Fatal("expected attachment for quoted path, got nil")
	}
	if a.Name != "test file.txt" {
		t.Errorf("expected 'test file.txt', got %s", a.Name)
	}
}

func TestDetectPathRelativePath(t *testing.T) {
	a := DetectPath("./some/relative/path.go")
	// May or may not exist — just verify it doesn't panic.
	_ = a
}

func TestDetectPathImageBinary(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "photo.png")
	if err := os.WriteFile(f, []byte{0x89, 0x50, 0x4E, 0x47}, 0o600); err != nil {
		t.Fatal(err)
	}

	a := DetectPath(f)
	if a == nil {
		t.Fatal("expected attachment, got nil")
	}
	if a.Type != FileTypeImage {
		t.Errorf("expected image type, got %s", a.Type)
	}
	if a.Lines != -1 {
		t.Errorf("expected -1 lines for image, got %d", a.Lines)
	}
}
