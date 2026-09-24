// Package cache stores small JSON files in Quarter's data folder. A cache is
// only ever a speed-up: anything missing, corrupt or from an older format is
// treated as "no cache", never as an error the user sees.
package cache

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

// Save writes dir/name through a temporary file and a rename, so a crash
// mid-write never leaves a half-written cache behind.
func Save[T any](dir, name string, version int, data T) error {
	b, err := json.Marshal(envelope[T]{Version: version, Data: data})
	if err != nil {
		return fmt.Errorf("cache %s: %w", name, err)
	}

	tmp, err := os.CreateTemp(dir, name+".*.tmp")
	if err != nil {
		return fmt.Errorf("cache %s: %w", name, err)
	}
	defer os.Remove(tmp.Name()) // no-op once renamed

	if _, err := tmp.Write(b); err != nil {
		tmp.Close()
		return fmt.Errorf("cache %s: %w", name, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("cache %s: %w", name, err)
	}
	if err := os.Rename(tmp.Name(), filepath.Join(dir, name)); err != nil {
		return fmt.Errorf("cache %s: %w", name, err)
	}
	return nil
}
