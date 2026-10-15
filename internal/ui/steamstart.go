package ui

// Steam starts quietly (Pablo, 2026-10-08; spec section "Steam starts
// quietly"). When Quarter opens with Steam closed, it starts Steam in the
// tray (-silent), so a game played soon after starts sooner. A game played
// while Steam is closed, or while the Steam Quarter started is still
// signing in, goes as "-silent -applaunch <id>", which Steam holds until it
// is ready, and the message line shows "Opening Steam" with the moving dots
// until the game runs. Once Steam runs, play goes as "-applaunch <id>"
// alone, which Steam starts without its "Launching..." window (Pablo,
// 2026-10-14). Where Steam doesn't say whether it runs (Linux, macOS), or
// with "steam_start": "normal", play opens steam:// links as before.

import (
	"log/slog"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/steam"
)

const (
	steamNormal = "normal"          // config.json's "steam_start": Steam opens with its window, as before
	bootReady   = 10 * time.Second  // Steam's ActiveUser is stale this soon after a start (it clears at 1.6 s)
	bootMax     = 120 * time.Second // a start not seen ready by then is no longer waited on
	launchCheck = 60 * time.Second  // no game by then: Quarter looks at why (Pablo: "make it 60")
	launchGive  = 120 * time.Second // still no game: Quarter stops waiting and says so
	launchDots  = dotsStep / 2      // the message's dots step with the logo band's
	launchSpace = "  "              // between the message and its dots
)

// steamBoot is a start of Steam that Quarter made, until Steam is seen
// signed in.
type steamBoot struct {
	at     time.Time // zero: no start under way
	sawOut bool      // Steam was seen running but not signed in yet
}

// launchWait is a game played quietly, waiting for Steam to start it.
type launchWait struct {
	appID   uint32 // 0: nothing waiting
	name    string
	since   time.Time
	checked bool   // the look at launchCheck was made
	text    string // the message before the dots
}

type (
	steamStartedMsg struct{ err error } // the start at opening was made
	launchSentMsg   struct{ err error } // a quiet play was handed to Steam
	launchDotsMsg   struct{}            // the message's dots step
)

// quietSteam reports whether Quarter starts Steam itself: Steam says
// whether it runs here, and the setting isn't "normal".
func (m Model) quietSteam() bool {
	return m.steamClient != nil && m.steamStartSetting != steamNormal
}

// startSteamAtOpen starts Steam in the tray as Quarter opens, when it is
// closed. It runs off the UI goroutine; nil when Quarter doesn't start Steam.
func (m Model) startSteamAtOpen() tea.Cmd {
	if !m.quietSteam() || m.startSteam == nil {
		return nil
	}
	client, start := m.steamClient, m.startSteam
	return func() tea.Msg {
		if c, known := client(); !known || c.Running {
			return nil
		}
		return steamStartedMsg{err: start()}
	}
}

func (m Model) steamStarted(msg steamStartedMsg) (Model, tea.Cmd) {
	if msg.err != nil {
		slog.Warn("steam start", "err", msg.err)
		return m, nil
	}
	m.boot = steamBoot{at: m.now()}
	return m, m.startPoll(pollEvery)
}

// booting reports a start Quarter made that Steam hasn't finished.
func (m Model) booting() bool {
	return !m.boot.at.IsZero() && m.now().Sub(m.boot.at) < bootMax
}

// notedClient takes what Steam said about itself: a start Quarter made ends
// once Steam is signed in. Right after a start ActiveUser still holds the
// last run's account, so signed in counts only after Steam was seen signing
// in, or after bootReady.
func (m Model) notedClient(c steam.Client) Model {
	if !m.booting() {
		m.boot = steamBoot{}
		return m
	}
	switch {
	case c.Running && !c.SignedIn:
		m.boot.sawOut = true
	case c.Running && (m.boot.sawOut || m.now().Sub(m.boot.at) >= bootReady):
		m.boot = steamBoot{}
	}
	return m
}

// playWay is how a game is played.
type playWay int

const (
	wayLink  playWay = iota // the steam:// link, as before
	wayQuiet                // "-silent -applaunch": Steam closed, or the start Quarter made still under way
	wayReady                // "-applaunch": Steam running, so no "Launching..." window
)

// wayToPlay picks how a game is played: quietly while Steam is closed or
// the start Quarter made is still under way, by -applaunch alone once
// Steam runs, and by the steam:// link where Steam doesn't say whether it
// runs or with "steam_start": "normal".
func (m Model) wayToPlay() playWay {
	if !m.quietSteam() {
		return wayLink
	}
	c, known := m.steamClient()
	switch {
	case !known:
	case !c.Running || m.notedClient(c).booting():
		if m.playQuiet != nil {
			return wayQuiet
		}
	case m.playReady != nil:
		return wayReady
	}
	return wayLink
}

// playGame plays g the way wayToPlay picks.
func (m Model) playGame(g library.Game) (Model, tea.Cmd, bool) {
	switch m.wayToPlay() {
	case wayQuiet:
		return m.launchQuietly(g)
	case wayReady:
		return m.launchReady(g)
	}
	return m.openFor(steam.PlayURI(g.AppID), markerRunning+" Launching "+g.Name)
}

// launchReady plays g through the running Steam by "-applaunch", which
// brings no "Launching..." window. If steam.exe can't be run, the steam://
// link starts it, window and all.
func (m Model) launchReady(g library.Game) (Model, tea.Cmd, bool) {
	m.statusID++
	m.status, m.statusWrong = markerRunning+" Launching "+g.Name, false
	play, open, id, appID := m.playReady, m.open, m.statusID, g.AppID
	return m, func() tea.Msg {
		err := play(appID)
		if err == nil {
			return openDoneMsg{id: id}
		}
		slog.Warn("steam applaunch", "err", err)
		return openURI(open, steam.PlayURI(appID), id)()
	}, true
}

// launchQuietly plays g through "-silent -applaunch", and waits for it with
// "Opening Steam" and the dots in the message line.
func (m Model) launchQuietly(g library.Game) (Model, tea.Cmd, bool) {
	if c, _ := m.steamClient(); !c.Running && !m.booting() {
		m.boot = steamBoot{at: m.now()} // this play starts Steam
	}
	m.launch = launchWait{appID: g.AppID, name: g.Name, since: m.now(), text: "Opening Steam"}
	m.statusID++
	m.status, m.statusWrong = "", false
	return m, tea.Batch(sendLaunch(m.playQuiet, g.AppID), m.startPoll(pollEvery), launchTick()), true
}

func sendLaunch(play func(uint32) error, appID uint32) tea.Cmd {
	return func() tea.Msg { return launchSentMsg{err: play(appID)} }
}

func launchTick() tea.Cmd {
	return tea.Tick(launchDots, func(time.Time) tea.Msg { return launchDotsMsg{} })
}

// launchSent takes the hand-off: Steam couldn't be started, the wait ends
// with why.
func (m Model) launchSent(msg launchSentMsg) (Model, tea.Cmd) {
	if msg.err == nil || m.launch.appID == 0 {
		return m, nil
	}
	slog.Warn("steam play", "err", msg.err)
	return m.stopLaunch("Could not start Steam: " + msg.err.Error()), nil
}

// launchDotsStep moves the dots while a game waits.
func (m Model) launchDotsStep() (Model, tea.Cmd) {
	if m.launch.appID == 0 {
		return m, nil
	}
	return m, launchTick()
}

// launchRunning ends the wait when the game waited for runs.
func (m Model) launchRunning(appID uint32) Model {
	if m.launch.appID != 0 && appID == m.launch.appID {
		m.launch = launchWait{}
	}
	return m
}

// checkLaunch looks at a waiting game on each round of the checks: at
// launchCheck, Steam not running ends the wait, Steam still signing in
// keeps it (the game is held until it has), and Steam signed in without
// the game gets it once more; at launchGive the wait ends and says why.
func (m Model) checkLaunch(c steam.Client) (Model, tea.Cmd) {
	w := m.launch
	if w.appID == 0 {
		return m, nil
	}
	waited := m.now().Sub(w.since)
	switch {
	case waited >= launchCheck && !c.Running:
		return m.stopLaunch("Steam didn't start. Press enter to try again."), nil
	case waited >= launchGive && !c.SignedIn:
		return m.stopLaunch("Steam hasn't signed in; open it from the tray."), nil
	case waited >= launchGive:
		return m.stopLaunch("Steam is open but didn't start " + w.name + "; it may be updating. Click the logo to see Steam, or press enter to try again."), nil
	case waited < launchCheck || w.checked:
		return m, nil
	case !c.SignedIn:
		m.launch.checked, m.launch.text = true, "Steam is still signing in"
		return m, nil
	}
	m.launch.checked, m.launch.text = true, "Still opening Steam"
	return m, sendLaunch(m.playQuiet, w.appID)
}

// stopLaunch ends the wait with text, in light red, until the next message.
func (m Model) stopLaunch(text string) Model {
	m.launch = launchWait{}
	m.statusID++
	m.status, m.statusWrong = text, true
	return m
}

// launchText is the waiting message with its dots: fixed places, filled one
// by one, as in the logo band, but the hover's small square, the band's
// blocks being too big beside text (Pablo, 2026-10-08: "the small square is
// good"); Consolas, Cascadia Mono and JetBrains Mono all have it.
func (m Model) launchText() string {
	n := int(m.now().UnixMilli()/dotsStep.Milliseconds()) % (dotSlots + 1)
	slots := make([]string, dotSlots)
	for i := range slots {
		slots[i] = " "
		if i < n {
			slots[i] = markerHover
		}
	}
	return m.launch.text + launchSpace + strings.Join(slots, " ")
}
