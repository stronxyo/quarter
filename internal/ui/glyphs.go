package ui

import (
	"charm.land/bubbles/v2/key"

	"github.com/stronxyo/quarter/internal/intro"
)

// The classic Windows console (conhost) draws a character missing from its
// font as a box: it doesn't borrow it from another font, as Windows
// Terminal does. Its fonts (Consolas, and the Cascadia Mono that ships with
// Windows) lack the arrow, the stars, some triangles, and the quarter and
// eighth blocks (Pablo's screenshots, 2026-10-02). There Quarter uses
// stand-ins both fonts have, and shows no game logos, which it can't draw
// cleanly with half blocks alone (Pablo, after the Console lab: "we go with
// c"). Lucida Console lacks even the frames and isn't covered.

// glyphs is one set of the characters that differ.
type glyphs struct {
	link, uninstalled, favourite, favouriteUninstalled, running string
	sparkSteps, barParts                                        string
}

var (
	fullGlyphs  = glyphs{"↗", "▹", "✦", "✧", "▶", "▁▂▃▄▅▆▇█", " ▏▎▍▌▋▊▉"}
	basicGlyphs = glyphs{"→", "›", "♦", "◊", "►", "░░▒▒▓▓██", "    ▌▌▌▌"}
)

// applyGlyphs sets the characters for the whole UI, and what is built from
// them; New calls it once, before anything is drawn.
func applyGlyphs(basic bool) {
	g := fullGlyphs
	if basic {
		g = basicGlyphs
	}
	markerLink, markerUninstalled, markerRunning = g.link, g.uninstalled, g.running
	markerFavourite, markerFavouriteUninstalled = g.favourite, g.favouriteUninstalled
	sparkSteps, barParts = g.sparkSteps, g.barParts
	supportText = "buy me a coffee " + markerLink
	keys.Support = key.NewBinding(key.WithKeys("c"), key.WithHelp("c", supportText))
	helpSupportFull = key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "coffee "+markerLink))
	helpRunning = key.NewBinding(key.WithKeys("enter"), key.WithHelp(markerRunning, "running"))
	intro.BasicGlyphs = basic
}
