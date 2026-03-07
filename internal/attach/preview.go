package attach

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

const (
	defaultTailLines = 5 // log files: show last N lines
	defaultHeadLines = 8 // structured/code: show first N lines
)

// Preview generates an excerpt from a file for display in the TUI.
func Preview(a *Attachment, maxLines int) string {
	if a == nil {
		return ""
	}

	switch a.Type {
	case FileTypeLog:
		if maxLines <= 0 {
			maxLines = defaultTailLines
		}
		return tailLines(a.Path, maxLines)
	case FileTypeText, FileTypeCode, FileTypeStructured:
		if maxLines <= 0 {
			maxLines = defaultHeadLines
		}
		return headLines(a.Path, maxLines)
	case FileTypeImage:
		return fmt.Sprintf("%s  %s", a.Name, FormatSize(a.Size))
	default:
		return fmt.Sprintf("%s  %s", a.Name, FormatSize(a.Size))
	}
}

// RenderCard generates an object card for display in the editor panel.
func RenderCard(a *Attachment, maxLines int) string {
	if a == nil {
		return ""
	}

	var b strings.Builder

	// Header line.
	meta := FormatSize(a.Size)
	if a.Lines >= 0 {
		meta += fmt.Sprintf("  %d lines", a.Lines)
	}
	fmt.Fprintf(&b, "%s %s  %s\n", a.Label, a.Name, meta)

	// Preview content.
	preview := Preview(a, maxLines)
	if preview != "" {
		for _, line := range strings.Split(preview, "\n") {
			fmt.Fprintf(&b, "  %s\n", line)
		}
	}

	return b.String()
}

func headLines(path string, n int) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() && len(lines) < n {
		lines = append(lines, scanner.Text())
	}
	return strings.Join(lines, "\n")
}

func tailLines(path string, n int) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()

	ring := make([]string, 0, n)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		if len(ring) >= n {
			ring = ring[1:]
		}
		ring = append(ring, scanner.Text())
	}
	return strings.Join(ring, "\n")
}
