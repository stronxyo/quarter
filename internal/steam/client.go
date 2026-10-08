package steam

import (
	"path/filepath"
	"strconv"
	"strings"
)

// Steam starts quietly (Pablo, 2026-10-07; specs/STEAM_QUIET_START.md):
// Valve's -silent starts the client in the tray with no window ("Silent
// startup mode (tray mode only)", Valve Developer Community, "Command line
// options (Steam)"), and -applaunch <id> starts a game through it. Tried on
// Pablo's PC, 2026-10-08: with Steam closed, "steam -silent -applaunch <id>"
// started the game with no Steam window, and a -applaunch sent while Steam
// was still signing in was held and run once it had.

// Client is what Steam's client says about itself.
type Client struct {
	Running  bool // the client is running
	SignedIn bool // and an account is signed in, so games can start
}

// ClientState reports Steam's client. known is false where Quarter can't
// tell (see client_other.go).
func ClientState() (c Client, known bool) {
	return clientState()
}

// StartQuiet starts Steam in the tray, with no window. steamPath is the
// --steam-path override, empty to find Steam. Steam must be closed for
// -silent to apply (Valve).
func StartQuiet(steamPath string) error {
	return startClient(steamPath, "-silent")
}

// PlayQuiet starts a game through Steam, first starting Steam in the tray
// when it isn't running.
func PlayQuiet(steamPath string, appID uint32) error {
	return startClient(steamPath, "-silent", "-applaunch", strconv.FormatUint(uint64(appID), 10))
}

// isSteamImage reports whether a program's full path is Steam's client,
// steam.exe, in any letter case.
func isSteamImage(path string) bool {
	return strings.EqualFold(filepath.Base(filepath.FromSlash(path)), "steam.exe")
}
