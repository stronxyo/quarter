package ui

import (
	"errors"
	"math/rand/v2"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stronxyo/quarter/internal/webapi"
)

// Store details and logo downloads each pause after a network failure and
// try one request when the pause ends (see specs, Batch 8 B3). No network:
// 15 s, doubling, at most 5 minutes. Rate limited: the server's Retry-After,
// or 60 s, doubling, at most 10 minutes. A server error waits like no
// network. Any answer from either server means the network is back, so a
// pause for no network ends at once.
const (
	offlineFirst = 15 * time.Second
	offlineMost  = 5 * time.Minute
	busyFirst    = 60 * time.Second
	busyMost     = 10 * time.Minute
	pauseSteps   = 16 // doubling stops long before this; it guards the shift
)

// pause is one source's wait after failures.
type pause struct {
	until   time.Time
	step    int  // failures in a row
	offline bool // paused because no answer came
	busy    bool // paused because the server asked for fewer requests
}

func (p pause) active(now time.Time) bool { return now.Before(p.until) }

// resumeMsg ends a pause: the store's, or the logos'.
type resumeMsg struct{ logos bool }

// halfJitter waits between half and all of d, so retries spread out
// without ever coming back at once.
func halfJitter(d time.Duration) time.Duration {
	return d/2 + rand.N(d/2+1)
}

// failed pauses p after err, a failure that says nothing about the game,
// and returns the command that ends the pause.
func (m *Model) failed(p *pause, err error, logos bool) tea.Cmd {
	var limited *webapi.RateLimitError
	first, most := offlineFirst, offlineMost
	p.offline, p.busy = false, false
	var wait time.Duration
	switch {
	case errors.As(err, &limited):
		p.busy = true
		m.netDown = false // the server answered
		first, most = busyFirst, busyMost
		if limited.RetryAfter > 0 {
			wait = min(limited.RetryAfter, busyMost)
		}
	case errors.Is(err, webapi.ErrUnreachable):
		p.offline = true
		m.netDown = true
	default:
		m.netDown = false // the server answered, with an error
	}
	if wait == 0 {
		wait = m.jitter(min(first<<p.step, most))
	}
	p.step = min(p.step+1, pauseSteps)
	p.until = m.now().Add(wait)
	return tea.Tick(wait, func(time.Time) tea.Msg { return resumeMsg{logos: logos} })
}

// isNetFailure reports whether err is a failure of the network or the
// server, not an answer about the game.
func isNetFailure(err error) bool {
	var limited *webapi.RateLimitError
	return errors.As(err, &limited) || errors.Is(err, webapi.ErrUnreachable) || errors.Is(err, webapi.ErrServer)
}

// answered is called whenever a server answers: the network is up, so a
// pause for no network, the other source's too, ends now.
func (m *Model) answered() tea.Cmd {
	m.netDown = false
	var cmds []tea.Cmd
	if m.storePause.offline {
		m.storePause = pause{}
		cmds = append(cmds, func() tea.Msg { return resumeMsg{} })
	}
	if m.logoPause.offline {
		m.logoPause = pause{}
		cmds = append(cmds, func() tea.Msg { return resumeMsg{logos: true} })
	}
	return tea.Batch(cmds...)
}

// resume starts the source's next request once its pause has ended: the
// selected game first.
func (m Model) resume(msg resumeMsg) (Model, tea.Cmd) {
	if msg.logos {
		if m.logoPause.active(m.now()) {
			return m, nil // paused again since
		}
		if g, ok := m.selected(); ok {
			_, cols, rows, _ := m.logoBand()
			key := logoKey{g.AppID, cols, rows}
			if m.logoPending[key] {
				return m, func() tea.Msg { return logoWantMsg{key} }
			}
		}
		return m, nil
	}
	if m.storePause.active(m.now()) {
		return m, nil
	}
	if g, ok := m.selected(); ok && m.needsStore(g.AppID) {
		m.wanted = g.AppID
	}
	return m, m.nextStoreFetch()
}
