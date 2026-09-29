package steam

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/vdf"
)

// LibraryFolder is one Steam library folder, and how many games Steam says
// are installed in it (from libraryfolders.vdf, so known even when the
// folder's drive isn't connected).
type LibraryFolder struct {
	Path  string
	Games int
}

// Libraries returns every Steam library folder's path, starting with the
// install itself. If libraryfolders.vdf can't be read, it still returns the
// install folder, together with the error as a warning.
func Libraries(steamPath string) ([]string, error) {
	folders, err := LibraryFolders(steamPath)
	paths := make([]string, len(folders))
	for i, f := range folders {
		paths[i] = f.Path
	}
	return paths, err
}

// LibraryFolders returns every Steam library folder, starting with the
// install itself. Steam keeps libraryfolders.vdf in steamapps, and a copy in
// config, which is read when the first can't be. If neither can, only the
// install folder is returned, with the first file's error as a warning.
func LibraryFolders(steamPath string) ([]LibraryFolder, error) {
	folders := []LibraryFolder{{Path: filepath.Clean(steamPath)}}
	root, err := readVDF(filepath.Join(steamPath, "steamapps", "libraryfolders.vdf"))
	if err != nil {
		copyRoot, copyErr := readVDF(filepath.Join(steamPath, "config", "libraryfolders.vdf"))
		if copyErr != nil {
			return folders, err
		}
		root = copyRoot
	}
	return dedupe(append(folders, libraryEntries(root, steamPath)...)), nil
}

func libraryEntries(root *vdf.Node, steamPath string) []LibraryFolder {
	var folders []LibraryFolder
	for _, entry := range root.Get("libraryfolders").Entries() {
		path := entry.Value // old format: "1" "D:\\Games"
		if p := entry.Get("path"); p != nil {
			path = p.Value // current format: "1" { "path" "D:\\Games" ... }
		}
		if path == "" {
			continue
		}
		// Steam writes absolute paths; relative ones (used by the test
		// fixture) resolve against the install.
		if !filepath.IsAbs(path) {
			path = filepath.Join(steamPath, path)
		}
		games := 0
		for _, app := range entry.Get("apps").Entries() {
			if id, err := strconv.ParseUint(app.Key, 10, 32); err == nil && IsGame(uint32(id)) {
				games++
			}
		}
		folders = append(folders, LibraryFolder{Path: realCase(filepath.Clean(path)), Games: games})
	}
	return folders
}

// dedupe drops repeated paths, ignoring case, keeping the first occurrence
// and the largest game count seen for it.
func dedupe(folders []LibraryFolder) []LibraryFolder {
	seen := make(map[string]int, len(folders))
	out := folders[:0]
	for _, f := range folders {
		k := strings.ToLower(f.Path)
		if i, ok := seen[k]; ok {
			out[i].Games = max(out[i].Games, f.Games)
			continue
		}
		seen[k] = len(out)
		out = append(out, f)
	}
	return out
}

// connected reports whether a library folder is there to read: a library
// on a drive that is unplugged or unmounted isn't.
func (f LibraryFolder) connected() bool {
	info, err := os.Stat(filepath.Join(f.Path, "steamapps"))
	return err == nil && info.IsDir()
}

// readRetry is how long readVDF waits before reading a file a second time:
// Steam may have been writing it.
var readRetry = func() { time.Sleep(config.SteamReadRetry) }

// errEmpty means a Steam file exists but has nothing in it.
var errEmpty = errors.New("empty file")

// readVDF parses a Steam file. A file that can't be read or parsed, or is
// empty, is read once more after a short wait, since Steam may be writing
// it; a missing file isn't.
func readVDF(path string) (*vdf.Node, error) {
	root, err := readVDFOnce(path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		readRetry()
		root, err = readVDFOnce(path)
	}
	if err != nil {
		return nil, &FileError{Name: filepath.Base(path), Err: err}
	}
	return root, nil
}

// FileError is a Steam file that couldn't be read, named by its file name,
// so the UI can say which one.
type FileError struct {
	Name string
	Err  error
}

func (e *FileError) Error() string { return e.Name + ": " + e.Err.Error() }
func (e *FileError) Unwrap() error { return e.Err }

func readVDFOnce(path string) (*vdf.Node, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(strings.TrimSpace(string(b))) == 0 {
		return nil, errEmpty
	}
	return vdf.Parse(strings.NewReader(string(b)))
}
