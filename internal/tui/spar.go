package tui

import (
	"fmt"
	"io"
	"strings"

	"github.com/ppiankov/vectorpad/internal/vectorcourt"
)

// ANSI color codes — works over SSH, no terminal library dependency.
const (
	ansiReset  = "\033[0m"
	ansiGreen  = "\033[32m"
	ansiRed    = "\033[31m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
	ansiBold   = "\033[1m"
	ansiDim    = "\033[2m"
)

// RenderSpar reads spar events from the channel and writes ANSI-formatted
// output to w. Blocks until the channel is closed or a final event arrives.
func RenderSpar(events <-chan vectorcourt.SparEvent, w io.Writer) {
	for ev := range events {
		switch ev.Stage {
		case "round_start":
			_, _ = fmt.Fprintf(w, "\n%s%s=== %s ===%s\n", ansiBold, ansiCyan, ev.Message, ansiReset)

		case "branch_action":
			color := personaColor(ev.Message)
			persona := padRight(ev.Persona, 12)
			_, _ = fmt.Fprintf(w, "  %s%s%s %s%s\n", color, persona, ansiReset, ev.Message, ansiReset)

		case "injection":
			_, _ = fmt.Fprintf(w, "  %s%s%s  %sinjection: %s%s\n", ansiYellow, padRight(ev.Persona, 12), ansiReset, ansiCyan, ev.Message, ansiReset)

		case "evidence":
			_, _ = fmt.Fprintf(w, "  %s%s%s  %sevidence: %s%s\n", ansiDim, padRight(ev.Persona, 12), ansiReset, ansiDim, ev.Message, ansiReset)

		case "censor_reopen":
			_, _ = fmt.Fprintf(w, "  %s%s%s  %scensor reopen: %s%s\n", ansiYellow, padRight(ev.Persona, 12), ansiReset, ansiYellow, ev.Message, ansiReset)

		case "completed":
			_, _ = fmt.Fprintf(w, "\n%s%s✓ %s%s\n", ansiBold, ansiGreen, ev.Message, ansiReset)

		case "failed":
			_, _ = fmt.Fprintf(w, "\n%s%s✗ %s%s\n", ansiBold, ansiRed, ev.Message, ansiReset)

		default:
			if ev.Message != "" {
				_, _ = fmt.Fprintf(w, "  %s[%s] %s%s\n", ansiDim, ev.Stage, ev.Message, ansiReset)
			}
		}
	}
}

// personaColor returns ANSI color based on action keywords in the message.
func personaColor(msg string) string {
	lower := strings.ToLower(msg)
	if strings.Contains(lower, "killed") || strings.Contains(lower, "rejected") {
		return ansiRed
	}
	if strings.Contains(lower, "split") || strings.Contains(lower, "new") {
		return ansiYellow
	}
	return ansiGreen
}

func padRight(s string, n int) string {
	if len(s) >= n {
		return s
	}
	return s + strings.Repeat(" ", n-len(s))
}
