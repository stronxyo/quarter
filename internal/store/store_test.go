package store

import (
	"os"
	"path/filepath"
	"testing"
)

type sample struct {
	Names []string
	Count int
}

func TestRoundTrip(t *testing.T) {
	dir := t.TempDir()
	want := sample{Names: []string{"Portal", "Portal 2"}, Count: 2}
	if err := Save(dir, "lib.json", 1, want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, ok := Load[sample](dir, "lib.json", 1)
	if !ok || got.Count != 2 || len(got.Names) != 2 || got.Names[1] != "Portal 2" {
		t.Errorf("got %+v ok=%v, want %+v", got, ok, want)
	}
}

func TestLoadNoCache(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "corrupt.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Save(dir, "old.json", 1, sample{Count: 1}); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name, file string
	}{
		{"missing", "missing.json"},
		{"corrupt", "corrupt.json"},
		{"older version", "old.json"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, ok := Load[sample](dir, tt.file, 2); ok {
				t.Error("want ok=false")
			}
		})
	}
}

func TestFailedSaveKeepsOldCache(t *testing.T) {
	dir := t.TempDir()
	if err := Save(dir, "lib.json", 1, sample{Count: 1}); err != nil {
		t.Fatal(err)
	}

	// A channel can't be encoded, so this write fails before touching the file.
	if err := Save(dir, "lib.json", 1, make(chan int)); err == nil {
		t.Fatal("want an error saving an unencodable value")
	}

	got, ok := Load[sample](dir, "lib.json", 1)
	if !ok || got.Count != 1 {
		t.Errorf("old cache lost: got %+v ok=%v", got, ok)
	}
	if leftovers, _ := filepath.Glob(filepath.Join(dir, "*.tmp")); len(leftovers) != 0 {
		t.Errorf("temporary files left behind: %v", leftovers)
	}
}

func TestSaveReplacesExisting(t *testing.T) {
	dir := t.TempDir()
	for i := 1; i <= 2; i++ {
		if err := Save(dir, "lib.json", 1, sample{Count: i}); err != nil {
			t.Fatalf("Save %d: %v", i, err)
		}
	}
	if got, _ := Load[sample](dir, "lib.json", 1); got.Count != 2 {
		t.Errorf("got Count %d, want 2", got.Count)
	}
}
