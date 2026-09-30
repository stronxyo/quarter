package ui

import (
	"slices"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"

	"github.com/stronxyo/quarter/internal/library"
)

// appKeys are Quarter's own keys, on top of the list's navigation and search.
type appKeys struct {
	Play       key.Binding
	Install    key.Binding
	Uninstall  key.Binding
	Store      key.Binding
	Favourite  key.Binding
	SortName   key.Binding
	SortPlayed key.Binding
	SortLast   key.Binding
	Filter     key.Binding
	Refresh    key.Binding
	Details    key.Binding
	Logos      key.Binding
	History    key.Binding
	Help       key.Binding
}

var keys = appKeys{
	Play:       key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "play")),
	Install:    key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "install")),
	Uninstall:  key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "uninstall")),
	Store:      key.NewBinding(key.WithKeys("s"), key.WithHelp("s", "store page")),
	Favourite:  key.NewBinding(key.WithKeys("f"), key.WithHelp("f", "favourite")),
	SortName:   key.NewBinding(key.WithKeys("n"), key.WithHelp("n", "sort name")),
	SortPlayed: key.NewBinding(key.WithKeys("p"), key.WithHelp("p", "sort played")),
	SortLast:   key.NewBinding(key.WithKeys("l"), key.WithHelp("l", "sort last")),
	Filter:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "filter")),
	Refresh:    key.NewBinding(key.WithKeys("r"), key.WithHelp("r", "rescan")),
	Details:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "details")),
	Logos:      key.NewBinding(key.WithKeys("o"), key.WithHelp("o", "logos")),
	History:    key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "history")),
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

	// While history is open.
	helpHistNext   = key.NewBinding(key.WithKeys("h"), key.WithHelp("h", "next view"))
	helpHistPeriod = key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "period"))
	helpHistBack   = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "games"))

	// The selected game's keys, by its state (see gameKeys).
	helpInstallBoth  = key.NewBinding(key.WithKeys("enter", "i"), key.WithHelp("enter/i", "install"))
	helpEnterInstall = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "install"))
	helpUninstall    = key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "uninstall"))
	helpRunning      = key.NewBinding(key.WithKeys("enter"), key.WithHelp(markerRunning, "running"))
	helpUpdating     = key.NewBinding(key.WithKeys("enter"), key.WithHelp(markerUpdating, "updating"))

	// While typing a search, and with the pop-up pane open.
	helpClear = key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "clear"))
	helpClose = key.NewBinding(key.WithKeys("d", "esc"), key.WithHelp("d/esc", "close"))

	// In the ? view only.
	helpPlayFull  = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "play/install"))
	helpPgKeys    = key.NewBinding(key.WithKeys("pgup", "pgdown"), key.WithHelp("pgup/dn", "page"))
	helpHomeEnd   = key.NewBinding(key.WithKeys("home", "end"), key.WithHelp("home/end", "top/end"))
	helpInstFull  = key.NewBinding(key.WithKeys("i"), key.WithHelp("i", "install"))
	helpUninsFull = key.NewBinding(key.WithKeys("u"), key.WithHelp("u", "uninstall x2"))
)

// gameKeys are the key line's keys for the selected game: only what its
// state allows.
func gameKeys(state gameState) []key.Binding {
	switch state {
	case gameRunning:
		return []key.Binding{helpRunning}
	case gameUpdating:
		return []key.Binding{helpUpdating}
	case gameInstalled:
		return []key.Binding{keys.Play, helpUninstall}
	case gameNotInstalled:
		return []key.Binding{helpInstallBoth}
	}
	return nil
}

// searchKeys are the key line while typing a search: what Enter does to the
// game under the cursor, and esc.
func searchKeys(state gameState) []key.Binding {
	enter := map[gameState]key.Binding{gameRunning: helpRunning, gameUpdating: helpUpdating, gameInstalled: keys.Play, gameNotInstalled: helpEnterInstall}
	if b, ok := enter[state]; ok {
		return []key.Binding{b, helpClear}
	}
	return []key.Binding{helpClear}
}

// shortHelp is the one line of keycaps under the list, as versions from widest
// to narrowest: the first that fits is shown. The arrow keys, the easiest to
// guess, give way first on narrow windows, then h history and the sort keys
// (all still listed under ?).
func shortHelp(game []key.Binding) [][]key.Binding {
	core := slices.Concat(game, []key.Binding{helpSearch, helpSort, keys.Filter, keys.History, keys.Help, helpQuit})
	without := func(drop ...key.Binding) []key.Binding {
		return slices.DeleteFunc(slices.Clone(core), func(b key.Binding) bool {
			return slices.ContainsFunc(drop, func(d key.Binding) bool { return d.Help() == b.Help() })
		})
	}
	return [][]key.Binding{
		append([]key.Binding{helpMove, helpPage}, core...),
		append([]key.Binding{helpPage}, core...),
		core,
		without(keys.History),
		without(keys.History, helpSort),
	}
}

// historyHelp is the key line while history is open, widest first.
func historyHelp() [][]key.Binding {
	core := []key.Binding{helpHistPeriod, helpHistNext, helpHistBack, helpQuit}
	return [][]key.Binding{
		append([]key.Binding{helpMove, helpPage, helpEnds}, core...),
		append([]key.Binding{helpMove, helpPage}, core...),
		append([]key.Binding{helpPage}, core...),
		core,
	}
}

// fullHelp is the ? view, every key grouped by what it does.
func fullHelp() [][]key.Binding {
	return [][]key.Binding{
		{helpMove, helpEnds, helpPage, helpPgKeys, helpHomeEnd},
		{helpPlayFull, helpInstFull, helpUninsFull, keys.Store, keys.Favourite},
		{helpSearch, keys.SortName, keys.SortPlayed, keys.SortLast},
		{keys.Filter, keys.History, keys.Details, keys.Logos, keys.Refresh, helpLess, helpQuit},
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
