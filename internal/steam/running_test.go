package steam

import (
	"runtime"
	"testing"
)

// The real value depends on what is running, so this only checks that the
// Windows registry answers and other platforms say "unknown".
func TestRunningApp(t *testing.T) {
	_, known := RunningApp()
	if runtime.GOOS != "windows" && known {
		t.Error("only Windows can know the running game")
	}
}
