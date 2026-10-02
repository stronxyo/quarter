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
//
// As it waits, like a cabinet's attract mode (Pablo, 2026-10-02): a band
// of light crosses the logo every 10 to 20 seconds, and an intro plays
// again every 45 to 60 seconds, at random and never the one before. Any
// key during a replay starts. Both are timed by the blink, so nothing else
// ticks while the screen waits.

// shineGap and againGap are the shortest and longest waits for the next
// shine, counted from the last one's end, and for the next intro, counted
// from the screen coming to rest.
var (
	shineGap = [2]time.Duration{10 * time.Second, 20 * time.Second}
	againGap = [2]time.Duration{45 * time.Second, 60 * time.Second}
)

// blinkEvery is how long insert quarter stays lit, and stays dark.
const blinkEvery = 530 * time.Millisecond

// introFrame is how often an intro is drawn.
const introFrame = time.Second / 30

// splashLeader is the width of a boot line, dots included, before its result.
const splashLeader = 30

// introOff is the "intro" setting that turns the intros off.
const introOff = "off"

// blinkMsg turns insert quarter on or off; introTickMsg and shineTickMsg
// draw the next frame.
type (
	blinkMsg     struct{}
	introTickMsg struct{}
	shineTickMsg struct{}
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

// waiting runs the start screen's attract mode, on each blink: once the
// screen rests, it sets when the logo shines and when an intro plays
// again, then starts each when its time comes.
func (m Model) waiting() (Model, tea.Cmd) {
	if !m.splash || m.introPending() {
		return m, nil
	}
	if _, ok := m.startScreen(); !ok {
		return m, nil // too small a window: no logo to shine, no intro
	}
	now := m.now()
	switch {
	case m.shineAt.IsZero(): // just came to rest
		m.shineAt, m.againAt = now.Add(m.between(shineGap)), now.Add(m.between(againGap))
	case m.introSetting != introOff && !now.Before(m.againAt):
		m.pickIntro(m.rng)
		m.introDone, m.introRun, m.introAgain = false, nil, true
		m.shineAt, m.shineStart = time.Time{}, time.Time{} // set again once it rests
		m.startIntro()
		return m, m.introTick()
	case m.shineStart.IsZero() && !now.Before(m.shineAt):
		m.shineStart = now
		m.shineAt = now.Add(intro.ShineLength + m.between(shineGap))
		return m, m.shineTick()
	}
	return m, nil
}

// between is a random wait from gap[0] to gap[1].
func (m Model) between(gap [2]time.Duration) time.Duration {
	return gap[0] + time.Duration(m.rng.Int64N(int64(gap[1]-gap[0])+1))
}

// shineTick schedules the shine's next frame.
func (m Model) shineTick() tea.Cmd {
	return tea.Tick(introFrame, func(time.Time) tea.Msg { return shineTickMsg{} })
}

// stepShine ends the shine when it has crossed the logo.
func (m Model) stepShine() (Model, tea.Cmd) {
	if !m.splash || m.shineStart.IsZero() {
		return m, nil
	}
	if m.now().Sub(m.shineStart) >= intro.ShineLength {
		m.shineStart = time.Time{}
		return m, nil
	}
	return m, m.shineTick()
}

// splashKey handles a key on the splash. "Press any key" means any: q too
// (pressed out of habit, it mustn't quit), so only ctrl+c quits. During
// the intro a key skips to the start screen; after it, a key starts.
func (m Model) splashKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if msg.String() == "ctrl+c" {
		return m, tea.Quit
	}
	if m.introIndex >= 0 && !m.introDone && !m.introAgain {
		m.introDone = true
		return m, nil
	}
	m.splash, m.splashSeen = false, true
	return m, m.persist()
}

// startScreen lays the start screen out for the window, with the system
// check as its header. The layout is kept (startMemo) while the window and
// the check stay the same: it was laid out again on every blink and
// animation frame (D3 profile).
func (m Model) startScreen() (intro.Screen, bool) {
	steam, games := "...", "..."
	switch m.state {
	case stateReady:
		steam, games = "OK", fmt.Sprintf("%d games", len(m.games))
	case stateFailed:
		steam, games = "NOT FOUND", "-"
	}
	key := fmt.Sprintf("%dx%d|%s|%s", m.width, m.height, steam, games)
	if m.start != nil && m.start.key == key {
		return m.start.sc, m.start.ok
	}
	check := func(label, result string) intro.Line {
		dots := strings.Repeat(".", max(splashLeader-len(label)-2, 3))
		return intro.Line{{Text: label, Tone: intro.Plain}, {Text: " " + dots + " ", Tone: intro.Faint}, {Text: result, Tone: intro.Bright}}
	}
	sc, ok := intro.NewScreen(m.width, m.height, "QUARTER SYSTEM CHECK", []intro.Line{
		check("Looking for Steam", steam),
		check("Reading libraries", games),
	})
	if m.start != nil {
		*m.start = startMemo{key: key, sc: sc, ok: ok}
	}
	return sc, ok
}

// startMemo is the start screen laid out last, for the window and system
// check in key, and its two resting pictures (insert quarter dark, lit) once
// drawn. Shared by the model's copies; the key keeps it right for each.
type startMemo struct {
	key  string
	sc   intro.Screen
	ok   bool
	rest [2]string
}

// restView is the start screen as it waits, kept once drawn: the blink only
// ever shows these two pictures.
func (m Model) restView(sc intro.Screen) string {
	i := 0
	if m.blinkOn {
		i = 1
	}
	if m.start == nil {
		return sc.Rest(m.blinkOn)
	}
	if m.start.rest[i] == "" {
		m.start.rest[i] = sc.Rest(m.blinkOn)
	}
	return m.start.rest[i]
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
	if !m.shineStart.IsZero() {
		if frame, done := sc.Shine(m.blinkOn, m.now().Sub(m.shineStart)); !done {
			return frame
		}
	}
	return m.restView(sc)
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
