package intro

import (
	"math/rand/v2"
	"regexp"
	"strings"
	"testing"
	"time"
)

var sgr = regexp.MustCompile("\x1b\\[[0-9;]*m")

func plain(s string) string { return sgr.ReplaceAllString(s, "") }

const title = "QUARTER SYSTEM CHECK"

var checks = []Line{
	{{"Looking for Steam", Plain}, {" ........... ", Faint}, {"OK", Bright}},
}

// common is what every monospace font draws (as internal/ui's check).
func common(r rune) bool {
	switch {
	case r < 0x250, r >= 0x2500 && r <= 0x25FF:
		return true
	}
	return r == '·'
}

// fits checks a frame: no more lines or columns than the window, and only
// characters every font has.
func fits(t *testing.T, name, frame string, w, h int) {
	t.Helper()
	lines := strings.Split(plain(frame), "\n")
	if len(lines) > h {
		t.Errorf("%s: %d lines, window has %d", name, len(lines), h)
	}
	for _, l := range lines {
		if n := len([]rune(l)); n > w {
			t.Errorf("%s: a line is %d wide, window is %d", name, n, w)
		}
		for _, r := range l {
			if !common(r) {
				t.Errorf("%s: %q (%U) is missing from common fonts", name, r, r)
				return
			}
		}
	}
}

// The rest screen: the logo big on big windows, normal on smaller ones,
// none when even that doesn't fit; insert quarter only when lit.
func TestRest(t *testing.T) {
	for _, tc := range []struct {
		w, h  int
		scale int
	}{{200, 60, 2}, {160, 44, 2}, {150, 30, 1}, {90, 30, 1}, {70, 30, 0}, {120, 15, 0}} {
		sc, ok := NewScreen(tc.w, tc.h, title, checks)
		if ok != (tc.scale > 0) || ok && sc.s.scale != tc.scale {
			t.Errorf("%dx%d: ok %v scale %d, want scale %d", tc.w, tc.h, ok, sc.s.scale, tc.scale)
			continue
		}
		if !ok {
			continue
		}
		lit, dark := plain(sc.Rest(true)), plain(sc.Rest(false))
		fits(t, "rest", lit, tc.w, tc.h)
		for _, want := range []string{"QUARTER SYSTEM CHECK", "OK", "press any key", "██"} {
			if !strings.Contains(lit, want) {
				t.Errorf("%dx%d: the rest screen should show %q", tc.w, tc.h, want)
			}
		}
		if lit == dark {
			t.Errorf("%dx%d: insert quarter should blink", tc.w, tc.h)
		}
		if tc.scale == 1 && !strings.Contains(lit, "insert quarter") {
			t.Errorf("%dx%d: the normal logo has insert quarter as a line", tc.w, tc.h)
		}
	}
}

// Every intro, at every size, runs to its end without drawing outside the
// window or a character fonts lack, and lands close to the rest screen.
func TestEveryIntro(t *testing.T) {
	for _, size := range [][2]int{{90, 30}, {160, 44}, {240, 70}} {
		w, h := size[0], size[1]
		sc, ok := NewScreen(w, h, title, checks)
		if !ok {
			t.Fatalf("%dx%d should fit", w, h)
		}
		rest := strings.Split(plain(sc.Rest(false)), "\n")
		for i := range Count() {
			run := sc.Start(i, 1)
			name := Name(i)
			var last string
			for f := 0.0; ; f += 1.0 / fps {
				frame, done := sc.Frame(run, time.Duration(f*float64(time.Second)))
				if done {
					break
				}
				fits(t, name, frame, w, h)
				last = frame
			}
			if last == "" {
				t.Errorf("%s: no frames", name)
				continue
			}
			// The last frame before the rest screen: most of the logo in place,
			// so the switch to rest isn't a jump.
			got := strings.Split(plain(last), "\n")
			same, total := 0, 0
			for _, p := range sc.s.logo {
				if p.kind == kindBorder {
					continue
				}
				total++
				if p.y < len(got) && p.y < len(rest) {
					g, r := []rune(got[p.y]), []rune(rest[p.y])
					if p.x < len(g) && p.x < len(r) && g[p.x] == r[p.x] {
						same++
					}
				}
			}
			if share := float64(same) / float64(total); share < 0.8 {
				t.Errorf("%s at %dx%d: the last frame has %.0f%% of the logo in place, want 80%%", name, w, h, 100*share)
			}
		}
	}
}

// A run made for another size ends at once rather than drawing wrongly.
func TestRunOutlivesResize(t *testing.T) {
	big, _ := NewScreen(160, 44, title, checks)
	small, _ := NewScreen(90, 30, title, checks)
	run := big.Start(0, 1)
	if _, done := small.Frame(run, 0); !done {
		t.Error("a run from another logo size should end")
	}
}

// Pick never repeats the last intro, and every other one comes up.
func TestPickNeverRepeats(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	seen := map[int]bool{}
	last := -1
	for range 2000 {
		i := Pick(last, r)
		if i == last || i < 0 || i >= Count() {
			t.Fatalf("Pick(%d) = %d", last, i)
		}
		seen[i] = true
		last = i
	}
	if len(seen) != Count() {
		t.Errorf("every intro should come up, %d of %d did", len(seen), Count())
	}
}

// Keys are unique and found again.
func TestKeys(t *testing.T) {
	seen := map[string]bool{}
	for i := range Count() {
		k := Key(i)
		if seen[k] || strings.ContainsAny(k, " ()") {
			t.Errorf("key %q is repeated or has spaces or brackets", k)
		}
		seen[k] = true
		if j, ok := Find(k); !ok || j != i {
			t.Errorf("Find(%q) = %d, want %d", k, j, i)
		}
	}
	if _, ok := Find("crt-power-on"); !ok {
		t.Error(`"crt-power-on" should be an intro`)
	}
}

// The letters sit in the middle of the box: as many rows between the top
// edge and the t's top as between the q's bottom and the bottom edge.
func TestLogoCentredInBox(t *testing.T) {
	for _, size := range [][2]int{{160, 44}, {110, 34}} {
		sc, ok := NewScreen(size[0], size[1], title, checks)
		if !ok {
			t.Fatalf("%v should fit", size)
		}
		s := sc.s
		first, last := s.h, -1
		for _, p := range s.logo {
			if p.c.r == '█' {
				first, last = min(first, p.y), max(last, p.y)
			}
		}
		above, below := first-s.ly-1, s.ly+s.bh-2-last
		if above != below || above < 1 {
			t.Errorf("scale %d: %d rows above the letters, %d below; want equal", s.scale, above, below)
		}
	}
}
