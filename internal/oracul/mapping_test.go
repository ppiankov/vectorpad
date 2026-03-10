package oracul

import (
	"testing"

	"github.com/ppiankov/vectorpad/internal/classifier"
)

func TestMapSentencesAllTags(t *testing.T) {
	sentences := []classifier.Sentence{
		{Text: "We will use Kafka for messaging.", Tag: classifier.TagDecision},
		{Text: "Must support 10k msgs/sec.", Tag: classifier.TagConstraint},
		{Text: "Budget cannot exceed $500/mo.", Tag: classifier.TagConstraint},
		{Text: "We could also try RabbitMQ.", Tag: classifier.TagTentative},
		{Text: "Kafka might be overkill for this.", Tag: classifier.TagSpeculation},
		{Text: "The team has experience with Redis.", Tag: classifier.TagExplanation},
		{Text: "Should we consider NATS?", Tag: classifier.TagQuestion},
	}

	filing := MapSentences(sentences)

	if filing.Decision != "We will use Kafka for messaging." {
		t.Errorf("Decision = %q", filing.Decision)
	}
	if len(filing.Constraints) != 2 {
		t.Errorf("Constraints count = %d, want 2", len(filing.Constraints))
	}
	if len(filing.Alternatives) != 1 {
		t.Errorf("Alternatives count = %d, want 1", len(filing.Alternatives))
	}
	if len(filing.KnownRisks) != 1 {
		t.Errorf("KnownRisks count = %d, want 1", len(filing.KnownRisks))
	}
	if filing.Context != "The team has experience with Redis." {
		t.Errorf("Context = %q", filing.Context)
	}
}

func TestMapSentencesMultipleDecisions(t *testing.T) {
	sentences := []classifier.Sentence{
		{Text: "We will use Kafka.", Tag: classifier.TagDecision},
		{Text: "We will deploy to AWS.", Tag: classifier.TagDecision},
	}

	filing := MapSentences(sentences)

	if filing.Decision != "We will use Kafka." {
		t.Errorf("Decision = %q, want first decision", filing.Decision)
	}
	if filing.Context != "We will deploy to AWS." {
		t.Errorf("Context should contain second decision, got %q", filing.Context)
	}
}

func TestMapSentencesNoDecision(t *testing.T) {
	sentences := []classifier.Sentence{
		{Text: "Must support 10k msgs/sec.", Tag: classifier.TagConstraint},
		{Text: "The system handles payments.", Tag: classifier.TagExplanation},
	}

	filing := MapSentences(sentences)

	if filing.Decision == "" {
		t.Error("Decision should be non-empty when no DECISION tag")
	}
	// Should contain all text.
	if filing.Decision != "Must support 10k msgs/sec. The system handles payments." {
		t.Errorf("Decision = %q", filing.Decision)
	}
}

func TestMapSentencesEmpty(t *testing.T) {
	filing := MapSentences(nil)

	if filing.Decision != "" {
		t.Errorf("Decision = %q, want empty", filing.Decision)
	}
}

func TestMapSentencesQuestionsOnly(t *testing.T) {
	sentences := []classifier.Sentence{
		{Text: "Should we use Kafka or RabbitMQ?", Tag: classifier.TagQuestion},
	}

	filing := MapSentences(sentences)

	// Questions are excluded from filing fields, but the fallback
	// uses all text as decision.
	if filing.Decision != "Should we use Kafka or RabbitMQ?" {
		t.Errorf("Decision = %q", filing.Decision)
	}
}

func TestExtractQuestion(t *testing.T) {
	sentences := []classifier.Sentence{
		{Text: "We will use Kafka.", Tag: classifier.TagDecision},
		{Text: "Should we consider NATS?", Tag: classifier.TagQuestion},
	}

	q := ExtractQuestion(sentences, "full text")
	if q != "Should we consider NATS?" {
		t.Errorf("question = %q", q)
	}
}

func TestExtractQuestionFallback(t *testing.T) {
	sentences := []classifier.Sentence{
		{Text: "We will use Kafka.", Tag: classifier.TagDecision},
	}

	q := ExtractQuestion(sentences, "full text fallback")
	if q != "full text fallback" {
		t.Errorf("question = %q, want fallback", q)
	}
}

func TestMapSentencesWhitespace(t *testing.T) {
	sentences := []classifier.Sentence{
		{Text: "  ", Tag: classifier.TagDecision},
		{Text: "Real decision.", Tag: classifier.TagDecision},
	}

	filing := MapSentences(sentences)

	if filing.Decision != "Real decision." {
		t.Errorf("Decision = %q, should skip whitespace-only", filing.Decision)
	}
}
