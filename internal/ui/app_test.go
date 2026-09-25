package ui

import (
	"errors"
	"fmt"
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
	var m tea.Model = New(Options{})
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
	for _, name := range []string{"Alpha", "Bravo", "Charlie", "[All]"} {
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
				m, _ = New(Options{}).Update(scanDoneMsg{err: errors.New("not found")})
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
	m, _ := New(Options{}).Update(scanDoneMsg{err: errors.New("steam install not found")})
	view := m.(Model).View().Content
	if !strings.Contains(view, "Steam not found") || !strings.Contains(view, "--steam-path") {
		t.Errorf("error screen missing guidance:\n%s", view)
	}
}

func TestWarningsFooter(t *testing.T) {
	var m tea.Model = New(Options{})
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
	m := New(Options{})
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
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // Installed
	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab}) // Recent
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
			if w := ansi.StringWidth(tablePart(line)); w > maxTableWidth {
				t.Errorf("table row is %d wide on a 200-column terminal, want at most %d: %q", w, maxTableWidth, line)
			}
		}
	}
}

func TestLogoWhenTall(t *testing.T) {
	gameNames := func(n int) []string {
		out := make([]string, n)
		for i := range out {
			out[i] = fmt.Sprintf("Game %02d", i)
		}
		return out
	}
	tests := []struct {
		name          string
		games         int
		width, height int
		wantLogo      bool
	}{
		{"tall window", 2, 80, 40, true},
		{"zoomed in, 11 games", 11, 110, 30, true},
		{"small library, short window", 2, 80, 24, true},
		{"big library, short window", 40, 80, 24, false},
		{"narrow window", 2, 50, 40, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m tea.Model = ready(t, gameNames(tt.games)...)
			m, _ = m.Update(tea.WindowSizeMsg{Width: tt.width, Height: tt.height})
			view := ansi.Strip(m.(Model).View().Content)

			hasLogo := strings.Contains(view, "██████┐")
			if hasLogo != tt.wantLogo {
				t.Errorf("logo shown: got %v, want %v", hasLogo, tt.wantLogo)
			}
			if !tt.wantLogo && !strings.Contains(view, "quarter_") {
				t.Error("without the logo, the compact quarter_ badge must show")
			}
			if lines := strings.Count(view, "\n") + 1; lines > tt.height {
				t.Errorf("view is %d lines, taller than the %d-line window", lines, tt.height)
			}
		})
	}
}

func TestLayoutFollowsSearch(t *testing.T) {
	names := make([]string, 40)
	for i := range names {
		names[i] = fmt.Sprintf("Game %02d", i)
	}
	var m tea.Model = ready(t, names...)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	mm := m.(Model)
	mm.list.SetFilterText("Game 07") // narrows to one game, so the logo now fits
	m, _ = mm.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	view := ansi.Strip(m.(Model).View().Content)
	if lines := strings.Count(view, "\n") + 1; lines > 24 {
		t.Errorf("after search the view is %d lines, taller than the window", lines)
	}
}

func TestSearchOnSectionLine(t *testing.T) {
	var m tea.Model = withLibrary(t)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 24})
	m, _ = m.Update(press("/"))
	m, _ = m.Update(press("r"))

	for _, line := range strings.Split(ansi.Strip(m.(Model).View().Content), "\n") {
		if !strings.Contains(line, "[All]") {
			continue
		}
		if !strings.Contains(line, "Search: r") {
			t.Errorf("search box should sit on the section line: %q", line)
		}
		if w := ansi.StringWidth(tablePart(line)); w > maxTableWidth {
			t.Errorf("section line is %d wide, want at most %d", w, maxTableWidth)
		}
		return
	}
	t.Fatal("no section line found")
}

// run executes a command and any batch it expands into, discarding messages.
func run(cmd tea.Cmd) {
	if cmd == nil {
		return
	}
	if batch, ok := cmd().(tea.BatchMsg); ok {
		for _, c := range batch {
			run(c)
		}
	}
}

func cached(names ...string) *library.Result {
	res := &library.Result{PlaytimeKnown: true}
	for i, n := range names {
		res.Games = append(res.Games, library.Game{AppID: uint32(i + 1), Name: n, Installed: true})
	}
	return res
}

func TestCachedStartIsInstant(t *testing.T) {
	var m tea.Model = New(Options{Cached: cached("Alpha", "Bravo")})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	view := ansi.Strip(m.(Model).View().Content)
	if m.(Model).state != stateReady || !strings.Contains(view, "Bravo") {
		t.Fatalf("cached games should show before any scan:\n%s", view)
	}
	if !strings.Contains(view, "updating…") {
		t.Error("section line should say a scan is running")
	}
}

func TestFreshScanReplacesCacheAndSaves(t *testing.T) {
	var saved []library.Result
	save := func(r library.Result) error { saved = append(saved, r); return nil }

	var m tea.Model = New(Options{Cached: cached("Alpha", "Bravo"), Save: save})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(press("j")) // select Bravo
	m, cmd := m.Update(scanDoneMsg{res: *cached("Alpha", "Bravo", "Charlie")})
	run(cmd)

	mm := m.(Model)
	if len(mm.list.Items()) != 3 {
		t.Errorf("fresh scan not applied: %d items", len(mm.list.Items()))
	}
	if sel := mm.list.SelectedItem().(gameItem); sel.Name != "Bravo" {
		t.Errorf("selection lost on refresh: %s", sel.Name)
	}
	if len(saved) != 1 || len(saved[0].Games) != 3 {
		t.Errorf("fresh scan should be saved once, got %d saves", len(saved))
	}
	if strings.Contains(ansi.Strip(mm.View().Content), "updating…") {
		t.Error("'updating…' should clear after the scan")
	}
}

func TestFailedRescanKeepsCache(t *testing.T) {
	var m tea.Model = New(Options{Cached: cached("Alpha")})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(scanDoneMsg{err: errors.New("steam install not found")})
	view := ansi.Strip(m.(Model).View().Content)
	if !strings.Contains(view, "Alpha") || !strings.Contains(view, "showing the last saved library") {
		t.Errorf("failed rescan should keep the cached list and say so:\n%s", view)
	}
}

func TestRefreshKey(t *testing.T) {
	var m tea.Model = ready(t, "Alpha")
	m, cmd := m.Update(press("r"))
	if cmd == nil || !m.(Model).scanning {
		t.Fatal("r should start a scan")
	}
	if _, cmd = m.Update(press("r")); cmd != nil {
		t.Error("r during a scan should not start a second one")
	}
}

// tablePart is a screen line without the detail pane beside the table. The
// search starts after the first character: the selected row's bar is there.
func tablePart(line string) string {
	if i := strings.IndexAny(line[min(1, len(line)):], "│┌└"); i >= 0 {
		line = line[:i+min(1, len(line))]
	}
	return strings.TrimRight(line, " ")
}
