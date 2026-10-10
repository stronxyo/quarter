package ui

import (
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stronxyo/quarter/internal/library"
)

var launched = time.Date(2026, 9, 29, 2, 33, 18, 0, time.UTC)

// library with Rocket League at the given numbers.
func rocketLeague(playtime time.Duration, last time.Time) scanDoneMsg {
	return scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: []library.Game{
		{AppID: 252950, Name: "Rocket League", Installed: true, Playtime: playtime, LastPlayed: last},
	}}}
}

func step(t *testing.T, m Model, msg tea.Msg) Model {
	t.Helper()
	next, _ := m.Update(msg)
	return next.(Model)
}

// A game stops; Steam saves its playtime a little later: Quarter re-reads
// until the new numbers show, then stops re-reading.
func TestPlaytimeAfterGameStops(t *testing.T) {
	m := step(t, New(Options{}), tea.WindowSizeMsg{Width: 120, Height: 40})
	m = step(t, m, rocketLeague(45*time.Hour, launched))
	m = step(t, m, runningMsg{appID: 252950})
	if m.afterPlay.appID != 0 {
		t.Fatal("nothing to re-read while the game runs")
	}
	m = step(t, m, runningMsg{appID: 0})
	if m.afterPlay.appID != 252950 || m.afterPlay.check != 0 {
		t.Fatalf("the stopped game should be waited on, got %+v", m.afterPlay)
	}

	// First check: Steam hasn't written yet, so the next check is set.
	m = step(t, m, afterPlayMsg{appID: 252950, check: 0})
	if !m.scanning {
		t.Fatal("the check should rescan")
	}
	m = step(t, m, rocketLeague(45*time.Hour, launched))
	if m.afterPlay.check != 1 || m.scanning {
		t.Fatalf("unchanged numbers: the next check, got %+v", m.afterPlay)
	}

	// A stale tick (an earlier check) does nothing.
	if m2 := step(t, m, afterPlayMsg{appID: 252950, check: 0}); m2.scanning {
		t.Error("a stale check should not rescan")
	}

	// Second check: the session is saved, the pane shows it, no more checks.
	m = step(t, m, afterPlayMsg{appID: 252950, check: 1})
	m = step(t, m, rocketLeague(45*time.Hour+15*time.Minute, launched.Add(15*time.Minute)))
	if m.afterPlay.appID != 0 {
		t.Errorf("new numbers: stop re-reading, got %+v", m.afterPlay)
	}
	if g, _ := m.gameByID(252950); g.Playtime != 45*time.Hour+15*time.Minute {
		t.Errorf("playtime %v, want the new one", g.Playtime)
	}
}

// A rescan already running (r pressed): the check waits for it instead of
// starting another, and that rescan doesn't count as the check.
func TestAfterPlayWaitsForRescan(t *testing.T) {
	m := step(t, New(Options{}), tea.WindowSizeMsg{Width: 120, Height: 40})
	m = step(t, m, rocketLeague(45*time.Hour, launched))
	m = step(t, m, runningMsg{appID: 252950})
	m = step(t, m, runningMsg{appID: 0})
	m = step(t, m, press("r"))
	next, cmd := m.Update(afterPlayMsg{appID: 252950, check: 0})
	m = next.(Model)
	if m.afterPlay.scanning || cmd == nil {
		t.Fatal("the check should wait and ask again")
	}
	m = step(t, m, rocketLeague(45*time.Hour, launched))
	if m.afterPlay.appID != 252950 || m.afterPlay.check != 0 {
		t.Errorf("r's rescan is not the check, got %+v", m.afterPlay)
	}
}

// Steam never writes (or the game isn't in the library): Quarter gives up
// after the last check.
func TestAfterPlayGivesUp(t *testing.T) {
	m := step(t, New(Options{}), tea.WindowSizeMsg{Width: 120, Height: 40})
	m = step(t, m, rocketLeague(45*time.Hour, launched))
	m = step(t, m, runningMsg{appID: 252950})
	m = step(t, m, runningMsg{appID: 0})
	for i := range afterPlayChecks {
		if m.afterPlay.appID == 0 {
			t.Fatalf("gave up after %d checks, want %d", i, len(afterPlayChecks))
		}
		m = step(t, m, afterPlayMsg{appID: 252950, check: i})
		m = step(t, m, rocketLeague(45*time.Hour, launched))
	}
	if m.afterPlay.appID != 0 {
		t.Errorf("should give up after the last check, got %+v", m.afterPlay)
	}
}
