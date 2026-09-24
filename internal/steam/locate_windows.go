//go:build windows

package steam

import "golang.org/x/sys/windows/registry"

func platformCandidates() []string {
	var dirs []string
	if k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Valve\Steam`, registry.QUERY_VALUE); err == nil {
		if path, _, err := k.GetStringValue("SteamPath"); err == nil && path != "" {
			dirs = append(dirs, path)
		}
		k.Close()
	}
	return append(dirs, `C:\Program Files (x86)\Steam`)
}
