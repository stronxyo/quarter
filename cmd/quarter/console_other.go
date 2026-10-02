//go:build !windows

package main

// colourOutput reports that colour codes can be printed: on macOS and Linux
// every terminal reads them, with nothing to switch on (see the Windows
// version for the classic console).
func colourOutput() (restore func(), ok bool) {
	return func() {}, true
}
