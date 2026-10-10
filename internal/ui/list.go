package ui

import (
	"fmt"
	"io"
	"time"

	"charm.land/bubbles/v2/list"
	"charm.land/bubbles/v2/paginator"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/library"
)

// gameItem adapts a library.Game to the Bubbles list.
type gameItem struct{ library.Game }

func (g gameItem) FilterValue() string { return g.Name }

// columns lays out one line: marker (favourite star or installed dot), name
// (takes the free space), played
// right-aligned, last played. The column header and every row use it, so
// they always line up. The table never grows past maxTableWidth, so on a wide
// terminal the numbers stay close to the names.
func columns(width int, marker, name, played, last string) string {
	width = min(width, maxTableWidth-rowStyle.GetHorizontalFrameSize())
	nameWidth := max(width-markerWidth-1-hoursWidth-2-agoWidth, 1)
	return fmt.Sprintf("%s %-*s%*s  %-*s", marker, nameWidth, ansi.Truncate(name, nameWidth, "…"), hoursWidth, played, agoWidth, last)
}

// rowDelegate draws one game per line.
type rowDelegate struct {
	playtimeKnown bool
	now           func() time.Time
	running       uint32                    // the game Steam is running, 0 for none
	dlCell        func(library.Game) string // a downloading game's last column; nil: "updating"
}

func (d rowDelegate) Height() int                         { return 1 }
func (d rowDelegate) Spacing() int                        { return 0 }
func (d rowDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d rowDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	g, ok := item.(gameItem)
	if !ok {
		return
	}
	selected := index == m.Index()
	running := d.running != 0 && g.AppID == d.running
	updating := g.Updating && !running

	// One marker per row: a big triangle while running, an arrow while Steam
	// updates it, else a favourite's star, else a small triangle (filled
	// when installed).
	marker := markerUninstalled
	switch {
	case running:
		marker = markerRunning // the big triangle: playing now
	case updating:
		marker = markerUpdating
	case g.Favourite:
		marker = favouriteMarker(g)
	case g.Installed:
		marker = markerInstalled
	}

	last := library.FormatAgo(g.LastPlayed, d.now())
	switch {
	case running:
		last = markerRunning + " running"
	case updating:
		last = markerUpdating + " updating"
		if d.dlCell != nil {
			last = d.dlCell(g.Game)
		}
	}
	// The marker takes one cell; it's drawn separately below.
	line := columns(m.Width()-rowStyle.GetHorizontalFrameSize(), " ", g.Label(),
		library.FormatHours(g.Playtime, d.playtimeKnown), last)[1:]

	// The selected, running and updating rows are one color from marker to
	// end: a colored star inside would end in a reset that switches the row's
	// color off after it. Other rows color the star and the text separately.
	switch {
	case selected && running:
		fmt.Fprint(w, selectedRunningStyle.Render(marker+line))
	case selected && updating:
		fmt.Fprint(w, selectedUpdateStyle.Render(marker+line))
	case selected:
		fmt.Fprint(w, selectedStyle.Render(marker+line))
	case running:
		fmt.Fprint(w, runningStyle.Render(marker+line))
	case updating:
		fmt.Fprint(w, updatingStyle.Render(marker+line))
	case g.Favourite:
		fmt.Fprint(w, rowStyle.Render(starStyle.Render(marker)+textStyle.Render(line)))
	default:
		fmt.Fprint(w, rowStyle.Render(textStyle.Render(marker+line)))
	}
}

func newList() list.Model {
	l := list.New(nil, rowDelegate{now: time.Now}, 0, 0)
	l.KeyMap = listKeys()
	// The app draws the title, search box and game count itself.
	l.SetShowTitle(false)
	l.SetShowFilter(false)
	l.SetShowStatusBar(false)
	// A page counter ("3/170") instead of one dot per page: dots don't fit
	// big libraries, and rendering them dominated startup time.
	l.Paginator.Type = paginator.Arabic
	l.FilterInput.Prompt = "Search: "
	s := l.FilterInput.Styles()
	s.Focused.Text, s.Blurred.Text = textStyle, textStyle
	l.FilterInput.SetStyles(s)
	l.SetShowHelp(false) // Quarter draws its own key help
	return l
}

func toItems(games []library.Game) []list.Item {
	items := make([]list.Item, len(games))
	for i, g := range games {
		items[i] = gameItem{g}
	}
	return items
}

// favouriteMarker is a filled star, or an empty one when the game isn't installed.
func favouriteMarker(g gameItem) string {
	if g.Installed {
		return markerFavourite
	}
	return markerFavouriteUninstalled
}
