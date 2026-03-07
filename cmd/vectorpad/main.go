package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/ppiankov/vectorpad/internal/ambiguity"
	"github.com/ppiankov/vectorpad/internal/classifier"
	"github.com/ppiankov/vectorpad/internal/preflight"
	"github.com/ppiankov/vectorpad/internal/stash"
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
		}
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
	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintln(stdout, "AMBIGUITY_JSON")
	_, _ = fmt.Fprintln(stdout, jsonAmbiguity)
	_, _ = fmt.Fprintln(stdout)
	_, _ = fmt.Fprintln(stdout, "PREFLIGHT_JSON")
	_, _ = fmt.Fprintln(stdout, jsonMetrics)
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
