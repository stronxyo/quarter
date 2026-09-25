//go:build !windows

package steam

import (
	"os/exec"
	"runtime"
)

// OpenURI hands a link to the desktop, which passes steam:// links to Steam.
func OpenURI(uri string) error {
	opener := "xdg-open"
	if runtime.GOOS == "darwin" {
		opener = "open"
	}
	cmd := exec.Command(opener, uri)
	if err := cmd.Start(); err != nil {
		return err
	}
	go cmd.Wait() //nolint:errcheck // the opener exits once it hands off; only reap it
	return nil
}
