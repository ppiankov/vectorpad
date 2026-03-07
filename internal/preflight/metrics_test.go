package preflight

import (
	"encoding/json"
	"math"
	"strings"
	"testing"

	"github.com/ppiankov/vectorpad/internal/classifier"
)

func TestComputeTokenWeightWithinTenPercent(t *testing.T) {
	t.Parallel()

	// Expected counts sourced from published cl100k_base examples.
	cases := []struct {
		name        string
		text        string
		actualCount int
	}{
		{
			name:        "long english tokenization",
			text:        "antidisestablishmentarianism",
			actualCount: 6,
		},
		{
			name:        "math spacing tokenization",
			text:        "2 + 2 = 4",
			actualCount: 7,
		},
		{
			name:        "multibyte tokenization",
			text:        "お誕生日おめでとう",
			actualCount: 9,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			sentences := classifier.Classify(tc.text)
			metrics := Compute(tc.text, sentences)

			if metrics.TokenWeight.ActualTiktoken != tc.actualCount {
				t.Fatalf(
					"expected actual tiktoken count %d, got %d",
					tc.actualCount,
					metrics.TokenWeight.ActualTiktoken,
				)
			}

			if math.Abs(metrics.TokenWeight.DeltaPercent) > 10 {
				t.Fatalf(
					"expected token estimate delta within 10%%, got %.2f%%",
					metrics.TokenWeight.DeltaPercent,
				)
			}
			if !metrics.TokenWeight.WithinTenPercent {
				t.Fatal("expected within_ten_percent to be true")
			}
		})
	}
}

func TestCountTiktokenStyleTokensSpacingAndMergeExamples(t *testing.T) {
	t.Parallel()

	cases := []struct {
		text        string
		actualCount int
	}{
		{text: "Hello world", actualCount: 2},
		{text: "Hello  world", actualCount: 3},
		{text: "Hello   world", actualCount: 3},
		{text: "Hello, world!", actualCount: 4},
		{text: "fireplace", actualCount: 2},
		{text: "firefox", actualCount: 1},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.text, func(t *testing.T) {
			t.Parallel()

			count := countTiktokenStyleTokens(tc.text)
			if count != tc.actualCount {
				t.Fatalf("expected %d tokens, got %d", tc.actualCount, count)
			}
		})
	}
}

func TestComputeVectorIntegrityFromLockedSentenceRatio(t *testing.T) {
	t.Parallel()

	sentences := []classifier.Sentence{
		{Text: "Do not remove the safety check.", LockPolicy: classifier.LockPolicyHard},
		{Text: "We will ship this behind a flag.", LockPolicy: classifier.LockPolicySoft},
		{Text: "Maybe we can stage this rollout.", LockPolicy: classifier.LockPolicyModalSpan},
		{Text: "How should we split this task?", LockPolicy: classifier.LockPolicyNone},
	}

	metrics := Compute("", sentences)
	if metrics.VectorIntegrity.LockedSentences != 3 {
		t.Fatalf("expected 3 locked sentences, got %d", metrics.VectorIntegrity.LockedSentences)
	}
	if metrics.VectorIntegrity.TotalSentences != 4 {
		t.Fatalf("expected 4 total sentences, got %d", metrics.VectorIntegrity.TotalSentences)
	}
	if metrics.VectorIntegrity.Ratio != 0.75 {
		t.Fatalf("expected ratio 0.75, got %.2f", metrics.VectorIntegrity.Ratio)
	}
	if metrics.VectorIntegrity.Percentage != 75 {
		t.Fatalf("expected percentage 75, got %.2f", metrics.VectorIntegrity.Percentage)
	}
}

func TestRenderHumanAndJSONOutputs(t *testing.T) {
	t.Parallel()

	text := strings.Join([]string{
		"Do not mutate schema without migration.",
		"We will ship the dry-run mode first.",
		"Maybe we can gate this with a feature flag.",
	}, " ")

	metrics := Compute(text, classifier.Classify(text))

	human := RenderHuman(metrics)
	requiredFragments := []string{
		"PREFLIGHT",
		"Token weight:",
		"Vector integrity:",
		"CPD projection:",
		"TTC projection:",
		"CDR projection:",
	}
	for _, fragment := range requiredFragments {
		if !strings.Contains(human, fragment) {
			t.Fatalf("expected human output to contain %q, got:\n%s", fragment, human)
		}
	}

	jsonOutput, err := RenderJSON(metrics)
	if err != nil {
		t.Fatalf("expected valid json output, got error: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal([]byte(jsonOutput), &decoded); err != nil {
		t.Fatalf("expected parseable json output, got: %v", err)
	}

	for _, key := range []string{
		"token_weight",
		"vector_integrity",
		"cpd_projection",
		"ttc_projection",
		"cdr_projection",
	} {
		if _, exists := decoded[key]; !exists {
			t.Fatalf("expected json output to include key %q", key)
		}
	}
}
