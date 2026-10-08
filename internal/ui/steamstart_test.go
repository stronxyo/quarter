package ui

import (
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/steam"
)

// fakeSteam stands in for Steam's client: what it says about itself, and
// what Quarter asked of it.
type fakeSteam struct {
	c      steam.Client
	known  bool
	starts int
	plays  []uint32
	opened []string
	clock  time.Time
}

// quietModel is a ready model on fakeSteam, with Portal 2 installed and
// selected, and its clock at f.clock.
func quietModel(t *testing.T, f *fakeSteam, s config.Settings) Model {
	t.Helper()
	f.clock = time.Date(2026, 10, 8, 12, 0, 0, 0, time.Local)
	m := New(Options{
		Open:        func(uri string) error { f.opened = append(f.opened, uri); return nil },
		SteamClient: func() (steam.Client, bool) { return f.c, f.known },
		StartSteam:  func() error { f.starts++; return nil },
		PlayQuiet:   func(id uint32) error { f.plays = append(f.plays, id); return nil },
		Settings:    s,
	})
	m.now = func() time.Time { return f.clock }
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 120, Height: 30})
	tm, _ = tm.Update(scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: []library.Game{portal2}}})
	return tm.(Model)
}

var enter = tea.KeyPressMsg{Code: tea.KeyEnter}

// playKey presses enter and hands the quiet play or the link to the fake,
// the way the program would, leaving the timers.
func playKey(t *testing.T, m Model) Model {
	t.Helper()
	tm, cmd := m.Update(enter)
	m = tm.(Model)
	for _, msg := range immediate(cmd) {
		tm, _ = m.Update(msg)
		m = tm.(Model)
	}
	return m
}

// immediate runs the commands in cmd that answer at once (opening a link,
// a quiet play), and skips the timers.
func immediate(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	var msgs []tea.Msg
	done := make(chan tea.Msg, 1)
	go func() { done <- cmd() }()
	select {
	case msg := <-done:
		if batch, ok := msg.(tea.BatchMsg); ok {
			for _, c := range batch {
				msgs = append(msgs, immediate(c)...)
			}
			return msgs
		}
		if msg != nil {
			msgs = append(msgs, msg)
		}
	case <-time.After(50 * time.Millisecond): // a timer
	}
	return msgs
}

// Quarter starts Steam in the tray as it opens, only when Steam is closed,
// Steam says whether it runs, and the setting isn't "normal".
func TestSteamStartsAtOpen(t *testing.T) {
	tests := []struct {
		name    string
		c       steam.Client
		known   bool
		setting string
		starts  int
	}{
		{"closed", steam.Client{}, true, "", 1},
		{"already running", steam.Client{Running: true, SignedIn: true}, true, "", 0},
		{"set to normal", steam.Client{}, true, steamNormal, 0},
		{"Steam doesn't say (Linux, macOS)", steam.Client{}, false, "", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeSteam{c: tt.c, known: tt.known}
			m := quietModel(t, f, config.Settings{SteamStart: tt.setting})
			for _, msg := range immediate(m.startSteamAtOpen()) {
				tm, _ := m.Update(msg)
				m = tm.(Model)
			}
			if f.starts != tt.starts {
				t.Errorf("Steam started %d times, want %d", f.starts, tt.starts)
			}
			if got := m.booting(); got != (tt.starts == 1) {
				t.Errorf("booting = %v after the start", got)
			}
		})
	}
	if New(Options{}).startSteamAtOpen() != nil {
		t.Error("without a Steam that says whether it runs, nothing is started")
	}
}

// Play goes quietly when Steam is closed or still starting from Quarter's
// own start, and by the steam:// link, as before, everywhere else.
func TestPlayPicksQuietOrLink(t *testing.T) {
	ready := steam.Client{Running: true, SignedIn: true}
	tests := []struct {
		name    string
		c       steam.Client
		known   bool
		setting string
		booted  bool // Quarter started Steam a moment ago
		quiet   bool
	}{
		{"Steam closed", steam.Client{}, true, "", false, true},
		{"Steam ready", ready, true, "", false, false},
		{"Steam signing in after Quarter's start", steam.Client{Running: true}, true, "", true, true},
		{"right after Quarter's start, last run's account still showing", ready, true, "", true, true},
		{"set to normal, Steam closed", steam.Client{}, true, steamNormal, false, false},
		{"Steam doesn't say", steam.Client{}, false, "", false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &fakeSteam{c: tt.c, known: tt.known}
			m := quietModel(t, f, config.Settings{SteamStart: tt.setting})
			if tt.booted {
				m.boot = steamBoot{at: f.clock.Add(-time.Second)}
			}
			m = playKey(t, m)
			if tt.quiet {
				if !slices.Equal(f.plays, []uint32{620}) || len(f.opened) != 0 {
					t.Fatalf("want a quiet play only: plays %v, links %v", f.plays, f.opened)
				}
				if text, _ := m.noticeText(); !strings.HasPrefix(text, "Opening Steam") {
					t.Errorf("message %q, want Opening Steam with the dots", text)
				}
				return
			}
			if len(f.plays) != 0 || !slices.Equal(f.opened, []string{"steam://rungameid/620"}) {
				t.Fatalf("want the link only: plays %v, links %v", f.plays, f.opened)
			}
			if text, _ := m.noticeText(); !strings.Contains(text, "Launching Portal 2") {
				t.Errorf("message %q, want today's Launching line", text)
			}
		})
	}
}

// Quarter's start ends once Steam is seen signed in, after signing in or
// after bootReady (ActiveUser is stale right after a start); then play
// goes by the link.
func TestBootEndsWhenSignedIn(t *testing.T) {
	f := &fakeSteam{c: steam.Client{Running: true}, known: true}
	m := quietModel(t, f, config.Settings{})
	m.boot = steamBoot{at: f.clock}
	m = m.notedClient(f.c)
	if !m.booting() || !m.boot.sawOut {
		t.Fatal("Steam signing in: the start is still under way")
	}
	f.c.SignedIn = true
	if m = m.notedClient(f.c); m.booting() {
		t.Fatal("signed in after signing in: the start is over")
	}
	m = playKey(t, m)
	if len(f.plays) != 0 || len(f.opened) != 1 {
		t.Errorf("Steam ready: want the link, got plays %v, links %v", f.plays, f.opened)
	}

	m.boot = steamBoot{at: f.clock}
	if m.notedClient(f.c).booting() == false {
		t.Error("an account showing straight after the start may be last run's: still under way")
	}
	f.clock = f.clock.Add(bootReady)
	if m.notedClient(f.c).booting() {
		t.Error("an account showing bootReady after the start is this run's: over")
	}
}

// The wait ends when the game runs, and a second play while it waits
// does nothing.
func TestLaunchWaitEnds(t *testing.T) {
	f := &fakeSteam{known: true}
	m := playKey(t, quietModel(t, f, config.Settings{}))
	m = playKey(t, m)
	if len(f.plays) != 1 || len(f.opened) != 0 {
		t.Fatalf("a second play while waiting: plays %v, links %v", f.plays, f.opened)
	}
	tm, _ := m.Update(pollMsg{asked: true, running: 620})
	m = tm.(Model)
	if m.launch.appID != 0 {
		t.Fatal("the game runs: the wait should be over")
	}
	if text, _ := m.noticeText(); strings.Contains(text, "Steam") {
		t.Errorf("message %q after the game started, want none", text)
	}
}

// At launchCheck and launchGive Quarter looks at why there's no game.
func TestLaunchChecks(t *testing.T) {
	t.Run("Steam not running", func(t *testing.T) {
		f := &fakeSteam{known: true}
		m := playKey(t, quietModel(t, f, config.Settings{}))
		f.clock = f.clock.Add(launchCheck)
		m, _ = m.checkLaunch(steam.Client{})
		text, wrong := m.noticeText()
		if m.launch.appID != 0 || !wrong || text != "Steam didn't start. Press enter to try again." {
			t.Errorf("got %q (wrong %v, waiting %v)", text, wrong, m.launch.appID != 0)
		}
		m = playKey(t, m)
		if len(f.plays) != 2 {
			t.Errorf("enter tries again: plays %v", f.plays)
		}
	})

	t.Run("Steam still signing in", func(t *testing.T) {
		f := &fakeSteam{known: true}
		m := playKey(t, quietModel(t, f, config.Settings{}))
		c := steam.Client{Running: true}
		f.clock = f.clock.Add(launchCheck - time.Second)
		if m, _ = m.checkLaunch(c); m.launch.checked {
			t.Fatal("nothing is looked at before launchCheck")
		}
		f.clock = f.clock.Add(time.Second)
		m, cmd := m.checkLaunch(c)
		if cmd != nil || m.launch.text != "Steam is still signing in" || !strings.HasPrefix(m.launchText(), "Steam is still signing in") {
			t.Errorf("signing in: no resend, the message says so; got %q, resend %v", m.launch.text, cmd != nil)
		}
		f.clock = f.clock.Add(launchGive - launchCheck)
		m, _ = m.checkLaunch(c)
		if text, wrong := m.noticeText(); m.launch.appID != 0 || !wrong || text != "Steam hasn't signed in; open it from the tray." {
			t.Errorf("at launchGive got %q", text)
		}
	})

	t.Run("Steam signed in, no game", func(t *testing.T) {
		f := &fakeSteam{known: true}
		m := playKey(t, quietModel(t, f, config.Settings{}))
		c := steam.Client{Running: true, SignedIn: true}
		f.clock = f.clock.Add(launchCheck)
		m, cmd := m.checkLaunch(c)
		for _, msg := range immediate(cmd) {
			tm, _ := m.Update(msg)
			m = tm.(Model)
		}
		if !slices.Equal(f.plays, []uint32{620, 620}) || m.launch.text != "Still opening Steam" {
			t.Fatalf("the game is sent once more: plays %v, message %q", f.plays, m.launch.text)
		}
		f.clock = f.clock.Add(pollEvery)
		if _, cmd = m.checkLaunch(c); cmd != nil {
			t.Error("only one resend")
		}
		f.clock = f.clock.Add(launchGive - launchCheck)
		m, _ = m.checkLaunch(c)
		want := "Steam is open but didn't start Portal 2; it may be updating. Click the logo to see Steam, or press enter to try again."
		if text, wrong := m.noticeText(); m.launch.appID != 0 || !wrong || text != want {
			t.Errorf("at launchGive got %q", text)
		}
	})
}

// Steam's state is asked only while Quarter waits on it, so idle rounds
// cost what they did.
func TestPollAsksSteamOnlyWhileWaiting(t *testing.T) {
	f := &fakeSteam{c: steam.Client{Running: true, SignedIn: true}, known: true}
	m := quietModel(t, f, config.Settings{})
	m.runningApp = func() (uint32, bool) { return 0, true }
	if msg := m.pollCmd(0)().(pollMsg); msg.client != nil {
		t.Error("nothing waiting: Steam's state isn't asked")
	}
	m.launch = launchWait{appID: 620, since: f.clock, text: "Opening Steam"}
	if msg := m.pollCmd(0)().(pollMsg); msg.client == nil || !msg.client.SignedIn {
		t.Errorf("a game waiting: Steam's state comes with the round, got %+v", msg.client)
	}
}

// "steam_start" survives other settings being saved.
func TestSteamStartSettingKept(t *testing.T) {
	var saved []config.Settings
	m := New(Options{
		Settings:     config.Settings{SteamStart: steamNormal},
		SaveSettings: func(s config.Settings) error { saved = append(saved, s); return nil },
	})
	run(m.persist())
	if len(saved) != 1 || saved[0].SteamStart != steamNormal {
		t.Errorf("saved %+v, want steam_start kept", saved)
	}
}
