package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stronxyo/quarter/internal/library"
)

// When a game stops, Steam writes its new playtime and last played date to
// localconfig.vdf a few seconds later (seen 2026-09-29: Rocket League closed
// at 02:48:29, the file written at 02:48:43, Steam still running). Quarter
// reads the library again after it stops, and again until the game's numbers
// change, so the list and pane show the session just played.

// afterPlayChecks are the waits before each re-read: 10, 20, 40, 70 and
// 130 seconds after the game stops, then Quarter gives up (r still rescans).
var afterPlayChecks = []time.Duration{10 * time.Second, 10 * time.Second, 20 * time.Second, 30 * time.Second, 60 * time.Second}

// afterPlay is the game waiting for its new numbers; appID 0 for none.
type afterPlay struct {
	appID      uint32
	lastPlayed time.Time     // its numbers before the re-read
	playtime   time.Duration //
	check      int           // which of afterPlayChecks is next
	scanning   bool          // the scan running now is this check's
}

// afterPlayMsg is the time for check number check of appID.
type afterPlayMsg struct {
	appID uint32
	check int
}

// gameStopped starts watching for the numbers of the game that just stopped.
// A second game stopping takes over: any re-read picks up both.
func (m *Model) gameStopped(id uint32) tea.Cmd {
	m.afterPlay = afterPlay{appID: id}
	if g, ok := m.gameByID(id); ok {
		m.afterPlay.lastPlayed, m.afterPlay.playtime = g.LastPlayed, g.Playtime
	}
	return afterPlayAfter(id, 0)
}

func afterPlayAfter(id uint32, check int) tea.Cmd {
	return tea.Tick(afterPlayChecks[check], func(time.Time) tea.Msg {
		return afterPlayMsg{appID: id, check: check}
	})
}

// afterPlayTick re-reads the library for the game waiting, unless it was
// replaced or a rescan is running already (then it asks again shortly).
func (m Model) afterPlayTick(msg afterPlayMsg) (Model, tea.Cmd) {
	if msg.appID == 0 || msg.appID != m.afterPlay.appID || msg.check != m.afterPlay.check {
		return m, nil
	}
	if m.scanning {
		return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg { return msg })
	}
	m.scanning, m.afterPlay.scanning = true, true
	return m, scan(m.steamPath)
}

// afterPlayScanned is called with each finished scan: when it was a check's
// and the game's numbers haven't moved yet, the next check is set.
func (m *Model) afterPlayScanned() tea.Cmd {
	a := m.afterPlay
	if a.appID == 0 || !a.scanning {
		return nil
	}
	m.afterPlay.scanning = false
	g, ok := m.gameByID(a.appID)
	if ok && (!g.LastPlayed.Equal(a.lastPlayed) || g.Playtime != a.playtime) || a.check+1 >= len(afterPlayChecks) {
		m.afterPlay = afterPlay{}
		return nil
	}
	m.afterPlay.check++
	return afterPlayAfter(a.appID, m.afterPlay.check)
}

// gameByID is the game with this appid from the last scan.
func (m Model) gameByID(id uint32) (library.Game, bool) {
	for _, list := range [][]library.Game{m.installed, m.uninstalled} {
		for _, g := range list {
			if g.AppID == id {
				return g, true
			}
		}
	}
	return library.Game{}, false
}
