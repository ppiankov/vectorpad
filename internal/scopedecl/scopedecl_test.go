package scopedecl

import (
	"testing"
)

func TestParseEmpty(t *testing.T) {
	d := Parse("")
	if !d.Empty() {
		t.Error("expected empty declaration for empty input")
	}
}

func TestParseFullDeclaration(t *testing.T) {
	block := `scope: 18 repos
operation: cleanup
targets: README.md, CONTRIBUTING.md
files: 36`
	d := Parse(block)

	if d.Repos != 18 {
		t.Errorf("expected 18 repos, got %d", d.Repos)
	}
	if d.Files != 36 {
		t.Errorf("expected 36 files, got %d", d.Files)
	}
	if d.Operation != "cleanup" {
		t.Errorf("expected operation 'cleanup', got %q", d.Operation)
	}
	if len(d.Targets) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(d.Targets))
	}
	if d.Targets[0] != "README.md" || d.Targets[1] != "CONTRIBUTING.md" {
		t.Errorf("unexpected targets: %v", d.Targets)
	}
}

func TestCrossReferenceClean(t *testing.T) {
	decl := Declaration{Repos: 18, Operation: "cleanup"}
	text := "Clean up badge formatting in each repo. Preserve philosophy sections and keep existing voice. Review each repo individually."
	result := CrossReference(decl, text)
	if !result.Clean() {
		for _, m := range result.Mismatches {
			t.Errorf("unexpected mismatch: [%s] %s", m.Type, m.Description)
		}
	}
}

func TestCrossReferenceScopeVsConstraints(t *testing.T) {
	decl := Declaration{Repos: 18, Operation: "cleanup"}
	text := "clean up READMEs for alignment"
	result := CrossReference(decl, text)

	if result.Clean() {
		t.Fatal("expected mismatches for unconstrained multi-repo directive")
	}

	types := mismatchTypes(result)
	if !contains(types, "scope_vs_constraints") {
		t.Errorf("expected scope_vs_constraints mismatch, got: %v", types)
	}
	if !contains(types, "operation_vs_preservation") {
		t.Errorf("expected operation_vs_preservation mismatch, got: %v", types)
	}
}

func TestCrossReferenceOperationVsVerbs(t *testing.T) {
	decl := Declaration{Operation: "cleanup"}
	text := "rewrite all documentation from scratch"
	result := CrossReference(decl, text)

	types := mismatchTypes(result)
	if !contains(types, "operation_vs_verbs") {
		t.Errorf("expected operation_vs_verbs mismatch, got: %v", types)
	}
}

func TestCrossReferenceTargetNotMentioned(t *testing.T) {
	decl := Declaration{Targets: []string{"README.md", "CHANGELOG.md"}}
	text := "update the README to include badges"
	result := CrossReference(decl, text)

	// CHANGELOG.md not mentioned in text.
	found := false
	for _, m := range result.Mismatches {
		if m.Type == "target_not_mentioned" && m.Declared == "target: CHANGELOG.md" {
			found = true
		}
	}
	if !found {
		t.Error("expected target_not_mentioned for CHANGELOG.md")
	}
}

func TestCrossReferenceEmptyDeclaration(t *testing.T) {
	decl := Declaration{}
	result := CrossReference(decl, "do anything")
	if !result.Clean() {
		t.Error("expected clean result for empty declaration")
	}
}

func TestCrossReferenceOperationMatchesSynonym(t *testing.T) {
	decl := Declaration{Operation: "cleanup"}
	text := "clean the badge formatting across repos. Preserve voice."
	result := CrossReference(decl, text)

	// Should NOT have operation_vs_verbs since "clean" matches "cleanup".
	for _, m := range result.Mismatches {
		if m.Type == "operation_vs_verbs" {
			t.Error("should not have operation_vs_verbs when verb matches operation synonym")
		}
	}
}

// --- helpers ---

func mismatchTypes(r Result) []string {
	var types []string
	for _, m := range r.Mismatches {
		types = append(types, m.Type)
	}
	return types
}

func contains(items []string, target string) bool {
	for _, i := range items {
		if i == target {
			return true
		}
	}
	return false
}
