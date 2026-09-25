package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSettingsRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := Settings{Favourites: []uint32{620, 400}, Sort: "last_played", SortDesc: true, Filter: "recent"}
	if err := SaveSettings(dir, want); err != nil {
		t.Fatalf("SaveSettings: %v", err)
	}
	got := LoadSettings(dir)
	if got.Sort != want.Sort || got.SortDesc != want.SortDesc || got.Filter != want.Filter ||
		len(got.Favourites) != 2 || got.Favourites[0] != 620 || got.Favourites[1] != 400 {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestSettingsMissing(t *testing.T) {
	got := LoadSettings(t.TempDir())
	if len(got.Favourites) != 0 || got.Sort != "" || got.Filter != "" {
		t.Errorf("missing file should give empty settings, got %+v", got)
	}
}

func TestCorruptSettingsMovedAside(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.json")
	if err := os.WriteFile(path, []byte(`{"favourites": [620`), 0o644); err != nil {
		t.Fatal(err)
	}

	if got := LoadSettings(dir); len(got.Favourites) != 0 {
		t.Errorf("corrupt file should give empty settings, got %+v", got)
	}
	if _, err := os.Stat(path + ".corrupt"); err != nil {
		t.Errorf("corrupt file should be kept as config.json.corrupt: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Error("the corrupt config.json should have been moved, not left in place")
	}
}
