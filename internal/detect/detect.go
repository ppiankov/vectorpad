package detect

import (
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"
	"time"
)

// Capabilities holds detected optional tool integrations.
// Fields are set once at startup and read-only thereafter.
type Capabilities struct {
	Pastewatch    bool   // pastewatch binary found in PATH
	PastewatchBin string // resolved path to pastewatch binary
	ContextSpec   bool   // contextspectre binary found in PATH
	ContextBin    string // resolved path to contextspectre binary
}

// PastewatchMode controls how VectorPad handles pastewatch scan results.
type PastewatchMode string

const (
	ModeInspect  PastewatchMode = "inspect"  // warn only, operator decides
	ModeAutowash PastewatchMode = "autowash" // substitute secrets automatically
	ModeOff      PastewatchMode = "off"      // disable scanning entirely
)

// ScanResult holds the outcome of a pastewatch scan on copy-out payload.
type ScanResult struct {
	Clean    bool     // no secrets detected
	Findings []string // human-readable finding descriptions
}

// Detect runs binary lookups for optional integrations.
// Call once at startup; the result is immutable for the session.
func Detect() Capabilities {
	var caps Capabilities

	// Check both binary names: pastewatch-cli (Homebrew) and pastewatch.
	for _, name := range []string{"pastewatch-cli", "pastewatch"} {
		if path, err := exec.LookPath(name); err == nil {
			caps.Pastewatch = true
			caps.PastewatchBin = path
			break
		}
	}

	if path, err := exec.LookPath("contextspectre"); err == nil {
		caps.ContextSpec = true
		caps.ContextBin = path
	}

	return caps
}

// ScanPayload runs pastewatch scan on the given text.
// Returns a clean result if pastewatch is not available or scanning is disabled.
// Uses --format json --check for structured output with exit code semantics.
func ScanPayload(caps Capabilities, mode PastewatchMode, payload string) ScanResult {
	if !caps.Pastewatch || mode == ModeOff {
		return ScanResult{Clean: true}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, caps.PastewatchBin, "scan", "--format", "json")
	cmd.Stdin = strings.NewReader(payload)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	// Exit code 0 = clean, non-zero = findings detected (or error).
	if err == nil {
		return ScanResult{Clean: true}
	}

	// Try to parse JSON output for findings.
	findings := parseScanFindings(stdout.Bytes())
	if len(findings) == 0 {
		// If we can't parse but got non-zero exit, report generically.
		return ScanResult{
			Clean:    false,
			Findings: []string{"pastewatch detected possible secrets"},
		}
	}

	return ScanResult{
		Clean:    false,
		Findings: findings,
	}
}

// StatusLabel returns a human-readable status string for the risk panel.
func StatusLabel(caps Capabilities, mode PastewatchMode) string {
	if !caps.Pastewatch {
		return "not installed"
	}
	return string(mode)
}

// scanFinding represents a single finding from pastewatch JSON output.
type scanFinding struct {
	RuleID   string `json:"rule_id"`
	Severity string `json:"severity"`
	Match    string `json:"match"`
	Message  string `json:"message"`
}

func parseScanFindings(data []byte) []string {
	// Try array of findings first.
	var findings []scanFinding
	if err := json.Unmarshal(data, &findings); err == nil && len(findings) > 0 {
		result := make([]string, 0, len(findings))
		for _, f := range findings {
			desc := f.RuleID
			if f.Message != "" {
				desc = f.Message
			}
			result = append(result, desc)
		}
		return result
	}

	// Try object with findings array.
	var wrapper struct {
		Findings []scanFinding `json:"findings"`
	}
	if err := json.Unmarshal(data, &wrapper); err == nil && len(wrapper.Findings) > 0 {
		result := make([]string, 0, len(wrapper.Findings))
		for _, f := range wrapper.Findings {
			desc := f.RuleID
			if f.Message != "" {
				desc = f.Message
			}
			result = append(result, desc)
		}
		return result
	}

	return nil
}
