package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

type helpEntry struct {
	key  string
	desc string
}

type helpModel struct {
	visible bool
	title   string
	entries []helpEntry
	width   int
	height  int
}

func (h *helpModel) toggle(title string, entries []helpEntry) {
	if h.visible {
		h.visible = false
		return
	}
	h.visible = true
	h.title = title
	h.entries = entries
}

func (h *helpModel) dismiss() {
	h.visible = false
}

var (
	helpBorderStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorAccent).
			Padding(1, 2)
	helpTitleStyle = lipgloss.NewStyle().
			Foreground(colorAccent).
			Bold(true)
	helpKeyStyle = lipgloss.NewStyle().
			Foreground(colorWhite).
			Bold(true).
			Width(14)
	helpDescStyle = lipgloss.NewStyle().
			Foreground(colorMuted)
)

func (h helpModel) View() string {
	if !h.visible || len(h.entries) == 0 {
		return ""
	}

	var b strings.Builder
	fmt.Fprintf(&b, "%s\n\n", helpTitleStyle.Render(h.title))

	for _, e := range h.entries {
		fmt.Fprintf(&b, "%s %s\n", helpKeyStyle.Render(e.key), helpDescStyle.Render(e.desc))
	}
	b.WriteString("\nPress Ctrl+H or Esc to close")

	content := helpBorderStyle.Render(b.String())

	contentWidth := lipgloss.Width(content)
	contentHeight := lipgloss.Height(content)

	padLeft := (h.width - contentWidth) / 2
	if padLeft < 0 {
		padLeft = 0
	}
	padTop := (h.height - contentHeight) / 2
	if padTop < 0 {
		padTop = 0
	}

	var out strings.Builder
	for range padTop {
		out.WriteString("\n")
	}
	for _, line := range strings.Split(content, "\n") {
		fmt.Fprintf(&out, "%s%s\n", strings.Repeat(" ", padLeft), line)
	}
	return out.String()
}

func appHelp() []helpEntry {
	return []helpEntry{
		{"Tab", "Next panel"},
		{"Shift+Tab", "Previous panel"},
		{"Ctrl+Y", "Copy vector to clipboard"},
		{"Ctrl+L", "Launch (copy + mark sent)"},
		{"Ctrl+S", "Stash current vector"},
		{"Ctrl+R", "Recall stash into editor"},
		{"Ctrl+K", "Yank from stash"},
		{"Ctrl+P", "Put yanked into editor"},
		{"Ctrl+X", "Prune stash entry"},
		{"↑/k ↓/j", "Navigate stash"},
		{"Ctrl+C", "Quit"},
		{"Ctrl+H", "Toggle this help"},
	}
}
