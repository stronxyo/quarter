package steam

import (
	"errors"
	"path/filepath"
	"strings"
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

func TestLocateSaysWhere(t *testing.T) {
	dir := t.TempDir()
	_, err := Locate(dir)
	var nf *NotFoundError
	if !errors.As(err, &nf) || nf.Given != filepath.Clean(dir) {
		t.Errorf("a named folder that isn't Steam should be reported as given: %v", err)
	}
	if !strings.Contains(err.Error(), "no steamapps folder") {
		t.Errorf("message: %q", err.Error())
	}
}

func TestPlatformCandidates(t *testing.T) {
	if len(platformCandidates()) == 0 {
		t.Error("every platform has somewhere to look")
	}
}
