package main

import "testing"

// config.json's "glyphs" wins; empty asks Windows whether this is the
// classic console (never true off Windows, nor under go test, which has no
// console window of its own when its output is piped).
func TestBasicGlyphsSetting(t *testing.T) {
	if !basicGlyphs("basic") || basicGlyphs("full") {
		t.Error(`"basic" and "full" should decide`)
	}
	if basicGlyphs("") != classicConsole() || basicGlyphs("something else") != classicConsole() {
		t.Error("anything else should leave it to the console")
	}
}
