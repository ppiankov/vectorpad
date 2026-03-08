package pressure

import (
	"testing"

	"github.com/ppiankov/vectorpad/internal/classifier"
)

func TestScoreLockedConstraint(t *testing.T) {
	sentences := []classifier.Sentence{
		{Text: "You must preserve all philosophy sections", Tag: classifier.TagConstraint, LockPolicy: classifier.LockPolicyHard},
	}
	scores := Score(sentences, nil)
	if len(scores) != 1 {
		t.Fatalf("expected 1 score, got %d", len(scores))
	}
	if scores[0].Level != LevelLow {
		t.Errorf("locked constraint should be low pressure, got level %d (score %d)", scores[0].Level, scores[0].Score)
	}
}

func TestScoreUnlockedExplanation(t *testing.T) {
	sentences := []classifier.Sentence{
		{Text: "This is just some context", Tag: classifier.TagExplanation, LockPolicy: classifier.LockPolicyNone},
	}
	scores := Score(sentences, nil)
	if scores[0].Level != LevelMedium {
		t.Errorf("unlocked explanation should be medium pressure, got level %d (score %d)", scores[0].Level, scores[0].Score)
	}
}

func TestScoreVagueVerbHighPressure(t *testing.T) {
	sentences := []classifier.Sentence{
		{Text: "clean up the code", Tag: classifier.TagExplanation, LockPolicy: classifier.LockPolicyNone},
	}
	scores := Score(sentences, []string{"clean"})
	if scores[0].Level != LevelHigh {
		t.Errorf("unlocked + vague verb should be high pressure, got level %d (score %d)", scores[0].Level, scores[0].Score)
	}
}

func TestScoreSpeculation(t *testing.T) {
	sentences := []classifier.Sentence{
		{Text: "Maybe we should consider alternative approaches to this problem", Tag: classifier.TagSpeculation, LockPolicy: classifier.LockPolicyNone},
	}
	scores := Score(sentences, nil)
	if scores[0].Level != LevelHigh {
		t.Errorf("unlocked speculation should be high pressure, got level %d (score %d)", scores[0].Level, scores[0].Score)
	}
}

func TestScoreMultipleSentences(t *testing.T) {
	sentences := []classifier.Sentence{
		{Text: "You must preserve the README voice", Tag: classifier.TagConstraint, LockPolicy: classifier.LockPolicyHard},
		{Text: "clean up", Tag: classifier.TagExplanation, LockPolicy: classifier.LockPolicyNone},
	}
	scores := Score(sentences, []string{"clean"})
	if len(scores) != 2 {
		t.Fatalf("expected 2 scores, got %d", len(scores))
	}
	if scores[0].Level >= scores[1].Level {
		t.Error("locked constraint should have lower pressure than unlocked vague verb")
	}
}

func TestScoreEmpty(t *testing.T) {
	scores := Score(nil, nil)
	if len(scores) != 0 {
		t.Errorf("expected 0 scores for nil input, got %d", len(scores))
	}
}

func TestScoreShortSentencePenalty(t *testing.T) {
	sentences := []classifier.Sentence{
		{Text: "fix it", Tag: classifier.TagExplanation, LockPolicy: classifier.LockPolicyNone},
	}
	scores := Score(sentences, nil)
	hasShort := false
	for _, s := range scores[0].Signals {
		if s == "very short" {
			hasShort = true
		}
	}
	if !hasShort {
		t.Error("expected 'very short' signal for 2-word sentence")
	}
}

func TestContainsWord(t *testing.T) {
	tests := []struct {
		text string
		word string
		want bool
	}{
		{"clean up the code", "clean", true},
		{"cleanup the code", "clean", false}, // not a word boundary
		{"the code is clean", "clean", true},
		{"CLEAN everything", "clean", true},
		{"unclean code", "clean", false},
	}
	for _, tt := range tests {
		got := containsWord(tt.text, tt.word)
		if got != tt.want {
			t.Errorf("containsWord(%q, %q) = %v, want %v", tt.text, tt.word, got, tt.want)
		}
	}
}
