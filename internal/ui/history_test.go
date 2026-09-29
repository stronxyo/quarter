package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/steam"
)

// historyModel has n games, all played in the last week, game N for N
// minutes more than game N-1 so game n ranks first, plus SteamVR (a tool).
func historyModel(t *testing.T, n, width, height int) tea.Model {
	t.Helper()
	now := time.Now()
	var games []library.Game
	var sessions []steam.Session
	for i := 1; i <= n; i++ {
		id := uint32(1000 + i)
		games = append(games, library.Game{AppID: id, Name: fmt.Sprintf("Game %02d", i), Installed: true, Playtime: time.Duration(i) * time.Hour})
		start := now.Add(-time.Duration(i) * time.Hour)
		sessions = append(sessions, steam.Session{AppID: id, Start: start, End: start.Add(time.Duration(10+i) * time.Minute), EndKnown: true})
	}
	vr := now.Add(-30 * time.Hour)
	sessions = append(sessions, steam.Session{AppID: 250820, Start: vr, End: vr.Add(5 * time.Hour), EndKnown: true})
	opened := &[]string{}
	var m tea.Model = New(Options{Sessions: sessions, Open: func(uri string) error { *opened = append(*opened, uri); return nil }})
	m, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	m, _ = m.Update(scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: games}})
	return m
}

func hit(m tea.Model, keys ...tea.KeyPressMsg) tea.Model {
	for _, k := range keys {
		m, _ = m.Update(k)
	}
	return m
}

var (
	keyUp    = tea.KeyPressMsg{Code: tea.KeyUp}
	keyDown  = tea.KeyPressMsg{Code: tea.KeyDown}
	keyLeft  = tea.KeyPressMsg{Code: tea.KeyLeft}
	keyRight = tea.KeyPressMsg{Code: tea.KeyRight}
	keyEsc   = tea.KeyPressMsg{Code: tea.KeyEscape}
	keyEnter = tea.KeyPressMsg{Code: tea.KeyEnter}
)

func screenText(m tea.Model) string { return ansi.Strip(m.(Model).View().Content) }

// selectedRank is the rank of the ranking's highlighted row, e.g. "1.".
func selectedRank(t *testing.T, m tea.Model) string {
	t.Helper()
	for _, l := range strings.Split(screenText(m), "\n") {
		if strings.HasPrefix(l, "│ ") && strings.Contains(l, "Game ") {
			return strings.Fields(l)[1]
		}
	}
	t.Fatalf("no selected row:\n%s", screenText(m))
	return ""
}

func TestHistoryCycle(t *testing.T) {
	m := historyModel(t, 3, 150, 40)
	m = hit(m, press("h"))
	view := screenText(m)
	for _, want := range []string{"MOST PLAYED", "[7 days]", "1.", "Game 03", "Last 16 weeks", "Recent sessions", "Total"} {
		if !strings.Contains(view, want) {
			t.Errorf("ranking: missing %q:\n%s", want, view)
		}
	}
	m = hit(m, press("h"))
	view = screenText(m)
	for _, want := range []string{"MOST PLAYED", "H I G H   S C O R E S", "1ST", "GAME 03", "less", "more", "LONGEST", "STREAK"} {
		if !strings.Contains(view, want) {
			t.Errorf("board: missing %q:\n%s", want, view)
		}
	}
	m = hit(m, press("h"))
	if view := screenText(m); !strings.Contains(view, "[All]") || strings.Contains(view, "MOST PLAYED") {
		t.Errorf("the third h should bring the games back:\n%s", view)
	}
	for _, keys := range [][]tea.KeyPressMsg{{press("h"), keyEsc}, {press("h"), press("h"), keyEsc}} {
		if view := screenText(hit(m, keys...)); !strings.Contains(view, "[All]") {
			t.Errorf("esc should bring the games back:\n%s", view)
		}
	}
}

// Five periods, no "since"; left and right wrap round.
func TestHistoryPeriods(t *testing.T) {
	m := hit(historyModel(t, 3, 150, 40), press("h"))
	if view := screenText(hit(m, keyLeft)); !strings.Contains(view, "[1 year]") {
		t.Errorf("left from 7 days should wrap to 1 year:\n%s", view)
	}
	if view := screenText(hit(m, keyRight, keyRight)); !strings.Contains(view, "[3 months]") {
		t.Errorf("right twice should reach 3 months:\n%s", view)
	}
	if view := screenText(hit(m, keyLeft, keyRight)); !strings.Contains(view, "[7 days]") {
		t.Errorf("right from 1 year should wrap to 7 days:\n%s", view)
	}
	if view := screenText(m); strings.Contains(view, "since") {
		t.Errorf("no period is 'since':\n%s", view)
	}
}

// The ranking moves like the game list: round past both ends, on across
// pages, g and G; the screen keeps its height on every page.
func TestHistoryMovesLikeTheList(t *testing.T) {
	m := hit(historyModel(t, 40, 140, 24), press("h"))
	per := m.(Model).list.Paginator.PerPage
	if per >= 40 {
		t.Fatalf("the window should hold fewer than 40 rows, holds %d", per)
	}
	if got := selectedRank(t, hit(m, keyUp)); got != "40." {
		t.Errorf("up from 1st: got %s, want 40.", got)
	}
	if got := selectedRank(t, hit(m, keyUp, keyDown)); got != "1." {
		t.Errorf("down from 40th: got %s, want 1.", got)
	}
	next := m
	for range per {
		next = hit(next, keyDown)
	}
	if got, want := selectedRank(t, next), fmt.Sprintf("%d.", per+1); got != want {
		t.Errorf("down past the page: got %s, want %s", got, want)
	}
	if !strings.Contains(screenText(next), "2/") {
		t.Errorf("the page counter should say page 2:\n%s", screenText(next))
	}
	if got := selectedRank(t, hit(m, press("G"))); got != "40." {
		t.Errorf("G: got %s", got)
	}
	if got := selectedRank(t, hit(m, press("G"), press("g"))); got != "1." {
		t.Errorf("g: got %s", got)
	}
	for range 41 {
		m = hit(m, keyDown)
		if n := len(strings.Split(m.(Model).View().Content, "\n")); n != 24 {
			t.Fatalf("the screen is %d rows, want the window's 24", n)
		}
	}
}

// The pane (and the logo, through selected) follow the ranked game.
func TestHistoryFollowsRankedGame(t *testing.T) {
	m := hit(historyModel(t, 3, 150, 40), press("h"), keyDown)
	g, ok := m.(Model).selected()
	if !ok || g.Name != "Game 02" {
		t.Fatalf("the game on show should be the ranked 2nd, Game 02: %+v", g)
	}
	if !strings.Contains(screenText(m), "GAME 02") {
		t.Errorf("the pane should show Game 02:\n%s", screenText(m))
	}
}

// Steam tools don't rank.
func TestHistoryLeavesOutTools(t *testing.T) {
	m := hit(historyModel(t, 3, 150, 40), press("h"))
	if view := screenText(m); strings.Contains(view, "App 250820") || strings.Contains(view, "4.") {
		t.Errorf("SteamVR shouldn't rank:\n%s", view)
	}
}

// With history open, keys that act on a game wait; refresh and quitting don't.
func TestHistoryOtherKeysWait(t *testing.T) {
	m := hit(historyModel(t, 3, 150, 40), press("h"))
	before := m.(Model)
	for _, k := range []tea.KeyPressMsg{keyEnter, press("f"), press("i"), press("/"), press("d"), press("o"), press("n")} {
		after, cmd := m.Update(k)
		if cmd != nil || after.(Model).histView != histRanking || len(after.(Model).favs) != len(before.favs) || after.(Model).list.SettingFilter() {
			t.Errorf("%q should wait while history is open", k.String())
		}
	}
	if _, cmd := m.Update(press("q")); cmd == nil {
		t.Error("q should still quit")
	} else if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Error("q should still quit")
	}
	if after, cmd := m.Update(press("r")); cmd == nil || !after.(Model).scanning {
		t.Error("r should still rescan")
	}
}

// Narrow window: the ranking takes the list's place, then the board does.
func TestHistoryNarrowWindow(t *testing.T) {
	m := hit(historyModel(t, 3, 90, 40), press("h"))
	if view := screenText(m); !strings.Contains(view, "MOST PLAYED") || strings.Contains(view, "Recent sessions") {
		t.Errorf("narrow: the ranking alone:\n%s", view)
	}
	m = hit(m, press("h"))
	if view := screenText(m); !strings.Contains(view, "H I G H   S C O R E S") {
		t.Errorf("narrow: the board in the list's place:\n%s", view)
	}
	for _, l := range strings.Split(screenText(m), "\n") {
		if w := ansi.StringWidth(l); w > 90 {
			t.Errorf("line wider than the window (%d): %q", w, l)
		}
	}
}

// No sessions yet: every view says so and nothing breaks.
func TestHistoryEmpty(t *testing.T) {
	var m tea.Model = New(Options{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 150, Height: 40})
	m, _ = m.Update(scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: []library.Game{{AppID: 1, Name: "Alpha", Installed: true}}}})
	m = hit(m, press("h"))
	if view := screenText(m); strings.Count(view, "Nothing played in this period.") != 2 {
		t.Errorf("the ranking and the pane should say nothing was played:\n%s", view)
	}
	m = hit(m, press("h"))
	if view := screenText(m); !strings.Contains(view, "1ST") || !strings.Contains(view, "---") {
		t.Errorf("an empty board shows its places dotted:\n%s", view)
	}
	m = hit(m, keyDown, keyUp, press("g"), press("G"), keyLeft)
	if len(strings.Split(m.(Model).View().Content, "\n")) != 40 {
		t.Error("the screen should keep its height")
	}
}

// New sessions saved after a scan update the ranking.
func TestHistoryUpdatesWithNewSessions(t *testing.T) {
	m := hit(historyModel(t, 2, 150, 40), press("h"))
	start := time.Now().Add(-time.Hour)
	all := []steam.Session{{AppID: 1001, Start: start, End: start.Add(50 * time.Hour / 10), EndKnown: true}}
	m, _ = m.Update(sessionsMsg{all: all})
	if view := screenText(m); strings.Contains(view, "Game 02") || !strings.Contains(view, "Game 01") {
		t.Errorf("the ranking should now hold only Game 01:\n%s", view)
	}
}

// h is in the key help, and the help changes while history is open.
func TestHistoryHelp(t *testing.T) {
	m := historyModel(t, 3, 150, 40)
	if !strings.Contains(screenText(m), "h history") {
		t.Error("the key line should offer h history")
	}
	view := screenText(hit(m, press("h")))
	for _, want := range []string{"←→ period", "h next view", "esc games"} {
		if !strings.Contains(view, want) {
			t.Errorf("history key line missing %q:\n%s", want, view)
		}
	}
}
