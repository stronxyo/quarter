package ui

import (
	"log/slog"
	"slices"

	tea "charm.land/bubbletea/v2"

	"github.com/stronxyo/quarter/internal/art"
	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/library"
)

// applySettings restores the last run's favourites, sort and filter. Unknown
// or empty values keep the defaults.
func (m *Model) applySettings(s config.Settings) {
	for _, id := range s.Favourites {
		m.favs[id] = true
	}
	if order, ok := library.ParseSortOrder(s.Sort); ok {
		m.sortBy, m.sortDesc = order, s.SortDesc
	}
	if f, ok := library.ParseFilter(s.Filter); ok {
		m.filterBy = f
	}
	m.paneHidden = s.HidePane
	m.logosHidden = s.HideLogos
	if b, ok := art.ParseBlocks(s.LogoBlocks); ok {
		m.logoBlocks = b
	}
	m.splashSeen = s.SplashSeen
}

// settings snapshots what should be remembered for the next run.
func (m Model) settings() config.Settings {
	favs := make([]uint32, 0, len(m.favs))
	for id := range m.favs {
		favs = append(favs, id)
	}
	slices.Sort(favs) // stable file contents
	return config.Settings{
		Favourites: favs,
		Sort:       m.sortBy.Key(),
		SortDesc:   m.sortDesc,
		Filter:     m.filterBy.Key(),
		HidePane:   m.paneHidden,
		HideLogos:  m.logosHidden,
		LogoBlocks: m.logoBlocks.Key(),
		SplashSeen: m.splashSeen,
	}
}

// persist saves the settings off the UI goroutine. A failure is logged: the
// app keeps working, it just won't remember this change.
func (m Model) persist() tea.Cmd {
	if m.saveSettings == nil {
		return nil
	}
	save, s := m.saveSettings, m.settings()
	return func() tea.Msg {
		if err := save(s); err != nil {
			slog.Warn("settings", "err", err)
		}
		return nil
	}
}

// rebuild makes the list's games: every installed game, plus each candidate
// the store has confirmed is a game or a demo, named by the store.
func (m *Model) rebuild() {
	games := slices.Clone(m.installed)
	for _, g := range m.uninstalled {
		e, ok := m.storeInfo[g.AppID]
		if !ok || e.Failed || (e.Info.Type != "game" && e.Info.Type != "demo") || e.Info.Name == "" {
			continue
		}
		g.Name = e.Info.Name
		games = append(games, g)
	}
	m.games = games
	m.decorate()
}

// decorate copies what Quarter knows beyond the scan onto the games (stars
// from the settings, the demo flag from the store), so filtering and drawing
// can read it from each game.
func (m *Model) decorate() {
	for i := range m.games {
		id := m.games[i].AppID
		m.games[i].Favourite = m.favs[id]
		m.games[i].Demo = m.storeInfo[id].Info.Type == "demo"
	}
}

// toggleFavourite stars or unstars the selected game and saves the change.
func (m Model) toggleFavourite() (Model, tea.Cmd) {
	it, ok := m.selected()
	if !ok {
		return m, nil
	}
	if m.favs[it.AppID] {
		delete(m.favs, it.AppID)
	} else {
		m.favs[it.AppID] = true
	}
	m.decorate()
	return m, tea.Batch(m.refreshList(), m.persist())
}
