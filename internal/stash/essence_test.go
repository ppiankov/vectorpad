package stash

import (
	"strings"
	"testing"
	"time"
)

func TestExtractEssenceEmpty(t *testing.T) {
	result := ExtractEssence(Stack{})
	if result != "" {
		t.Errorf("expected empty for empty stack, got %q", result)
	}
}

func TestExtractEssenceSingleItem(t *testing.T) {
	stack := Stack{
		Items: []Item{{Text: "context dilution attack vector"}},
	}
	result := ExtractEssence(stack)
	if result != "context dilution attack vector" {
		t.Errorf("expected single item text, got %q", result)
	}
}

func TestExtractEssenceMultipleItems(t *testing.T) {
	now := time.Now()
	stack := Stack{
		Label: "Token Economics",
		Items: []Item{
			{Text: "cache read amplification reduces token cost", Created: now},
			{Text: "prompt compression saves token budget", Created: now},
			{Text: "batch requests lower token overhead", Created: now},
		},
	}
	result := ExtractEssence(stack)

	// Should start with stack label.
	if !strings.HasPrefix(result, "Token Economics: ") {
		t.Errorf("expected label prefix, got %q", result)
	}

	// Should contain specifics after the colon.
	parts := strings.SplitN(result, ": ", 2)
	if len(parts) < 2 || parts[1] == "" {
		t.Errorf("expected specifics after colon, got %q", result)
	}
}

func TestExtractEssenceDeterministic(t *testing.T) {
	now := time.Now()
	stack := Stack{
		Label: "Testing",
		Items: []Item{
			{Text: "unit tests for classifier", Created: now},
			{Text: "integration tests for pipeline", Created: now},
		},
	}

	result1 := ExtractEssence(stack)
	result2 := ExtractEssence(stack)
	if result1 != result2 {
		t.Errorf("not deterministic: %q vs %q", result1, result2)
	}
}

func TestExtractEssenceUnclusteredLabel(t *testing.T) {
	now := time.Now()
	stack := Stack{
		Label: UnclusteredStackLabel,
		Items: []Item{
			{Text: "alpha bravo charlie", Created: now},
			{Text: "alpha delta echo", Created: now},
		},
	}
	result := ExtractEssence(stack)

	// Should not use "Unclustered" as prefix.
	if strings.HasPrefix(result, UnclusteredStackLabel) {
		t.Errorf("should not use unclustered label, got %q", result)
	}
}
