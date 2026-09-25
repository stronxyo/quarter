package config

import (
	"log/slog"
	"os"
	"path/filepath"

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
	LogoBlocks string   `json:"logo_blocks"` // "sixths" for fonts without eighth blocks
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
	path := filepath.Join(dir, settingsFile)
	if _, err := os.Stat(path); err == nil {
		if err := os.Rename(path, path+".corrupt"); err != nil {
			slog.Warn("settings unreadable and could not be moved aside", "err", err)
		} else {
			slog.Warn("settings unreadable, moved to " + settingsFile + ".corrupt")
		}
	}
	return Settings{}
}

// SaveSettings writes config.json to dir.
func SaveSettings(dir string, s Settings) error {
	return store.Save(dir, settingsFile, settingsVersion, s)
}
