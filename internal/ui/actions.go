package ui

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/stronxyo/quarter/internal/steam"
)

// statusLifetime is how long an action's message stays in the footer.
const statusLifetime = 3 * time.Second

// openDoneMsg reports whether a steam:// link was handed off.
type openDoneMsg struct {
	id  int
	err error
}

// clearStatusMsg removes a status message, unless a newer one replaced it.
type clearStatusMsg struct{ id int }

// action runs the play, install or store key for the selected game.
// handled is false when msg is not an action key.
func (m Model) action(msg tea.KeyPressMsg) (_ Model, _ tea.Cmd, handled bool) {
	it, ok := m.selected()
	if !ok {
		return m, nil, false
	}
	var uri, status string
	switch {
	case key.Matches(msg, keys.Play) && it.Installed:
		uri, status = steam.PlayURI(it.AppID), "▶ Launching "+it.Name
	case key.Matches(msg, keys.Play), key.Matches(msg, keys.Install):
		uri, status = steam.InstallURI(it.AppID), "⬇ Opening install for "+it.Name
	case key.Matches(msg, keys.Store):
		uri, status = steam.StoreURI(it.AppID), "Opening the store page for "+it.Name
	default:
		return m, nil, false
	}
	m.statusID++
	m.status = status
	return m, openURI(m.open, uri, m.statusID), true
}

// openURI hands a link to the OS off the UI goroutine.
func openURI(open func(string) error, uri string, id int) tea.Cmd {
	return func() tea.Msg {
		if open == nil {
			return openDoneMsg{id: id, err: fmt.Errorf("opening links is not available")}
		}
		return openDoneMsg{id: id, err: open(uri)}
	}
}

func clearStatusAfter(id int) tea.Cmd {
	return tea.Tick(statusLifetime, func(time.Time) tea.Msg { return clearStatusMsg{id: id} })
}
