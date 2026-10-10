package ui

import (
	"path/filepath"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stronxyo/quarter/internal/library"
)

func pollModel(t *testing.T, ask func() (uint32, bool)) Model {
	t.Helper()
	m := New(Options{RunningApp: ask, Open: func(string) error { return nil }})
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	tm, _ = tm.Update(scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: updatingGames}})
	return tm.(Model)
}

// What a round asks: the running game, and the download log once it is
// watched; nothing to ask, no round.
func TestPollRound(t *testing.T) {
	if (Model{}).pollCmd(0) != nil {
		t.Error("nothing to ask means no round")
	}
	m := pollModel(t, func() (uint32, bool) { return 620, true })
	msg := m.pollCmd(0)().(pollMsg)
	if !msg.asked || msg.running != 620 || msg.downloads != nil {
		t.Errorf("only the running game before the log is watched: %+v", msg)
	}

	root := t.TempDir()
	writeFile(t, filepath.Join(root, "logs", "content_log.txt"), "")
	m.steamRoot, m.dlWatch = root, true
	msg = m.pollCmd(0)().(pollMsg)
	if !msg.asked || msg.downloads == nil || !msg.downloads.logOK {
		t.Errorf("one round brings both: %+v", msg)
	}
}

// Where Steam doesn't say which game runs, it isn't asked again.
func TestPollStopsAsking(t *testing.T) {
	m := pollModel(t, func() (uint32, bool) { return 0, false })
	tm, cmd := m.Update(m.pollCmd(0)())
	if tm.(Model).runningApp != nil || cmd != nil {
		t.Error("an unknown answer stops the asking, and with nothing else to check the rounds stop")
	}
}

// One round at a time: the next is scheduled when one comes back.
func TestPollOneAtATime(t *testing.T) {
	m := pollModel(t, func() (uint32, bool) { return 0, true })
	if !m.polling {
		t.Fatal("the first round is on its way from the start")
	}
	if m.startPoll(pollEvery) != nil {
		t.Error("no second round while one is on its way")
	}
	tm, cmd := m.Update(pollMsg{asked: true})
	if !tm.(Model).polling || cmd == nil {
		t.Error("a round that came back schedules the next")
	}
}

// A round that found nothing new shows the screen drawn last; anything
// else, or the screen getting old, draws it again.
func TestQuietPollKeepsScreen(t *testing.T) {
	clock := time.Date(2026, 9, 30, 12, 0, 0, 0, time.Local)
	m := pollModel(t, func() (uint32, bool) { return 0, true })
	m.now = func() time.Time { return clock }
	first := m.View()

	tm, _ := m.Update(pollMsg{asked: true}) // nothing new
	m = tm.(Model)
	if _, ok := m.keptView(); !ok {
		t.Fatal("after a quiet round the screen drawn last should be kept")
	}
	if m.View().Content != first.Content {
		t.Error("the kept screen should be the one drawn last")
	}

	clock = clock.Add(viewMaxAge)
	if _, ok := m.keptView(); ok {
		t.Error("a screen this old is drawn again")
	}
	clock = clock.Add(-viewMaxAge)

	tm, _ = m.Update(pollMsg{asked: true, running: 1}) // Alpha started
	if _, ok := tm.(Model).keptView(); ok {
		t.Error("a game starting changes the screen")
	}
	tm, _ = m.Update(press("j"))
	if _, ok := tm.(Model).keptView(); ok {
		t.Error("a key changes the screen")
	}
}

// Two models updated from the same one never take each other's screen.
func TestKeptScreenPerModel(t *testing.T) {
	m := pollModel(t, func() (uint32, bool) { return 0, true })
	a, _ := m.Update(press("j"))
	b, _ := m.Update(press("k"))
	if a.(Model).viewGen == b.(Model).viewGen {
		t.Error("each update draws its own screen")
	}
	a.View()
	if _, ok := b.(Model).keptView(); ok {
		t.Error("b should not show a's screen")
	}
}

// While Steam's download log is there, it sees a download end: the reads
// every 30 s while a game updates wait, and go on as the fallback.
func TestUpdateReadsWaitForLog(t *testing.T) {
	m := pollModel(t, nil)
	m.logOK = true
	m, cmd := m.updateTick()
	if m.scanning || cmd == nil || !m.updateWatch {
		t.Error("with the log there, no read, but the next tick is scheduled")
	}
}
