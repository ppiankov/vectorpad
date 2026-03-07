package vector

import (
	"fmt"
	"strings"

	"github.com/ppiankov/vectorpad/internal/classifier"
)

type group struct {
	tag   classifier.Tag
	title string
}

var groupOrder = []group{
	{tag: classifier.TagConstraint, title: "Constraints"},
	{tag: classifier.TagDecision, title: "Decisions"},
	{tag: classifier.TagTentative, title: "Tentatives"},
	{tag: classifier.TagQuestion, title: "Questions"},
	{tag: classifier.TagSpeculation, title: "Speculations"},
	{tag: classifier.TagExplanation, title: "Explanations"},
}

// Render formats classified sentences as a structured Vector Block.
func Render(sentences []classifier.Sentence) string {
	grouped := make(map[classifier.Tag][]classifier.Sentence, len(groupOrder))
	for _, sentence := range sentences {
		grouped[sentence.Tag] = append(grouped[sentence.Tag], sentence)
	}

	var b strings.Builder
	b.WriteString("VECTOR\n")

	for _, current := range groupOrder {
		b.WriteString(fmt.Sprintf("  %s:\n", current.title))
		items := grouped[current.tag]
		if len(items) == 0 {
			b.WriteString("    - (none)\n")
			continue
		}

		for _, item := range items {
			b.WriteString("    - ")
			b.WriteString(formatLabel(item))
			b.WriteString(" ")
			b.WriteString(item.Text)
			b.WriteString("\n")
		}
	}

	return strings.TrimRight(b.String(), "\n")
}

func formatLabel(sentence classifier.Sentence) string {
	base := fmt.Sprintf("[%s]", sentence.Tag)

	switch sentence.LockPolicy {
	case classifier.LockPolicyHard, classifier.LockPolicySoft:
		return base + "[LOCKED]"
	case classifier.LockPolicyModalSpan:
		if len(sentence.LockedSpans) == 0 {
			return base
		}

		spans := make([]string, 0, len(sentence.LockedSpans))
		for _, span := range sentence.LockedSpans {
			spans = append(spans, span.Text)
		}
		return fmt.Sprintf("%s[LOCKED:%s]", base, strings.Join(spans, "|"))
	default:
		return base
	}
}
