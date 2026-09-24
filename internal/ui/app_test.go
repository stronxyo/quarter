package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/library"
)

func press(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
}

// ready returns a model sized 80x24 with the given games loaded.
func ready(t *testing.T, games ...string) Model {
	t.Helper()
	res := library.Result{PlaytimeKnown: true}
	for i, name := range games {
		res.Games = append(res.Games, library.Game{AppID: uint32(i + 1), Name: name, Installed: true})
	}
	var m tea.Model = New("")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(scanDoneMsg{res: res})
	return m.(Model)
}

func TestScanFillsList(t *testing.T) {
	m := ready(t, "Alpha", "Bravo", "Charlie")
	if m.state != stateReady {
		t.Fatalf("state: got %v, want ready", m.state)
	}
	if n := len(m.list.Items()); n != 3 {
		t.Errorf("items: got %d, want 3", n)
	}
	view := m.View().Content
	for _, name := range []string{"Alpha", "Bravo", "Charlie", "QUARTER"} {
		if !strings.Contains(view, name) {
			t.Errorf("view missing %q", name)
		}
	}
	if !m.View().AltScreen {
		t.Error("UI should use the full screen")
	}
}

func TestMoveSelection(t *testing.T) {
	var m tea.Model = ready(t, "Alpha", "Bravo", "Charlie")
	m, _ = m.Update(press("j"))
	m, _ = m.Update(press("j"))
	if got := m.(Model).list.Index(); got != 2 {
		t.Errorf("after j j: index %d, want 2", got)
	}
	m, _ = m.Update(press("k"))
	if got := m.(Model).list.Index(); got != 1 {
		t.Errorf("after k: index %d, want 1", got)
	}
}

func TestQuit(t *testing.T) {
	for _, name := range []string{"ready", "failed"} {
		t.Run(name, func(t *testing.T) {
			var m tea.Model = ready(t, "Alpha")
			if name == "failed" {
				m, _ = New("").Update(scanDoneMsg{err: errors.New("not found")})
			}
			_, cmd := m.Update(press("q"))
			if cmd == nil {
				t.Fatal("q returned no command")
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Error("q should quit")
			}
		})
	}
}

func TestSteamNotFound(t *testing.T) {
	m, _ := New("").Update(scanDoneMsg{err: errors.New("steam install not found")})
	view := m.(Model).View().Content
	if !strings.Contains(view, "Steam not found") || !strings.Contains(view, "--steam-path") {
		t.Errorf("error screen missing guidance:\n%s", view)
	}
}

func TestWarningsFooter(t *testing.T) {
	var m tea.Model = New("")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(scanDoneMsg{res: library.Result{
		Games:    []library.Game{{AppID: 1, Name: "Alpha"}},
		Warnings: []error{errors.New("bad file")},
	}})
	if view := m.(Model).View().Content; !strings.Contains(view, "1 file(s) could not be read") {
		t.Errorf("footer missing:\n%s", view)
	}
}

func TestLongNameTruncated(t *testing.T) {
	m := ready(t, strings.Repeat("Very Long Game Name ", 10))
	for _, line := range strings.Split(m.View().Content, "\n") {
		if w := ansi.StringWidth(line); w > 80 {
			t.Errorf("line wider than the window (%d): %q", w, line)
		}
	}
}

// withLibrary returns a ready model with games that differ in playtime and recency.
func withLibrary(t *testing.T) Model {
	t.Helper()
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	res := library.Result{PlaytimeKnown: true, Games: []library.Game{
		{AppID: 1, Name: "Alpha", Playtime: time.Hour, LastPlayed: now.Add(-30 * 24 * time.Hour)},
		{AppID: 2, Name: "Bravo", Playtime: 50 * time.Hour, LastPlayed: now.Add(-24 * time.Hour)},
		{AppID: 3, Name: "Rocket League", Playtime: 20 * time.Hour},
	}}
	m := New("")
	m.now = func() time.Time { return now }
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	tm, _ = tm.Update(scanDoneMsg{res: res})
	return tm.(Model)
}

func visible(m Model) []string {
	var out []string
	for _, it := range m.list.VisibleItems() {
		out = append(out, it.(gameItem).Name)
	}
	return out
}

func TestSortKeys(t *testing.T) {
	var m tea.Model = withLibrary(t)
	order := func() string { return strings.Join(visible(m.(Model)), ",") }

	m, _ = m.Update(press("p"))
	if got := order(); got != "Bravo,Rocket League,Alpha" {
		t.Errorf("p: got %v, want most played first", got)
	}
	if !strings.Contains(ansi.Strip(m.(Model).View().Content), "Played ▼") {
		t.Error("column header should mark Played as sorted, highest first")
	}

	m, _ = m.Update(press("p"))
	if got := order(); got != "Alpha,Rocket League,Bravo" {
		t.Errorf("p again: got %v, want least played first", got)
	}

	m, _ = m.Update(press("l"))
	if got := order(); got != "Bravo,Alpha,Rocket League" {
		t.Errorf("l: got %v, want most recent first, never played last", got)
	}

	m, _ = m.Update(press("n"))
	if got := order(); got != "Alpha,Bravo,Rocket League" {
		t.Errorf("n: got %v, want A to Z", got)
	}
}

func TestFilterKey(t *testing.T) {
	var m tea.Model = withLibrary(t)
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
	if got := visible(m.(Model)); strings.Join(got, ",") != "Bravo" {
		t.Errorf("recent filter: got %v, want only Bravo", got)
	}
	if !strings.Contains(ansi.Strip(m.(Model).View().Content), "[Recent]") {
		t.Error("header should highlight the Recent filter")
	}
}

func TestSelectionSurvivesSort(t *testing.T) {
	var m tea.Model = withLibrary(t)
	m, _ = m.Update(press("j")) // Bravo
	m, _ = m.Update(press("p")) // most played: Bravo moves to the top
	sel := m.(Model).list.SelectedItem().(gameItem)
	if sel.Name != "Bravo" {
		t.Errorf("selection after sort: got %s, want Bravo", sel.Name)
	}
}

func TestSearchNarrows(t *testing.T) {
	m := withLibrary(t)
	m.list.SetFilterText("rock")
	if got := visible(m); len(got) != 1 || got[0] != "Rocket League" {
		t.Errorf("search rock: got %v", got)
	}
}

func TestKeysGoToSearchWhileTyping(t *testing.T) {
	var m tea.Model = withLibrary(t)
	m, _ = m.Update(press("/"))
	m, _ = m.Update(press("p"))
	if m.(Model).sortBy != library.SortName {
		t.Error("typing 'p' in the search box must not change the sort")
	}
}

func TestTableCappedOnWideTerminal(t *testing.T) {
	var m tea.Model = ready(t, "Alpha")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 200, Height: 24})
	for _, line := range strings.Split(ansi.Strip(m.(Model).View().Content), "\n") {
		if strings.Contains(line, "Alpha") || strings.Contains(line, "Played") {
			if w := ansi.StringWidth(strings.TrimRight(line, " ")); w > maxTableWidth {
				t.Errorf("table row is %d wide on a 200-column terminal, want at most %d: %q", w, maxTableWidth, line)
			}
		}
	}
}
