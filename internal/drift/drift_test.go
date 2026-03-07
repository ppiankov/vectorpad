package drift

import (
	"slices"
	"testing"
)

func TestDetectNoFalsePositivesOnSafeSubstitutions(t *testing.T) {
	t.Parallel()

	original := "We can just process 1,000 rows, basically.\nIf needed, keep headings."
	rewritten := "We can process 1000 rows. If needed keep headings."

	result := Detect(original, rewritten)
	if !result.Allowed {
		t.Fatalf("expected rewrite to be allowed, got drifts: %+v", result.Drifts)
	}
	if len(result.Drifts) != 0 {
		t.Fatalf("expected no drifts, got %d", len(result.Drifts))
	}
}

func TestModalityAxisDetectsAdditionsRemovalsAndChanges(t *testing.T) {
	t.Parallel()

	t.Run("addition", func(t *testing.T) {
		t.Parallel()

		result := Detect("We can deploy now.", "We can and must deploy now.")
		drift := expectOnlyAxis(t, result, AxisModality)
		expectContainsToken(t, drift.Added, "must")
	})

	t.Run("removal", func(t *testing.T) {
		t.Parallel()

		result := Detect("We can and should deploy now.", "We can deploy now.")
		drift := expectOnlyAxis(t, result, AxisModality)
		expectContainsToken(t, drift.Removed, "should")
	})

	t.Run("change", func(t *testing.T) {
		t.Parallel()

		result := Detect("We can deploy now.", "We must deploy now.")
		drift := expectOnlyAxis(t, result, AxisModality)
		expectContainsChange(t, drift.Changed, "can", "must", "upgrade")
	})
}

func TestNegationAxisDetectsAdditionsRemovalsAndChanges(t *testing.T) {
	t.Parallel()

	t.Run("addition", func(t *testing.T) {
		t.Parallel()

		result := Detect("Deploy now.", "Do not deploy now.")
		drift := expectOnlyAxis(t, result, AxisNegation)
		expectContainsToken(t, drift.Added, "not")
		expectContainsChange(t, drift.Changed, "positive", "negative", "polarity_flip")
	})

	t.Run("removal", func(t *testing.T) {
		t.Parallel()

		result := Detect("Do not deploy now.", "Deploy now.")
		drift := expectOnlyAxis(t, result, AxisNegation)
		expectContainsToken(t, drift.Removed, "not")
		expectContainsChange(t, drift.Changed, "negative", "positive", "polarity_flip")
	})

	t.Run("change", func(t *testing.T) {
		t.Parallel()

		result := Detect("Do not deploy now.", "Never deploy now.")
		drift := expectOnlyAxis(t, result, AxisNegation)
		expectContainsChange(t, drift.Changed, "not", "never", "change")
	})
}

func TestNumericAxisDetectsAdditionsRemovalsAndChanges(t *testing.T) {
	t.Parallel()

	t.Run("addition", func(t *testing.T) {
		t.Parallel()

		result := Detect("Retry after 5s.", "Retry after 5s for 2 attempts.")
		drift := expectOnlyAxis(t, result, AxisNumeric)
		expectContainsToken(t, drift.Added, "2")
	})

	t.Run("removal", func(t *testing.T) {
		t.Parallel()

		result := Detect("Retry after 5s for 2 attempts.", "Retry after 5s.")
		drift := expectOnlyAxis(t, result, AxisNumeric)
		expectContainsToken(t, drift.Removed, "2")
	})

	t.Run("change", func(t *testing.T) {
		t.Parallel()

		result := Detect("Retry after 5s.", "Retry after 7s.")
		drift := expectOnlyAxis(t, result, AxisNumeric)
		expectContainsChange(t, drift.Changed, "5s", "7s", "change")
	})
}

func TestScopeAxisDetectsAdditionsRemovalsAndChanges(t *testing.T) {
	t.Parallel()

	t.Run("addition", func(t *testing.T) {
		t.Parallel()

		result := Detect("Requests pass.", "Only requests pass.")
		drift := expectOnlyAxis(t, result, AxisScope)
		expectContainsToken(t, drift.Added, "only")
	})

	t.Run("removal", func(t *testing.T) {
		t.Parallel()

		result := Detect("Only requests pass.", "Requests pass.")
		drift := expectOnlyAxis(t, result, AxisScope)
		expectContainsToken(t, drift.Removed, "only")
	})

	t.Run("change", func(t *testing.T) {
		t.Parallel()

		result := Detect("At least 3 checks pass.", "At most 3 checks pass.")
		drift := expectOnlyAxis(t, result, AxisScope)
		expectContainsChange(t, drift.Changed, "at least", "at most", "change")
	})
}

func TestConditionalAxisDetectsAdditionsRemovalsAndChanges(t *testing.T) {
	t.Parallel()

	t.Run("addition", func(t *testing.T) {
		t.Parallel()

		result := Detect("Deploy now.", "Deploy if tests pass.")
		drift := expectOnlyAxis(t, result, AxisConditional)
		expectContainsToken(t, drift.Added, "if")
	})

	t.Run("removal", func(t *testing.T) {
		t.Parallel()

		result := Detect("Deploy if tests pass.", "Deploy now.")
		drift := expectOnlyAxis(t, result, AxisConditional)
		expectContainsToken(t, drift.Removed, "if")
	})

	t.Run("change", func(t *testing.T) {
		t.Parallel()

		result := Detect("Deploy if tests pass.", "Deploy provided tests pass.")
		drift := expectOnlyAxis(t, result, AxisConditional)
		expectContainsChange(t, drift.Changed, "if", "provided", "change")
	})
}

func TestCommitmentAxisDetectsAdditionsRemovalsAndChanges(t *testing.T) {
	t.Parallel()

	t.Run("addition", func(t *testing.T) {
		t.Parallel()

		result := Detect("We will ship this.", "Maybe we will ship this.")
		drift := expectOnlyAxis(t, result, AxisCommitment)
		expectContainsToken(t, drift.Added, "maybe")
	})

	t.Run("removal", func(t *testing.T) {
		t.Parallel()

		result := Detect("I think we will ship this.", "We will ship this.")
		drift := expectOnlyAxis(t, result, AxisCommitment)
		expectContainsToken(t, drift.Removed, "i think")
	})

	t.Run("change", func(t *testing.T) {
		t.Parallel()

		result := Detect("I think we will ship this.", "Probably we will ship this.")
		drift := expectOnlyAxis(t, result, AxisCommitment)
		expectContainsChange(t, drift.Changed, "i think", "probably", "change")
	})
}

func TestDetectCombinationReportsAllFailedAxes(t *testing.T) {
	t.Parallel()

	original := "I think we can deploy 2 services if checks pass."
	rewritten := "We must deploy 3 services unless checks pass."

	result := Detect(original, rewritten)
	if result.Allowed {
		t.Fatal("expected rewrite to be blocked")
	}

	for _, axis := range []Axis{
		AxisModality,
		AxisNegation,
		AxisNumeric,
		AxisConditional,
		AxisCommitment,
	} {
		drift := expectHasAxis(t, result, axis)
		if len(drift.Added) == 0 && len(drift.Removed) == 0 && len(drift.Changed) == 0 {
			t.Fatalf("expected axis %q to report what changed", axis)
		}
	}
}

func expectOnlyAxis(t *testing.T, result Result, axis Axis) AxisDrift {
	t.Helper()

	if result.Allowed {
		t.Fatal("expected drift result to be disallowed")
	}
	if len(result.Drifts) != 1 {
		t.Fatalf("expected 1 drift axis, got %d", len(result.Drifts))
	}
	if result.Drifts[0].Axis != axis {
		t.Fatalf("expected drift axis %q, got %q", axis, result.Drifts[0].Axis)
	}

	return result.Drifts[0]
}

func expectHasAxis(t *testing.T, result Result, axis Axis) AxisDrift {
	t.Helper()

	for _, drift := range result.Drifts {
		if drift.Axis == axis {
			return drift
		}
	}

	t.Fatalf("expected drift axis %q in result: %+v", axis, result.Drifts)
	return AxisDrift{}
}

func expectContainsToken(t *testing.T, tokens []string, expected string) {
	t.Helper()

	if !slices.Contains(tokens, expected) {
		t.Fatalf("expected token %q in %v", expected, tokens)
	}
}

func expectContainsChange(
	t *testing.T,
	changes []TokenChange,
	expectedFrom string,
	expectedTo string,
	expectedKind string,
) {
	t.Helper()

	for _, change := range changes {
		if change.From == expectedFrom && change.To == expectedTo && change.Kind == expectedKind {
			return
		}
	}

	t.Fatalf(
		"expected change %q -> %q (%s) in %+v",
		expectedFrom,
		expectedTo,
		expectedKind,
		changes,
	)
}
