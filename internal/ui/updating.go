package ui

import (
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stronxyo/quarter/internal/library"
)

// updatePoll is how often Quarter reads the library again while Steam is
// updating a game, so the game turns playable on screen when Steam is done.
const updatePoll = 30 * time.Second

// updatePollMsg is the time for the next read while a game updates.
type updatePollMsg struct{}

// watchUpdates schedules the next read when a game is updating and none is
// scheduled yet.
func (m *Model) watchUpdates() tea.Cmd {
	if m.updateWatch || !slices.ContainsFunc(m.installed, func(g library.Game) bool { return g.Updating }) {
		return nil
	}
	m.updateWatch = true
	return tea.Tick(updatePoll, func(time.Time) tea.Msg { return updatePollMsg{} })
}

// updateTick reads the library again, unless a read is running already
// (its result sets the next one).
func (m Model) updateTick() (Model, tea.Cmd) {
	m.updateWatch = false
	if m.scanning {
		return m, m.watchUpdates()
	}
	m.scanning = true
	return m, scan(m.steamPath)
}
