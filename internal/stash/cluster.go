package stash

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"
)

const (
	clusterSimilarityThreshold   = 0.40
	uniquenessLowThreshold       = 0.75
	uniquenessMediumThreshold    = 0.40
	stackLabelTokenLimit         = 2
	sharedTokenOccurrenceMinimum = 2
)

var (
	tokenPattern = regexp.MustCompile(`[a-z0-9]+`)
	stopWords    = map[string]struct{}{
		"a": {}, "an": {}, "and": {}, "as": {}, "at": {}, "be": {}, "by": {}, "can": {}, "for": {},
		"from": {}, "i": {}, "if": {}, "in": {}, "into": {}, "is": {}, "it": {}, "not": {}, "of": {},
		"on": {}, "or": {}, "our": {}, "that": {}, "the": {}, "this": {}, "to": {}, "we": {}, "will": {},
		"with": {}, "you": {}, "your": {},
	}
)

func Cluster(stacks []Stack, now time.Time) []Stack {
	items := flattenItems(stacks)
	return ClusterItems(items, now)
}

func ClusterItems(items []Item, now time.Time) []Stack {
	if len(items) == 0 {
		return nil
	}

	sortedItems := append([]Item(nil), items...)
	sortItems(sortedItems)

	tokenSets := make([]map[string]struct{}, len(sortedItems))
	for index, item := range sortedItems {
		tokenSets[index] = Tokenize(item.Text)
	}

	parent := make([]int, len(sortedItems))
	for index := range parent {
		parent[index] = index
	}

	for left := 0; left < len(sortedItems); left++ {
		for right := left + 1; right < len(sortedItems); right++ {
			similarity := JaccardSimilarity(tokenSets[left], tokenSets[right])
			if similarity > clusterSimilarityThreshold {
				union(parent, left, right)
			}
		}
	}

	components := make(map[int][]int)
	for index := range sortedItems {
		root := find(parent, index)
		components[root] = append(components[root], index)
	}

	componentIndices := make([][]int, 0, len(components))
	for _, indices := range components {
		sort.Ints(indices)
		componentIndices = append(componentIndices, indices)
	}
	sort.Slice(componentIndices, func(left, right int) bool {
		leftItem := sortedItems[componentIndices[left][0]]
		rightItem := sortedItems[componentIndices[right][0]]
		return compareItems(leftItem, rightItem) < 0
	})

	clustered := make([]Stack, 0, len(componentIndices)+1)
	unclusteredItems := make([]Item, 0)
	for _, indices := range componentIndices {
		if len(indices) == 1 {
			item := sortedItems[indices[0]]
			item.Uniqueness = UniquenessHigh
			unclusteredItems = append(unclusteredItems, item)
			continue
		}

		clusterItems := make([]Item, 0, len(indices))
		for _, index := range indices {
			clusterItems = append(clusterItems, sortedItems[index])
		}
		sortItems(clusterItems)
		clusterItems = applyUniquenessScores(clusterItems)

		label := buildStackLabel(indices, tokenSets)
		created, updated := stackBounds(clusterItems, now)
		clustered = append(clustered, Stack{
			Label:   label,
			Created: created,
			Updated: updated,
			Items:   clusterItems,
		})
	}

	sort.Slice(clustered, func(left, right int) bool {
		if clustered[left].Label == clustered[right].Label {
			return clustered[left].Created.Before(clustered[right].Created)
		}
		return clustered[left].Label < clustered[right].Label
	})
	applyStackIDs(clustered)

	if len(unclusteredItems) > 0 {
		sortItems(unclusteredItems)
		created, updated := stackBounds(unclusteredItems, now)
		clustered = append(clustered, Stack{
			ID:      UnclusteredStackID,
			Label:   UnclusteredStackLabel,
			Created: created,
			Updated: updated,
			Items:   unclusteredItems,
		})
	}

	return clustered
}

func Tokenize(text string) map[string]struct{} {
	matches := tokenPattern.FindAllString(strings.ToLower(text), -1)
	tokens := make(map[string]struct{}, len(matches))
	for _, token := range matches {
		if _, blocked := stopWords[token]; blocked {
			continue
		}
		tokens[token] = struct{}{}
	}
	return tokens
}

func JaccardSimilarity(left map[string]struct{}, right map[string]struct{}) float64 {
	if len(left) == 0 && len(right) == 0 {
		return 0
	}

	intersection := 0
	union := len(left)
	for token := range right {
		if _, ok := left[token]; ok {
			intersection++
			continue
		}
		union++
	}

	if union == 0 {
		return 0
	}
	return float64(intersection) / float64(union)
}

func flattenItems(stacks []Stack) []Item {
	total := 0
	for _, stack := range stacks {
		total += len(stack.Items)
	}

	items := make([]Item, 0, total)
	for _, stack := range stacks {
		items = append(items, stack.Items...)
	}
	return items
}

func find(parent []int, index int) int {
	if parent[index] == index {
		return index
	}
	parent[index] = find(parent, parent[index])
	return parent[index]
}

func union(parent []int, left int, right int) {
	leftRoot := find(parent, left)
	rightRoot := find(parent, right)
	if leftRoot == rightRoot {
		return
	}
	parent[rightRoot] = leftRoot
}

func buildStackLabel(indices []int, tokenSets []map[string]struct{}) string {
	if len(indices) == 0 {
		return "Cluster"
	}

	tokenCount := map[string]int{}
	for _, index := range indices {
		for token := range tokenSets[index] {
			tokenCount[token]++
		}
	}

	type weightedToken struct {
		token string
		count int
	}

	weighted := make([]weightedToken, 0, len(tokenCount))
	for token, count := range tokenCount {
		if count >= sharedTokenOccurrenceMinimum {
			weighted = append(weighted, weightedToken{token: token, count: count})
		}
	}
	if len(weighted) == 0 {
		for token, count := range tokenCount {
			weighted = append(weighted, weightedToken{token: token, count: count})
		}
	}

	sort.Slice(weighted, func(left, right int) bool {
		if weighted[left].count == weighted[right].count {
			return weighted[left].token < weighted[right].token
		}
		return weighted[left].count > weighted[right].count
	})

	if len(weighted) == 0 {
		return "Cluster"
	}

	parts := make([]string, 0, stackLabelTokenLimit)
	for index := 0; index < len(weighted) && index < stackLabelTokenLimit; index++ {
		parts = append(parts, titleToken(weighted[index].token))
	}
	if len(parts) == 0 {
		return "Cluster"
	}

	return strings.Join(parts, " ")
}

func titleToken(token string) string {
	if token == "" {
		return token
	}
	if len(token) == 1 {
		return strings.ToUpper(token)
	}
	return strings.ToUpper(token[:1]) + token[1:]
}

func applyStackIDs(stacks []Stack) {
	seen := map[string]int{}
	for index := range stacks {
		base := slugify(stacks[index].Label)
		if base == "" {
			base = "stack"
		}
		count := seen[base]
		if count == 0 {
			stacks[index].ID = base
		} else {
			stacks[index].ID = fmt.Sprintf("%s-%d", base, count+1)
		}
		seen[base] = count + 1
	}
}

func slugify(label string) string {
	tokens := tokenPattern.FindAllString(strings.ToLower(label), -1)
	return strings.Join(tokens, "-")
}

func stackBounds(items []Item, now time.Time) (time.Time, time.Time) {
	if len(items) == 0 {
		return now, now
	}

	created := items[0].Created
	updated := items[0].Created
	for _, item := range items[1:] {
		if item.Created.Before(created) {
			created = item.Created
		}
		if item.Created.After(updated) {
			updated = item.Created
		}
	}
	return created, updated
}

func applyUniquenessScores(items []Item) []Item {
	if len(items) <= 1 {
		for index := range items {
			items[index].Uniqueness = UniquenessHigh
		}
		return items
	}

	tokenSets := make([]map[string]struct{}, len(items))
	for index, item := range items {
		tokenSets[index] = Tokenize(item.Text)
	}

	for index := range items {
		maxSimilarity := 0.0
		for other := range items {
			if index == other {
				continue
			}
			similarity := JaccardSimilarity(tokenSets[index], tokenSets[other])
			if similarity > maxSimilarity {
				maxSimilarity = similarity
			}
		}
		items[index].Uniqueness = uniquenessFromSimilarity(maxSimilarity)
	}

	return items
}

func uniquenessFromSimilarity(similarity float64) Uniqueness {
	switch {
	case similarity >= uniquenessLowThreshold:
		return UniquenessLow
	case similarity >= uniquenessMediumThreshold:
		return UniquenessMedium
	default:
		return UniquenessHigh
	}
}

func sortItems(items []Item) {
	sort.Slice(items, func(left, right int) bool {
		return compareItems(items[left], items[right]) < 0
	})
}

func compareItems(left Item, right Item) int {
	switch {
	case left.Created.Before(right.Created):
		return -1
	case left.Created.After(right.Created):
		return 1
	case left.ID < right.ID:
		return -1
	case left.ID > right.ID:
		return 1
	case left.Text < right.Text:
		return -1
	case left.Text > right.Text:
		return 1
	default:
		return 0
	}
}
