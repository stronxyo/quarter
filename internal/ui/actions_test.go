package ui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/library"
)

// withOpener returns a ready model whose link opener records what it was
// given, or fails with err when err is set.
func withOpener(t *testing.T, err error, games ...library.Game) (Model, *[]string) {
	t.Helper()
	opened := &[]string{}
	m := New(Options{Open: func(uri string) error {
		*opened = append(*opened, uri)
		return err
	}})
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	tm, _ = tm.Update(scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: games}})
	return tm.(Model), opened
}

// act presses a key and feeds the result of the open back in, the way the
// program would, without waiting for the message's clear timer.
func act(m tea.Model, k tea.KeyPressMsg) tea.Model {
	m, cmd := m.Update(k)
	if cmd == nil {
		return m
	}
	if msg, ok := cmd().(openDoneMsg); ok {
		m, _ = m.Update(msg) // its clear timer is returned, not run
	}
	return m
}

var portal2 = library.Game{AppID: 620, Name: "Portal 2", Installed: true}

func TestActionKeys(t *testing.T) {
	tests := []struct {
		name    string
		game    library.Game
		key     tea.KeyPressMsg
		wantURI string
		status  string
	}{
		{"enter plays", portal2, tea.KeyPressMsg{Code: tea.KeyEnter}, "steam://rungameid/620", "Launching Portal 2"},
		{"i installs", portal2, press("i"), "steam://install/620", "install for Portal 2"},
		{"s opens the store", portal2, press("s"), "steam://store/620", "store page for Portal 2"},
		{"enter on an uninstalled game installs", library.Game{AppID: 400, Name: "Portal"}, tea.KeyPressMsg{Code: tea.KeyEnter}, "steam://install/400", "install for Portal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, opened := withOpener(t, nil, tt.game)
			got := act(m, tt.key)
			if len(*opened) != 1 || (*opened)[0] != tt.wantURI {
				t.Errorf("opened %v, want [%s]", *opened, tt.wantURI)
			}
			if view := ansi.Strip(got.(Model).View().Content); !strings.Contains(view, tt.status) {
				t.Errorf("footer missing %q", tt.status)
			}
		})
	}
}

func TestActionKeysIgnoredWhileTypingSearch(t *testing.T) {
	m, opened := withOpener(t, nil, portal2)
	var tm tea.Model = m
	tm = act(tm, press("/"))
	tm = act(tm, press("s"))
	act(tm, press("i"))
	if len(*opened) != 0 {
		t.Errorf("keys typed into the search box opened %v", *opened)
	}
}

func TestFailedOpenShowsError(t *testing.T) {
	m, _ := withOpener(t, errors.New("no handler"), portal2)
	got := act(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if view := ansi.Strip(got.(Model).View().Content); !strings.Contains(view, "Could not open Steam: no handler") {
		t.Errorf("error not shown:\n%s", view)
	}
}

func TestStatusClears(t *testing.T) {
	m, _ := withOpener(t, nil, portal2)
	var tm tea.Model = act(m, press("s"))
	first := tm.(Model).statusID
	tm = act(tm, press("i"))

	tm, _ = tm.Update(clearStatusMsg{id: first}) // the older timer fires
	if tm.(Model).status == "" {
		t.Error("an older timer must not clear a newer message")
	}
	tm, _ = tm.Update(clearStatusMsg{id: tm.(Model).statusID})
	if tm.(Model).status != "" {
		t.Error("the current timer should clear the message")
	}
}

func TestHelpToggle(t *testing.T) {
	m, _ := withOpener(t, nil, portal2)
	short := ansi.Strip(m.View().Content)
	if !strings.Contains(short, "enter play") || !strings.Contains(short, "i install") || strings.Contains(short, "store page") {
		t.Errorf("short help should show enter and i, not s:\n%s", short)
	}
	var tm tea.Model = act(m, press("?"))
	full := ansi.Strip(tm.(Model).View().Content)
	for _, want := range []string{"i", "install", "store page", "rescan"} {
		if !strings.Contains(full, want) {
			t.Errorf("full help missing %q", want)
		}
	}
}

// The key line always fits: at 80 columns ↑↓ move gives way so i install fits.
func TestShortHelpFits(t *testing.T) {
	for _, w := range []int{80, 120} {
		m, _ := withOpener(t, nil, portal2)
		var tm tea.Model = m
		tm, _ = tm.Update(tea.WindowSizeMsg{Width: w, Height: 30})
		line := lineWith(ansi.Strip(tm.(Model).View().Content), "i install")
		if line == "" || !strings.Contains(line, "q quit") {
			t.Fatalf("width %d: the key line should hold i install through q quit: %q", w, line)
		}
		if got := ansi.StringWidth(line); got > w {
			t.Errorf("width %d: key line is %d wide", w, got)
		}
		if wantMove := w >= 120; strings.Contains(line, "move") != wantMove {
			t.Errorf("width %d: move shown should be %v: %q", w, wantMove, line)
		}
	}
}
