//go:build windows

package steam

import (
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// platformCandidates are the places Steam is found on Windows: where the
// user's Steam says it is, where its installer put it (machine-wide), and
// the installer's default.
func platformCandidates() []string {
	var dirs []string
	add := func(root registry.Key, path, name string) {
		k, err := registry.OpenKey(root, path, registry.QUERY_VALUE)
		if err != nil {
			return
		}
		defer k.Close()
		if dir, _, err := k.GetStringValue(name); err == nil && dir != "" {
			dirs = append(dirs, dir)
		}
	}
	add(registry.CURRENT_USER, `Software\Valve\Steam`, "SteamPath")
	add(registry.LOCAL_MACHINE, `SOFTWARE\WOW6432Node\Valve\Steam`, "InstallPath")
	return append(dirs, `C:\Program Files (x86)\Steam`)
}

// realCase returns dir spelled the way the file system spells it: Steam keeps
// its own path lowercased in the registry. Only the casing changes; when
// Windows resolves dir to somewhere else (a junction or a substituted
// drive), dir is kept as it is.
func realCase(dir string) string {
	p, err := windows.UTF16PtrFromString(dir)
	if err != nil {
		return dir
	}
	h, err := windows.CreateFile(p, 0, windows.FILE_SHARE_READ|windows.FILE_SHARE_WRITE|windows.FILE_SHARE_DELETE,
		nil, windows.OPEN_EXISTING, windows.FILE_FLAG_BACKUP_SEMANTICS, 0) // backup semantics opens folders
	if err != nil {
		return dir
	}
	defer windows.CloseHandle(h)
	buf := make([]uint16, windows.MAX_LONG_PATH)
	n, err := windows.GetFinalPathNameByHandle(h, &buf[0], uint32(len(buf)), 0)
	if err != nil || int(n) > len(buf) {
		return dir
	}
	real := strings.TrimPrefix(windows.UTF16ToString(buf[:n]), `\\?\`)
	if !strings.EqualFold(real, dir) {
		return dir
	}
	return real
}
