package tui

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/ppiankov/vectorpad/internal/config"
	"github.com/ppiankov/vectorpad/internal/detect"
	"github.com/ppiankov/vectorpad/internal/flight"
	"github.com/ppiankov/vectorpad/internal/negativespace"
	"github.com/ppiankov/vectorpad/internal/pressure"
	"github.com/ppiankov/vectorpad/internal/sidecar"
	"github.com/ppiankov/vectorpad/internal/stash"
	"github.com/ppiankov/vectorpad/internal/vectorcourt"
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
	gate     *vectorcourt.GateResult
	err      error
	textHash string
}

// precedentStatus represents the current state of the precedent search.
type precedentStatus int

const (
	precedentIdle precedentStatus = iota
	precedentPending
	precedentLoading
	precedentLoaded
)

// precedentState tracks the debounced precedent search.
type precedentState struct {
	status   precedentStatus
	textHash string
	seq      int
}

// deliberationStatus represents the state of an async VectorCourt submit.
type deliberationStatus int

const (
	deliberationIdle   deliberationStatus = iota
	deliberationActive                    // API call in flight
)

// deliberationState tracks an in-progress VectorCourt deliberation.
type deliberationState struct {
	status    deliberationStatus
	startTime time.Time
	cancel    context.CancelFunc
	flightID  string                       // ID of the flight record to update with VectorCourt data
	sparMsg   string                       // latest spar event message for live display
	queuePos  int                          // queue position (0 = not queued / processing)
	sparCh    <-chan vectorcourt.SparEvent // SSE event channel (nil when not streaming)
}

// deliberationTickMsg fires every second to update the elapsed timer.
type deliberationTickMsg struct{}

// deliberationSubmittedMsg signals that /v1/submit returned successfully.
// The Update handler starts polling and SSE streaming.
type deliberationSubmittedMsg struct {
	submissionID string
	caseID       string
	position     int
	client       *vectorcourt.Client
	question     string
	gate         *vectorcourt.GateResult
	vcSnapshot   *flight.VectorCourtSnapshot
}

// deliberationPollMsg carries an intermediate poll result.
type deliberationPollMsg struct {
	status       *vectorcourt.SubmissionStatus
	err          error
	client       *vectorcourt.Client
	submissionID string
	caseID       string
	question     string
	gate         *vectorcourt.GateResult
	vcSnapshot   *flight.VectorCourtSnapshot
}

// deliberationSparMsg carries a live spar event into the TUI.
type deliberationSparMsg struct {
	event vectorcourt.SparEvent
}

// deliberationResultMsg carries the async VectorCourt verdict back to Update.
type deliberationResultMsg struct {
	statusMsg  string
	err        error
	vcSnapshot *flight.VectorCourtSnapshot
}

// precedentTickMsg fires after the precedent debounce period.
type precedentTickMsg struct{ seq int }

// precedentResultMsg carries the async precedent search result back to Update.
type precedentResultMsg struct {
	search   *vectorcourt.PrecedentSearch
	err      error
	textHash string
}

// instantPrecState tracks the debounced instant precedent lookup (300ms).
type instantPrecState struct {
	status   precedentStatus
	textHash string
	seq      int
}

// instantPrecTickMsg fires after the 300ms instant precedent debounce.
type instantPrecTickMsg struct{ seq int }

// instantPrecResultMsg carries the async instant precedent result.
type instantPrecResultMsg struct {
	result   *vectorcourt.InstantPrecedentResult
	err      error
	textHash string
}

// AppModel is the top-level Bubbletea model for the three-panel TUI.
type AppModel struct {
	focus        panel
	stash        stashPanel
	editor       editorPanel
	risk         riskPanel
	help         helpModel
	launch       launchOverlay
	scope        scopeOverlay
	store        *stash.Store
	recorder     *flight.Recorder
	caps         detect.Capabilities
	pwMode       detect.PastewatchMode
	lastScan     detect.ScanResult
	preflight    preflightState
	precedent    precedentState
	instantPrec  instantPrecState
	deliberation deliberationState
	width        int
	height       int
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
	// Load VectorCourt account status and prediction debt on startup (nil if no key or fetch fails).
	m.refreshAccountStatus()
	m.refreshPredictionDebt()
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

// refreshAccountStatus loads VectorCourt account status into the risk panel.
// Returns nil (hidden section) if no API key is configured or fetch fails.
func (m *AppModel) refreshAccountStatus() {
	cfg, err := config.Load()
	if err != nil || cfg.VectorCourt.APIKey == "" {
		m.risk.accountStatus = nil
		return
	}
	client := vectorcourt.NewClient(cfg.Endpoint(), cfg.VectorCourt.APIKey)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	status, err := client.Account(ctx)
	if err != nil {
		m.risk.accountStatus = nil
		return
	}
	m.risk.accountStatus = status
}

// refreshPredictionDebt loads VectorCourt prediction debt into the risk panel.
func (m *AppModel) refreshPredictionDebt() {
	cfg, err := config.Load()
	if err != nil || cfg.VectorCourt.APIKey == "" {
		m.risk.predictionDebt = nil
		return
	}
	client := vectorcourt.NewClient(cfg.Endpoint(), cfg.VectorCourt.APIKey)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	debt, err := client.GetPredictionDebt(ctx)
	if err != nil {
		m.risk.predictionDebt = nil
		return
	}
	m.risk.predictionDebt = debt
}

// maybeSchedulePreflight resets the debounce timer if text changed and VectorCourt key is configured.
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

// startPreflightCheck fires an async preflight API call if VectorCourt key is configured.
func (m *AppModel) startPreflightCheck() tea.Cmd {
	cfg, err := config.Load()
	if err != nil || cfg.VectorCourt.APIKey == "" {
		m.preflight.status = preflightIdle
		return nil
	}
	m.preflight.status = preflightChecking
	text := m.editor.value()
	hash := m.preflight.textHash
	sentences := m.editor.sentences
	return func() tea.Msg {
		filing := vectorcourt.MapSentences(sentences)
		question := vectorcourt.ExtractQuestion(sentences, text)
		client := vectorcourt.NewClient(cfg.Endpoint(), cfg.VectorCourt.APIKey)
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

// maybeSchedulePrecedent resets the precedent debounce timer if text changed.
func (m *AppModel) maybeSchedulePrecedent(editorCmd tea.Cmd) tea.Cmd {
	text := m.editor.value()
	hash := textFingerprint(text)
	if hash == m.precedent.textHash || text == "" {
		return editorCmd
	}
	m.precedent.textHash = hash
	m.precedent.seq++
	m.precedent.status = precedentPending
	m.risk.precedentSearch = nil
	seq := m.precedent.seq
	tickCmd := tea.Tick(3*time.Second, func(_ time.Time) tea.Msg {
		return precedentTickMsg{seq: seq}
	})
	if editorCmd != nil {
		return tea.Batch(editorCmd, tickCmd)
	}
	return tickCmd
}

// startPrecedentSearch fires an async precedent search if VectorCourt key is configured.
func (m *AppModel) startPrecedentSearch() tea.Cmd {
	cfg, err := config.Load()
	if err != nil || cfg.VectorCourt.APIKey == "" {
		m.precedent.status = precedentIdle
		return nil
	}
	m.precedent.status = precedentLoading
	text := m.editor.value()
	hash := m.precedent.textHash
	sentences := m.editor.sentences
	return func() tea.Msg {
		question := vectorcourt.ExtractQuestion(sentences, text)
		client := vectorcourt.NewClient(cfg.Endpoint(), cfg.VectorCourt.APIKey)
		search, err := client.SearchPrecedents(context.Background(), question, 3)
		return precedentResultMsg{search: search, err: err, textHash: hash}
	}
}

// maybeScheduleInstantPrec resets the 300ms instant precedent debounce on text change.
func (m *AppModel) maybeScheduleInstantPrec(editorCmd tea.Cmd) tea.Cmd {
	text := m.editor.value()
	hash := textFingerprint(text)
	if hash == m.instantPrec.textHash || text == "" {
		return editorCmd
	}
	m.instantPrec.textHash = hash
	m.instantPrec.seq++
	m.instantPrec.status = precedentPending
	m.risk.instantPrecedent = nil
	seq := m.instantPrec.seq
	tickCmd := tea.Tick(300*time.Millisecond, func(_ time.Time) tea.Msg {
		return instantPrecTickMsg{seq: seq}
	})
	if editorCmd != nil {
		return tea.Batch(editorCmd, tickCmd)
	}
	return tickCmd
}

// startInstantPrecSearch fires an async instant precedent lookup.
func (m *AppModel) startInstantPrecSearch() tea.Cmd {
	cfg, err := config.Load()
	if err != nil || cfg.VectorCourt.APIKey == "" {
		m.instantPrec.status = precedentIdle
		return nil
	}
	m.instantPrec.status = precedentLoading
	text := m.editor.value()
	hash := m.instantPrec.textHash
	return func() tea.Msg {
		client := vectorcourt.NewClient(cfg.Endpoint(), cfg.VectorCourt.APIKey)
		result, err := client.InstantPrecedents(context.Background(), text, 5)
		return instantPrecResultMsg{result: result, err: err, textHash: hash}
	}
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

	case precedentTickMsg:
		if msg.seq != m.precedent.seq {
			return m, nil
		}
		return m, m.startPrecedentSearch()

	case precedentResultMsg:
		if msg.textHash != m.precedent.textHash {
			return m, nil
		}
		if msg.err != nil {
			m.precedent.status = precedentIdle
			return m, nil
		}
		m.precedent.status = precedentLoaded
		m.risk.precedentSearch = msg.search
		return m, nil

	case instantPrecTickMsg:
		if msg.seq != m.instantPrec.seq {
			return m, nil
		}
		return m, m.startInstantPrecSearch()

	case instantPrecResultMsg:
		if msg.textHash != m.instantPrec.textHash {
			return m, nil
		}
		if msg.err != nil {
			m.instantPrec.status = precedentIdle
			return m, nil
		}
		m.instantPrec.status = precedentLoaded
		m.risk.instantPrecedent = msg.result
		return m, nil

	case deliberationTickMsg:
		if m.deliberation.status != deliberationActive {
			m.editor.deliberationMsg = ""
			return m, nil
		}
		elapsed := time.Since(m.deliberation.startTime).Truncate(time.Second)
		if m.deliberation.sparMsg != "" {
			m.editor.deliberationMsg = fmt.Sprintf(" ⏳ %s (%s)", m.deliberation.sparMsg, elapsed)
		} else if m.deliberation.queuePos > 0 {
			m.editor.deliberationMsg = fmt.Sprintf(" ⏳ queued #%d (%s)", m.deliberation.queuePos, elapsed)
		} else {
			m.editor.deliberationMsg = fmt.Sprintf(" ⏳ deliberating... (%s)", elapsed)
		}
		return m, tea.Tick(time.Second, func(_ time.Time) tea.Msg {
			return deliberationTickMsg{}
		})

	case deliberationSubmittedMsg:
		if m.deliberation.status != deliberationActive {
			return m, nil
		}
		m.deliberation.queuePos = msg.position
		m.editor.copyMsg = fmt.Sprintf("submitted — queue #%d", msg.position)

		// Start polling and SSE stream in parallel.
		pollCmd := m.startPollCmd(msg)
		streamCmd := m.startStreamCmd(msg.submissionID, msg.client)
		return m, tea.Batch(pollCmd, streamCmd)

	case deliberationPollMsg:
		if m.deliberation.status != deliberationActive {
			return m, nil
		}
		if msg.err != nil {
			return m, tea.Tick(pollInterval, func(_ time.Time) tea.Msg {
				return m.pollOnce(msg.client, msg.submissionID, msg.caseID, msg.question, msg.gate, msg.vcSnapshot)
			})
		}
		switch msg.status.Status {
		case "completed":
			raw := msg.status.Verdict
			if len(raw) == 0 {
				// Fetch full case if verdict not inline.
				var err error
				raw, err = msg.client.GetCase(context.Background(), msg.caseID)
				if err != nil {
					return m, nil
				}
			}
			stashVerdict(raw, msg.question)
			// Trigger the result message to finalize.
			resultCmd := func() tea.Msg {
				return deliberationResultMsg{
					statusMsg:  formatVerdictSummary(raw, msg.gate),
					vcSnapshot: msg.vcSnapshot,
				}
			}
			return m, resultCmd
		case "failed":
			errMsg := "deliberation failed"
			if msg.status.Error != "" {
				errMsg = msg.status.Error
			}
			resultCmd := func() tea.Msg {
				return deliberationResultMsg{
					err:        fmt.Errorf("%s", errMsg),
					vcSnapshot: msg.vcSnapshot,
				}
			}
			return m, resultCmd
		default:
			m.deliberation.queuePos = msg.status.Position
			return m, tea.Tick(pollInterval, func(_ time.Time) tea.Msg {
				return m.pollOnce(msg.client, msg.submissionID, msg.caseID, msg.question, msg.gate, msg.vcSnapshot)
			})
		}

	case deliberationSparMsg:
		if m.deliberation.status != deliberationActive {
			return m, nil
		}
		m.deliberation.sparMsg = msg.event.Message
		// Continue reading from the stream.
		if m.deliberation.sparCh != nil && !msg.event.Final {
			ch := m.deliberation.sparCh
			return m, func() tea.Msg { return readNextSpar(ch) }
		}
		return m, nil

	case deliberationResultMsg:
		// Attach VectorCourt snapshot to flight record (best-effort).
		if m.recorder != nil && m.deliberation.flightID != "" && msg.vcSnapshot != nil {
			_ = m.recorder.UpdateVectorCourt(m.deliberation.flightID, msg.vcSnapshot)
		}
		m.deliberation.status = deliberationIdle
		m.deliberation.cancel = nil
		m.deliberation.flightID = ""
		m.deliberation.sparMsg = ""
		m.deliberation.queuePos = 0
		m.deliberation.sparCh = nil
		m.editor.deliberationMsg = ""
		if msg.err != nil {
			m.editor.copyStatus = copyError
			m.editor.copyMsg = fmt.Sprintf("vectorcourt: %v", msg.err)
		} else {
			m.editor.copyStatus = copyCopied
			m.editor.copyMsg = fmt.Sprintf("launched: %s", msg.statusMsg)
		}
		m.refreshFeedback()
		m.refreshAccountStatus()
		return m, nil

	case sidecarInjectResultMsg:
		m.deliberation.status = deliberationIdle
		m.deliberation.cancel = nil
		m.deliberation.flightID = ""
		m.deliberation.sparMsg = ""
		m.deliberation.queuePos = 0
		m.deliberation.sparCh = nil
		m.editor.deliberationMsg = ""
		if msg.err != nil {
			m.editor.copyStatus = copyError
			m.editor.copyMsg = fmt.Sprintf("sidecar: %v", msg.err)
		} else {
			m.editor.copyStatus = copyCopied
			m.editor.copyMsg = msg.statusMsg
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
			// Cancel in-progress deliberation on quit.
			if m.deliberation.cancel != nil {
				m.deliberation.cancel()
			}
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
		cmd = m.maybeSchedulePreflight(cmd)
		cmd = m.maybeScheduleInstantPrec(cmd)
		return m, m.maybeSchedulePrecedent(cmd)
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
	cmd = m.maybeSchedulePreflight(cmd)
	cmd = m.maybeScheduleInstantPrec(cmd)
	return m, m.maybeSchedulePrecedent(cmd)
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
		var cmd tea.Cmd
		if t != nil {
			cmd = m.executeLaunch(t)
		}
		m.launch.dismiss()
		return m, cmd
	case "1", "2", "3", "4", "5", "6", "7", "8":
		t := m.launch.selectByKey(msg.String())
		var cmd tea.Cmd
		if t != nil {
			cmd = m.executeLaunch(t)
		}
		m.launch.dismiss()
		return m, cmd
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

func (m *AppModel) executeLaunch(t *launchTarget) tea.Cmd {
	if !t.available {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = fmt.Sprintf("%s: not available", t.name)
		return nil
	}

	payload := m.editor.buildCopyPayload()
	if payload == "" {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = "nothing to launch"
		return nil
	}

	// Run pastewatch scan before launching.
	m.lastScan = detect.ScanPayload(m.caps, m.pwMode, payload)
	if !m.lastScan.Clean {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = "blocked: secrets detected (see risk panel)"
		return nil
	}

	// Record the launch in the flight log (before async path).
	flightID := m.recordLaunch(t.name, payload)

	// VectorCourt: async non-blocking submit.
	if t.name == "VectorCourt" {
		m.deliberation.flightID = flightID
		return m.startDeliberation(payload)
	}

	// Sidecar: inject into active Claude Code session.
	if t.name == "Sidecar (inject)" {
		return m.executeSidecarInject(payload)
	}

	// Sidecar + deliberate: VectorCourt first, then inject.
	if t.name == "Sidecar + deliberate" {
		m.deliberation.flightID = flightID
		return m.startSidecarDeliberation(payload)
	}

	statusMsg, err := t.action(payload)
	if err != nil {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = fmt.Sprintf("launch failed: %v", err)
		return nil
	}
	m.editor.copyStatus = copyCopied
	m.editor.copyMsg = fmt.Sprintf("launched: %s", statusMsg)

	// Refresh contextspectre feedback and VectorCourt account status after launch.
	m.refreshFeedback()
	m.refreshAccountStatus()
	return nil
}

// startDeliberation begins an async VectorCourt submit and returns the tick command.
func (m *AppModel) startDeliberation(payload string) tea.Cmd {
	// If deliberation already in progress, reject.
	if m.deliberation.status == deliberationActive {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = "deliberation already in progress"
		return nil
	}

	m.deliberation.status = deliberationActive
	m.deliberation.startTime = time.Now()
	m.editor.copyStatus = copyCopied
	m.editor.copyMsg = "deliberating... (0s)"

	ctx, cancel := context.WithCancel(context.Background())
	m.deliberation.cancel = cancel

	// Capture state for the goroutine.
	sentences := m.editor.sentences
	text := payload
	precedentCount := 0
	if m.risk.precedentSearch != nil {
		precedentCount = len(m.risk.precedentSearch.Precedents)
	}

	submitCmd := func() tea.Msg {
		cfg, err := config.Load()
		if err != nil {
			return deliberationResultMsg{err: fmt.Errorf("load config: %w", err)}
		}
		if cfg.VectorCourt.APIKey == "" {
			return deliberationResultMsg{err: fmt.Errorf("no API key configured")}
		}

		filing := vectorcourt.MapSentences(sentences)
		question := vectorcourt.ExtractQuestion(sentences, text)
		client := vectorcourt.NewClient(cfg.Endpoint(), cfg.VectorCourt.APIKey)

		// Preflight gate.
		gate, err := client.PreflightGate(ctx, question, filing)
		if err != nil {
			return deliberationResultMsg{err: fmt.Errorf("preflight: %w", err)}
		}
		if !gate.Allowed {
			snap := &flight.VectorCourtSnapshot{Preflight: "REJECTED"}
			if gate.Tier != "" {
				snap.Tier = gate.Tier
			}
			return deliberationResultMsg{err: fmt.Errorf("REJECTED: %s", gate.Reason), vcSnapshot: snap}
		}

		// Build VectorCourt snapshot from preflight gate.
		snap := &flight.VectorCourtSnapshot{
			Tier:           gate.Tier,
			Preflight:      "ACCEPTED",
			Warnings:       gate.Warnings,
			PrecedentCount: precedentCount,
		}
		if gate.Quality > 0 {
			snap.FilingQuality = gate.Quality
		}

		// Try async submit flow; fall back to sync Consult on 404/501.
		sub, submitErr := client.Submit(ctx, &vectorcourt.SubmitRequest{
			Question: question,
			Filing:   filing,
		})
		if submitErr != nil {
			var apiErr *vectorcourt.APIError
			if errors.As(submitErr, &apiErr) && (apiErr.StatusCode == http.StatusNotFound || apiErr.StatusCode == http.StatusNotImplemented) {
				// Server doesn't support async flow — use sync Consult.
				raw, err := client.Consult(ctx, &vectorcourt.ConsultRequest{
					Question: question,
					Filing:   filing,
				})
				if err != nil {
					return deliberationResultMsg{err: fmt.Errorf("consult: %w", err), vcSnapshot: snap}
				}
				stashVerdict(raw, question)
				return deliberationResultMsg{statusMsg: formatVerdictSummary(raw, gate), vcSnapshot: snap}
			}
			return deliberationResultMsg{err: fmt.Errorf("submit: %w", submitErr), vcSnapshot: snap}
		}

		// Async submit accepted — hand off to poll+stream phase.
		return deliberationSubmittedMsg{
			submissionID: sub.SubmissionID,
			caseID:       sub.CaseID,
			position:     sub.Position,
			client:       client,
			question:     question,
			gate:         gate,
			vcSnapshot:   snap,
		}
	}

	tickCmd := tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return deliberationTickMsg{}
	})

	return tea.Batch(submitCmd, tickCmd)
}

const pollInterval = 2 * time.Second

// startPollCmd returns a tea.Cmd that does the first poll after a delay.
func (m *AppModel) startPollCmd(msg deliberationSubmittedMsg) tea.Cmd {
	return tea.Tick(pollInterval, func(_ time.Time) tea.Msg {
		return m.pollOnce(msg.client, msg.submissionID, msg.caseID, msg.question, msg.gate, msg.vcSnapshot)
	})
}

// pollOnce performs a single poll and returns a deliberationPollMsg.
func (m *AppModel) pollOnce(client *vectorcourt.Client, submissionID, caseID, question string, gate *vectorcourt.GateResult, snap *flight.VectorCourtSnapshot) tea.Msg {
	status, err := client.PollSubmission(context.Background(), submissionID)
	if err != nil {
		return deliberationPollMsg{
			err:          err,
			client:       client,
			submissionID: submissionID,
			caseID:       caseID,
			question:     question,
			gate:         gate,
			vcSnapshot:   snap,
		}
	}

	return deliberationPollMsg{
		status:       status,
		client:       client,
		submissionID: submissionID,
		caseID:       caseID,
		question:     question,
		gate:         gate,
		vcSnapshot:   snap,
	}
}

// startStreamCmd connects to the SSE stream and returns the first event read command.
func (m *AppModel) startStreamCmd(submissionID string, client *vectorcourt.Client) tea.Cmd {
	endpoint := client.Endpoint()

	return func() tea.Msg {
		ch, err := vectorcourt.StreamSpar(context.Background(), endpoint, submissionID)
		if err != nil {
			return nil
		}
		m.deliberation.sparCh = ch
		return readNextSpar(ch)
	}
}

// readNextSpar reads the next event from the spar channel.
func readNextSpar(ch <-chan vectorcourt.SparEvent) tea.Msg {
	ev, ok := <-ch
	if !ok {
		return nil
	}
	return deliberationSparMsg{event: ev}
}

// executeSidecarInject injects the payload directly into the active Claude Code session.
func (m *AppModel) executeSidecarInject(payload string) tea.Cmd {
	cwd, err := os.Getwd()
	if err != nil {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = fmt.Sprintf("sidecar: %v", err)
		return nil
	}

	sessions, err := sidecar.DiscoverSessions(cwd)
	if err != nil || len(sessions) == 0 {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = "sidecar: no active session found"
		return nil
	}

	// Use the most recently modified session.
	session := sessions[0]
	if err := sidecar.InjectUserMessage(session.Path, payload); err != nil {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = fmt.Sprintf("sidecar inject failed: %v", err)
		return nil
	}

	m.editor.copyStatus = copyCopied
	m.editor.copyMsg = fmt.Sprintf("injected into session %s", session.ID[:8])
	return nil
}

// sidecarInjectResultMsg carries the result of a sidecar deliberate+inject.
type sidecarInjectResultMsg struct {
	statusMsg string
	err       error
}

// startSidecarDeliberation sends to VectorCourt for framing advice, then injects.
func (m *AppModel) startSidecarDeliberation(payload string) tea.Cmd {
	if m.deliberation.status == deliberationActive {
		m.editor.copyStatus = copyError
		m.editor.copyMsg = "deliberation already in progress"
		return nil
	}

	m.deliberation.status = deliberationActive
	m.deliberation.startTime = time.Now()
	m.editor.copyStatus = copyCopied
	m.editor.copyMsg = "deliberating framing... (0s)"

	ctx, cancel := context.WithCancel(context.Background())
	m.deliberation.cancel = cancel

	sentences := m.editor.sentences
	text := payload

	submitCmd := func() tea.Msg {
		cfg, err := config.Load()
		if err != nil {
			return deliberationResultMsg{err: fmt.Errorf("load config: %w", err)}
		}
		if cfg.VectorCourt.APIKey == "" {
			return deliberationResultMsg{err: fmt.Errorf("no API key configured")}
		}

		filing := vectorcourt.MapSentences(sentences)
		question := vectorcourt.ExtractQuestion(sentences, text)
		client := vectorcourt.NewClient(cfg.Endpoint(), cfg.VectorCourt.APIKey)

		// Preflight gate.
		gate, err := client.PreflightGate(ctx, question, filing)
		if err != nil {
			return deliberationResultMsg{err: fmt.Errorf("preflight: %w", err)}
		}
		if !gate.Allowed {
			return deliberationResultMsg{err: fmt.Errorf("REJECTED: %s", gate.Reason)}
		}

		// Submit for deliberation.
		sub, submitErr := client.Submit(ctx, &vectorcourt.SubmitRequest{
			Question: question,
			Filing:   filing,
		})

		var raw json.RawMessage
		if submitErr != nil {
			// Fallback to sync Consult.
			var apiErr *vectorcourt.APIError
			if errors.As(submitErr, &apiErr) && (apiErr.StatusCode == http.StatusNotFound || apiErr.StatusCode == http.StatusNotImplemented) {
				raw, err = client.Consult(ctx, &vectorcourt.ConsultRequest{
					Question: question,
					Filing:   filing,
				})
				if err != nil {
					return deliberationResultMsg{err: fmt.Errorf("consult: %w", err)}
				}
			} else {
				return deliberationResultMsg{err: fmt.Errorf("submit: %w", submitErr)}
			}
		} else {
			// Poll until completed.
			raw, err = pollForVerdict(ctx, client, sub.SubmissionID, sub.CaseID)
			if err != nil {
				return deliberationResultMsg{err: fmt.Errorf("poll: %w", err)}
			}
		}

		// Stash the verdict.
		stashVerdict(raw, question)

		// Now inject the original payload into the sidecar session.
		cwd, err := os.Getwd()
		if err != nil {
			return sidecarInjectResultMsg{err: fmt.Errorf("getwd: %w", err)}
		}
		sessions, err := sidecar.DiscoverSessions(cwd)
		if err != nil || len(sessions) == 0 {
			return sidecarInjectResultMsg{err: fmt.Errorf("no active session")}
		}
		if err := sidecar.InjectUserMessage(sessions[0].Path, text); err != nil {
			return sidecarInjectResultMsg{err: fmt.Errorf("inject: %w", err)}
		}

		summary := formatVerdictSummary(raw, gate)
		return sidecarInjectResultMsg{
			statusMsg: fmt.Sprintf("deliberated + injected: %s", summary),
		}
	}

	tickCmd := tea.Tick(time.Second, func(_ time.Time) tea.Msg {
		return deliberationTickMsg{}
	})

	return tea.Batch(submitCmd, tickCmd)
}

// pollForVerdict polls a submission until completed and returns the verdict.
func pollForVerdict(ctx context.Context, client *vectorcourt.Client, submissionID, caseID string) (json.RawMessage, error) {
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(pollInterval):
			status, err := client.PollSubmission(ctx, submissionID)
			if err != nil {
				continue
			}
			switch status.Status {
			case "completed":
				if len(status.Verdict) > 0 {
					return status.Verdict, nil
				}
				return client.GetCase(ctx, caseID)
			case "failed":
				if status.Error != "" {
					return nil, fmt.Errorf("deliberation failed: %s", status.Error)
				}
				return nil, fmt.Errorf("deliberation failed")
			}
		}
	}
}

// recordLaunch appends a flight record for the launch and returns its ID.
func (m *AppModel) recordLaunch(target, payload string) string {
	if m.recorder == nil {
		return ""
	}
	ns := negativespace.Analyze(payload)
	var gapClasses []string
	for _, g := range ns.Gaps {
		gapClasses = append(gapClasses, string(g.Class))
	}
	rec := flight.Record{
		ID:     flight.GenerateID(),
		Target: target,
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
	}
	_ = m.recorder.Append(rec)
	return rec.ID
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
