package ui

import (
	"fmt"
	"io"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/library"
)

// gameItem adapts a library.Game to the Bubbles list.
type gameItem struct{ library.Game }

func (g gameItem) FilterValue() string { return g.Name }

// rowDelegate draws one game per line: marker, name, hours right-aligned.
type rowDelegate struct {
	playtimeKnown bool
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

	// Row width minus the style's left padding/border, the marker and gaps.
	nameWidth := max(m.Width()-style.GetHorizontalFrameSize()-2-1-hoursWidth, 1)
	name := ansi.Truncate(g.Name, nameWidth, "…")
	line := fmt.Sprintf("%s %-*s %*s", marker, nameWidth, name, hoursWidth, library.FormatHours(g.Playtime, d.playtimeKnown))
	fmt.Fprint(w, style.Render(line))
}

func newList() list.Model {
	l := list.New(nil, rowDelegate{}, 0, 0)
	l.KeyMap = listKeys()
	l.Title = "QUARTER"
	l.Styles.Title = titleStyle
	l.SetStatusBarItemName("game", "games")
	l.FilterInput.Prompt = "Search: "
	return l
}

func toItems(games []library.Game) []list.Item {
	items := make([]list.Item, len(games))
	for i, g := range games {
		items[i] = gameItem{g}
	}
	return items
}
