package library

import (
	"testing"
	"time"
)

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
