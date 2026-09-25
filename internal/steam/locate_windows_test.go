//go:build windows

package steam

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Steam's registry keeps its path lowercased; Quarter shows the real casing.
func TestRealCase(t *testing.T) {
	want := filepath.Join(realCase(t.TempDir()), "Program Files (x86)", "Steam")
	if err := os.MkdirAll(want, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := realCase(strings.ToLower(want)); got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	missing := strings.ToLower(filepath.Join(want, "Nope"))
	if got := realCase(missing); got != missing {
		t.Errorf("a missing folder should come back unchanged, got %q", got)
	}
}
