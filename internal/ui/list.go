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
	running       uint32 // the game Steam is running, 0 for none
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
	style := rowStyle
	switch {
	case selected && running:
		style = selectedRunningStyle
	case selected:
		style = selectedStyle
	case running:
		style = runningStyle
	}

	// A favourite's star takes the place of the installed dot. On a row that
	// is colored as a whole (selected or running), the star's own color would
	// end in a reset that switches the row's color off after it.
	marker := " "
	switch {
	case g.Favourite && (selected || running):
		marker = markerFavourite
	case g.Favourite:
		marker = starStyle.Render(markerFavourite)
	case g.Installed:
		marker = markerInstalled
	}

	last := library.FormatAgo(g.LastPlayed, d.now())
	if running {
		last = markerRunning + " running"
	}
	line := columns(m.Width()-style.GetHorizontalFrameSize(), marker, g.Label(),
		library.FormatHours(g.Playtime, d.playtimeKnown), last)
	fmt.Fprint(w, style.Render(line))
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
