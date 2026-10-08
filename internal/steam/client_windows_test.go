//go:build windows

package steam

import (
	"os"
	"os/exec"
	"testing"
)

// A pid left in the registry after Steam quit, or reused by another
// program, is not Steam.
func TestLiveSteam(t *testing.T) {
	if liveSteam(uint32(os.Getpid())) {
		t.Error("a live process that isn't steam.exe counted as Steam")
	}
	cmd := exec.Command("cmd", "/c", "exit")
	if err := cmd.Run(); err != nil {
		t.Skip("no cmd to make a finished process:", err)
	}
	if liveSteam(uint32(cmd.ProcessState.Pid())) {
		t.Error("a finished process counted as Steam")
	}
}
