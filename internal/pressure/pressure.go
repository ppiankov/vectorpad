package pressure

import (
	"github.com/ppiankov/vectorpad/internal/classifier"
)

// Level represents the pressure tier for a sentence.
type Level int

const (
	LevelLow    Level = iota // locked constraint, no risk signals
	LevelMedium              // unlocked or soft-locked, minor signals
	LevelHigh                // unlocked + vague/scope-expanding, significant risk
)

// SentenceScore holds the pressure assessment for one sentence.
type SentenceScore struct {
	Index   int      `json:"index"`
	Level   Level    `json:"level"`
	Score   int      `json:"score"` // 0-100 composite
	Signals []string `json:"signals,omitempty"`
}

// Score computes per-sentence pressure from classification and vague verbs.
func Score(sentences []classifier.Sentence, vagueVerbs []string) []SentenceScore {
	vagueSet := make(map[string]bool, len(vagueVerbs))
	for _, v := range vagueVerbs {
		vagueSet[v] = true
	}

	scores := make([]SentenceScore, len(sentences))
	for i, s := range sentences {
		score, signals := scoreSentence(s, vagueSet)
		level := LevelLow
		if score >= 60 {
			level = LevelHigh
		} else if score >= 30 {
			level = LevelMedium
		}
		scores[i] = SentenceScore{
			Index:   i,
			Level:   level,
			Score:   score,
			Signals: signals,
		}
	}
	return scores
}

func scoreSentence(s classifier.Sentence, vagueVerbs map[string]bool) (int, []string) {
	score := 0
	var signals []string

	// Lock policy contribution (0-40 points).
	switch s.LockPolicy {
	case classifier.LockPolicyHard:
		// Locked constraint — low pressure.
	case classifier.LockPolicySoft:
		score += 15
		signals = append(signals, "soft-locked")
	case classifier.LockPolicyModalSpan:
		score += 20
		signals = append(signals, "modal-span")
	case classifier.LockPolicyNone:
		score += 30
		signals = append(signals, "unlocked")
	}

	// Tag contribution (0-20 points).
	switch s.Tag {
	case classifier.TagSpeculation:
		score += 30
		signals = append(signals, "speculation")
	case classifier.TagTentative:
		score += 15
		signals = append(signals, "tentative")
	case classifier.TagQuestion:
		score += 10
		signals = append(signals, "question")
	case classifier.TagExplanation:
		score += 5
	}

	// Vague verb presence (0-30 points).
	for verb := range vagueVerbs {
		if containsWord(s.Text, verb) {
			score += 30
			signals = append(signals, "vague: "+verb)
			break // only count once
		}
	}

	// Brevity penalty — very short sentences are higher risk (0-20 points).
	words := countWords(s.Text)
	if words <= 3 {
		score += 20
		signals = append(signals, "very short")
	} else if words <= 6 {
		score += 10
		signals = append(signals, "short")
	}

	if score > 100 {
		score = 100
	}
	return score, signals
}

func containsWord(text, word string) bool {
	lower := toLower(text)
	wLower := toLower(word)
	idx := 0
	for {
		pos := indexOf(lower[idx:], wLower)
		if pos < 0 {
			return false
		}
		start := idx + pos
		end := start + len(wLower)
		// Check word boundaries.
		startOK := start == 0 || !isAlpha(lower[start-1])
		endOK := end >= len(lower) || !isAlpha(lower[end])
		if startOK && endOK {
			return true
		}
		idx = start + 1
	}
}

func countWords(text string) int {
	count := 0
	inWord := false
	for _, c := range text {
		if c == ' ' || c == '\t' || c == '\n' {
			inWord = false
		} else if !inWord {
			inWord = true
			count++
		}
	}
	return count
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}

func indexOf(s, sub string) int {
	if len(sub) > len(s) {
		return -1
	}
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func isAlpha(b byte) bool {
	return (b >= 'a' && b <= 'z') || (b >= 'A' && b <= 'Z')
}
