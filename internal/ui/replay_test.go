package ui

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/replay"
	"github.com/stronxyo/quarter/internal/steam"
)

// pastYears is what a fake Steam gives: 2025 with two months, per game.
func pastYears() []replay.Year {
	oct := time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC)
	nov := time.Date(2025, 11, 1, 0, 0, 0, 0, time.UTC)
	return []replay.Year{{Year: 2025, Played: 40 * time.Hour, Months: []replay.Month{
		{Start: oct, Played: 30 * time.Hour, Games: map[uint32]time.Duration{1: 30 * time.Hour}},
		{Start: nov, Played: 10 * time.Hour, Games: map[uint32]time.Duration{2: 10 * time.Hour}},
	}}}
}

// replayModel has two games, a session in the current month, and a fake
// Steam; saved and deleted count what reached the data folder.
type replayFake struct {
	fetched, saved, deleted int
	years                   []replay.Year
	missing                 []int
	err                     error
}

func replayModel(t *testing.T, fake *replayFake, kept []replay.Year) tea.Model {
	t.Helper()
	start := time.Now().Add(-time.Hour)
	var m tea.Model = New(Options{
		Sessions: []steam.Session{{AppID: 1, Start: start, End: start.Add(30 * time.Minute), EndKnown: true}},
		Replay:   kept,
		FetchReplay: func(context.Context) ([]replay.Year, []int, error) {
			fake.fetched++
			return fake.years, fake.missing, fake.err
		},
		SaveReplay:   func([]replay.Year) error { fake.saved++; return nil },
		DeleteReplay: func() error { fake.deleted++; return nil },
	})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	m, _ = m.Update(scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: []library.Game{
		{AppID: 1, Name: "Alpha", Installed: true, Playtime: 100 * time.Hour},
		{AppID: 2, Name: "Bravo", Installed: true, Playtime: 20 * time.Hour},
	}}})
	return m
}

// run1 runs a command and feeds its message back (one level, no timers).
func run1(m tea.Model, cmd tea.Cmd) tea.Model {
	for _, msg := range messages(cmd) {
		if _, ok := msg.(replayDoneMsg); ok {
			m, cmd = m.Update(msg)
			run(cmd) // the save
		}
	}
	return m
}

// The past years are offered only inside history, and only until added.
func TestPastYearsOfferedInHistoryOnly(t *testing.T) {
	fake := &replayFake{years: pastYears()}
	m := replayModel(t, fake, nil)
	if strings.Contains(screenText(m), "past years") {
		t.Error("the games view shouldn't mention the past years")
	}
	if after := hit(m, press("a")); after.(Model).replay != replayOff || fake.fetched != 0 {
		t.Error("a in the games view should do nothing")
	}
	m = hit(m, press("h"))
	view := screenText(m)
	if !strings.Contains(view, "Older years? a add your past years from Steam") || !strings.Contains(keyLine(m), "a past years") {
		t.Errorf("history should offer them in the pane and the key line:\n%s", view)
	}
}

// a asks; esc declines and nothing is fetched; y fetches, keeps and shows.
func TestPastYearsBroughtIn(t *testing.T) {
	fake := &replayFake{years: pastYears(), missing: []int{2022}}
	m := hit(replayModel(t, fake, nil), press("h"), press("a"))
	view := screenText(m)
	for _, want := range []string{"BRING IN YOUR PAST YEARS?", "Steam Replay", "no login", "replay.json", "Public in Steam Replay", "y bring them in", "esc not now"} {
		if !strings.Contains(view, want) {
			t.Errorf("the question is missing %q:\n%s", want, view)
		}
	}
	if declined := hit(m, keyEsc); declined.(Model).replay != replayOff || fake.fetched != 0 {
		t.Error("esc should decline without asking Steam")
	}
	m, cmd := m.Update(press("y"))
	m = run1(m, cmd)
	if fake.fetched != 1 || fake.saved != 1 || m.(Model).replay != replayOn {
		t.Fatalf("y should fetch once and keep them: fetched %d saved %d", fake.fetched, fake.saved)
	}
	view = screenText(m)
	if !strings.Contains(view, "Past years added: 2025 (not public: 2022)") {
		t.Errorf("the message should name the years added and those not public:\n%s", view)
	}
	if strings.Contains(view, "Older years?") {
		t.Error("the offer goes once they're in")
	}
	// The board: the calendar's month average and the years line.
	board := screenText(hit(m, press("h")))
	for _, want := range []string{"▪ month average, past years", "YEARS", "2025 40h"} {
		if !strings.Contains(board, want) {
			t.Errorf("board missing %q:\n%s", want, board)
		}
	}
	if !strings.Contains(board, "▪") {
		t.Error("October and November 2025 should be shaded on the calendar")
	}
}

// 1 year counts the past-years months inside it.
func TestPastYearsInPeriods(t *testing.T) {
	m := replayModel(t, &replayFake{}, pastYears())
	mm := m.(Model)
	yearAgo := mm.hist.DaysBack(365, mm.now())
	want := time.Duration(0)
	for _, mo := range pastYears()[0].Months {
		if !mo.Start.Before(time.Date(yearAgo.Year(), yearAgo.Month(), yearAgo.Day(), 0, 0, 0, 0, time.UTC)) {
			want += mo.Games[1]
		}
	}
	if got := mm.hist.Played(1, yearAgo); got != want+30*time.Minute {
		t.Errorf("Alpha over 1 year: got %v, want %v", got, want+30*time.Minute)
	}
}

// Kept from an earlier run, they show at once; a asks to remove them, and
// y deletes the file.
func TestPastYearsRemoved(t *testing.T) {
	fake := &replayFake{}
	m := replayModel(t, fake, pastYears())
	if m.(Model).replay != replayOn || !m.(Model).hist.HasReplay() {
		t.Fatal("past years kept before should be in at start")
	}
	m = hit(m, press("h"), press("a"))
	if !strings.Contains(screenText(m), "REMOVE YOUR PAST YEARS?") {
		t.Fatalf("a should ask to remove them:\n%s", screenText(m))
	}
	if kept := hit(m, keyEsc); kept.(Model).replay != replayOn || fake.deleted != 0 {
		t.Error("esc should keep them")
	}
	m, cmd := m.Update(press("y"))
	run(cmd)
	if fake.deleted != 1 || m.(Model).replay != replayOff || m.(Model).hist.HasReplay() {
		t.Errorf("y should delete them: deleted %d", fake.deleted)
	}
	if !strings.Contains(screenText(m), "Past years removed from your history") {
		t.Error("the removal should be said")
	}
}

// Steam gives nothing, or can't be reached: said in light red, nothing kept.
func TestPastYearsFailures(t *testing.T) {
	for _, tt := range []struct {
		fake replayFake
		want string
	}{
		{replayFake{missing: []int{2022, 2023, 2024, 2025}}, "Steam gave no past years: set each year to Public in Steam Replay"},
		{replayFake{err: errors.New("dial tcp: no route")}, "Could not reach Steam for your past years"},
	} {
		fake := tt.fake
		m := hit(replayModel(t, &fake, nil), press("h"), press("a"))
		m, cmd := m.Update(press("y"))
		m = run1(m, cmd)
		view := m.(Model).View().Content
		if !strings.Contains(ansi.Strip(view), tt.want) || fake.saved != 0 || m.(Model).replay != replayOff {
			t.Errorf("want %q, nothing kept, back to off:\n%s", tt.want, ansi.Strip(view))
		}
		if line := lineWith(view, tt.want[:20]); !strings.Contains(line, "38;2;255;123;114m") {
			t.Errorf("the message should be light red: %q", line)
		}
	}
}

// Without a way to fetch (no data folder), a does nothing and isn't offered.
func TestPastYearsUnavailable(t *testing.T) {
	var m tea.Model = New(Options{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 160, Height: 50})
	m, _ = m.Update(scanDoneMsg{res: library.Result{Games: []library.Game{{AppID: 1, Name: "Alpha", Installed: true}}}})
	m = hit(m, press("h"), press("a"))
	if m.(Model).replay != replayOff || strings.Contains(screenText(m), "Older years?") {
		t.Error("with no way to fetch, nothing should be offered")
	}
}

// ? lists a: in full on wide windows, short on narrow ones, fitting both.
func TestPastYearsInFullHelp(t *testing.T) {
	for _, tt := range []struct {
		width int
		want  string
	}{{100, "past years (in history)"}, {80, "years"}} {
		m, _ := withOpener(t, nil, portal2)
		var tm tea.Model = m
		tm, _ = tm.Update(tea.WindowSizeMsg{Width: tt.width, Height: 30})
		full := screenText(hit(tm, press("?")))
		if !strings.Contains(full, tt.want) || !strings.Contains(full, "quit") {
			t.Errorf("width %d: ? should list %q and still fit:\n%s", tt.width, tt.want, full)
		}
	}
}
