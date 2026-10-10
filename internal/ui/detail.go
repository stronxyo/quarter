package ui

import (
	"fmt"
	"net/url"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/library"
)

var (
	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(colorLogoBorder).
			Padding(0, 1)
	paneTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
)

// paneLabelWidth fits the longest label, "Achievements", plus a space.
const paneLabelWidth = 13

// The achievement bar: a heavy line for unlocked, a light line for the rest.
// Both are box drawing, which every monospace font has (the middle-third
// block first used here, U+1FB0B, is missing from most).
const (
	barCells      = 14
	barCountWidth = 8 // fits "999/999 "
	barFilled     = "━"
	barEmpty      = "─"
)

// roomForPane reports whether the window has room for the list and the detail
// pane next to each other. There, d shows or hides the pane; otherwise it
// opens the pane as a pop-up.
func (m Model) roomForPane() bool {
	return m.width >= minTableWidth+paneGap+paneMinWidth
}

// sideBySide reports whether the pane is showing next to the list.
func (m Model) sideBySide() bool {
	return m.roomForPane() && !m.paneHidden
}

// tableWidth is the list's width: up to maxTableWidth, less whatever the
// detail pane needs beside it at its narrowest.
func (m Model) tableWidth() int {
	if m.sideBySide() {
		return min(m.width-paneGap-paneMinWidth, maxTableWidth)
	}
	return min(m.width, maxTableWidth)
}

// paneWidth is the pane's width. Beside the list, the pane and the logo above
// it fill the rest of the window, out to its right edge; as a pop-up on a
// narrow window, up to paneMaxWidth.
func (m Model) paneWidth() int {
	if m.sideBySide() {
		return m.width - paneGap - m.tableWidth()
	}
	return min(m.width, paneMaxWidth)
}

// paneX is the column the right block (logo band and pane) starts at, so it
// ends at the window's right edge.
func (m Model) paneX() int {
	return m.width - m.paneWidth()
}

// detailView draws the selected game's pane, width by height cells. On a
// wide pane its contents stay a column at most paneTextWidth wide, on the
// left, so lines stay readable.
func (m Model) detailView(width, height int) string {
	inner := min(width-paneStyle.GetHorizontalFrameSize(), paneTextWidth)
	var b strings.Builder

	g, ok := m.selected()
	if !ok {
		b.WriteString(dimStyle.Render("No game selected."))
	} else {
		// The name always shows here; the logo has its own place, beside
		// the quarter_ box.
		b.WriteString(paneTitleStyle.Width(inner).Render(strings.ToUpper(g.Label())) + "\n")

		switch {
		case g.AppID == m.running:
			b.WriteString(runningLabelStyle.Render(markerRunning+" RUNNING NOW") + "\n")
		case g.Updating:
			b.WriteString(strings.Join(m.dlView().lines(g, inner), "\n") + "\n")
		}

		info, state := m.storeStatus(g.AppID)
		switch state {
		case storeReady:
			var who []string
			if len(info.Developers) > 0 {
				who = append(who, info.Developers[0])
			}
			if y := info.Year(); y != "" {
				who = append(who, y)
			}
			if len(who) > 0 {
				b.WriteString(textStyle.Render(ansi.Truncate(strings.Join(who, " · "), inner, "…")) + "\n")
			}
			if len(info.Genres) > 0 {
				b.WriteString(dimStyle.Render(ansi.Truncate(strings.Join(info.Genres, ", "), inner, "…")) + "\n")
			}
		case storeLoading:
			b.WriteString(dimStyle.Render("loading store info…") + "\n")
		case storeUnknown:
			if m.fetchStore != nil {
				b.WriteString(dimStyle.Render("loading store info…") + "\n")
			}
		case storeNotFound:
			b.WriteString(dimStyle.Render("not on the Steam store") + "\n")
		case storeOffline:
			msg := "store info unavailable, retrying"
			if m.storePause.busy {
				msg = "store busy, retrying"
			}
			b.WriteString(alertStyle.Render(msg) + "\n")
		}
		b.WriteString("\n")

		row := func(label, value string) {
			b.WriteString(dimStyle.Render(fmt.Sprintf("%-*s", paneLabelWidth, label)))
			b.WriteString(textStyle.Render(ansi.Truncate(value, max(inner-paneLabelWidth, 1), "…")))
			b.WriteString("\n")
		}
		row("Played", library.FormatHours(g.Playtime, m.playtimeKnown))
		row("Last played", library.FormatAgo(g.LastPlayed, m.now()))
		// Progress comes from Steam's local cache; when that is empty, the
		// store still knows how many the game has, and none are unlocked.
		if a := g.Achievements; a.Total > 0 {
			row("Achievements", achievementBar(a.Achieved, a.Total))
		} else if info, state := m.storeStatus(g.AppID); state == storeReady && info.Achievements > 0 {
			row("Achievements", achievementBar(0, info.Achievements))
		}
		if g.Installed {
			row("Size", library.FormatSize(g.SizeOnDisk))
			row("Library", g.LibraryPath)
		} else {
			row("Status", "not installed")
		}
		// Who publishes it and its official site, a block of their own
		// (Pablo, 2026-10-02). The site's name ends in an arrow, always
		// shown even when the name is cut: w or a click opens it (mouse.go).
		// No terminal link (OSC 8): Windows Terminal underlines those with
		// dots, and Quarter's own click works in every terminal.
		if state == storeReady && (len(info.Publishers) > 0 || info.Website != "") {
			b.WriteString("\n")
			if len(info.Publishers) > 0 {
				row("Publisher", info.Publishers[0])
			}
			if info.Website != "" {
				name := ansi.Truncate(siteName(info.Website), max(inner-paneLabelWidth-2, 1), "…")
				row("Website", name+" "+markerLink)
			}
		}
		if g.Favourite {
			b.WriteString("\n" + starStyle.Render(markerFavourite+" Favourite") + "\n")
		}
		if state == storeReady && info.Description != "" {
			b.WriteString("\n" + textStyle.Width(inner).Render(info.Description))
		}
	}
	// A long description is cut to fit inside the frame, so the bottom
	// border always shows.
	lines := strings.Split(strings.TrimRight(b.String(), "\n"), "\n")
	if room := max(height-paneStyle.GetVerticalFrameSize(), 0); len(lines) > room {
		lines = lines[:room]
	}
	return paneStyle.Width(width).Height(height).MaxHeight(height).Render(strings.Join(lines, "\n"))
}

// selected returns the game on show: the list's highlighted game, or with
// history open the ranking's, so the logo and the pane follow it.
func (m Model) selected() (library.Game, bool) {
	if m.histView != histOff {
		return m.rankedGame()
	}
	it, ok := m.list.SelectedItem().(gameItem)
	return it.Game, ok
}

// siteName is a website as the pane shows it: no scheme, no "www.", no
// trailing slash ("http://www.rocketleague.com/" is "rocketleague.com").
func siteName(site string) string {
	u, err := url.Parse(site)
	if err != nil || u.Host == "" {
		return site
	}
	return strings.TrimPrefix(u.Host, "www.") + strings.TrimRight(u.Path, "/")
}

// achievementBar draws the count, then a bar: a thicker orange band for what's
// unlocked and a thin dim line for the rest, e.g. "72/88   ━━━━━━━━━━━───".
// The count is padded so bars line up from game to game.
func achievementBar(achieved, total int) string {
	filled := (achieved*barCells + total/2) / total // rounded
	if achieved > 0 && filled == 0 {
		filled = 1 // any progress shows
	}
	if achieved < total && filled == barCells {
		filled = barCells - 1 // not full until it is
	}
	count := fmt.Sprintf("%-*s", barCountWidth, fmt.Sprintf("%d/%d", achieved, total))
	return count + starStyle.Render(strings.Repeat(barFilled, filled)) + dimStyle.Render(strings.Repeat(barEmpty, barCells-filled))
}
