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
			Border(lipgloss.DoubleBorder()).
			BorderForeground(colorLogoBorder).
			Padding(0, 1)
	paneTitleStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
)

// sideBySide reports whether the window has room for the list and the detail
// pane next to each other. Otherwise the pane opens as a pop-up with d.
func (m Model) sideBySide() bool {
	return m.width >= minTableWidth+paneGap+paneWidth
}

// tableWidth is the list's width: up to maxTableWidth, less whatever the
// detail pane needs beside it.
func (m Model) tableWidth() int {
	if m.sideBySide() {
		return min(m.width-paneGap-paneWidth, maxTableWidth)
	}
	return min(m.width, maxTableWidth)
}

// detailView draws the selected game's pane, width by height cells.
func (m Model) detailView(width, height int) string {
	inner := width - paneStyle.GetHorizontalFrameSize()
	var b strings.Builder

	g, ok := m.selected()
	if !ok {
		b.WriteString(dimStyle.Render("No game selected."))
	} else {
		// The game's logo goes here; until then, its name.
		b.WriteString(paneTitleStyle.Width(inner).Render(strings.ToUpper(g.Label())))
		b.WriteString("\n\n")
		row := func(label, value string) {
			b.WriteString(dimStyle.Render(fmt.Sprintf("%-12s", label)))
			b.WriteString(ansi.Truncate(value, max(inner-12, 1), "…"))
			b.WriteString("\n")
		}
		row("Played", library.FormatHours(g.Playtime, m.playtimeKnown))
		row("Last played", library.FormatAgo(g.LastPlayed, m.now()))
		if g.Installed {
			row("Size", library.FormatSize(g.SizeOnDisk))
			row("Library", g.LibraryPath)
		} else {
			row("Status", "not installed")
		}
		if g.Favourite {
			b.WriteString("\n" + starStyle.Render(markerFavourite+" Favourite"))
		}
	}
	return paneStyle.Width(width).Height(height).Render(strings.TrimRight(b.String(), "\n"))
}

// selected returns the highlighted game, if the list has one.
func (m Model) selected() (library.Game, bool) {
	it, ok := m.list.SelectedItem().(gameItem)
	return it.Game, ok
}
