package stash

import (
	"sort"
	"strings"
)

const (
	essenceSharedThreshold = 0.5 // token must appear in >50% of items
	essenceMaxSpecifics    = 5   // max unique specifics per item to include
)

// ExtractEssence collapses a stack into a single summary sentence.
// Deterministic: same input always produces same output.
func ExtractEssence(stack Stack) string {
	if len(stack.Items) == 0 {
		return ""
	}
	if len(stack.Items) == 1 {
		return stack.Items[0].Text
	}

	// Tokenize all items.
	tokenSets := make([]map[string]struct{}, len(stack.Items))
	for i, item := range stack.Items {
		tokenSets[i] = Tokenize(item.Text)
	}

	// Find shared tokens (appear in >50% of items).
	tokenCount := map[string]int{}
	for _, ts := range tokenSets {
		for token := range ts {
			tokenCount[token]++
		}
	}

	threshold := int(float64(len(stack.Items))*essenceSharedThreshold) + 1
	if threshold > len(stack.Items) {
		threshold = len(stack.Items)
	}

	shared := map[string]struct{}{}
	for token, count := range tokenCount {
		if count >= threshold {
			shared[token] = struct{}{}
		}
	}

	// Collect unique specifics per item (tokens not in shared set).
	var specifics []string
	seen := map[string]struct{}{}
	for _, ts := range tokenSets {
		for token := range ts {
			if _, isShared := shared[token]; isShared {
				continue
			}
			if _, already := seen[token]; already {
				continue
			}
			seen[token] = struct{}{}
			specifics = append(specifics, token)
		}
	}
	sort.Strings(specifics)
	if len(specifics) > essenceMaxSpecifics {
		specifics = specifics[:essenceMaxSpecifics]
	}

	// Build essence.
	theme := stack.Label
	if theme == "" || theme == UnclusteredStackLabel {
		// Use most frequent shared tokens as theme.
		theme = buildThemeFromShared(shared)
	}

	if len(specifics) == 0 {
		return theme
	}

	return theme + ": " + strings.Join(specifics, ", ")
}

func buildThemeFromShared(shared map[string]struct{}) string {
	tokens := make([]string, 0, len(shared))
	for t := range shared {
		tokens = append(tokens, t)
	}
	sort.Strings(tokens)

	if len(tokens) == 0 {
		return "Summary"
	}
	if len(tokens) > 3 {
		tokens = tokens[:3]
	}

	// Title case the first token.
	if len(tokens[0]) > 0 {
		tokens[0] = strings.ToUpper(tokens[0][:1]) + tokens[0][1:]
	}
	return strings.Join(tokens, " ")
}
