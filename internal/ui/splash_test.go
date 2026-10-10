package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/intro"
	"github.com/stronxyo/quarter/internal/library"
)

// splashedWith is a model on the start screen, w x h, with n games scanned,
// the given settings, a clock the test moves, and the settings it saves.
func splashedWith(t *testing.T, w, h, n int, s config.Settings) (Model, *time.Time, *[]config.Settings) {
	t.Helper()
	saved := &[]config.Settings{}
	clock := time.Date(2026, 9, 28, 12, 0, 0, 0, time.UTC)
	m := New(Options{Splash: true, Settings: s, SaveSettings: func(s config.Settings) error { *saved = append(*saved, s); return nil }})
	m.now = func() time.Time { return clock }
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: w, Height: h})
	res := library.Result{PlaytimeKnown: true}
	for i := range n {
		res.Games = append(res.Games, library.Game{AppID: uint32(i + 1), Name: "Game", Installed: true})
	}
	tm, _ = tm.Update(scanDoneMsg{res: res})
	return tm.(Model), &clock, saved
}

// splashed is the start screen with intros off, resting.
func splashed(t *testing.T, w, h, n int) (Model, *[]config.Settings) {
	t.Helper()
	m, _, saved := splashedWith(t, w, h, n, config.Settings{Intro: introOff})
	return m, saved
}

func screen(m Model) string { return ansi.Strip(m.View().Content) }

// centred reports whether text sits in the middle of a w-wide screen.
func centred(t *testing.T, view, text string, w int) {
	t.Helper()
	for _, l := range strings.Split(view, "\n") {
		if i := strings.Index(l, text); i >= 0 {
			left := ansi.StringWidth(l[:i])
			if d := left - (w - left - len(text)); d < -1 || d > 1 {
				t.Errorf("%q should be centred: %d left, %d right", text, left, w-left-len(text))
			}
			return
		}
	}
	t.Errorf("%q should be on the screen", text)
}

// The start screen: the system check, the logo, insert quarter blinking and
// press any key, the title and the prompt centred, all within the window.
func TestSplashRest(t *testing.T) {
	m, _ := splashed(t, 120, 40, 300)
	dark := screen(m)
	for _, want := range []string{"QUARTER SYSTEM CHECK", "Looking for Steam", "OK", "300 games", "press any key", "██"} {
		if !strings.Contains(dark, want) {
			t.Errorf("the start screen should show %q:\n%s", want, dark)
		}
	}
	next, cmd := m.Update(blinkMsg{})
	lit := screen(next.(Model))
	if cmd == nil || !strings.Contains(lit, "insert quarter") || strings.Contains(dark, "insert quarter") {
		t.Error("insert quarter should blink")
	}
	for _, l := range strings.Split(lit, "\n") {
		if ansi.StringWidth(l) > 120 {
			t.Errorf("a line is wider than the window: %q", l)
		}
	}
	centred(t, lit, "QUARTER SYSTEM CHECK", 120)
	centred(t, lit, "press any key", 120)
}

// On a big window the logo is twice the size and insert quarter is drawn in
// its letters.
func TestSplashBigWindow(t *testing.T) {
	m, _ := splashed(t, 200, 60, 3)
	small, _ := splashed(t, 120, 40, 3)
	next, _ := m.Update(blinkMsg{})
	lit := screen(next.(Model))
	if strings.Count(lit, "█") < 2*strings.Count(screen(small), "█") {
		t.Error("a big window should have the logo at double size")
	}
	if !strings.ContainsAny(lit, "▀▄") || strings.Contains(lit, "insert quarter") {
		t.Error("beside the big logo, insert quarter is drawn in its letters")
	}
}

// Before the scan answers, the checks show dots; if Steam isn't found, they
// say so.
func TestSplashFollowsScan(t *testing.T) {
	m := New(Options{Splash: true, Settings: config.Settings{Intro: introOff}})
	m.width, m.height = 120, 40
	if v := screen(m); !strings.Contains(v, "Looking for Steam") || strings.Contains(v, "OK") {
		t.Errorf("while scanning, no result yet:\n%s", v)
	}
	next, _ := m.Update(scanDoneMsg{err: errors.New("steam install not found")})
	if v := screen(next.(Model)); !strings.Contains(v, "NOT FOUND") {
		t.Errorf("a failed scan should show NOT FOUND:\n%s", v)
	}
}

// The chosen intro is saved; every intro plays, then the start screen rests.
func TestIntroPlaysThenRests(t *testing.T) {
	m, _, saved := splashedWith(t, 160, 44, 3, config.Settings{})
	if m.introIndex < 0 {
		t.Fatal("an intro should be chosen")
	}
	messages(m.Init()) // saves which one, for the next start
	if len(*saved) == 0 || (*saved)[len(*saved)-1].LastIntro != intro.Key(m.introIndex) {
		t.Errorf("the chosen intro should be saved, got %+v", *saved)
	}
	// Each intro, by name: it plays, then rests.
	for i := range intro.Count() {
		m, clock, _ := splashedWith(t, 160, 44, 3, config.Settings{Intro: intro.Key(i)})
		rest, _ := m.startScreen()
		next, cmd := m.Update(introTickMsg{})
		m = next.(Model)
		if m.introRun == nil || cmd == nil {
			t.Fatalf("%s: the intro should be running and ask for the next frame", intro.Key(i))
		}
		*clock = clock.Add(intro.Length(i) / 2)
		// Colours included: some intros animate colour only (a light sweep).
		if v := m.View().Content; v == rest.Rest(false) || v == rest.Rest(true) {
			t.Errorf("halfway, %s's frame should show, not the start screen", intro.Key(i))
		}
		*clock = clock.Add(intro.Length(i))
		next, cmd = m.Update(introTickMsg{})
		m = next.(Model)
		if !m.introDone || cmd != nil || !m.splash {
			t.Errorf("%s: once over, the intro stops and the start screen rests", intro.Key(i))
		}
	}
}

// Nothing of the start screen shows before an intro: an empty screen until
// the window's size is known, then straight away the intro's first frame,
// for every intro.
func TestNoGlimpseBeforeIntro(t *testing.T) {
	m := New(Options{Splash: true})
	if m.introIndex < 0 {
		t.Fatal("an intro should be chosen")
	}
	if v := m.View().Content; v != "" {
		t.Errorf("before the window's size is known, the screen should be empty, got %q", v)
	}
	for i := range intro.Count() {
		m, _, _ := splashedWith(t, 160, 44, 3, config.Settings{Intro: intro.Key(i)})
		sc, _ := m.startScreen()
		first := m.View().Content
		if m.introRun == nil || first == sc.Rest(false) || first == sc.Rest(true) {
			t.Errorf("%s: the first frame should be the intro's, not the start screen", intro.Key(i))
		}
	}
}

// The first key skips the intro to the start screen; the next one starts.
func TestKeySkipsIntro(t *testing.T) {
	m, _, _ := splashedWith(t, 160, 44, 3, config.Settings{})
	next, _ := m.Update(introTickMsg{})
	next, _ = next.Update(press("x"))
	m = next.(Model)
	if !m.introDone || !m.splash {
		t.Fatal("a key during the intro should skip to the start screen")
	}
	next, _ = m.Update(press("x"))
	if next.(Model).splash {
		t.Error("the next key should start")
	}
}

// "intro": "off" means no intro; a name means that one.
func TestIntroSetting(t *testing.T) {
	m, _, _ := splashedWith(t, 160, 44, 3, config.Settings{Intro: "off"})
	if m.introIndex != -1 || m.introTick() != nil {
		t.Error(`"off" should play no intro`)
	}
	want, _ := intro.Find("crt-power-on")
	if m, _, _ := splashedWith(t, 160, 44, 3, config.Settings{Intro: "crt-power-on"}); m.introIndex != want {
		t.Errorf("a named intro should play, got %d", m.introIndex)
	}
}

// The intro never repeats the one played last time.
func TestIntroNotRepeated(t *testing.T) {
	for i := range intro.Count() {
		m, _, _ := splashedWith(t, 160, 44, 3, config.Settings{LastIntro: intro.Key(i)})
		if m.introIndex == i {
			t.Errorf("last time's intro, %s, came up again", intro.Key(i))
		}
	}
}

// Bubble Tea sends the window's size twice at startup (once as it starts,
// once from checkResize, the same size): the intro plays on through that.
func TestIntroSurvivesStartupSizes(t *testing.T) {
	m := New(Options{Splash: true})
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 160, Height: 44})
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 160, Height: 44})
	tm, cmd := tm.Update(introTickMsg{})
	if got := tm.(Model); got.introDone || got.introRun == nil || cmd == nil {
		t.Error("the same size again is not a resize: the intro should still be playing")
	}
}

// Resizing mid-intro goes straight to the start screen at the new size.
func TestIntroResize(t *testing.T) {
	m, _, _ := splashedWith(t, 160, 44, 3, config.Settings{})
	next, _ := m.Update(introTickMsg{})
	next, _ = next.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if !next.(Model).introDone {
		t.Error("a resize mid-intro should end it")
	}
}

// Any key starts, and the start screen is saved as seen; the blinking stops.
func TestSplashAnyKeyStarts(t *testing.T) {
	m, saved := splashed(t, 120, 40, 3)
	next, cmd := m.Update(press("x"))
	m = next.(Model)
	if m.splash || strings.Contains(screen(m), "QUARTER SYSTEM CHECK") {
		t.Fatal("a key should leave the start screen for the list")
	}
	messages(cmd) // runs the save
	if len(*saved) != 1 || !(*saved)[0].SplashSeen {
		t.Errorf("leaving should save it as seen, got %+v", *saved)
	}
	if m.blink() != nil {
		t.Error("the blinking should stop once the start screen is gone")
	}
}

// On the start screen only ctrl+c quits, during an intro too.
func TestSplashQuit(t *testing.T) {
	for _, s := range []config.Settings{{Intro: introOff}, {}} {
		m, _, _ := splashedWith(t, 120, 40, 3, s)
		_, cmd := m.Update(tea.KeyPressMsg{Code: 'c', Mod: tea.ModCtrl})
		if _, ok := cmd().(tea.QuitMsg); !ok {
			t.Error("ctrl+c should quit from the start screen")
		}
	}
}

// q is "any key" on the start screen: it skips the intro, then starts, and
// never quits.
func TestSplashQIsAnyKey(t *testing.T) {
	m, _, _ := splashedWith(t, 160, 44, 3, config.Settings{})
	next, _ := m.Update(introTickMsg{})
	next, cmd := next.Update(press("q"))
	if got := next.(Model); !got.introDone || !got.splash || isQuit(cmd) {
		t.Fatal("q during the intro should skip it, not quit")
	}
	next, cmd = next.Update(press("q"))
	if next.(Model).splash || isQuit(cmd) {
		t.Error("q on the start screen should start, not quit")
	}
}

// isQuit reports whether cmd, or any command batched in it, quits.
func isQuit(cmd tea.Cmd) bool {
	for _, msg := range messages(cmd) {
		if _, ok := msg.(tea.QuitMsg); ok {
			return true
		}
	}
	return false
}

// Without the start screen, Quarter opens on the list; no game logo is
// drawn over the start screen while it is up.
func TestSplashOnlyWhenAsked(t *testing.T) {
	m := ready(t, "Alpha")
	if m.splash || strings.Contains(screen(m), "QUARTER SYSTEM CHECK") {
		t.Error("without Splash, Quarter opens on the list")
	}
	s, _ := splashed(t, 150, 40, 3)
	s.logos = &fakeLogos{paths: map[uint32][]string{}}
	if s.showsLogos() {
		t.Error("no game logo while the start screen is up")
	}
}

// On a window too small for the logo box, the compact screen: the badge,
// insert quarter as text, nothing overflowing.
func TestSplashSmallWindow(t *testing.T) {
	m, _ := splashed(t, 60, 20, 3)
	view := screen(m)
	if !strings.Contains(view, "quarter_") || strings.Contains(view, "██") {
		t.Errorf("a small window should show the badge, not the big logo:\n%s", view)
	}
	next, _ := m.Update(blinkMsg{})
	if !strings.Contains(screen(next.(Model)), "insert quarter") {
		t.Error("a small window should show insert quarter as text")
	}
	for _, l := range strings.Split(view, "\n") {
		if ansi.StringWidth(l) > 60 {
			t.Errorf("a line is wider than the window: %q", l)
		}
	}
}

// As the start screen waits, a band of light crosses the logo every 10 to
// 20 seconds: colour only, then the screen rests again. With intros off,
// no intro plays again.
func TestShineWhileWaiting(t *testing.T) {
	m, clock, _ := splashedWith(t, 160, 44, 3, config.Settings{Intro: introOff})
	next, _ := m.Update(blinkMsg{})
	m = next.(Model)
	if wait := m.shineAt.Sub(*clock); wait < shineGap[0] || wait > shineGap[1] {
		t.Fatalf("the first shine should come 10 to 20 s after the screen rests, got %v", wait)
	}
	sc, _ := m.startScreen()
	*clock = m.shineAt
	next, _ = m.Update(blinkMsg{})
	m = next.(Model)
	if m.shineStart.IsZero() {
		t.Fatal("the shine should start when its time comes")
	}
	if gap := m.shineAt.Sub(m.shineStart) - intro.ShineLength; gap < shineGap[0] || gap > shineGap[1] {
		t.Errorf("the next shine should come 10 to 20 s after this one ends, got %v", gap)
	}
	*clock = clock.Add(intro.ShineLength / 2)
	v := m.View().Content
	if v == sc.Rest(m.blinkOn) {
		t.Error("halfway, the logo should shine")
	}
	if ansi.Strip(v) != ansi.Strip(sc.Rest(m.blinkOn)) {
		t.Error("the shine should change colours only")
	}
	*clock = clock.Add(intro.ShineLength)
	next, cmd := m.Update(shineTickMsg{})
	m = next.(Model)
	if !m.shineStart.IsZero() || cmd != nil || m.View().Content != sc.Rest(m.blinkOn) {
		t.Error("once across, the shine stops and the screen rests")
	}
	*clock = clock.Add(2 * time.Minute)
	next, _ = m.Update(blinkMsg{})
	if next.(Model).introPending() {
		t.Error(`with "intro": "off", no intro plays again`)
	}
}

// Every 45 to 60 seconds an intro plays again, never the one before; once
// over, the waits start again from the rest. A key during it starts.
func TestIntroPlaysAgain(t *testing.T) {
	m, clock, _ := splashedWith(t, 160, 44, 3, config.Settings{})
	played := func(m Model) Model {
		t.Helper()
		next, _ := m.Update(introTickMsg{})
		*clock = clock.Add(intro.Length(m.introIndex))
		next, _ = next.Update(introTickMsg{})
		next, _ = next.Update(blinkMsg{}) // resting: the waits are set
		m = next.(Model)
		if m.introPending() {
			t.Fatal("the intro should be over")
		}
		if wait := m.againAt.Sub(*clock); wait < againGap[0] || wait > againGap[1] {
			t.Fatalf("the next intro should come 45 to 60 s after the rest, got %v", wait)
		}
		return m
	}
	m = played(m)
	for range 3 {
		before := m.introIndex
		*clock = m.againAt
		next, _ := m.Update(blinkMsg{})
		m = next.(Model)
		if !m.introPending() || m.introRun == nil {
			t.Fatal("an intro should play again when its time comes")
		}
		if m.introIndex == before {
			t.Errorf("%s played twice in a row", intro.Key(before))
		}
		sc, _ := m.startScreen()
		if v := m.View().Content; v == sc.Rest(false) || v == sc.Rest(true) {
			t.Error("the replay's frame should show, not the start screen")
		}
		if next, _ := m.Update(press("x")); next.(Model).splash {
			t.Error("a key during a replay should start")
		}
		m = played(m)
	}
}

// A window too small for the logo box has nothing to shine and no intro.
func TestSmallWindowWaitsQuietly(t *testing.T) {
	m, clock, _ := splashedWith(t, 60, 20, 3, config.Settings{})
	for range 3 {
		*clock = clock.Add(time.Minute)
		next, _ := m.Update(blinkMsg{})
		m = next.(Model)
	}
	if !m.shineAt.IsZero() || !m.shineStart.IsZero() || m.introPending() {
		t.Error("a small window should neither shine nor play an intro")
	}
}

// The start screen's layout and its two resting pictures are kept, and
// always match a fresh drawing: through blinks, a resize, and the scan
// changing the system check.
func TestStartScreenKept(t *testing.T) {
	m, _, _ := splashedWith(t, 160, 44, 3, config.Settings{Intro: introOff})
	fresh := func(m Model) string {
		t.Helper()
		c := m
		c.start = &startMemo{} // nothing kept: drawn from scratch
		return c.View().Content
	}
	var tm tea.Model = m
	for range 4 {
		tm, _ = tm.Update(blinkMsg{})
		if got := tm.(Model); got.View().Content != fresh(got) {
			t.Fatal("a kept resting picture should equal a fresh drawing")
		}
	}
	key := tm.(Model).start.key
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	if got := tm.(Model); got.View().Content != fresh(got) || got.start.key == key {
		t.Error("a resize should lay the start screen out again")
	}
	tm, _ = tm.Update(scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: bigLibrary(7).Games}})
	if got := tm.(Model); !strings.Contains(screen(got), "7 games") || got.View().Content != fresh(got) {
		t.Errorf("a new scan should show in the check:\n%s", screen(got))
	}
}

// While the start screen is up the list isn't laid out (it doesn't show);
// the key that leaves it lays the list out for the window as it is then,
// resized meanwhile or not.
func TestListLaidOutAfterStart(t *testing.T) {
	m, _, _ := splashedWith(t, 160, 44, 300, config.Settings{Intro: introOff})
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	tm, _ = tm.Update(press("x"))
	got := tm.(Model)
	if got.splash {
		t.Fatal("a key should start")
	}
	want := got
	want.layout()
	if got.list.Height() != want.list.Height() || got.list.Width() != want.list.Width() || got.list.Height() == 0 {
		t.Errorf("the list should be laid out for 100x30: %dx%d, want %dx%d", got.list.Width(), got.list.Height(), want.list.Width(), want.list.Height())
	}
	lines := strings.Split(screen(got), "\n")
	if len(lines) > 30 {
		t.Errorf("the screen is %d rows in a 30-row window", len(lines))
	}
	for _, l := range lines {
		if ansi.StringWidth(l) > 100 {
			t.Errorf("a line is wider than the window: %q", l)
		}
	}
}
