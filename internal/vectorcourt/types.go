package vectorcourt

import "encoding/json"

// CaseFiling mirrors VectorCourt's schema.CaseFiling for the /v1/consult API.
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

// ConsultResponse wraps the raw JSON envelope from VectorCourt.
// VP passes this through without parsing verdict internals.
type ConsultResponse struct {
	Raw json.RawMessage
}

// PreflightResult mirrors VectorCourt's engine.PreflightResult.
type PreflightResult struct {
	Verdict       string   `json:"verdict"`
	Reason        string   `json:"reason,omitempty"`
	Tier          string   `json:"tier,omitempty"`
	FilingQuality float64  `json:"filing_quality,omitempty"`
	Warnings      []string `json:"warnings,omitempty"`
}

// AccountStatus mirrors VectorCourt's GET /v1/account response.
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

// OutcomeRequest is the JSON body for POST /v1/cases/:id/outcome.
type OutcomeRequest struct {
	Result string `json:"result"` // success, failure, partial
	Note   string `json:"note,omitempty"`
}

// OutcomeResponse is the response from the outcome endpoint.
type OutcomeResponse struct {
	CaseID  string `json:"case_id"`
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// PredictionDebt mirrors VectorCourt's GET /v1/health/prediction-debt response.
type PredictionDebt struct {
	OpenPredictions    int     `json:"open_predictions"`
	OverduePredictions int     `json:"overdue_predictions"`
	OldestOpenDays     int     `json:"oldest_open_days"`
	DebtRatio          float64 `json:"debt_ratio"`
	Band               string  `json:"band"` // healthy, accumulating, critical
	Note               string  `json:"note,omitempty"`
}

// InstantPrecedentMatch is a single match from the instant precedent endpoint.
type InstantPrecedentMatch struct {
	CaseID     string  `json:"case_id"`
	Question   string  `json:"question"`
	Similarity float64 `json:"similarity"`
	Verdict    string  `json:"verdict"`
	AgeDays    int     `json:"age_days"`
}

// InstantPrecedentResult is the response from GET /v1/precedents.
type InstantPrecedentResult struct {
	Matches       []InstantPrecedentMatch `json:"matches"`
	MatchCount    int                     `json:"match_count"`
	TopSimilarity float64                 `json:"top_similarity"`
	Note          string                  `json:"note,omitempty"`
}

// SparEvent is a single SSE event from the live spar stream.
type SparEvent struct {
	ID        int    `json:"id"`
	Stage     string `json:"stage"`
	Message   string `json:"message"`
	Persona   string `json:"persona,omitempty"`
	Timestamp string `json:"timestamp"`
	Final     bool   `json:"final"`
}

// EscalationDecision is the council's escalation disposition from the verdict.
type EscalationDecision struct {
	Mode      string                  `json:"mode"` // auto_resolve, resolve_with_caveat, human_clarification, human_approval
	Score     float64                 `json:"score"`
	Triggers  []string                `json:"triggers,omitempty"`
	Questions []ClarificationQuestion `json:"questions,omitempty"`
}

// ClarificationQuestion is a structured question the council asks the human.
type ClarificationQuestion struct {
	ID                  string `json:"id"`
	Type                string `json:"type"` // constraint_missing, constraint_conflict, preference, stakes_confirm, approval, falsification
	Question            string `json:"question"`
	Context             string `json:"context"`
	ConstraintField     string `json:"constraint_field,omitempty"`
	DefaultIfUnanswered string `json:"default_if_unanswered,omitempty"`
	ImpactOnVerdict     string `json:"impact_on_verdict"` // high, medium, low
}

// ClarificationAnswer captures the human's response to one clarification question.
type ClarificationAnswer struct {
	QuestionID string `json:"question_id"`
	Answer     string `json:"answer"`
	Confidence string `json:"confidence"` // firm, preferred, uncertain
	Notes      string `json:"notes,omitempty"`
}

// ClarifyRequest is the JSON body for POST /v1/cases/:id/clarify.
type ClarifyRequest struct {
	Answers []ClarificationAnswer `json:"answers"`
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
