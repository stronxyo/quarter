package steam

import (
	"os"
	"path/filepath"
	"testing"
)

func TestURIs(t *testing.T) {
	tests := []struct {
		got, want string
	}{
		{PlayURI(620), "steam://rungameid/620"},
		{InstallURI(620), "steam://install/620"},
		{StoreURI(620), "steam://store/620"},
		{LibraryURI(620), "steam://nav/games/details/620"},
		{LibraryHomeURI, "steam://nav/games"},
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
