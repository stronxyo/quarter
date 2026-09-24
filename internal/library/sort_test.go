package library

import (
	"strings"
	"testing"
	"time"
)

func names(games []Game) string {
	out := make([]string, len(games))
	for i, g := range games {
		out[i] = g.Name
	}
	return strings.Join(out, ",")
}

func TestView(t *testing.T) {
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	day := 24 * time.Hour
	games := []Game{
		{Name: "charlie", Playtime: 5 * time.Hour, LastPlayed: now.Add(-20 * day)},
		{Name: "Alpha", Playtime: 10 * time.Hour, LastPlayed: now.Add(-1 * day)},
		{Name: "bravo", Playtime: 10 * time.Hour},
		{Name: "Delta", Playtime: 0, LastPlayed: now.Add(-14 * day)},
	}

	tests := []struct {
		name string
		f    Filter
		s    SortOrder
		desc bool
		want string
	}{
		{"name A to Z, case-insensitive", FilterAll, SortName, false, "Alpha,bravo,charlie,Delta"},
		{"name Z to A", FilterAll, SortName, true, "Delta,charlie,bravo,Alpha"},
		{"most played first, ties by name", FilterAll, SortPlaytime, true, "Alpha,bravo,charlie,Delta"},
		{"least played first, ties still by name", FilterAll, SortPlaytime, false, "Delta,charlie,Alpha,bravo"},
		{"most recent first, never played last", FilterAll, SortLastPlayed, true, "Alpha,Delta,charlie,bravo"},
		{"recent keeps the last 14 days", FilterRecent, SortName, false, "Alpha,Delta"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := names(View(games, tt.f, tt.s, tt.desc, now)); got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}

	if games[0].Name != "charlie" {
		t.Error("View must not reorder the input slice")
	}
}

func TestDefaults(t *testing.T) {
	if SortName.DefaultDesc() || !SortPlaytime.DefaultDesc() || !SortLastPlayed.DefaultDesc() {
		t.Error("names should default to A to Z, playtime and last played to highest first")
	}
	if FilterAll.Next() != FilterRecent || FilterRecent.Next() != FilterAll {
		t.Error("filter does not cycle")
	}
}
