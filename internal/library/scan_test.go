package library

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// Steam that no one has signed in to yet: no playtime, and no warning,
// since nothing is damaged.
func TestScanNeverSignedIn(t *testing.T) {
	steam := t.TempDir()
	if err := os.MkdirAll(filepath.Join(steam, "steamapps"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(steam, "steamapps", "libraryfolders.vdf"), []byte(`"libraryfolders" { }`), 0o644); err != nil {
		t.Fatal(err)
	}
	res, err := Scan(steam)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if !res.NoAccount || res.PlaytimeKnown || len(res.Warnings) != 0 || len(res.Games) != 0 {
		t.Errorf("got NoAccount %v, PlaytimeKnown %v, warnings %v, games %d", res.NoAccount, res.PlaytimeKnown, res.Warnings, len(res.Games))
	}
}

func TestScanFixture(t *testing.T) {
	res, err := Scan("../../testdata/steam")
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if !res.PlaytimeKnown {
		t.Error("fixture has an account with localconfig.vdf; playtime should be known")
	}
	if len(res.Warnings) != 1 {
		t.Errorf("want 1 warning (broken appmanifest_999.acf), got %v", res.Warnings)
	}
	// Half-Life (70) is mid-update: listed, marked updating.
	if len(res.Games) != 3 || res.Games[2].Name != "Portal 2" || res.Games[2].Playtime != 754*time.Minute {
		t.Errorf("unexpected games: %+v", res.Games)
	}
	if hl := res.Games[0]; hl.AppID != 70 || !hl.Updating || !hl.Installed || hl.Installing || hl.Paused {
		t.Errorf("Half-Life should be listed as updating, not installing or paused: %+v", hl)
	}
}

func TestScanNoSteam(t *testing.T) {
	if _, err := Scan(t.TempDir()); err == nil {
		t.Error("want an error when Steam is not found")
	}
}

func TestScanUninstalledCandidates(t *testing.T) {
	res, err := Scan("../../testdata/steam")
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	// Fixture: every app with play data (620, 400, and 70, mid-update) is
	// listed, so none is a candidate.
	if len(res.Uninstalled) != 0 {
		t.Errorf("want no candidates, got %+v", res.Uninstalled)
	}

	// Steam with 620 installed and play data for 70, which isn't.
	steam := t.TempDir()
	write := func(name, text string) {
		p := filepath.Join(steam, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("steamapps/appmanifest_620.acf", `"AppState" { "appid" "620" "name" "Portal 2" "StateFlags" "4" }`)
	write("config/loginusers.vdf", `"users" { "76561197960265730" { "Timestamp" "1" } }`)
	write("userdata/2/config/localconfig.vdf", `"UserLocalConfigStore" { "Software" { "Valve" { "Steam" { "apps" {
		"620" { "Playtime" "10" } "70" { "Playtime" "30" } } } } } }`)
	res, err = Scan(steam)
	if err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(res.Uninstalled) != 1 {
		t.Fatalf("want 70 as the only candidate, got %+v", res.Uninstalled)
	}
	if c := res.Uninstalled[0]; c.AppID != 70 || c.Installed || c.Name != "" || c.Playtime != 30*time.Minute {
		t.Errorf("candidate should be 70, not installed, unnamed, 30 minutes: %+v", c)
	}
}
