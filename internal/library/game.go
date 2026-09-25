// Package library builds Quarter's view of the user's games by merging every
// data source into one Game per app.
package library

import (
	"strings"
	"time"
)

// Game is one entry in the library.
type Game struct {
	AppID       uint32
	Name        string
	Installed   bool
	LibraryPath string
	SizeOnDisk  int64
	Playtime    time.Duration
	LastPlayed  time.Time // zero if never played
	Favourite   bool      `json:"-"` // comes from settings, not from Steam, so never cached
	Demo        bool      `json:"-"` // the store says this is a demo; set from store data
}

// trialWords end the names of builds that aren't the full game. Steam's own
// app type would be exact; until the store data is read, the name is the
// signal.
var trialWords = []string{"Demo", "Playtest"}

// Label is the name to show: a trailing "Demo" or "Playtest" becomes a
// "(Demo)" or "(Playtest)" tag, so "Slow Roads Demo" reads "Slow Roads (Demo)".
// A demo the store knows about is tagged even when its name doesn't say so.
func (g Game) Label() string {
	for _, w := range trialWords {
		base, ok := strings.CutSuffix(g.Name, " "+w)
		if !ok {
			continue
		}
		base = strings.TrimRight(base, " -\u2013:") // space, hyphen, en dash, colon
		if base == "" {
			return g.Name
		}
		return base + " (" + w + ")"
	}
	if g.Demo {
		return g.Name + " (Demo)"
	}
	return g.Name
}
