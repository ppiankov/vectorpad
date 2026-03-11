package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"

	"github.com/ppiankov/vectorpad/internal/stash"
)

type stashPanel struct {
	stacks       []stash.Stack
	cursor       int
	scrollOffset int
	width        int
	height       int
}

func newStashPanel() stashPanel {
	return stashPanel{}
}

func (p *stashPanel) loadStacks(stacks []stash.Stack) {
	p.stacks = stacks
	if p.cursor >= len(stacks) {
		p.cursor = max(0, len(stacks)-1)
	}
}

func (p *stashPanel) visibleRows() int {
	h := p.height - 3 // title + top/bottom border padding
	if h < 1 {
		return 1
	}
	return h
}

func (p *stashPanel) moveUp() {
	if p.cursor > 0 {
		p.cursor--
	}
	if p.cursor < p.scrollOffset {
		p.scrollOffset = p.cursor
	}
}

func (p *stashPanel) moveDown() {
	if p.cursor < len(p.stacks)-1 {
		p.cursor++
	}
	visible := p.visibleRows()
	if p.cursor >= p.scrollOffset+visible {
		p.scrollOffset = p.cursor - visible + 1
	}
}

func (p *stashPanel) selectedStack() *stash.Stack {
	if len(p.stacks) == 0 || p.cursor >= len(p.stacks) {
		return nil
	}
	return &p.stacks[p.cursor]
}

func (p stashPanel) View(focused bool) string {
	var b strings.Builder

	b.WriteString(stylePanelTitle.Render("STASH"))
	b.WriteString("\n")

	if len(p.stacks) == 0 {
		b.WriteString(styleMuted.Render(" no stashed ideas"))
		return b.String()
	}

	visible := p.visibleRows()
	end := p.scrollOffset + visible
	if end > len(p.stacks) {
		end = len(p.stacks)
	}

	for i := p.scrollOffset; i < end; i++ {
		stack := p.stacks[i]
		line := formatStackLine(stack, p.width-4)

		if i == p.cursor && focused {
			b.WriteString(styleSelected.Render(line))
		} else {
			ageStyle := ageToStyle(stackAge(stack))
			b.WriteString(ageStyle.Render(line))
		}
		b.WriteString("\n")
	}

	// Scroll indicators
	if p.scrollOffset > 0 {
		b.WriteString(styleMuted.Render(" ↑ more"))
		b.WriteString("\n")
	}
	if end < len(p.stacks) {
		b.WriteString(styleMuted.Render(" ↓ more"))
		b.WriteString("\n")
	}

	return b.String()
}

func formatStackLine(stack stash.Stack, maxWidth int) string {
	label := stack.Label
	count := len(stack.Items)
	symbols := uniquenessSymbols(stack)
	line := fmt.Sprintf(" %s %s %d", symbols, label, count)
	if len(line) > maxWidth && maxWidth > 0 {
		line = line[:maxWidth]
	}
	return line
}

// uniquenessSymbols returns a visual indicator for the stack's dominant uniqueness.
// ◆ = verdict, ● = high (novel), ○ = medium (overlaps), ◌ = low (near-duplicate)
func uniquenessSymbols(stack stash.Stack) string {
	if len(stack.Items) == 0 {
		return "◌"
	}

	// Verdict stacks get a distinct diamond symbol.
	for _, item := range stack.Items {
		if item.Source == stash.SourceVerdict {
			return "◆"
		}
	}

	high, med, low := 0, 0, 0
	for _, item := range stack.Items {
		switch item.Uniqueness {
		case stash.UniquenessHigh:
			high++
		case stash.UniquenessMedium:
			med++
		default:
			low++
		}
	}

	// Show the dominant tier symbol.
	if high >= med && high >= low {
		return "●"
	}
	if med >= low {
		return "○"
	}
	return "◌"
}

func stackAge(stack stash.Stack) stash.AgeTier {
	if len(stack.Items) == 0 {
		return stash.AgeTierStale
	}
	latest := stack.Updated
	for _, item := range stack.Items {
		if item.Created.After(latest) {
			latest = item.Created
		}
	}
	return ageTierFromDuration(time.Since(latest))
}

func ageTierFromDuration(d time.Duration) stash.AgeTier {
	switch {
	case d < 24*time.Hour:
		return stash.AgeTierFresh
	case d < 7*24*time.Hour:
		return stash.AgeTierRecent
	case d < 30*24*time.Hour:
		return stash.AgeTierAging
	default:
		return stash.AgeTierStale
	}
}

func ageToStyle(tier stash.AgeTier) lipgloss.Style {
	switch tier {
	case stash.AgeTierFresh:
		return styleStashFresh
	case stash.AgeTierRecent:
		return styleStashRecent
	case stash.AgeTierAging:
		return styleStashAging
	default:
		return styleStashStale
	}
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
