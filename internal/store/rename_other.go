//go:build !windows

package store

import "os"

// rename replaces newpath with oldpath; only Windows needs retries.
func rename(oldpath, newpath string) error { return os.Rename(oldpath, newpath) }
