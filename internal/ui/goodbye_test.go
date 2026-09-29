package ui

import (
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/intro"
)

// playing is a model on a clock the test moves, 160 wide.
func playing(t *testing.T, s config.Settings) (Model, *time.Time) {
	t.Helper()
	clock := time.Date(2026, 9, 28, 20, 0, 0, 0, time.UTC)
	m := New(Options{Settings: s})
	m.now = func() time.Time { return clock }
	next, _ := m.Update(tea.WindowSizeMsg{Width: 160, Height: 44})
	return next.(Model), &clock
}

func steamRuns(m Model, id uint32) Model {
	next, _ := m.Update(runningMsg{appID: id})
	return next.(Model)
}

var rng = rand.New(rand.NewPCG(5, 6))

// Played time adds up every game Steam ran while Quarter was open, a game
// still running at the goodbye included.
func TestPlayedThisSession(t *testing.T) {
	m, clock := playing(t, config.Settings{})
	m = steamRuns(m, 252950)
	*clock = clock.Add(90 * time.Minute)
	m = steamRuns(m, 0)
	*clock = clock.Add(20 * time.Minute) // browsing, not playing
	m = steamRuns(m, 620)
	*clock = clock.Add(40 * time.Minute) // still running as Quarter closes
	text, _ := m.Farewell(rng)
	if got := ansi.Strip(text); !strings.Contains(got, "played this session: 2h 10m") {
		t.Errorf("90 + 40 minutes played, got:\n%s", got)
	}
}

// Nothing played, no played line; just the goodbye.
func TestGoodbyeWithoutPlaying(t *testing.T) {
	m, _ := playing(t, config.Settings{})
	text, _ := m.Farewell(rng)
	if text == "" || strings.Contains(text, "played") {
		t.Errorf("a goodbye without a played line, got:\n%s", ansi.Strip(text))
	}
}

// "goodbye": "off" prints nothing; a name prints that one.
func TestGoodbyeSetting(t *testing.T) {
	m, _ := playing(t, config.Settings{Goodbye: goodbyeOff})
	if text, _ := m.Farewell(rng); text != "" {
		t.Error(`"off" should print nothing`)
	}
	m, _ = playing(t, config.Settings{Goodbye: "keep-the-change"})
	i, _ := intro.FindGoodbye("keep-the-change")
	if text, s := m.Farewell(rng); text != intro.Goodbye(i, 160, 0) || s.LastGoodbye != "keep-the-change" {
		t.Error("a named goodbye should print that one")
	}
}

// The goodbye is saved, and the next quit picks another.
func TestGoodbyeNotRepeated(t *testing.T) {
	for i := range intro.GoodbyeCount() {
		m, _ := playing(t, config.Settings{LastGoodbye: intro.GoodbyeKey(i)})
		text, s := m.Farewell(rng)
		if s.LastGoodbye == intro.GoodbyeKey(i) || text == intro.Goodbye(i, 160, 0) {
			t.Errorf("%s came twice in a row", intro.GoodbyeKey(i))
		}
		if _, ok := intro.FindGoodbye(s.LastGoodbye); !ok {
			t.Errorf("the goodbye shown should be saved, got %q", s.LastGoodbye)
		}
	}
}
