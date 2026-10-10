package steam

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestURIs(t *testing.T) {
	tests := []struct {
		got, want string
	}{
		{PlayURI(620), "steam://rungameid/620"},
		{InstallURI(620), "steam://install/620"},
		{StoreURI(620), "steam://store/620"},
		{LibraryURI(620), "steam://nav/games/details/620"},
		{UninstallURI(620), "steam://uninstall/620"},
	}
	for _, tt := range tests {
		if tt.got != tt.want {
			t.Errorf("got %q, want %q", tt.got, tt.want)
		}
	}
}

func TestInLibraryCache(t *testing.T) {
	steamPath := t.TempDir()
	if err := os.MkdirAll(LibraryCacheDir(steamPath, 620), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(LibraryCacheDir(steamPath, 70), nil, 0o644); err != nil { // a file, not a folder
		t.Fatal(err)
	}
	for id, want := range map[uint32]bool{620: true, 70: false, 400: false} {
		if got := InLibraryCache(steamPath, id); got != want {
			t.Errorf("app %d: got %v, want %v", id, got, want)
		}
	}
	if want := filepath.Join(steamPath, "appcache", "librarycache", "620"); LibraryCacheDir(steamPath, 620) != want {
		t.Errorf("got %q, want %q", LibraryCacheDir(steamPath, 620), want)
	}
}

func TestLibraryRefusals(t *testing.T) {
	steamPath := t.TempDir()
	if err := os.MkdirAll(filepath.Join(steamPath, "logs"), 0o755); err != nil {
		t.Fatal(err)
	}
	long := strings.Repeat("x", 2<<20) // a line longer than a scanner's buffer
	log := "[2026-10-01 16:08:27] SteamUI: ERROR: Unrecognized request details 3810660\n" +
		"[2026-10-01 16:09:00] SteamUI: WARNING: Failed to load app details cache data for 220780\n" +
		long + "\n" +
		"[2026-10-01 16:13:29] SteamUI: ERROR: Unrecognized request details 3810660\n" +
		"[2026-10-01 16:14:00] SteamUI: ERROR: Unrecognized request details <id>\n" +
		"[2026-10-01 16:15:00] SteamUI: ERROR: Unrecognized request details 820520\r\n"
	if err := os.WriteFile(filepath.Join(steamPath, "logs", "webhelper_js.txt"), []byte(log), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LibraryRefusals(steamPath, time.UTC)
	want := map[uint32]time.Time{3810660: time.Date(2026, 10, 1, 16, 13, 29, 0, time.UTC), 820520: time.Date(2026, 10, 1, 16, 15, 0, 0, time.UTC)}
	if err != nil || len(got) != 2 || !got[3810660].Equal(want[3810660]) || !got[820520].Equal(want[820520]) {
		t.Errorf("got %v, %v; want %v", got, err, want)
	}
	if !RefusedSince(steamPath, 820520, time.Date(2026, 10, 1, 16, 15, 0, 500, time.UTC), time.UTC) {
		t.Error("a refusal in the same second counts")
	}
	if RefusedSince(steamPath, 820520, time.Date(2026, 10, 1, 16, 15, 1, 0, time.UTC), time.UTC) {
		t.Error("a refusal before the click doesn't")
	}
	if _, err := LibraryRefusals(t.TempDir(), time.UTC); err == nil {
		t.Error("no log is an error")
	}
}
