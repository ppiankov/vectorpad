package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

type status int

const (
	statusEditing status = iota
	statusCopied
	statusError
)

type model struct {
	textarea textarea.Model
	status   status
	message  string
	width    int
	height   int
}

func NewSpike() model {
	ta := textarea.New()
	ta.Placeholder = "Paste your directive here..."
	ta.Focus()
	ta.CharLimit = 0 // unlimited
	ta.SetWidth(80)
	ta.SetHeight(20)

	return model{
		textarea: ta,
		status:   statusEditing,
	}
}

func (m model) Init() tea.Cmd {
	return textarea.Blink
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.textarea.SetWidth(msg.Width - 4)
		m.textarea.SetHeight(msg.Height - 6)
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "esc":
			return m, tea.Quit
		case "ctrl+y":
			// Copy all content to clipboard
			content := m.textarea.Value()
			if content == "" {
				m.status = statusError
				m.message = "nothing to copy"
				return m, nil
			}
			if err := copyToClipboard(content); err != nil {
				m.status = statusError
				m.message = fmt.Sprintf("copy failed: %v", err)
			} else {
				m.status = statusCopied
				lines := strings.Count(content, "\n") + 1
				m.message = fmt.Sprintf("copied %d lines (%d bytes)", lines, len(content))
			}
			return m, nil
		}
	}

	// Reset status on any edit
	if m.status != statusEditing {
		m.status = statusEditing
		m.message = ""
	}

	var cmd tea.Cmd
	m.textarea, cmd = m.textarea.Update(msg)
	return m, cmd
}

var (
	titleStyle  = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12"))
	statusStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("10"))
	errorStyle  = lipgloss.NewStyle().Foreground(lipgloss.Color("9"))
	helpStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
)

func (m model) View() string {
	var b strings.Builder

	b.WriteString(titleStyle.Render("vectorpad spike"))
	b.WriteString("\n\n")
	b.WriteString(m.textarea.View())
	b.WriteString("\n")

	switch m.status {
	case statusCopied:
		b.WriteString(statusStyle.Render(m.message))
	case statusError:
		b.WriteString(errorStyle.Render(m.message))
	default:
		content := m.textarea.Value()
		lines := 0
		bytes := len(content)
		if content != "" {
			lines = strings.Count(content, "\n") + 1
		}
		b.WriteString(helpStyle.Render(fmt.Sprintf("%d lines | %d bytes | ctrl+y copy all | esc quit", lines, bytes)))
	}

	return b.String()
}

// copyToClipboard is in clipboard.go
