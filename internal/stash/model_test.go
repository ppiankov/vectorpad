package stash

import (
	"testing"
	"time"
)

func TestComputeAgeTierBoundaries(t *testing.T) {
	now := time.Date(2026, time.March, 7, 15, 0, 0, 0, time.UTC)

	testCases := []struct {
		name    string
		created time.Time
		want    AgeTier
	}{
		{name: "future timestamp is fresh", created: now.Add(1 * time.Hour), want: AgeTierFresh},
		{name: "less than 24h is fresh", created: now.Add(-23 * time.Hour), want: AgeTierFresh},
		{name: "exactly 24h is recent", created: now.Add(-24 * time.Hour), want: AgeTierRecent},
		{name: "less than 7d is recent", created: now.Add(-(7*24*time.Hour - time.Hour)), want: AgeTierRecent},
		{name: "exactly 7d is aging", created: now.Add(-7 * 24 * time.Hour), want: AgeTierAging},
		{name: "less than 30d is aging", created: now.Add(-(30*24*time.Hour - time.Hour)), want: AgeTierAging},
		{name: "exactly 30d is stale", created: now.Add(-30 * 24 * time.Hour), want: AgeTierStale},
	}

	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			got := ComputeAgeTier(testCase.created, now)
			if got != testCase.want {
				t.Fatalf("expected age tier %q, got %q", testCase.want, got)
			}
		})
	}
}
