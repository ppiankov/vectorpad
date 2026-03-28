package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/ppiankov/vectorpad/internal/config"
	"github.com/ppiankov/vectorpad/internal/sidecar"
	"github.com/ppiankov/vectorpad/internal/stash"
	"github.com/ppiankov/vectorpad/internal/vectorcourt"
)

// launchTarget represents a destination for the vector payload.
type launchTarget struct {
	key       string // display key ("1", "2", etc.)
	name      string // display name
	available bool   // detected at startup
	action    func(payload string) (string, error)
}

// launchOverlay is the target picker shown on ctrl+l.
type launchOverlay struct {
	visible bool
	targets []launchTarget
	cursor  int
}

func newLaunchOverlay() launchOverlay {
	targets := []launchTarget{
		{
			key:       "1",
			name:      "Clipboard",
			available: true,
			action: func(payload string) (string, error) {
				return "copied to clipboard", copyToClipboard(payload)
			},
		},
		{
			key:       "2",
			name:      "Claude for Mac",
			available: macAppExists("Claude"),
			action: func(payload string) (string, error) {
				if err := copyToClipboard(payload); err != nil {
					return "", err
				}
				return "copied + focused Claude", openMacApp("Claude")
			},
		},
		{
			key:       "3",
			name:      "ChatGPT for Mac",
			available: macAppExists("ChatGPT"),
			action: func(payload string) (string, error) {
				if err := copyToClipboard(payload); err != nil {
					return "", err
				}
				return "copied + focused ChatGPT", openMacApp("ChatGPT")
			},
		},
		{
			key:       "4",
			name:      "Claude Code CLI",
			available: cliExists("claude"),
			action: func(payload string) (string, error) {
				if err := copyToClipboard(payload); err != nil {
					return "", err
				}
				return "copied — paste into active session", nil
			},
		},
		{
			key:       "5",
			name:      "File (prompt.md)",
			available: true,
			action: func(payload string) (string, error) {
				path := filepath.Join(".", "prompt.md")
				if err := os.WriteFile(path, []byte(payload), 0o644); err != nil {
					return "", fmt.Errorf("write failed: %w", err)
				}
				abs, _ := filepath.Abs(path)
				return fmt.Sprintf("saved to %s", abs), nil
			},
		},
	}

	// Target 6: VectorCourt — only available when API key is configured.
	// Action is nil: async deliberation is handled by startDeliberation in app.go.
	targets = append(targets, launchTarget{
		key:       "6",
		name:      "VectorCourt",
		available: vcKeyConfigured(),
		action:    nil,
	})

	// Target 7: Sidecar — direct inject into active Claude Code session.
	targets = append(targets, launchTarget{
		key:       "7",
		name:      "Sidecar (inject)",
		available: sidecarAvailable(),
		action:    nil, // handled in executeLaunch
	})

	// Target 8: Sidecar + VectorCourt — deliberate on framing, then inject.
	targets = append(targets, launchTarget{
		key:       "8",
		name:      "Sidecar + deliberate",
		available: sidecarAvailable() && vcKeyConfigured(),
		action:    nil, // handled in executeLaunch
	})

	return launchOverlay{targets: targets}
}

// sidecarAvailable returns true if an active Claude Code session exists for the current project.
func sidecarAvailable() bool {
	cwd, err := os.Getwd()
	if err != nil {
		return false
	}
	sessions, err := sidecar.DiscoverSessions(cwd)
	if err != nil {
		return false
	}
	return len(sessions) > 0
}

// vcKeyConfigured returns true if a VectorCourt API key is set in config.
func vcKeyConfigured() bool {
	cfg, err := config.Load()
	if err != nil {
		return false
	}
	return cfg.VectorCourt.APIKey != ""
}

// stashVerdict saves the verdict JSON to the stash with a verdict: prefix.
func stashVerdict(raw json.RawMessage, question string) {
	store, err := stash.NewDefaultStore()
	if err != nil {
		return
	}
	// Pretty-print for readability in stash.
	var pretty json.RawMessage
	formatted := string(raw)
	if json.Unmarshal(raw, &pretty) == nil {
		if f, err := json.MarshalIndent(pretty, "", "  "); err == nil {
			formatted = string(f)
		}
	}
	title := "verdict"
	if question != "" {
		title = question
		if len(title) > 60 {
			title = title[:57] + "..."
		}
	}
	text := fmt.Sprintf("verdict: %s\n\n%s", title, formatted)
	_, _ = store.AddWithMeta(text, stash.SourceVerdict, title, stash.ItemTypeVerdict, "", nil)
}

// formatVerdictSummary extracts a brief status line from the verdict JSON.
func formatVerdictSummary(raw json.RawMessage, gate *vectorcourt.GateResult) string {
	var envelope struct {
		Verdict    string                          `json:"verdict"`
		Status     string                          `json:"status"`
		CaseID     string                          `json:"case_id"`
		Escalation *vectorcourt.EscalationDecision `json:"escalation,omitempty"`
	}
	if json.Unmarshal(raw, &envelope) == nil {
		parts := []string{"vectorcourt"}
		if envelope.Status != "" {
			parts = append(parts, envelope.Status)
		}
		if envelope.Verdict != "" {
			v := envelope.Verdict
			if len(v) > 60 {
				v = v[:57] + "..."
			}
			parts = append(parts, v)
		}
		if gate != nil && gate.Tier != "" {
			parts = append(parts, fmt.Sprintf("(tier: %s)", gate.Tier))
		}
		if envelope.Escalation != nil && envelope.Escalation.Mode == "human_clarification" {
			parts = append(parts, fmt.Sprintf("⚠ %d questions — run: vectorpad clarify %s", len(envelope.Escalation.Questions), envelope.CaseID))
		}
		return strings.Join(parts, " — ")
	}
	return fmt.Sprintf("vectorcourt — verdict received (%d bytes)", len(raw))
}

// ParseEscalation extracts escalation data from the raw verdict JSON.
func ParseEscalation(raw json.RawMessage) (*vectorcourt.EscalationDecision, string) {
	var envelope struct {
		CaseID     string                          `json:"case_id"`
		Escalation *vectorcourt.EscalationDecision `json:"escalation,omitempty"`
	}
	if json.Unmarshal(raw, &envelope) == nil && envelope.Escalation != nil {
		return envelope.Escalation, envelope.CaseID
	}
	return nil, ""
}

func (o *launchOverlay) show() {
	o.visible = true
	o.cursor = 0
}

func (o *launchOverlay) dismiss() {
	o.visible = false
}

func (o *launchOverlay) moveUp() {
	if o.cursor > 0 {
		o.cursor--
	}
}

func (o *launchOverlay) moveDown() {
	if o.cursor < len(o.targets)-1 {
		o.cursor++
	}
}

func (o *launchOverlay) selected() *launchTarget {
	if o.cursor >= 0 && o.cursor < len(o.targets) {
		return &o.targets[o.cursor]
	}
	return nil
}

// selectByKey returns the target matching a number key, or nil.
func (o *launchOverlay) selectByKey(k string) *launchTarget {
	for i := range o.targets {
		if o.targets[i].key == k {
			return &o.targets[i]
		}
	}
	return nil
}

func (o launchOverlay) View() string {
	if !o.visible {
		return ""
	}

	var b strings.Builder
	b.WriteString(helpTitleStyle.Render("Launch To..."))
	b.WriteString("\n\n")

	for i, t := range o.targets {
		prefix := "  "
		if i == o.cursor {
			prefix = "> "
		}

		line := fmt.Sprintf("%s[%s] %s", prefix, t.key, t.name)
		if !t.available {
			b.WriteString(styleDim.Render(line + " (not found)"))
		} else if i == o.cursor {
			b.WriteString(styleHeader.Render(line))
		} else {
			b.WriteString(styleMuted.Render(line))
		}
		b.WriteString("\n")
	}

	b.WriteString("\n")
	b.WriteString(styleDim.Render(" Enter to select  Esc to cancel"))

	return helpBorderStyle.Render(b.String())
}

// macAppExists checks if a macOS application bundle exists.
func macAppExists(name string) bool {
	if runtime.GOOS != "darwin" {
		return false
	}
	// Check common app locations.
	paths := []string{
		filepath.Join("/Applications", name+".app"),
		filepath.Join(os.Getenv("HOME"), "Applications", name+".app"),
	}
	for _, p := range paths {
		if _, err := os.Stat(p); err == nil {
			return true
		}
	}
	return false
}

// openMacApp focuses a macOS application via open -a.
func openMacApp(name string) error {
	if runtime.GOOS != "darwin" {
		return fmt.Errorf("%s: not on macOS", name)
	}
	return exec.Command("open", "-a", name).Run()
}

// cliExists checks if a CLI tool is in PATH.
func cliExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}
