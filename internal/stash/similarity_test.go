package stash

import (
	"math"
	"testing"
)

func TestCosineSimilarityIdentical(t *testing.T) {
	a := []float32{1, 2, 3, 4}
	score := CosineSimilarity(a, a)
	if math.Abs(score-1.0) > 0.001 {
		t.Errorf("identical vectors should have similarity ~1.0, got %f", score)
	}
}

func TestCosineSimilarityOrthogonal(t *testing.T) {
	a := []float32{1, 0, 0}
	b := []float32{0, 1, 0}
	score := CosineSimilarity(a, b)
	if math.Abs(score) > 0.001 {
		t.Errorf("orthogonal vectors should have similarity ~0.0, got %f", score)
	}
}

func TestCosineSimilarityOpposite(t *testing.T) {
	a := []float32{1, 2, 3}
	b := []float32{-1, -2, -3}
	score := CosineSimilarity(a, b)
	if math.Abs(score+1.0) > 0.001 {
		t.Errorf("opposite vectors should have similarity ~-1.0, got %f", score)
	}
}

func TestCosineSimilarityEmpty(t *testing.T) {
	score := CosineSimilarity(nil, nil)
	if score != 0 {
		t.Errorf("empty vectors should have similarity 0, got %f", score)
	}
}

func TestCosineSimilarityLengthMismatch(t *testing.T) {
	a := []float32{1, 2}
	b := []float32{1, 2, 3}
	score := CosineSimilarity(a, b)
	if score != 0 {
		t.Errorf("mismatched vectors should have similarity 0, got %f", score)
	}
}

func TestCosineSimilarityZeroVector(t *testing.T) {
	a := []float32{0, 0, 0}
	b := []float32{1, 2, 3}
	score := CosineSimilarity(a, b)
	if score != 0 {
		t.Errorf("zero vector should have similarity 0, got %f", score)
	}
}

func TestClassifySimilarity(t *testing.T) {
	tests := []struct {
		score float64
		want  SimilarityLevel
	}{
		{0.95, SimilarityNearDuplicate},
		{0.90, SimilarityNearDuplicate},
		{0.85, SimilaritySameIdea},
		{0.80, SimilaritySameIdea},
		{0.70, SimilarityRelated},
		{0.65, SimilarityRelated},
		{0.50, SimilarityDifferent},
		{0.0, SimilarityDifferent},
	}
	for _, tt := range tests {
		got := ClassifySimilarity(tt.score)
		if got != tt.want {
			t.Errorf("ClassifySimilarity(%f) = %s, want %s", tt.score, got, tt.want)
		}
	}
}
