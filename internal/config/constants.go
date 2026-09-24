// Package config holds app-wide paths, settings and constants.
package config

import "time"

// AppName names the binary and the app data folder.
const AppName = "quarter"

// SteamPathEnv overrides Steam detection when set.
const SteamPathEnv = "QUARTER_STEAM_PATH"

// NonGameAppIDs are Steam tools that install like games but are not games.
// Source: steamcommunity.com/sharedfiles/filedetails/?id=2013973376 (2026-09-24).
var NonGameAppIDs = map[uint32]string{
	228980:  "Steamworks Common Redistributables",
	250820:  "SteamVR",
	1070560: "Steam Linux Runtime 1.0 (scout)",
	1391110: "Steam Linux Runtime 2.0 (soldier)",
	1628350: "Steam Linux Runtime 3.0 (sniper)",
	4183110: "Steam Linux Runtime 4.0",
	4185400: "Steam Linux Runtime 4.0 ARM64",
	4690330: "Legacy Steam Runtime (scout)",
	1161040: "Proton BattlEye Runtime",
	1826330: "Proton EasyAntiCheat Runtime",
	3086180: "Proton Voice Files",
	2230260: "Proton Next",
	2180100: "Proton Hotfix",
	1493710: "Proton Experimental",
	4628710: "Proton 11.0",
	4628740: "Proton 11.0 (ARM64)",
	3658110: "Proton 10.0",
	2805730: "Proton 9.0",
	2348590: "Proton 8.0",
	1887720: "Proton 7.0",
	1580130: "Proton 6.3",
	1420170: "Proton 5.13",
	1245040: "Proton 5.0",
	1113280: "Proton 4.11",
	1054830: "Proton 4.2",
	996510:  "Proton 3.16 beta",
	961940:  "Proton 3.16",
	930400:  "Proton 3.7 beta",
	858280:  "Proton 3.7",
}

// LogFile is written in DataDir while the UI runs, since the terminal is taken.
const LogFile = "quarter.log"

// RecentWindow is how far back the Recent filter looks.
const RecentWindow = 14 * 24 * time.Hour

// LibraryCache is the scanned library saved in DataDir, shown instantly on the
// next launch. Bump LibraryCacheVersion whenever the cached fields change.
const (
	LibraryCache        = "library.json"
	LibraryCacheVersion = 1
)
