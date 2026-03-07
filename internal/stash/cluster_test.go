package stash

import (
	"math"
	"testing"
	"time"
)

func TestJaccardSimilarity(t *testing.T) {
	left := map[string]struct{}{"token": {}, "economics": {}, "cache": {}}
	right := map[string]struct{}{"token": {}, "economics": {}, "vector": {}}

	got := JaccardSimilarity(left, right)
	want := 0.5
	if math.Abs(got-want) > 0.0001 {
		t.Fatalf("expected %.2f similarity, got %.4f", want, got)
	}
}

func TestClusterItemsGroupsRelatedIdeas(t *testing.T) {
	now := time.Date(2026, time.March, 7, 12, 0, 0, 0, time.UTC)
	items := []Item{
		{
			ID:      "item-1",
			Text:    "token economics cache amplification metric",
			Created: now.Add(-2 * time.Hour),
			Source:  SourceCLI,
		},
		{
			ID:      "item-2",
			Text:    "cache amplification for token spend economics",
			Created: now.Add(-1 * time.Hour),
			Source:  SourceCLI,
		},
		{
			ID:      "item-3",
			Text:    "terminal shortcuts for stash navigation",
			Created: now,
			Source:  SourceCLI,
		},
	}

	stacks := ClusterItems(items, now)
	if len(stacks) != 2 {
		t.Fatalf("expected 2 stacks, got %d", len(stacks))
	}

	var clustered Stack
	var unclustered Stack
	for _, stack := range stacks {
		if stack.ID == UnclusteredStackID {
			unclustered = stack
			continue
		}
		clustered = stack
	}

	if len(clustered.Items) != 2 {
		t.Fatalf("expected related stack to contain 2 items, got %d", len(clustered.Items))
	}
	for _, item := range clustered.Items {
		if item.Uniqueness != UniquenessMedium {
			t.Fatalf("expected clustered item uniqueness=medium, got %s", item.Uniqueness)
		}
	}

	if unclustered.ID != UnclusteredStackID {
		t.Fatalf("expected %q stack for unrelated item", UnclusteredStackID)
	}
	if len(unclustered.Items) != 1 {
		t.Fatalf("expected unclustered stack to contain 1 item, got %d", len(unclustered.Items))
	}
	if unclustered.Items[0].ID != "item-3" {
		t.Fatalf("expected item-3 in unclustered stack, got %q", unclustered.Items[0].ID)
	}
}

func TestClusterItemsThresholdRequiresGreaterThanFortyPercent(t *testing.T) {
	now := time.Date(2026, time.March, 7, 12, 0, 0, 0, time.UTC)
	items := []Item{
		{ID: "item-1", Text: "alpha beta gamma", Created: now.Add(-2 * time.Hour), Source: SourceCLI},
		{ID: "item-2", Text: "alpha beta delta epsilon", Created: now, Source: SourceCLI},
	}

	stacks := ClusterItems(items, now)
	if len(stacks) != 1 {
		t.Fatalf("expected only unclustered stack, got %d stacks", len(stacks))
	}
	if stacks[0].ID != UnclusteredStackID {
		t.Fatalf("expected %q stack, got %q", UnclusteredStackID, stacks[0].ID)
	}
	if len(stacks[0].Items) != 2 {
		t.Fatalf("expected 2 items in unclustered stack, got %d", len(stacks[0].Items))
	}
}
