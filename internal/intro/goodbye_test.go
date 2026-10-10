package intro

import (
	"math/rand/v2"
	"strings"
	"testing"
	"time"
)

// Every character the goodbyes use is in the logo's font, so none is drawn
// as a gap.
func TestGoodbyeLetters(t *testing.T) {
	for _, g := range goodbyes {
		for _, r := range g {
			if _, ok := logoFont[r]; !ok && r != ' ' {
				t.Errorf("%q: %q is missing from the logo font", g, r)
			}
		}
	}
}

// Every goodbye, at every width: within the terminal, only common
// characters, one column in from the left; too narrow for big letters,
// plain text.
func TestGoodbyeFits(t *testing.T) {
	for i := range GoodbyeCount() {
		for _, w := range []int{200, 160, 100, 60} {
			out := Goodbye(i, w, time.Hour)
			fits(t, goodbyes[i], out, w, 20)
			lines := strings.Split(strings.Trim(plain(out), "\n"), "\n")
			rows := len(logoText(goodbyes[i]))
			big := len(lines) > 3 // the big rows, a blank, the played line
			if !big {
				rows = 1
				if lines[0] != " "+goodbyes[i] {
					t.Errorf("%q at %d: a line too wide is plain text, got %q", goodbyes[i], w, lines[0])
				}
			}
			if big && len([]rune(logoText(goodbyes[i])[0])) > w-2 {
				t.Errorf("%q at %d: big letters wider than the terminal", goodbyes[i], w)
			}
			// The block's left edge: the least indented of its rows (some
			// letters start a pixel in on their top row).
			left := w
			for _, l := range lines[:rows] {
				if strings.TrimSpace(l) != "" {
					left = min(left, len(l)-len(strings.TrimLeft(l, " ")))
				}
			}
			if left != 1 || !strings.HasPrefix(lines[len(lines)-1], " played") {
				t.Errorf("%q at %d: not one column in from the left, %d in", goodbyes[i], w, left)
			}
		}
	}
}

// The played time: under the big line when there was any, in hours and
// minutes; nothing when there wasn't.
func TestGoodbyePlayed(t *testing.T) {
	for d, want := range map[time.Duration]string{
		0:                            "",
		20 * time.Second:             "played this session: 1m",
		45 * time.Minute:             "played this session: 45m",
		2 * time.Hour:                "played this session: 2h",
		2*time.Hour + 10*time.Minute: "played this session: 2h 10m",
	} {
		out := plain(Goodbye(0, 160, d))
		if want == "" && strings.Contains(out, "played") || want != "" && !strings.Contains(out, want) {
			t.Errorf("%v played: want %q in %q", d, want, out)
		}
	}
}

// Keys have no spaces or punctuation, and find their goodbye again.
func TestGoodbyeKeys(t *testing.T) {
	seen := map[string]bool{}
	for i := range GoodbyeCount() {
		k := GoodbyeKey(i)
		if seen[k] || strings.ContainsAny(k, " .'?") {
			t.Errorf("key %q repeats or has spaces or punctuation", k)
		}
		seen[k] = true
		if j, ok := FindGoodbye(k); !ok || j != i {
			t.Errorf("FindGoodbye(%q) = %d, want %d", k, j, i)
		}
	}
	if k := GoodbyeKey(6); k != "its-now-safe-to-close-your-terminal" {
		t.Errorf("key %q", k)
	}
}

// The same goodbye never comes twice in a row, and every one comes up.
func TestPickGoodbyeNeverRepeats(t *testing.T) {
	r := rand.New(rand.NewPCG(3, 4))
	seen, last := map[int]bool{}, -1
	for range 1000 {
		i := PickGoodbye(last, r)
		if i == last {
			t.Fatalf("goodbye %d twice in a row", i)
		}
		seen[i], last = true, i
	}
	if len(seen) != GoodbyeCount() {
		t.Errorf("%d of %d goodbyes came up", len(seen), GoodbyeCount())
	}
}
