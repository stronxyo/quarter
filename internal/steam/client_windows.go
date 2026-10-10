//go:build windows

package steam

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/windows"
	"golang.org/x/sys/windows/registry"
)

// stillActive is the exit code Windows reports for a process still running.
const stillActive = 259

// clientState reads HKCU\Software\Valve\Steam\ActiveProcess. Steam writes
// its pid there as it starts but leaves it after quitting (seen on Pablo's
// PC, 2026-10-07), so the pid only counts while it is a live steam.exe.
// ActiveUser is 0 while Steam signs in and the account's id once it has
// (measured 2026-10-08: 0 from 1.6 s after the start, the account 26.6 s
// after); a stale id stays there between runs, so it is read only with a
// live Steam.
func clientState() (Client, bool) {
	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Valve\Steam\ActiveProcess`, registry.QUERY_VALUE)
	if err != nil {
		return Client{}, true // Steam has never run for this user
	}
	defer k.Close()
	pid, _, err := k.GetIntegerValue("pid")
	if err != nil || pid == 0 || !liveSteam(uint32(pid)) {
		return Client{}, true
	}
	user, _, err := k.GetIntegerValue("ActiveUser")
	return Client{Running: true, SignedIn: err == nil && user != 0}, true
}

// liveSteam reports whether pid is a running steam.exe.
func liveSteam(pid uint32) bool {
	h, err := windows.OpenProcess(windows.PROCESS_QUERY_LIMITED_INFORMATION, false, pid)
	if err != nil {
		return false
	}
	defer windows.CloseHandle(h)
	var code uint32
	if err := windows.GetExitCodeProcess(h, &code); err != nil || code != stillActive {
		return false
	}
	buf := make([]uint16, windows.MAX_LONG_PATH)
	n := uint32(len(buf))
	if err := windows.QueryFullProcessImageName(h, 0, &buf[0], &n); err != nil {
		return false
	}
	return isSteamImage(windows.UTF16ToString(buf[:n]))
}

// startClient runs Steam's client with args, from its own folder, the way
// its shortcut does, and doesn't wait for it. It runs detached from the
// terminal, so closing the terminal leaves Steam running.
func startClient(steamPath string, args ...string) error {
	exe, err := clientExe(steamPath)
	if err != nil {
		return err
	}
	cmd := exec.Command(exe, args...)
	cmd.Dir = filepath.Dir(exe)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.DETACHED_PROCESS | windows.CREATE_NEW_PROCESS_GROUP,
	}
	if err := cmd.Start(); err != nil {
		return err
	}
	return cmd.Process.Release()
}

// clientExe is steam.exe: in the --steam-path folder when given, else where
// Steam's registry says (SteamExe), else in the folder Locate finds.
func clientExe(steamPath string) (string, error) {
	var candidates []string
	if steamPath != "" {
		candidates = append(candidates, filepath.Join(steamPath, "steam.exe"))
	} else {
		if k, err := registry.OpenKey(registry.CURRENT_USER, `Software\Valve\Steam`, registry.QUERY_VALUE); err == nil {
			if exe, _, err := k.GetStringValue("SteamExe"); err == nil && exe != "" {
				candidates = append(candidates, filepath.FromSlash(exe))
			}
			k.Close()
		}
		if dir, err := Locate(""); err == nil {
			candidates = append(candidates, filepath.Join(dir, "steam.exe"))
		}
	}
	for _, exe := range candidates {
		if info, err := os.Stat(exe); err == nil && !info.IsDir() {
			return exe, nil
		}
	}
	return "", errors.New("steam.exe not found")
}
