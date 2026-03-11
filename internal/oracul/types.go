package oracul

import "encoding/json"

// CaseFiling mirrors Oracul's schema.CaseFiling for the /v1/consult API.
// Kept in sync manually — no cross-module import.
type CaseFiling struct {
	Decision        string   `json:"decision"`
	Context         string   `json:"context,omitempty"`
	Constraints     []string `json:"constraints,omitempty"`
	Alternatives    []string `json:"alternatives,omitempty"`
	KnownRisks      []string `json:"known_risks,omitempty"`
	SuccessCriteria string   `json:"success_criteria,omitempty"`
	Evidence        []string `json:"evidence,omitempty"`
}

// ConsultRequest is the JSON body for POST /v1/consult.
type ConsultRequest struct {
	Question string      `json:"question"`
	Filing   *CaseFiling `json:"filing,omitempty"`
}

// ConsultResponse wraps the raw JSON envelope from Oracul.
// VP passes this through without parsing verdict internals.
type ConsultResponse struct {
	Raw json.RawMessage
}

// PreflightResult mirrors Oracul's engine.PreflightResult.
type PreflightResult struct {
	Verdict       string   `json:"verdict"`
	Reason        string   `json:"reason,omitempty"`
	Tier          string   `json:"tier,omitempty"`
	FilingQuality float64  `json:"filing_quality,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
}

// AccountStatus mirrors Oracul's GET /v1/account response.
type AccountStatus struct {
	Tier             string `json:"tier"`
	SubmissionsToday int    `json:"submissions_today"`
	DailyLimit       int    `json:"daily_limit"`
	ResetsAt         string `json:"resets_at"`
	Active           bool   `json:"active"`
}

// PrecedentPrediction is a single prediction attached to a precedent case.
type PrecedentPrediction struct {
	Statement   string  `json:"statement"`
	Probability float64 `json:"probability"`
	Resolved    bool    `json:"resolved"`
	Correct     *bool   `json:"correct,omitempty"`
	ActualValue string  `json:"actual_value,omitempty"`
}

// PrecedentResult is a single precedent case returned by the search endpoint.
type PrecedentResult struct {
	CaseID             string                `json:"case_id"`
	Question           string                `json:"question"`
	VerdictStatus      string                `json:"verdict_status"`
	Confidence         float64               `json:"confidence"`
	CreatedAt          string                `json:"created_at"`
	SimilarityScore    float64               `json:"similarity_score"`
	Predictions        []PrecedentPrediction `json:"predictions,omitempty"`
	OutcomeCount       int                   `json:"outcome_count"`
	OutcomeCorrectRate float64               `json:"outcome_correct_rate"`
	ClaimFamilies      []string              `json:"claim_families,omitempty"`
}

// RefClassSummary summarizes the reference class for a precedent search.
type RefClassSummary struct {
	TotalCases       int      `json:"total_cases"`
	ResolvedCases    int      `json:"resolved_cases"`
	SuccessRate      float64  `json:"success_rate"`
	TopClaimFamilies []string `json:"top_claim_families,omitempty"`
}

// PrecedentSearch is the response from GET /v1/precedents/search.
type PrecedentSearch struct {
	Precedents      []PrecedentResult `json:"precedents"`
	TotalSimilar    int               `json:"total_similar_cases"`
	RefClassSummary *RefClassSummary  `json:"reference_class_summary,omitempty"`
}

// GateResult is the outcome of a preflight gate check.
type GateResult struct {
	Allowed  bool
	Verdict  string
	Tier     string
	Quality  float64
	Warnings []string
	Reason   string
}
