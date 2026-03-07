package ambiguity

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
)

const (
	amberThreshold             = 3.0
	redThreshold               = 6.0
	vagueVerbRatioThreshold    = 1.5
	specificDirectiveThreshold = 1.0
)

var (
	wordPattern         = regexp.MustCompile(`[\p{L}\p{N}]+(?:['’\-][\p{L}\p{N}]+)?`)
	explicitPathPattern = regexp.MustCompile(
		`(?i)(?:\./|\.\./|/)?[a-z0-9._-]+(?:/[a-z0-9._-]+)+|[a-z0-9_-]+\.(?:md|markdown|go|ts|tsx|js|jsx|json|yaml|yml|toml|txt|py|sh|swift|rs|java|rb|php|c|cc|cpp|h|hpp)`,
	)
	preservationPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bpreserve\b`),
		regexp.MustCompile(`(?i)\bkeep\b`),
		regexp.MustCompile(`(?i)\bdon'?t\s+change\b`),
		regexp.MustCompile(`(?i)\bdo\s+not\s+change\b`),
	}
	scopeMarkers = []scopeMarkerPattern{
		{name: "all repos", pattern: regexp.MustCompile(`(?i)\ball\s+repos\b`)},
		{name: "across the project", pattern: regexp.MustCompile(`(?i)\bacross\s+the\s+project\b`)},
		{name: "across projects", pattern: regexp.MustCompile(`(?i)\bacross\s+projects\b`)},
		{name: "across repos", pattern: regexp.MustCompile(`(?i)\bacross\s+repos\b`)},
		{name: "every repo", pattern: regexp.MustCompile(`(?i)\bevery\s+repo\b`)},
		{name: "across the codebase", pattern: regexp.MustCompile(`(?i)\bacross\s+the\s+codebase\b`)},
		{name: "whole codebase", pattern: regexp.MustCompile(`(?i)\bwhole\s+codebase\b`)},
		{name: "entire codebase", pattern: regexp.MustCompile(`(?i)\bentire\s+codebase\b`)},
		{name: "entire project", pattern: regexp.MustCompile(`(?i)\bentire\s+project\b`)},
		{name: "all files", pattern: regexp.MustCompile(`(?i)\ball\s+files\b`)},
		{name: "every file", pattern: regexp.MustCompile(`(?i)\bevery\s+file\b`)},
		{name: "all readmes", pattern: regexp.MustCompile(`(?i)\ball\s+readmes\b`)},
		{name: "every readme", pattern: regexp.MustCompile(`(?i)\bevery\s+readme\b`)},
		{name: "all directories", pattern: regexp.MustCompile(`(?i)\ball\s+directories\b`)},
		{name: "every directory", pattern: regexp.MustCompile(`(?i)\bevery\s+directory\b`)},
	}
)

type scopeMarkerPattern struct {
	name    string
	pattern *regexp.Regexp
}

// Scope is externally resolved execution scope.
type Scope struct {
	Repos     int
	Files     int
	FileTypes []string
	Targets   int
}

// BlastRadius summarizes resolved and text-derived target scope.
type BlastRadius struct {
	Repos         int      `json:"repos"`
	Files         int      `json:"files"`
	FileTypes     []string `json:"file_types,omitempty"`
	ExplicitPaths []string `json:"explicit_paths,omitempty"`
	ScopeMarkers  []string `json:"scope_markers,omitempty"`
	Targets       int      `json:"targets"`
}

// Severity is the ambiguity warning level.
type Severity string

const (
	SeverityNone  Severity = "none"
	SeverityAmber Severity = "amber"
	SeverityRed   Severity = "red"
)

// Warning captures non-blocking smoke detector state.
type Warning struct {
	Active      bool     `json:"active"`
	Severity    Severity `json:"severity"`
	Escalated   bool     `json:"escalated"`
	NonBlocking bool     `json:"non_blocking"`
	Message     string   `json:"message,omitempty"`
}

// Result contains deterministic ambiguity analysis for one directive.
type Result struct {
	Directive                  string      `json:"directive"`
	InstructionWords           int         `json:"instruction_words"`
	BrevityToScopeRatio        float64     `json:"brevity_to_scope_ratio"`
	BlastRadius                BlastRadius `json:"blast_radius"`
	VagueVerbs                 []string    `json:"vague_verbs,omitempty"`
	HasPreservationConstraints bool        `json:"has_preservation_constraints"`
	Warning                    Warning     `json:"warning"`
}

// Analyze computes deterministic ambiguity signals from directive text and resolved scope.
func Analyze(directive string, scope Scope) Result {
	trimmedDirective := strings.TrimSpace(directive)
	instructionWords := countInstructionWords(trimmedDirective)
	explicitPaths := detectExplicitPaths(trimmedDirective)
	scopeMarkerMatches := detectScopeMarkers(trimmedDirective)
	vagueVerbs := DetectVagueVerbs(trimmedDirective)
	hasPreservationConstraints := hasPreservationConstraints(trimmedDirective)

	blastRadius := computeBlastRadius(scope, explicitPaths, scopeMarkerMatches)
	ratio := computeBrevityToScopeRatio(blastRadius.Targets, instructionWords)
	warning := evaluateWarning(ratio, blastRadius.Targets, scope.Repos, len(vagueVerbs) > 0)

	return Result{
		Directive:                  trimmedDirective,
		InstructionWords:           instructionWords,
		BrevityToScopeRatio:        ratio,
		BlastRadius:                blastRadius,
		VagueVerbs:                 vagueVerbs,
		HasPreservationConstraints: hasPreservationConstraints,
		Warning:                    warning,
	}
}

// RenderHuman formats ambiguity analysis for terminal output.
func RenderHuman(result Result) string {
	var b strings.Builder

	b.WriteString("AMBIGUITY\n")
	b.WriteString(
		fmt.Sprintf(
			"  Blast radius: %d repos | %d files | %d targets\n",
			result.BlastRadius.Repos,
			result.BlastRadius.Files,
			result.BlastRadius.Targets,
		),
	)

	if len(result.BlastRadius.FileTypes) > 0 {
		b.WriteString(
			fmt.Sprintf("  File types: %s\n", strings.Join(result.BlastRadius.FileTypes, ", ")),
		)
	}

	if len(result.BlastRadius.ExplicitPaths) > 0 {
		b.WriteString(
			fmt.Sprintf("  Explicit paths: %s\n", strings.Join(result.BlastRadius.ExplicitPaths, ", ")),
		)
	}

	if len(result.BlastRadius.ScopeMarkers) > 0 {
		b.WriteString(
			fmt.Sprintf("  Scope markers: %s\n", strings.Join(result.BlastRadius.ScopeMarkers, ", ")),
		)
	}

	b.WriteString(
		fmt.Sprintf(
			"  Brevity ratio: %.2f (%d targets / %d words)\n",
			result.BrevityToScopeRatio,
			result.BlastRadius.Targets,
			result.InstructionWords,
		),
	)

	if len(result.VagueVerbs) > 0 {
		b.WriteString(
			fmt.Sprintf("  Vague verbs (English-only): %s\n", strings.Join(result.VagueVerbs, ", ")),
		)
	}

	if result.Warning.Active {
		b.WriteString(
			fmt.Sprintf("  Warning: %s (%s)\n", result.Warning.Severity, result.Warning.Message),
		)
	} else {
		b.WriteString("  Warning: none\n")
	}

	if result.Warning.NonBlocking {
		b.WriteString("  Non-blocking: dismiss and proceed\n")
	}

	return strings.TrimRight(b.String(), "\n")
}

// RenderJSON returns an indented JSON representation of ambiguity analysis.
func RenderJSON(result Result) (string, error) {
	body, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func countInstructionWords(directive string) int {
	if directive == "" {
		return 0
	}
	return len(wordPattern.FindAllString(directive, -1))
}

func detectExplicitPaths(directive string) []string {
	if directive == "" {
		return nil
	}

	matches := explicitPathPattern.FindAllString(directive, -1)
	if len(matches) == 0 {
		return nil
	}

	return uniquePreserveOrder(matches)
}

func detectScopeMarkers(directive string) []string {
	if directive == "" {
		return nil
	}

	matches := make([]string, 0)
	for _, marker := range scopeMarkers {
		if marker.pattern.MatchString(directive) {
			matches = append(matches, marker.name)
		}
	}

	if len(matches) == 0 {
		return nil
	}

	sort.Strings(matches)
	return matches
}

func hasPreservationConstraints(directive string) bool {
	for _, pattern := range preservationPatterns {
		if pattern.MatchString(directive) {
			return true
		}
	}
	return false
}

func computeBlastRadius(scope Scope, explicitPaths []string, scopeMarkerMatches []string) BlastRadius {
	fileTypes := uniquePreserveOrder(scope.FileTypes)
	resolvedTargets := scope.Targets
	if scope.Repos > resolvedTargets {
		resolvedTargets = scope.Repos
	}
	if scope.Files > resolvedTargets {
		resolvedTargets = scope.Files
	}
	if len(fileTypes) > resolvedTargets {
		resolvedTargets = len(fileTypes)
	}

	targets := resolvedTargets + len(explicitPaths) + len(scopeMarkerMatches)

	return BlastRadius{
		Repos:         scope.Repos,
		Files:         scope.Files,
		FileTypes:     fileTypes,
		ExplicitPaths: explicitPaths,
		ScopeMarkers:  scopeMarkerMatches,
		Targets:       targets,
	}
}

func computeBrevityToScopeRatio(targets int, instructionWords int) float64 {
	if targets <= 0 || instructionWords <= 0 {
		return 0
	}
	return float64(targets) / float64(instructionWords)
}

func evaluateWarning(ratio float64, targets int, repos int, hasVagueVerb bool) Warning {
	warning := Warning{
		Active:      false,
		Severity:    SeverityNone,
		Escalated:   false,
		NonBlocking: true,
	}

	if targets <= 0 || ratio < specificDirectiveThreshold {
		return warning
	}

	severity := SeverityNone
	switch {
	case ratio > redThreshold:
		severity = SeverityRed
	case ratio > amberThreshold:
		severity = SeverityAmber
	}

	isBroadScope := targets > 1 || repos > 1
	escalated := false
	if isBroadScope && hasVagueVerb && ratio > vagueVerbRatioThreshold {
		switch severity {
		case SeverityNone:
			severity = SeverityAmber
			escalated = true
		case SeverityAmber:
			severity = SeverityRed
			escalated = true
		}
	}

	if severity == SeverityNone {
		return warning
	}

	warning.Active = true
	warning.Severity = severity
	warning.Escalated = escalated
	warning.Message = warningMessage(severity, escalated)
	return warning
}

func warningMessage(severity Severity, escalated bool) string {
	switch {
	case severity == SeverityRed && escalated:
		return "short directive for broad scope with vague wording"
	case severity == SeverityRed:
		return "directive is very short relative to scope"
	case severity == SeverityAmber && escalated:
		return "vague wording increases ambiguity for this scope"
	case severity == SeverityAmber:
		return "directive is short relative to scope"
	default:
		return ""
	}
}

func uniquePreserveOrder(values []string) []string {
	if len(values) == 0 {
		return nil
	}

	seen := make(map[string]struct{}, len(values))
	unique := make([]string, 0, len(values))
	for _, value := range values {
		clean := strings.TrimSpace(value)
		if clean == "" {
			continue
		}
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		unique = append(unique, clean)
	}

	if len(unique) == 0 {
		return nil
	}

	return unique
}
