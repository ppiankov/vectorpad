package stash

import (
	"encoding/json"
	"fmt"
	"strings"
)

// VerdictDiff represents the comparison between two verdict items.
type VerdictDiff struct {
	ID1    string
	ID2    string
	Title1 string
	Title2 string
	// Field-level changes.
	Changes []FieldChange
}

// FieldChange represents a single changed field between two verdicts.
type FieldChange struct {
	Field string
	Old   string
	New   string
}

// DiffVerdicts compares two verdict items and returns the differences.
func DiffVerdicts(a, b Item) (*VerdictDiff, error) {
	if a.Type != ItemTypeVerdict || b.Type != ItemTypeVerdict {
		return nil, fmt.Errorf("both items must be verdicts")
	}

	jsonA := ExtractVerdictJSON(a.Text)
	jsonB := ExtractVerdictJSON(b.Text)

	var mapA, mapB map[string]interface{}
	if err := json.Unmarshal([]byte(jsonA), &mapA); err != nil {
		return nil, fmt.Errorf("parse verdict 1: %w", err)
	}
	if err := json.Unmarshal([]byte(jsonB), &mapB); err != nil {
		return nil, fmt.Errorf("parse verdict 2: %w", err)
	}

	diff := &VerdictDiff{
		ID1:    itemDisplayID(a),
		ID2:    itemDisplayID(b),
		Title1: a.Title,
		Title2: b.Title,
	}

	// Compare top-level fields.
	allKeys := make(map[string]bool)
	for k := range mapA {
		allKeys[k] = true
	}
	for k := range mapB {
		allKeys[k] = true
	}

	for k := range allKeys {
		oldVal := formatValue(mapA[k])
		newVal := formatValue(mapB[k])
		if oldVal != newVal {
			diff.Changes = append(diff.Changes, FieldChange{
				Field: k,
				Old:   oldVal,
				New:   newVal,
			})
		}
	}

	return diff, nil
}

// ExtractVerdictJSON pulls the JSON portion from a verdict text.
// Format: "verdict: <title>\n\n<json>"
func ExtractVerdictJSON(text string) string {
	idx := strings.Index(text, "\n\n")
	if idx < 0 {
		return text
	}
	return strings.TrimSpace(text[idx+2:])
}

func itemDisplayID(item Item) string {
	if item.ClaimID != "" {
		return item.ClaimID
	}
	return item.ID
}

func formatValue(v interface{}) string {
	if v == nil {
		return ""
	}
	switch val := v.(type) {
	case string:
		return val
	case float64:
		if val == float64(int(val)) {
			return fmt.Sprintf("%d", int(val))
		}
		return fmt.Sprintf("%.2f", val)
	case bool:
		return fmt.Sprintf("%t", val)
	case []interface{}:
		var parts []string
		for _, item := range val {
			parts = append(parts, formatValue(item))
		}
		return strings.Join(parts, ", ")
	case map[string]interface{}:
		b, _ := json.Marshal(val)
		return string(b)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// Render formats the diff as a human-readable string.
func (d *VerdictDiff) Render() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("Comparing %s → %s\n\n", d.ID1, d.ID2))

	if len(d.Changes) == 0 {
		b.WriteString("  (no differences)\n")
		return b.String()
	}

	for _, c := range d.Changes {
		if c.Old == "" {
			b.WriteString(fmt.Sprintf("  + %s: %s\n", c.Field, c.New))
		} else if c.New == "" {
			b.WriteString(fmt.Sprintf("  - %s: %s\n", c.Field, c.Old))
		} else {
			b.WriteString(fmt.Sprintf("  ~ %s:\n", c.Field))
			b.WriteString(fmt.Sprintf("    v1: %s\n", c.Old))
			b.WriteString(fmt.Sprintf("    v2: %s\n", c.New))
		}
	}

	return b.String()
}
