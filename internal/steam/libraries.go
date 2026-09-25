package steam

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/stronxyo/quarter/internal/vdf"
)

// Libraries returns every Steam library folder, starting with the install
// itself. If libraryfolders.vdf can't be read, it still returns the install
// folder, together with the error as a warning.
func Libraries(steamPath string) ([]string, error) {
	libs := []string{filepath.Clean(steamPath)}

	root, err := readVDF(filepath.Join(steamPath, "steamapps", "libraryfolders.vdf"))
	if err != nil {
		return libs, err
	}
	return dedupe(append(libs, libraryPaths(root, steamPath)...)), nil
}

func libraryPaths(root *vdf.Node, steamPath string) []string {
	var paths []string
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
		paths = append(paths, realCase(filepath.Clean(path)))
	}
	return paths
}

// dedupe drops repeated paths, ignoring case, keeping the first occurrence.
func dedupe(paths []string) []string {
	seen := make(map[string]bool, len(paths))
	out := paths[:0]
	for _, p := range paths {
		k := strings.ToLower(p)
		if !seen[k] {
			seen[k] = true
			out = append(out, p)
		}
	}
	return out
}

func readVDF(path string) (*vdf.Node, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	root, err := vdf.Parse(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return root, nil
}
