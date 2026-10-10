package ui

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stronxyo/quarter/internal/library"
)

var updatingGames = []library.Game{
	{AppID: 1, Name: "Alpha", Installed: true},
	{AppID: 2, Name: "Bravo", Installed: true, Updating: true},
}

func TestUpdatingGameMarked(t *testing.T) {
	m, _ := withOpener(t, nil, updatingGames...)
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 150, Height: 30})

	if line := lineWith(runningView(tm), "Bravo"); !strings.Contains(line, "↓ Bravo") || !strings.Contains(line, "↓ updating") {
		t.Errorf("Bravo's row should show it's updating: %q", line)
	}
	if line := lineWith(runningView(tm), "Alpha"); strings.Contains(line, "updating") {
		t.Errorf("Alpha isn't updating: %q", line)
	}
	blue := "38;2;137;180;250m" // #89B4FA
	for _, l := range strings.Split(tm.(Model).View().Content, "\n") {
		if strings.Contains(l, "Bravo") && !strings.Contains(l, blue) {
			t.Errorf("Bravo's row should be blue: %q", l)
		}
	}

	tm, _ = tm.Update(press("j")) // select Bravo
	if v := runningView(tm); !strings.Contains(v, "↓ UPDATING") || strings.Contains(v, "IN STEAM") {
		t.Errorf("the pane should say Bravo is updating:\n%s", runningView(tm))
	}
}

// Enter on an updating game says so and starts nothing.
func TestUpdatingGameNotLaunched(t *testing.T) {
	m, opened := withOpener(t, nil, updatingGames...)
	var tm tea.Model = m
	tm, _ = tm.Update(press("j"))
	tm, _ = tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
	if len(*opened) != 0 {
		t.Errorf("nothing should be opened, got %v", *opened)
	}
	if got := tm.(Model).status; got != "Bravo is updating in Steam" {
		t.Errorf("status: got %q", got)
	}
}

// While a game updates, the library is read again every 30 s; once no game
// is updating, the reads stop.
func TestUpdatePolling(t *testing.T) {
	m, _ := withOpener(t, nil, updatingGames...)
	if !m.updateWatch {
		t.Fatal("an updating game should schedule a read")
	}
	if cmd := m.watchUpdates(); cmd != nil {
		t.Error("only one read is scheduled at a time")
	}

	// A read already running when the time comes: no second one.
	m.scanning = true
	m, cmd := m.updateTick()
	if cmd == nil || !m.updateWatch || !m.scanning {
		t.Error("with a read running, the next one should be scheduled instead")
	}

	m.scanning = false
	m, cmd = m.updateTick()
	if cmd == nil || !m.scanning {
		t.Fatal("the time for a read should start one")
	}

	// Steam finished: the game is playable, the reads stop.
	done := []library.Game{updatingGames[0], {AppID: 2, Name: "Bravo", Installed: true}}
	tm, _ := m.Update(scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: done}})
	if tm.(Model).updateWatch {
		t.Error("no game is updating: no more reads")
	}
	if line := lineWith(runningView(tm), "Bravo"); strings.Contains(line, "updating") {
		t.Errorf("Bravo is playable now: %q", line)
	}
}

// Every marker and message uses characters the common fonts have: the install
// message used U+2B07, which Cascadia Mono and JetBrains Mono both lack.
func TestInstallMessageArrow(t *testing.T) {
	m, _ := withOpener(t, nil, library.Game{AppID: 3, Name: "Charlie"})
	tm, _ := m.Update(press("i"))
	if got := tm.(Model).status; got != "↓ Opening install for Charlie" {
		t.Errorf("status: got %q", got)
	}
}
