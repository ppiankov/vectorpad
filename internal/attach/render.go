package attach

import (
	"fmt"
	"strings"
)

// RenderObjectCard generates a bordered card for display in the TUI editor panel.
// This is the Bubbletea-aware version with box drawing characters.
func RenderObjectCard(a *Attachment, maxLines int, width int) string {
	if a == nil || width < 20 {
		return ""
	}

	var b strings.Builder
	innerW := width - 4 // box borders + padding

	// Header: label + name + metadata
	meta := FormatSize(a.Size)
	if a.Lines >= 0 {
		meta += fmt.Sprintf("  %d lines", a.Lines)
	}
	// Top border
	topLabel := fmt.Sprintf("─ %s %s ", a.Label, a.Name)
	if len(topLabel) > innerW {
		topLabel = topLabel[:innerW]
	}
	b.WriteString("┌" + topLabel + strings.Repeat("─", max(0, innerW-len(topLabel))) + "┐\n")

	// Metadata line
	metaLine := fmt.Sprintf("  %s", meta)
	b.WriteString("│" + pad(metaLine, innerW) + "│\n")

	// Preview content for text types
	if IsTextType(a.Type) {
		preview := Preview(a, maxLines)
		if preview != "" {
			b.WriteString("│" + strings.Repeat("─", innerW) + "│\n")
			for _, line := range strings.Split(preview, "\n") {
				display := "  " + line
				if len(display) > innerW {
					display = display[:innerW-1] + "…"
				}
				b.WriteString("│" + pad(display, innerW) + "│\n")
			}
		}
	}

	// Bottom border
	b.WriteString("└" + strings.Repeat("─", innerW) + "┘")

	return b.String()
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s[:width]
	}
	return s + strings.Repeat(" ", width-len(s))
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
