package tui

import (
	"fmt"
	"time"

	"github.com/charmbracelet/bubbles/key"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

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

// AppModel is the top-level Bubbletea model for the three-panel TUI.
type AppModel struct {
	focus  panel
	stash  stashPanel
	editor editorPanel
	risk   riskPanel
	help   helpModel
	store  *stash.Store
	width  int
	height int
}

// NewApp creates the application model. store may be nil if stash is unavailable.
func NewApp(store *stash.Store) AppModel {
	m := AppModel{
		focus:  panelEditor,
		stash:  newStashPanel(),
		editor: newEditorPanel(),
		risk:   newRiskPanel(),
		store:  store,
	}
	m.editor.focus()
	m.loadStash()
	return m
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
			m.editor.copyAll()
			return m, nil
		}
		if key.Matches(msg, keys.Launch) {
			m.editor.copyAll()
			if m.editor.copyStatus == copyCopied {
				m.editor.copyMsg = "launched " + m.editor.copyMsg
			}
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
		m.risk.analyze(m.editor.value())
		return m, cmd
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
	return m, nil
}

func (m *AppModel) updateEditorKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// Editor gets all remaining keys as text input.
	cmd := m.editor.update(msg)
	m.risk.analyze(m.editor.value())
	return m, cmd
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
	m.risk.analyze(m.editor.value())
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

func (m AppModel) View() string {
	if m.width == 0 || m.height == 0 {
		return "initializing..."
	}

	// Help overlay takes over the whole screen.
	if m.help.visible {
		return m.help.View()
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
		content := m.risk.View(m.focus == panelRisk)
		border := panelBorderStyle(m.focus == panelRisk)
		panels = append(panels, border.Width(riskW-2).Height(m.height-2).Render(content))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top, panels...)
}

func panelBorderStyle(focused bool) lipgloss.Style {
	if focused {
		return styleFocusBorder.Copy()
	}
	return styleInactiveBorder.Copy()
}
