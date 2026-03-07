package attach

import "testing"

func TestClassifyExtensionKnownTypes(t *testing.T) {
	tests := []struct {
		name     string
		wantType FileType
		wantLbl  string
	}{
		{"app.log", FileTypeLog, "[log]"},
		{"readme.md", FileTypeText, "[text]"},
		{"config.json", FileTypeStructured, "[json]"},
		{"values.yaml", FileTypeStructured, "[yaml]"},
		{"values.yml", FileTypeStructured, "[yaml]"},
		{"diagram.png", FileTypeImage, "[image]"},
		{"photo.jpg", FileTypeImage, "[image]"},
		{"main.go", FileTypeCode, "[code]"},
		{"app.ts", FileTypeCode, "[code]"},
		{"script.py", FileTypeCode, "[code]"},
		{"lib.rs", FileTypeCode, "[code]"},
		{"app.swift", FileTypeCode, "[code]"},
	}

	for _, tt := range tests {
		ft, label := ClassifyExtension(tt.name)
		if ft != tt.wantType {
			t.Errorf("ClassifyExtension(%q) type = %s, want %s", tt.name, ft, tt.wantType)
		}
		if label != tt.wantLbl {
			t.Errorf("ClassifyExtension(%q) label = %s, want %s", tt.name, label, tt.wantLbl)
		}
	}
}

func TestClassifyExtensionUnknownType(t *testing.T) {
	ft, label := ClassifyExtension("data.parquet")
	if ft != FileTypeBinary {
		t.Errorf("expected binary, got %s", ft)
	}
	if label != "[file]" {
		t.Errorf("expected [file], got %s", label)
	}
}

func TestClassifyExtensionCaseInsensitive(t *testing.T) {
	ft, _ := ClassifyExtension("README.MD")
	if ft != FileTypeText {
		t.Errorf("expected text for .MD, got %s", ft)
	}
}

func TestIsTextType(t *testing.T) {
	textTypes := []FileType{FileTypeText, FileTypeLog, FileTypeStructured, FileTypeCode}
	for _, ft := range textTypes {
		if !IsTextType(ft) {
			t.Errorf("expected IsTextType(%s) = true", ft)
		}
	}

	nonTextTypes := []FileType{FileTypeImage, FileTypeBinary}
	for _, ft := range nonTextTypes {
		if IsTextType(ft) {
			t.Errorf("expected IsTextType(%s) = false", ft)
		}
	}
}
