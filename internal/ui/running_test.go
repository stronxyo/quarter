package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/library"
)

func runningView(m tea.Model) string { return ansi.Strip(m.(Model).View().Content) }

func lineWith(view, name string) string {
	for _, l := range strings.Split(view, "\n") {
		if strings.Contains(l, name) {
			return l
		}
	}
	return ""
}

func TestRunningGameMarked(t *testing.T) {
	m, opened := withOpener(t, nil,
		library.Game{AppID: 1, Name: "Alpha", Installed: true},
		library.Game{AppID: 2, Name: "Bravo", Installed: true, Favourite: true},
	)
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 150, Height: 30})
	tm, _ = tm.Update(runningMsg{appID: 2})

	if line := lineWith(runningView(tm), "Bravo"); !strings.Contains(line, "▶ running") {
		t.Errorf("Bravo's row should say it's running: %q", line)
	}
	if line := lineWith(runningView(tm), "Alpha"); strings.Contains(line, "running") {
		t.Errorf("Alpha isn't running: %q", line)
	}

	tm, _ = tm.Update(press("j")) // select Bravo
	if !strings.Contains(runningView(tm), "RUNNING NOW") {
		t.Error("the pane should say Bravo is running")
	}

	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter}) // its clear timer isn't run
	if len(*opened) != 0 {
		t.Errorf("Enter on the running game must not launch it again, opened %v", *opened)
	}
	if !strings.Contains(runningView(tm), "Bravo is already running") {
		t.Error("Enter on the running game should say it's already running")
	}

	tm, _ = tm.Update(runningMsg{appID: 0})
	view := runningView(tm)
	if strings.Contains(lineWith(view, "• Bravo"), "running") || strings.Contains(view, "RUNNING NOW") {
		t.Error("the mark should clear when the game stops")
	}
}

// A running game shows the big triangle, even when it's a favourite, and the
// whole row stays green.
func TestRunningFavouriteShowsTheRunningMarker(t *testing.T) {
	m, _ := withOpener(t, nil,
		library.Game{AppID: 1, Name: "Alpha", Installed: true},
		library.Game{AppID: 2, Name: "Bravo", Installed: true},
	)
	m.favs[2] = true
	m.decorate()
	m.refreshList()
	var tm tea.Model = m
	tm, _ = tm.Update(runningMsg{appID: 2}) // running, but not selected

	raw := lineWith(tm.(Model).View().Content, "Bravo")
	marker, name := strings.Index(raw, markerRunning), strings.Index(raw, "Bravo")
	if marker < 0 || name < 0 || marker > name {
		t.Fatalf("the running game should be marked %s before its name: %q", markerRunning, raw)
	}
	if strings.Contains(raw[marker:name], "\x1b[m") {
		t.Errorf("a color reset between the marker and the name breaks the running color: %q", raw)
	}
	if strings.Contains(raw, markerFavourite) {
		t.Errorf("while running, the star gives way to the running marker: %q", raw)
	}
}

func TestCheckRunning(t *testing.T) {
	if checkRunning(nil, 0) != nil {
		t.Error("no way to ask means no polling")
	}
	if msg := checkRunning(func() (uint32, bool) { return 0, false }, 0)(); msg != nil {
		t.Errorf("an unknown answer should stop the polling, got %v", msg)
	}
	if msg := checkRunning(func() (uint32, bool) { return 620, true }, 0)(); msg != (runningMsg{appID: 620}) {
		t.Errorf("got %v, want runningMsg{620}", msg)
	}
}
