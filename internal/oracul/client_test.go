package oracul

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestConsultSuccess(t *testing.T) {
	envelope := map[string]string{"status": "completed", "verdict": "proceed"}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/consult" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Header.Get("X-Oracul-Key") != "test_key" {
			t.Errorf("auth header = %q", r.Header.Get("X-Oracul-Key"))
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(envelope)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test_key")
	raw, err := client.Consult(context.Background(), &ConsultRequest{
		Question: "Should we use Kafka?",
		Filing: &CaseFiling{
			Decision:    "Use Kafka for messaging",
			Constraints: []string{"Must handle 10k msg/s"},
		},
	})
	if err != nil {
		t.Fatalf("Consult: %v", err)
	}

	var result map[string]string
	if err := json.Unmarshal(raw, &result); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if result["status"] != "completed" {
		t.Errorf("status = %q", result["status"])
	}
}

func TestConsultAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid key"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "bad_key")
	_, err := client.Consult(context.Background(), &ConsultRequest{Question: "test"})
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("status = %d, want 401", apiErr.StatusCode)
	}
}

func TestConsultRateLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key")
	_, err := client.Consult(context.Background(), &ConsultRequest{Question: "test"})
	if err == nil {
		t.Fatal("expected error")
	}

	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 429 {
		t.Errorf("status = %d, want 429", apiErr.StatusCode)
	}
}

func TestAccountSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/account" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %q", r.Method)
		}
		if r.Header.Get("X-Oracul-Key") != "test_key" {
			t.Errorf("auth header = %q", r.Header.Get("X-Oracul-Key"))
		}
		_ = json.NewEncoder(w).Encode(AccountStatus{
			Tier:             "standard",
			SubmissionsToday: 7,
			DailyLimit:       15,
			ResetsAt:         "2026-03-11T00:00:00Z",
			Active:           true,
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test_key")
	status, err := client.Account(context.Background())
	if err != nil {
		t.Fatalf("Account: %v", err)
	}
	if status.Tier != "standard" {
		t.Errorf("tier = %q", status.Tier)
	}
	if status.SubmissionsToday != 7 {
		t.Errorf("submissions_today = %d", status.SubmissionsToday)
	}
	if status.DailyLimit != 15 {
		t.Errorf("daily_limit = %d", status.DailyLimit)
	}
	if !status.Active {
		t.Error("expected active=true")
	}
}

func TestAccountUnauthorized(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid key"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "bad_key")
	_, err := client.Account(context.Background())
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("status = %d, want 401", apiErr.StatusCode)
	}
}

func TestSearchPrecedentsSuccess(t *testing.T) {
	boolTrue := true
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/precedents/search" {
			t.Errorf("path = %q", r.URL.Path)
		}
		if r.Method != http.MethodGet {
			t.Errorf("method = %q", r.Method)
		}
		if r.URL.Query().Get("q") != "Should we use Kafka?" {
			t.Errorf("q = %q", r.URL.Query().Get("q"))
		}
		if r.URL.Query().Get("limit") != "3" {
			t.Errorf("limit = %q", r.URL.Query().Get("limit"))
		}
		_ = json.NewEncoder(w).Encode(PrecedentSearch{
			Precedents: []PrecedentResult{
				{
					CaseID:          "case-001",
					Question:        "Should we use Kafka for events?",
					SimilarityScore: 0.72,
					Confidence:      0.85,
					OutcomeCount:    3,
					Predictions: []PrecedentPrediction{
						{Statement: "ops burden < 10h/month", Probability: 0.8, Resolved: true, Correct: &boolTrue},
					},
				},
				{
					CaseID:          "case-002",
					Question:        "RabbitMQ vs SQS",
					SimilarityScore: 0.58,
					Confidence:      0.70,
				},
			},
			TotalSimilar: 12,
			RefClassSummary: &RefClassSummary{
				TotalCases:    12,
				ResolvedCases: 8,
				SuccessRate:   0.625,
			},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "test_key")
	result, err := client.SearchPrecedents(context.Background(), "Should we use Kafka?", 3)
	if err != nil {
		t.Fatalf("SearchPrecedents: %v", err)
	}
	if len(result.Precedents) != 2 {
		t.Errorf("precedents count = %d, want 2", len(result.Precedents))
	}
	if result.TotalSimilar != 12 {
		t.Errorf("total_similar = %d", result.TotalSimilar)
	}
	if result.RefClassSummary == nil {
		t.Fatal("expected ref class summary")
	}
	if result.RefClassSummary.SuccessRate != 0.625 {
		t.Errorf("success_rate = %f", result.RefClassSummary.SuccessRate)
	}
	if result.Precedents[0].SimilarityScore != 0.72 {
		t.Errorf("similarity = %f", result.Precedents[0].SimilarityScore)
	}
}

func TestSearchPrecedentsEmpty(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(PrecedentSearch{
			Precedents:   []PrecedentResult{},
			TotalSimilar: 0,
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key")
	result, err := client.SearchPrecedents(context.Background(), "test", 5)
	if err != nil {
		t.Fatalf("SearchPrecedents: %v", err)
	}
	if len(result.Precedents) != 0 {
		t.Errorf("expected empty, got %d", len(result.Precedents))
	}
}

func TestSearchPrecedentsAuthError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid key"})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "bad_key")
	_, err := client.SearchPrecedents(context.Background(), "test", 5)
	if err == nil {
		t.Fatal("expected error")
	}
	apiErr, ok := err.(*APIError)
	if !ok {
		t.Fatalf("expected *APIError, got %T", err)
	}
	if apiErr.StatusCode != 401 {
		t.Errorf("status = %d, want 401", apiErr.StatusCode)
	}
}

func TestPreflightSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/preflight" {
			t.Errorf("path = %q", r.URL.Path)
		}
		_ = json.NewEncoder(w).Encode(PreflightResult{
			Verdict:       "ACCEPTED",
			Tier:          "standard",
			FilingQuality: 0.85,
			Warnings:      []string{"no success criteria"},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key")
	result, err := client.Preflight(context.Background(), "Should we use Kafka?", &CaseFiling{
		Decision: "Use Kafka",
	})
	if err != nil {
		t.Fatalf("Preflight: %v", err)
	}
	if result.Verdict != "ACCEPTED" {
		t.Errorf("verdict = %q", result.Verdict)
	}
	if result.Tier != "standard" {
		t.Errorf("tier = %q", result.Tier)
	}
	if len(result.Warnings) != 1 {
		t.Errorf("warnings count = %d", len(result.Warnings))
	}
}

func TestPreflightGateAccepted(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(PreflightResult{
			Verdict:       "ACCEPTED",
			Tier:          "simple",
			FilingQuality: 0.95,
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key")
	gate, err := client.PreflightGate(context.Background(), "test?", &CaseFiling{Decision: "test"})
	if err != nil {
		t.Fatalf("PreflightGate: %v", err)
	}
	if !gate.Allowed {
		t.Error("expected Allowed=true")
	}
	if gate.Tier != "simple" {
		t.Errorf("tier = %q", gate.Tier)
	}
}

func TestPreflightGateRejected(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(PreflightResult{
			Verdict: "REJECTED",
			Reason:  "topic refusal: medical advice",
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key")
	gate, err := client.PreflightGate(context.Background(), "test?", &CaseFiling{Decision: "test"})
	if err != nil {
		t.Fatalf("PreflightGate: %v", err)
	}
	if gate.Allowed {
		t.Error("expected Allowed=false")
	}
	if gate.Reason != "topic refusal: medical advice" {
		t.Errorf("reason = %q", gate.Reason)
	}
}

func TestPreflightGateWithWarnings(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(PreflightResult{
			Verdict:  "ACCEPTED",
			Tier:     "standard",
			Warnings: []string{"no constraints", "no success criteria"},
		})
	}))
	defer srv.Close()

	client := NewClient(srv.URL, "key")
	gate, err := client.PreflightGate(context.Background(), "test?", &CaseFiling{Decision: "test"})
	if err != nil {
		t.Fatalf("PreflightGate: %v", err)
	}
	if !gate.Allowed {
		t.Error("expected Allowed=true with warnings")
	}
	if len(gate.Warnings) != 2 {
		t.Errorf("warnings count = %d, want 2", len(gate.Warnings))
	}
}
