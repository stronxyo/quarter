package config

import (
	"log/slog"

	"github.com/stronxyo/quarter/internal/store"
)

// Settings are the choices Quarter remembers between runs.
type Settings struct {
	Favourites []uint32 `json:"favourites"`
	Sort       string   `json:"sort"`        // column name, e.g. "last_played"
	SortDesc   bool     `json:"sort_desc"`   // true for highest or newest first
	Filter     string   `json:"filter"`      // filter name, e.g. "recent"
	HidePane   bool     `json:"hide_pane"`   // d hid the detail pane on a wide window
	HideLogos  bool     `json:"hide_logos"`  // o hid game logos
	LogoBlocks string   `json:"logo_blocks"` // "octants" for twice the detail, in fonts with block octants; "" quarter blocks
	SplashSeen bool     `json:"splash_seen"` // the start screen has been shown
	Intro      string   `json:"intro"`       // "" a random intro each start and as it waits, "off", or one intro's name
	LastIntro  string   `json:"last_intro"`  // the intro played last, so the next start picks another

	Goodbye     string `json:"goodbye"`      // "" a random goodbye on quitting, "off", or one goodbye's name
	LastGoodbye string `json:"last_goodbye"` // the goodbye shown last, so the next quit picks another

	Mouse string `json:"mouse"` // "" Quarter takes the mouse; "off" leaves it to the terminal (plain text selection)

	Glyphs string `json:"glyphs"` // "" found out, "basic" the classic Windows console's way (stand-in characters, no game logos), "full" every character

	SteamStart string `json:"steam_start"` // "" Steam starts in the tray when Quarter opens, and games start without its window (Windows); "normal" Steam opens as it always has
}

const (
	settingsFile    = "config.json"
	settingsVersion = 1
)

// LoadSettings reads config.json from dir. A missing file gives empty
// settings. A file that exists but can't be read is renamed to
// config.json.corrupt, so the user's favourites are never silently overwritten.
func LoadSettings(dir string) Settings {
	s, ok := store.Load[Settings](dir, settingsFile, settingsVersion)
	if ok {
		return s
	}
	switch moved, err := store.SetAside(dir, settingsFile); {
	case err != nil:
		slog.Warn("settings unreadable and could not be moved aside", "err", err)
	case moved:
		slog.Warn("settings unreadable, moved to " + settingsFile + ".corrupt")
	}
	return Settings{}
}

// SaveSettings writes config.json to dir.
func SaveSettings(dir string, s Settings) error {
	return store.Save(dir, settingsFile, settingsVersion, s)
}
