package ui

import (
	"math/rand/v2"
	"time"

	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/intro"
)

// The goodbye printed as Quarter closes (see internal/intro): one line at
// random, never last time's, and the time played in games this session.

// goodbyeOff is the "goodbye" setting that turns it off.
const goodbyeOff = "off"

// fallbackWidth is the terminal's width when the window's size never came.
const fallbackWidth = 80

// countPlayed adds a finished run of the game playing now to the session's
// played time, and starts timing the next game, if one is starting.
func (m *Model) countPlayed(next uint32) {
	if m.running != 0 && !m.playingSince.IsZero() {
		m.played += m.now().Sub(m.playingSince)
	}
	m.playingSince = time.Time{}
	if next != 0 {
		m.playingSince = m.now()
	}
}

// Farewell is what to print as Quarter closes, and the settings to save
// with it (they remember which goodbye it was). The text is empty when the
// goodbye is off.
func (m Model) Farewell(r *rand.Rand) (string, config.Settings) {
	i, ok := intro.FindGoodbye(m.goodbyeSetting)
	switch {
	case m.goodbyeSetting == goodbyeOff:
		return "", m.settings()
	case !ok:
		last, known := intro.FindGoodbye(m.lastGoodbye)
		if !known {
			last = -1
		}
		i = intro.PickGoodbye(last, r)
	}
	m.lastGoodbye = intro.GoodbyeKey(i)
	played := m.played
	if m.running != 0 && !m.playingSince.IsZero() {
		played += m.now().Sub(m.playingSince) // a game still running
	}
	width := m.width
	if width <= 0 {
		width = fallbackWidth
	}
	return intro.Goodbye(i, width, played), m.settings()
}
