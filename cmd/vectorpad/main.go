package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ppiankov/vectorpad/internal/ambiguity"
	"github.com/ppiankov/vectorpad/internal/classifier"
	"github.com/ppiankov/vectorpad/internal/detect"
	"github.com/ppiankov/vectorpad/internal/negativespace"
	"github.com/ppiankov/vectorpad/internal/preflight"
	"github.com/ppiankov/vectorpad/internal/stash"
	"github.com/ppiankov/vectorpad/internal/tui"
	"github.com/ppiankov/vectorpad/internal/vector"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args, os.Stdin, os.Stdout, os.Stderr))
}

func run(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	if len(args) > 1 {
		switch args[1] {
		case "version":
			_, _ = fmt.Fprintln(stdout, "vectorpad", version)
			return 0
		case "add":
			return runAdd(args[2:], stdout, stderr)
		case "tui":
			return runTUI(stderr)
		case "completion":
			return runCompletion(args[2:], stdout, stderr)
		}
	}

	// No subcommand: if stdin is a terminal, launch TUI; otherwise read pipe.
	if f, ok := stdin.(*os.File); ok && isTerminal(f) {
		return runTUI(stderr)
	}

	input, err := io.ReadAll(stdin)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: failed to read stdin: %v\n", err)
		return 1
	}

	sentences := classifier.Classify(string(input))
	block := vector.Render(sentences)
	metrics := preflight.Compute(string(input), sentences)
	humanMetrics := preflight.RenderHuman(metrics)
	jsonMetrics, err := preflight.RenderJSON(metrics)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: failed to encode preflight metrics: %v\n", err)
		return 1
	}

	ambiguityResult := ambiguity.Analyze(string(input), ambiguity.Scope{})
	humanAmbiguity := ambiguity.RenderHuman(ambiguityResult)
	jsonAmbiguity, err := ambiguity.RenderJSON(ambiguityResult)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: failed to encode ambiguity analysis: %v\n", err)
		return 1
	}

	nudges := ambiguity.SelectNudges(ambiguityResult)
	negSpace := negativespace.Analyze(string(input))

	_, _ = fmt.Fprintln(stdout, block)
	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintln(stdout, humanMetrics)
	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintln(stdout, humanAmbiguity)
	if len(nudges) > 0 {
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintln(stdout, "NUDGES")
		for _, nudge := range nudges {
			_, _ = fmt.Fprintf(stdout, "  - [%s] %s\n", nudge.Type, nudge.Prompt)
		}
	}
	if !negSpace.Clean() {
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintln(stdout, "GAPS (what you didn't say)")
		for _, gap := range negSpace.Gaps {
			_, _ = fmt.Fprintf(stdout, "  [%s] %s\n", gap.Class, gap.Description)
			_, _ = fmt.Fprintf(stdout, "    → %s\n", gap.NudgePrompt)
		}
	}
	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintln(stdout, "AMBIGUITY_JSON")
	_, _ = fmt.Fprintln(stdout, jsonAmbiguity)
	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintln(stdout, "PREFLIGHT_JSON")
	_, _ = fmt.Fprintln(stdout, jsonMetrics)
	return 0
}

func runTUI(stderr io.Writer) int {
	store, _ := stash.NewDefaultStore()
	caps := detect.Detect()
	app := tui.NewApp(store, caps)
	p := tea.NewProgram(app, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	return 0
}

func runAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	text := strings.TrimSpace(strings.Join(args, " "))
	if text == "" {
		_, _ = fmt.Fprintln(stderr, "error: add requires an idea string")
		return 1
	}

	store, err := stash.NewDefaultStore()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: failed to resolve stash path: %v\n", err)
		return 1
	}

	item, err := store.Add(text, stash.SourceCLI)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: failed to stash idea: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "stashed %s\n", item.ID)
	return 0
}

func runCompletion(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: vectorpad completion <bash|zsh|fish>")
		return 1
	}
	switch args[0] {
	case "bash":
		_, _ = fmt.Fprint(stdout, completionBash)
	case "zsh":
		_, _ = fmt.Fprint(stdout, completionZsh)
	case "fish":
		_, _ = fmt.Fprint(stdout, completionFish)
	default:
		_, _ = fmt.Fprintf(stderr, "unknown shell: %s (supported: bash, zsh, fish)\n", args[0])
		return 1
	}
	return 0
}

const completionBash = `# vectorpad bash completion
_vectorpad() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local prev="${COMP_WORDS[COMP_CWORD-1]}"

    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=($(compgen -W "tui add version completion" -- "${cur}"))
        return 0
    fi

    case "${prev}" in
        completion)
            COMPREPLY=($(compgen -W "bash zsh fish" -- "${cur}"))
            return 0
            ;;
    esac
}
complete -F _vectorpad vectorpad
`

const completionZsh = `#compdef vectorpad

_vectorpad() {
    local -a commands
    commands=(
        'tui:launch interactive TUI'
        'add:quick-add idea to stash'
        'version:print version'
        'completion:generate shell completions'
    )

    _arguments -C \
        '1:command:->command' \
        '*::arg:->args'

    case $state in
        command)
            _describe 'command' commands
            ;;
        args)
            case $words[1] in
                completion)
                    _values 'shell' bash zsh fish
                    ;;
            esac
            ;;
    esac
}

_vectorpad "$@"
`

const completionFish = `# vectorpad fish completion
complete -c vectorpad -f
complete -c vectorpad -n '__fish_use_subcommand' -a tui -d 'Launch interactive TUI'
complete -c vectorpad -n '__fish_use_subcommand' -a add -d 'Quick-add idea to stash'
complete -c vectorpad -n '__fish_use_subcommand' -a version -d 'Print version'
complete -c vectorpad -n '__fish_use_subcommand' -a completion -d 'Generate shell completions'
complete -c vectorpad -n '__fish_seen_subcommand_from completion' -a 'bash zsh fish'
`

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
