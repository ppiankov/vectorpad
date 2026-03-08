package flight

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func testRecorder(t *testing.T) *Recorder {
	t.Helper()
	dir := t.TempDir()
	return &Recorder{path: filepath.Join(dir, "log.jsonl")}
}

func TestAppendAndRecent(t *testing.T) {
	r := testRecorder(t)

	rec := Record{
		Target: "clipboard",
		Text:   "update all repos",
		Metrics: MetricsSnapshot{
			Tokens: 42, Integrity: 0.85, CPD: 0.003, TTC: 2.1, CDR: 0.7,
		},
		Gaps: []string{"preservation", "success"},
	}

	if err := r.Append(rec); err != nil {
		t.Fatalf("append: %v", err)
	}

	records, err := r.Recent(10)
	if err != nil {
		t.Fatalf("recent: %v", err)
	}
	if len(records) != 1 {
		t.Fatalf("expected 1 record, got %d", len(records))
	}
	if records[0].Target != "clipboard" {
		t.Errorf("expected target clipboard, got %s", records[0].Target)
	}
	if records[0].ID == "" {
		t.Error("expected auto-generated ID")
	}
}

func TestRecentNewestFirst(t *testing.T) {
	r := testRecorder(t)

	for i, text := range []string{"first", "second", "third"} {
		rec := Record{
			Text:     text,
			Launched: time.Now().Add(time.Duration(i) * time.Second),
		}
		if err := r.Append(rec); err != nil {
			t.Fatal(err)
		}
	}

	records, err := r.Recent(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}
	if records[0].Text != "third" {
		t.Errorf("expected 'third' first, got %q", records[0].Text)
	}
}

func TestAnnotate(t *testing.T) {
	r := testRecorder(t)

	rec := Record{ID: "test-id-123", Text: "some directive"}
	if err := r.Append(rec); err != nil {
		t.Fatal(err)
	}

	if err := r.Annotate("test-id-123", "bad", "deleted all READMEs"); err != nil {
		t.Fatal(err)
	}

	records, err := r.Recent(10)
	if err != nil {
		t.Fatal(err)
	}
	if records[0].Outcome != "bad" {
		t.Errorf("expected outcome 'bad', got %q", records[0].Outcome)
	}
	if records[0].Note != "deleted all READMEs" {
		t.Errorf("expected note, got %q", records[0].Note)
	}
}

func TestAnnotateNotFound(t *testing.T) {
	r := testRecorder(t)
	if err := r.Annotate("nonexistent", "good", ""); err == nil {
		t.Error("expected error for nonexistent ID")
	}
}

func TestComputeStats(t *testing.T) {
	r := testRecorder(t)

	records := []Record{
		{ID: "1", Metrics: MetricsSnapshot{CDR: 0.2}, Outcome: "good", Gaps: []string{"preservation"}},
		{ID: "2", Metrics: MetricsSnapshot{CDR: 0.8}, Outcome: "bad", Gaps: []string{"preservation", "success"}},
		{ID: "3", Metrics: MetricsSnapshot{CDR: 0.5}, Outcome: "good", Gaps: []string{"success"}},
		{ID: "4", Metrics: MetricsSnapshot{CDR: 0.9}}, // not annotated
	}
	for _, rec := range records {
		if err := r.Append(rec); err != nil {
			t.Fatal(err)
		}
	}

	stats, err := r.ComputeStats()
	if err != nil {
		t.Fatal(err)
	}

	if stats.TotalLaunches != 4 {
		t.Errorf("expected 4 launches, got %d", stats.TotalLaunches)
	}
	if stats.Annotated != 3 {
		t.Errorf("expected 3 annotated, got %d", stats.Annotated)
	}
	if stats.OutcomeCounts["good"] != 2 {
		t.Errorf("expected 2 good outcomes, got %d", stats.OutcomeCounts["good"])
	}
	if len(stats.TopGaps) < 2 {
		t.Errorf("expected at least 2 gap types, got %d", len(stats.TopGaps))
	}
	if stats.TopGaps[0].Class != "preservation" && stats.TopGaps[0].Class != "success" {
		t.Errorf("expected preservation or success as top gap, got %s", stats.TopGaps[0].Class)
	}
}

func TestEmptyLog(t *testing.T) {
	r := testRecorder(t)

	records, err := r.Recent(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 0 {
		t.Errorf("expected 0 records, got %d", len(records))
	}

	stats, err := r.ComputeStats()
	if err != nil {
		t.Fatal(err)
	}
	if stats.TotalLaunches != 0 {
		t.Errorf("expected 0 launches, got %d", stats.TotalLaunches)
	}
}

func TestAppendSurvivesCrash(t *testing.T) {
	r := testRecorder(t)

	// Append a record.
	if err := r.Append(Record{Text: "first"}); err != nil {
		t.Fatal(err)
	}

	// Verify the file exists and has content.
	info, err := os.Stat(r.path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Error("expected non-empty flight log")
	}

	// Append another — should not corrupt the first.
	if err := r.Append(Record{Text: "second"}); err != nil {
		t.Fatal(err)
	}

	records, err := r.Recent(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(records) != 2 {
		t.Fatalf("expected 2 records after two appends, got %d", len(records))
	}
}
