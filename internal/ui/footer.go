package ui

import (
	"errors"
	"fmt"
	"strings"

	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/steam"
)

// missingText names the libraries whose drive isn't connected, and how many
// games that hides until it is.
func missingText(missing []steam.LibraryFolder) string {
	games := 0
	for _, l := range missing {
		games += l.Games
	}
	what := fmt.Sprintf("%d libraries not connected", len(missing))
	if len(missing) == 1 {
		what = "Library " + missing[0].Path + " not connected"
	}
	if games == 0 {
		return what + "."
	}
	return what + ", " + plural(games, "game") + " hidden."
}

// warningText says which Steam file couldn't be read and what that costs,
// or how many when several couldn't. The details are in the log.
func warningText(warnings []error) string {
	if len(warnings) > 1 {
		return fmt.Sprintf("%d Steam files can't be read, see %s.", len(warnings), config.LogFile)
	}
	var fe *steam.FileError
	if !errors.As(warnings[0], &fe) {
		return fmt.Sprintf("A Steam file can't be read, see %s.", config.LogFile)
	}
	switch name := fe.Name; {
	case strings.HasPrefix(name, "appmanifest_"):
		app := strings.TrimSuffix(strings.TrimPrefix(name, "appmanifest_"), ".acf")
		return fmt.Sprintf("Steam's file for app %s (%s) can't be read, so that game is hidden.", app, name)
	case name == "localconfig.vdf":
		return "Steam's playtime file (localconfig.vdf) can't be read, so playtime is hidden."
	case name == "loginusers.vdf":
		return "Steam's sign-in list (loginusers.vdf) can't be read, so playtime is hidden."
	case name == "libraryfolders.vdf":
		return "Steam's library list (libraryfolders.vdf) can't be read, so only games in the Steam folder show."
	default:
		return fmt.Sprintf("Steam's file %s can't be read, see %s.", name, config.LogFile)
	}
}

// plural is n and the noun, with an s unless n is 1.
func plural(n int, noun string) string {
	if n == 1 {
		return "1 " + noun
	}
	return fmt.Sprintf("%d %ss", n, noun)
}
