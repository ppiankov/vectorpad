package negativespace

import (
	"regexp"
	"strings"
)

// GapClass identifies a category of missing constraint.
type GapClass string

const (
	GapPreservation  GapClass = "preservation"   // destructive verbs without protection clauses
	GapSuccessCrit   GapClass = "success"        // action without measurable outcome
	GapReviewProcess GapClass = "review"         // multi-target without review/approval
	GapRollback      GapClass = "rollback"       // destructive scope without undo/backup
	GapScopeBoundary GapClass = "scope_boundary" // broad quantifiers without exclusions
	GapIdentity      GapClass = "identity"       // content-touching verbs without voice/style constraints
)

// Gap represents a single detected missing constraint class.
type Gap struct {
	Class       GapClass `json:"class"`
	Signal      string   `json:"signal"`       // what triggered the gap (the verb/scope found)
	Description string   `json:"description"`  // human-readable explanation
	NudgePrompt string   `json:"nudge_prompt"` // suggested question to ask
}

// Result holds the negative space analysis output.
type Result struct {
	Gaps          []Gap `json:"gaps"`
	ActionSignals int   `json:"action_signals"` // count of action verbs detected
	ScopeSignals  int   `json:"scope_signals"`  // count of scope markers detected
}

// Clean returns true when no gaps were detected.
func (r Result) Clean() bool {
	return len(r.Gaps) == 0
}

// Analyze detects missing constraint classes in a directive.
// It cross-references detected actions and scope against present constraints.
func Analyze(text string) Result {
	lower := strings.ToLower(text)

	actions := detectActions(lower)
	scope := detectScope(lower)
	constraints := detectConstraints(lower)

	var gaps []Gap

	// Preservation gap: destructive verbs without protection clauses.
	if hasAny(actions, destructiveVerbs) && !constraints.hasPreservation {
		gaps = append(gaps, Gap{
			Class:       GapPreservation,
			Signal:      firstMatch(actions, destructiveVerbs),
			Description: "Destructive action without preservation constraints",
			NudgePrompt: "What must NOT change? Name specific files, sections, or properties to protect.",
		})
	}

	// Success criteria gap: action verbs without measurable outcome.
	if len(actions) > 0 && !constraints.hasSuccessCriteria {
		gaps = append(gaps, Gap{
			Class:       GapSuccessCrit,
			Signal:      actions[0],
			Description: "Action without success criteria",
			NudgePrompt: "What does 'done' look like? Describe the expected outcome or acceptance test.",
		})
	}

	// Review process gap: multi-target scope without review/approval language.
	if len(scope) > 0 && !constraints.hasReviewProcess {
		gaps = append(gaps, Gap{
			Class:       GapReviewProcess,
			Signal:      strings.Join(scope, ", "),
			Description: "Multiple targets without review process",
			NudgePrompt: "Will you review each target before applying, or apply all at once?",
		})
	}

	// Rollback gap: destructive action + scope without undo mention.
	if hasAny(actions, destructiveVerbs) && len(scope) > 0 && !constraints.hasRollback {
		gaps = append(gaps, Gap{
			Class:       GapRollback,
			Signal:      firstMatch(actions, destructiveVerbs),
			Description: "Destructive scope without rollback plan",
			NudgePrompt: "How do you undo this if it goes wrong? Is there a backup or dry-run option?",
		})
	}

	// Scope boundary gap: broad quantifiers without exclusions.
	if hasBroadQuantifier(lower) && !constraints.hasExclusions {
		gaps = append(gaps, Gap{
			Class:       GapScopeBoundary,
			Signal:      firstBroadQuantifier(lower),
			Description: "Broad scope without exclusions or boundaries",
			NudgePrompt: "Does 'all' really mean all? List any exceptions or directories to skip.",
		})
	}

	// Identity gap: content-touching verbs without voice/style constraints.
	if hasAny(actions, contentVerbs) && !constraints.hasIdentity {
		gaps = append(gaps, Gap{
			Class:       GapIdentity,
			Signal:      firstMatch(actions, contentVerbs),
			Description: "Content modification without voice or style constraints",
			NudgePrompt: "What should the result sound like? Preserve existing voice, match a reference, or rewrite freely?",
		})
	}

	return Result{
		Gaps:          gaps,
		ActionSignals: len(actions),
		ScopeSignals:  len(scope),
	}
}

// Action verb categories.
var destructiveVerbs = []string{
	"clean", "remove", "delete", "drop", "replace", "rewrite",
	"overwrite", "strip", "purge", "nuke", "wipe",
}

var contentVerbs = []string{
	"rewrite", "update", "edit", "revise", "rephrase",
	"clean up", "standardize", "align", "normalize", "format",
}

// All action verbs (union of categories).
var actionPatterns []*regexp.Regexp

func init() {
	seen := make(map[string]bool)
	var allVerbs []string
	for _, v := range destructiveVerbs {
		if !seen[v] {
			seen[v] = true
			allVerbs = append(allVerbs, v)
		}
	}
	for _, v := range contentVerbs {
		if !seen[v] {
			seen[v] = true
			allVerbs = append(allVerbs, v)
		}
	}
	// Add general action verbs.
	for _, v := range []string{
		"refactor", "migrate", "move", "rename", "convert",
		"fix", "patch", "improve", "simplify", "tidy", "organize",
	} {
		if !seen[v] {
			seen[v] = true
			allVerbs = append(allVerbs, v)
		}
	}
	for _, v := range allVerbs {
		actionPatterns = append(actionPatterns, regexp.MustCompile(`(?i)\b`+regexp.QuoteMeta(v)+`\b`))
	}
}

func detectActions(lower string) []string {
	var found []string
	seen := make(map[string]bool)
	for _, p := range actionPatterns {
		if m := p.FindString(lower); m != "" {
			word := strings.ToLower(m)
			if !seen[word] {
				seen[word] = true
				found = append(found, word)
			}
		}
	}
	return found
}

// Scope markers — multi-word phrases indicating broad targets.
var scopePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\ball\s+repos?\b`),
	regexp.MustCompile(`(?i)\bevery\s+repo\b`),
	regexp.MustCompile(`(?i)\bacross\s+(all\s+)?repos?\b`),
	regexp.MustCompile(`(?i)\bacross\s+the\s+codebase\b`),
	regexp.MustCompile(`(?i)\bacross\s+(all\s+)?projects?\b`),
	regexp.MustCompile(`(?i)\ball\s+files?\b`),
	regexp.MustCompile(`(?i)\bevery\s+file\b`),
	regexp.MustCompile(`(?i)\ball\s+readmes?\b`),
	regexp.MustCompile(`(?i)\ball\s+packages?\b`),
	regexp.MustCompile(`(?i)\ball\s+services?\b`),
	regexp.MustCompile(`(?i)\beverywhere\b`),
	regexp.MustCompile(`(?i)\b\d+\s+repos?\b`),
}

func detectScope(lower string) []string {
	var found []string
	for _, p := range scopePatterns {
		if m := p.FindString(lower); m != "" {
			found = append(found, strings.TrimSpace(m))
		}
	}
	return found
}

// Broad quantifier detection.
var broadQuantifiers = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\ball\b`),
	regexp.MustCompile(`(?i)\bevery\b`),
	regexp.MustCompile(`(?i)\beverywhere\b`),
	regexp.MustCompile(`(?i)\beach\s+(?:repo|file|project|service|package)\b`),
}

func hasBroadQuantifier(lower string) bool {
	for _, p := range broadQuantifiers {
		if p.MatchString(lower) {
			return true
		}
	}
	return false
}

func firstBroadQuantifier(lower string) string {
	for _, p := range broadQuantifiers {
		if m := p.FindString(lower); m != "" {
			return strings.TrimSpace(m)
		}
	}
	return ""
}

// Constraint detection — what protection clauses are present.
type constraintSet struct {
	hasPreservation    bool
	hasSuccessCriteria bool
	hasReviewProcess   bool
	hasRollback        bool
	hasExclusions      bool
	hasIdentity        bool
}

var (
	preservationPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bpreserve\b`),
		regexp.MustCompile(`(?i)\bkeep\b`),
		regexp.MustCompile(`(?i)\bdon'?t\s+change\b`),
		regexp.MustCompile(`(?i)\bdo\s+not\s+change\b`),
		regexp.MustCompile(`(?i)\bdon'?t\s+touch\b`),
		regexp.MustCompile(`(?i)\bdo\s+not\s+touch\b`),
		regexp.MustCompile(`(?i)\bdon'?t\s+remove\b`),
		regexp.MustCompile(`(?i)\bdo\s+not\s+remove\b`),
		regexp.MustCompile(`(?i)\bleave\s+.*\s+alone\b`),
		regexp.MustCompile(`(?i)\bmust\s+stay\b`),
		regexp.MustCompile(`(?i)\bprotect\b`),
	}

	successPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bshould\s+(?:look|be|have|contain|produce|output|result)\b`),
		regexp.MustCompile(`(?i)\bexpect(?:ed)?\s+(?:output|result|outcome)\b`),
		regexp.MustCompile(`(?i)\bacceptance\s+criter\b`),
		regexp.MustCompile(`(?i)\bdone\s+when\b`),
		regexp.MustCompile(`(?i)\bsuccess\s+(?:looks|means|is)\b`),
		regexp.MustCompile(`(?i)\bverif(?:y|ied)\b`),
		regexp.MustCompile(`(?i)\btest\s+(?:that|by|with)\b`),
	}

	reviewPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\breview\b`),
		regexp.MustCompile(`(?i)\bapprov(?:e|al)\b`),
		regexp.MustCompile(`(?i)\bdiff\s+(?:each|per|before)\b`),
		regexp.MustCompile(`(?i)\bone\s+(?:at\s+a\s+time|by\s+one)\b`),
		regexp.MustCompile(`(?i)\bper[- ]repo\b`),
		regexp.MustCompile(`(?i)\bindividually\b`),
		regexp.MustCompile(`(?i)\bdry[- ]?run\b`),
	}

	rollbackPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bundo\b`),
		regexp.MustCompile(`(?i)\brevert\b`),
		regexp.MustCompile(`(?i)\brollback\b`),
		regexp.MustCompile(`(?i)\broll\s+back\b`),
		regexp.MustCompile(`(?i)\bbackup\b`),
		regexp.MustCompile(`(?i)\bback\s+up\b`),
		regexp.MustCompile(`(?i)\brestore\b`),
		regexp.MustCompile(`(?i)\bdry[- ]?run\b`),
		regexp.MustCompile(`(?i)\bgit\s+stash\b`),
	}

	exclusionPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bexcept\b`),
		regexp.MustCompile(`(?i)\bexclud(?:e|ing)\b`),
		regexp.MustCompile(`(?i)\bbut\s+not\b`),
		regexp.MustCompile(`(?i)\bskip\b`),
		regexp.MustCompile(`(?i)\bignor(?:e|ing)\b`),
		regexp.MustCompile(`(?i)\bunless\b`),
		regexp.MustCompile(`(?i)\bnot\s+including\b`),
	}

	identityPatterns = []*regexp.Regexp{
		regexp.MustCompile(`(?i)\bvoice\b`),
		regexp.MustCompile(`(?i)\btone\b`),
		regexp.MustCompile(`(?i)\bstyle\b`),
		regexp.MustCompile(`(?i)\bpersonality\b`),
		regexp.MustCompile(`(?i)\bbranding\b`),
		regexp.MustCompile(`(?i)\bmatch\s+(?:the|existing|current)\b`),
		regexp.MustCompile(`(?i)\bsound\s+like\b`),
		regexp.MustCompile(`(?i)\bkeep\s+the\s+(?:same|existing)\b`),
	}
)

func detectConstraints(lower string) constraintSet {
	return constraintSet{
		hasPreservation:    matchesAny(lower, preservationPatterns),
		hasSuccessCriteria: matchesAny(lower, successPatterns),
		hasReviewProcess:   matchesAny(lower, reviewPatterns),
		hasRollback:        matchesAny(lower, rollbackPatterns),
		hasExclusions:      matchesAny(lower, exclusionPatterns),
		hasIdentity:        matchesAny(lower, identityPatterns),
	}
}

func matchesAny(text string, patterns []*regexp.Regexp) bool {
	for _, p := range patterns {
		if p.MatchString(text) {
			return true
		}
	}
	return false
}

func hasAny(found []string, targets []string) bool {
	set := make(map[string]bool, len(targets))
	for _, t := range targets {
		set[t] = true
	}
	for _, f := range found {
		if set[f] {
			return true
		}
	}
	return false
}

func firstMatch(found []string, targets []string) string {
	set := make(map[string]bool, len(targets))
	for _, t := range targets {
		set[t] = true
	}
	for _, f := range found {
		if set[f] {
			return f
		}
	}
	return ""
}
