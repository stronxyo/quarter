package ui

import (
	"time"

	tea "charm.land/bubbletea/v2"
)

// runningPoll is how often Quarter asks Steam which game is running.
const runningPoll = 2 * time.Second

// runningMsg reports the game Steam is running, 0 for none.
type runningMsg struct{ appID uint32 }

// checkRunning asks after a delay, off the UI goroutine. A nil ask (or a
// platform where Steam doesn't say) stops the polling.
func checkRunning(ask func() (uint32, bool), after time.Duration) tea.Cmd {
	if ask == nil {
		return nil
	}
	return tea.Tick(after, func(time.Time) tea.Msg {
		id, known := ask()
		if !known {
			return nil
		}
		return runningMsg{appID: id}
	})
}
