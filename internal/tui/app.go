package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ppiankov/vectorpad/internal/config"
	"github.com/ppiankov/vectorpad/internal/detect"
	"github.com/ppiankov/vectorpad/internal/flight"
	"github.com/ppiankov/vectorpad/internal/negativespace"
	"github.com/ppiankov/vectorpad/internal/oracul"
	"github.com/ppiankov/vectorpad/internal/pressure"
	"github.com/ppiankov/vectorpad/internal/stash"
)

// Panel focus targets.
type panel int

const (
	panelStash  panel = iota
	panelEditor panel = iota
	panelRisk   panel = iota
	panelCount  panel = 3
)

// Width breakpoints for responsive layout.
const (
	breakpointStash = 80  // stash hidden below this
	breakpointRisk  = 120 // risk collapses to status bar below this
)

// preflightStatus represents the current state of the live preflight check.
type preflightStatus int

const (
	preflightIdle     preflightStatus = iota // no key or no text
	preflightPending                         // debounce timer running
	preflightChecking                        // API call in progress
	preflightReady                           // passed clean
	preflightWarn                            // passed with warnings
	preflightBlocked                         // rejected
)

// preflightState tracks the debounced live preflight indicator.
type preflightState struct {
	status   preflightStatus
	warnings []string
	reason   string
	textHash string // hash of text that was last checked
	seq      int    // debounce sequence number to discard stale ticks
}

// preflightTickMsg fires after the debounce period.
type preflightTickMsg struct{ seq int }

// preflightResultMsg carries the async preflight result back to Update.
type preflightResultMsg struct {
	gate     *oracul.GateResult
	err      error
	textHash string
}

// AppModel is the top-level Bubbletea model for the three-panel TUI.
type AppModel struct {
	focus     panel
	stash     stashPanel
	editor    editorPanel
	risk      riskPanel
	help      helpModel
	launch    launchOverlay
	scope     scopeOverlay
	store     *stash.Store
	recorder  *flight.Recorder
	caps      detect.Capabilities
	pwMode    detect.PastewatchMode
	lastScan  detect.ScanResult
	preflight preflightState
	width     int
	height    int
}

// NewApp creates the application model. store may be nil if stash is unavailable.
func NewApp(store *stash.Store, caps detect.Capabilities) AppModel {
	rec, _ := flight.NewRecorder() // best-effort; nil recorder is fine
	m := AppModel{
		focus:    panelEditor,
		stash:    newStashPanel(),
		editor:   newEditorPanel(),
		risk:     newRiskPanel(),
		launch:   newLaunchOverlay(),
		scope:    newScopeOverlay(),
		store:    store,
		recorder: rec,
		caps:     caps,
		pwMode:   detect.ModeInspect,
		lastScan: detect.ScanResult{Clean: true},
	}
	m.editor.focus()
	m.loadStash()
	// Load contextspectre feedback on startup (nil if unavailable).
	m.risk.feedback = detect.ReadFeedback(caps)
	m.risk.decisionEcon = detect.ReadDecisionEconomics(caps)
	// Load Oracul account status on startup (nil if no key or fetch fails).
	m.refreshAccountStatus()
	return m
}

// syncRisk updates risk panel analysis including drift from editor baseline.
func (m *AppModel) syncRisk() {
	m.risk.analyzeText(m.editor.value())
	m.risk.driftResult = m.editor.driftResult
	m.risk.removedConstraints = m.editor.removedConstraints
	m.risk.scopeResult = m.editor.scopeResult
	// Recompute pressure with vague verbs from ambiguity analysis.
	m.risk.pressureScores = pressure.Score(m.editor.sentences, m.risk.result.VagueVerbs)
	m.risk.decomposeResult = m.editor.decomposeResult
}

// refreshFeedback reads contextspectre telemetry and updates the risk panel.
func (m *AppModel) refreshFeedback() {
	m.risk.feedback = detect.ReadFeedback(m.caps)
	m.risk.decisionEcon = detect.ReadDecisionEconomics(m.caps)
}

// refreshAccountStatus loads Oracul account status into the risk panel.
// Returns nil (hidden section) if no API key is configured or fetch fails.
func (m *AppModel) refreshAccountStatus() {
	cfg, err := config.Load()
	if err != nil || cfg.Oracul.APIKey == "" {
		m.risk.accountStatus = nil
		return
	}
	client := oracul.NewClient(cfg.Endpoint(), cfg.Oracul.APIKey)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	status, err := client.Account(ctx)
	if err != nil {
		m.risk.accountStatus = nil
		return
	}
	m.risk.accountStatus = status
}

// maybeSchedulePreflight resets the debounce timer if text changed and Oracul key is configured.
// Wraps the editor cmd so both fire.
func (m *AppModel) maybeSchedulePreflight(editorCmd tea.Cmd) tea.Cmd {
	text := m.editor.value()
	hash := textFingerprint(text)
	if hash == m.preflight.textHash || text == "" {
		return editorCmd
	}
	// Text changed — reset debounce.
	m.preflight.textHash = hash
	m.preflight.seq++
	m.preflight.status = preflightPending
	m.risk.preflightReadiness = nil
	seq := m.preflight.seq
	tickCmd := tea.Tick(2*time.Second, func(_ time.Time) tea.Msg {
		return preflightTickMsg{seq: seq}
	})
	if editorCmd != nil {
		return tea.Batch(editorCmd, tickCmd)
	}
	return tickCmd
}

// startPreflightCheck fires an async preflight API call if Oracul key is configured.
func (m *AppModel) startPreflightCheck() tea.Cmd {
	cfg, err := config.Load()
	if err != nil || cfg.Oracul.APIKey == "" {
		m.preflight.status = preflightIdle
		return nil
	}
	m.preflight.status = preflightChecking
	text := m.editor.value()
	hash := m.preflight.textHash
	sentences := m.editor.sentences
	return func() tea.Msg {
		filing := oracul.MapSentences(sentences)
		question := oracul.ExtractQuestion(sentences, text)
		client := oracul.NewClient(cfg.Endpoint(), cfg.Oracul.APIKey)
		gate, err := client.PreflightGate(context.Background(), question, filing)
		return preflightResultMsg{gate: gate, err: err, textHash: hash}
	}
}

// textFingerprint returns a cheap fingerprint of the text for change detection.
func textFingerprint(text string) string {
	if len(text) <= 100 {
		return text
	}
	return fmt.Sprintf("%d:%s...%s", len(text), text[:50], text[len(text)-50:])
}

func (m *AppModel) loadStash() {
	if m.store == nil {
		return
	}
	file, err := m.store.Load()
	if err != nil {
		return
	}
	m.stash.loadStacks(file.Stacks)
}

func (m AppModel) Init() tea.Cmd {
	return nil
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case preflightTickMsg:
		// Debounce expired — start async preflight if seq matches.
		if msg.seq != m.preflight.seq {
			return m, nil
		}
		return m, m.startPreflightCheck()

	case preflightResultMsg:
		// Async preflight result arrived.
		if msg.textHash != m.preflight.textHash {
			return m, nil // text changed since check started, discard
		}
		if msg.err != nil {
			m.preflight.status = preflightIdle
			return m, nil
		}
		m.risk.preflightReadiness = msg.gate
		if !msg.gate.Allowed {
			m.preflight.status = preflightBlocked
			m.preflight.reason = msg.gate.Reason
			m.preflight.warnings = nil
		} else if len(msg.gate.Warnings) > 0 {
			m.preflight.status = preflightWarn
			m.preflight.warnings = msg.gate.Warnings
			m.preflight.reason = ""
		} else {
			m.preflight.status = preflightReady
			m.preflight.warnings = nil
			m.preflight.reason = ""
		}
		return m, nil

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.help.width = msg.Width
		m.help.height = msg.Height
		m.resizePanels()
		return m, nil

	case tea.KeyMsg:
		// Help overlay intercepts all keys when visible.
		if m.help.visible {
			m.help.dismiss()
			return m, nil
		}

		// Launch overlay intercepts keys when visible.
		if m.launch.visible {
			return m.updateLaunchKeys(msg)
		}

		// Scope overlay intercepts keys when visible.
		if m.scope.visible {
			return m.updateScopeKeys(msg)
		}

		// Global keys.
		if key.Matches(msg, keys.Quit) {
			return m, tea.Quit
		}
		if key.Matches(msg, keys.Help) {
			m.help.toggle("VectorPad", appHelp())
			return m, nil
		}
		if key.Matches(msg, keys.Tab) {
			m.cycleFocus(1)
			return m, nil
		}
		if key.Matches(msg, keys.Stab) {
			m.cycleFocus(-1)
			return m, nil
		}

		// Global editor actions (work from any panel).
		if key.Matches(msg, keys.Copy) {
			m.copyWithScan()
			return m, nil
		}
		if key.Matches(msg, keys.Launch) {
			m.launch.show()
			return m, nil
		}
		if key.Matches(msg, keys.Scope) {
			m.scope.show()
			return m, nil
		}
		if key.Matches(msg, keys.Decompose) {
			m.decomposeIntoStash()
			return m, nil
		}
		if key.Matches(msg, keys.Stash) {
			m.stashCurrentVector()
			return m, nil
		}
		if key.Matches(msg, keys.Recall) {
			m.recallFromStash()
			return m, nil
		}

		// Panel-specific keys.
		switch m.focus {
		case panelStash:
			return m.updateStashKeys(msg)
		case panelEditor:
			return m.updateEditorKeys(msg)
		}
	}

	// Delegate to editor for text input when focused.
	if m.focus == panelEditor {
		cmd := m.editor.update(msg)
		m.syncRisk()
		return m, m.maybeSchedulePreflight(cmd)
	}

	return m, nil
}

func (m *AppModel) updateStashKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	if key.Matches(msg, keys.Up) {
		m.stash.moveUp()
		return m, nil
	}
	if key.Matches(msg, keys.Down) {
		m.stash.moveDown()
		return m, nil
	}
	if key.Matches(msg, keys.Prune) {
		m.pruneSelectedStack()
		return m, nil
	}
	if key.Matches(msg, keys.Essence) {
		m.extractEssence()
		return m, nil
	}
	return m, nil
}

func (m *AppModel) updateEditorKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Editor gets all remaining keys as text input.
	cmd := m.editor.update(msg)
	m.syncRisk()
	return m, m.maybeSchedulePreflight(cmd)
}

func (m *AppModel) cycleFocus(direction int) {
	// Determine which panels are visible.
	panels := m.visiblePanels()
	if len(panels) == 0 {
		return
	}

	// Find current position.
	currentIdx := 0
	for i, p := range panels {
		if p == m.focus {
			currentIdx = i
			break
		}
	}

	// Cycle.
	nextIdx := (currentIdx + direction + len(panels)) % len(panels)
	m.setFocus(panels[nextIdx])
}

func (m *AppModel) visiblePanels() []panel {
	var panels []panel
	if m.width >= breakpointStash {
		panels = append(panels, panelStash)
	}
	panels = append(panels, panelEditor)
	if m.width >= breakpointRisk {
		panels = append(panels, panelRisk)
	}
	return panels
}

func (m *AppModel) setFocus(p panel) {
	m.focus = p
	if p == panelEditor {
		m.editor.focus()
	} else {
		m.editor.blur()
	}
}

func (m *AppModel) resizePanels() {
	stashWidth, editorWidth, riskWidth := m.columnWidths()

	m.stash.width = stashWidth
	m.stash.height = m.height

	m.editor.resize(editorWidth, m.height)

	m.risk.width = riskWidth
	m.risk.height = m.height
}

func (m AppModel) columnWidths() (stashW, editorW, riskW int) {
	w := m.width

	if w < breakpointStash {
		// Editor only.
		return 0, w, 0
	}
	if w < breakpointRisk {
		// Stash + editor.
		stashW = w / 4
		editorW = w - stashW
		return stashW, editorW, 0
	}
	// All three panels.
	stashW = w / 5
	riskW = w / 4
	editorW = w - stashW - riskW
	return stashW, editorW, riskW
}

func (m *AppModel) copyWithScan() {
	content := m.editor.buildCopyPayload()
	if content == "" {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = "nothing to copy"
		return
	}

	// Run pastewatch scan on the full payload (text + serialized attachments).
	m.lastScan = detect.ScanPayload(m.caps, m.pwMode, content)
	if !m.lastScan.Clean {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = "blocked: secrets detected (see risk panel)"
		return
	}

	m.editor.copyAll()
}

func (m *AppModel) stashCurrentVector() {
	if m.store == nil {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = "stash unavailable"
		return
	}
	content := m.editor.value()
	if content == "" {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = "nothing to stash"
		return
	}
	item, err := m.store.Add(content, stash.SourcePaste)
	if err != nil {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = fmt.Sprintf("stash failed: %v", err)
		return
	}
	m.editor.copyStatus = copyCopied
	m.editor.copyMsg = fmt.Sprintf("stashed %s", item.ID)
	m.loadStash()
}

func (m *AppModel) recallFromStash() {
	stack := m.stash.selectedStack()
	if stack == nil || len(stack.Items) == 0 {
		return
	}
	// Recall the latest item from selected stack.
	latest := stack.Items[len(stack.Items)-1]
	m.editor.setValue(latest.Text)
	m.syncRisk()
	m.setFocus(panelEditor)
}

func (m *AppModel) pruneSelectedStack() {
	if m.store == nil {
		return
	}
	stack := m.stash.selectedStack()
	if stack == nil {
		return
	}
	// Load current state, remove the selected stack's items, re-cluster, save.
	file, err := m.store.Load()
	if err != nil {
		return
	}
	removeIDs := make(map[string]struct{}, len(stack.Items))
	for _, item := range stack.Items {
		removeIDs[item.ID] = struct{}{}
	}
	var kept []stash.Item
	for _, s := range file.Stacks {
		for _, item := range s.Items {
			if _, remove := removeIDs[item.ID]; !remove {
				kept = append(kept, item)
			}
		}
	}
	file.Stacks = stash.ClusterItems(kept, time.Now().UTC())
	_ = m.store.Save(file)
	m.loadStash()
}

func (m *AppModel) decomposeIntoStash() {
	if !m.editor.decomposeResult.Triggered {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = "nothing to decompose"
		return
	}
	if m.store == nil {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = "stash unavailable"
		return
	}

	count := 0
	for _, sv := range m.editor.decomposeResult.SubVectors {
		if sv.Text == "" {
			continue
		}
		_, err := m.store.Add(sv.Text, stash.SourcePaste)
		if err != nil {
			m.editor.copyStatus = copyError
			m.editor.copyMsg = fmt.Sprintf("decompose stash failed: %v", err)
			return
		}
		count++
	}

	m.editor.copyStatus = copyCopied
	m.editor.copyMsg = fmt.Sprintf("decomposed into %d sub-vectors in stash", count)
	m.loadStash()
}

func (m *AppModel) extractEssence() {
	stack := m.stash.selectedStack()
	if stack == nil || len(stack.Items) == 0 {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = "no stash stack selected"
		return
	}

	essence := stash.ExtractEssence(*stack)
	if essence == "" {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = "empty essence"
		return
	}

	m.editor.setValue(essence)
	m.syncRisk()
	m.setFocus(panelEditor)
	m.editor.copyStatus = copyCopied
	m.editor.copyMsg = fmt.Sprintf("essence from %q (%d items)", stack.Label, len(stack.Items))
}

func (m *AppModel) updateLaunchKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.launch.dismiss()
		return m, nil
	case "up", "k":
		m.launch.moveUp()
		return m, nil
	case "down", "j":
		m.launch.moveDown()
		return m, nil
	case "enter":
		t := m.launch.selected()
		if t != nil {
			m.executeLaunch(t)
		}
		m.launch.dismiss()
		return m, nil
	case "1", "2", "3", "4", "5", "6":
		t := m.launch.selectByKey(msg.String())
		if t != nil {
			m.executeLaunch(t)
		}
		m.launch.dismiss()
		return m, nil
	}
	return m, nil
}

func (m *AppModel) updateScopeKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.scope.dismiss()
		return m, nil
	case "ctrl+d":
		// Apply scope and dismiss.
		m.editor.setScope(m.scope.value())
		m.syncRisk()
		m.scope.dismiss()
		return m, nil
	}
	// Pass through to scope textarea.
	cmd := m.scope.update(msg)
	return m, cmd
}

func (m *AppModel) executeLaunch(t *launchTarget) {
	if !t.available {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = fmt.Sprintf("%s: not available", t.name)
		return
	}

	payload := m.editor.buildCopyPayload()
	if payload == "" {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = "nothing to launch"
		return
	}

	// Run pastewatch scan before launching.
	m.lastScan = detect.ScanPayload(m.caps, m.pwMode, payload)
	if !m.lastScan.Clean {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = "blocked: secrets detected (see risk panel)"
		return
	}

	statusMsg, err := t.action(payload)
	if err != nil {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = fmt.Sprintf("launch failed: %v", err)
		return
	}
	m.editor.copyStatus = copyCopied
	m.editor.copyMsg = fmt.Sprintf("launched: %s", statusMsg)

	// Refresh contextspectre feedback and Oracul account status after launch.
	m.refreshFeedback()
	m.refreshAccountStatus()

	// Record the launch in the flight log.
	if m.recorder != nil {
		ns := negativespace.Analyze(payload)
		var gapClasses []string
		for _, g := range ns.Gaps {
			gapClasses = append(gapClasses, string(g.Class))
		}
		_ = m.recorder.Append(flight.Record{
			Target: t.name,
			Text:   payload,
			Metrics: flight.MetricsSnapshot{
				Tokens:    m.editor.metrics.TokenWeight.Estimated,
				Integrity: m.editor.metrics.VectorIntegrity.Ratio,
				CPD:       m.editor.metrics.CPDProjection,
				TTC:       m.editor.metrics.TTCProjection,
				CDR:       m.editor.metrics.CDRProjection,
			},
			Gaps:       gapClasses,
			VagueVerbs: m.risk.result.VagueVerbs,
		})
	}
}

func (m AppModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "initializing..."
	}

	// Overlays take over the whole screen.
	if m.help.visible {
		return m.help.View()
	}
	if m.launch.visible {
		m.launch.targets = newLaunchOverlay().targets // refresh availability
		return renderCenteredOverlay(m.launch.View(), m.width, m.height)
	}
	if m.scope.visible {
		return m.scope.view(m.width, m.height)
	}

	stashW, editorW, riskW := m.columnWidths()

	var panels []string

	// Stash panel.
	if stashW > 0 {
		content := m.stash.View(m.focus == panelStash)
		border := panelBorderStyle(m.focus == panelStash)
		panels = append(panels, border.Width(stashW-2).Height(m.height-2).Render(content))
	}

	// Editor panel.
	{
		content := m.editor.View(m.focus == panelEditor)
		border := panelBorderStyle(m.focus == panelEditor)
		panels = append(panels, border.Width(editorW-2).Height(m.height-2).Render(content))
	}

	// Risk panel.
	if riskW > 0 {
		content := m.risk.ViewWithCaps(m.caps, m.pwMode, m.lastScan)
		border := panelBorderStyle(m.focus == panelRisk)
		panels = append(panels, border.Width(riskW-2).Height(m.height-2).Render(content))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, panels...)
}

func renderCenteredOverlay(content string, width, height int) string {
	contentWidth := lipgloss.Width(content)
	contentHeight := lipgloss.Height(content)

	padLeft := (width - contentWidth) / 2
	if padLeft < 0 {
		padLeft = 0
	}
	padTop := (height - contentHeight) / 2
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

func panelBorderStyle(focused bool) lipgloss.Style {
	if focused {
		return styleFocusBorder
	}
	return styleInactiveBorder
}
