package detect

import (
	"os/exec"
	"testing"
)

func TestDetectFindsInstalledBinaries(t *testing.T) {
	caps := Detect()

	// We can't control what's installed, but we can verify the struct is populated correctly.
	if caps.Pastewatch {
		if caps.PastewatchBin == "" {
			t.Error("Pastewatch detected but PastewatchBin is empty")
		}
	} else {
		if caps.PastewatchBin != "" {
			t.Error("Pastewatch not detected but PastewatchBin is set")
		}
	}

	if caps.ContextSpec {
		if caps.ContextBin == "" {
			t.Error("ContextSpec detected but ContextBin is empty")
		}
	} else {
		if caps.ContextBin != "" {
			t.Error("ContextSpec not detected but ContextBin is set")
		}
	}
}

func TestDetectPastewatchMatchesLookPath(t *testing.T) {
	caps := Detect()
	// Detect() checks both pastewatch-cli and pastewatch, so mirror that logic.
	_, err1 := exec.LookPath("pastewatch-cli")
	_, err2 := exec.LookPath("pastewatch")
	expected := err1 == nil || err2 == nil
	if caps.Pastewatch != expected {
		t.Errorf("Pastewatch detection mismatch: got %v, LookPath says %v", caps.Pastewatch, expected)
	}
}

func TestScanPayloadCleanWhenPastewatchAbsent(t *testing.T) {
	caps := Capabilities{Pastewatch: false}
	result := ScanPayload(caps, ModeInspect, "some secret payload")
	if !result.Clean {
		t.Error("expected clean result when pastewatch is not installed")
	}
}

func TestScanPayloadCleanWhenModeOff(t *testing.T) {
	caps := Capabilities{Pastewatch: true, PastewatchBin: "/usr/bin/pastewatch"}
	result := ScanPayload(caps, ModeOff, "some secret payload")
	if !result.Clean {
		t.Error("expected clean result when mode is off")
	}
}

func TestStatusLabel(t *testing.T) {
	tests := []struct {
		caps     Capabilities
		mode     PastewatchMode
		expected string
	}{
		{Capabilities{Pastewatch: false}, ModeInspect, "not installed"},
		{Capabilities{Pastewatch: true}, ModeInspect, "inspect"},
		{Capabilities{Pastewatch: true}, ModeAutowash, "autowash"},
		{Capabilities{Pastewatch: true}, ModeOff, "off"},
	}

	for _, tt := range tests {
		result := StatusLabel(tt.caps, tt.mode)
		if result != tt.expected {
			t.Errorf("StatusLabel(%v, %s) = %q, want %q", tt.caps.Pastewatch, tt.mode, result, tt.expected)
		}
	}
}

func TestParseScanFindingsArray(t *testing.T) {
	data := `[{"rule_id":"api-key","severity":"high","match":"sk-...","message":"API key detected"}]`
	findings := parseScanFindings([]byte(data))
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
	if findings[0] != "API key detected" {
		t.Errorf("expected 'API key detected', got %q", findings[0])
	}
}

func TestParseScanFindingsWrapper(t *testing.T) {
	data := `{"findings":[{"rule_id":"conn-string","severity":"high","message":"Connection string detected"}]}`
	findings := parseScanFindings([]byte(data))
	if len(findings) != 1 {
		t.Fatalf("expected 1 finding, got %d", len(findings))
	}
}

func TestParseScanFindingsEmpty(t *testing.T) {
	findings := parseScanFindings([]byte("not json"))
	if findings != nil {
		t.Errorf("expected nil for invalid JSON, got %v", findings)
	}
}

func TestParseScanFindingsFallsBackToRuleID(t *testing.T) {
	data := `[{"rule_id":"generic-secret","severity":"medium"}]`
	findings := parseScanFindings([]byte(data))
	if len(findings) != 1 || findings[0] != "generic-secret" {
		t.Errorf("expected fallback to rule_id, got %v", findings)
	}
}
