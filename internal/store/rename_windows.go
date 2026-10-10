//go:build windows

package store

import (
	"errors"
	"os"
	"time"

	"golang.org/x/sys/windows"
)

// renameTries and renameWait give a file held for a moment by antivirus or
// the search indexer about a second to be let go (golang/go#29106).
const (
	renameTries = 40
	renameWait  = 25 * time.Millisecond
)

// rename replaces newpath with oldpath, retrying while Windows reports the
// file in use; any other error returns at once.
func rename(oldpath, newpath string) error {
	var err error
	for range renameTries {
		err = os.Rename(oldpath, newpath)
		if err == nil || !(errors.Is(err, windows.ERROR_ACCESS_DENIED) || errors.Is(err, windows.ERROR_SHARING_VIOLATION)) {
			return err
		}
		time.Sleep(renameWait)
	}
	return err
}
