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

// columns lays out one line: marker, name (takes the free space), played
// right-aligned, last played. The column header and every row use it, so
// they always line up. The table never grows past maxTableWidth, so on a wide
// terminal the numbers stay close to the names.
func columns(width int, marker, name, played, last string) string {
	width = min(width, maxTableWidth-rowStyle.GetHorizontalFrameSize())
	nameWidth := max(width-1-1-hoursWidth-2-agoWidth, 1)
	return fmt.Sprintf("%s %-*s%*s  %-*s", marker, nameWidth, ansi.Truncate(name, nameWidth, "…"), hoursWidth, played, agoWidth, last)
}

// rowDelegate draws one game per line.
type rowDelegate struct {
	playtimeKnown bool
	now           func() time.Time
}

func (d rowDelegate) Height() int                         { return 1 }
func (d rowDelegate) Spacing() int                        { return 0 }
func (d rowDelegate) Update(tea.Msg, *list.Model) tea.Cmd { return nil }

func (d rowDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	g, ok := item.(gameItem)
	if !ok {
		return
	}
	style := rowStyle
	if index == m.Index() {
		style = selectedStyle
	}
	marker := " "
	if g.Installed {
		marker = markerInstalled
	}
	line := columns(m.Width()-style.GetHorizontalFrameSize(), marker, g.Name,
		library.FormatHours(g.Playtime, d.playtimeKnown), library.FormatAgo(g.LastPlayed, d.now()))
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
