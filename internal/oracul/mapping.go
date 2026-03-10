package oracul

import (
	"strings"

	"github.com/ppiankov/vectorpad/internal/classifier"
)

// MapSentences converts classified sentences into an Oracul CaseFiling.
// Tag mapping:
//
//	DECISION    -> decision (first match; extras go to context)
//	CONSTRAINT  -> constraints[]
//	TENTATIVE   -> alternatives[]
//	SPECULATION -> known_risks[]
//	EXPLANATION -> context (concatenated)
//	QUESTION    -> ignored (used as raw question in ConsultRequest)
func MapSentences(sentences []classifier.Sentence) *CaseFiling {
	var (
		decision     string
		context      []string
		constraints  []string
		alternatives []string
		knownRisks   []string
	)

	for _, s := range sentences {
		text := strings.TrimSpace(s.Text)
		if text == "" {
			continue
		}

		switch s.Tag {
		case classifier.TagDecision:
			if decision == "" {
				decision = text
			} else {
				context = append(context, text)
			}
		case classifier.TagConstraint:
			constraints = append(constraints, text)
		case classifier.TagTentative:
			alternatives = append(alternatives, text)
		case classifier.TagSpeculation:
			knownRisks = append(knownRisks, text)
		case classifier.TagExplanation:
			context = append(context, text)
		case classifier.TagQuestion:
			// Questions are not part of the filing.
			// They become the top-level question in ConsultRequest.
		}
	}

	// If no decision sentence, use all text as the decision.
	if decision == "" {
		var all []string
		for _, s := range sentences {
			t := strings.TrimSpace(s.Text)
			if t != "" {
				all = append(all, t)
			}
		}
		decision = strings.Join(all, " ")
	}

	filing := &CaseFiling{
		Decision:     decision,
		Constraints:  constraints,
		Alternatives: alternatives,
		KnownRisks:   knownRisks,
	}

	if len(context) > 0 {
		filing.Context = strings.Join(context, ". ")
	}

	return filing
}

// ExtractQuestion returns the first QUESTION sentence text, or the
// full input text if no QUESTION is found. Used as the top-level
// question field in ConsultRequest.
func ExtractQuestion(sentences []classifier.Sentence, fullText string) string {
	for _, s := range sentences {
		if s.Tag == classifier.TagQuestion {
			text := strings.TrimSpace(s.Text)
			if text != "" {
				return text
			}
		}
	}
	return fullText
}
