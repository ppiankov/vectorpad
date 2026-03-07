package attach

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSerializePathOnly(t *testing.T) {
	a := &Attachment{Path: "/tmp/test.log", Name: "test.log", Type: FileTypeLog}
	result := Serialize(a, SerializePathOnly, 0)
	if result != "[Attached: /tmp/test.log]" {
		t.Errorf("unexpected path-only: %q", result)
	}
}

func TestSerializeExcerptWithFile(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "test.log")
	content := "line1\nline2\nline3\n"
	if err := os.WriteFile(f, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}

	a := DetectPath(f)
	if a == nil {
		t.Fatal("expected attachment")
	}

	result := Serialize(a, SerializeExcerpt, 5)
	if !strings.Contains(result, "Attached log: test.log") {
		t.Errorf("expected metadata header, got: %s", result)
	}
	if !strings.Contains(result, "Relevant excerpt:") {
		t.Errorf("expected excerpt section, got: %s", result)
	}
}

func TestSerializeEvidenceWithFile(t *testing.T) {
	tmp := t.TempDir()
	f := filepath.Join(tmp, "data.json")
	if err := os.WriteFile(f, []byte(`{"key": "value"}`), 0o600); err != nil {
		t.Fatal(err)
	}

	a := DetectPath(f)
	if a == nil {
		t.Fatal("expected attachment")
	}

	result := Serialize(a, SerializeEvidence, 5)
	if !strings.Contains(result, "Evidence from data.json:") {
		t.Errorf("expected evidence header, got: %s", result)
	}
}

func TestSerializeImageNoExcerpt(t *testing.T) {
	a := &Attachment{
		Path:  "/tmp/photo.png",
		Name:  "photo.png",
		Type:  FileTypeImage,
		Size:  131072,
		Lines: -1,
	}
	result := Serialize(a, SerializeExcerpt, 0)
	if !strings.Contains(result, "Attached image: photo.png") {
		t.Errorf("expected image metadata, got: %s", result)
	}
	if strings.Contains(result, "excerpt") {
		t.Error("image should not have excerpt")
	}
}

func TestSerializeNilAttachment(t *testing.T) {
	result := Serialize(nil, SerializeExcerpt, 0)
	if result != "" {
		t.Errorf("expected empty string for nil attachment, got: %q", result)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes int64
		want  string
	}{
		{0, "0B"},
		{512, "512B"},
		{1024, "1.0KB"},
		{47104, "46.0KB"},
		{1048576, "1.0MB"},
		{131072, "128.0KB"},
	}
	for _, tt := range tests {
		got := FormatSize(tt.bytes)
		if got != tt.want {
			t.Errorf("FormatSize(%d) = %q, want %q", tt.bytes, got, tt.want)
		}
	}
}
