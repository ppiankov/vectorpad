package detect

import (
	"math"
	"testing"
)

const fixtureStatsJSON = `{
  "session_id": "abc123",
  "health": {
    "grade": "B",
    "signal_tokens": 5000,
    "noise_tokens": 15000,
    "total_tokens": 20000,
    "signal_percent": 25.0,
    "noise_percent": 75.0
  },
  "context": {
    "tokens": 50000,
    "percent": 25.0,
    "window": 200000,
    "turns_remaining": 42
  },
  "cost": {
    "model": "claude-opus-4-6",
    "total_cost": 3.14,
    "cost_per_turn": 0.05,
    "input_tokens": 1000,
    "output_tokens": 5000
  },
  "decision_economics": {
    "cpd": 0.18,
    "ttc": 5,
    "cdr": 0.12,
    "total_decisions": 8,
    "decision_density": 0.004,
    "per_epoch": [
      {"epoch": 0, "cpd": 0.25, "ttc": 3, "cdr": 0.15, "decisions": 3, "density": 0.005},
      {"epoch": 1, "cpd": 0.12, "ttc": 7, "cdr": 0.09, "decisions": 5, "density": 0.003}
    ]
  }
}`

func TestParseFeedback(t *testing.T) {
	fb := parseFeedback([]byte(fixtureStatsJSON))
	if fb == nil {
		t.Fatal("expected non-nil feedback")
	}

	if fb.Grade != "B" {
		t.Errorf("grade: got %q, want B", fb.Grade)
	}
	if fb.ContextPercent != 25.0 {
		t.Errorf("context_percent: got %f, want 25.0", fb.ContextPercent)
	}
	if fb.TurnsRemaining != 42 {
		t.Errorf("turns_remaining: got %d, want 42", fb.TurnsRemaining)
	}
	if !fb.ChainHealthy {
		t.Error("chain_healthy: expected true for grade B")
	}
	if fb.Model != "claude-opus-4-6" {
		t.Errorf("model: got %q, want claude-opus-4-6", fb.Model)
	}
	if math.Abs(fb.TotalCost-3.14) > 0.01 {
		t.Errorf("total_cost: got %f, want 3.14", fb.TotalCost)
	}
}

func TestParseFeedbackGradeF(t *testing.T) {
	data := `{"health":{"grade":"F"},"context":{"percent":90,"turns_remaining":2},"cost":{"model":"opus","total_cost":50}}`
	fb := parseFeedback([]byte(data))
	if fb == nil {
		t.Fatal("expected non-nil feedback")
	}
	if fb.ChainHealthy {
		t.Error("chain_healthy: expected false for grade F")
	}
}

func TestParseFeedbackInvalidJSON(t *testing.T) {
	fb := parseFeedback([]byte("not json"))
	if fb != nil {
		t.Error("expected nil for invalid JSON")
	}
}

func TestParseDecisionEcon(t *testing.T) {
	de := parseDecisionEcon([]byte(fixtureStatsJSON))
	if de == nil {
		t.Fatal("expected non-nil decision economics")
	}

	if math.Abs(de.CPD-0.18) > 0.001 {
		t.Errorf("cpd: got %f, want 0.18", de.CPD)
	}
	if de.TTC != 5 {
		t.Errorf("ttc: got %d, want 5", de.TTC)
	}
	if math.Abs(de.CDR-0.12) > 0.001 {
		t.Errorf("cdr: got %f, want 0.12", de.CDR)
	}
	if de.TotalDecisions != 8 {
		t.Errorf("total_decisions: got %d, want 8", de.TotalDecisions)
	}
	if len(de.PerEpoch) != 2 {
		t.Fatalf("per_epoch: got %d, want 2", len(de.PerEpoch))
	}
	if de.PerEpoch[0].Epoch != 0 || de.PerEpoch[0].Decisions != 3 {
		t.Errorf("epoch 0: got %+v", de.PerEpoch[0])
	}
}

func TestParseDecisionEconEmpty(t *testing.T) {
	data := `{"decision_economics":{"cpd":0,"ttc":0,"cdr":0,"total_decisions":0}}`
	de := parseDecisionEcon([]byte(data))
	if de != nil {
		t.Error("expected nil for empty decision economics")
	}
}

func TestParseDecisionEconInvalidJSON(t *testing.T) {
	de := parseDecisionEcon([]byte("garbage"))
	if de != nil {
		t.Error("expected nil for invalid JSON")
	}
}

func TestParseDecisionEconNoField(t *testing.T) {
	data := `{"health":{"grade":"A"}}`
	de := parseDecisionEcon([]byte(data))
	if de != nil {
		t.Error("expected nil when decision_economics is absent")
	}
}
