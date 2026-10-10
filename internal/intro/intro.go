package intro

import (
	"math/rand/v2"
	"strings"
	"time"
)

// Count is how many intros there are.
func Count() int { return len(intros) }

// Name is intro i's name, as shown to people.
func Name(i int) string { return intros[i].name }

// Key is intro i's name for config.json: lower case, dashes, no notes in
// brackets ("CRT power-on" is "crt-power-on").
func Key(i int) string {
	name := intros[i].name
	if cut := strings.Index(name, " ("); cut > 0 {
		name = name[:cut]
	}
	return strings.ReplaceAll(strings.ToLower(name), " ", "-")
}

// Find is the intro with the given key.
func Find(key string) (int, bool) {
	for i := range intros {
		if Key(i) == key {
			return i, true
		}
	}
	return 0, false
}

// Length is how long intro i runs.
func Length(i int) time.Duration {
	return time.Duration(intros[i].dur * float64(time.Second))
}

// Pick chooses an intro at random, never last (the previous start's), so
// the same one never plays twice in a row. last may be -1.
func Pick(last int, r *rand.Rand) int { return pick(len(intros), last, r) }

// Run is one playing of an intro: which one, and the randomness it drew,
// fixed for the run so its frames are consistent.
type Run struct {
	intro int
	fx    fxState
}

// Start begins intro i on this screen.
func (sc Screen) Start(i int, seed uint64) *Run {
	rng := rand.New(rand.NewPCG(seed, uint64(i)+1))
	run := &Run{intro: i, fx: fxState{rnd: make([][3]float64, len(sc.s.logo)), cache: map[string][]float64{}}}
	for k := range run.fx.rnd {
		run.fx.rnd[k] = [3]float64{rng.Float64(), rng.Float64(), rng.Float64()}
	}
	return run
}

// fps is the rate the intros are drawn at: enough for cell-sized motion.
const fps = 30

// Frame is the run's picture at elapsed. done is true once the intro is
// over (the caller then shows Rest), and also when the screen no longer
// matches the run (the window was resized mid-intro).
func (sc Screen) Frame(run *Run, elapsed time.Duration) (frame string, done bool) {
	it := intros[run.intro]
	t := elapsed.Seconds()
	if t >= it.dur || len(run.fx.rnd) != len(sc.s.logo) {
		return "", true
	}
	run.fx.frame = int(t * fps)
	g := newGrid(sc.s.w, sc.s.h)
	it.draw(g, sc.s, t, &run.fx)
	sc.frame(g)
	return g.String(), false
}

// BasicGlyphs is set for the classic Windows console, whose fonts lack some
// characters the intros draw (the UI's glyphs.go).
var BasicGlyphs bool

// diamond is the pattern on the back of the Card flip's card: ◆, or ♦
// where ◆ is missing.
func diamond() rune {
	if BasicGlyphs {
		return '♦'
	}
	return '◆'
}
