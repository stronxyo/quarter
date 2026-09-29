package ui

import (
	"context"
	"errors"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/art"
	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/webapi"
)

// fakeLogos serves logos from temporary files and counts what was asked.
type fakeLogos struct {
	paths     map[uint32][]string
	missing   map[uint32]bool
	cdn       map[uint32]string // what a download finds
	fail      error             // every download fails with this, when set
	downloads int
	reads     int
}

func (f *fakeLogos) Paths(id uint32) []string { f.reads++; return f.paths[id] }
func (f *fakeLogos) Saved(id uint32) bool     { return false }
func (f *fakeLogos) Missing(id uint32) bool   { return f.missing[id] }
func (f *fakeLogos) Download(_ context.Context, id uint32) (string, error) {
	f.downloads++
	if f.fail != nil {
		return "", f.fail
	}
	p, ok := f.cdn[id]
	if !ok {
		return "", art.ErrNoLogo
	}
	f.paths[id] = append(f.paths[id], p)
	return p, nil
}

// logoPNG writes a transparent PNG with an opaque ellipse: a usable logo.
func logoPNG(t *testing.T) string {
	t.Helper()
	img := image.NewNRGBA(image.Rect(0, 0, 320, 120))
	for y := range 120 {
		for x := range 320 {
			fx, fy := (float64(x)+0.5)/160-1, (float64(y)+0.5)/60-1
			if fx*fx+fy*fy <= 1 {
				img.SetNRGBA(x, y, color.NRGBA{255, 255, 255, 255})
			}
		}
	}
	p := filepath.Join(t.TempDir(), "logo.png")
	f, err := os.Create(p)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
	return p
}

// pump runs commands the way the program would and feeds their messages
// back, a few rounds deep. Tests using it set no clocks that wait.
func pump(m tea.Model, cmd tea.Cmd) tea.Model {
	queue := []tea.Cmd{cmd}
	for round := 0; round < 20 && len(queue) > 0; round++ {
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		switch msg := c().(type) {
		case tea.BatchMsg:
			queue = append(queue, msg...)
		case logoDoneMsg, logoWantMsg, resumeMsg:
			var next tea.Cmd
			m, next = m.Update(msg)
			queue = append(queue, next)
		}
	}
	return m
}

// withLogos returns a wide model with Alpha (a logo on disk) and Bravo.
func withLogos(t *testing.T, logos *fakeLogos, s config.Settings) (tea.Model, *[]config.Settings) {
	t.Helper()
	saved := &[]config.Settings{}
	m := New(Options{Settings: s, Logos: func(string) LogoSource { return logos },
		SaveSettings: func(s config.Settings) error { *saved = append(*saved, s); return nil }})
	m.debounce = 0
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 150, Height: 40})
	tm, cmd := tm.Update(scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: []library.Game{
		{AppID: 1, Name: "Alpha", Installed: true},
		{AppID: 2, Name: "Bravo", Installed: true},
	}}})
	return pump(tm, cmd), saved
}

func paneText(m tea.Model) string { return ansi.Strip(m.(Model).View().Content) }

// messages runs a command, flattening batches, and returns what it sent.
func messages(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	msg := cmd()
	if batch, ok := msg.(tea.BatchMsg); ok {
		var out []tea.Msg
		for _, c := range batch {
			out = append(out, messages(c)...)
		}
		return out
	}
	return []tea.Msg{msg}
}

// hasGameLogo reports whether a drawn game logo is on screen: its curved
// edges use quarter blocks (▖ to ▟), which nothing else in Quarter draws.
func hasGameLogo(view string) bool {
	return strings.ContainsFunc(view, func(r rune) bool { return r >= 0x2596 && r <= 0x259F })
}

func TestLogoShowsForSelectedGame(t *testing.T) {
	logos := &fakeLogos{paths: map[uint32][]string{1: {logoPNG(t)}}}
	m, _ := withLogos(t, logos, config.Settings{})
	view := paneText(m)
	if !hasGameLogo(view) || !strings.Contains(view, "ALPHA") {
		t.Errorf("Alpha's logo should show, and its name in the pane:\n%s", view)
	}
	// The logo sits beside the quarter_ box: on a line of the box, right of it.
	for _, l := range strings.Split(view, "\n") {
		if hasGameLogo(l) && !strings.ContainsAny(l, "│╭╰") {
			t.Errorf("the logo should share its lines with the quarter_ box: %q", l)
		}
	}
}

// Offline, a download fails without marking the game as having no logo:
// the name shows meanwhile, downloads pause, and when they resume with the
// network back, the logo comes.
func TestLogoOfflineThenBack(t *testing.T) {
	logos := &fakeLogos{paths: map[uint32][]string{1: {logoPNG(t)}}, cdn: map[uint32]string{2: logoPNG(t)},
		fail: webapi.Unreachable("logo 2", errors.New("dial tcp: no route"))}
	m, _ := withLogos(t, logos, config.Settings{})
	mm := m.(Model)
	clock := mm.now()
	mm.now = func() time.Time { return clock }
	mm.jitter = func(time.Duration) time.Duration { return time.Millisecond }
	m, cmd := tea.Model(mm).Update(press("j")) // Bravo: not on disk, downloaded
	m = pump(m, cmd)

	mm = m.(Model)
	if logos.downloads != 1 || mm.logoNone[2] {
		t.Fatalf("one download, and no 'no logo' mark: downloads %d, none %v", logos.downloads, mm.logoNone[2])
	}
	if !mm.logoPause.active(mm.now()) || !mm.netDown {
		t.Error("downloads should pause, and offline show")
	}
	_, cols, rows, _ := mm.logoBand()
	drawn, _ := art.Name("Bravo", cols, rows)
	view := paneText(m)
	if !strings.Contains(view, strings.TrimSpace(drawn[0])) || !strings.Contains(view, "· offline") {
		t.Errorf("the name should show while offline, and the marker:\n%s", view)
	}

	logos.fail = nil
	m = later(m, time.Hour)
	m = pump(m, func() tea.Msg { return resumeMsg{logos: true} })
	if logos.downloads != 2 || !hasGameLogo(paneText(m)) || strings.Contains(paneText(m), "offline") {
		t.Errorf("back online: want a second download and Bravo's logo, downloads %d:\n%s", logos.downloads, paneText(m))
	}
}

// Bravo has no logo anywhere: its name shows in the pixel font where the
// logo would be, and plainly in the pane as always.
func TestNoLogoShowsPixelName(t *testing.T) {
	logos := &fakeLogos{paths: map[uint32][]string{1: {logoPNG(t)}}, missing: map[uint32]bool{2: true}}
	m, _ := withLogos(t, logos, config.Settings{})
	m, cmd := m.Update(press("j"))
	m = pump(m, cmd)
	_, cols, rows, _ := m.(Model).logoBand()
	drawn, _ := art.Name("Bravo", cols, rows)
	if !strings.Contains(paneText(m), strings.TrimSpace(drawn[0])) || !strings.Contains(paneText(m), "BRAVO") {
		t.Fatalf("Bravo's name should show in the pixel font and in the pane:\n%s", paneText(m))
	}
	if !m.(Model).logoNone[2] || logos.downloads != 0 {
		t.Error("a game the CDN is known to lack should not be downloaded")
	}
}

// Every character on screen comes from ranges that common monospace fonts
// (JetBrains Mono, Cascadia Mono) have: a character outside them shows as
// "?" in those fonts. The favourite stars are the one known exception: no
// monospace font has them, and terminals borrow them from a symbol font.
func TestScreenUsesCommonCharacters(t *testing.T) {
	logos := &fakeLogos{paths: map[uint32][]string{1: {logoPNG(t)}}}
	m, _ := withLogos(t, logos, config.Settings{Favourites: []uint32{2}})
	common := func(r rune) bool {
		switch {
		case r < 0x250: // Latin letters, digits, punctuation
			return true
		case r >= 0x2190 && r <= 0x21FF: // arrows, for the key help
			return true
		case r >= 0x2500 && r <= 0x25FF: // box lines, blocks, triangles
			return true
		}
		return r == '·' || r == '…' || string(r) == markerFavourite || string(r) == markerFavouriteUninstalled
	}
	for _, r := range paneText(m) {
		if !common(r) {
			t.Errorf("%q (%U) is missing from common monospace fonts", r, r)
		}
	}
	if !hasGameLogo(paneText(m)) {
		t.Fatal("the check needs a logo on screen")
	}
}

// Without room beside the quarter_ box, no logo is drawn or read.
func TestNoLogoBandWhenNarrow(t *testing.T) {
	logos := &fakeLogos{paths: map[uint32][]string{1: {logoPNG(t)}}}
	m, _ := withLogos(t, logos, config.Settings{})
	logos.reads = 0
	m, cmd := m.Update(tea.WindowSizeMsg{Width: 100, Height: 40})
	m = pump(m, cmd)
	if hasGameLogo(paneText(m)) || logos.reads != 0 {
		t.Errorf("a 100-column window has no room beside the box: reads %d", logos.reads)
	}
}

// A drawing that arrives after the selection moved on is kept for later
// but never shown on another game.
func TestLateLogoStaysWithItsGame(t *testing.T) {
	logos := &fakeLogos{paths: map[uint32][]string{}, missing: map[uint32]bool{1: true, 2: true}}
	m, _ := withLogos(t, logos, config.Settings{})
	m, _ = m.Update(press("j")) // Bravo
	_, cols, rows, _ := m.(Model).logoBand()
	m, _ = m.Update(logoDoneMsg{key: logoKey{1, cols, rows}, lines: []string{"▞▚▞▚"}})
	if strings.Contains(paneText(m), "▞▚▞▚") {
		t.Error("Alpha's logo must not show on Bravo")
	}
	m, _ = m.Update(press("k")) // back to Alpha
	if !strings.Contains(paneText(m), "▞▚▞▚") {
		t.Error("Alpha's logo should show when Alpha is selected again")
	}
}

// With nothing on disk, the logo is downloaded once the selection settles.
func TestLogoDownload(t *testing.T) {
	logos := &fakeLogos{paths: map[uint32][]string{}, cdn: map[uint32]string{1: logoPNG(t)}}
	m, _ := withLogos(t, logos, config.Settings{})
	if logos.downloads != 1 || !hasGameLogo(paneText(m)) {
		t.Errorf("the logo should be downloaded once and shown, downloads %d:\n%s", logos.downloads, paneText(m))
	}
}

// Scrolling past a game before its download starts downloads nothing.
func TestLogoDownloadSkippedWhenScrolledPast(t *testing.T) {
	logos := &fakeLogos{paths: map[uint32][]string{}, cdn: map[uint32]string{1: logoPNG(t)}, missing: map[uint32]bool{2: true}}
	m := New(Options{Logos: func(string) LogoSource { return logos }})
	m.debounce = 0
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 150, Height: 40})
	tm, cmd := tm.Update(scanDoneMsg{res: library.Result{Games: []library.Game{
		{AppID: 1, Name: "Alpha", Installed: true}, {AppID: 2, Name: "Bravo", Installed: true},
	}}})
	// Find Alpha's "nothing on disk" answer, then move on before its turn.
	var want logoDoneMsg
	for _, msg := range messages(cmd) {
		if msg, ok := msg.(logoDoneMsg); ok && msg.download {
			want = msg
		}
	}
	if want.key.appID != 1 {
		t.Fatal("Alpha's logo should be looked for on disk first")
	}
	tm, next := tm.Update(want)
	tm, _ = tm.Update(press("j"))
	pump(tm, next)
	if logos.downloads != 0 {
		t.Errorf("Alpha was scrolled past, but %d downloads ran", logos.downloads)
	}
}

// o hides logos and remembers it; hidden, nothing is read.
func TestLogosToggle(t *testing.T) {
	logos := &fakeLogos{paths: map[uint32][]string{1: {logoPNG(t)}}}
	m, saved := withLogos(t, logos, config.Settings{})
	m, cmd := m.Update(press("o"))
	run(cmd)
	if !strings.Contains(paneText(m), "ALPHA") || hasGameLogo(paneText(m)) {
		t.Errorf("with logos hidden, the pane should show the name:\n%s", paneText(m))
	}
	if len(*saved) != 1 || !(*saved)[0].HideLogos {
		t.Fatalf("hiding logos should be saved, got %+v", *saved)
	}

	logos = &fakeLogos{paths: map[uint32][]string{1: {logoPNG(t)}}}
	withLogos(t, logos, config.Settings{HideLogos: true})
	if logos.reads != 0 {
		t.Errorf("with logos hidden, no logo should be read, got %d reads", logos.reads)
	}
}
