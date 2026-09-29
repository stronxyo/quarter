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
	if len(res.Games) != 2 || res.Games[1].Name != "Portal 2" || res.Games[1].Playtime != 754*time.Minute {
		t.Errorf("unexpected games: %+v", res.Games)
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
	// Fixture: 620 and 400 are installed; 70 has play data but isn't
	// installed (it's mid-update); 70's library page is broken.
	var ids []uint32
	for _, g := range res.Uninstalled {
		if g.Installed || g.Name != "" {
			t.Errorf("candidate %d should be not installed and unnamed: %+v", g.AppID, g)
		}
		ids = append(ids, g.AppID)
	}
	if len(ids) != 1 || ids[0] != 70 {
		t.Fatalf("want only 70 as a candidate, got %v", ids)
	}
	if res.Uninstalled[0].Playtime != 30*time.Minute {
		t.Errorf("70's playtime should come from localconfig.vdf: %v", res.Uninstalled[0].Playtime)
	}
}
