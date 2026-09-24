package steam

import (
	"strings"
	"testing"
	"time"

	"github.com/stronxyo/quarter/internal/vdf"
)

func TestInstalledGamesFixture(t *testing.T) {
	games, warnings := InstalledGames(fixture)

	// The fixture's broken appmanifest_999.acf must be reported, not fatal.
	if len(warnings) != 1 || !strings.Contains(warnings[0].Error(), "appmanifest_999.acf") {
		t.Errorf("warnings: got %v, want one for appmanifest_999.acf", warnings)
	}

	// 620 and 400 are installed games; 228980 is a tool; 70 is mid-update.
	var ids []uint32
	for _, g := range games {
		ids = append(ids, g.AppID)
	}
	if len(ids) != 2 || ids[0] != 620 || ids[1] != 400 {
		t.Fatalf("got appids %v, want [620 400]", ids)
	}

	portal2 := games[0]
	if portal2.Name != "Portal 2" || portal2.SizeOnDisk != 12884901888 || !portal2.LastUpdated.Equal(time.Unix(1717000000, 0)) {
		t.Errorf("Portal 2 parsed wrong: %+v", portal2)
	}
}

func TestParseManifest(t *testing.T) {
	tests := []struct {
		name      string
		in        string
		wantErr   bool
		wantName  string
		installed bool
	}{
		{"installed", `"AppState" { "appid" "620" "name" "Portal 2" "StateFlags" "4" }`, false, "Portal 2", true},
		{"installed with update pending", `"AppState" { "appid" "620" "name" "Portal 2" "StateFlags" "6" }`, false, "Portal 2", true},
		{"updating, not playable", `"AppState" { "appid" "70" "name" "Half-Life" "StateFlags" "1026" }`, false, "Half-Life", false},
		{"missing name", `"AppState" { "appid" "5" "StateFlags" "4" }`, false, "App 5", true},
		{"lowercase keys", `"appstate" { "APPID" "7" "Name" "X" "stateflags" "4" }`, false, "X", true},
		{"bad numbers", `"AppState" { "appid" "8" "name" "Y" "StateFlags" "x" "SizeOnDisk" "-5" }`, false, "Y", false},
		{"no appid", `"AppState" { "name" "Z" }`, true, "", false},
		{"zero appid", `"AppState" { "appid" "0" }`, true, "", false},
		{"no AppState", `"Other" { }`, true, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := vdf.Parse(strings.NewReader(tt.in))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			m, err := parseManifest(root, "lib")
			if tt.wantErr {
				if err == nil {
					t.Fatal("want an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("parseManifest: %v", err)
			}
			if m.Name != tt.wantName || m.Installed() != tt.installed {
				t.Errorf("got name %q installed %v, want %q %v", m.Name, m.Installed(), tt.wantName, tt.installed)
			}
		})
	}
}

func TestIsGame(t *testing.T) {
	if IsGame(228980) || IsGame(1493710) {
		t.Error("Steam tools must not count as games")
	}
	if !IsGame(620) {
		t.Error("Portal 2 is a game")
	}
}
