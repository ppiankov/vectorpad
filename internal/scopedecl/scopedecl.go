package scopedecl

import (
	"regexp"
	"strconv"
	"strings"
)

// Declaration holds an operator's explicit scope declaration.
type Declaration struct {
	Repos     int      `json:"repos,omitempty"`
	Files     int      `json:"files,omitempty"`
	Targets   []string `json:"targets,omitempty"`
	Operation string   `json:"operation,omitempty"`
}

// Empty returns true when no scope has been declared.
func (d Declaration) Empty() bool {
	return d.Repos == 0 && d.Files == 0 && len(d.Targets) == 0 && d.Operation == ""
}

// Mismatch describes a gap between declared scope and directive text.
type Mismatch struct {
	Type        string `json:"type"`
	Declared    string `json:"declared"`
	Detected    string `json:"detected"`
	Description string `json:"description"`
}

// Result holds the cross-reference analysis.
type Result struct {
	Declaration Declaration `json:"declaration"`
	Mismatches  []Mismatch  `json:"mismatches,omitempty"`
}

// Clean returns true when no mismatches were found.
func (r Result) Clean() bool {
	return len(r.Mismatches) == 0
}

// Parse extracts a scope declaration from a structured block.
// Format: key-value lines like "scope: 18 repos" or "targets: README.md".
func Parse(block string) Declaration {
	var d Declaration
	for _, line := range strings.Split(block, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key, value := splitKV(line)
		if key == "" {
			continue
		}
		switch key {
		case "scope":
			d.Repos = extractNumber(value)
		case "files":
			d.Files = extractNumber(value)
		case "operation":
			d.Operation = value
		case "targets":
			for _, t := range strings.Split(value, ",") {
				t = strings.TrimSpace(t)
				if t != "" {
					d.Targets = append(d.Targets, t)
				}
			}
		}
	}
	return d
}

// CrossReference checks declared scope against directive text for mismatches.
func CrossReference(decl Declaration, text string) Result {
	if decl.Empty() {
		return Result{Declaration: decl}
	}

	lower := strings.ToLower(text)
	var mismatches []Mismatch

	// Check: declared repos but no per-repo constraints.
	if decl.Repos > 1 && !hasPerRepoConstraint(lower) {
		mismatches = append(mismatches, Mismatch{
			Type:        "scope_vs_constraints",
			Declared:    strconv.Itoa(decl.Repos) + " repos",
			Detected:    "0 per-repo constraints",
			Description: "Multiple repos declared but text has no per-repo review or constraints",
		})
	}

	// Check: declared operation is destructive but no preservation.
	if isDestructiveOp(decl.Operation) && !hasPreservation(lower) {
		mismatches = append(mismatches, Mismatch{
			Type:        "operation_vs_preservation",
			Declared:    "operation: " + decl.Operation,
			Detected:    "no preservation clauses",
			Description: "Destructive operation declared but text has no preservation constraints",
		})
	}

	// Check: declared operation vs text verbs mismatch.
	if decl.Operation != "" {
		textVerbs := detectOperationVerbs(lower)
		if len(textVerbs) > 0 && !verbMatchesOperation(decl.Operation, textVerbs) {
			mismatches = append(mismatches, Mismatch{
				Type:        "operation_vs_verbs",
				Declared:    "operation: " + decl.Operation,
				Detected:    "text uses: " + strings.Join(textVerbs, ", "),
				Description: "Declared operation doesn't match verbs used in text",
			})
		}
	}

	// Check: declared targets but text doesn't mention them.
	for _, target := range decl.Targets {
		if !strings.Contains(lower, strings.ToLower(target)) {
			mismatches = append(mismatches, Mismatch{
				Type:        "target_not_mentioned",
				Declared:    "target: " + target,
				Detected:    "not found in text",
				Description: "Declared target '" + target + "' not mentioned in directive",
			})
		}
	}

	return Result{Declaration: decl, Mismatches: mismatches}
}

func splitKV(line string) (string, string) {
	idx := strings.IndexByte(line, ':')
	if idx < 0 {
		return "", ""
	}
	return strings.TrimSpace(strings.ToLower(line[:idx])), strings.TrimSpace(line[idx+1:])
}

var numberPattern = regexp.MustCompile(`\d+`)

func extractNumber(s string) int {
	m := numberPattern.FindString(s)
	if m == "" {
		return 0
	}
	n, _ := strconv.Atoi(m)
	return n
}

var perRepoPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bper[- ]repo\b`),
	regexp.MustCompile(`(?i)\beach\s+repo\b`),
	regexp.MustCompile(`(?i)\bindividually\b`),
	regexp.MustCompile(`(?i)\bone\s+(?:at\s+a\s+time|by\s+one)\b`),
	regexp.MustCompile(`(?i)\breview\s+(?:each|every)\b`),
	regexp.MustCompile(`(?i)\bdiff\s+(?:each|per|before)\b`),
}

func hasPerRepoConstraint(lower string) bool {
	for _, p := range perRepoPatterns {
		if p.MatchString(lower) {
			return true
		}
	}
	return false
}

var preservationPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\bpreserve\b`),
	regexp.MustCompile(`(?i)\bkeep\b`),
	regexp.MustCompile(`(?i)\bdon'?t\s+change\b`),
	regexp.MustCompile(`(?i)\bdo\s+not\s+change\b`),
	regexp.MustCompile(`(?i)\bprotect\b`),
}

func hasPreservation(lower string) bool {
	for _, p := range preservationPatterns {
		if p.MatchString(lower) {
			return true
		}
	}
	return false
}

var destructiveOps = map[string]bool{
	"cleanup": true, "clean up": true, "delete": true, "remove": true,
	"rewrite": true, "replace": true, "overwrite": true, "migration": true,
	"refactor": true, "restructure": true,
}

func isDestructiveOp(op string) bool {
	return destructiveOps[strings.ToLower(op)]
}

var operationVerbPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(clean|delete|remove|rewrite|replace|overwrite|update|refactor|migrate|restructure)\b`),
}

func detectOperationVerbs(lower string) []string {
	var verbs []string
	seen := make(map[string]bool)
	for _, p := range operationVerbPatterns {
		for _, m := range p.FindAllString(lower, -1) {
			v := strings.ToLower(m)
			if !seen[v] {
				seen[v] = true
				verbs = append(verbs, v)
			}
		}
	}
	return verbs
}

// verbMatchesOperation checks if any detected verb aligns with the declared operation.
func verbMatchesOperation(operation string, verbs []string) bool {
	opLower := strings.ToLower(operation)
	for _, v := range verbs {
		if strings.Contains(opLower, v) || strings.Contains(v, opLower) {
			return true
		}
	}
	// Map related terms.
	opSynonyms := map[string][]string{
		"cleanup":     {"clean", "tidy", "organize"},
		"clean up":    {"clean", "tidy", "organize"},
		"migration":   {"migrate", "move", "convert"},
		"refactor":    {"refactor", "restructure", "simplify"},
		"restructure": {"restructure", "refactor", "reorganize"},
	}
	if synonyms, ok := opSynonyms[opLower]; ok {
		for _, v := range verbs {
			for _, s := range synonyms {
				if v == s {
					return true
				}
			}
		}
	}
	return false
}
