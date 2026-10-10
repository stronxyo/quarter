package ui

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/intro"
	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/steam"
	"github.com/stronxyo/quarter/internal/webapi"
)

// consoleSafe are the characters beyond ASCII that Quarter may draw in the
// classic Windows console: each checked in the font files of Consolas and of
// the Cascadia Mono that ships with Windows (fontTools, 2026-10-02), the
// classic console's two default fonts, which borrow nothing from others.
const consoleSafe = "·…←↑→↓─━│┌┐└┘├┤┬┴┼═╭╮╯╰▀▄█▌▐░▒▓■▪▲▸►▼●◊♦›♥"

// basicModel is a model as the classic console runs it (BasicGlyphs), the
// whole UI's characters put back to the full set when the test ends.
func basicModel(t *testing.T, opts Options) Model {
	t.Helper()
	t.Cleanup(func() { applyGlyphs(false) })
	opts.BasicGlyphs = true
	return New(opts)
}

// sweepLibrary is a library with every kind of row: installed and not,
// favourites, a game running, one downloading, one paused; store details
// with websites; play history and past years.
func sweepLibrary() (library.Result, Options) {
	lib := bigLibrary(300)
	for i := range lib.Games {
		lib.Games[i].Installed = i%3 != 0
		if i%5 == 0 {
			lib.Games[i].Achievements = steam.Achievements{Achieved: i % 40, Total: 40}
		}
	}
	lib.Games[1].Updating = true
	lib.Games[2].Updating = true
	info := map[uint32]webapi.StoreEntry{}
	for _, g := range lib.Games {
		info[g.AppID] = webapi.StoreEntry{Fetched: time.Now(), Info: webapi.StoreInfo{
			Name: g.Name, Type: "game", Developers: []string{"Studio"}, Publishers: []string{"Publisher"},
			Genres: []string{"Action", "Racing"}, ReleaseDate: "6 Jul, 2015", Website: "https://www.example.com/game/",
			Description: "A game about driving very fast through a long and winding course.", Achievements: 40,
		}}
	}
	return lib, Options{
		Cached:     &lib,
		StoreCache: info,
		Sessions:   bigSessions(lib),
		Replay:     bigYears(lib),
		Settings:   config.Settings{Favourites: []uint32{1, 2, 4, 9}},
		Open:       func(string) error { return nil },
	}
}

// drawn adds every character of m's screen to seen, with where it was seen.
func drawn(seen map[rune]string, where string, m tea.Model) {
	for _, r := range ansi.Strip(m.(Model).View().Content) {
		if r > 0x7e {
			if _, ok := seen[r]; !ok {
				seen[r] = where
			}
		}
	}
}

// sweep drives Quarter through every screen at w x h in the classic
// console's mode and collects the characters drawn.
func sweep(t *testing.T, seen map[rune]string, w, h int) {
	t.Helper()
	lib, opts := sweepLibrary()
	at := func(s string) string { return fmt.Sprintf("%dx%d %s", w, h, s) }
	var m tea.Model = basicModel(t, opts)
	m, _ = m.Update(tea.WindowSizeMsg{Width: w, Height: h})
	m, _ = m.Update(scanDoneMsg{res: lib})
	mm := m.(Model)
	mm.running = lib.Games[3].AppID
	mm.downloads = map[uint32]steam.Download{
		lib.Games[1].AppID: {AppID: lib.Games[1].AppID, Phase: steam.PhaseDownloading, Done: 40 << 30, Total: 100 << 30, Rate: 10 << 20, At: time.Now()},
		lib.Games[2].AppID: {AppID: lib.Games[2].AppID, Phase: steam.PhaseNone, Done: 10 << 30, Total: 100 << 30},
	}
	mm.list.SetDelegate(mm.delegate())
	m = mm
	drawn(seen, at("list"), m)
	for i := range 6 { // through the running, downloading and favourite rows, the pane following
		m, _ = m.Update(press("down"))
		drawn(seen, at(fmt.Sprintf("list, row %d", i+1)), m)
	}
	m, _ = m.Update(dlBlinkMsg{})
	drawn(seen, at("download blink"), m)
	for _, k := range []string{"d", "?", "?", "d", "o", "tab", "tab", "n", "p", "l"} {
		m, _ = m.Update(press(k))
		drawn(seen, at("after "+k), m)
	}
	// The pointer over every target: the hover's mark.
	for _, tg := range m.(Model).targets() {
		mt, _ := m.Update(tea.MouseMotionMsg{X: (tg.x0 + tg.x1) / 2, Y: tg.y})
		drawn(seen, at(fmt.Sprintf("hover %d", tg.kind)), mt)
	}
	h1 := hit(m, press("h"))
	drawn(seen, at("history ranking"), h1)
	for _, k := range []string{"tab", "tab", "tab", "h", "tab", "a", "esc", "h"} {
		h1, _ = h1.Update(press(k))
		drawn(seen, at("history "+k), h1)
	}
	s := typed(m.(Model), "/ra")
	drawn(seen, at("search"), s)

	// What goes wrong: Steam not found, no games, a library offline, a
	// damaged file, the store offline.
	for name, msg := range map[string]scanDoneMsg{
		"steam not found": {err: errors.New("steam install not found")},
		"no games":        {res: library.Result{PlaytimeKnown: true}},
		"damaged":         {res: library.Result{PlaytimeKnown: true, Games: lib.Games[:5], Warnings: []error{&steam.FileError{Name: "localconfig.vdf", Err: errors.New("unexpected end of file")}}}},
		"not connected":   {res: library.Result{PlaytimeKnown: true, Games: lib.Games[:5], Missing: []steam.LibraryFolder{{Path: `E:\SteamLibrary`, Games: 3}}}},
	} {
		var e tea.Model = basicModel(t, Options{})
		e, _ = e.Update(tea.WindowSizeMsg{Width: w, Height: h})
		e, _ = e.Update(msg)
		drawn(seen, at(name), e)
	}
	off := basicModel(t, Options{FetchStore: func(context.Context, uint32) (webapi.StoreInfo, error) { return webapi.StoreInfo{}, nil }})
	off.storePause = pause{until: time.Now().Add(time.Hour), offline: true}
	var o tea.Model = off
	o, _ = o.Update(tea.WindowSizeMsg{Width: w, Height: h})
	o, _ = o.Update(scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: lib.Games[:5]}})
	drawn(seen, at("store offline"), o)
}

// sweepStart drives the start screen at w x h: every intro, frame by frame,
// the shine, the blink, the compact screen; and every goodbye.
func sweepStart(t *testing.T, seen map[rune]string, w, h int) {
	t.Helper()
	at := func(s string) string { return fmt.Sprintf("%dx%d %s", w, h, s) }
	for i := range intro.Count() {
		var m tea.Model = basicModel(t, Options{Splash: true, Settings: config.Settings{Intro: intro.Key(i)}})
		mm := m.(Model)
		clock := time.Now()
		mm.now = func() time.Time { return clock }
		m, _ = tea.Model(mm).Update(tea.WindowSizeMsg{Width: w, Height: h})
		m, _ = m.Update(scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: bigLibrary(3).Games}})
		for f := time.Duration(0); f <= intro.Length(i)+introFrame; f += introFrame {
			clock = clock.Add(introFrame)
			m, _ = m.Update(introTickMsg{})
			drawn(seen, at("intro "+intro.Key(i)), m)
		}
		m, _ = m.Update(blinkMsg{})
		drawn(seen, at("start screen"), m)
		sc, _ := m.(Model).startScreen()
		for f := time.Duration(0); f < intro.ShineLength; f += introFrame {
			if frame, done := sc.Shine(true, f); !done {
				for _, r := range ansi.Strip(frame) {
					if r > 0x7e {
						if _, ok := seen[r]; !ok {
							seen[r] = at("shine")
						}
					}
				}
			}
		}
	}
	for i := range intro.GoodbyeCount() {
		for _, r := range ansi.Strip(intro.Goodbye(i, w, 90*time.Minute)) {
			if r > 0x7e {
				if _, ok := seen[r]; !ok {
					seen[r] = at("goodbye " + intro.GoodbyeKey(i))
				}
			}
		}
	}
}

// In the classic console's mode, every screen Quarter can show, at four
// sizes, every intro frame, the shine and every goodbye, draws only
// characters its fonts have: nothing turns into a box.
func TestBasicGlyphsEverywhere(t *testing.T) {
	seen := map[rune]string{}
	for _, size := range [][2]int{{60, 20}, {80, 24}, {120, 40}, {200, 60}} {
		sweep(t, seen, size[0], size[1])
	}
	for _, size := range [][2]int{{60, 20}, {160, 44}} {
		sweepStart(t, seen, size[0], size[1])
	}
	var found []string
	for r, where := range seen {
		found = append(found, string(r))
		if !strings.ContainsRune(consoleSafe, r) {
			t.Errorf("U+%04X %q isn't in the classic console's fonts, drawn on %s", r, string(r), where)
		}
	}
	slices.Sort(found)
	t.Logf("characters beyond ASCII drawn: %s", strings.Join(found, ""))
}

// In the classic console no game logo shows: none drawn as text, none as a
// picture, none even read or downloaded; o says why. The same library in
// any other terminal shows the logo.
func TestClassicConsoleNoLogos(t *testing.T) {
	for _, basic := range []bool{true, false} {
		logos := &fakeLogos{paths: map[uint32][]string{1: {logoPNG(t)}}, cdn: map[uint32]string{2: logoPNG(t)}}
		t.Cleanup(func() { applyGlyphs(false) })
		m := New(Options{BasicGlyphs: basic, Logos: func(string) LogoSource { return logos }})
		m.debounce, m.settle = 0, 0
		m.sixel, m.cellW, m.cellH = true, 10, 20 // a terminal that draws pictures, as far as Quarter knows
		var tm tea.Model = m
		tm, _ = tm.Update(tea.WindowSizeMsg{Width: 150, Height: 40})
		tm, cmd := tm.Update(scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: []library.Game{
			{AppID: 1, Name: "Alpha", Installed: true},
			{AppID: 2, Name: "Bravo", Installed: true},
		}}})
		tm = pump(tm, cmd)
		tm = pump(tm.Update(press("down")))
		view := paneText(tm)
		got := tm.(Model)
		if !basic {
			if got.wantPicture() == nil && !hasGameLogo(view) && logos.reads == 0 {
				t.Error("in another terminal the logo should show")
			}
			continue
		}
		if hasGameLogo(view) || got.showsLogos() || got.wantPicture() != nil {
			t.Errorf("the classic console should show no game logo:\n%s", view)
		}
		if logos.reads != 0 || logos.downloads != 0 {
			t.Errorf("no logo should even be read or downloaded: %d reads, %d downloads", logos.reads, logos.downloads)
		}
		tm, _ = tm.Update(press("o"))
		if s := tm.(Model).status; !strings.Contains(s, "Windows Terminal") {
			t.Errorf("o should say where logos show, got %q", s)
		}
		if strings.Contains(paneText(tm), "██") && hasGameLogo(paneText(tm)) {
			t.Error("o must not bring logos back here")
		}
	}
}

// basicGlyphs' stand-ins and the full set it replaces stay in step: the
// same number of steps for the bars, and every stand-in different from the
// character it stands for.
func TestGlyphSetsMatch(t *testing.T) {
	f, b := fullGlyphs, basicGlyphs
	if len([]rune(f.sparkSteps)) != len([]rune(b.sparkSteps)) || len([]rune(f.barParts)) != len([]rune(b.barParts)) {
		t.Error("the bars need as many steps in both sets")
	}
	for _, pair := range [][2]string{{f.link, b.link}, {f.uninstalled, b.uninstalled}, {f.favourite, b.favourite}, {f.favouriteUninstalled, b.favouriteUninstalled}, {f.running, b.running}} {
		if pair[0] == pair[1] || len([]rune(pair[1])) != 1 {
			t.Errorf("%q should have a one-character stand-in, got %q", pair[0], pair[1])
		}
	}
	// One cell each, so rows line up in both sets.
	for _, g := range []glyphs{f, b} {
		for _, s := range []string{g.link, g.uninstalled, g.favourite, g.favouriteUninstalled, g.running} {
			if ansi.StringWidth(s) != 1 {
				t.Errorf("%q should be one cell wide", s)
			}
		}
	}
}
