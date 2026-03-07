package vector

import (
	"strings"
	"testing"

	"github.com/ppiankov/vectorpad/internal/classifier"
)

func TestRenderGroupsByCategory(t *testing.T) {
	sentences := []classifier.Sentence{
		{
			Text:       "Do not rewrite unless checks pass.",
			Tag:        classifier.TagConstraint,
			LockPolicy: classifier.LockPolicyHard,
		},
		{
			Text:       "We will ship behind a flag.",
			Tag:        classifier.TagDecision,
			LockPolicy: classifier.LockPolicySoft,
		},
		{
			Text:       "Maybe we can run this in phases.",
			Tag:        classifier.TagTentative,
			LockPolicy: classifier.LockPolicyModalSpan,
			LockedSpans: []classifier.LockedSpan{
				{Start: 0, End: 5, Text: "Maybe"},
			},
		},
		{
			Text:       "Can we compact first?",
			Tag:        classifier.TagQuestion,
			LockPolicy: classifier.LockPolicyNone,
		},
		{
			Text:       "What if we try a branch?",
			Tag:        classifier.TagSpeculation,
			LockPolicy: classifier.LockPolicyNone,
		},
		{
			Text:       "This sentence documents context.",
			Tag:        classifier.TagExplanation,
			LockPolicy: classifier.LockPolicyNone,
		},
	}

	got := Render(sentences)

	expectedFragments := []string{
		"VECTOR",
		"  Constraints:",
		"    - [CONSTRAINT][LOCKED] Do not rewrite unless checks pass.",
		"  Decisions:",
		"    - [DECISION][LOCKED] We will ship behind a flag.",
		"  Tentatives:",
		"    - [TENTATIVE][LOCKED:Maybe] Maybe we can run this in phases.",
		"  Questions:",
		"    - [QUESTION] Can we compact first?",
		"  Speculations:",
		"    - [SPECULATION] What if we try a branch?",
		"  Explanations:",
		"    - [EXPLANATION] This sentence documents context.",
	}

	for _, fragment := range expectedFragments {
		if !strings.Contains(got, fragment) {
			t.Fatalf("expected output to contain %q, got:\n%s", fragment, got)
		}
	}
}

func TestRenderEmptyInput(t *testing.T) {
	got := Render(nil)

	emptyGroups := []string{
		"  Constraints:\n    - (none)",
		"  Decisions:\n    - (none)",
		"  Tentatives:\n    - (none)",
		"  Questions:\n    - (none)",
		"  Speculations:\n    - (none)",
		"  Explanations:\n    - (none)",
	}

	for _, group := range emptyGroups {
		if !strings.Contains(got, group) {
			t.Fatalf("expected empty group block %q, got:\n%s", group, got)
		}
	}
}
