package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/library"
)

// withSettings returns a ready model started from s, recording every save.
func withSettings(t *testing.T, s config.Settings) (tea.Model, *[]config.Settings) {
	t.Helper()
	saved := &[]config.Settings{}
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	m := New(Options{Settings: s, SaveSettings: func(s config.Settings) error {
		*saved = append(*saved, s)
		return nil
	}})
	m.now = func() time.Time { return now }
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	tm, _ = tm.Update(scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: []library.Game{
		{AppID: 1, Name: "Alpha", Installed: true, Playtime: time.Hour, LastPlayed: now.Add(-30 * 24 * time.Hour)},
		{AppID: 2, Name: "Bravo", Installed: true, Playtime: 50 * time.Hour, LastPlayed: now.Add(-24 * time.Hour)},
		{AppID: 3, Name: "Charlie", Installed: true, Playtime: 20 * time.Hour, LastPlayed: now.Add(-2 * time.Hour)},
	}}})
	return tm, saved
}

func TestRestoresLastView(t *testing.T) {
	m, _ := withSettings(t, config.Settings{Favourites: []uint32{2}, Sort: "last_played", SortDesc: true, Filter: "recent"})
	if got := strings.Join(visible(m.(Model)), ","); got != "Charlie,Bravo" {
		t.Errorf("restored view: got %v, want recent games newest first", got)
	}
	view := ansi.Strip(m.(Model).View().Content)
	if !strings.Contains(view, "Last played ▼") || !strings.Contains(view, "[Recent]") {
		t.Errorf("header should show the restored sort and filter:\n%s", view)
	}
	if !strings.Contains(view, markerFavourite+" Bravo") {
		t.Error("Bravo should be starred")
	}
}

func TestUnknownSavedValuesUseDefaults(t *testing.T) {
	m, _ := withSettings(t, config.Settings{Sort: "rating", Filter: "wishlist"})
	mm := m.(Model)
	if mm.sortBy != library.SortName || mm.sortDesc || mm.filterBy != library.FilterAll {
		t.Errorf("got sort %v desc %v filter %v, want the defaults", mm.sortBy, mm.sortDesc, mm.filterBy)
	}
}

func TestSortAndFilterAreSaved(t *testing.T) {
	m, saved := withSettings(t, config.Settings{})
	m, cmd := m.Update(press("l"))
	run(cmd)
	m, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // Installed
	run(cmd)
	_, cmd = m.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // Recent
	run(cmd)

	if len(*saved) != 3 {
		t.Fatalf("want a save per change, got %d", len(*saved))
	}
	last := (*saved)[2]
	if last.Sort != "last_played" || !last.SortDesc || last.Filter != "recent" {
		t.Errorf("saved %+v, want last_played, newest first, recent", last)
	}
}

func TestFavouriteToggle(t *testing.T) {
	m, saved := withSettings(t, config.Settings{})
	m, _ = m.Update(press("j")) // Bravo
	m, cmd := m.Update(press("f"))
	run(cmd)
	if len(*saved) != 1 || len((*saved)[0].Favourites) != 1 || (*saved)[0].Favourites[0] != 2 {
		t.Fatalf("starring should save [2], got %+v", *saved)
	}

	// Filter to favourites: only Bravo.
	for m.(Model).filterBy != library.FilterFavourites {
		m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	}
	if got := strings.Join(visible(m.(Model)), ","); got != "Bravo" {
		t.Errorf("favourites filter: got %v, want Bravo", got)
	}

	m, cmd = m.Update(press("f")) // unstar
	run(cmd)
	if last := (*saved)[len(*saved)-1]; len(last.Favourites) != 0 {
		t.Errorf("unstarring should save no favourites, got %v", last.Favourites)
	}
	if len(visible(m.(Model))) != 0 {
		t.Error("the favourites filter should now be empty")
	}
}

func TestSelectedFavouriteKeepsRowColor(t *testing.T) {
	m, _ := withSettings(t, config.Settings{Favourites: []uint32{1}}) // Alpha, selected first
	for _, line := range strings.Split(m.(Model).View().Content, "\n") {
		star := strings.Index(line, markerFavourite)
		name := strings.Index(line, "Alpha")
		if star < 0 || name < 0 {
			continue
		}
		if strings.Contains(line[star:name], "\x1b[m") {
			t.Errorf("a color reset between the star and the name switches off the selected highlight: %q", line)
		}
		return
	}
	t.Fatal("selected starred row not found")
}
