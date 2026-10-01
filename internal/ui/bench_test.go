package ui

import (
	"fmt"
	"slices"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/replay"
	"github.com/stronxyo/quarter/internal/steam"
	"github.com/stronxyo/quarter/internal/store"
	"github.com/stronxyo/quarter/internal/webapi"
)

// The brief's speed targets, and Phase C's (specs/BATCH_08_SPEC.md, C1 and
// C4), on a large library: TestSpeedTargets fails the build past them.
const (
	benchGames     = 5000
	benchSessions  = 20000 // about five years of heavy play
	startupTarget  = 100 * time.Millisecond
	keystrokeFrame = 16 * time.Millisecond // one frame at 60 fps: any key, any screen
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

// bigSessions are benchSessions sessions going back from now, spread over
// the games.
func bigSessions(res library.Result) []steam.Session {
	now := time.Now()
	s := make([]steam.Session, 0, benchSessions)
	for i := range benchSessions {
		start := now.Add(-time.Duration(i) * 131 * time.Minute)
		s = append(s, steam.Session{AppID: res.Games[(i*7919)%len(res.Games)].AppID, Start: start, End: start.Add(time.Duration(20+i%180) * time.Minute), EndKnown: true})
	}
	slices.SortFunc(s, func(a, b steam.Session) int { return a.Start.Compare(b.Start) })
	return s
}

// bigYears are four past years, 200 games played each month.
func bigYears(res library.Result) []replay.Year {
	var years []replay.Year
	for y := 2022; y <= 2025; y++ {
		year := replay.Year{Year: y, Games: map[uint32]time.Duration{}}
		for mo := range 12 {
			month := replay.Month{Start: time.Date(y, time.Month(mo+1), 1, 0, 0, 0, 0, time.UTC), Games: map[uint32]time.Duration{}}
			for k := range 200 {
				id := res.Games[(mo*200+k)%len(res.Games)].AppID
				month.Games[id] += time.Hour
				year.Games[id] += time.Hour
				month.Played += time.Hour
			}
			year.Played += month.Played
			year.Months = append(year.Months, month)
		}
		years = append(years, year)
	}
	return years
}

// benchData saves what Quarter reads at startup for the large library.
func benchData(tb testing.TB) (dir string, lib library.Result) {
	tb.Helper()
	dir, lib = tb.TempDir(), bigLibrary(benchGames)
	info := map[uint32]webapi.StoreEntry{}
	for _, g := range lib.Games {
		info[g.AppID] = webapi.StoreEntry{Fetched: time.Now(), Info: webapi.StoreInfo{
			Name: g.Name, Type: "game", Developers: []string{"Studio"}, Genres: []string{"Action", "Racing"}, ReleaseDate: "6 Jul, 2015",
			Description: "A game about driving very fast through a long and winding course, against the clock and other players.",
		}}
	}
	for name, v := range map[string]any{"library.json": lib, "store.json": info, "sessions.json": bigSessions(lib), "replay.json": bigYears(lib)} {
		if err := store.Save(dir, name, 1, v); err != nil {
			tb.Fatal(err)
		}
	}
	return dir, lib
}

// startup does what launching Quarter does before the first frame: read the
// saved library, store info, sessions and past years, build the UI, size it
// and draw the start screen's first frame. (Go's own start and Bubble Tea's
// wait for its next frame come on top: about 30 ms on Pablo's PC, C1.)
func startup(dir string) string {
	res, _ := store.Load[library.Result](dir, "library.json", 1)
	info, _ := store.Load[map[uint32]webapi.StoreEntry](dir, "store.json", 1)
	sessions, _ := store.Load[[]steam.Session](dir, "sessions.json", 1)
	years, _ := store.Load[[]replay.Year](dir, "replay.json", 1)
	var m tea.Model = New(Options{Cached: &res, StoreCache: info, Sessions: sessions, Replay: years, Splash: true})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 160, Height: 45})
	return m.(Model).View().Content
}

// benchModel is the large library on screen, past the start screen, with a
// logo band (no logos on disk); change may alter the library first.
func benchModel(tb testing.TB, change ...func(*library.Result)) Model {
	tb.Helper()
	lib := bigLibrary(benchGames)
	for _, c := range change {
		c(&lib)
	}
	logos := &fakeLogos{paths: map[uint32][]string{}}
	var m tea.Model = New(Options{Cached: &lib, Sessions: bigSessions(lib), Replay: bigYears(lib), Logos: func(string) LogoSource { return logos }})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 160, Height: 45})
	m, _ = m.Update(scanDoneMsg{res: lib})
	return m.(Model)
}

// keystroke does what typing one search character does: filter every game and
// render the result.
func keystroke(m Model, query string) string {
	m.list.SetFilterText(query)
	return m.View().Content
}

// press handles one key and draws the screen, as the program does.
func pressDraw(m Model, k tea.KeyPressMsg) Model {
	tm, _ := m.Update(k)
	tm.(Model).View()
	return tm.(Model)
}

// holding is the model on a Sixel terminal with a key held down, on a clock
// the returned step moves 33 ms (a key repeat) at a time.
func holding(m Model) (Model, func(Model) Model) {
	tm, _ := m.Update(uv.PrimaryDeviceAttributesEvent{62, 4, 22})
	tm, _ = tm.Update(uv.CellSizeEvent{Width: 10, Height: 20})
	m = tm.(Model)
	clock := time.Now()
	m.now = func() time.Time { return clock }
	step := func(m Model) Model {
		clock = clock.Add(33 * time.Millisecond)
		return pressDraw(m, tea.KeyPressMsg{Code: tea.KeyDown})
	}
	for range 3 {
		m = step(m) // two quick steps: held
	}
	return m, step
}

// downloadingBlink is the large library with its selected game
// downloading, and one blink of its arrow with the screen drawn.
func downloadingBlink(tb testing.TB) (Model, func(Model) Model) {
	first, _ := benchModel(tb).selected()
	m := benchModel(tb, func(lib *library.Result) { lib.Games[first.AppID-1].Updating = true })
	g, _ := m.selected()
	if !g.Updating {
		tb.Fatalf("the downloading game should be selected, got %q", g.Name)
	}
	m.downloads = map[uint32]steam.Download{g.AppID: {AppID: g.AppID, Phase: steam.PhaseDownloading, Done: 1e9, Total: 1e10, Rate: 1e7, At: time.Now()}}
	return m, func(m Model) Model {
		tm, _ := m.Update(dlBlinkMsg{})
		tm.(Model).View()
		return tm.(Model)
	}
}

func BenchmarkStartup(b *testing.B) {
	dir, _ := benchData(b)
	for b.Loop() {
		startup(dir)
	}
}

func BenchmarkSearchKeystroke(b *testing.B) {
	m := benchModel(b)
	for b.Loop() {
		keystroke(m, "roc")
	}
}

func BenchmarkHoldStep(b *testing.B) {
	m, step := holding(benchModel(b))
	for b.Loop() {
		m = step(m)
	}
}

func BenchmarkBoard(b *testing.B) {
	board := pressDraw(pressDraw(benchModel(b), press("h")), press("h"))
	for b.Loop() {
		pressDraw(board, tea.KeyPressMsg{Code: tea.KeyTab})
	}
}

func BenchmarkBlink(b *testing.B) {
	m, blink := downloadingBlink(b)
	for b.Loop() {
		m = blink(m)
	}
}

// fastest returns the quickest of five timings of f, so a busy machine does
// not fail the check by accident. Windows' clock ticks about every 0.5 ms,
// so each timing covers a batch of runs (at least 5 ms worth) and divides.
func fastest(f func()) time.Duration {
	batch := 1
	for start := time.Now(); ; batch *= 2 {
		for range batch {
			f()
		}
		if time.Since(start) >= 5*time.Millisecond || batch >= 1<<12 {
			break
		}
		start = time.Now()
	}
	best := time.Duration(1<<63 - 1)
	for range 5 {
		start := time.Now()
		for range batch {
			f()
		}
		best = min(best, time.Since(start)/time.Duration(batch))
	}
	return best
}

func TestSpeedTargets(t *testing.T) {
	if testing.Short() {
		t.Skip("speed check skipped in -short mode")
	}
	dir, _ := benchData(t)
	m := benchModel(t)
	held, step := holding(m)
	ranking := pressDraw(m, press("h"))
	board := pressDraw(ranking, press("h"))
	dl, blink := downloadingBlink(t)

	checks := []struct {
		name   string
		target time.Duration
		f      func()
	}{
		{"startup, saved files to the first frame", startupTarget, func() { startup(dir) }},
		{"search keystroke", keystrokeFrame, func() { keystroke(m, "roc") }},
		{"a held step, pictures on", keystrokeFrame, func() { step(held) }},
		{"h, the ranking", keystrokeFrame, func() { pressDraw(m, press("h")) }},
		{"h, the HIGH SCORES board", keystrokeFrame, func() { pressDraw(ranking, press("h")) }},
		{"tab on the board", keystrokeFrame, func() { pressDraw(board, tea.KeyPressMsg{Code: tea.KeyTab}) }},
		{"a download's blink", keystrokeFrame, func() { blink(dl) }},
	}
	for _, c := range checks {
		if got := fastest(c.f); got > c.target {
			t.Errorf("%s, %d games and %d sessions: %v, target %v", c.name, benchGames, benchSessions, got, c.target)
		} else {
			t.Logf("%-42s %8v (target %v)", c.name, got, c.target)
		}
	}
}
