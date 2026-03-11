package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/ppiankov/vectorpad/internal/ambiguity"
	"github.com/ppiankov/vectorpad/internal/classifier"
	"github.com/ppiankov/vectorpad/internal/config"
	"github.com/ppiankov/vectorpad/internal/decompose"
	"github.com/ppiankov/vectorpad/internal/detect"
	"github.com/ppiankov/vectorpad/internal/flight"
	"github.com/ppiankov/vectorpad/internal/negativespace"
	"github.com/ppiankov/vectorpad/internal/oracul"
	"github.com/ppiankov/vectorpad/internal/preflight"
	"github.com/ppiankov/vectorpad/internal/pressure"
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
		case "log":
			return runLog(args[2:], stdout, stderr)
		case "stash":
			return runStash(args[2:], stdout, stderr)
		case "config":
			return runConfig(args[2:], stdout, stderr)
		case "submit":
			return runSubmit(args[2:], stdin, stdout, stderr)
		case "export":
			return runExport(args[2:], stdin, stdout, stderr)
		case "precedent":
			return runPrecedent(args[2:], stdin, stdout, stderr)
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
	// Pressure heat map.
	pressureScores := pressure.Score(sentences, ambiguityResult.VagueVerbs)
	if len(pressureScores) > 0 {
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintln(stdout, "PRESSURE")
		for i, ps := range pressureScores {
			var level string
			switch ps.Level {
			case pressure.LevelHigh:
				level = "HIGH"
			case pressure.LevelMedium:
				level = "MED"
			default:
				level = "LOW"
			}
			signals := ""
			if len(ps.Signals) > 0 {
				signals = " [" + strings.Join(ps.Signals, ", ") + "]"
			}
			_, _ = fmt.Fprintf(stdout, "  S%d: %s (%d)%s\n", i+1, level, ps.Score, signals)
		}
	}

	// Vector decomposition.
	decompResult := decompose.Decompose(sentences, 3)
	if decompResult.Triggered {
		_, _ = fmt.Fprintln(stdout)
		_, _ = fmt.Fprintf(stdout, "DECOMPOSE (%d sub-vectors)\n", len(decompResult.SubVectors))
		for i, sv := range decompResult.SubVectors {
			_, _ = fmt.Fprintf(stdout, "  %d. [%s] %d sentences\n", i+1, sv.Label, len(sv.Sentences))
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

func runLog(args []string, stdout io.Writer, stderr io.Writer) int {
	rec, err := flight.NewRecorder()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	// Parse flags.
	if len(args) > 0 && args[0] == "--stats" {
		stats, err := rec.ComputeStats()
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "Total launches: %d\n", stats.TotalLaunches)
		_, _ = fmt.Fprintf(stdout, "Annotated: %d\n", stats.Annotated)
		for outcome, count := range stats.OutcomeCounts {
			avg := stats.AvgCDRByOutcome[outcome]
			_, _ = fmt.Fprintf(stdout, "  %s: %d (avg CDR: %.2f)\n", outcome, count, avg)
		}
		if len(stats.TopGaps) > 0 {
			_, _ = fmt.Fprintln(stdout, "Top gaps:")
			for _, g := range stats.TopGaps {
				_, _ = fmt.Fprintf(stdout, "  %s: %d\n", g.Class, g.Count)
			}
		}
		if stats.Oracul != nil {
			_, _ = fmt.Fprintln(stdout, "Oracul:")
			_, _ = fmt.Fprintf(stdout, "  submits: %d\n", stats.Oracul.TotalSubmits)
			_, _ = fmt.Fprintf(stdout, "  avg filing quality: %.0f%%\n", stats.Oracul.AvgFilingQuality*100)
			_, _ = fmt.Fprintf(stdout, "  rejection rate: %.0f%%\n", stats.Oracul.RejectionRate*100)
			if len(stats.Oracul.TopWarnings) > 0 {
				_, _ = fmt.Fprintln(stdout, "  top warnings:")
				for _, w := range stats.Oracul.TopWarnings {
					_, _ = fmt.Fprintf(stdout, "    %s: %d\n", w.Class, w.Count)
				}
			}
		}
		return 0
	}

	if len(args) >= 3 && args[0] == "--annotate" {
		id := args[1]
		outcome := args[2]
		note := ""
		if len(args) > 3 {
			note = strings.Join(args[3:], " ")
		}
		if err := rec.Annotate(id, outcome, note); err != nil {
			_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "annotated %s: %s\n", id, outcome)
		return 0
	}

	// Default: show recent launches.
	records, err := rec.Recent(10)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if len(records) == 0 {
		_, _ = fmt.Fprintln(stdout, "no launches recorded yet")
		return 0
	}
	for _, r := range records {
		ts := r.Launched.Format("2006-01-02 15:04")
		outcome := r.Outcome
		if outcome == "" {
			outcome = "-"
		}
		text := r.Text
		if len(text) > 60 {
			text = text[:57] + "..."
		}
		_, _ = fmt.Fprintf(stdout, "%s  %s  [%s] CDR:%.2f  %s\n", r.ID[:8], ts, outcome, r.Metrics.CDR, text)
	}
	return 0
}

func runStash(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "usage: vectorpad stash <add|list|compare|show|cluster|evolve|reindex>")
		return 1
	}

	store, err := stash.NewDefaultStore()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	switch args[0] {
	case "add":
		return stashAdd(args[1:], store, stdout, stderr)
	case "list":
		return stashList(args[1:], store, stdout, stderr)
	case "compare":
		return stashCompare(args[1:], store, stdout, stderr)
	case "show":
		return stashShow(args[1:], store, stdout, stderr)
	case "cluster":
		return stashCluster(store, stdout, stderr)
	case "evolve":
		return stashEvolve(args[1:], store, stdout, stderr)
	case "reindex":
		return stashReindex(store, stdout, stderr)
	default:
		_, _ = fmt.Fprintf(stderr, "unknown stash command: %s\n", args[0])
		return 1
	}
}

func stashAdd(args []string, store *stash.Store, stdout io.Writer, stderr io.Writer) int {
	// Parse flags: --type, --project, --title, --tag
	var text, title, project, tag string
	var itemType stash.ItemType
	positional := []string{}

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--type":
			if i+1 < len(args) {
				i++
				itemType = stash.ItemType(args[i])
			}
		case "--project":
			if i+1 < len(args) {
				i++
				project = args[i]
			}
		case "--title":
			if i+1 < len(args) {
				i++
				title = args[i]
			}
		case "--tag":
			if i+1 < len(args) {
				i++
				tag = args[i]
			}
		default:
			positional = append(positional, args[i])
		}
	}

	text = strings.TrimSpace(strings.Join(positional, " "))
	if text == "" {
		_, _ = fmt.Fprintln(stderr, "error: stash add requires idea text")
		return 1
	}

	var tags []string
	if tag != "" {
		tags = strings.Split(tag, ",")
	}

	item, err := store.AddWithMeta(text, stash.SourceCLI, title, itemType, project, tags)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "stashed %s", item.ID[:8])
	if item.ClaimID != "" {
		_, _ = fmt.Fprintf(stdout, " [%s]", item.ClaimID)
	}
	_, _ = fmt.Fprintln(stdout)
	return 0
}

func stashList(args []string, store *stash.Store, stdout io.Writer, stderr io.Writer) int {
	db := store.DB()
	if db == nil {
		// Fall back to JSON stash listing.
		file, err := store.Load()
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		for _, stack := range file.Stacks {
			for _, item := range stack.Items {
				printItem(stdout, item)
			}
		}
		return 0
	}

	// Parse filters.
	var project, itemType, tag string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--project":
			if i+1 < len(args) {
				i++
				project = args[i]
			}
		case "--type":
			if i+1 < len(args) {
				i++
				itemType = args[i]
			}
		case "--tag":
			if i+1 < len(args) {
				i++
				tag = args[i]
			}
		}
	}

	items, err := db.Filter(project, itemType, tag)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if len(items) == 0 {
		_, _ = fmt.Fprintln(stdout, "no stash entries")
		return 0
	}
	for _, item := range items {
		printItem(stdout, item)
	}
	return 0
}

func stashCompare(args []string, store *stash.Store, stdout io.Writer, stderr io.Writer) int {
	text := strings.TrimSpace(strings.Join(args, " "))
	if text == "" {
		_, _ = fmt.Fprintln(stderr, "error: stash compare requires text to compare")
		return 1
	}

	db := store.DB()
	embedder := store.EmbedderClient()
	if db == nil || embedder == nil || !embedder.Available() {
		_, _ = fmt.Fprintln(stderr, "error: compare requires SQLite stash and Ollama (run: ollama pull nomic-embed-text)")
		return 1
	}

	vec, err := embedder.Embed(text)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: embed failed: %v\n", err)
		return 1
	}

	results, err := db.FindSimilar(vec, stash.ThresholdRelated(), 10)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if len(results) == 0 {
		_, _ = fmt.Fprintln(stdout, "no similar entries found")
		return 0
	}

	for _, r := range results {
		text := r.Item.Text
		if len(text) > 60 {
			text = text[:57] + "..."
		}
		_, _ = fmt.Fprintf(stdout, "%.2f [%s] %s  %s\n", r.Score, r.Level, r.Item.ID[:8], text)
	}
	return 0
}

func stashShow(args []string, store *stash.Store, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "error: stash show requires an ID")
		return 1
	}

	db := store.DB()
	if db == nil {
		_, _ = fmt.Fprintln(stderr, "error: stash show requires SQLite stash")
		return 1
	}

	item, err := db.Get(args[0])
	if err != nil {
		// Try prefix match.
		items, allErr := db.All()
		if allErr != nil {
			_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		found := false
		for _, it := range items {
			if strings.HasPrefix(it.ID, args[0]) {
				item = it
				found = true
				break
			}
		}
		if !found {
			_, _ = fmt.Fprintf(stderr, "error: item not found: %s\n", args[0])
			return 1
		}
	}

	_, _ = fmt.Fprintf(stdout, "ID:       %s\n", item.ID)
	if item.ClaimID != "" {
		_, _ = fmt.Fprintf(stdout, "Claim:    %s\n", item.ClaimID)
	}
	if item.Title != "" {
		_, _ = fmt.Fprintf(stdout, "Title:    %s\n", item.Title)
	}
	if item.Type != "" {
		_, _ = fmt.Fprintf(stdout, "Type:     %s\n", string(item.Type))
	}
	if item.Project != "" {
		_, _ = fmt.Fprintf(stdout, "Project:  %s\n", item.Project)
	}
	if len(item.Tags) > 0 {
		_, _ = fmt.Fprintf(stdout, "Tags:     %s\n", strings.Join(item.Tags, ", "))
	}
	_, _ = fmt.Fprintf(stdout, "Created:  %s\n", item.Created.Format("2006-01-02 15:04"))
	_, _ = fmt.Fprintf(stdout, "Source:   %s\n", string(item.Source))
	hasEmbed := "no"
	if len(item.Embedding) > 0 {
		hasEmbed = fmt.Sprintf("yes (%d dims)", len(item.Embedding))
	}
	_, _ = fmt.Fprintf(stdout, "Embedded: %s\n", hasEmbed)
	_, _ = fmt.Fprintf(stdout, "\n%s\n", item.Text)
	return 0
}

func stashCluster(store *stash.Store, stdout io.Writer, stderr io.Writer) int {
	file, err := store.Load()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if len(file.Stacks) == 0 {
		_, _ = fmt.Fprintln(stdout, "no stash entries")
		return 0
	}

	for _, stack := range file.Stacks {
		_, _ = fmt.Fprintf(stdout, "[%s] %s (%d items)\n", stack.ID, stack.Label, len(stack.Items))
		for _, item := range stack.Items {
			text := item.Text
			if len(text) > 60 {
				text = text[:57] + "..."
			}
			_, _ = fmt.Fprintf(stdout, "  %s  %s\n", item.ID[:8], text)
		}
	}
	return 0
}

func stashEvolve(args []string, store *stash.Store, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "error: stash evolve requires a claim ID")
		return 1
	}

	db := store.DB()
	if db == nil {
		_, _ = fmt.Fprintln(stderr, "error: stash evolve requires SQLite stash")
		return 1
	}

	items, err := db.ByClaimID(args[0])
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if len(items) == 0 {
		_, _ = fmt.Fprintf(stdout, "no entries for claim %s\n", args[0])
		return 0
	}

	_, _ = fmt.Fprintf(stdout, "Claim %s: %d versions\n\n", args[0], len(items))
	for i, item := range items {
		_, _ = fmt.Fprintf(stdout, "v%d  %s  %s\n", i+1, item.Created.Format("2006-01-02 15:04"), item.ID[:8])
		_, _ = fmt.Fprintf(stdout, "  %s\n\n", item.Text)
	}
	return 0
}

func stashReindex(store *stash.Store, stdout io.Writer, stderr io.Writer) int {
	db := store.DB()
	embedder := store.EmbedderClient()
	if db == nil || embedder == nil || !embedder.Available() {
		_, _ = fmt.Fprintln(stderr, "error: reindex requires SQLite stash and Ollama (run: ollama pull nomic-embed-text)")
		return 1
	}

	items, err := db.All()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	reindexed := 0
	for _, item := range items {
		vec, err := embedder.Embed(item.Text)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "warn: failed to embed %s: %v\n", item.ID[:8], err)
			continue
		}
		if err := db.UpdateEmbedding(item.ID, vec); err != nil {
			_, _ = fmt.Fprintf(stderr, "warn: failed to update %s: %v\n", item.ID[:8], err)
			continue
		}
		reindexed++
	}

	_, _ = fmt.Fprintf(stdout, "reindexed %d/%d items\n", reindexed, len(items))
	return 0
}

func printItem(stdout io.Writer, item stash.Item) {
	text := item.Text
	if len(text) > 60 {
		text = text[:57] + "..."
	}
	id := item.ID
	if len(id) > 8 {
		id = id[:8]
	}
	symbol := "●"
	if item.Source == stash.SourceVerdict {
		symbol = "◆"
	}
	meta := ""
	if item.Type != "" {
		meta += " [" + string(item.Type) + "]"
	}
	if item.Project != "" {
		meta += " (" + item.Project + ")"
	}
	_, _ = fmt.Fprintf(stdout, "%s %s%s  %s\n", symbol, id, meta, text)
}

func runConfig(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) < 2 {
		_, _ = fmt.Fprintln(stderr, "usage: vectorpad config <set|get> <key> [value]")
		return 1
	}

	switch args[0] {
	case "set":
		if len(args) < 3 {
			_, _ = fmt.Fprintln(stderr, "usage: vectorpad config set <key> <value>")
			return 1
		}
		if err := config.Set(args[1], args[2]); err != nil {
			_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintf(stdout, "set %s\n", args[1])
		return 0

	case "get":
		val, err := config.Get(args[1])
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		// Mask API keys in output.
		if strings.Contains(args[1], "api_key") && len(val) > 8 {
			val = val[:4] + strings.Repeat("*", len(val)-8) + val[len(val)-4:]
		}
		_, _ = fmt.Fprintln(stdout, val)
		return 0

	default:
		_, _ = fmt.Fprintf(stderr, "unknown config command: %s\n", args[0])
		return 1
	}
}

func runSubmit(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	// Parse flags.
	var target, output string
	dryRun := false
	noPreflight := false

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--to":
			if i+1 < len(args) {
				i++
				target = args[i]
			}
		case "--output", "-o":
			if i+1 < len(args) {
				i++
				output = args[i]
			}
		case "--dry-run":
			dryRun = true
		case "--no-preflight":
			noPreflight = true
		}
	}

	if target != "oracul" {
		_, _ = fmt.Fprintln(stderr, "usage: vectorpad submit --to oracul [--output file.json] [--dry-run] [--no-preflight]")
		return 1
	}

	// Load config.
	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if cfg.Oracul.APIKey == "" {
		_, _ = fmt.Fprintln(stderr, "error: no API key configured (run: vectorpad config set oracul.api_key <key>)")
		return 1
	}

	// Read input.
	input, err := io.ReadAll(stdin)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	text := strings.TrimSpace(string(input))
	if text == "" {
		_, _ = fmt.Fprintln(stderr, "error: empty input")
		return 1
	}

	// Classify and map.
	sentences := classifier.Classify(text)
	filing := oracul.MapSentences(sentences)
	question := oracul.ExtractQuestion(sentences, text)

	req := &oracul.ConsultRequest{
		Question: question,
		Filing:   filing,
	}

	// Dry run: print request and exit.
	if dryRun {
		data, err := json.MarshalIndent(req, "", "  ")
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintln(stdout, string(data))
		return 0
	}

	client := oracul.NewClient(cfg.Endpoint(), cfg.Oracul.APIKey)

	// Preflight gate.
	if !noPreflight {
		_, _ = fmt.Fprintln(stderr, "running preflight check...")
		gate, err := client.PreflightGate(context.Background(), question, filing)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "preflight error: %v\n", err)
			return 1
		}

		if !gate.Allowed {
			_, _ = fmt.Fprintf(stderr, "REJECTED: %s\n", gate.Reason)
			return 1
		}

		if len(gate.Warnings) > 0 {
			_, _ = fmt.Fprintf(stderr, "preflight warnings (tier: %s, quality: %.0f%%):\n", gate.Tier, gate.Quality*100)
			for _, w := range gate.Warnings {
				_, _ = fmt.Fprintf(stderr, "  - %s\n", w)
			}
		} else {
			_, _ = fmt.Fprintf(stderr, "preflight: ACCEPTED (tier: %s, quality: %.0f%%)\n", gate.Tier, gate.Quality*100)
		}
	}

	// Submit.
	_, _ = fmt.Fprintln(stderr, "deliberation in progress...")
	raw, err := client.Consult(context.Background(), req)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	// Pretty-print the response.
	var formatted []byte
	var pretty json.RawMessage
	if json.Unmarshal(raw, &pretty) == nil {
		if f, err := json.MarshalIndent(pretty, "", "  "); err == nil {
			formatted = f
		}
	}
	if formatted == nil {
		formatted = raw
	}

	// Write to file if --output specified.
	if output != "" {
		if err := os.WriteFile(output, append(formatted, '\n'), 0644); err != nil {
			_, _ = fmt.Fprintf(stderr, "error: write %s: %v\n", output, err)
			return 1
		}
		_, _ = fmt.Fprintf(stderr, "verdict written to %s\n", output)
		return 0
	}

	_, _ = fmt.Fprintln(stdout, string(formatted))
	return 0
}

func runExport(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	var format string
	for i := 0; i < len(args); i++ {
		if args[i] == "--format" && i+1 < len(args) {
			i++
			format = args[i]
		}
	}

	if format != "oracul" {
		_, _ = fmt.Fprintln(stderr, "usage: vectorpad export --format oracul")
		return 1
	}

	input, err := io.ReadAll(stdin)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	text := strings.TrimSpace(string(input))
	if text == "" {
		_, _ = fmt.Fprintln(stderr, "error: empty input")
		return 1
	}

	sentences := classifier.Classify(text)
	filing := oracul.MapSentences(sentences)
	question := oracul.ExtractQuestion(sentences, text)

	req := &oracul.ConsultRequest{
		Question: question,
		Filing:   filing,
	}

	data, err := json.MarshalIndent(req, "", "  ")
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintln(stdout, string(data))
	return 0
}

func runPrecedent(args []string, stdin io.Reader, stdout io.Writer, stderr io.Writer) int {
	limit := 5
	jsonOutput := false
	var query string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--limit":
			if i+1 < len(args) {
				i++
				n, err := strconv.Atoi(args[i])
				if err == nil && n > 0 && n <= 20 {
					limit = n
				}
			}
		case "--json":
			jsonOutput = true
		default:
			query = args[i]
		}
	}

	// Read from stdin if no positional arg.
	if query == "" {
		input, err := io.ReadAll(stdin)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		query = strings.TrimSpace(string(input))
	}
	if query == "" {
		_, _ = fmt.Fprintln(stderr, "usage: vectorpad precedent [--limit N] [--json] <question>")
		return 1
	}

	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}
	if cfg.Oracul.APIKey == "" {
		_, _ = fmt.Fprintln(stderr, "error: no API key configured (run: vectorpad config set oracul.api_key <key>)")
		return 1
	}

	// Extract question from classified text.
	sentences := classifier.Classify(query)
	question := oracul.ExtractQuestion(sentences, query)

	client := oracul.NewClient(cfg.Endpoint(), cfg.Oracul.APIKey)
	result, err := client.SearchPrecedents(context.Background(), question, limit)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
		return 1
	}

	if jsonOutput {
		data, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "error: %v\n", err)
			return 1
		}
		_, _ = fmt.Fprintln(stdout, string(data))
		return 0
	}

	// Human-readable output.
	if len(result.Precedents) == 0 {
		_, _ = fmt.Fprintln(stdout, "No precedents found.")
		return 0
	}

	_, _ = fmt.Fprintf(stdout, "Found %d precedents (%d similar cases):\n\n", len(result.Precedents), result.TotalSimilar)
	for i, pr := range result.Precedents {
		_, _ = fmt.Fprintf(stdout, "%d. [%.2f] %s\n", i+1, pr.SimilarityScore, pr.Question)
		_, _ = fmt.Fprintf(stdout, "   Status: %s | Confidence: %.2f\n", pr.VerdictStatus, pr.Confidence)
		if pr.OutcomeCount > 0 {
			_, _ = fmt.Fprintf(stdout, "   Outcomes: %d (%.0f%% correct)\n", pr.OutcomeCount, pr.OutcomeCorrectRate*100)
		}
		if len(pr.ClaimFamilies) > 0 {
			_, _ = fmt.Fprintf(stdout, "   Families: %s\n", strings.Join(pr.ClaimFamilies, ", "))
		}
		_, _ = fmt.Fprintln(stdout)
	}

	if rc := result.RefClassSummary; rc != nil {
		_, _ = fmt.Fprintf(stdout, "Reference class: %d/%d resolved, %.1f%% success\n", rc.ResolvedCases, rc.TotalCases, rc.SuccessRate*100)
		if len(rc.TopClaimFamilies) > 0 {
			_, _ = fmt.Fprintf(stdout, "Top families: %s\n", strings.Join(rc.TopClaimFamilies, ", "))
		}
	}

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
        COMPREPLY=($(compgen -W "tui add stash version completion log config submit export precedent" -- "${cur}"))
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
        'log:view flight log'
        'stash:manage claim registry'
        'config:get or set configuration'
        'submit:submit case to Oracul for deliberation'
        'export:export classified case as JSON'
        'precedent:search for similar past decisions'
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
complete -c vectorpad -n '__fish_use_subcommand' -a log -d 'View flight log'
complete -c vectorpad -n '__fish_use_subcommand' -a stash -d 'Manage claim registry'
complete -c vectorpad -n '__fish_use_subcommand' -a config -d 'Get or set configuration'
complete -c vectorpad -n '__fish_use_subcommand' -a submit -d 'Submit case to Oracul'
complete -c vectorpad -n '__fish_use_subcommand' -a export -d 'Export classified case'
complete -c vectorpad -n '__fish_use_subcommand' -a precedent -d 'Search similar past decisions'
complete -c vectorpad -n '__fish_seen_subcommand_from completion' -a 'bash zsh fish'
`

func isTerminal(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
