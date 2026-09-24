package ui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"

	"github.com/stronxyo/quarter/internal/library"
)

// appKeys are Quarter's own keys, on top of the list's navigation and search.
type appKeys struct {
	Play       key.Binding
	Install    key.Binding
	Store      key.Binding
	SortName   key.Binding
	SortPlayed key.Binding
	SortLast   key.Binding
	Filter     key.Binding
	Refresh    key.Binding
	Help       key.Binding
}

var keys = appKeys{
	Play:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "play")),
	Install:    key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "install")),
	Store:      key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "store page")),
	SortName:   key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "sort by name")),
	SortPlayed: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "sort by played")),
	SortLast:   key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "sort by last played")),
	Filter:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "filter")),
	Refresh:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rescan")),
	Help:       key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "more")),
}

// Help-only entries: these describe keys the list handles, or group several
// keys under one label, and are never matched against input.
var (
	helpMove   = key.NewBinding(key.WithKeys("up", "down"), key.WithHelp("↑↓", "move"))
	helpEnds   = key.NewBinding(key.WithKeys("g", "G"), key.WithHelp("g/G", "top/bottom"))
	helpPage   = key.NewBinding(key.WithKeys("left", "right"), key.WithHelp("←→", "page"))
	helpSearch = key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "search"))
	helpSort   = key.NewBinding(key.WithKeys("n", "p", "l"), key.WithHelp("n/p/l", "sort"))
	helpQuit   = key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))
	helpLess   = key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "less"))
)

// shortHelp fits one line on an 80-column terminal.
func shortHelp() []key.Binding {
	return []key.Binding{helpMove, keys.Play, helpSearch, helpSort, keys.Filter, keys.Help, helpQuit}
}

// fullHelp is the ? view, every key grouped by what it does.
func fullHelp() [][]key.Binding {
	return [][]key.Binding{
		{helpMove, helpEnds, helpPage},
		{keys.Play, keys.Install, keys.Store},
		{helpSearch, keys.SortName, keys.SortPlayed, keys.SortLast},
		{keys.Filter, keys.Refresh, helpLess, helpQuit},
	}
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
// which Quarter uses to sort). Help is Quarter's own, so the list's is off.
func listKeys() list.KeyMap {
	k := list.DefaultKeyMap()
	k.Quit = key.NewBinding(key.WithKeys("q"))
	k.ForceQuit = key.NewBinding(key.WithKeys("ctrl+c"))
	k.Filter = key.NewBinding(key.WithKeys("/"))
	k.PrevPage = key.NewBinding(key.WithKeys("left", "pgup"))
	k.NextPage = key.NewBinding(key.WithKeys("right", "pgdown"))
	k.ShowFullHelp.SetEnabled(false)
	k.CloseFullHelp.SetEnabled(false)
	return k
}
