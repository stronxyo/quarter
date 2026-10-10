//go:build windows

package steam

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"golang.org/x/sys/windows"
)

// Steam's registry keeps its path lowercased; Quarter shows the real casing.
func TestRealCase(t *testing.T) {
	// The temp folder can hold a short 8.3 name (GitHub's runners: RUNNER~1),
	// which realCase leaves as it is; the test works under its long name.
	want := filepath.Join(realCase(longName(t, t.TempDir())), "Program Files (x86)", "Steam")
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

// longName expands any short 8.3 names in path (GetLongPathNameW).
func longName(t *testing.T, path string) string {
	t.Helper()
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	buf := make([]uint16, windows.MAX_LONG_PATH)
	n, err := windows.GetLongPathName(p, &buf[0], uint32(len(buf)))
	if err != nil || int(n) > len(buf) {
		t.Fatalf("long name of %q: %v", path, err)
	}
	return windows.UTF16ToString(buf[:n])
}
