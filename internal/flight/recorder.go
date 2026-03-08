package flight

import (
	"bufio"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Record captures a launched vector with its full analysis snapshot.
type Record struct {
	ID         string          `json:"id"`
	Launched   time.Time       `json:"launched"`
	Target     string          `json:"target"`
	Text       string          `json:"text"`
	Metrics    MetricsSnapshot `json:"metrics"`
	Gaps       []string        `json:"gaps,omitempty"`
	VagueVerbs []string        `json:"vague_verbs,omitempty"`
	Outcome    string          `json:"outcome,omitempty"` // good, bad, partial, or empty
	Note       string          `json:"note,omitempty"`
}

// MetricsSnapshot captures the key metrics at launch time.
type MetricsSnapshot struct {
	Tokens    int     `json:"tokens"`
	Integrity float64 `json:"integrity"`
	CPD       float64 `json:"cpd"`
	TTC       float64 `json:"ttc"`
	CDR       float64 `json:"cdr"`
}

// Stats holds aggregate pattern analysis from the flight log.
type Stats struct {
	TotalLaunches   int                `json:"total_launches"`
	Annotated       int                `json:"annotated"`
	OutcomeCounts   map[string]int     `json:"outcome_counts"`
	AvgCDRByOutcome map[string]float64 `json:"avg_cdr_by_outcome"`
	TopGaps         []GapFrequency     `json:"top_gaps,omitempty"`
}

// GapFrequency records how often a gap class appears.
type GapFrequency struct {
	Class string `json:"class"`
	Count int    `json:"count"`
}

// Recorder manages the append-only flight log.
type Recorder struct {
	path string
}

// NewRecorder creates a recorder writing to the default flight log location.
func NewRecorder() (*Recorder, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("resolve home dir: %w", err)
	}
	dir := filepath.Join(home, ".vectorpad", "flight")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("create flight dir: %w", err)
	}
	return &Recorder{path: filepath.Join(dir, "log.jsonl")}, nil
}

// Append writes a new record to the flight log.
func (r *Recorder) Append(rec Record) error {
	if rec.ID == "" {
		rec.ID = generateID()
	}
	if rec.Launched.IsZero() {
		rec.Launched = time.Now()
	}

	data, err := json.Marshal(rec)
	if err != nil {
		return fmt.Errorf("marshal record: %w", err)
	}

	f, err := os.OpenFile(r.path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return fmt.Errorf("open flight log: %w", err)
	}
	defer func() { _ = f.Close() }()

	_, err = f.Write(append(data, '\n'))
	return err
}

// Annotate sets the outcome and note for a record by ID.
// Rewrites the log file with the updated record.
func (r *Recorder) Annotate(id, outcome, note string) error {
	records, err := r.loadAll()
	if err != nil {
		return err
	}

	found := false
	for i := range records {
		if records[i].ID == id {
			records[i].Outcome = outcome
			records[i].Note = note
			found = true
			break
		}
	}
	if !found {
		return fmt.Errorf("record %s not found", id)
	}

	return r.writeAll(records)
}

// Recent returns the most recent n records (newest first).
func (r *Recorder) Recent(n int) ([]Record, error) {
	records, err := r.loadAll()
	if err != nil {
		return nil, err
	}

	// Reverse for newest-first.
	for i, j := 0, len(records)-1; i < j; i, j = i+1, j-1 {
		records[i], records[j] = records[j], records[i]
	}

	if n > 0 && len(records) > n {
		records = records[:n]
	}
	return records, nil
}

// ComputeStats analyzes the flight log for patterns.
func (r *Recorder) ComputeStats() (Stats, error) {
	records, err := r.loadAll()
	if err != nil {
		return Stats{}, err
	}

	stats := Stats{
		TotalLaunches:   len(records),
		OutcomeCounts:   make(map[string]int),
		AvgCDRByOutcome: make(map[string]float64),
	}

	cdrSums := make(map[string]float64)
	cdrCounts := make(map[string]int)
	gapCounts := make(map[string]int)

	for _, rec := range records {
		if rec.Outcome != "" {
			stats.Annotated++
			stats.OutcomeCounts[rec.Outcome]++
			cdrSums[rec.Outcome] += rec.Metrics.CDR
			cdrCounts[rec.Outcome]++
		}
		for _, gap := range rec.Gaps {
			gapCounts[gap]++
		}
	}

	for outcome, sum := range cdrSums {
		if cdrCounts[outcome] > 0 {
			stats.AvgCDRByOutcome[outcome] = sum / float64(cdrCounts[outcome])
		}
	}

	// Sort gaps by frequency.
	for class, count := range gapCounts {
		stats.TopGaps = append(stats.TopGaps, GapFrequency{Class: class, Count: count})
	}
	sort.Slice(stats.TopGaps, func(i, j int) bool {
		return stats.TopGaps[i].Count > stats.TopGaps[j].Count
	})
	if len(stats.TopGaps) > 5 {
		stats.TopGaps = stats.TopGaps[:5]
	}

	return stats, nil
}

func (r *Recorder) loadAll() ([]Record, error) {
	f, err := os.Open(r.path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("open flight log: %w", err)
	}
	defer func() { _ = f.Close() }()

	var records []Record
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024) // up to 1MB lines
	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var rec Record
		if err := json.Unmarshal(line, &rec); err != nil {
			continue // skip malformed lines
		}
		records = append(records, rec)
	}
	return records, scanner.Err()
}

func (r *Recorder) writeAll(records []Record) error {
	tmp := r.path + ".tmp"
	f, err := os.Create(tmp)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}

	for _, rec := range records {
		data, err := json.Marshal(rec)
		if err != nil {
			_ = f.Close()
			_ = os.Remove(tmp)
			return fmt.Errorf("marshal record: %w", err)
		}
		if _, err := f.Write(append(data, '\n')); err != nil {
			_ = f.Close()
			_ = os.Remove(tmp)
			return err
		}
	}

	if err := f.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, r.path)
}

func generateID() string {
	b := make([]byte, 8)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
