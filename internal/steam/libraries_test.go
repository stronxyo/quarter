package steam

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/stronxyo/quarter/internal/vdf"
)

func TestLibrariesFixture(t *testing.T) {
	got, err := Libraries(fixture)
	if err != nil {
		t.Fatalf("Libraries: %v", err)
	}
	want := []string{filepath.Clean(fixture), filepath.Clean("../../testdata/library2")}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestLibrariesMissingFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, "steamapps"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Libraries(dir)
	if err == nil {
		t.Error("want a warning for the missing libraryfolders.vdf")
	}
	if len(got) != 1 || got[0] != filepath.Clean(dir) {
		t.Errorf("got %q, want only the install folder", got)
	}
}

func TestLibraryPaths(t *testing.T) {
	steamPath := filepath.Join(os.TempDir(), "steam")
	abs := func(p string) string { return filepath.Join(os.TempDir(), p) }

	tests := []struct {
		name string
		in   string
		want []string
	}{
		{
			name: "current format",
			in:   `"libraryfolders" { "0" { "path" "` + escape(abs("games")) + `" } }`,
			want: []string{abs("games")},
		},
		{
			name: "old format",
			in:   `"LibraryFolders" { "TimeNextStatsReport" "" "1" "` + escape(abs("old")) + `" }`,
			want: []string{abs("old")},
		},
		{
			name: "entry without path skipped",
			in:   `"libraryfolders" { "0" { "label" "x" } }`,
			want: nil,
		},
		{
			name: "missing block",
			in:   `"other" { }`,
			want: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root, err := vdf.Parse(strings.NewReader(tt.in))
			if err != nil {
				t.Fatalf("Parse: %v", err)
			}
			if got := libraryPaths(root, steamPath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestDedupe(t *testing.T) {
	got := dedupe([]string{`C:\Steam`, `D:\Games`, `c:\steam`})
	want := []string{`C:\Steam`, `D:\Games`}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

// escape doubles backslashes the way Steam writes Windows paths in .vdf files.
func escape(p string) string { return strings.ReplaceAll(p, `\`, `\\`) }
