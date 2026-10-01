package intro

import (
	"testing"
	"time"
)

// frameTarget is one frame at the intros' 30 fps (Phase C, C4).
const frameTarget = time.Second / fps

// Every frame of every intro, on a big window, is drawn well inside a frame
// at 30 fps; each frame is timed over a batch of 10 for the clock's sake.
func TestIntroFrameSpeed(t *testing.T) {
	if testing.Short() {
		t.Skip("speed check skipped in -short mode")
	}
	sc, ok := NewScreen(240, 67, title, checks)
	if !ok {
		t.Fatal("the big screen should fit")
	}
	var slowest time.Duration
	var slowestName string
	for i := range Count() {
		run := sc.Start(i, 1)
		for f := 0; ; f++ {
			at := time.Duration(f) * time.Second / fps
			start := time.Now()
			if _, done := sc.Frame(run, at); done {
				break
			}
			for range 9 {
				sc.Frame(run, at)
			}
			if d := time.Since(start) / 10; d > slowest {
				slowest, slowestName = d, Name(i)
			}
		}
	}
	if slowest > frameTarget {
		t.Errorf("the slowest intro frame (%s) took %v, target %v", slowestName, slowest, frameTarget)
	} else {
		t.Logf("slowest intro frame: %v (%s), target %v", slowest, slowestName, frameTarget)
	}
}
