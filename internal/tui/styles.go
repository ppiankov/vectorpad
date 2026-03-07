package tui

import "github.com/charmbracelet/lipgloss"

var (
	colorGreen  = lipgloss.Color("#00CC66")
	colorYellow = lipgloss.Color("#FFCC00")
	colorRed    = lipgloss.Color("#FF4444")
	colorAmber  = lipgloss.Color("#FF8C00")
	colorMuted  = lipgloss.Color("#888888")
	colorAccent = lipgloss.Color("#7B68EE")
	colorWhite  = lipgloss.Color("#FFFFFF")
	colorDim    = lipgloss.Color("#555555")
	colorCyan   = lipgloss.Color("#00CCCC")
)

var (
	styleTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent)

	styleHeader = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorWhite)

	styleMuted = lipgloss.NewStyle().
			Foreground(colorMuted)

	styleSelected = lipgloss.NewStyle().
			Background(colorAccent).
			Foreground(colorWhite)

	styleWarning = lipgloss.NewStyle().
			Foreground(colorAmber).
			Bold(true)

	styleError = lipgloss.NewStyle().
			Foreground(colorRed).
			Bold(true)

	styleSuccess = lipgloss.NewStyle().
			Foreground(colorGreen)

	styleLocked = lipgloss.NewStyle().
			Foreground(colorCyan).
			Bold(true)

	styleFocusBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorAccent)

	styleInactiveBorder = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(colorDim)

	stylePanelTitle = lipgloss.NewStyle().
			Bold(true).
			Foreground(colorAccent).
			PaddingLeft(1)

	styleDim = lipgloss.NewStyle().
			Foreground(colorDim)

	styleStashFresh  = lipgloss.NewStyle().Foreground(colorWhite)
	styleStashRecent = lipgloss.NewStyle().Foreground(colorMuted)
	styleStashAging  = lipgloss.NewStyle().Foreground(colorDim)
	styleStashStale  = lipgloss.NewStyle().Foreground(lipgloss.Color("#333333"))
)

// severityColor returns a color for ambiguity warning severity.
func severityColor(severity string) lipgloss.Color {
	switch severity {
	case "red":
		return colorRed
	case "amber":
		return colorAmber
	default:
		return colorGreen
	}
}
