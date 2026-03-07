package attach

import (
	"path/filepath"
	"strings"
)

// extensionMap maps file extensions to (FileType, label).
var extensionMap = map[string]struct {
	fileType FileType
	label    string
}{
	".log":   {FileTypeLog, "[log]"},
	".txt":   {FileTypeText, "[text]"},
	".md":    {FileTypeText, "[text]"},
	".json":  {FileTypeStructured, "[json]"},
	".yaml":  {FileTypeStructured, "[yaml]"},
	".yml":   {FileTypeStructured, "[yaml]"},
	".toml":  {FileTypeStructured, "[toml]"},
	".png":   {FileTypeImage, "[image]"},
	".jpg":   {FileTypeImage, "[image]"},
	".jpeg":  {FileTypeImage, "[image]"},
	".svg":   {FileTypeImage, "[image]"},
	".gif":   {FileTypeImage, "[image]"},
	".go":    {FileTypeCode, "[code]"},
	".ts":    {FileTypeCode, "[code]"},
	".js":    {FileTypeCode, "[code]"},
	".py":    {FileTypeCode, "[code]"},
	".rs":    {FileTypeCode, "[code]"},
	".rb":    {FileTypeCode, "[code]"},
	".sh":    {FileTypeCode, "[code]"},
	".swift": {FileTypeCode, "[code]"},
}

// ClassifyExtension returns the FileType and display label for a filename.
func ClassifyExtension(name string) (FileType, string) {
	ext := strings.ToLower(filepath.Ext(name))
	if entry, ok := extensionMap[ext]; ok {
		return entry.fileType, entry.label
	}
	return FileTypeBinary, "[file]"
}

// IsTextType returns true if the file type supports line counting and excerpts.
func IsTextType(ft FileType) bool {
	switch ft {
	case FileTypeText, FileTypeLog, FileTypeStructured, FileTypeCode:
		return true
	default:
		return false
	}
}
