package steam

import "testing"

// Only Steam's own client counts, in any letter case and either slash.
func TestIsSteamImage(t *testing.T) {
	for path, want := range map[string]bool{
		`C:\Program Files (x86)\Steam\steam.exe`:          true,
		`c:/program files (x86)/steam/Steam.EXE`:          true,
		`C:\Program Files (x86)\Steam\steamwebhelper.exe`: false,
		`C:\Games\steam.exe.bak`:                          false,
		`C:\Tools\quarter.exe`:                            false,
	} {
		if got := isSteamImage(path); got != want {
			t.Errorf("isSteamImage(%q) = %v, want %v", path, got, want)
		}
	}
}
