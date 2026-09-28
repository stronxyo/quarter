package ui

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/stronxyo/quarter/internal/intro"
)

// The start screen (Batch 8, A3): an old PC checking itself as it boots,
// the quarter_ logo, and "insert quarter" blinking like an arcade cabinet
// waiting for a coin. On each start one of the intro animations leads into
// it, at random and never the same twice in a row. It shows the real scan
// as it happens, so it never adds waiting of its own: the first key skips
// the intro, the next one starts. config.json's "intro" turns the intros
// off ("off") or picks one by name.

// blinkEvery is how long insert quarter stays lit, and stays dark.
const blinkEvery = 530 * time.Millisecond

// introFrame is how often an intro is drawn.
const introFrame = time.Second / 30

// splashLeader is the width of a boot line, dots included, before its result.
const splashLeader = 30

// introOff is the "intro" setting that turns the intros off.
const introOff = "off"

// blinkMsg turns insert quarter on or off; introTickMsg draws the next frame.
type (
	blinkMsg     struct{}
	introTickMsg struct{}
)

// blink schedules the next blink while the splash is up.
func (m Model) blink() tea.Cmd {
	if !m.splash {
		return nil
	}
	return tea.Tick(blinkEvery, func(time.Time) tea.Msg { return blinkMsg{} })
}

// pickIntro chooses the intro for this start: none when the setting is off,
// the named one, or a random one other than last time's.
func (m *Model) pickIntro(r *rand.Rand) {
	m.introIndex = -1
	switch i, named := intro.Find(m.introSetting); {
	case !m.splash || m.introSetting == introOff:
		return
	case named:
		m.introIndex = i
	default:
		last, ok := intro.Find(m.lastIntro)
		if !ok {
			last = -1
		}
		m.introIndex = intro.Pick(last, r)
	}
	m.lastIntro = intro.Key(m.introIndex)
}

// introTick schedules the next frame while an intro plays.
func (m Model) introTick() tea.Cmd {
	if !m.splash || m.introIndex < 0 || m.introDone {
		return nil
	}
	return tea.Tick(introFrame, func(time.Time) tea.Msg { return introTickMsg{} })
}

// introPending reports whether this start's intro is still to play.
func (m Model) introPending() bool {
	return m.splash && m.introIndex >= 0 && !m.introDone
}

// startIntro starts the intro as soon as the window's size is known, so its
// first frame is the first thing on screen: nothing of the start screen
// shows before it. Too small a window has no intro.
func (m *Model) startIntro() {
	if !m.introPending() || m.introRun != nil || m.width == 0 {
		return
	}
	sc, ok := m.startScreen()
	if !ok {
		m.introDone = true
		return
	}
	m.introRun, m.introStart = sc.Start(m.introIndex, uint64(m.now().UnixNano())), m.now()
}

// stepIntro ends the intro when its time is up.
func (m Model) stepIntro() (Model, tea.Cmd) {
	if !m.introPending() {
		return m, nil
	}
	m.startIntro()
	if m.introDone {
		return m, nil
	}
	if m.now().Sub(m.introStart) >= intro.Length(m.introIndex) {
		m.introDone = true
		return m, nil
	}
	return m, m.introTick()
}

// splashKey handles a key on the splash. "Press any key" means any: q too
// (pressed out of habit, it mustn't quit), so only ctrl+c quits. During
// the intro a key skips to the start screen; after it, a key starts.
func (m Model) splashKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if m.introIndex >= 0 && !m.introDone {
		m.introDone = true
		return m, nil
	}
	m.splash, m.splashSeen = false, true
	return m, m.persist()
}

// startScreen lays the start screen out for the window, with the system
// check as its header.
func (m Model) startScreen() (intro.Screen, bool) {
	steam, games := "...", "..."
	switch m.state {
	case stateReady:
		steam, games = "OK", fmt.Sprintf("%d games", len(m.games))
	case stateFailed:
		steam, games = "NOT FOUND", "-"
	}
	check := func(label, result string) intro.Line {
		dots := strings.Repeat(".", max(splashLeader-len(label)-2, 3))
		return intro.Line{{Text: label, Tone: intro.Plain}, {Text: " " + dots + " ", Tone: intro.Faint}, {Text: result, Tone: intro.Bright}}
	}
	return intro.NewScreen(m.width, m.height, "QUARTER SYSTEM CHECK", []intro.Line{
		check("Looking for Steam", steam),
		check("Reading libraries", games),
	})
}

// splashView is the whole screen while the splash is up: the intro's frame
// while it plays, then the start screen.
func (m Model) splashView() string {
	if m.width == 0 || m.height == 0 {
		return "" // the window's size isn't known yet: nothing, rather than a wrong screen
	}
	sc, ok := m.startScreen()
	if !ok {
		return m.compactSplash()
	}
	if m.introPending() {
		if m.introRun == nil {
			return "" // not started: never a glimpse of the start screen first
		}
		if frame, done := sc.Frame(m.introRun, m.now().Sub(m.introStart)); !done {
			return frame
		}
	}
	return sc.Rest(m.blinkOn)
}

// compactSplash is the start screen for windows too small for the logo box:
// the checks, the badge and insert quarter as text.
func (m Model) compactSplash() string {
	steam := "..."
	switch m.state {
	case stateReady:
		steam = "OK"
	case stateFailed:
		steam = "NOT FOUND"
	}
	coin := strings.Repeat(" ", len("insert quarter"))
	if m.blinkOn {
		coin = activeTabStyle.Render("insert quarter")
	}
	screen := lipgloss.JoinVertical(lipgloss.Center,
		dimStyle.Render("QUARTER SYSTEM CHECK"), "",
		textStyle.Render("Looking for Steam ")+activeTabStyle.Render(steam), "",
		titleStyle.Render("quarter_"), "", coin, dimStyle.Render("press any key"))
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, screen)
}
