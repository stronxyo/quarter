package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/webapi"
)

var (
	notInstalled = library.Game{AppID: 400, Name: "Portal"}
	updatingGame = library.Game{AppID: 70, Name: "Half-Life", Installed: true, Updating: true}
)

// keyLine is the line of keycaps at the bottom of the screen.
func keyLine(m tea.Model) string { return lineWith(ansi.Strip(m.(Model).View().Content), "q quit") }

// pressAll presses keys: each key's link is handed over (it comes back at
// once); a command that doesn't, like a message's clear timer, is left.
func pressAll(m tea.Model, keys ...tea.KeyPressMsg) tea.Model {
	for _, k := range keys {
		var cmd tea.Cmd
		m, cmd = m.Update(k)
		if cmd == nil {
			continue
		}
		done := make(chan tea.Msg, 1)
		go func() { done <- cmd() }()
		select {
		case msg := <-done:
			if o, ok := msg.(openDoneMsg); ok {
				m, _ = m.Update(o)
			}
		case <-time.After(100 * time.Millisecond):
		}
	}
	return m
}

func TestKeysFollowTheGame(t *testing.T) {
	enter := tea.KeyPressMsg{Code: tea.KeyEnter}
	tests := []struct {
		name    string
		game    library.Game
		running bool
		keys    []tea.KeyPressMsg
		line    []string // on the key line
		notLine []string // not on it
		opened  []string // links sent to Steam, in order
		status  string
	}{
		{"installed: enter plays", portal2, false, []tea.KeyPressMsg{enter}, []string{"enter play", "u uninstall"}, []string{"i install"}, []string{"steam://rungameid/620"}, "Launching Portal 2"},
		{"installed: i does nothing", portal2, false, []tea.KeyPressMsg{press("i")}, nil, nil, nil, ""},
		{"not installed: enter installs", notInstalled, false, []tea.KeyPressMsg{enter}, []string{"enter/i install"}, []string{"enter play", "uninstall"}, []string{"steam://install/400"}, "install for Portal"},
		{"not installed: i installs", notInstalled, false, []tea.KeyPressMsg{press("i")}, nil, nil, []string{"steam://install/400"}, ""},
		{"not installed: u does nothing", notInstalled, false, []tea.KeyPressMsg{press("u"), press("u")}, nil, nil, nil, ""},
		{"running: nothing acts", portal2, true, []tea.KeyPressMsg{enter, press("i"), press("u"), press("u"), press("s")}, []string{"▶ running"}, []string{"enter play", "uninstall"}, nil, "Portal 2 is already running"},
		{"updating: enter says so", updatingGame, false, []tea.KeyPressMsg{enter, press("i"), press("u"), press("u")}, []string{"↓ updating"}, []string{"enter play", "uninstall", "install"}, nil, "Half-Life is updating"},
		{"updating: the store page works", updatingGame, false, []tea.KeyPressMsg{press("s")}, nil, nil, []string{"steam://store/70"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, opened := withOpener(t, nil, tt.game)
			var tm tea.Model = m
			tm, _ = tm.Update(tea.WindowSizeMsg{Width: 150, Height: 30})
			if tt.running {
				tm, _ = tm.Update(runningMsg{appID: tt.game.AppID})
			}
			line := keyLine(tm)
			for _, w := range tt.line {
				if !strings.Contains(line, w) {
					t.Errorf("key line missing %q: %q", w, line)
				}
			}
			for _, w := range tt.notLine {
				if strings.Contains(line, w) {
					t.Errorf("key line shouldn't show %q: %q", w, line)
				}
			}
			tm = pressAll(tm, tt.keys...)
			if strings.Join(*opened, " ") != strings.Join(tt.opened, " ") {
				t.Errorf("opened %v, want %v", *opened, tt.opened)
			}
			if tt.status != "" && !strings.Contains(ansi.Strip(tm.(Model).View().Content), tt.status) {
				t.Errorf("message %q missing", tt.status)
			}
		})
	}
}

// No game at all: the key line offers no game keys, and they do nothing.
func TestKeysWithNoGame(t *testing.T) {
	m, opened := withOpener(t, nil)
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 150, Height: 30})
	line := keyLine(tm)
	for _, w := range []string{"enter", "install", "uninstall", "running"} {
		if strings.Contains(line, w) {
			t.Errorf("no game: the key line shouldn't show %q: %q", w, line)
		}
	}
	pressAll(tm, tea.KeyPressMsg{Code: tea.KeyEnter}, press("i"), press("u"), press("u"), press("s"))
	if len(*opened) != 0 {
		t.Errorf("nothing to act on, but opened %v", *opened)
	}
}

// The first u warns in light red; the second, within the message's time,
// uninstalls. The message going, or moving to another game, cancels it.
func TestUninstallTakesTwoPresses(t *testing.T) {
	other := library.Game{AppID: 999, Name: "Zeta", Installed: true}
	m, opened := withOpener(t, nil, portal2, other)
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 150, Height: 30})

	tm = pressAll(tm, press("u"))
	if len(*opened) != 0 {
		t.Fatal("the first u must not uninstall")
	}
	view := tm.(Model).View().Content
	if !strings.Contains(ansi.Strip(view), "Press u again to uninstall Portal 2") {
		t.Errorf("the warning is missing:\n%s", ansi.Strip(view))
	}
	if line := lineWith(view, "Press u again"); !strings.Contains(line, "38;2;255;123;114m") {
		t.Errorf("the warning should be light red: %q", line)
	}
	armed := pressAll(tm, press("u"))
	if strings.Join(*opened, " ") != "steam://uninstall/620" {
		t.Errorf("the second u should uninstall: opened %v", *opened)
	}
	if armed.(Model).uninstallArmed != 0 {
		t.Error("after uninstalling, the next u warns again")
	}

	// The message's time runs out: the next u only warns again.
	late, openedLate := withOpener(t, nil, portal2, other)
	tm = late
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 150, Height: 30})
	tm = pressAll(tm, press("u"))
	tm, _ = tm.Update(clearStatusMsg{id: tm.(Model).statusID})
	pressAll(tm, press("u"))
	if len(*openedLate) != 0 {
		t.Errorf("after the warning went, one u must not uninstall: opened %v", *openedLate)
	}

	// Moving to another game cancels it.
	moved, openedMoved := withOpener(t, nil, portal2, other)
	tm = moved
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 150, Height: 30})
	tm = pressAll(tm, press("u"), tea.KeyPressMsg{Code: tea.KeyDown}, press("u"))
	if len(*openedMoved) != 0 {
		t.Errorf("moving away should cancel: opened %v", *openedMoved)
	}
	if !strings.Contains(ansi.Strip(tm.(Model).View().Content), "Press u again to uninstall Zeta") {
		t.Error("u on the new game should warn about it")
	}
}

// Typing a search, Enter plays or installs the game under the cursor at
// once; the key line says what Enter will do, and esc clears.
func TestSearchEnterActsAtOnce(t *testing.T) {
	for _, tt := range []struct {
		game library.Game
		line string
		want string
	}{
		{portal2, "enter play", "steam://rungameid/620"},
		{notInstalled, "enter install", "steam://install/400"},
	} {
		m, opened := withOpener(t, nil, tt.game)
		var tm tea.Model = m
		tm, _ = tm.Update(tea.WindowSizeMsg{Width: 150, Height: 30})
		tm = pressAll(tm, press("/"), press("p"))
		line := lineWith(ansi.Strip(tm.(Model).View().Content), "esc clear")
		if !strings.Contains(line, tt.line) || strings.Contains(line, "tab filter") {
			t.Errorf("%s: while typing, the key line should be %q and esc clear: %q", tt.game.Name, tt.line, line)
		}
		tm, cmd := tm.Update(tea.KeyPressMsg{Code: tea.KeyEnter})
		run(cmd) // the link is handed over inside the batch
		if tm.(Model).list.SettingFilter() {
			t.Errorf("%s: Enter should end typing", tt.game.Name)
		}
		if strings.Join(*opened, " ") != tt.want {
			t.Errorf("%s: opened %v, want %s", tt.game.Name, *opened, tt.want)
		}
	}
}

// s on a game the store has no page for says so.
func TestStorePageNotOnStore(t *testing.T) {
	m := New(Options{Open: func(string) error { t.Error("nothing should open"); return nil },
		StoreCache: map[uint32]webapi.StoreEntry{620: {Failed: true, Fetched: time.Now()}}})
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 150, Height: 30})
	tm, _ = tm.Update(scanDoneMsg{res: library.Result{Games: []library.Game{portal2}}})
	tm = pressAll(tm, press("s"))
	if !strings.Contains(ansi.Strip(tm.(Model).View().Content), "Portal 2 is not on the Steam store") {
		t.Errorf("s should say the store has no page:\n%s", ansi.Strip(tm.(Model).View().Content))
	}
}

// o on a window with no room for logos says so and changes nothing; with
// the pane hidden by d on a wide window it still toggles.
func TestLogosKeyOnNarrowWindow(t *testing.T) {
	m, _ := withOpener(t, nil, portal2) // 80 wide: no logo band
	tm := pressAll(m, press("o"))
	if tm.(Model).logosHidden || !strings.Contains(ansi.Strip(tm.(Model).View().Content), "Logos show on wider windows") {
		t.Error("o on a narrow window should explain and change nothing")
	}
	var wide tea.Model = m
	wide, _ = wide.Update(tea.WindowSizeMsg{Width: 170, Height: 45})
	wide = pressAll(wide, press("d"), press("o"))
	if !wide.(Model).logosHidden {
		t.Error("with room, o should hide logos even while d hid the pane")
	}
}

// The pop-up pane shows how to close it.
func TestPopUpKeyLine(t *testing.T) {
	m, _ := withOpener(t, nil, portal2)
	tm := pressAll(m, press("d"))
	if line := keyLine(tm); !strings.Contains(line, "d/esc close") {
		t.Errorf("the pop-up's key line should say how to close it: %q", line)
	}
}

// ? lists every key, the list's page and end keys too.
func TestFullHelpListsEverything(t *testing.T) {
	m, _ := withOpener(t, nil, portal2)
	full := ansi.Strip(pressAll(m, press("?")).(Model).View().Content)
	for _, want := range []string{"pgup/dn", "home/end", "play/install", "i", "install", "uninstall x2", "store page", "history", "rescan", "quit"} {
		if !strings.Contains(full, want) {
			t.Errorf("? missing %q:\n%s", want, full)
		}
	}
}
