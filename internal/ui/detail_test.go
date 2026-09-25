package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/library"
)

func sized(t *testing.T, width, height int) tea.Model {
	t.Helper()
	var m tea.Model = New(Options{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	m, _ = m.Update(scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: []library.Game{
		{AppID: 1, Name: "Alpha", Installed: true, Playtime: 3 * time.Hour, SizeOnDisk: 13_314_398_618, LibraryPath: `D:\SteamLibrary`},
		{AppID: 2, Name: "Bravo Demo", Installed: true},
	}}})
	return m
}

// fits fails the test if any line is wider, or the view taller, than the window.
func fits(t *testing.T, m tea.Model, width, height int) string {
	t.Helper()
	view := ansi.Strip(m.(Model).View().Content)
	lines := strings.Split(view, "\n")
	if len(lines) > height {
		t.Errorf("view is %d lines, window is %d", len(lines), height)
	}
	for _, l := range lines {
		if w := ansi.StringWidth(l); w > width {
			t.Errorf("line is %d wide, window is %d: %q", w, width, l)
		}
	}
	return view
}

func TestPaneBesideListOnWideWindow(t *testing.T) {
	m := sized(t, 140, 30)
	view := fits(t, m, 140, 30)
	for _, want := range []string{"ALPHA", "Played      3.0h", "Size        12.4 GB", `Library     D:\SteamLibrary`} {
		if !strings.Contains(view, want) {
			t.Errorf("pane missing %q", want)
		}
	}

	m, _ = m.Update(press("j"))
	view = fits(t, m, 140, 30)
	if !strings.Contains(view, "BRAVO (DEMO)") {
		t.Error("pane should follow the selection to Bravo, with its demo tag")
	}
}

func TestPanePopUpOnNarrowWindow(t *testing.T) {
	m := sized(t, 80, 30)
	if view := fits(t, m, 80, 30); strings.Contains(view, "ALPHA") {
		t.Fatal("narrow windows should not show the pane until d is pressed")
	}

	m, _ = m.Update(press("d"))
	view := fits(t, m, 80, 30)
	if !strings.Contains(view, "ALPHA") || strings.Contains(view, "Bravo") {
		t.Errorf("d should open the pane over the list:\n%s", view)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if view := fits(t, m, 80, 30); strings.Contains(view, "ALPHA") || !strings.Contains(view, "Bravo") {
		t.Error("esc should close the pane and show the list again")
	}
}

func TestDetailsKeyIgnoredOnWideWindow(t *testing.T) {
	m := sized(t, 140, 30)
	m, _ = m.Update(press("d"))
	if m.(Model).detailOpen {
		t.Error("d should do nothing when the pane is already beside the list")
	}
}
