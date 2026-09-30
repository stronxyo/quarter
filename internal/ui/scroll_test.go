package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/library"
)

// scrollModel is a Sixel terminal with n games, all with a logo on disk but
// those in noDisk, on a clock the test moves; the first game's picture is on
// screen.
func scrollModel(t *testing.T, n int, noDisk ...uint32) (Model, *time.Time, *fakeLogos) {
	t.Helper()
	clock := time.Date(2026, 9, 30, 12, 0, 0, 0, time.Local)
	logos := &fakeLogos{paths: map[uint32][]string{}, missing: map[uint32]bool{}}
	png := logoPNG(t)
	var games []library.Game
	for i := range n {
		id := uint32(i + 1)
		games = append(games, library.Game{AppID: id, Name: fmt.Sprintf("Game %03d", id), Installed: true})
		logos.paths[id] = []string{png}
	}
	for _, id := range noDisk {
		delete(logos.paths, id)
		logos.missing[id] = true // and none on the CDN: never downloaded
	}
	m := New(Options{Settings: config.Settings{}, Logos: func(string) LogoSource { return logos }})
	m.now = func() time.Time { return clock }
	m.debounce = 0
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 150, Height: 40})
	tm, cmd := tm.Update(scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: games}})
	tm = pump(tm, cmd)
	tm, cmd = tm.Update(uv.PrimaryDeviceAttributesEvent{62, 4, 22})
	tm = pump(tm, cmd)
	tm, _ = tm.Update(uv.CellSizeEvent{Width: 10, Height: 20})
	clock = clock.Add(time.Second) // landed on the first game
	tm, cmd = tm.Update(settleMsg{})
	tm = pump(tm, cmd)
	m = tm.(Model)
	m.pictureShown = m.pictureSig()
	return m, &clock, logos
}

func holdStep(m Model, clock *time.Time, d time.Duration) (Model, tea.Cmd) {
	*clock = clock.Add(d)
	tm, cmd := m.Update(tea.KeyPressMsg{Code: tea.KeyDown})
	return tm.(Model), cmd
}

// bandText is the logo band's text on screen.
func bandText(m Model) string {
	x, cols, rows, _ := m.logoBand()
	var b strings.Builder
	for _, l := range strings.Split(ansi.Strip(m.View().Content), "\n")[logoPadTop : logoPadTop+rows] {
		b.WriteString(ansi.Cut(l, x, x+cols) + "\n")
	}
	return b.String()
}

// Holding down reads no logo for the games passed, drops the last picture at
// once and shows the dots.
func TestHoldingDown(t *testing.T) {
	m, clock, logos := scrollModel(t, 200)
	if m.pictureShown.cols == 0 {
		t.Fatal("the first game's picture should be on screen")
	}
	reads := logos.reads
	m, _ = holdStep(m, clock, 0)                    // a press
	m, _ = holdStep(m, clock, 500*time.Millisecond) // the first repeat
	m, cmd := holdStep(m, clock, 33*time.Millisecond)
	if m.pictureShown.cols != 0 {
		t.Error("the picture should be erased at the first quick step")
	}
	if raw, ok := cmd().(tea.BatchMsg); !ok || len(raw) == 0 {
		t.Error("the erase should be sent at once")
	}
	for range 100 {
		m, _ = holdStep(m, clock, 33*time.Millisecond)
	}
	if logos.reads != reads || len(m.logoPending) != 0 {
		t.Errorf("no logo should be read while holding: %d reads, %d pending", logos.reads-reads, len(m.logoPending))
	}
	if !m.showsDots() || !strings.Contains(bandText(m)+bandTextLater(m, clock), dot) {
		t.Errorf("the band should show the dots while holding:\n%s", bandText(m))
	}
}

// bandTextLater is the band one dot step later, in case this one is the
// empty step.
func bandTextLater(m Model, clock *time.Time) string {
	*clock = clock.Add(dotsStep)
	defer func() { *clock = clock.Add(-dotsStep) }()
	m.lastMove, m.prevMove = *clock, clock.Add(-33*time.Millisecond)
	return bandText(m)
}

// A single step keeps the picture until the next one is ready, and shows no
// dots: nothing flashes.
func TestSingleStepNoDots(t *testing.T) {
	m, clock, _ := scrollModel(t, 20)
	m, _ = holdStep(m, clock, 0)
	if m.pictureShown.cols == 0 || m.showsDots() || m.holding() {
		t.Error("a single step should neither erase the picture at once nor show dots")
	}
}

// Landing reads the game's logo, then the games next to it, once each; a
// neighbour with nothing on disk is read once and left until landed on.
func TestLandingPreparesNeighbours(t *testing.T) {
	m, clock, logos := scrollModel(t, 20, 12)
	for range 10 {
		m, _ = holdStep(m, clock, 33*time.Millisecond)
	}
	*clock = clock.Add(time.Second)
	tm, cmd := m.Update(settleMsg{})
	m = pump(tm, cmd).(Model)
	g, _ := m.selected()
	_, cols, rows, _ := m.logoBand()
	if g.AppID != 11 || !m.hasLogo(logoKey{11, cols, rows}) {
		t.Fatalf("the landed game's logo should be drawn: on %d", g.AppID)
	}
	for _, id := range []uint32{10, 9, 13} {
		if !m.hasLogo(logoKey{id, cols, rows}) {
			t.Errorf("game %d, next to it, should be ready", id)
		}
	}
	if !m.logoNoDisk[12] {
		t.Error("game 12 has nothing on disk and should be left")
	}
	reads := logos.reads
	for range 5 {
		tm, cmd = m.Update(settleMsg{})
		m = pump(tm, cmd).(Model)
	}
	if logos.reads != reads {
		t.Errorf("nothing should be read again: %d reads", logos.reads-reads)
	}
}

// A late logo shows the dots after dotsWait, not before.
func TestDotsForLateLogo(t *testing.T) {
	m, clock, _ := scrollModel(t, 20)
	tm, _ := m.Update(press("G")) // the last game: not prepared
	m = tm.(Model)
	*clock = clock.Add(m.settle)
	tm, _ = m.Update(settleMsg{}) // the read starts, its answer is held back
	m = tm.(Model)
	if m.showsDots() {
		t.Error("no dots before dotsWait")
	}
	*clock = clock.Add(dotsWait)
	if !m.showsDots() {
		t.Error("the dots should show for a logo still on its way")
	}
	m.dotsOn = true
	m.logoPending = map[logoKey]bool{}
	if m, cmd := m.dotsTick(); cmd != nil || m.dotsOn {
		t.Error("the dots stop once they no longer show")
	}
}

// The dots fill fixed places: the first dot stays where it is as the others
// come.
func TestDotsDoNotShift(t *testing.T) {
	m, clock, _ := scrollModel(t, 20)
	var at []int
	for n := 1; n <= dotSlots; n++ {
		*clock = time.UnixMilli(int64(n) * dotsStep.Milliseconds())
		at = append(at, strings.Index(ansi.Strip(m.dotsView(40, 3)), dot))
	}
	if at[0] < 0 || at[0] != at[1] || at[1] != at[2] {
		t.Errorf("the first dot should stay put: %v", at)
	}
}

func TestKeptLogos(t *testing.T) {
	m, _, _ := scrollModel(t, 60)
	g, _ := m.selected()
	_, cols, rows, _ := m.logoBand()
	for id := uint32(1); id <= 60; id++ {
		key := logoKey{id, cols, rows}
		m.logoCache[key] = drawnLogo{picture: "p"}
		m.keepLogo(key)
	}
	if len(m.logoCache) > keptLogos || !m.hasLogo(logoKey{g.AppID, cols, rows}) {
		t.Errorf("at most %d kept, the selected game's always: %d kept", keptLogos, len(m.logoCache))
	}
}
