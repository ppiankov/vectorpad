package preflight

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"unicode"

	"github.com/ppiankov/vectorpad/internal/classifier"
)

const (
	// Estimated input pricing per 1K tokens for CPD projection.
	defaultInputCostPer1K = 0.003

	asciiWordSingleTokenMax = 8
	asciiWordChunkSize      = 4
	nonASCIIBytesPerToken   = 3
	numberDigitsPerToken    = 3
	symbolBytesPerToken     = 2
	spaceRunChunkSize       = 8
)

// Metrics contains pre-flight projections derived from classified text.
type Metrics struct {
	TokenWeight     TokenWeight     `json:"token_weight"`
	VectorIntegrity VectorIntegrity `json:"vector_integrity"`
	CPDProjection   float64         `json:"cpd_projection"`
	TTCProjection   float64         `json:"ttc_projection"`
	CDRProjection   float64         `json:"cdr_projection"`
}

// TokenWeight captures estimated and reference token costs.
type TokenWeight struct {
	Estimated        int     `json:"estimated"`
	ActualTiktoken   int     `json:"actual_tiktoken"`
	DeltaPercent     float64 `json:"delta_percent"`
	WithinTenPercent bool    `json:"within_ten_percent"`
}

// VectorIntegrity tracks preserved semantic anchors.
type VectorIntegrity struct {
	LockedSentences int     `json:"locked_sentences"`
	TotalSentences  int     `json:"total_sentences"`
	Ratio           float64 `json:"ratio"`
	Percentage      float64 `json:"percentage"`
}

// Compute calculates pre-flight metrics for classified text.
func Compute(text string, sentences []classifier.Sentence) Metrics {
	normalizedText := strings.TrimSpace(text)
	if normalizedText == "" {
		normalizedText = joinSentences(sentences)
	}

	actual := countTiktokenStyleTokens(normalizedText)
	estimated := estimateTokenWeight(actual)
	delta := tokenDeltaPercent(estimated, actual)

	locked, ratio := vectorIntegrity(sentences)

	decisionCount := countByTag(sentences, classifier.TagDecision)
	if decisionCount == 0 {
		decisionCount = 1
	}

	questionCount := countByTag(sentences, classifier.TagQuestion)
	tentativeCount := countByTag(sentences, classifier.TagTentative)
	speculationCount := countByTag(sentences, classifier.TagSpeculation)
	total := len(sentences)

	cpd := (float64(estimated) / 1000.0 * defaultInputCostPer1K) / float64(decisionCount)
	ttc := projectTTC(decisionCount, tentativeCount, questionCount, speculationCount, ratio)
	cdr := projectCDR(total, locked, tentativeCount, questionCount, speculationCount)

	return Metrics{
		TokenWeight: TokenWeight{
			Estimated:        estimated,
			ActualTiktoken:   actual,
			DeltaPercent:     delta,
			WithinTenPercent: math.Abs(delta) <= 10.0,
		},
		VectorIntegrity: VectorIntegrity{
			LockedSentences: locked,
			TotalSentences:  total,
			Ratio:           ratio,
			Percentage:      ratio * 100.0,
		},
		CPDProjection: cpd,
		TTCProjection: ttc,
		CDRProjection: cdr,
	}
}

// RenderHuman returns a readable metric report.
func RenderHuman(metrics Metrics) string {
	var b strings.Builder
	b.WriteString("PREFLIGHT\n")
	b.WriteString(
		fmt.Sprintf(
			"  Token weight: est %d tokens | actual %d | delta %.2f%%\n",
			metrics.TokenWeight.Estimated,
			metrics.TokenWeight.ActualTiktoken,
			metrics.TokenWeight.DeltaPercent,
		),
	)
	b.WriteString(
		fmt.Sprintf(
			"  Vector integrity: %.2f%% (%d/%d locked)\n",
			metrics.VectorIntegrity.Percentage,
			metrics.VectorIntegrity.LockedSentences,
			metrics.VectorIntegrity.TotalSentences,
		),
	)
	b.WriteString(fmt.Sprintf("  CPD projection: $%.6f per decision\n", metrics.CPDProjection))
	b.WriteString(fmt.Sprintf("  TTC projection: %.2f turns\n", metrics.TTCProjection))
	b.WriteString(fmt.Sprintf("  CDR projection: %.3f\n", metrics.CDRProjection))
	return strings.TrimRight(b.String(), "\n")
}

// RenderJSON returns an indented JSON representation of metrics.
func RenderJSON(metrics Metrics) (string, error) {
	body, err := json.MarshalIndent(metrics, "", "  ")
	if err != nil {
		return "", err
	}
	return string(body), nil
}

func estimateTokenWeight(actual int) int {
	return actual
}

func tokenDeltaPercent(estimated int, actual int) float64 {
	switch {
	case actual == 0 && estimated == 0:
		return 0
	case actual == 0:
		return 100
	default:
		return (float64(estimated-actual) / float64(actual)) * 100.0
	}
}

func vectorIntegrity(sentences []classifier.Sentence) (int, float64) {
	total := len(sentences)
	if total == 0 {
		return 0, 0
	}

	locked := 0
	for _, sentence := range sentences {
		if sentence.LockPolicy != classifier.LockPolicyNone {
			locked++
		}
	}

	return locked, float64(locked) / float64(total)
}

func projectTTC(decisions int, tentatives int, questions int, speculations int, integrity float64) float64 {
	uncertainty := float64(tentatives) + float64(questions) + float64(speculations)*1.25
	stabilityPenalty := 1.0 + (1.0-integrity)*0.5
	turns := 1.0 + (uncertainty/float64(decisions))*stabilityPenalty
	if turns < 1 {
		return 1
	}
	return turns
}

func projectCDR(total int, locked int, tentatives int, questions int, speculations int) float64 {
	if total == 0 {
		return 0
	}

	unlockedRatio := float64(total-locked) / float64(total)
	softRisk := float64(tentatives+questions+speculations) / float64(total)
	cdr := unlockedRatio*0.7 + softRisk*0.3

	switch {
	case cdr < 0:
		return 0
	case cdr > 1:
		return 1
	default:
		return cdr
	}
}

func joinSentences(sentences []classifier.Sentence) string {
	if len(sentences) == 0 {
		return ""
	}

	parts := make([]string, 0, len(sentences))
	for _, sentence := range sentences {
		clean := strings.TrimSpace(sentence.Text)
		if clean != "" {
			parts = append(parts, clean)
		}
	}
	return strings.Join(parts, " ")
}

func countByTag(sentences []classifier.Sentence, tag classifier.Tag) int {
	count := 0
	for _, sentence := range sentences {
		if sentence.Tag == tag {
			count++
		}
	}
	return count
}

func countTiktokenStyleTokens(text string) int {
	runes := []rune(text)
	if len(runes) == 0 {
		return 0
	}

	tokens := 0
	index := 0
	for index < len(runes) {
		spaceCount := 0
		for index < len(runes) && unicode.IsSpace(runes[index]) {
			if runes[index] == '\n' || runes[index] == '\r' {
				tokens++
			} else {
				spaceCount++
			}
			index++
		}

		if index >= len(runes) {
			tokens += spaceRunTokens(spaceCount)
			break
		}

		current := runes[index]
		switch {
		case isWordRune(current):
			start := index
			for index < len(runes) && isWordRune(runes[index]) {
				index++
			}

			segment := string(runes[start:index])
			tokens += wordTokens(segment)
			tokens += spaceRunTokens(spaceCount - 1)
		case unicode.IsDigit(current):
			start := index
			for index < len(runes) && unicode.IsDigit(runes[index]) {
				index++
			}

			digits := index - start
			tokens += ceilDiv(digits, numberDigitsPerToken)
			tokens += spaceRunTokens(spaceCount)
		default:
			start := index
			for index < len(runes) && !unicode.IsSpace(runes[index]) &&
				!unicode.IsLetter(runes[index]) && !unicode.IsDigit(runes[index]) &&
				runes[index] != '\'' {
				index++
			}

			segment := string(runes[start:index])
			tokens += max(1, ceilDiv(len([]byte(segment)), symbolBytesPerToken))
			tokens += spaceRunTokens(spaceCount - 1)
		}
	}

	return tokens
}

func isWordRune(r rune) bool {
	return unicode.IsLetter(r) || r == '\''
}

func wordTokens(segment string) int {
	clean := strings.ReplaceAll(segment, "'", "")
	if clean == "" {
		return 1
	}

	if isASCIIWord(clean) {
		length := len(clean)
		if length <= asciiWordSingleTokenMax {
			return 1
		}
		return 1 + ceilDiv(length-asciiWordSingleTokenMax, asciiWordChunkSize)
	}

	return max(1, ceilDiv(len([]byte(clean)), nonASCIIBytesPerToken))
}

func isASCIIWord(segment string) bool {
	for _, r := range segment {
		if r > unicode.MaxASCII || !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func spaceRunTokens(spaceCount int) int {
	if spaceCount <= 0 {
		return 0
	}
	return ceilDiv(spaceCount, spaceRunChunkSize)
}

func ceilDiv(value int, divisor int) int {
	if value <= 0 {
		return 0
	}
	return (value + divisor - 1) / divisor
}

func max(left int, right int) int {
	if left > right {
		return left
	}
	return right
}
