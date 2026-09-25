package ui

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"

	"github.com/stronxyo/quarter/internal/art"
	"github.com/stronxyo/quarter/internal/library"
)

// LogoSource finds game logos: files already on this PC, and downloads from
// Steam's CDN. *art.Sources is the real one.
type LogoSource interface {
	Paths(appID uint32) []string                                // logos on disk, best first
	Saved(appID uint32) bool                                    // Quarter downloaded this one already
	Missing(appID uint32) bool                                  // the CDN recently had none
	Download(ctx context.Context, appID uint32) (string, error) // fetch and keep the CDN's logo
}

// Game logos in the detail pane: at most logoMaxRows tall, and never more
// than logoShare of the pane, so the details below stay visible.
const (
	logoMaxRows = 14
	logoShare   = 0.4
	logoMinRows = 3 // shorter than this, the pane shows the name instead
)

// logoKey is one drawing of one game's logo: each pane size draws anew.
type logoKey struct {
	appID      uint32
	cols, rows int
}

// logoDoneMsg carries a drawn logo, or why there is none.
type logoDoneMsg struct {
	key      logoKey
	lines    []string
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

// logoArea is the logo's size inside a pane of width x height cells, or 0
// rows when there's no room for one.
func logoArea(width, height int) (cols, rows int) {
	cols = width - paneStyle.GetHorizontalFrameSize()
	rows = min(logoMaxRows, int(float64(height-paneStyle.GetVerticalFrameSize())*logoShare))
	if rows < logoMinRows || cols < 1 {
		return cols, 0
	}
	return cols, rows
}

// paneSize is the detail pane's size as body draws it: beside the list, or
// over it on a narrow window.
func (m Model) paneSize() (width, height int) {
	height = 2 + m.list.Height() // section line and column header, then the rows
	if !m.sideBySide() {
		height--
	}
	return m.paneWidth(), height
}

// showsLogos reports whether the pane draws logos at all.
func (m Model) showsLogos() bool {
	return m.logos != nil && !m.logosHidden && (m.sideBySide() || m.detailOpen)
}

// logoFor reports what the pane can show for appID at the given area.
func (m Model) logoFor(key logoKey) ([]string, logoState) {
	if lines, ok := m.logoCache[key]; ok {
		return lines, logoReady
	}
	if m.logoPending[key] || m.logoFetching == key.appID {
		return nil, logoLoading
	}
	return nil, logoName
}

// ensureLogo starts drawing the selected game's logo when the pane needs it
// and nothing is on the way yet.
func (m *Model) ensureLogo() tea.Cmd {
	g, ok := m.selected()
	if !ok || !m.showsLogos() || m.logoNone[g.AppID] {
		return nil
	}
	cols, rows := logoArea(m.paneSize())
	if rows == 0 {
		return nil
	}
	key := logoKey{g.AppID, cols, rows}
	if _, ok := m.logoCache[key]; ok || m.logoPending[key] {
		return nil
	}
	m.logoPending[key] = true
	return drawLogo(m.logos, key, m.logoBlocks, false)
}

// drawLogo reads the game's logo from disk (or, with download, from Steam's
// CDN first) and draws it, off the UI goroutine.
func drawLogo(src LogoSource, key logoKey, blocks art.Blocks, download bool) tea.Cmd {
	return func() tea.Msg {
		done := logoDoneMsg{key: key, fetched: download}
		if download {
			if _, err := src.Download(context.Background(), key.appID); err != nil {
				done.err = err
				return done
			}
		}
		// Steam's copy first; if it can't be drawn, Quarter's download.
		var err error = art.ErrUnusable
		for _, p := range src.Paths(key.appID) {
			if done.lines, err = art.RenderFile(p, key.cols, key.rows, blocks); err == nil {
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
	case msg.err == nil:
		m.logoCache[msg.key] = msg.lines
	case msg.download:
		m.logoPending[msg.key] = true // waiting for its turn on the CDN
		return m, m.logoWantAfter(msg.key)
	default:
		// No logo anywhere, so the pane shows the name for this run. A
		// network problem isn't remembered on disk: the next run asks again.
		if msg.fetched && !errors.Is(msg.err, art.ErrNoLogo) && !errors.Is(msg.err, art.ErrUnusable) {
			slog.Warn("logo", "app", msg.key.appID, "err", msg.err)
		}
		m.logoNone[msg.key.appID] = true
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
	g, ok := m.selected()
	if !ok || g.AppID != msg.key.appID || m.logos == nil {
		delete(m.logoPending, msg.key)
		return m, nil
	}
	if m.logoFetching != 0 {
		return m, m.logoWantAfter(msg.key) // another download is out
	}
	m.logoFetching = msg.key.appID
	return m, drawLogo(m.logos, msg.key, m.logoBlocks, true)
}

// logoView fills the pane's logo area, cols x rows: the logo in the QUARTER
// gradient, centered, or the game's name while there's none.
func (m Model) logoView(g library.Game, cols, rows int) string {
	lines, state := m.logoFor(logoKey{g.AppID, cols, rows})
	if state != logoReady {
		name := paneTitleStyle.Width(cols).Render(strings.ToUpper(g.Label()))
		return lipgloss.Place(cols, rows, lipgloss.Left, lipgloss.Center, name)
	}
	colors := lipgloss.Blend1D(len(lines), colorLogoTop, colorLogoBottom)
	styled := make([]string, len(lines))
	for i, l := range lines {
		styled[i] = lipgloss.NewStyle().Foreground(colors[i]).Render(l)
	}
	return lipgloss.Place(cols, rows, lipgloss.Center, lipgloss.Center, strings.Join(styled, "\n"))
}
