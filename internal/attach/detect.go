package attach

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DetectPath checks if pasted text is a file path.
// Returns an Attachment if the text is a single-line path to an existing file.
// Returns nil if the text is not a path or the file doesn't exist.
func DetectPath(text string) *Attachment {
	trimmed := strings.TrimSpace(text)

	// Multi-line text is never a path.
	if strings.ContainsAny(trimmed, "\n\r") {
		return nil
	}

	// Must look like a path.
	if !looksLikePath(trimmed) {
		return nil
	}

	// Expand ~ to home directory.
	resolved := expandHome(trimmed)

	// Resolve to absolute path.
	abs, err := filepath.Abs(resolved)
	if err != nil {
		return nil
	}

	// File must exist.
	info, err := os.Stat(abs)
	if err != nil || info.IsDir() {
		return nil
	}

	name := filepath.Base(abs)
	fileType, label := ClassifyExtension(name)

	lines := -1
	if IsTextType(fileType) {
		lines = countLines(abs)
	}

	return &Attachment{
		Path:     abs,
		Name:     name,
		Type:     fileType,
		Label:    label,
		Size:     info.Size(),
		Lines:    lines,
		Modified: info.ModTime(),
	}
}

func looksLikePath(text string) bool {
	unquoted := strings.Trim(text, "\"'")

	if strings.HasPrefix(unquoted, "/") {
		return true
	}
	if strings.HasPrefix(unquoted, "~/") {
		return true
	}
	if strings.HasPrefix(unquoted, "./") {
		return true
	}
	if strings.HasPrefix(unquoted, "../") {
		return true
	}
	return false
}

func expandHome(path string) string {
	unquoted := strings.Trim(path, "\"'")
	if strings.HasPrefix(unquoted, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return unquoted
		}
		return filepath.Join(home, unquoted[2:])
	}
	return unquoted
}

func countLines(path string) int {
	f, err := os.Open(path)
	if err != nil {
		return -1
	}
	defer func() { _ = f.Close() }()

	count := 0
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		count++
	}
	return count
}

// FormatSize returns a human-readable file size.
func FormatSize(bytes int64) string {
	switch {
	case bytes >= 1024*1024:
		return fmt.Sprintf("%.1fMB", float64(bytes)/(1024*1024))
	case bytes >= 1024:
		return fmt.Sprintf("%.1fKB", float64(bytes)/1024)
	default:
		return fmt.Sprintf("%dB", bytes)
	}
}
