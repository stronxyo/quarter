package ui

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/library"
)

// laidOut returns a model w x h with n games (the first has a logo on disk)
// after the given keys.
func laidOut(t *testing.T, w, h, n int, keys ...string) Model {
	t.Helper()
	logos := &fakeLogos{paths: map[uint32][]string{1: {logoPNG(t)}}, missing: map[uint32]bool{}}
	games := make([]library.Game, n)
	for i := range games {
		games[i] = library.Game{AppID: uint32(i + 1), Name: fmt.Sprintf("Game %02d", i+1), Installed: true}
		logos.missing[uint32(i+2)] = true
	}
	m := New(Options{Settings: config.Settings{}, Logos: func(string) LogoSource { return logos }, SaveSettings: func(config.Settings) error { return nil }})
	m.debounce, m.settle = 0, 0
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: w, Height: h})
	tm, cmd := tm.Update(scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: games}})
	tm = pump(tm, cmd)
	for _, k := range keys {
		msg := press(k)
		if k == "tab" {
			msg = tea.KeyPressMsg{Code: tea.KeyTab}
		}
		tm, cmd = tm.Update(msg)
		tm = pump(tm, cmd)
	}
	return tm.(Model)
}

// The screen's layout at every size and mode (Batch 8, A2): the quarter_ box
// and the list on the left, the logo and the pane pinned to the right edge,
// lined up, the section line reaching the pane, nothing past the window.
func TestLayoutAnchoredLeftAndRight(t *testing.T) {
	widths := []int{60, 80, 97, 98, 106, 121, 122, 150, 162, 260}
	heights := []int{20, 45}
	modes := []struct {
		name  string
		games int
		keys  []string
	}{
		{"plain", 60, nil},
		{"pane hidden", 60, []string{"d"}},
		{"logos hidden", 60, []string{"o"}},
		{"searching", 60, []string{"/", "G", "a"}},
		{"full help", 60, []string{"?"}},
		{"few games", 3, nil},
	}
	for _, md := range modes {
		for _, w := range widths {
			for _, h := range heights {
				name := fmt.Sprintf("%s %dx%d", md.name, w, h)
				m := laidOut(t, w, h, md.games, md.keys...)
				lines := strings.Split(ansi.Strip(m.View().Content), "\n")
				if len(lines) > h {
					t.Errorf("%s: %d lines, window has %d", name, len(lines), h)
				}
				for _, l := range lines {
					if lw := ansi.StringWidth(l); lw > w {
						t.Errorf("%s: a line is %d wide: %q", name, lw, l)
					}
				}
				if m.tableWidth() > maxTableWidth {
					t.Errorf("%s: list %d wide", name, m.tableWidth())
				}
				if m.showLogo() && !strings.HasPrefix(lines[logoPadTop], strings.Repeat(" ", logoIndent)+"╭") {
					t.Errorf("%s: the quarter_ box should start at column %d: %q", name, logoIndent, lines[logoPadTop])
				}
				if !m.sideBySide() {
					continue
				}
				x := m.paneX()
				if x+m.paneWidth() != w || x-m.tableWidth() != paneGap {
					t.Errorf("%s: pane at %d, %d wide, list %d: should run from the list to the right edge", name, x, m.paneWidth(), m.tableWidth())
				}
				// Its contents: a column at most paneTextWidth wide, on the left.
				col := min(m.paneWidth()-paneStyle.GetHorizontalFrameSize(), paneTextWidth)
				left := x + 2 // border, then padding
				titled := false
				for _, l := range lines {
					r := []rune(l)
					if len(r) != w || !strings.ContainsRune("│", r[x]) {
						continue
					}
					inside := string(r[x+2 : w-2])
					if text := strings.TrimSpace(inside); text != "" {
						start := len([]rune(inside)) - len([]rune(strings.TrimLeft(inside, " ")))
						if x+2+start < left || ansi.StringWidth(strings.TrimRight(inside, " ")) > left-x-2+col {
							t.Errorf("%s: pane text outside its column [%d, %d): %q", name, left, left+col, inside)
						}
						if strings.HasPrefix(text, "GAME ") && x+2+start == left {
							titled = true
						}
					}
				}
				if !titled {
					t.Errorf("%s: the game's title should start the column at %d", name, left)
				}
				// The pane's frame: its left edge at x, its right edge the
				// window's last column.
				framed := 0
				for _, l := range lines {
					r := []rune(l)
					if len(r) == w && strings.ContainsRune("╭│╰", r[x]) && strings.ContainsRune("╮│╯", r[w-1]) {
						framed++
					}
				}
				if framed < 3 {
					t.Errorf("%s: the pane's frame should run from column %d to the right edge", name, x)
				}
				// The section line runs to two columns before the pane.
				for _, l := range lines {
					if r := []rune(l); strings.Contains(l, "▸─") && len(r) >= x {
						if got := ansi.StringWidth(strings.TrimRight(string(r[:x]), " ")); got != x-paneGap {
							t.Errorf("%s: the section line ends at %d, want %d: %q", name, got, x-paneGap, l)
						}
					}
				}
				// The logo band, when shown, starts where the pane starts.
				if bx, cols, _, ok := m.logoBand(); ok && m.showsLogos() {
					if bx != x || cols != m.paneWidth() {
						t.Errorf("%s: logo band at %d, %d wide; pane at %d, %d wide", name, bx, cols, x, m.paneWidth())
					}
					lo, hi := w, -1 // the logo's leftmost and rightmost columns
					for _, l := range lines[:logoHeight()] {
						r := []rune(l)
						for c := logoIndent + logoBoxWidth; c < len(r); c++ {
							if r[c] >= 0x2580 && r[c] <= 0x259F {
								lo, hi = min(lo, c), max(hi, c)
							}
						}
					}
					if hi < 0 || lo < x {
						t.Errorf("%s: the logo should be drawn in the band at %d, found %d to %d", name, x, lo, hi)
					} else if d := (lo - x) - (x + cols - 1 - hi); d < -1 || d > 1 {
						t.Errorf("%s: the logo should be centred over the pane: %d free left, %d right", name, lo-x, x+cols-1-hi)
					}
				}
			}
		}
	}
}
