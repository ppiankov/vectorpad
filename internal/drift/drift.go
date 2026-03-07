package drift

import (
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// Axis identifies the meaning dimension checked for drift.
type Axis string

const (
	AxisModality    Axis = "modality"
	AxisNegation    Axis = "negation"
	AxisNumeric     Axis = "numeric"
	AxisScope       Axis = "scope"
	AxisConditional Axis = "conditional"
	AxisCommitment  Axis = "commitment"
)

var (
	modalityPattern = regexp.MustCompile(`(?i)\b(can|must|should|will|might)\b`)

	negationPattern = regexp.MustCompile(
		`(?i)\b(cannot|can\s+not|can't|not|never|no|unless|except)\b`,
	)

	numericPattern = regexp.MustCompile(`(?i)\b([-+]?\d[\d,]*(?:\.\d+)?)(?:\s*(%|ms|s|m|h|d))?\b`)

	scopePattern = regexp.MustCompile(
		`(?i)\b(at\s+least|at\s+most|only|any|all|every|none|exactly)\b`,
	)

	conditionalPattern = regexp.MustCompile(
		`(?i)\b(if|unless|except|provided(?:\s+that)?|assuming)\b`,
	)

	commitmentPattern = regexp.MustCompile(
		`(?i)\b(i\s+think|i\s+believe|maybe|probably|likely|perhaps|possibly|seems?|might|could)\b`,
	)
)

var modalStrength = map[string]int{
	"might":  1,
	"can":    2,
	"should": 3,
	"will":   4,
	"must":   5,
}

// Result captures whether a rewrite is safe to auto-apply and why not.
type Result struct {
	Allowed bool        `json:"allowed"`
	Drifts  []AxisDrift `json:"drifts,omitempty"`
}

// AxisDrift describes what changed for one meaning axis.
type AxisDrift struct {
	Axis    Axis          `json:"axis"`
	Added   []string      `json:"added,omitempty"`
	Removed []string      `json:"removed,omitempty"`
	Changed []TokenChange `json:"changed,omitempty"`
}

// TokenChange describes a replacement-level mutation.
type TokenChange struct {
	From string `json:"from"`
	To   string `json:"to"`
	Kind string `json:"kind,omitempty"`
}

// Detect reports drift across all guardrail axes.
func Detect(original string, rewritten string) Result {
	original = strings.TrimSpace(original)
	rewritten = strings.TrimSpace(rewritten)

	checkers := []func(string, string) (AxisDrift, bool){
		detectModalityDrift,
		detectNegationDrift,
		detectNumericDrift,
		detectScopeDrift,
		detectConditionalDrift,
		detectCommitmentDrift,
	}

	drifts := make([]AxisDrift, 0, len(checkers))
	for _, checker := range checkers {
		drift, found := checker(original, rewritten)
		if found {
			drifts = append(drifts, drift)
		}
	}

	return Result{
		Allowed: len(drifts) == 0,
		Drifts:  drifts,
	}
}

func detectModalityDrift(original string, rewritten string) (AxisDrift, bool) {
	delta := diffTokens(
		extractByPattern(original, modalityPattern, normalizePhrase),
		extractByPattern(rewritten, modalityPattern, normalizePhrase),
		sortByModalityStrength,
		func(from string, to string) string {
			fromStrength := modalStrength[from]
			toStrength := modalStrength[to]

			switch {
			case toStrength > fromStrength:
				return "upgrade"
			case toStrength < fromStrength:
				return "downgrade"
			default:
				return "change"
			}
		},
	)

	return buildAxisDrift(AxisModality, delta)
}

func detectNegationDrift(original string, rewritten string) (AxisDrift, bool) {
	originalTokens := extractByPattern(original, negationPattern, normalizeNegation)
	rewrittenTokens := extractByPattern(rewritten, negationPattern, normalizeNegation)

	delta := diffTokens(originalTokens, rewrittenTokens, sort.Strings, func(string, string) string {
		return "change"
	})

	originalNegative := len(originalTokens) > 0
	rewrittenNegative := len(rewrittenTokens) > 0
	if originalNegative != rewrittenNegative {
		fromPolarity := "positive"
		if originalNegative {
			fromPolarity = "negative"
		}

		toPolarity := "positive"
		if rewrittenNegative {
			toPolarity = "negative"
		}

		delta.changed = append(delta.changed, TokenChange{
			From: fromPolarity,
			To:   toPolarity,
			Kind: "polarity_flip",
		})
	}

	return buildAxisDrift(AxisNegation, delta)
}

func detectNumericDrift(original string, rewritten string) (AxisDrift, bool) {
	delta := diffTokens(
		extractNumericTokens(original),
		extractNumericTokens(rewritten),
		sort.Strings,
		func(string, string) string {
			return "change"
		},
	)

	return buildAxisDrift(AxisNumeric, delta)
}

func detectScopeDrift(original string, rewritten string) (AxisDrift, bool) {
	delta := diffTokens(
		extractByPattern(original, scopePattern, normalizePhrase),
		extractByPattern(rewritten, scopePattern, normalizePhrase),
		sort.Strings,
		func(string, string) string {
			return "change"
		},
	)

	return buildAxisDrift(AxisScope, delta)
}

func detectConditionalDrift(original string, rewritten string) (AxisDrift, bool) {
	delta := diffTokens(
		extractByPattern(original, conditionalPattern, normalizeConditional),
		extractByPattern(rewritten, conditionalPattern, normalizeConditional),
		sort.Strings,
		func(string, string) string {
			return "change"
		},
	)

	return buildAxisDrift(AxisConditional, delta)
}

func detectCommitmentDrift(original string, rewritten string) (AxisDrift, bool) {
	delta := diffTokens(
		extractByPattern(original, commitmentPattern, normalizeCommitment),
		extractByPattern(rewritten, commitmentPattern, normalizeCommitment),
		sort.Strings,
		func(string, string) string {
			return "change"
		},
	)

	return buildAxisDrift(AxisCommitment, delta)
}

type tokenDelta struct {
	added   []string
	removed []string
	changed []TokenChange
}

func diffTokens(
	original []string,
	rewritten []string,
	sortTokens func([]string),
	changeKind func(from string, to string) string,
) tokenDelta {
	originalCounts := countTokens(original)
	rewrittenCounts := countTokens(rewritten)

	added := make([]string, 0)
	for token, rewrittenCount := range rewrittenCounts {
		if rewrittenCount > originalCounts[token] {
			added = appendRepeated(added, token, rewrittenCount-originalCounts[token])
		}
	}

	removed := make([]string, 0)
	for token, originalCount := range originalCounts {
		if originalCount > rewrittenCounts[token] {
			removed = appendRepeated(removed, token, originalCount-rewrittenCounts[token])
		}
	}

	sortTokens(added)
	sortTokens(removed)

	pairCount := min(len(added), len(removed))
	changes := make([]TokenChange, 0, pairCount)
	for index := range pairCount {
		change := TokenChange{From: removed[index], To: added[index]}
		if changeKind != nil {
			change.Kind = changeKind(change.From, change.To)
		}
		changes = append(changes, change)
	}

	if pairCount > 0 {
		added = added[pairCount:]
		removed = removed[pairCount:]
	}

	return tokenDelta{
		added:   added,
		removed: removed,
		changed: changes,
	}
}

func buildAxisDrift(axis Axis, delta tokenDelta) (AxisDrift, bool) {
	if len(delta.added) == 0 && len(delta.removed) == 0 && len(delta.changed) == 0 {
		return AxisDrift{}, false
	}

	return AxisDrift{
		Axis:    axis,
		Added:   delta.added,
		Removed: delta.removed,
		Changed: delta.changed,
	}, true
}

func extractByPattern(text string, pattern *regexp.Regexp, normalize func(string) string) []string {
	matches := pattern.FindAllString(text, -1)
	tokens := make([]string, 0, len(matches))
	for _, match := range matches {
		token := normalize(match)
		if token == "" {
			continue
		}
		tokens = append(tokens, token)
	}

	return tokens
}

func extractNumericTokens(text string) []string {
	matches := numericPattern.FindAllStringSubmatch(text, -1)
	tokens := make([]string, 0, len(matches))

	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		number := canonicalizeNumber(match[1])
		if number == "" {
			continue
		}

		unit := ""
		if len(match) > 2 {
			unit = strings.ToLower(strings.TrimSpace(match[2]))
		}

		tokens = append(tokens, number+unit)
	}

	return tokens
}

func canonicalizeNumber(raw string) string {
	clean := strings.TrimSpace(strings.ReplaceAll(raw, ",", ""))
	if clean == "" {
		return ""
	}

	parsed, err := strconv.ParseFloat(clean, 64)
	if err != nil {
		return strings.TrimLeft(clean, "+")
	}

	return strconv.FormatFloat(parsed, 'f', -1, 64)
}

func normalizePhrase(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(value)), " ")
}

func normalizeNegation(value string) string {
	token := normalizePhrase(value)
	switch token {
	case "can not", "can't":
		return "cannot"
	default:
		return token
	}
}

func normalizeConditional(value string) string {
	token := normalizePhrase(value)
	if token == "provided that" {
		return "provided"
	}

	return token
}

func normalizeCommitment(value string) string {
	token := normalizePhrase(value)
	if token == "seem" {
		return "seems"
	}

	return token
}

func countTokens(tokens []string) map[string]int {
	counts := make(map[string]int, len(tokens))
	for _, token := range tokens {
		counts[token]++
	}

	return counts
}

func appendRepeated(tokens []string, value string, count int) []string {
	for range count {
		tokens = append(tokens, value)
	}

	return tokens
}

func sortByModalityStrength(tokens []string) {
	sort.Slice(tokens, func(i int, j int) bool {
		left := modalStrength[tokens[i]]
		right := modalStrength[tokens[j]]
		if left == right {
			return tokens[i] < tokens[j]
		}
		return left < right
	})
}
