//go:build !windows

package steam

// runningApp is unknown here: Steam for Linux stopped writing RunningAppID to
// registry.vdf in 2023 (ValveSoftware/steam-for-linux#9672, still open), and
// a wrong answer would be worse than none.
func runningApp() (uint32, bool) {
	return 0, false
}
