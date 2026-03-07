package ambiguity

import (
	"strings"
	"testing"
)

func TestBlastRadiusCountsExplicitPathsAndScopeMarkers(t *testing.T) {
	t.Parallel()

	directive := strings.Join([]string{
		"Update ./cmd/vectorpad/main.go and internal/preflight/metrics.go",
		"across all repos and every README.",
	}, " ")

	result := Analyze(directive, Scope{})

	if len(result.BlastRadius.ExplicitPaths) != 2 {
		t.Fatalf("expected 2 explicit paths, got %d", len(result.BlastRadius.ExplicitPaths))
	}
	if len(result.BlastRadius.ScopeMarkers) != 2 {
		t.Fatalf("expected 2 scope markers, got %d", len(result.BlastRadius.ScopeMarkers))
	}
	if result.BlastRadius.Targets != 4 {
		t.Fatalf("expected 4 targets (paths + markers), got %d", result.BlastRadius.Targets)
	}
}

func TestNoAlertForNarrowSpecificDirective(t *testing.T) {
	t.Parallel()

	directive := "Update cmd/vectorpad/main.go to print the ambiguity summary after metrics output."
	result := Analyze(directive, Scope{Repos: 1, Files: 1})

	if result.BrevityToScopeRatio >= 1.0 {
		t.Fatalf("expected ratio below 1.0, got %.2f", result.BrevityToScopeRatio)
	}
	if result.Warning.Active {
		t.Fatalf("expected no warning, got %s", result.Warning.Severity)
	}
	if !result.Warning.NonBlocking {
		t.Fatal("warning state must be non-blocking")
	}
}

func TestBroadVerboseDirectiveDoesNotAlert(t *testing.T) {
	t.Parallel()

	directive := strings.Join([]string{
		"For each repository in the workspace, normalize README heading spacing,",
		"leave all architecture sections unchanged, preserve examples, preserve tone,",
		"and only align badge order and markdown list formatting.",
	}, " ")

	result := Analyze(directive, Scope{Repos: 18, Files: 18, FileTypes: []string{"README.md"}})

	if result.Warning.Active {
		t.Fatalf("expected no warning, got %s", result.Warning.Severity)
	}
}

func TestBroadBriefDirectiveAlerts(t *testing.T) {
	t.Parallel()

	result := Analyze("rewrite README files", Scope{Repos: 12, Files: 12, FileTypes: []string{"README.md"}})

	if !result.Warning.Active {
		t.Fatal("expected warning for broad and brief directive")
	}
	if result.Warning.Severity != SeverityAmber {
		t.Fatalf("expected amber warning at ratio %.2f, got %s", result.BrevityToScopeRatio, result.Warning.Severity)
	}
	if !result.Warning.NonBlocking {
		t.Fatal("warning must be non-blocking")
	}
}

func TestVagueVerbBroadScopeEscalatesAlert(t *testing.T) {
	t.Parallel()

	result := Analyze("refactor docs now carefully", Scope{Repos: 5, Files: 16})

	if !result.Warning.Active {
		t.Fatal("expected warning for broad directive")
	}
	if result.Warning.Severity != SeverityRed {
		t.Fatalf("expected red severity after vague-verb escalation, got %s", result.Warning.Severity)
	}
	if !result.Warning.Escalated {
		t.Fatal("expected warning to be escalated by vague verb")
	}
}

func TestNoAlertWhenRatioBelowOne(t *testing.T) {
	t.Parallel()

	directive := "clean only src docs after preserving section headers and examples"
	result := Analyze(directive, Scope{Repos: 2, Files: 2})

	if result.BrevityToScopeRatio >= 1.0 {
		t.Fatalf("expected ratio below 1.0, got %.2f", result.BrevityToScopeRatio)
	}
	if result.Warning.Active {
		t.Fatalf("expected no warning below ratio threshold, got %s", result.Warning.Severity)
	}
}

func TestVagueVerbDetectionIsEnglishOnly(t *testing.T) {
	t.Parallel()

	result := Analyze("整理 文档 并 保持 结构", Scope{Repos: 4, Files: 12})

	if len(result.VagueVerbs) != 0 {
		t.Fatalf("expected no English vague verbs, got %v", result.VagueVerbs)
	}
}
