package ui

// Logos as real pictures (Sixel), on terminals that draw them (Windows
// Terminal 1.22 and later, and others). The screen keeps the logo band blank;
// the picture is written into that blank area directly, a moment after the
// screen has drawn it, and erased before the next one. Terminals without
// Sixel keep the text-block logos.

import (
	"image"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/art"
)

// A cell's size in pixels when the terminal doesn't say: small enough that
// the picture never spills out of the band.
const (
	fallbackCellW = 8
	fallbackCellH = 16
	pictureDelay  = 60 * time.Millisecond // after the screen has drawn the blank band
)

// askPictures asks the terminal whether it draws Sixel (device attribute 4)
// and how big a cell is in pixels.
func askPictures() tea.Cmd {
	return tea.Sequence(tea.Raw(ansi.RequestPrimaryDeviceAttributes), tea.Raw(askCellSize))
}

// askCellSize asks the terminal for a cell's size in pixels (XTWINOPS 16).
const askCellSize = "\x1b[16t"

// sizeSettledMsg comes a moment after a window size change, to draw the
// pictures that waited for it.
type sizeSettledMsg struct{}

// resized notes a window size change. A zoom changes a cell's size in pixels
// too, so the terminal is asked again.
func (m *Model) resized() tea.Cmd {
	m.resizedAt = m.now()
	if !m.sixel {
		return nil
	}
	return tea.Batch(tea.Raw(askCellSize), tea.Tick(m.debounce, func(time.Time) tea.Msg { return sizeSettledMsg{} }))
}

// sixelSig is what the band shows as a picture: which game, where, and at
// what size; the zero value is none.
type sixelSig struct {
	appID         uint32
	x, cols, rows int
	w, h          int // the screen, whose redraws erase pictures
	ready         bool
}

// pictureMsg draws the picture for sig, if it is still the one wanted.
type pictureMsg struct{ sig sixelSig }

func (m Model) picturesMsg(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case uv.PrimaryDeviceAttributesEvent:
		if slices.Contains(msg, 4) && !m.sixel {
			m.sixel = true
			clear(m.logoCache) // drawn as text so far
		}
	case uv.CellSizeEvent:
		if msg.Width > 0 && msg.Height > 0 && (msg.Width != m.cellW || msg.Height != m.cellH) {
			m.cellW, m.cellH = msg.Width, msg.Height
			if m.sixel {
				clear(m.logoCache)
			}
		}
	case pictureMsg:
		if msg.sig != m.pictureWant {
			return m, nil
		}
		return m.drawPicture()
	}
	return m, nil
}

// cellSize is a cell in pixels.
func (m Model) cellSize() (int, int) {
	if m.cellW > 0 && m.cellH > 0 {
		return m.cellW, m.cellH
	}
	return fallbackCellW, fallbackCellH
}

// pictureSize is the band in pixels, or zero when logos are text.
func (m Model) pictureSize(cols, rows int) image.Point {
	if !m.sixel {
		return image.Point{}
	}
	cw, ch := m.cellSize()
	return image.Pt(cols*cw, rows*ch)
}

// pictureSig is the picture the band should show now.
func (m Model) pictureSig() sixelSig {
	if !m.sixel || !m.showsLogos() {
		return sixelSig{}
	}
	g, ok := m.selected()
	if !ok {
		return sixelSig{}
	}
	x, cols, rows, _ := m.logoBand()
	logo, state := m.logoFor(logoKey{g.AppID, cols, rows})
	ready := state == logoReady && logo.picture != "" || state == logoName && (m.logoNone[g.AppID] || m.logoPause.active(m.now()))
	return sixelSig{g.AppID, x, cols, rows, m.width, m.height, ready}
}

// wantPicture schedules a redraw of the band's picture when what it should
// show has changed. While a key is held the band shows the dots: the last
// picture is erased at once.
func (m *Model) wantPicture() tea.Cmd {
	if !m.sixel {
		return nil
	}
	sig := m.pictureSig()
	if m.holding() {
		sig = sixelSig{}
	}
	if sig == m.pictureWant {
		return nil
	}
	m.pictureWant = sig
	if sig == (sixelSig{}) && m.pictureShown.cols > 0 {
		drawn, erase := m.drawPicture()
		m.pictureShown = drawn.pictureShown
		return erase
	}
	return tea.Tick(pictureDelay, func(time.Time) tea.Msg { return pictureMsg{sig} })
}

// drawPicture erases the last picture and draws the wanted one.
func (m Model) drawPicture() (Model, tea.Cmd) {
	var b strings.Builder
	b.WriteString(ansi.SaveCursor)
	if old := m.pictureShown; old.cols > 0 && old.w == m.width && old.h == m.height {
		for r := range old.rows {
			b.WriteString(ansi.CursorPosition(old.x+1, logoPadTop+r+1))
			b.WriteString(ansi.EraseCharacter(old.cols))
		}
	}
	sig := m.pictureWant
	if sig.ready {
		if pic := m.picture(sig); pic != "" {
			b.WriteString(ansi.CursorPosition(sig.x+1, logoPadTop+1))
			b.WriteString(pic)
		}
	}
	b.WriteString(ansi.RestoreCursor)
	m.pictureShown = sig
	return m, tea.Raw(b.String())
}

// picture is the Sixel for sig: the drawn logo, or the game's name in the
// pixel font when it has none.
func (m Model) picture(sig sixelSig) string {
	logo, state := m.logoFor(logoKey{sig.appID, sig.cols, sig.rows})
	if state == logoReady {
		return logo.picture
	}
	g, ok := m.selected()
	if !ok {
		return ""
	}
	lines, ok := art.Name(g.Label(), sig.cols, sig.rows)
	if !ok {
		return ""
	}
	cw, ch := m.cellSize()
	return art.LinesImage(lines, sig.cols*cw, sig.rows*ch, colorLogoTop, colorLogoBottom)
}
