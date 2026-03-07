package ambiguity

import "strings"

// NudgeType identifies one structured nudge prompt.
type NudgeType string

const (
	NudgePreservationConstraint NudgeType = "preservation_constraint"
	NudgeScopeConsistency       NudgeType = "scope_consistency"
	NudgeReferenceExample       NudgeType = "reference_example"
)

// Nudge is a non-blocking prompt shown to the operator.
type Nudge struct {
	Type        NudgeType `json:"type"`
	Prompt      string    `json:"prompt"`
	Dismissable bool      `json:"dismissable"`
}

// NudgeResponse captures operator input for one nudge.
type NudgeResponse struct {
	Type      NudgeType
	Answer    string
	Dismissed bool
}

// SelectNudges returns structured nudges for ambiguous directives.
func SelectNudges(result Result) []Nudge {
	if !result.Warning.Active || result.HasPreservationConstraints {
		return nil
	}

	nudges := make([]Nudge, 0, 3)
	if result.BlastRadius.Targets > 1 {
		nudges = append(nudges, Nudge{
			Type:        NudgePreservationConstraint,
			Prompt:      "What must NOT change? Examples: voice, narrative sections, file structure, specific paragraphs.",
			Dismissable: true,
		})
	}

	if result.BlastRadius.Repos > 1 {
		nudges = append(nudges, Nudge{
			Type:        NudgeScopeConsistency,
			Prompt:      "Same transformation everywhere, or should each repo be treated individually?",
			Dismissable: true,
		})
	}

	if result.BlastRadius.Repos > 3 {
		nudges = append(nudges, Nudge{
			Type:        NudgeReferenceExample,
			Prompt:      "Show one example done correctly. The system will match the rest to this reference.",
			Dismissable: true,
		})
	}

	if len(nudges) == 0 {
		return nil
	}

	return nudges
}

// ApplyNudgeResponses appends answered nudge constraints to the directive before launch.
func ApplyNudgeResponses(directive string, responses []NudgeResponse) string {
	base := strings.TrimSpace(directive)
	if len(responses) == 0 {
		return base
	}

	responseByType := make(map[NudgeType]NudgeResponse, len(responses))
	for _, response := range responses {
		responseByType[response.Type] = response
	}

	constraintLines := []string{}
	constraintLines = appendConstraintLine(
		constraintLines,
		"Preserve",
		responseByType[NudgePreservationConstraint],
	)
	constraintLines = appendConstraintLine(
		constraintLines,
		"Scope rule",
		responseByType[NudgeScopeConsistency],
	)
	constraintLines = appendConstraintLine(
		constraintLines,
		"Reference example",
		responseByType[NudgeReferenceExample],
	)

	if len(constraintLines) == 0 {
		return base
	}

	var b strings.Builder
	b.WriteString(base)
	b.WriteString("\n\nLaunch constraints:\n")
	for _, line := range constraintLines {
		b.WriteString("- ")
		b.WriteString(line)
		b.WriteString("\n")
	}
	return strings.TrimRight(b.String(), "\n")
}

func appendConstraintLine(lines []string, label string, response NudgeResponse) []string {
	if response.Dismissed {
		return lines
	}

	answer := strings.TrimSpace(response.Answer)
	if answer == "" {
		return lines
	}

	return append(lines, label+": "+answer)
}
