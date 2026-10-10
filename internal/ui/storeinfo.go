package ui

import (
	"context"
	"errors"
	"log/slog"
	"maps"
	"slices"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/webapi"
)

// storeDebounce waits for the selection to settle, so scrolling past games
// doesn't send a request for each.
const storeDebounce = 250 * time.Millisecond

// storeWantMsg asks for a game's store details once the selection settled.
type storeWantMsg struct{ appID uint32 }

// storeDoneMsg carries one store answer.
type storeDoneMsg struct {
	appID uint32
	info  webapi.StoreInfo
	err   error
	at    time.Time
}

// storeState is what the UI knows about fetching one game's store details.
type storeState int

const (
	storeUnknown  storeState = iota // not fetched yet
	storeLoading                    // a request is out
	storeReady                      // details known
	storeNotFound                   // the store has no page for it
	storeOffline                    // requests are paused after a network failure; retried later
)

// storeStatus reports what the pane can show for appID. Details already
// known keep showing while they are refreshed or requests are paused.
func (m Model) storeStatus(appID uint32) (webapi.StoreInfo, storeState) {
	e, ok := m.storeInfo[appID]
	switch {
	case ok && !e.Failed:
		return e.Info, storeReady
	case m.fetching == appID && appID != 0:
		return webapi.StoreInfo{}, storeLoading
	case ok:
		return webapi.StoreInfo{}, storeNotFound
	case m.storePause.active(m.now()) || m.serverFailed[appID]:
		return webapi.StoreInfo{}, storeOffline
	}
	return webapi.StoreInfo{}, storeUnknown
}

// needsStore reports whether appID's details are missing or stale.
func (m Model) needsStore(appID uint32) bool {
	if m.fetchStore == nil || appID == 0 {
		return false
	}
	e, ok := m.storeInfo[appID]
	return !ok || !e.Fresh(m.now(), config.StoreRefresh, config.StoreRetry)
}

// wantStore schedules a fetch for the selected game once it stays selected.
func wantStore(appID uint32, after time.Duration) tea.Cmd {
	return tea.Tick(after, func(time.Time) tea.Msg { return storeWantMsg{appID: appID} })
}

// nextStoreFetch starts the next request: the game the user is looking at
// first, then, in the background, every other game in list order. One
// request is out at a time; the client's rate limit spaces them.
func (m *Model) nextStoreFetch() tea.Cmd {
	if m.fetching != 0 || m.fetchStore == nil || m.storePause.active(m.now()) {
		return nil // a paused store keeps the wanted game for when it resumes
	}
	id := uint32(0)
	if m.needsStore(m.wanted) {
		id = m.wanted
	} else {
		// Installed games first, then the candidates. A game the server
		// failed on waits for the user to select it, so one bad answer
		// doesn't hold up the rest.
		for _, g := range slices.Concat(m.installed, m.uninstalled) {
			if m.needsStore(g.AppID) && !m.serverFailed[g.AppID] {
				id = g.AppID
				break
			}
		}
	}
	m.wanted = 0
	if id == 0 {
		return nil
	}
	m.fetching = id
	fetch := m.fetchStore
	return func() tea.Msg {
		info, err := fetch(context.Background(), id)
		return storeDoneMsg{appID: id, info: info, err: err, at: time.Now()}
	}
}

// applyStore records an answer, saves the cache, and moves on to the next.
func (m Model) applyStore(msg storeDoneMsg) (Model, tea.Cmd) {
	m.fetching = 0
	var cmds []tea.Cmd
	switch {
	case msg.err == nil:
		m.storeInfo[msg.appID] = webapi.StoreEntry{Info: msg.info, Fetched: msg.at}
		m.storePause = pause{}
		delete(m.serverFailed, msg.appID)
		cmds = append(cmds, m.persistStore(), m.answered())
	case isNetFailure(msg.err):
		// Says nothing about the game, so nothing is cached: the store
		// pauses and the game is asked again when it resumes.
		slog.Warn("store", "app", msg.appID, "err", msg.err)
		if errors.Is(msg.err, webapi.ErrServer) {
			m.serverFailed[msg.appID] = true
		}
		if msg.appID == m.selectedID() {
			m.wanted = msg.appID
		}
		cmds = append(cmds, m.failed(&m.storePause, msg.err, false))
		if !errors.Is(msg.err, webapi.ErrUnreachable) {
			cmds = append(cmds, m.answered())
		}
	default:
		// Not on the store, or an answer that can't be read: remembered as
		// a failure, asked again after StoreRetry.
		if !errors.Is(msg.err, webapi.ErrNotOnStore) {
			slog.Warn("store", "app", msg.appID, "err", msg.err)
		}
		m.storeInfo[msg.appID] = webapi.StoreEntry{Failed: true, Fetched: msg.at}
		cmds = append(cmds, m.persistStore(), m.answered())
	}
	m.rebuild()
	return m, tea.Batch(append(cmds, m.refreshList(), m.nextStoreFetch())...)
}

// selectedID is the selected game's appid, 0 for none.
func (m Model) selectedID() uint32 {
	g, _ := m.selected()
	return g.AppID
}

// persistStore saves the store cache off the UI goroutine.
func (m Model) persistStore() tea.Cmd {
	if m.saveStore == nil {
		return nil
	}
	save, snapshot := m.saveStore, maps.Clone(m.storeInfo)
	return func() tea.Msg {
		if err := save(snapshot); err != nil {
			slog.Warn("store cache", "err", err)
		}
		return nil
	}
}
