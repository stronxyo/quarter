package ui

import (
	"context"
	"errors"
	"log/slog"
	"maps"
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
	storeOffline                    // the last request failed; retried later
)

// storeStatus reports what the pane can show for appID.
func (m Model) storeStatus(appID uint32) (webapi.StoreInfo, storeState) {
	if m.fetching == appID && appID != 0 {
		if e, ok := m.storeInfo[appID]; ok && !e.Failed {
			return e.Info, storeReady // refreshing old details: keep showing them
		}
		return webapi.StoreInfo{}, storeLoading
	}
	if m.offline[appID] {
		return webapi.StoreInfo{}, storeOffline
	}
	e, ok := m.storeInfo[appID]
	switch {
	case !ok:
		return webapi.StoreInfo{}, storeUnknown
	case e.Failed:
		return webapi.StoreInfo{}, storeNotFound
	}
	return e.Info, storeReady
}

// needsStore reports whether appID's details are missing or stale.
func (m Model) needsStore(appID uint32) bool {
	if m.fetchStore == nil || appID == 0 || m.offline[appID] {
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
	if m.fetching != 0 || m.fetchStore == nil {
		return nil
	}
	id := uint32(0)
	if m.needsStore(m.wanted) {
		id = m.wanted
	} else {
		for _, g := range m.games {
			if m.needsStore(g.AppID) {
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
	var save tea.Cmd
	switch {
	case msg.err == nil:
		m.storeInfo[msg.appID] = webapi.StoreEntry{Info: msg.info, Fetched: msg.at}
		save = m.persistStore()
	case errors.Is(msg.err, webapi.ErrNotOnStore):
		m.storeInfo[msg.appID] = webapi.StoreEntry{Failed: true, Fetched: msg.at}
		save = m.persistStore()
	default:
		// A network problem says nothing about the game, so it isn't cached;
		// the game is skipped for this run instead.
		slog.Warn("store", "app", msg.appID, "err", msg.err)
		m.offline[msg.appID] = true
	}
	m.decorate()
	return m, tea.Batch(m.refreshList(), save, m.nextStoreFetch())
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
