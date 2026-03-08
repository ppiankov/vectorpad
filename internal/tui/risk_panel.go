package tui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/ppiankov/vectorpad/internal/ambiguity"
	"github.com/ppiankov/vectorpad/internal/detect"
	"github.com/ppiankov/vectorpad/internal/drift"
	"github.com/ppiankov/vectorpad/internal/negativespace"
)

type riskPanel struct {
	result             ambiguity.Result
	nudges             []ambiguity.Nudge
	negSpace           negativespace.Result
	driftResult        drift.Result
	removedConstraints []string
	width              int
	height             int
}

func newRiskPanel() riskPanel {
	return riskPanel{}
}

func (p *riskPanel) analyzeText(text string) {
	p.result = ambiguity.Analyze(text, ambiguity.Scope{})
	p.nudges = ambiguity.SelectNudges(p.result)
	p.negSpace = negativespace.Analyze(text)
}

func (p riskPanel) View(_ bool) string {
	return p.render(detect.Capabilities{}, detect.ModeInspect, detect.ScanResult{Clean: true})
}

func (p riskPanel) ViewWithCaps(caps detect.Capabilities, mode detect.PastewatchMode, scan detect.ScanResult) string {
	return p.render(caps, mode, scan)
}

func (p riskPanel) render(caps detect.Capabilities, mode detect.PastewatchMode, scan detect.ScanResult) string {
	var b strings.Builder

	b.WriteString(stylePanelTitle.Render("RISK PANEL"))
	b.WriteString("\n")

	r := p.result
	if r.InstructionWords == 0 {
		b.WriteString(styleMuted.Render(" awaiting input"))
		b.WriteString("\n\n")
		p.renderPastewatchStatus(&b, caps, mode, scan)
		return b.String()
	}

	// Blast radius
	b.WriteString(styleMuted.Render(" blast radius"))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  repos: %d  files: %d\n", r.BlastRadius.Repos, r.BlastRadius.Files))

	// Brevity ratio
	b.WriteString(styleMuted.Render(" brevity ratio"))
	b.WriteString("\n")
	ratioStr := fmt.Sprintf("  %.1fx (%d targets / %d words)", r.BrevityToScopeRatio, r.BlastRadius.Targets, r.InstructionWords)
	b.WriteString(ratioStr)
	b.WriteString("\n")

	// Vague verbs
	if len(r.VagueVerbs) > 0 {
		b.WriteString(styleWarning.Render(fmt.Sprintf(" vague: %s", strings.Join(r.VagueVerbs, ", "))))
		b.WriteString("\n")
	}

	// Warning
	if r.Warning.Active {
		sev := string(r.Warning.Severity)
		color := severityColor(sev)
		warnStyle := lipgloss.NewStyle().Foreground(color).Bold(true)
		b.WriteString(warnStyle.Render(fmt.Sprintf(" ⚠ %s", r.Warning.Message)))
		b.WriteString("\n")
	} else {
		b.WriteString(styleSuccess.Render(" ✓ no warning"))
		b.WriteString("\n")
	}

	// Meaning drift from baseline
	if !p.driftResult.Allowed && len(p.driftResult.Drifts) > 0 {
		b.WriteString("\n")
		b.WriteString(stylePanelTitle.Render("DRIFT"))
		b.WriteString("\n")
		for _, d := range p.driftResult.Drifts {
			for _, c := range d.Changed {
				label := describeDriftChange(d.Axis, c)
				b.WriteString(styleWarning.Render("  " + label))
				b.WriteString("\n")
			}
			for _, a := range d.Added {
				b.WriteString(styleMuted.Render(fmt.Sprintf("  +%s: %s", d.Axis, a)))
				b.WriteString("\n")
			}
			for _, r := range d.Removed {
				b.WriteString(styleWarning.Render(fmt.Sprintf("  -%s: %s", d.Axis, r)))
				b.WriteString("\n")
			}
		}
	}

	// Constraint pinning — warn on removed constraints
	if len(p.removedConstraints) > 0 {
		b.WriteString("\n")
		b.WriteString(styleError.Render(" CONSTRAINT REMOVED"))
		b.WriteString("\n")
		for _, c := range p.removedConstraints {
			text := c
			if len(text) > 50 {
				text = text[:47] + "..."
			}
			b.WriteString(styleWarning.Render(fmt.Sprintf("  -%s", text)))
			b.WriteString("\n")
		}
	}

	// Negative space — missing constraint classes
	if !p.negSpace.Clean() {
		b.WriteString("\n")
		b.WriteString(stylePanelTitle.Render("GAPS"))
		b.WriteString("\n")
		for _, gap := range p.negSpace.Gaps {
			b.WriteString(styleWarning.Render(fmt.Sprintf("  [%s]", gap.Class)))
			b.WriteString("\n")
			b.WriteString(styleMuted.Render(fmt.Sprintf("  %s", gap.Description)))
			b.WriteString("\n")
		}
	}

	// Clipboard scan result
	b.WriteString("\n")
	p.renderPastewatchStatus(&b, caps, mode, scan)

	// Nudges
	if len(p.nudges) > 0 {
		b.WriteString("\n")
		b.WriteString(stylePanelTitle.Render("NUDGES"))
		b.WriteString("\n")
		for _, nudge := range p.nudges {
			b.WriteString(styleWarning.Render(fmt.Sprintf("  [%s]", nudge.Type)))
			b.WriteString("\n")
			b.WriteString(styleMuted.Render(fmt.Sprintf("  %s", nudge.Prompt)))
			b.WriteString("\n")
		}
	}

	return b.String()
}

func (p riskPanel) renderPastewatchStatus(b *strings.Builder, caps detect.Capabilities, mode detect.PastewatchMode, scan detect.ScanResult) {
	label := detect.StatusLabel(caps, mode)
	b.WriteString(styleMuted.Render(" pastewatch: "))

	if !caps.Pastewatch {
		b.WriteString(styleMuted.Render(label))
		b.WriteString("\n")
		return
	}

	b.WriteString(styleMuted.Render(label))
	b.WriteString("\n")

	// Show scan result if there was a scan.
	if scan.Clean {
		b.WriteString(styleSuccess.Render("  clipboard: ✓ clean"))
		b.WriteString("\n")
	} else {
		b.WriteString(styleError.Render("  clipboard: ⚠ secrets detected"))
		b.WriteString("\n")
		for _, finding := range scan.Findings {
			b.WriteString(styleError.Render(fmt.Sprintf("    - %s", finding)))
			b.WriteString("\n")
		}
	}
}

func describeDriftChange(axis drift.Axis, c drift.TokenChange) string {
	switch c.Kind {
	case "upgrade":
		return fmt.Sprintf("%s: strengthened '%s' → '%s'", axis, c.From, c.To)
	case "downgrade":
		return fmt.Sprintf("%s: weakened '%s' → '%s'", axis, c.From, c.To)
	case "polarity_flip":
		return fmt.Sprintf("%s: flipped %s → %s", axis, c.From, c.To)
	default:
		return fmt.Sprintf("%s: changed '%s' → '%s'", axis, c.From, c.To)
	}
}
