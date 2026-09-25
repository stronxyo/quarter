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

// The game's logo sits beside the quarter_ box, above the detail pane, in a
// band with no frame of its own, level with the inside of the box.
const logoBandMinCols = 24 // narrower than this, there's no band

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

// logoBand is where the selected game's logo goes: its left column, and its
// size in cells. ok is false when there's no room: no big quarter_ logo, no
// pane beside the list, or too narrow a strip beside the box.
func (m Model) logoBand() (x, cols, rows int, ok bool) {
	if !m.showLogo() || !m.sideBySide() {
		return 0, 0, 0, false
	}
	x = max(m.tableWidth(), logoIndent+logoBoxWidth) + paneGap
	cols = min(m.width-x, m.paneWidth())
	rows = len(logoArt) + logoBoxStyle.GetVerticalPadding() // the box's inside
	return x, cols, rows, cols >= logoBandMinCols
}

// showsLogos reports whether game logos are drawn at all right now.
func (m Model) showsLogos() bool {
	_, _, _, ok := m.logoBand()
	return m.logos != nil && !m.logosHidden && ok
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
	_, cols, rows, _ := m.logoBand()
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

// logoView fills the logo band, cols x rows: the logo in the QUARTER
// gradient, centered. Without one (none anywhere, or still on its way), the
// game's name in Quarter's pixel font, in the same style; nothing when even
// that doesn't fit, since the pane below names the game anyway.
func (m Model) logoView(g library.Game, cols, rows int) string {
	lines, state := m.logoFor(logoKey{g.AppID, cols, rows})
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
// level with the inside of its box.
func (m Model) withLogoBand(header string) string {
	x, cols, rows, ok := m.logoBand()
	g, selected := m.selected()
	if !ok || !selected || m.logos == nil || m.logosHidden {
		return header
	}
	above := logoPadTop + 1 // the box's top border
	band := strings.Repeat("\n", above) + m.logoView(g, cols, rows)
	gap := strings.Repeat(" ", max(x-lipgloss.Width(header), 0))
	return lipgloss.JoinHorizontal(lipgloss.Top, header, gap, band)
}
