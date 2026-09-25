//go:build windows

package steam

import (
	"strings"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

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
