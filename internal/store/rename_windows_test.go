//go:build windows

package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"golang.org/x/sys/windows"
)

// A file held open for a moment (as antivirus does) is replaced once it is
// let go, instead of the save failing.
func TestWriteFileWaitsForHeldFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.json")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	p, err := windows.UTF16PtrFromString(path)
	if err != nil {
		t.Fatal(err)
	}
	// Shared for reading only: a rename over it fails while it is open.
	h, err := windows.CreateFile(p, windows.GENERIC_READ, windows.FILE_SHARE_READ, nil, windows.OPEN_EXISTING, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	released := make(chan struct{})
	go func() {
		time.Sleep(150 * time.Millisecond)
		windows.CloseHandle(h)
		close(released)
	}()
	if err := WriteFile(dir, "a.json", []byte("new")); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	<-released
	if b, _ := os.ReadFile(path); string(b) != "new" {
		t.Errorf("got %q, want new", b)
	}
}
