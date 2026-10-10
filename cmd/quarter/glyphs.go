package main

// basicGlyphs reports whether Quarter should draw with the stand-ins the
// classic Windows console's fonts have (the UI's glyphs.go): config.json's
// "glyphs" decides when set ("basic" or "full"); otherwise it is the
// classic console itself, asked of Windows (console_windows.go). Not an
// environment variable: a console opened from Windows Terminal inherits
// its WT_SESSION (found 2026-10-02).
func basicGlyphs(setting string) bool {
	switch setting {
	case "basic":
		return true
	case "full":
		return false
	}
	return classicConsole()
}
