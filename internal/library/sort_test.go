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
	if FilterAll.Next() != FilterRecent || FilterRecent.Next() != FilterFavourites || FilterFavourites.Next() != FilterAll {
		t.Error("filter does not cycle")
	}
}

func TestFavouritesFilter(t *testing.T) {
	games := []Game{{Name: "Alpha", Favourite: true}, {Name: "Bravo"}, {Name: "Charlie", Favourite: true}}
	if got := names(View(games, FilterFavourites, SortName, false, time.Now())); got != "Alpha,Charlie" {
		t.Errorf("got %v, want Alpha,Charlie", got)
	}
}

func TestSavedNames(t *testing.T) {
	for _, s := range []SortOrder{SortName, SortPlaytime, SortLastPlayed} {
		if got, ok := ParseSortOrder(s.Key()); !ok || got != s {
			t.Errorf("sort %v does not round-trip through %q", s, s.Key())
		}
	}
	for _, f := range Filters() {
		if got, ok := ParseFilter(f.Key()); !ok || got != f {
			t.Errorf("filter %v does not round-trip through %q", f, f.Key())
		}
	}
	if _, ok := ParseSortOrder("rating"); ok {
		t.Error("unknown sort name should not parse")
	}
	if f, ok := ParseFilter(""); ok || f != FilterAll {
		t.Error("unknown filter name should fall back to All")
	}
}

func TestLabel(t *testing.T) {
	tests := []struct{ name, want string }{
		{"Slow Roads Demo", "Slow Roads (Demo)"},
		{"Poetry of Pace: Rally Demo", "Poetry of Pace: Rally (Demo)"},
		{"Dumbfire Playtest", "Dumbfire (Playtest)"},
		{"Metropolis 1998 - Demo", "Metropolis 1998 (Demo)"},
		{"FIFA 2013 Demo", "FIFA 2013 (Demo)"},
		{"Sonic \u2013 Demo", "Sonic (Demo)"},
		{"Rocket League", "Rocket League"},
		{"Demolition Derby", "Demolition Derby"},
		{"Demo", "Demo"},
		{"The Demo Disc Collection", "The Demo Disc Collection"},
	}
	for _, tt := range tests {
		if got := (Game{Name: tt.name}).Label(); got != tt.want {
			t.Errorf("%q: got %q, want %q", tt.name, got, tt.want)
		}
	}
}
