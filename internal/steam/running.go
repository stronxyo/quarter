package steam

// RunningApp returns the appid of the game Steam is running, 0 when none is.
// known is false where Steam doesn't publish it (see running_other.go).
func RunningApp() (appID uint32, known bool) {
	return runningApp()
}
