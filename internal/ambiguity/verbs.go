package ambiguity

import (
	"regexp"
	"sort"
)

// English-only vague verb lexicon for ambiguity hinting.
// Non-English directives still get ratio/blast-radius detection.
var vagueVerbLexicon = []vagueVerbEntry{
	{label: "clean", pattern: regexp.MustCompile(`(?i)\bclean(?:\s+up)?\b`)},
	{label: "fix", pattern: regexp.MustCompile(`(?i)\bfix(?:es|ed|ing)?\b`)},
	{label: "align", pattern: regexp.MustCompile(`(?i)\balign(?:ed|ing|ment)?\b`)},
	{label: "standardize", pattern: regexp.MustCompile(`(?i)\bstandardi[sz]e(?:d|s|ing|ation)?\b`)},
	{label: "improve", pattern: regexp.MustCompile(`(?i)\bimprov(?:e|es|ed|ing|ement)\b`)},
	{label: "update", pattern: regexp.MustCompile(`(?i)\bupdat(?:e|es|ed|ing)\b`)},
	{label: "refactor", pattern: regexp.MustCompile(`(?i)\brefactor(?:s|ed|ing)?\b`)},
	{label: "simplify", pattern: regexp.MustCompile(`(?i)\bsimplif(?:y|ies|ied|ying|ication)\b`)},
	{label: "tidy", pattern: regexp.MustCompile(`(?i)\btid(?:y|ies|ied|ying)\b`)},
	{label: "organize", pattern: regexp.MustCompile(`(?i)\borgani[sz](?:e|es|ed|ing|ation)\b`)},
}

type vagueVerbEntry struct {
	label   string
	pattern *regexp.Regexp
}

// DetectVagueVerbs returns matched vague verbs from the English-only lexicon.
func DetectVagueVerbs(directive string) []string {
	if directive == "" {
		return nil
	}

	matches := make([]string, 0)
	for _, entry := range vagueVerbLexicon {
		if entry.pattern.MatchString(directive) {
			matches = append(matches, entry.label)
		}
	}

	if len(matches) == 0 {
		return nil
	}

	sort.Strings(matches)
	return matches
}
