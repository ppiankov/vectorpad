package classifier

import (
	"regexp"
	"sort"
	"strings"
)

var (
	constraintTriggers = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bmust\s+not\b`),
		regexp.MustCompile(`(?i)\bmust\b`),
		regexp.MustCompile(`(?i)\bnever\b`),
		regexp.MustCompile(`(?i)\bdo\s+not\b`),
		regexp.MustCompile(`(?i)\bonly\s+if\b`),
		regexp.MustCompile(`(?i)\bunless\b`),
		regexp.MustCompile(`(?i)\bexcept\b`),
		regexp.MustCompile(`(?i)\bat\s+least\b`),
		regexp.MustCompile(`(?i)\bat\s+most\b`),
		regexp.MustCompile(`(?i)\bno\s+more\s+than\b`),
		regexp.MustCompile(`(?i)\bno\s+less\s+than\b`),
		regexp.MustCompile(`(?i)\b\d+(\.\d+)?(%|ms|s|m|h|d)?\b`),
		regexp.MustCompile(`(?i)\b\d{4}-\d{2}-\d{2}\b`),
		regexp.MustCompile(`(?i)\b\d{1,2}/\d{1,2}/\d{2,4}\b`),
	}

	shouldConstraintTrigger = regexp.MustCompile(`(?i)\bshould\b`)

	decisionTriggers = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bwe\s+will\b`),
		regexp.MustCompile(`(?i)\bwe'll\b`),
		regexp.MustCompile(`(?i)\blet'?s\b`),
		regexp.MustCompile(`(?i)\bdecide\b`),
		regexp.MustCompile(`(?i)\bgo\s+with\b`),
		regexp.MustCompile(`(?i)\bship\b`),
		regexp.MustCompile(`(?i)\bimplement\b`),
		regexp.MustCompile(`(?i)\bthe\s+plan\s+is\b`),
	}

	questionStartTrigger = regexp.MustCompile(`(?i)^\s*(should\s+we|can\s+we|how\s+do\s+we)\b`)
	questionEndTrigger   = regexp.MustCompile(`\?\s*$`)

	speculationTriggers = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bwhat\s+if\b`),
		regexp.MustCompile(`(?i)\bimagine\b`),
		regexp.MustCompile(`(?i)\bsuppose\b`),
		regexp.MustCompile(`(?i)\balternatively\b`),
		regexp.MustCompile(`(?i)\bcould\s+we\s+try\b`),
		regexp.MustCompile(`(?i)\banother\s+approach\b`),
	}
)

type tentativeTrigger struct {
	name string
	re   *regexp.Regexp
}

var tentativeTriggers = []tentativeTrigger{
	{name: "I think", re: regexp.MustCompile(`(?i)\bi\s+think\b`)},
	{name: "maybe", re: regexp.MustCompile(`(?i)\bmaybe\b`)},
	{name: "probably", re: regexp.MustCompile(`(?i)\bprobably\b`)},
	{name: "likely", re: regexp.MustCompile(`(?i)\blikely\b`)},
	{name: "seems", re: regexp.MustCompile(`(?i)\bseems\b`)},
	{name: "might", re: regexp.MustCompile(`(?i)\bmight\b`)},
	{name: "could", re: regexp.MustCompile(`(?i)\bcould\b`)},
}

// Classify converts raw text into classified sentences.
//
// Tag priority is deterministic and first-match-wins:
// CONSTRAINT -> DECISION -> TENTATIVE -> QUESTION -> SPECULATION -> EXPLANATION
func Classify(text string) []Sentence {
	parts := splitSentences(text)
	sentences := make([]Sentence, 0, len(parts))

	for _, sentenceText := range parts {
		tag, policy, spans := classifySentence(sentenceText)
		sentences = append(sentences, Sentence{
			Text:        sentenceText,
			Tag:         tag,
			LockPolicy:  policy,
			LockedSpans: spans,
		})
	}

	return sentences
}

func splitSentences(text string) []string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil
	}

	normalized := strings.ReplaceAll(trimmed, "\r\n", "\n")
	re := regexp.MustCompile(`(?m)[^.!?\n]+[.!?]?`)
	matches := re.FindAllString(normalized, -1)

	sentences := make([]string, 0, len(matches))
	for _, match := range matches {
		clean := strings.TrimSpace(match)
		if clean == "" {
			continue
		}
		sentences = append(sentences, clean)
	}

	return sentences
}

func classifySentence(text string) (Tag, LockPolicy, []LockedSpan) {
	if matchAny(constraintTriggers, text) ||
		(shouldConstraintTrigger.MatchString(text) && !questionStartTrigger.MatchString(text)) {
		return TagConstraint, LockPolicyHard, nil
	}
	if matchAny(decisionTriggers, text) {
		return TagDecision, LockPolicySoft, nil
	}

	tentativeSpans := findTentativeSpans(text)
	if len(tentativeSpans) > 0 {
		return TagTentative, LockPolicyModalSpan, tentativeSpans
	}

	if questionStartTrigger.MatchString(text) || questionEndTrigger.MatchString(text) {
		return TagQuestion, LockPolicyNone, nil
	}
	if matchAny(speculationTriggers, text) {
		return TagSpeculation, LockPolicyNone, nil
	}

	return TagExplanation, LockPolicyNone, nil
}

func matchAny(triggers []*regexp.Regexp, text string) bool {
	for _, trigger := range triggers {
		if trigger.MatchString(text) {
			return true
		}
	}
	return false
}

func findTentativeSpans(text string) []LockedSpan {
	type indexedSpan struct {
		start int
		end   int
		text  string
	}

	found := make([]indexedSpan, 0)
	for _, trigger := range tentativeTriggers {
		matches := trigger.re.FindAllStringIndex(text, -1)
		for _, match := range matches {
			start := match[0]
			end := match[1]
			if trigger.name == "could" && followsCouldWeTry(text, end) {
				continue
			}
			found = append(found, indexedSpan{
				start: start,
				end:   end,
				text:  strings.TrimSpace(text[start:end]),
			})
		}
	}

	if len(found) == 0 {
		return nil
	}

	sort.SliceStable(found, func(i, j int) bool {
		if found[i].start == found[j].start {
			return found[i].end < found[j].end
		}
		return found[i].start < found[j].start
	})

	deduped := make([]LockedSpan, 0, len(found))
	var lastStart, lastEnd int
	for i, span := range found {
		if i == 0 || span.start != lastStart || span.end != lastEnd {
			deduped = append(deduped, LockedSpan{
				Start: span.start,
				End:   span.end,
				Text:  span.text,
			})
			lastStart = span.start
			lastEnd = span.end
		}
	}

	return deduped
}

func followsCouldWeTry(text string, offset int) bool {
	rest := strings.ToLower(text[offset:])
	return strings.HasPrefix(strings.TrimLeft(rest, " \t"), "we try")
}
