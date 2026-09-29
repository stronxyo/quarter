// Package store saves small JSON files in Quarter's data folder: the library
// cache and the settings. Load treats anything missing, corrupt or from an
// older format as "not there"; callers decide what that means.
package store

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

type envelope[T any] struct {
	Version int `json:"version"`
	Data    T   `json:"data"`
}

// Load reads dir/name. ok is false when the file is missing, unreadable, or
// was written with a different version.
func Load[T any](dir, name string, version int) (data T, ok bool) {
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return data, false
	}
	var env envelope[T]
	if err := json.Unmarshal(b, &env); err != nil || env.Version != version {
		return data, false
	}
	return env.Data, true
}

// SetAside renames dir/name to name.corrupt, if it exists, so a file Load
// couldn't read is kept for the user instead of being overwritten. moved
// reports whether there was a file.
func SetAside(dir, name string) (moved bool, err error) {
	path := filepath.Join(dir, name)
	if _, err := os.Stat(path); err != nil {
		return false, nil
	}
	if err := os.Rename(path, path+".corrupt"); err != nil {
		return true, fmt.Errorf("store %s: %w", name, err)
	}
	return true, nil
}

// Save writes data to dir/name as JSON, through WriteFile.
func Save[T any](dir, name string, version int, data T) error {
	b, err := json.Marshal(envelope[T]{Version: version, Data: data})
	if err != nil {
		return fmt.Errorf("store %s: %w", name, err)
	}
	return WriteFile(dir, name, b)
}

// WriteFile writes b to dir/name through a temporary file and a rename, so a
// crash mid-write never leaves a half-written file behind.
func WriteFile(dir, name string, b []byte) error {
	tmp, err := os.CreateTemp(dir, name+".*.tmp")
	if err != nil {
		return fmt.Errorf("store %s: %w", name, err)
	}
	defer os.Remove(tmp.Name()) // no-op once renamed

	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return fmt.Errorf("store %s: %w", name, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("store %s: %w", name, err)
	}
	if err := rename(tmp.Name(), filepath.Join(dir, name)); err != nil {
		return fmt.Errorf("store %s: %w", name, err)
	}
	return nil
}
