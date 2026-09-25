package ui

import (
	"context"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/art"
	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/library"
)

// fakeLogos serves logos from temporary files and counts what was asked.
type fakeLogos struct {
	paths     map[uint32][]string
	missing   map[uint32]bool
	cdn       map[uint32]string // what a download finds
	downloads int
	reads     int
}

func (f *fakeLogos) Paths(id uint32) []string { f.reads++; return f.paths[id] }
func (f *fakeLogos) Saved(id uint32) bool     { return false }
func (f *fakeLogos) Missing(id uint32) bool   { return f.missing[id] }
func (f *fakeLogos) Download(_ context.Context, id uint32) (string, error) {
	f.downloads++
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
		case logoDoneMsg, logoWantMsg:
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
// edges use octant characters, which nothing else in Quarter draws.
func hasGameLogo(view string) bool {
	return strings.ContainsFunc(view, func(r rune) bool { return r >= 0x1CD00 && r <= 0x1CDE5 })
}

func TestLogoShowsForSelectedGame(t *testing.T) {
	logos := &fakeLogos{paths: map[uint32][]string{1: {logoPNG(t)}}}
	m, _ := withLogos(t, logos, config.Settings{})
	view := paneText(m)
	if !hasGameLogo(view) || strings.Contains(view, "ALPHA") {
		t.Errorf("Alpha's pane should show its logo, not its name:\n%s", view)
	}
}

// Bravo has no logo anywhere: its name shows, in the same area, so the
// details below sit on the same line as Alpha's.
func TestLogoAreaKeepsItsHeight(t *testing.T) {
	logos := &fakeLogos{paths: map[uint32][]string{1: {logoPNG(t)}}, missing: map[uint32]bool{2: true}}
	m, _ := withLogos(t, logos, config.Settings{})
	row := func(m tea.Model) int {
		for i, l := range strings.Split(paneText(m), "\n") {
			if strings.Contains(l, "Last played  ") {
				return i
			}
		}
		return -1
	}
	alpha := row(m)
	m, cmd := m.Update(press("j"))
	m = pump(m, cmd)
	cols, rows := logoArea(m.(Model).paneSize())
	drawn, _ := art.Name("Bravo", cols, rows)
	if !strings.Contains(paneText(m), strings.TrimSpace(drawn[0])) || strings.Contains(paneText(m), "BRAVO") {
		t.Fatalf("Bravo has no logo, so its name should show in the pixel font:\n%s", paneText(m))
	}
	if bravo := row(m); alpha < 0 || bravo != alpha {
		t.Errorf("the details moved: row %d with a logo, %d without", alpha, bravo)
	}
	if !m.(Model).logoNone[2] || logos.downloads != 0 {
		t.Error("a game the CDN is known to lack should not be downloaded")
	}
}

// A drawing that arrives after the selection moved on is kept for later
// but never shown on another game.
func TestLateLogoStaysWithItsGame(t *testing.T) {
	logos := &fakeLogos{paths: map[uint32][]string{}, missing: map[uint32]bool{1: true, 2: true}}
	m, _ := withLogos(t, logos, config.Settings{})
	m, _ = m.Update(press("j")) // Bravo
	cols, rows := logoArea(m.(Model).paneSize())
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
