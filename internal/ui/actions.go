package ui

import (
	"fmt"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/steam"
)

// statusLifetime is how long an action's message stays in the footer.
const statusLifetime = 3 * time.Second

// openDoneMsg reports whether a link (steam:// or a web page) was handed off.
type openDoneMsg struct {
	id  int
	uri string
	err error
}

// clearStatusMsg removes a status message, unless a newer one replaced it.
type clearStatusMsg struct{ id int }

// gameState is what the selected game allows, for its keys and key line.
type gameState int

const (
	gameNone         gameState = iota // no game selected
	gameRunning                       // running: no game key acts on it
	gameUpdating                      // Steam is updating it: only the store page
	gameInstalled                     // play, uninstall, store page
	gameNotInstalled                  // install (enter or i), store page
)

// stateOf tells what the selected game allows.
func (m Model) stateOf() (library.Game, gameState) {
	g, ok := m.selected()
	switch {
	case !ok:
		return g, gameNone
	case m.running != 0 && g.AppID == m.running:
		return g, gameRunning
	case g.Updating:
		return g, gameUpdating
	case g.Installed:
		return g, gameInstalled
	}
	return g, gameNotInstalled
}

// action runs a game key (play, install, uninstall, store page, website,
// SteamDB) for the selected game, as far as its state allows (see the
// spec's "Keys that follow the game"); SteamDB opens for any game, running
// too, and the website for any game the store gave one for (without, w does
// nothing, as it isn't on the key line). handled is false when msg is not
// a game key.
func (m Model) action(msg tea.KeyPressMsg) (_ Model, _ tea.Cmd, handled bool) {
	play, install, uninstall, store := key.Matches(msg, keys.Play), key.Matches(msg, keys.Install), key.Matches(msg, keys.Uninstall), key.Matches(msg, keys.Store)
	web, db := key.Matches(msg, keys.Website), key.Matches(msg, keys.SteamDB)
	if !play && !install && !uninstall && !store && !web && !db {
		return m, nil, false
	}
	it, state := m.stateOf()
	switch {
	case state == gameNone:
		return m, nil, true
	case db:
		return m.openFor(fmt.Sprintf(config.SteamDBURL, it.AppID), "Opening SteamDB for "+it.Name)
	case web:
		if site := m.selectedWebsite(); site != "" {
			return m.openFor(site, "Opening "+siteName(site))
		}
		return m, nil, true
	case state == gameRunning:
		return m, m.note(it.Name + " is already running"), true
	case store:
		if _, s := m.storeStatus(it.AppID); s == storeNotFound {
			return m, m.note(it.Name + " is not on the Steam store"), true
		}
		return m.openFor(steam.StoreURI(it.AppID), "Opening the store page for "+it.Name)
	case state == gameUpdating && play:
		return m, m.note(it.Name + " is updating in Steam"), true
	case state == gameInstalled && play && m.launch.appID != 0:
		return m, nil, true // a game is waiting for Steam: nothing until it starts
	case state == gameInstalled && play && m.playQuietly():
		return m.launchQuietly(it)
	case state == gameInstalled && play:
		return m.openFor(steam.PlayURI(it.AppID), markerRunning+" Launching "+it.Name)
	case state == gameInstalled && uninstall:
		return m.uninstall(it)
	case state == gameNotInstalled && (play || install):
		return m.openFor(steam.InstallURI(it.AppID), markerUpdating+" Opening install for "+it.Name)
	}
	return m, nil, true // a key this game doesn't take: nothing happens
}

// selectedWebsite is the selected game's official site, "" until the store
// has given one.
func (m Model) selectedWebsite() string {
	g, ok := m.selected()
	if !ok {
		return ""
	}
	if info, state := m.storeStatus(g.AppID); state == storeReady {
		return info.Website
	}
	return ""
}

// uninstall asks Steam to uninstall g on the second u: the first only
// warns, in the message line, for as long as the message stays.
func (m Model) uninstall(g library.Game) (Model, tea.Cmd, bool) {
	if m.uninstallArmed == g.AppID {
		m.uninstallArmed = 0
		return m.openFor(steam.UninstallURI(g.AppID), "Opening uninstall for "+g.Name)
	}
	m.uninstallArmed = g.AppID
	m.statusID++
	m.status, m.statusWrong = "Press u again to uninstall "+g.Name, true
	return m, clearStatusAfter(m.statusID), true
}

// openFor hands uri to Steam with status in the message line.
func (m Model) openFor(uri, status string) (Model, tea.Cmd, bool) {
	m.statusID++
	m.status, m.statusWrong = status, false
	return m, openURI(m.open, uri, m.statusID), true
}

// note shows a short message that needs no action, and clears it later.
func (m *Model) note(text string) tea.Cmd {
	m.statusID++
	m.status, m.statusWrong = text, false
	return clearStatusAfter(m.statusID)
}

// openURI hands a link to the OS off the UI goroutine.
func openURI(open func(string) error, uri string, id int) tea.Cmd {
	return func() tea.Msg {
		if open == nil {
			return openDoneMsg{id: id, uri: uri, err: fmt.Errorf("opening links is not available")}
		}
		return openDoneMsg{id: id, uri: uri, err: open(uri)}
	}
}

func clearStatusAfter(id int) tea.Cmd {
	return tea.Tick(statusLifetime, func(time.Time) tea.Msg { return clearStatusMsg{id: id} })
}
