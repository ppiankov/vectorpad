package stash

import "math"

// SimilarityLevel classifies how close two items are.
type SimilarityLevel string

const (
	SimilarityNearDuplicate SimilarityLevel = "near_duplicate" // >0.90
	SimilaritySameIdea      SimilarityLevel = "same_idea"      // 0.80-0.90
	SimilarityRelated       SimilarityLevel = "related"        // 0.65-0.80
	SimilarityDifferent     SimilarityLevel = "different"      // <0.65
)

const (
	thresholdNearDuplicate = 0.90
	thresholdSameIdea      = 0.80
	thresholdRelated       = 0.65
)

// SimilarResult pairs an item with its similarity score to a query.
type SimilarResult struct {
	Item  Item
	Score float64
	Level SimilarityLevel
}

// CosineSimilarity computes cosine similarity between two float32 vectors.
// Returns 0 if either vector has zero magnitude.
func CosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) || len(a) == 0 {
		return 0
	}

	var dot, magA, magB float64
	for i := range a {
		fa, fb := float64(a[i]), float64(b[i])
		dot += fa * fb
		magA += fa * fa
		magB += fb * fb
	}

	if magA == 0 || magB == 0 {
		return 0
	}
	return dot / (math.Sqrt(magA) * math.Sqrt(magB))
}

// ThresholdRelated returns the minimum score for "related" classification.
func ThresholdRelated() float64 {
	return thresholdRelated
}

// ClassifySimilarity maps a cosine similarity score to a level.
func ClassifySimilarity(score float64) SimilarityLevel {
	switch {
	case score >= thresholdNearDuplicate:
		return SimilarityNearDuplicate
	case score >= thresholdSameIdea:
		return SimilaritySameIdea
	case score >= thresholdRelated:
		return SimilarityRelated
	default:
		return SimilarityDifferent
	}
}
