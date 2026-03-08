package detect

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"time"
)

const feedbackTimeout = 2 * time.Second

// Feedback holds quick telemetry from contextspectre status-line.
type Feedback struct {
	Grade          string  `json:"grade"`
	ContextPercent float64 `json:"context_percent"`
	TurnsRemaining int     `json:"turns_remaining"`
	ChainHealthy   bool    `json:"chain_healthy"`
	Model          string  `json:"model"`
	TotalCost      float64 `json:"total_cost"`
}

// DecisionEcon holds actual CPD/TTC/CDR from contextspectre stats.
type DecisionEcon struct {
	CPD            float64         `json:"cpd"`
	TTC            int             `json:"ttc"`
	CDR            float64         `json:"cdr"`
	TotalDecisions int             `json:"total_decisions"`
	PerEpoch       []EpochDecision `json:"per_epoch,omitempty"`
}

// EpochDecision holds per-epoch decision metrics.
type EpochDecision struct {
	Epoch     int     `json:"epoch"`
	CPD       float64 `json:"cpd"`
	TTC       int     `json:"ttc"`
	CDR       float64 `json:"cdr"`
	Decisions int     `json:"decisions"`
}

// ReadFeedback runs contextspectre stats --cwd --format json and extracts
// quick telemetry (health, context, cost). Returns nil if unavailable.
func ReadFeedback(caps Capabilities) *Feedback {
	if !caps.ContextSpec {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), feedbackTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, caps.ContextBin, "stats", "--cwd", "--format", "json")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil
	}

	return parseFeedback(stdout.Bytes())
}

// ReadDecisionEconomics runs contextspectre stats --cwd --format json and
// extracts the decision_economics section. Returns nil if unavailable.
func ReadDecisionEconomics(caps Capabilities) *DecisionEcon {
	if !caps.ContextSpec {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), feedbackTimeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, caps.ContextBin, "stats", "--cwd", "--format", "json")
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	if err := cmd.Run(); err != nil {
		return nil
	}

	return parseDecisionEcon(stdout.Bytes())
}

func parseFeedback(data []byte) *Feedback {
	var raw struct {
		Health struct {
			Grade string `json:"grade"`
		} `json:"health"`
		Context struct {
			Percent        float64 `json:"percent"`
			TurnsRemaining int     `json:"turns_remaining"`
		} `json:"context"`
		Cost struct {
			Model     string  `json:"model"`
			TotalCost float64 `json:"total_cost"`
		} `json:"cost"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	return &Feedback{
		Grade:          raw.Health.Grade,
		ContextPercent: raw.Context.Percent,
		TurnsRemaining: raw.Context.TurnsRemaining,
		Model:          raw.Cost.Model,
		TotalCost:      raw.Cost.TotalCost,
		ChainHealthy:   raw.Health.Grade != "F",
	}
}

func parseDecisionEcon(data []byte) *DecisionEcon {
	var raw struct {
		DE struct {
			CPD            float64 `json:"cpd"`
			TTC            int     `json:"ttc"`
			CDR            float64 `json:"cdr"`
			TotalDecisions int     `json:"total_decisions"`
			PerEpoch       []struct {
				Epoch     int     `json:"epoch"`
				CPD       float64 `json:"cpd"`
				TTC       int     `json:"ttc"`
				CDR       float64 `json:"cdr"`
				Decisions int     `json:"decisions"`
			} `json:"per_epoch"`
		} `json:"decision_economics"`
	}

	if err := json.Unmarshal(data, &raw); err != nil {
		return nil
	}

	if raw.DE.TotalDecisions == 0 && raw.DE.CPD == 0 {
		return nil // no decision data
	}

	econ := &DecisionEcon{
		CPD:            raw.DE.CPD,
		TTC:            raw.DE.TTC,
		CDR:            raw.DE.CDR,
		TotalDecisions: raw.DE.TotalDecisions,
	}

	for _, ep := range raw.DE.PerEpoch {
		econ.PerEpoch = append(econ.PerEpoch, EpochDecision{
			Epoch:     ep.Epoch,
			CPD:       ep.CPD,
			TTC:       ep.TTC,
			CDR:       ep.CDR,
			Decisions: ep.Decisions,
		})
	}

	return econ
}
