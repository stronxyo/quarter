package ui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/library"
)

// splashed is a model on its first run, w x h, with n games scanned, and
// the settings it saves.
func splashed(t *testing.T, w, h, n int) (Model, *[]config.Settings) {
	t.Helper()
	saved := &[]config.Settings{}
	m := New(Options{Splash: true, SaveSettings: func(s config.Settings) error { *saved = append(*saved, s); return nil }})
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: w, Height: h})
	res := library.Result{PlaytimeKnown: true}
	for i := range n {
		res.Games = append(res.Games, library.Game{AppID: uint32(i + 1), Name: "Game", Installed: true})
	}
	tm, _ = tm.Update(scanDoneMsg{res: res})
	return tm.(Model), saved
}

func screen(m Model) string { return ansi.Strip(m.View().Content) }

// The first run opens on the boot check, the logo and a blinking INSERT
// QUARTER, all within the window.
func TestSplashFirstRun(t *testing.T) {
	m, _ := splashed(t, 120, 40, 300)
	view := screen(m)
	for _, want := range []string{"QUARTER SYSTEM CHECK", "Detecting Steam", "OK", "300 games", "press any key", "██"} {
		if !strings.Contains(view, want) {
			t.Errorf("the splash should show %q:\n%s", want, view)
		}
	}
	if strings.Contains(view, "INSERT QUARTER") {
		t.Error("INSERT QUARTER starts dark, then blinks on")
	}
	next, cmd := m.Update(blinkMsg{})
	m = next.(Model)
	if !strings.Contains(screen(m), "INSERT QUARTER") || cmd == nil {
		t.Error("a blink should light INSERT QUARTER and schedule the next")
	}
	for _, l := range strings.Split(screen(m), "\n") {
		if ansi.StringWidth(l) > 120 {
			t.Errorf("a line is wider than the window: %q", l)
		}
	}
}

// Before the scan answers, the checks show dots; if Steam isn't found, they
// say so.
func TestSplashFollowsScan(t *testing.T) {
	m := New(Options{Splash: true})
	m.width, m.height = 120, 40
	if v := screen(m); !strings.Contains(v, "Detecting Steam") || strings.Contains(v, "OK") {
		t.Errorf("while scanning, no result yet:\n%s", v)
	}
	next, _ := m.Update(scanDoneMsg{err: errors.New("steam install not found")})
	if v := screen(next.(Model)); !strings.Contains(v, "NOT FOUND") {
		t.Errorf("a failed scan should show NOT FOUND:\n%s", v)
	}
}

// Any key starts, and the splash is saved as seen; the blinking stops.
func TestSplashAnyKeyStarts(t *testing.T) {
	m, saved := splashed(t, 120, 40, 3)
	next, cmd := m.Update(press("x"))
	m = next.(Model)
	if m.splash || strings.Contains(screen(m), "QUARTER SYSTEM CHECK") {
		t.Fatal("a key should leave the splash for the list")
	}
	messages(cmd) // runs the save
	if len(*saved) != 1 || !(*saved)[0].SplashSeen {
		t.Errorf("leaving the splash should save it as seen, got %+v", *saved)
	}
	if m.blink() != nil {
		t.Error("the blinking should stop once the splash is gone")
	}
}

// q and ctrl+c quit from the splash, as everywhere.
func TestSplashQuit(t *testing.T) {
	for _, k := range []tea.KeyPressMsg{press("q"), {Code: 'c', Mod: tea.ModCtrl}} {
		m, _ := splashed(t, 120, 40, 3)
		_, cmd := m.Update(k)
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Errorf("%s should quit from the splash", k.String())
		}
	}
}

// Once seen, Quarter opens on the list; and no game logo is drawn over the
// splash while it is up.
func TestSplashOnlyOnce(t *testing.T) {
	m := ready(t, "Alpha")
	if m.splash || strings.Contains(screen(m), "QUARTER SYSTEM CHECK") {
		t.Error("without Splash, Quarter opens on the list")
	}
	s, _ := splashed(t, 150, 40, 3)
	s.logos = &fakeLogos{paths: map[uint32][]string{}}
	if s.showsLogos() {
		t.Error("no game logo while the splash is up")
	}
}

// On a small window the badge replaces the big logo, and nothing overflows.
func TestSplashSmallWindow(t *testing.T) {
	m, _ := splashed(t, 60, 20, 3)
	view := screen(m)
	if !strings.Contains(view, "quarter_") || strings.Contains(view, "██") {
		t.Errorf("a small window should show the badge, not the big logo:\n%s", view)
	}
	for _, l := range strings.Split(view, "\n") {
		if ansi.StringWidth(l) > 60 {
			t.Errorf("a line is wider than the window: %q", l)
		}
	}
}
