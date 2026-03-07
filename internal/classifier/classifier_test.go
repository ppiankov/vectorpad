package classifier

import (
	"strings"
	"testing"
)

func TestClassifyAllTags(t *testing.T) {
	input := strings.Join([]string{
		"Do not run compaction unless token drift is below 5%.",
		"We will ship this behind a feature flag.",
		"Maybe we can stage this rollout.",
		"Can we compact before sending to the API?",
		"What if we try a canary branch first.",
		"This paragraph provides background context.",
	}, " ")

	got := Classify(input)
	if len(got) != 6 {
		t.Fatalf("expected 6 sentences, got %d", len(got))
	}

	cases := []struct {
		index      int
		wantTag    Tag
		wantPolicy LockPolicy
	}{
		{index: 0, wantTag: TagConstraint, wantPolicy: LockPolicyHard},
		{index: 1, wantTag: TagDecision, wantPolicy: LockPolicySoft},
		{index: 2, wantTag: TagTentative, wantPolicy: LockPolicyModalSpan},
		{index: 3, wantTag: TagQuestion, wantPolicy: LockPolicyNone},
		{index: 4, wantTag: TagSpeculation, wantPolicy: LockPolicyNone},
		{index: 5, wantTag: TagExplanation, wantPolicy: LockPolicyNone},
	}

	for _, tc := range cases {
		if got[tc.index].Tag != tc.wantTag {
			t.Fatalf("sentence %d: expected tag %s, got %s", tc.index, tc.wantTag, got[tc.index].Tag)
		}
		if got[tc.index].LockPolicy != tc.wantPolicy {
			t.Fatalf(
				"sentence %d: expected policy %s, got %s",
				tc.index,
				tc.wantPolicy,
				got[tc.index].LockPolicy,
			)
		}
	}
}

func TestClassifyPriorityConstraintWinsOverDecision(t *testing.T) {
	input := "We will ship this only if latency stays under 120ms."
	got := Classify(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 sentence, got %d", len(got))
	}

	if got[0].Tag != TagConstraint {
		t.Fatalf("expected constraint tag for mixed trigger sentence, got %s", got[0].Tag)
	}
	if got[0].LockPolicy != LockPolicyHard {
		t.Fatalf("expected hard lock for constraint, got %s", got[0].LockPolicy)
	}
}

func TestClassifyNoTriggersFallsBackToExplanation(t *testing.T) {
	input := "The cache stores serialized vectors for replay."
	got := Classify(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 sentence, got %d", len(got))
	}
	if got[0].Tag != TagExplanation {
		t.Fatalf("expected explanation tag, got %s", got[0].Tag)
	}
}

func TestClassifyTentativeLocksOnlyModalSpan(t *testing.T) {
	input := "I think we can finish this by Friday."
	got := Classify(input)
	if len(got) != 1 {
		t.Fatalf("expected 1 sentence, got %d", len(got))
	}

	sentence := got[0]
	if sentence.Tag != TagTentative {
		t.Fatalf("expected tentative tag, got %s", sentence.Tag)
	}
	if sentence.LockPolicy != LockPolicyModalSpan {
		t.Fatalf("expected modal span lock policy, got %s", sentence.LockPolicy)
	}
	if len(sentence.LockedSpans) == 0 {
		t.Fatal("expected at least one locked modal span")
	}
	if sentence.LockedSpans[0].Text != "I think" {
		t.Fatalf("expected first locked span to be 'I think', got %q", sentence.LockedSpans[0].Text)
	}
	if sentence.LockedSpans[0].End-sentence.LockedSpans[0].Start >= len(sentence.Text) {
		t.Fatal("tentative lock span should not cover the entire sentence")
	}
}

func TestClassifyEmptyInput(t *testing.T) {
	got := Classify(" \n\t ")
	if len(got) != 0 {
		t.Fatalf("expected no sentences for empty input, got %d", len(got))
	}
}
