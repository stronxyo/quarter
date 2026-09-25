package ui

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/library"
)

var (
	paneStyle = lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(colorLogoBorder).
			Padding(0, 1)
	paneTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
)

// paneLabelWidth fits the longest label, "Achievements", plus a space.
const paneLabelWidth = 13

// The achievement bar: a middle-third block (U+1FB0B) for unlocked, a light
// rule for the rest.
const (
	barCells      = 14
	barCountWidth = 8 // fits "999/999 "
	barFilled     = "🬋"
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

// paneWidth grows with the window, from paneMinWidth up to paneMaxWidth, so
// big windows give the game logo its full width.
func (m Model) paneWidth() int {
	if m.sideBySide() {
		return min(m.width-paneGap-m.tableWidth(), paneMaxWidth)
	}
	return min(m.width, paneMaxWidth)
}

// detailView draws the selected game's pane, width by height cells.
func (m Model) detailView(width, height int) string {
	inner := width - paneStyle.GetHorizontalFrameSize()
	var b strings.Builder

	g, ok := m.selected()
	if !ok {
		b.WriteString(dimStyle.Render("No game selected."))
	} else {
		// The game's logo, in an area of fixed height so the lines below
		// don't jump from game to game; its name when logos are off or the
		// pane is too short for one.
		if cols, rows := logoArea(width, height); m.logos != nil && !m.logosHidden && rows > 0 {
			b.WriteString(m.logoView(g, cols, rows) + "\n")
		} else {
			b.WriteString(paneTitleStyle.Width(inner).Render(strings.ToUpper(g.Label())) + "\n")
		}

		if g.AppID == m.running {
			b.WriteString(runningLabelStyle.Render(markerRunning+" RUNNING NOW") + "\n")
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
			b.WriteString(dimStyle.Render("store info unavailable") + "\n")
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

// selected returns the highlighted game, if the list has one.
func (m Model) selected() (library.Game, bool) {
	it, ok := m.list.SelectedItem().(gameItem)
	return it.Game, ok
}

// achievementBar draws the count, then a bar: a thicker orange band for what's
// unlocked and a thin dim line for the rest, e.g. "72/88   🬋🬋🬋🬋🬋🬋🬋🬋🬋🬋🬋───".
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
