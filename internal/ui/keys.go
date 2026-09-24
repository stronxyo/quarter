package ui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"

	"github.com/stronxyo/quarter/internal/library"
)

// appKeys are Quarter's own keys, on top of the list's navigation and search.
type appKeys struct {
	SortName   key.Binding
	SortPlayed key.Binding
	SortLast   key.Binding
	Filter     key.Binding
}

var keys = appKeys{
	SortName:   key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "name")),
	SortPlayed: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "played")),
	SortLast:   key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "last played")),
	Filter:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "filter")),
}

// shortHelp fits one line on an 80-column terminal, so the sort keys share
// one entry; fullHelp (the ? view) lists them separately.
func (k appKeys) shortHelp() []key.Binding {
	return []key.Binding{key.NewBinding(key.WithKeys("n", "p", "l"), key.WithHelp("n/p/l", "sort")), k.Filter}
}

func (k appKeys) fullHelp() []key.Binding {
	return []key.Binding{k.SortName, k.SortPlayed, k.SortLast, k.Filter}
}

// sortKeys maps each sort key to the column it sorts by.
var sortKeys = []struct {
	binding key.Binding
	order   library.SortOrder
}{
	{keys.SortName, library.SortName},
	{keys.SortPlayed, library.SortPlaytime},
	{keys.SortLast, library.SortLastPlayed},
}

// listKeys sets the keys Quarter relies on explicitly, rather than trusting the
// list's defaults (in bubbles v2.2.1 its Quit binding is "v", and "l" pages,
// which Quarter uses to sort).
func listKeys() list.KeyMap {
	k := list.DefaultKeyMap()
	k.Quit = key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))
	k.ForceQuit = key.NewBinding(key.WithKeys("ctrl+c"))
	k.Filter = key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search"))
	k.PrevPage = key.NewBinding(key.WithKeys("left", "pgup"), key.WithHelp("←/pgup", "prev page"))
	k.NextPage = key.NewBinding(key.WithKeys("right", "pgdown"), key.WithHelp("→/pgdn", "next page"))
	return k
}
