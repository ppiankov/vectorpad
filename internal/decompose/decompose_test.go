package decompose

import (
	"testing"

	"github.com/ppiankov/vectorpad/internal/classifier"
)

func TestDecomposeNoTargets(t *testing.T) {
	sentences := []classifier.Sentence{
		{Text: "Clean up the code", Tag: classifier.TagExplanation},
		{Text: "Make it better", Tag: classifier.TagExplanation},
	}
	result := Decompose(sentences, 3)
	if result.Triggered {
		t.Error("should not trigger with no targets")
	}
}

func TestDecomposeSingleTarget(t *testing.T) {
	sentences := []classifier.Sentence{
		{Text: "Update main.go to add logging", Tag: classifier.TagExplanation},
		{Text: "Keep the error handling", Tag: classifier.TagConstraint},
	}
	result := Decompose(sentences, 3)
	if result.Triggered {
		t.Error("should not trigger with single target group")
	}
}

func TestDecomposeMultipleTargets(t *testing.T) {
	sentences := []classifier.Sentence{
		{Text: "You must preserve the API contract", Tag: classifier.TagConstraint, LockPolicy: classifier.LockPolicyHard},
		{Text: "Update internal/auth/auth.go to add JWT validation", Tag: classifier.TagExplanation},
		{Text: "Fix internal/cache/cache.go to handle TTL expiry", Tag: classifier.TagExplanation},
		{Text: "Add tests to internal/api/handler.go for the new endpoint", Tag: classifier.TagExplanation},
	}
	result := Decompose(sentences, 3)
	if !result.Triggered {
		t.Fatal("should trigger with 3+ distinct target groups")
	}
	if len(result.SubVectors) < 2 {
		t.Fatalf("expected at least 2 sub-vectors, got %d", len(result.SubVectors))
	}
}

func TestDecomposeSharedPreamble(t *testing.T) {
	sentences := []classifier.Sentence{
		{Text: "You must preserve backward compatibility", Tag: classifier.TagConstraint, LockPolicy: classifier.LockPolicyHard},
		{Text: "Update internal/auth/auth.go to validate tokens", Tag: classifier.TagExplanation},
		{Text: "Fix internal/cache/cache.go to expire stale entries", Tag: classifier.TagExplanation},
		{Text: "Add retry logic to internal/api/client.go", Tag: classifier.TagExplanation},
	}
	result := Decompose(sentences, 3)
	if !result.Triggered {
		t.Fatal("should trigger")
	}

	// Each sub-vector should contain the shared constraint.
	for _, sv := range result.SubVectors {
		hasConstraint := false
		for _, s := range sv.Sentences {
			if s.Tag == classifier.TagConstraint {
				hasConstraint = true
				break
			}
		}
		if !hasConstraint {
			t.Errorf("sub-vector %q missing shared constraint", sv.Label)
		}
	}
}

func TestDecomposePreservesAllContent(t *testing.T) {
	sentences := []classifier.Sentence{
		{Text: "Context for the task", Tag: classifier.TagExplanation},
		{Text: "Fix internal/auth/auth.go", Tag: classifier.TagExplanation},
		{Text: "Fix internal/cache/cache.go", Tag: classifier.TagExplanation},
		{Text: "Fix internal/api/api.go", Tag: classifier.TagExplanation},
	}
	result := Decompose(sentences, 3)
	if !result.Triggered {
		t.Fatal("should trigger")
	}

	// All original sentence texts should appear in at least one sub-vector.
	for _, s := range sentences {
		found := false
		for _, sv := range result.SubVectors {
			for _, svs := range sv.Sentences {
				if svs.Text == s.Text {
					found = true
					break
				}
			}
			if found {
				break
			}
		}
		if !found {
			t.Errorf("sentence %q not found in any sub-vector", s.Text)
		}
	}
}

func TestDecomposeThreshold(t *testing.T) {
	sentences := []classifier.Sentence{
		{Text: "Fix auth.go", Tag: classifier.TagExplanation},
		{Text: "Fix cache.go", Tag: classifier.TagExplanation},
	}
	// With threshold 5, 2 targets should not trigger.
	result := Decompose(sentences, 5)
	if result.Triggered {
		t.Error("should not trigger below threshold")
	}
}

func TestDecomposeEmpty(t *testing.T) {
	result := Decompose(nil, 3)
	if result.Triggered {
		t.Error("should not trigger on nil input")
	}
}

func TestExtractTargetsRepoPattern(t *testing.T) {
	targets := extractTargets("Update ppiankov/vectorpad and ppiankov/chainwatch")
	if len(targets) < 2 {
		t.Errorf("expected at least 2 targets from repo patterns, got %d: %v", len(targets), targets)
	}
}

func TestExtractTargetsFilePattern(t *testing.T) {
	targets := extractTargets("Fix main.go and config.yaml")
	if len(targets) < 2 {
		t.Errorf("expected at least 2 targets from file patterns, got %d: %v", len(targets), targets)
	}
}

func TestExtractTargetsScopeMarker(t *testing.T) {
	targets := extractTargets("Apply this to all repos")
	found := false
	for _, t := range targets {
		if t == "all repos" {
			found = true
		}
	}
	if !found {
		t.Error("expected 'all repos' scope marker")
	}
}
