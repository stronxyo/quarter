package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/webapi"
)

// withCandidates is a library of one installed game and three candidates,
// with the store's answers for them.
func withCandidates(t *testing.T, opened *[]string) tea.Model {
	t.Helper()
	src := &fakeStoreSource{
		info: map[uint32]webapi.StoreInfo{
			1:  {Type: "game", Name: "Alpha"},
			10: {Type: "game", Name: "Old Favourite"},
			11: {Type: "dlc", Name: "Alpha Soundtrack"},
		},
		errs: map[uint32]error{12: webapi.ErrNotOnStore},
	}
	m := New(Options{FetchStore: src.fetch, Open: func(uri string) error {
		*opened = append(*opened, uri)
		return nil
	}})
	m.debounce = 0
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 150, Height: 30})
	tm, cmd := tm.Update(scanDoneMsg{res: library.Result{
		PlaytimeKnown: true,
		Games:         []library.Game{{AppID: 1, Name: "Alpha", Installed: true}},
		Uninstalled: []library.Game{
			{AppID: 10, Playtime: 5 * time.Hour},
			{AppID: 11},
			{AppID: 12},
		},
	}})
	return drive(tm, cmd)
}

func TestCandidatesJoinWhenTheStoreSaysGame(t *testing.T) {
	var opened []string
	m := withCandidates(t, &opened)
	view := ansi.Strip(m.(Model).View().Content)

	if line := lineWith(view, "Old Favourite"); !strings.Contains(line, markerUninstalled+" Old Favourite") || !strings.Contains(line, "5.0h") {
		t.Errorf("the store says 10 is a game: it should show, not installed, with its local playtime: %q", line)
	}
	for _, hidden := range []string{"Alpha Soundtrack", "App 12"} {
		if strings.Contains(view, hidden) {
			t.Errorf("%q is not a game and should not show", hidden)
		}
	}
	if !strings.Contains(view, "2 games") {
		t.Error("count should be the installed game plus the one resolved candidate")
	}
}

func TestInstalledFilterHidesCandidates(t *testing.T) {
	var opened []string
	m := withCandidates(t, &opened)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // Installed
	if strings.Contains(ansi.Strip(m.(Model).View().Content), "Old Favourite") {
		t.Error("the Installed filter should hide games that aren't installed")
	}
}

func TestEnterOnCandidateInstalls(t *testing.T) {
	var opened []string
	m := withCandidates(t, &opened)
	for !strings.Contains(lineWith(ansi.Strip(m.(Model).View().Content), "Old Favourite"), "│") {
		m, _ = m.Update(press("j"))
	}
	act(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(opened) != 1 || opened[0] != "steam://install/10" {
		t.Errorf("Enter on a game that isn't installed should open the install dialog, opened %v", opened)
	}
}

func TestCandidatesShowAtOnceFromCache(t *testing.T) {
	now := time.Now()
	m := New(Options{
		Cached: &library.Result{
			Games:       []library.Game{{AppID: 1, Name: "Alpha", Installed: true}},
			Uninstalled: []library.Game{{AppID: 10}},
		},
		StoreCache: map[uint32]webapi.StoreEntry{10: {Info: webapi.StoreInfo{Type: "game", Name: "Old Favourite"}, Fetched: now}},
	})
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 150, Height: 30})
	if !strings.Contains(ansi.Strip(tm.(Model).View().Content), "Old Favourite") {
		t.Error("a candidate resolved on an earlier run should show at once")
	}
}
