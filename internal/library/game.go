// Package library builds Quarter's view of the user's games by merging every
// data source into one Game per app.
package library

import "time"

// Game is one entry in the library.
type Game struct {
	AppID       uint32
	Name        string
	Installed   bool
	LibraryPath string
	SizeOnDisk  int64
	Playtime    time.Duration
	LastPlayed  time.Time // zero if never played
}
