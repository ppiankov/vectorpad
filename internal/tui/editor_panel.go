package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/ppiankov/vectorpad/internal/attach"
	"github.com/ppiankov/vectorpad/internal/classifier"
	"github.com/ppiankov/vectorpad/internal/preflight"
	"github.com/ppiankov/vectorpad/internal/vector"
)

type copyStatus int

const (
	copyIdle copyStatus = iota
	copyCopied
	copyError
)

type editorPanel struct {
	textarea    textarea.Model
	sentences   []classifier.Sentence
	vectorBlock string
	metrics     preflight.Metrics
	attachments []*attach.Attachment
	attachCfgs  []attach.ExcerptConfig
	copyStatus  copyStatus
	copyMsg     string
	width       int
	height      int
}

func newEditorPanel() editorPanel {
	ta := textarea.New()
	ta.Placeholder = "Paste your directive here..."
	ta.CharLimit = 0
	ta.SetWidth(60)
	ta.SetHeight(10)
	return editorPanel{textarea: ta}
}

func (p *editorPanel) focus() {
	p.textarea.Focus()
}

func (p *editorPanel) blur() {
	p.textarea.Blur()
}

func (p *editorPanel) resize(width, height int) {
	p.width = width
	p.height = height
	editorHeight := height - 12 // room for classified view + dashboard
	if editorHeight < 3 {
		editorHeight = 3
	}
	p.textarea.SetWidth(width - 2)
	p.textarea.SetHeight(editorHeight)
}

func (p *editorPanel) update(msg tea.Msg) tea.Cmd {
	prevValue := p.textarea.Value()
	var cmd tea.Cmd
	p.textarea, cmd = p.textarea.Update(msg)

	// Detect paste: if new content was added, check if it's a file path.
	newValue := p.textarea.Value()
	if newValue != prevValue {
		added := extractPastedText(prevValue, newValue)
		if a := attach.DetectPath(added); a != nil {
			p.addAttachment(a)
			// Remove the path from textarea — it's now an object.
			p.textarea.SetValue(prevValue)
		}
	}

	p.reclassify()
	return cmd
}

func (p *editorPanel) addAttachment(a *attach.Attachment) {
	p.attachments = append(p.attachments, a)
	p.attachCfgs = append(p.attachCfgs, attach.DefaultExcerptConfig(a))
	p.copyStatus = copyCopied
	p.copyMsg = fmt.Sprintf("attached: %s %s", a.Label, a.Name)
}

func (p *editorPanel) removeAttachment(idx int) {
	if idx < 0 || idx >= len(p.attachments) {
		return
	}
	p.attachments = append(p.attachments[:idx], p.attachments[idx+1:]...)
	p.attachCfgs = append(p.attachCfgs[:idx], p.attachCfgs[idx+1:]...)
}

// extractPastedText returns the text that was added between old and new values.
func extractPastedText(old, new string) string {
	if strings.HasPrefix(new, old) {
		return strings.TrimSpace(new[len(old):])
	}
	return ""
}

func (p *editorPanel) reclassify() {
	content := p.textarea.Value()
	if content == "" {
		p.sentences = nil
		p.vectorBlock = ""
		p.metrics = preflight.Metrics{}
		return
	}
	p.sentences = classifier.Classify(content)
	p.vectorBlock = vector.Render(p.sentences)
	p.metrics = preflight.Compute(content, p.sentences)
}

func (p *editorPanel) value() string {
	return p.textarea.Value()
}

func (p *editorPanel) setValue(text string) {
	p.textarea.SetValue(text)
	p.reclassify()
}

func (p *editorPanel) copyAll() {
	content := p.buildCopyPayload()
	if content == "" {
		p.copyStatus = copyError
		p.copyMsg = "nothing to copy"
		return
	}
	if err := copyToClipboard(content); err != nil {
		p.copyStatus = copyError
		p.copyMsg = fmt.Sprintf("copy failed: %v", err)
	} else {
		p.copyStatus = copyCopied
		lines := strings.Count(content, "\n") + 1
		p.copyMsg = fmt.Sprintf("copied %d lines (%d bytes)", lines, len(content))
	}
}

// buildCopyPayload assembles the vector text plus serialized attachments.
func (p *editorPanel) buildCopyPayload() string {
	text := p.textarea.Value()
	if len(p.attachments) == 0 {
		return text
	}

	var b strings.Builder
	if text != "" {
		b.WriteString(text)
		b.WriteString("\n\n")
	}
	for i, a := range p.attachments {
		cfg := p.attachCfgs[i]
		serialized := attach.Serialize(a, cfg.Mode, cfg.Lines)
		b.WriteString(serialized)
		if i < len(p.attachments)-1 {
			b.WriteString("\n\n")
		}
	}
	return b.String()
}

func (p editorPanel) View(focused bool) string {
	var b strings.Builder

	b.WriteString(stylePanelTitle.Render("VECTOR EDITOR"))
	b.WriteString("\n")
	b.WriteString(p.textarea.View())
	b.WriteString("\n")

	// Classified view (compact)
	if len(p.sentences) > 0 {
		b.WriteString(styleMuted.Render("─── classified ───"))
		b.WriteString("\n")
		maxLines := 6
		lines := strings.Split(p.vectorBlock, "\n")
		shown := 0
		for _, line := range lines {
			if shown >= maxLines {
				b.WriteString(styleMuted.Render("  ..."))
				b.WriteString("\n")
				break
			}
			trimmed := strings.TrimSpace(line)
			if trimmed == "" || trimmed == "VECTOR" {
				continue
			}
			b.WriteString(renderClassifiedLine(trimmed))
			b.WriteString("\n")
			shown++
		}
	}

	// Attachment cards
	if len(p.attachments) > 0 {
		b.WriteString(styleMuted.Render("─── attachments ───"))
		b.WriteString("\n")
		for _, a := range p.attachments {
			card := attach.RenderCard(a, 3)
			b.WriteString(styleMuted.Render(card))
		}
	}

	// Dashboard bar
	b.WriteString(styleMuted.Render("─── dashboard ───"))
	b.WriteString("\n")
	b.WriteString(p.renderDashboard())
	b.WriteString("\n")

	// Copy status
	switch p.copyStatus {
	case copyCopied:
		b.WriteString(styleSuccess.Render(p.copyMsg))
	case copyError:
		b.WriteString(styleError.Render(p.copyMsg))
	}

	return b.String()
}

func (p editorPanel) renderDashboard() string {
	m := p.metrics
	if m.TokenWeight.Estimated == 0 && len(p.sentences) == 0 {
		return styleMuted.Render(" tokens: 0 | paste to begin")
	}

	return styleMuted.Render(fmt.Sprintf(
		" tokens: %d | integrity: %.0f%% | CPD: $%.4f | TTC: %.1f | CDR: %.2f",
		m.TokenWeight.Estimated,
		m.VectorIntegrity.Percentage,
		m.CPDProjection,
		m.TTCProjection,
		m.CDRProjection,
	))
}

func renderClassifiedLine(line string) string {
	if strings.Contains(line, "[LOCKED]") || strings.Contains(line, "[LOCKED:") {
		return styleLocked.Render("  " + line)
	}
	return styleMuted.Render("  " + line)
}
