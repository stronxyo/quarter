package ui

// Moving fast through the list (Pablo, 2026-09-30, from the Scroll lab: "lets
// go with c the moving dots"). A logo is read only once the selection has
// landed on a game; the games next to it are then prepared, so one step to a
// neighbour shows its logo at once; at most keptLogos stay drawn. While a key
// is held, the band drops the last logo and shows three dots filling in, as
// it does after landing when a logo takes a while.

import (
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

const (
	logoSettle = 120 * time.Millisecond // a pause this long on a game is a landing
	quickStep  = 150 * time.Millisecond // two moves this close are a held key
	dotsWait   = 200 * time.Millisecond // after landing, the dots only for a logo this late
	dotsStep   = 180 * time.Millisecond // one dot more each step, then none
	dotSlots   = 3
	dot        = "██"
	keptLogos  = 24 // a converted picture is about 13 KB; 5,000 kept would be 65 MB
)

// neighbourSteps are the games prepared around the selection, nearest first.
var neighbourSteps = []int{1, -1, 2, -2}

type (
	settleMsg struct{}
	dotsMsg   struct{}
)

// moved notes a change of selection, and checks back once it may have landed.
func (m *Model) moved() tea.Cmd {
	m.prevMove, m.lastMove = m.lastMove, m.now()
	return tea.Tick(m.settle, func(time.Time) tea.Msg { return settleMsg{} })
}

// landed reports whether the selection has stayed on its game long enough
// for its logo to be read.
func (m Model) landed() bool {
	return m.now().Sub(m.lastMove) >= m.settle
}

// holding reports a held key: the selection moved twice in quick steps and
// hasn't landed yet.
func (m Model) holding() bool {
	return !m.landed() && m.lastMove.Sub(m.prevMove) < quickStep
}

// showsDots reports whether the band shows the dots: while a key is held, or
// once landed, when the logo is still on its way after dotsWait.
func (m Model) showsDots() bool {
	if !m.showsLogos() {
		return false
	}
	if m.holding() {
		return true
	}
	g, ok := m.selected()
	if !ok || m.now().Sub(m.lastMove) < dotsWait {
		return false
	}
	_, cols, rows, _ := m.logoBand()
	_, state := m.logoFor(logoKey{g.AppID, cols, rows})
	return state == logoLoading
}

// dotsView is the band with the dots: fixed places, filled one by one, so
// they never shift.
func (m Model) dotsView(cols, rows int) string {
	n := int(m.now().UnixMilli()/dotsStep.Milliseconds()) % (dotSlots + 1)
	slots := make([]string, dotSlots)
	for i := range slots {
		slots[i] = strings.Repeat(" ", len([]rune(dot)))
		if i < n {
			slots[i] = dot
		}
	}
	return lipgloss.Place(cols, rows, lipgloss.Center, lipgloss.Center, starStyle.Render(strings.Join(slots, "  ")))
}

// startDots starts the dots' steps when they show, once.
func (m *Model) startDots() tea.Cmd {
	if m.dotsOn || !m.showsDots() {
		return nil
	}
	m.dotsOn = true
	return tea.Tick(dotsStep/2, func(time.Time) tea.Msg { return dotsMsg{} })
}

func (m Model) dotsTick() (Model, tea.Cmd) {
	if !m.showsDots() {
		m.dotsOn = false
		return m, nil
	}
	return m, tea.Tick(dotsStep/2, func(time.Time) tea.Msg { return dotsMsg{} })
}

// prepareNeighbours reads the logos of the games next to the selection, from
// disk only, once the selected game's own logo is done.
func (m *Model) prepareNeighbours() tea.Cmd {
	if !m.landed() || m.histView != histOff || !m.showsLogos() || m.sixel && m.now().Sub(m.resizedAt) < m.debounce {
		return nil
	}
	g, ok := m.selected()
	_, cols, rows, _ := m.logoBand()
	if !ok || m.logoPending[logoKey{g.AppID, cols, rows}] {
		return nil
	}
	items := m.list.VisibleItems()
	var cmds []tea.Cmd
	for _, d := range neighbourSteps {
		j := m.list.Index() + d
		if j < 0 || j >= len(items) {
			continue
		}
		it, ok := items[j].(gameItem)
		if !ok || m.logoNone[it.AppID] || m.logoNoDisk[it.AppID] {
			continue
		}
		key := logoKey{it.AppID, cols, rows}
		if m.hasLogo(key) || m.logoPending[key] {
			continue
		}
		m.logoPending[key] = true
		draw := drawLogo(m.logos, key, m.logoBlocks, false, m.pictureSize(cols, rows))
		cmds = append(cmds, func() tea.Msg {
			done, _ := draw().(logoDoneMsg)
			done.neighbour = true
			return done
		})
	}
	return tea.Batch(cmds...)
}

// keepLogo notes a drawn logo and drops the oldest past keptLogos, never the
// selected game's.
func (m *Model) keepLogo(key logoKey) {
	order := slices.DeleteFunc(slices.Clone(m.logoOrder), func(k logoKey) bool {
		_, ok := m.logoCache[k]
		return k == key || !ok
	})
	order = append(order, key)
	sel, _ := m.selected()
	for len(order) > keptLogos {
		i := 0
		if order[0].appID == sel.AppID {
			i = 1
		}
		delete(m.logoCache, order[i])
		order = slices.Delete(order, i, i+1)
	}
	m.logoOrder = order
}
