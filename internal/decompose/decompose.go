package decompose

import (
	"regexp"
	"strings"

	"github.com/ppiankov/vectorpad/internal/classifier"
)

// SubVector is a focused subset of sentences extracted from a broad directive.
type SubVector struct {
	Label     string                `json:"label"`
	Sentences []classifier.Sentence `json:"sentences"`
	Text      string                `json:"text"`
}

// Result holds the decomposition output.
type Result struct {
	Original   string      `json:"original"`
	SubVectors []SubVector `json:"sub_vectors"`
	Triggered  bool        `json:"triggered"` // true if decomposition was warranted
}

// targetPatterns detect scope references in sentences.
var targetPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b[\w-]+/[\w.-]+\b`),                                              // repo-like: org/repo
	regexp.MustCompile(`(?i)\b[\w.-]+\.(go|py|ts|js|md|yml|yaml|json|toml|rs|swift|rb|sh)\b`), // file names
	regexp.MustCompile(`(?i)\b(src|cmd|internal|pkg|lib|app|tests?|docs?)/[\w.-]+\b`),         // path segments
}

// scopeGroupPatterns identify broad scope markers that define target groups.
var scopeGroupPatterns = []struct {
	name    string
	pattern *regexp.Regexp
}{
	{"all repos", regexp.MustCompile(`(?i)\ball\s+repos?\b`)},
	{"every repo", regexp.MustCompile(`(?i)\bevery\s+repo\b`)},
	{"all services", regexp.MustCompile(`(?i)\ball\s+services?\b`)},
	{"all packages", regexp.MustCompile(`(?i)\ball\s+packages?\b`)},
	{"all files", regexp.MustCompile(`(?i)\ball\s+files?\b`)},
	{"each repo", regexp.MustCompile(`(?i)\beach\s+repo\b`)},
}

// Decompose splits a classified sentence list into focused sub-vectors.
// It triggers when blast radius > threshold targets.
func Decompose(sentences []classifier.Sentence, threshold int) Result {
	if threshold <= 0 {
		threshold = 3
	}

	fullText := joinSentences(sentences)
	groups := groupByTarget(sentences)

	// Don't decompose if there aren't enough distinct target groups.
	if len(groups) <= 1 {
		return Result{Original: fullText, Triggered: false}
	}

	// Count total distinct targets across groups.
	totalTargets := 0
	for _, g := range groups {
		totalTargets += len(g.targets)
	}
	if totalTargets < threshold {
		return Result{Original: fullText, Triggered: false}
	}

	// Build sub-vectors: shared preamble + group-specific sentences.
	shared := findSharedSentences(sentences, groups)
	subVectors := buildSubVectors(shared, groups)

	return Result{
		Original:   fullText,
		SubVectors: subVectors,
		Triggered:  true,
	}
}

type sentenceGroup struct {
	label     string
	targets   []string
	sentences []classifier.Sentence
}

func groupByTarget(sentences []classifier.Sentence) []sentenceGroup {
	groupMap := make(map[string]*sentenceGroup)
	var groupOrder []string

	for _, s := range sentences {
		targets := extractTargets(s.Text)
		if len(targets) == 0 {
			continue // no specific target — will be shared
		}

		// Use first target as group key.
		key := targets[0]
		if g, ok := groupMap[key]; ok {
			g.sentences = append(g.sentences, s)
			// Merge targets.
			for _, t := range targets[1:] {
				if !containsStr(g.targets, t) {
					g.targets = append(g.targets, t)
				}
			}
		} else {
			groupMap[key] = &sentenceGroup{
				label:     key,
				targets:   targets,
				sentences: []classifier.Sentence{s},
			}
			groupOrder = append(groupOrder, key)
		}
	}

	groups := make([]sentenceGroup, 0, len(groupOrder))
	for _, key := range groupOrder {
		groups = append(groups, *groupMap[key])
	}
	return groups
}

func extractTargets(text string) []string {
	var targets []string
	seen := make(map[string]bool)

	for _, p := range targetPatterns {
		matches := p.FindAllString(text, -1)
		for _, m := range matches {
			lower := strings.ToLower(m)
			if !seen[lower] {
				seen[lower] = true
				targets = append(targets, m)
			}
		}
	}

	// Also check for broad scope group markers.
	for _, sg := range scopeGroupPatterns {
		if sg.pattern.MatchString(text) && !seen[sg.name] {
			seen[sg.name] = true
			targets = append(targets, sg.name)
		}
	}

	return targets
}

func findSharedSentences(sentences []classifier.Sentence, groups []sentenceGroup) []classifier.Sentence {
	// Sentences that aren't in any group are shared.
	inGroup := make(map[string]bool)
	for _, g := range groups {
		for _, s := range g.sentences {
			inGroup[s.Text] = true
		}
	}

	var shared []classifier.Sentence
	for _, s := range sentences {
		if !inGroup[s.Text] {
			shared = append(shared, s)
		}
	}
	return shared
}

func buildSubVectors(shared []classifier.Sentence, groups []sentenceGroup) []SubVector {
	subVectors := make([]SubVector, 0, len(groups))

	for _, g := range groups {
		combined := make([]classifier.Sentence, 0, len(shared)+len(g.sentences))
		combined = append(combined, shared...)
		combined = append(combined, g.sentences...)

		subVectors = append(subVectors, SubVector{
			Label:     g.label,
			Sentences: combined,
			Text:      joinSentences(combined),
		})
	}

	return subVectors
}

func joinSentences(sentences []classifier.Sentence) string {
	parts := make([]string, len(sentences))
	for i, s := range sentences {
		parts[i] = s.Text
	}
	return strings.Join(parts, " ")
}

func containsStr(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
