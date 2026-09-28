package ui

import (
	"fmt"
	"image"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"

	"github.com/stronxyo/quarter/internal/config"
)

// sixelModel is the wide test model on a terminal that draws Sixel, with
// cells of 10 x 20 pixels.
func sixelModel(t *testing.T) tea.Model {
	t.Helper()
	logos := &fakeLogos{paths: map[uint32][]string{1: {logoPNG(t)}}}
	m, _ := withLogos(t, logos, config.Settings{})
	m, cmd := m.Update(uv.PrimaryDeviceAttributesEvent{62, 4, 22})
	m = pump(m, cmd)
	m, cmd = m.Update(uv.CellSizeEvent{Width: 10, Height: 20})
	return pump(m, cmd)
}

// The picture drawn for the band, as sent to the terminal.
func drawnPicture(t *testing.T, m tea.Model) (sixelSig, string) {
	t.Helper()
	sig := m.(Model).pictureWant
	if !sig.ready {
		t.Fatalf("the picture should be ready: %+v", sig)
	}
	_, cmd := m.Update(pictureMsg{sig})
	raw, ok := cmd().(tea.RawMsg)
	if !ok {
		t.Fatal("drawing the picture should send it raw")
	}
	return sig, fmt.Sprint(raw.Msg)
}

// On a Sixel terminal the band stays blank and the logo is sent as a
// picture at the band's top-left cell.
func TestLogoAsPicture(t *testing.T) {
	m := sixelModel(t)
	if hasGameLogo(paneText(m)) {
		t.Error("the band should be blank: the picture is drawn over it")
	}
	sig, s := drawnPicture(t, m)
	at := fmt.Sprintf("\x1b[%d;%dH\x1bP", logoPadTop+1, sig.x+1)
	if !strings.Contains(s, at) || !strings.HasSuffix(s, "\x1b\\\x1b8") {
		t.Errorf("the picture should start at the band and end the Sixel: %q...", s[:min(len(s), 80)])
	}
	if want := fmt.Sprintf("\"1;1;%d;%d", sig.cols*10, sig.rows*20); !strings.Contains(s, want) {
		t.Errorf("the picture should fill the band in pixels, want %s", want)
	}
}

// A zoom arrives as a new window size: the cell size is asked again, and the
// picture follows the new one. Pictures of the old band size are dropped.
func TestPictureFollowsZoom(t *testing.T) {
	m := sixelModel(t)
	old := m.(Model).pictureWant

	m, cmd := m.Update(tea.WindowSizeMsg{Width: 200, Height: 50})
	asked := false
	var later []tea.Msg // what the resize started, answered after the zoom
	for _, msg := range messages(cmd) {
		if raw, ok := msg.(tea.RawMsg); ok && fmt.Sprint(raw.Msg) == askCellSize {
			asked = true
		} else {
			later = append(later, msg)
		}
	}
	if !asked {
		t.Error("a new window size should ask the cell size again")
	}
	m, cmd = m.Update(uv.CellSizeEvent{Width: 7, Height: 14})
	m = pump(m, cmd)
	for _, msg := range later { // a picture drawn at the old cell size comes back
		m, cmd = m.Update(msg)
		m = pump(m, cmd)
	}
	sig, s := drawnPicture(t, m)
	if want := fmt.Sprintf("\"1;1;%d;%d", sig.cols*7, sig.rows*14); !strings.Contains(s, want) {
		t.Errorf("the picture should follow the new cell size, want %s", want)
	}
	for k, logo := range m.(Model).logoCache {
		if logo.picture != "" && k.cols == old.cols && k.cols != sig.cols {
			t.Error("the picture for the old band size should be dropped")
		}
	}
}

// A picture drawn for a cell size that changed while it was drawn is not
// shown; it is drawn again.
func TestStalePictureDropped(t *testing.T) {
	m := sixelModel(t).(Model)
	_, cols, rows, _ := m.logoBand()
	key := logoKey{1, cols, rows}
	delete(m.logoCache, key)
	next, _ := m.applyLogo(logoDoneMsg{key: key, picture: "old", px: image.Pt(cols*8, rows*16)})
	if logo := next.logoCache[key]; logo.picture == "old" {
		t.Error("a picture of the old cell size should not be kept")
	}
}

// While the window is still changing size, pictures wait for it to settle.
func TestPictureWaitsForResize(t *testing.T) {
	m := sixelModel(t).(Model)
	m.debounce = time.Hour
	m.logoCache = map[logoKey]drawnLogo{}
	m.width = 190
	m.resizedAt = m.now()
	if cmd := m.ensureLogo(); cmd != nil {
		t.Error("no picture should be drawn while the size is changing")
	}
}
