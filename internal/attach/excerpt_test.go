package attach

import "testing"

func TestDefaultExcerptConfigLog(t *testing.T) {
	a := &Attachment{Type: FileTypeLog}
	cfg := DefaultExcerptConfig(a)
	if cfg.Mode != SerializeExcerpt {
		t.Errorf("expected excerpt mode, got %s", cfg.Mode)
	}
	if cfg.Lines != defaultTailLines {
		t.Errorf("expected %d lines for log, got %d", defaultTailLines, cfg.Lines)
	}
}

func TestDefaultExcerptConfigCode(t *testing.T) {
	a := &Attachment{Type: FileTypeCode}
	cfg := DefaultExcerptConfig(a)
	if cfg.Lines != defaultHeadLines {
		t.Errorf("expected %d lines for code, got %d", defaultHeadLines, cfg.Lines)
	}
}

func TestDefaultExcerptConfigNil(t *testing.T) {
	cfg := DefaultExcerptConfig(nil)
	if cfg.Mode != SerializeExcerpt {
		t.Errorf("expected excerpt mode for nil, got %s", cfg.Mode)
	}
}
