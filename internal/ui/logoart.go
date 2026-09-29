package ui

import (
	"context"
	"errors"
	"image"
	"log/slog"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/stronxyo/quarter/internal/art"
	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/webapi"
)

// LogoSource finds game logos: files already on this PC, and downloads from
// Steam's CDN. *art.Sources is the real one.
type LogoSource interface {
	Paths(appID uint32) []string                                // logos on disk, best first
	Saved(appID uint32) bool                                    // Quarter downloaded this one already
	Missing(appID uint32) bool                                  // the CDN recently had none
	Download(ctx context.Context, appID uint32) (string, error) // fetch and keep the CDN's logo
}

// The game's logo sits beside the quarter_ box, above the detail pane, in a
// band with no frame of its own, as tall as the box, border to border.
const logoBandMinCols = 24 // narrower than this, there's no band

// logoKey is one drawing of one game's logo: each pane size draws anew.
type logoKey struct {
	appID      uint32
	cols, rows int
}

// drawnLogo is one drawn logo: text blocks, or on terminals that draw
// Sixel, a picture (see picture.go).
type drawnLogo struct {
	lines   []string
	picture string
}

// logoDoneMsg carries a drawn logo, or why there is none.
type logoDoneMsg struct {
	key      logoKey
	lines    []string
	picture  string
	px       image.Point // the picture's size in pixels, zero for text
	err      error
	download bool // none on disk: the CDN may have one
	fetched  bool // this answer came from a download
}

// logoWantMsg asks for a download once the selection has settled.
type logoWantMsg struct{ key logoKey }

// logoState is what the pane shows in the logo area.
type logoState int

const (
	logoName    logoState = iota // the game's name: no logo, or not yet
	logoReady                    // the drawn logo
	logoLoading                  // being read or downloaded
)

// logoBand is where the selected game's logo goes: its left column, and its
// size in cells: over the pane, pinned to the right edge. ok is false when
// there's no room: no big quarter_ logo, no pane beside the list, or the pane
// doesn't clear the quarter_ box.
func (m Model) logoBand() (x, cols, rows int, ok bool) {
	if !m.showLogo() || !m.sideBySide() {
		return 0, 0, 0, false
	}
	// Over the pane, as wide as it, when the quarter_ box fits to its left.
	x, cols = m.paneX(), m.paneWidth()
	rows = len(logoArt) + logoBoxStyle.GetVerticalFrameSize() // the box, border to border
	return x, cols, rows, x >= logoIndent+logoBoxWidth+paneGap && cols >= logoBandMinCols
}

// showsLogos reports whether game logos are drawn at all right now.
func (m Model) showsLogos() bool {
	_, _, _, ok := m.logoBand()
	return m.logos != nil && !m.logosHidden && !m.splash && ok
}

// logoFor reports what the pane can show for appID at the given area.
func (m Model) logoFor(key logoKey) (drawnLogo, logoState) {
	if logo, ok := m.logoCache[key]; ok {
		return logo, logoReady
	}
	if m.logoFetching == key.appID || m.logoPending[key] && !m.logoPause.active(m.now()) {
		return drawnLogo{}, logoLoading
	}
	return drawnLogo{}, logoName // none, or downloads are paused: the name meanwhile
}

// ensureLogo starts drawing the selected game's logo when the pane needs it
// and nothing is on the way yet.
func (m *Model) ensureLogo() tea.Cmd {
	g, ok := m.selected()
	if !ok || !m.showsLogos() || m.logoNone[g.AppID] {
		return nil
	}
	_, cols, rows, _ := m.logoBand()
	key := logoKey{g.AppID, cols, rows}
	if logo, ok := m.logoCache[key]; ok && (!m.sixel || logo.picture != "") || m.logoPending[key] {
		return nil
	}
	if m.sixel {
		// Pictures wait for the window to stop changing size, and only
		// the band's current size is kept: dragging the window's edge would
		// otherwise draw and keep a picture at every width it passes.
		if m.now().Sub(m.resizedAt) < m.debounce {
			return nil
		}
		for k, logo := range m.logoCache {
			if logo.picture != "" && (k.cols != cols || k.rows != rows) {
				delete(m.logoCache, k)
			}
		}
	}
	delete(m.logoCache, key) // drawn as text before pictures were known
	m.logoPending[key] = true
	return drawLogo(m.logos, key, m.logoBlocks, false, m.pictureSize(cols, rows))
}

// drawLogo reads the game's logo from disk (or, with download, from Steam's
// CDN first) and draws it, off the UI goroutine: as a pic-sized picture when
// pic is set, otherwise in text blocks.
func drawLogo(src LogoSource, key logoKey, blocks art.Blocks, download bool, pic image.Point) tea.Cmd {
	return func() tea.Msg {
		done := logoDoneMsg{key: key, px: pic, fetched: download}
		if download {
			if _, err := src.Download(context.Background(), key.appID); err != nil {
				done.err = err
				return done
			}
		}
		// Steam's copy first; if it can't be drawn, Quarter's download.
		var err error = art.ErrUnusable
		for _, p := range src.Paths(key.appID) {
			if pic.X > 0 {
				done.picture, err = art.RenderImageFile(p, pic.X, pic.Y, colorAccent)
			} else {
				done.lines, err = art.RenderFile(p, key.cols, key.rows, blocks)
			}
			if err == nil {
				return done
			}
		}
		done.err = err
		// Nothing drawable on disk: the CDN may have one, unless it was
		// already downloaded or is known to have none.
		done.download = !download && !src.Saved(key.appID) && !src.Missing(key.appID)
		return done
	}
}

// applyLogo stores a drawn logo, or remembers why there is none, and asks
// for a download when the disk had nothing.
func (m Model) applyLogo(msg logoDoneMsg) (Model, tea.Cmd) {
	delete(m.logoPending, msg.key)
	if msg.fetched {
		m.logoFetching = 0
	}
	switch {
	case msg.err == nil && msg.px != m.pictureSize(msg.key.cols, msg.key.rows):
		// Drawn for a cell size or a mode that has changed since: drop it,
		// and the next update draws it again.
	case msg.err == nil:
		m.logoCache[msg.key] = drawnLogo{msg.lines, msg.picture}
	case msg.download:
		m.logoPending[msg.key] = true // waiting for its turn on the CDN
		return m, m.logoWantAfter(msg.key)
	case msg.fetched && isNetFailure(msg.err):
		// Says nothing about the logo: downloads pause, the pane shows the
		// name meanwhile, and the logo is asked again when they resume.
		slog.Warn("logo", "app", msg.key.appID, "err", msg.err)
		m.logoPending[msg.key] = true
		cmd := m.failed(&m.logoPause, msg.err, true)
		if !errors.Is(msg.err, webapi.ErrUnreachable) {
			cmd = tea.Batch(cmd, m.answered())
		}
		return m, cmd
	default:
		// No logo anywhere, so the pane shows the name for this run.
		if msg.fetched && !errors.Is(msg.err, art.ErrNoLogo) && !errors.Is(msg.err, art.ErrUnusable) {
			slog.Warn("logo", "app", msg.key.appID, "err", msg.err)
		}
		m.logoNone[msg.key.appID] = true
	}
	if msg.fetched {
		// The CDN answered (a logo, or none): the network is up.
		m.logoPause = pause{}
		return m, m.answered()
	}
	return m, nil
}

func (m Model) logoWantAfter(key logoKey) tea.Cmd {
	return tea.Tick(m.debounce, func(time.Time) tea.Msg { return logoWantMsg{key} })
}

// wantLogo starts the download once the selection has settled on the game,
// one download at a time. Scrolled past, the game is dropped; selecting it
// again starts over.
func (m Model) wantLogo(msg logoWantMsg) (Model, tea.Cmd) {
	if !m.logoPending[msg.key] {
		return m, nil // drawn or dropped since this was asked
	}
	g, ok := m.selected()
	if !ok || g.AppID != msg.key.appID || m.logos == nil {
		delete(m.logoPending, msg.key)
		return m, nil
	}
	if m.logoFetching != 0 {
		return m, m.logoWantAfter(msg.key) // another download is out
	}
	if m.logoPause.active(m.now()) {
		return m, nil // downloads are paused; resuming asks again
	}
	m.logoFetching = msg.key.appID
	return m, drawLogo(m.logos, msg.key, m.logoBlocks, true, m.pictureSize(msg.key.cols, msg.key.rows))
}

// logoView fills the logo band, cols x rows: the logo in the QUARTER
// gradient, centered. Without one (none anywhere, or still on its way), the
// game's name in Quarter's pixel font, in the same style; nothing when even
// that doesn't fit, since the pane below names the game anyway. On a Sixel
// terminal the band stays blank for the picture.
func (m Model) logoView(g library.Game, cols, rows int) string {
	if m.sixel { // blank: the picture is drawn into it (picture.go)
		return lipgloss.Place(cols, rows, lipgloss.Left, lipgloss.Top, "")
	}
	logo, state := m.logoFor(logoKey{g.AppID, cols, rows})
	lines := logo.lines
	if state != logoReady {
		var ok bool
		if lines, ok = art.Name(g.Label(), cols, rows); !ok {
			return lipgloss.Place(cols, rows, lipgloss.Left, lipgloss.Top, "")
		}
	}
	colors := lipgloss.Blend1D(len(lines), colorLogoTop, colorLogoBottom)
	styled := make([]string, len(lines))
	for i, l := range lines {
		styled[i] = lipgloss.NewStyle().Foreground(colors[i]).Render(l)
	}
	return lipgloss.Place(cols, rows, lipgloss.Center, lipgloss.Center, strings.Join(styled, "\n"))
}

// withLogoBand puts the selected game's logo beside the quarter_ block,
// as tall as its box.
func (m Model) withLogoBand(header string) string {
	x, cols, rows, ok := m.logoBand()
	g, selected := m.selected()
	if !ok || !selected || m.logos == nil || m.logosHidden {
		return header
	}
	above := logoPadTop // level with the box's top border
	band := strings.Repeat("\n", above) + m.logoView(g, cols, rows)
	gap := strings.Repeat(" ", max(x-lipgloss.Width(header), 0))
	return lipgloss.JoinHorizontal(lipgloss.Top, header, gap, band)
}
