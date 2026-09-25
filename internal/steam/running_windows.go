//go:build windows

package steam

import "golang.org/x/sys/windows/registry"

// runningApp reads HKCU\Software\Valve\Steam\RunningAppID, which Steam keeps
// at the running game's appid and resets to 0 when it quits.
func runningApp() (uint32, bool) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Valve\Steam`, registry.QUERY_VALUE)
	if err != nil {
		return 0, false
	}
	defer k.Close()
	id, _, err := k.GetIntegerValue("RunningAppID")
	if err != nil {
		return 0, false
	}
	return uint32(id), true
}
