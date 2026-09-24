package steam

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/stronxyo/quarter/internal/config"
)

const fixture = "../../testdata/steam"

func TestLocateOverride(t *testing.T) {
	got, err := Locate(fixture)
	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	if got != filepath.Clean(fixture) {
		t.Errorf("got %q, want %q", got, filepath.Clean(fixture))
	}
}

func TestLocateEnv(t *testing.T) {
	t.Setenv(config.SteamPathEnv, fixture)
	got, err := Locate("")
	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	if got != filepath.Clean(fixture) {
		t.Errorf("got %q, want %q", got, filepath.Clean(fixture))
	}
}

func TestLocateNotSteam(t *testing.T) {
	for _, dir := range []string{t.TempDir(), filepath.Join(t.TempDir(), "missing")} {
		if _, err := Locate(dir); !errors.Is(err, ErrNotFound) {
			t.Errorf("Locate(%q): got %v, want ErrNotFound", dir, err)
		}
	}
}
