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
	return m.width >= minTableWidth+paneGap+paneMinWidth
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
		// The game's logo goes here; until then, its name.
		b.WriteString(paneTitleStyle.Width(inner).Render(strings.ToUpper(g.Label())))
		b.WriteString("\n")

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
				b.WriteString(ansi.Truncate(strings.Join(who, " · "), inner, "…") + "\n")
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
			b.WriteString("\n" + starStyle.Render(markerFavourite+" Favourite") + "\n")
		}
		if state == storeReady && info.Description != "" {
			b.WriteString("\n" + lipgloss.NewStyle().Width(inner).Render(info.Description))
		}
	}
	// MaxHeight cuts a long description at the pane's bottom edge.
	return paneStyle.Width(width).Height(height).MaxHeight(height).Render(strings.TrimRight(b.String(), "\n"))
}

// selected returns the highlighted game, if the list has one.
func (m Model) selected() (library.Game, bool) {
	it, ok := m.list.SelectedItem().(gameItem)
	return it.Game, ok
}
