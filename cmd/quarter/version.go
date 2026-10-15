package main

import "runtime/debug"

// version is set at release time with -ldflags "-X main.version=<tag>".
var version = "dev"

// appVersion is what quarter --version prints: the tag a release build was
// given, else the version Go records in the program (go install of v0.2.0
// records v0.2.0), else dev.
func appVersion() string {
	info, ok := debug.ReadBuildInfo()
	return pickVersion(version, info, ok)
}

// pickVersion keeps a version set at build time; without one it takes the
// main module's version from the build info, unless Go knew none ("(devel)").
func pickVersion(set string, info *debug.BuildInfo, ok bool) string {
	if set != "dev" || !ok || info == nil {
		return set
	}
	if v := info.Main.Version; v != "" && v != "(devel)" {
		return v
	}
	return set
}
