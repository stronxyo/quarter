//go:build !windows

package steam

import "errors"

// clientState is unknown here, so Quarter keeps opening steam:// links as
// before: no one has tried -silent on Linux or macOS for Quarter yet, and on
// macOS "open -a Steam --args" drops the arguments once Steam is running.
func clientState() (Client, bool) {
	return Client{}, false
}

func startClient(string, ...string) error {
	return errors.New("starting Steam quietly is not available here")
}
