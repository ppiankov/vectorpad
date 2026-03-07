package stash

import "time"

const (
	// CurrentVersion tracks the on-disk stash schema version.
	CurrentVersion = 1

	UnclusteredStackID    = "unclustered"
	UnclusteredStackLabel = "Unclustered"
)

type Uniqueness string

const (
	UniquenessHigh   Uniqueness = "high"
	UniquenessMedium Uniqueness = "medium"
	UniquenessLow    Uniqueness = "low"
)

type Source string

const (
	SourceCLI   Source = "cli"
	SourcePaste Source = "paste"
)

type AgeTier string

const (
	AgeTierFresh  AgeTier = "fresh"
	AgeTierRecent AgeTier = "recent"
	AgeTierAging  AgeTier = "aging"
	AgeTierStale  AgeTier = "stale"
)

const (
	freshWindow  = 24 * time.Hour
	recentWindow = 7 * 24 * time.Hour
	agingWindow  = 30 * 24 * time.Hour
)

type StashFile struct {
	Stacks  []Stack `json:"stacks"`
	Version int     `json:"version"`
}

type Stack struct {
	ID      string    `json:"id"`
	Label   string    `json:"label"`
	Created time.Time `json:"created"`
	Updated time.Time `json:"updated"`
	Items   []Item    `json:"items"`
}

type Item struct {
	ID         string     `json:"id"`
	Text       string     `json:"text"`
	Created    time.Time  `json:"created"`
	Uniqueness Uniqueness `json:"uniqueness"`
	Source     Source     `json:"source"`
}

func (item Item) AgeTier(now time.Time) AgeTier {
	return ComputeAgeTier(item.Created, now)
}

func ComputeAgeTier(created time.Time, now time.Time) AgeTier {
	if now.Before(created) {
		return AgeTierFresh
	}

	elapsed := now.Sub(created)
	switch {
	case elapsed < freshWindow:
		return AgeTierFresh
	case elapsed < recentWindow:
		return AgeTierRecent
	case elapsed < agingWindow:
		return AgeTierAging
	default:
		return AgeTierStale
	}
}
