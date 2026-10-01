package ui

import tea "charm.land/bubbletea/v2"

// runningMsg reports the game Steam is running, 0 for none. The poll
// (poll.go) asks Steam every pollEvery.
type runningMsg struct{ appID uint32 }

// applyRunning takes the game Steam is running: a change counts the play
// time and, when a game stopped, reads its new playtime (afterplay.go).
func (m Model) applyRunning(msg runningMsg) (Model, tea.Cmd) {
	if msg.appID == m.running {
		return m, nil
	}
	var reread tea.Cmd
	m.countPlayed(msg.appID)
	if m.running != 0 {
		reread = m.gameStopped(m.running)
	}
	m.running = msg.appID
	m.list.SetDelegate(m.delegate())
	return m, reread
}
