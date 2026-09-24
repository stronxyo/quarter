package steam

import (
	"testing"
	"time"
)

func TestReadPlayStatsFixture(t *testing.T) {
	stats, err := ReadPlayStats(fixture, Account{AccountID: 2})
	if err != nil {
		t.Fatalf("ReadPlayStats: %v", err)
	}

	tests := []struct {
		appID      uint32
		playtime   time.Duration
		lastPlayed time.Time
	}{
		{620, 754 * time.Minute, time.Unix(1726000000, 0)},
		{400, 90 * time.Minute, time.Time{}},
		{70, 30 * time.Minute, time.Unix(1600000000, 0)},
	}
	for _, tt := range tests {
		got, ok := stats[tt.appID]
		if !ok {
			t.Errorf("app %d missing", tt.appID)
			continue
		}
		if got.Playtime != tt.playtime || !got.LastPlayed.Equal(tt.lastPlayed) {
			t.Errorf("app %d: got %+v, want playtime %v last played %v", tt.appID, got, tt.playtime, tt.lastPlayed)
		}
	}

	// An entry with neither playtime nor last played carries no stats.
	if _, ok := stats[12345]; ok {
		t.Error("app 12345 has no play data and should be absent")
	}
}

func TestReadPlayStatsMissingFile(t *testing.T) {
	if _, err := ReadPlayStats(fixture, Account{AccountID: 99}); err == nil {
		t.Error("want an error for a missing localconfig.vdf")
	}
}
