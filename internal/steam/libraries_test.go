package steam

import (
	"errors"
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
	mkdir(t, dir, "steamapps")
	got, err := Libraries(dir)
	if err == nil {
		t.Error("want a warning for the missing libraryfolders.vdf")
	}
	if len(got) != 1 || got[0] != filepath.Clean(dir) {
		t.Errorf("got %q, want only the install folder", got)
	}
}

func TestLibraryEntries(t *testing.T) {
	steamPath := filepath.Join(os.TempDir(), "steam")
	abs := func(p string) string { return filepath.Join(os.TempDir(), p) }
	tests := []struct {
		name string
		in   string
		want []LibraryFolder
	}{
		{
			name: "current format, counting games but not Steam tools",
			in:   `"libraryfolders" { "0" { "path" "` + escape(abs("games")) + `" "apps" { "620" "1" "228980" "2" "400" "3" } } }`,
			want: []LibraryFolder{{Path: abs("games"), Games: 2}},
		},
		{
			name: "old format",
			in:   `"LibraryFolders" { "TimeNextStatsReport" "" "1" "` + escape(abs("old")) + `" }`,
			want: []LibraryFolder{{Path: abs("old")}},
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
			if got := libraryEntries(root, steamPath); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("got %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestDedupe(t *testing.T) {
	got := dedupe([]LibraryFolder{{Path: `C:\Steam`}, {Path: `D:\Games`, Games: 1}, {Path: `c:\steam`, Games: 4}})
	want := []LibraryFolder{{Path: `C:\Steam`, Games: 4}, {Path: `D:\Games`, Games: 1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// A library on a drive that isn't connected is reported with its game
// count, and the rest of the library still reads.
func TestInstalledGamesMissingDrive(t *testing.T) {
	steam := t.TempDir()
	gone := filepath.Join(t.TempDir(), "unplugged")
	write(t, steam, "steamapps/libraryfolders.vdf",
		`"libraryfolders" { "0" { "path" "`+escape(steam)+`" } "1" { "path" "`+escape(gone)+`" "apps" { "10" "1" "20" "1" } } }`)
	write(t, steam, "steamapps/appmanifest_620.acf", `"AppState" { "appid" "620" "name" "Portal 2" "StateFlags" "4" }`)
	games, missing, warnings := InstalledGames(steam)
	if len(games) != 1 || len(warnings) != 0 {
		t.Errorf("the connected library should read cleanly: %v %v", games, warnings)
	}
	if len(missing) != 1 || missing[0].Path != gone || missing[0].Games != 2 {
		t.Errorf("missing: got %+v, want %s with 2 games", missing, gone)
	}
}

// Steam keeps a copy of libraryfolders.vdf in config; it is read when the
// one in steamapps is damaged.
func TestLibraryFoldersConfigCopy(t *testing.T) {
	steam := t.TempDir()
	other := t.TempDir()
	noRetry(t)
	write(t, steam, "steamapps/libraryfolders.vdf", `"libraryfolders" { "0" { "path" `) // cut off
	write(t, steam, "config/libraryfolders.vdf", `"libraryfolders" { "1" { "path" "`+escape(other)+`" } }`)
	got, err := LibraryFolders(steam)
	if err != nil {
		t.Fatalf("the copy should have been read: %v", err)
	}
	if len(got) != 2 || got[1].Path != realCase(filepath.Clean(other)) {
		t.Errorf("got %+v, want the install and %s", got, other)
	}
}

func TestReadVDF(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.vdf")

	// Empty: unreadable, even after the second try.
	noRetry(t)
	write(t, dir, "a.vdf", "  \r\n")
	_, err := readVDF(path)
	var fe *FileError
	if !errors.As(err, &fe) || fe.Name != "a.vdf" || !errors.Is(err, errEmpty) {
		t.Errorf("an empty file: got %v, want a FileError for a.vdf", err)
	}

	// Being written: the second read finds it complete.
	write(t, dir, "a.vdf", `"a" { "b" `)
	readRetry = func() { write(t, dir, "a.vdf", `"a" { "b" "c" }`) }
	root, err := readVDF(path)
	if err != nil || root.Path("a").ValueOf("b") != "c" {
		t.Errorf("the second read should succeed: %v", err)
	}

	// Missing: no second try.
	readRetry = func() { t.Error("a missing file shouldn't be read twice") }
	if _, err := readVDF(filepath.Join(dir, "none.vdf")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("got %v, want ErrNotExist", err)
	}
}

// noRetry makes readVDF's second read immediate for the test.
func noRetry(t *testing.T) {
	saved := readRetry
	readRetry = func() {}
	t.Cleanup(func() { readRetry = saved })
}

func mkdir(t *testing.T, dir, name string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, name), 0o755); err != nil {
		t.Fatal(err)
	}
}

func write(t *testing.T, dir, name, text string) {
	t.Helper()
	path := filepath.Join(dir, filepath.FromSlash(name))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// escape doubles backslashes the way Steam writes Windows paths in .vdf files.
func escape(p string) string { return strings.ReplaceAll(p, `\`, `\\`) }
