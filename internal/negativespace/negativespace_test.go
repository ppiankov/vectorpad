package negativespace

import (
	"testing"
)

func TestAnalyzeCleanDirective(t *testing.T) {
	// Well-constrained directive should have no gaps.
	text := "Update all READMEs to standardize badge format. Preserve philosophy sections, " +
		"keep existing voice. Review each repo individually before applying. " +
		"Expected result: badges are consistent, content unchanged. " +
		"Revert with git if anything looks wrong. Skip archived repos."
	result := Analyze(text)
	if !result.Clean() {
		for _, g := range result.Gaps {
			t.Errorf("unexpected gap: %s — %s (signal: %s)", g.Class, g.Description, g.Signal)
		}
	}
}

func TestAnalyzeREADMEMassacre(t *testing.T) {
	// The directive that started it all: 5 words, 18 repos, zero constraints.
	text := "clean up READMEs for alignment"
	result := Analyze(text)
	if result.Clean() {
		t.Fatal("expected gaps for the README Massacre directive")
	}

	// Should detect at least: preservation, success, identity gaps.
	classes := gapClasses(result)
	for _, expected := range []GapClass{GapPreservation, GapSuccessCrit, GapIdentity} {
		if !contains(classes, expected) {
			t.Errorf("expected gap class %s, got: %v", expected, classes)
		}
	}
}

func TestPreservationGap(t *testing.T) {
	// Destructive verb without protection → gap.
	result := Analyze("delete old config files from all repos")
	if !hasGap(result, GapPreservation) {
		t.Error("expected preservation gap for destructive verb without protection")
	}

	// Add preservation → no gap.
	result = Analyze("delete old config files from all repos but keep .env.example")
	if hasGap(result, GapPreservation) {
		t.Error("should not have preservation gap when 'keep' is present")
	}
}

func TestSuccessCriteriaGap(t *testing.T) {
	// Action without outcome → gap.
	result := Analyze("refactor the authentication module")
	if !hasGap(result, GapSuccessCrit) {
		t.Error("expected success criteria gap")
	}

	// With success criteria → no gap.
	result = Analyze("refactor the authentication module. Expected result: all tests pass and coverage stays above 85%")
	if hasGap(result, GapSuccessCrit) {
		t.Error("should not have success criteria gap when expected result is stated")
	}
}

func TestReviewProcessGap(t *testing.T) {
	// Multi-target scope without review → gap.
	result := Analyze("update all repos to use new CI template")
	if !hasGap(result, GapReviewProcess) {
		t.Error("expected review process gap for multi-target without review")
	}

	// With review → no gap.
	result = Analyze("update all repos to use new CI template. Review each repo before applying")
	if hasGap(result, GapReviewProcess) {
		t.Error("should not have review gap when review is mentioned")
	}
}

func TestRollbackGap(t *testing.T) {
	// Destructive + scope without rollback → gap.
	result := Analyze("remove deprecated APIs from all services")
	if !hasGap(result, GapRollback) {
		t.Error("expected rollback gap for destructive scope without undo plan")
	}

	// With backup → no gap.
	result = Analyze("remove deprecated APIs from all services. Backup each service before changes")
	if hasGap(result, GapRollback) {
		t.Error("should not have rollback gap when backup is mentioned")
	}
}

func TestScopeBoundaryGap(t *testing.T) {
	// Broad quantifier without exclusions → gap.
	result := Analyze("update every file in the project")
	if !hasGap(result, GapScopeBoundary) {
		t.Error("expected scope boundary gap for 'every' without exclusions")
	}

	// With exclusion → no gap.
	result = Analyze("update every file in the project except vendor/")
	if hasGap(result, GapScopeBoundary) {
		t.Error("should not have scope boundary gap when exclusion is present")
	}
}

func TestIdentityGap(t *testing.T) {
	// Content-touching verb without voice/style → gap.
	result := Analyze("rewrite the project documentation")
	if !hasGap(result, GapIdentity) {
		t.Error("expected identity gap for rewrite without voice constraint")
	}

	// With voice constraint → no gap.
	result = Analyze("rewrite the project documentation. Keep the existing voice and tone")
	if hasGap(result, GapIdentity) {
		t.Error("should not have identity gap when voice is mentioned")
	}
}

func TestNoActionNoGaps(t *testing.T) {
	// Purely descriptive text should have no gaps.
	result := Analyze("the system uses JWT tokens for authentication")
	if !result.Clean() {
		t.Errorf("expected no gaps for descriptive text, got: %v", gapClasses(result))
	}
}

func TestActionSignalCount(t *testing.T) {
	result := Analyze("clean up and refactor the code, then update the docs")
	if result.ActionSignals < 3 {
		t.Errorf("expected at least 3 action signals, got %d", result.ActionSignals)
	}
}

func TestScopeSignalCount(t *testing.T) {
	result := Analyze("apply this across all repos and all files")
	if result.ScopeSignals < 2 {
		t.Errorf("expected at least 2 scope signals, got %d", result.ScopeSignals)
	}
}

func TestDeterministic(t *testing.T) {
	text := "clean up READMEs for alignment"
	r1 := Analyze(text)
	r2 := Analyze(text)
	if len(r1.Gaps) != len(r2.Gaps) {
		t.Fatal("non-deterministic gap count")
	}
	for i := range r1.Gaps {
		if r1.Gaps[i].Class != r2.Gaps[i].Class {
			t.Errorf("non-deterministic gap order at %d: %s vs %s", i, r1.Gaps[i].Class, r2.Gaps[i].Class)
		}
	}
}

// --- helpers ---

func gapClasses(r Result) []GapClass {
	var classes []GapClass
	for _, g := range r.Gaps {
		classes = append(classes, g.Class)
	}
	return classes
}

func hasGap(r Result, class GapClass) bool {
	for _, g := range r.Gaps {
		if g.Class == class {
			return true
		}
	}
	return false
}

func contains(classes []GapClass, target GapClass) bool {
	for _, c := range classes {
		if c == target {
			return true
		}
	}
	return false
}
