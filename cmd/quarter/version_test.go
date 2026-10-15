package main

import (
	"runtime/debug"
	"testing"
)

// A release build keeps its tag; go install gets the module version Go
// recorded (it printed "dev" before, found 2026-10-15); a build Go knew no
// version for stays dev.
func TestPickVersion(t *testing.T) {
	built := func(v string) *debug.BuildInfo { return &debug.BuildInfo{Main: debug.Module{Version: v}} }
	cases := []struct {
		set  string
		info *debug.BuildInfo
		ok   bool
		want string
	}{
		{"v0.2.0", built("v0.1.0"), true, "v0.2.0"},
		{"dev", built("v0.2.0"), true, "v0.2.0"},
		{"dev", built("(devel)"), true, "dev"},
		{"dev", built(""), true, "dev"},
		{"dev", nil, false, "dev"},
	}
	for _, c := range cases {
		if got := pickVersion(c.set, c.info, c.ok); got != c.want {
			t.Errorf("pickVersion(%q, %v, %v) = %q, want %q", c.set, c.info != nil, c.ok, got, c.want)
		}
	}
}
