package ambiguity

import (
	"strings"
	"testing"
)

func TestNudgeProtocolTriggersWhenAmbiguousAndNoPreservationConstraints(t *testing.T) {
	t.Parallel()

	result := Analyze("clean up readmes", Scope{Repos: 4, Files: 12})
	if !result.Warning.Active {
		t.Fatal("expected ambiguous warning to be active")
	}
	if result.HasPreservationConstraints {
		t.Fatal("expected no preservation constraints in directive")
	}

	nudges := SelectNudges(result)
	if len(nudges) != 3 {
		t.Fatalf("expected 3 nudges, got %d", len(nudges))
	}
	for _, nudge := range nudges {
		if !nudge.Dismissable {
			t.Fatalf("expected nudge %s to be dismissable", nudge.Type)
		}
	}
}

func TestPreservationConstraintNudgeShownWhenScopeGreaterThanOne(t *testing.T) {
	t.Parallel()

	result := Analyze("update docs", Scope{Repos: 1, Files: 6})
	if !result.Warning.Active {
		t.Fatal("expected ambiguous warning to be active")
	}

	nudges := SelectNudges(result)
	if !hasNudge(nudges, NudgePreservationConstraint) {
		t.Fatal("expected preservation constraint nudge")
	}
}

func TestPreservationConstraintNudgeSuppressedWhenConstraintExists(t *testing.T) {
	t.Parallel()

	result := Analyze("clean up READMEs but keep architecture sections unchanged", Scope{Repos: 4, Files: 30})
	if !result.Warning.Active {
		t.Fatal("expected ambiguous warning to be active")
	}
	if !result.HasPreservationConstraints {
		t.Fatal("expected preservation constraints to be detected")
	}

	nudges := SelectNudges(result)
	if len(nudges) != 0 {
		t.Fatalf("expected no nudges when preservation constraints exist, got %d", len(nudges))
	}
}

func TestScopeConsistencyNudgeShownWhenReposGreaterThanOne(t *testing.T) {
	t.Parallel()

	result := Analyze("update docs", Scope{Repos: 2, Files: 6})
	nudges := SelectNudges(result)
	if !hasNudge(nudges, NudgeScopeConsistency) {
		t.Fatal("expected scope consistency nudge for multi-repo scope")
	}
}

func TestReferenceExampleNudgeShownWhenReposGreaterThanThree(t *testing.T) {
	t.Parallel()

	result := Analyze("update docs", Scope{Repos: 4, Files: 6})
	nudges := SelectNudges(result)
	if !hasNudge(nudges, NudgeReferenceExample) {
		t.Fatal("expected reference example nudge when repos > 3")
	}
}

func TestApplyNudgeResponsesAppendsAnsweredConstraints(t *testing.T) {
	t.Parallel()

	directive := "clean up READMEs"
	output := ApplyNudgeResponses(directive, []NudgeResponse{
		{Type: NudgePreservationConstraint, Answer: "Do not change voice or architecture sections."},
		{Type: NudgeScopeConsistency, Answer: "Apply same transformation across all repos."},
		{Type: NudgeReferenceExample, Dismissed: true},
	})

	if output == directive {
		t.Fatal("expected answered nudges to append launch constraints")
	}

	requiredFragments := []string{
		"Launch constraints:",
		"Preserve: Do not change voice or architecture sections.",
		"Scope rule: Apply same transformation across all repos.",
	}
	for _, fragment := range requiredFragments {
		if !strings.Contains(output, fragment) {
			t.Fatalf("expected output to contain %q, got:\n%s", fragment, output)
		}
	}

	if strings.Contains(output, "Reference example:") {
		t.Fatal("dismissed nudge response should not be appended")
	}
}

func TestApplyNudgeResponsesAllowsDismissAndProceed(t *testing.T) {
	t.Parallel()

	directive := "clean up READMEs"
	output := ApplyNudgeResponses(directive, []NudgeResponse{
		{Type: NudgePreservationConstraint, Dismissed: true},
		{Type: NudgeScopeConsistency, Dismissed: true},
		{Type: NudgeReferenceExample, Dismissed: true},
	})

	if output != directive {
		t.Fatalf("expected unchanged directive when all nudges dismissed, got %q", output)
	}
}

func hasNudge(nudges []Nudge, target NudgeType) bool {
	for _, nudge := range nudges {
		if nudge.Type == target {
			return true
		}
	}
	return false
}
