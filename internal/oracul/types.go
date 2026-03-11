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

// GateResult is the outcome of a preflight gate check.
type GateResult struct {
	Allowed  bool
	Verdict  string
	Tier     string
	Quality  float64
	Warnings []string
	Reason   string
}
