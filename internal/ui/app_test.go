package ui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/library"
)

func press(s string) tea.KeyPressMsg {
	return tea.KeyPressMsg{Code: []rune(s)[0], Text: s}
}

// ready returns a model sized 80x24 with the given games loaded.
func ready(t *testing.T, games ...string) Model {
	t.Helper()
	res := library.Result{PlaytimeKnown: true}
	for i, name := range games {
		res.Games = append(res.Games, library.Game{AppID: uint32(i + 1), Name: name, Installed: true})
	}
	var m tea.Model = New("")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(scanDoneMsg{res: res})
	return m.(Model)
}

func TestScanFillsList(t *testing.T) {
	m := ready(t, "Alpha", "Bravo", "Charlie")
	if m.state != stateReady {
		t.Fatalf("state: got %v, want ready", m.state)
	}
	if n := len(m.list.Items()); n != 3 {
		t.Errorf("items: got %d, want 3", n)
	}
	view := m.View().Content
	for _, name := range []string{"Alpha", "Bravo", "Charlie", "QUARTER"} {
		if !strings.Contains(view, name) {
			t.Errorf("view missing %q", name)
		}
	}
	if !m.View().AltScreen {
		t.Error("UI should use the full screen")
	}
}

func TestMoveSelection(t *testing.T) {
	var m tea.Model = ready(t, "Alpha", "Bravo", "Charlie")
	m, _ = m.Update(press("j"))
	m, _ = m.Update(press("j"))
	if got := m.(Model).list.Index(); got != 2 {
		t.Errorf("after j j: index %d, want 2", got)
	}
	m, _ = m.Update(press("k"))
	if got := m.(Model).list.Index(); got != 1 {
		t.Errorf("after k: index %d, want 1", got)
	}
}

func TestQuit(t *testing.T) {
	for _, name := range []string{"ready", "failed"} {
		t.Run(name, func(t *testing.T) {
			var m tea.Model = ready(t, "Alpha")
			if name == "failed" {
				m, _ = New("").Update(scanDoneMsg{err: errors.New("not found")})
			}
			_, cmd := m.Update(press("q"))
			if cmd == nil {
				t.Fatal("q returned no command")
			}
			if _, ok := cmd().(tea.QuitMsg); !ok {
				t.Error("q should quit")
			}
		})
	}
}

func TestSteamNotFound(t *testing.T) {
	m, _ := New("").Update(scanDoneMsg{err: errors.New("steam install not found")})
	view := m.(Model).View().Content
	if !strings.Contains(view, "Steam not found") || !strings.Contains(view, "--steam-path") {
		t.Errorf("error screen missing guidance:\n%s", view)
	}
}

func TestWarningsFooter(t *testing.T) {
	var m tea.Model = New("")
	m, _ = m.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	m, _ = m.Update(scanDoneMsg{res: library.Result{
		Games:    []library.Game{{AppID: 1, Name: "Alpha"}},
		Warnings: []error{errors.New("bad file")},
	}})
	if view := m.(Model).View().Content; !strings.Contains(view, "1 file(s) could not be read") {
		t.Errorf("footer missing:\n%s", view)
	}
}

func TestLongNameTruncated(t *testing.T) {
	m := ready(t, strings.Repeat("Very Long Game Name ", 10))
	for _, line := range strings.Split(m.View().Content, "\n") {
		if w := ansi.StringWidth(line); w > 80 {
			t.Errorf("line wider than the window (%d): %q", w, line)
		}
	}
}
