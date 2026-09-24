package ui

import (
	"fmt"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"

	"github.com/stronxyo/quarter/internal/cache"
	"github.com/stronxyo/quarter/internal/library"
)

// The brief's speed targets for a large library.
const (
	benchGames     = 5000
	startupTarget  = 100 * time.Millisecond
	keystrokeFrame = 16 * time.Millisecond // one frame at 60 fps
)

// bigLibrary builds a library of n games with varied names, playtime and dates.
func bigLibrary(n int) library.Result {
	words := []string{"Space", "Dark", "Legend", "Rocket", "Racing", "Souls", "City", "Quest", "Tactics", "Hollow", "Knight", "Star", "Farm", "Dungeon", "Rally", "Cyber"}
	now := time.Date(2026, 9, 24, 12, 0, 0, 0, time.UTC)
	res := library.Result{PlaytimeKnown: true}
	for i := range n {
		res.Games = append(res.Games, library.Game{
			AppID:      uint32(i + 1),
			Name:       fmt.Sprintf("%s %s %s %d", words[i%16], words[(i/16)%16], words[(i/256)%16], i),
			Installed:  true,
			Playtime:   time.Duration(i%900) * time.Hour,
			LastPlayed: now.Add(-time.Duration(i%400) * 24 * time.Hour),
		})
	}
	return res
}

// startup does what launching Quarter does before the first frame: read the
// cached library from disk, build the UI, size it, and render it.
func startup(dir string) string {
	res, ok := cache.Load[library.Result](dir, "library.json", 1)
	if !ok {
		panic("benchmark cache missing")
	}
	var m tea.Model = New("", &res, nil)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return m.(Model).View().Content
}

// keystroke does what typing one search character does: filter every game and
// render the result.
func keystroke(m Model, query string) string {
	m.list.SetFilterText(query)
	return m.View().Content
}

func setupBench(tb testing.TB) (dir string, ready Model) {
	tb.Helper()
	dir = tb.TempDir()
	lib := bigLibrary(benchGames)
	if err := cache.Save(dir, "library.json", 1, lib); err != nil {
		tb.Fatal(err)
	}
	var m tea.Model = New("", &lib, nil)
	m, _ = m.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	return dir, m.(Model)
}

func BenchmarkStartup(b *testing.B) {
	dir, _ := setupBench(b)
	for b.Loop() {
		startup(dir)
	}
}

func BenchmarkSearchKeystroke(b *testing.B) {
	_, m := setupBench(b)
	for b.Loop() {
		keystroke(m, "roc")
	}
}

// fastest runs f five times and returns the quickest, so a busy machine does
// not fail the check by accident.
func fastest(f func()) time.Duration {
	best := time.Duration(1<<63 - 1)
	for range 5 {
		start := time.Now()
		f()
		best = min(best, time.Since(start))
	}
	return best
}

func TestSpeedTargets(t *testing.T) {
	if testing.Short() {
		t.Skip("speed check skipped in -short mode")
	}
	dir, m := setupBench(t)

	if got := fastest(func() { startup(dir) }); got > startupTarget {
		t.Errorf("startup with %d games took %v, target %v", benchGames, got, startupTarget)
	} else {
		t.Logf("startup with %d games: %v (target %v)", benchGames, got, startupTarget)
	}

	if got := fastest(func() { keystroke(m, "roc") }); got > keystrokeFrame {
		t.Errorf("search keystroke over %d games took %v, target %v", benchGames, got, keystrokeFrame)
	} else {
		t.Logf("search keystroke over %d games: %v (target %v)", benchGames, got, keystrokeFrame)
	}
}
