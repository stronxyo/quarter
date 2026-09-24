package ui

import (
	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
)

// listKeys sets the keys Quarter relies on explicitly, rather than trusting the
// list's defaults (in bubbles v2.2.1 its Quit binding is "v").
func listKeys() list.KeyMap {
	k := list.DefaultKeyMap()
	k.Quit = key.NewBinding(key.WithKeys("q"), key.WithHelp("q", "quit"))
	k.ForceQuit = key.NewBinding(key.WithKeys("ctrl+c"))
	return k
}
