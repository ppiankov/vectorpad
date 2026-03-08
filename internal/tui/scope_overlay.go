package tui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

type scopeOverlay struct {
	visible  bool
	textarea textarea.Model
}

func newScopeOverlay() scopeOverlay {
	ta := textarea.New()
	ta.Placeholder = "scope: 18 repos\noperation: cleanup\ntargets: README.md"
	ta.CharLimit = 0
	ta.SetWidth(50)
	ta.SetHeight(4)
	return scopeOverlay{textarea: ta}
}

func (o *scopeOverlay) show() {
	o.visible = true
	o.textarea.Focus()
}

func (o *scopeOverlay) dismiss() {
	o.visible = false
	o.textarea.Blur()
}

func (o *scopeOverlay) value() string {
	return o.textarea.Value()
}

func (o *scopeOverlay) update(msg tea.Msg) tea.Cmd {
	var cmd tea.Cmd
	o.textarea, cmd = o.textarea.Update(msg)
	return cmd
}

func (o scopeOverlay) view(width, height int) string {
	if !o.visible {
		return ""
	}

	var b strings.Builder
	b.WriteString(stylePanelTitle.Render("DECLARE SCOPE"))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render(" Format: key: value (one per line)"))
	b.WriteString("\n")
	b.WriteString(styleMuted.Render(" Keys: scope, operation, targets, files"))
	b.WriteString("\n\n")
	b.WriteString(o.textarea.View())
	b.WriteString("\n\n")
	b.WriteString(styleDim.Render(" Enter: apply  Esc: cancel"))

	return renderCenteredOverlay(b.String(), width, height)
}
