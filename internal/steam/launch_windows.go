//go:build windows

package steam

import "golang.org/x/sys/windows"

// OpenURI hands a link to Windows, which passes steam:// links to Steam.
// ShellExecute starts no console process, so nothing flashes on screen.
func OpenURI(uri string) error {
	verb, err := windows.UTF16PtrFromString("open")
	if err != nil {
		return err
	}
	file, err := windows.UTF16PtrFromString(uri)
	if err != nil {
		return err
	}
	return windows.ShellExecute(0, verb, file, nil, nil, windows.SW_SHOWNORMAL)
}
