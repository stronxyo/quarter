package art

import (
	"strings"
	"testing"
	"unicode/utf8"
)

// Every glyph is 7 rows of the same width, drawn with pixels only.
func TestGlyphs(t *testing.T) {
	for r, g := range glyphs {
		for y, row := range g {
			if len(row) != len(g[0]) || strings.Trim(row, ".#") != "" {
				t.Errorf("%q row %d: %q", r, y, row)
			}
		}
	}
}

func TestNameFits(t *testing.T) {
	lines, ok := Name("Portal 2", 64, 10)
	if !ok || len(lines) != 4 {
		t.Fatalf("one line of text is 4 rows, got %d (ok %v)", len(lines), ok)
	}
	// P O R T A L, space, 2: six 5-wide letters and a digit, with gaps.
	if w := utf8.RuneCountInString(lines[0]); w != 7*5+6+spaceCols+1 {
		t.Errorf("width %d", w)
	}
	if strings.Trim(strings.Join(lines, ""), "█▀▄ ") != "" {
		t.Error("only half blocks and spaces expected")
	}
}

func TestNameWraps(t *testing.T) {
	lines, ok := Name("They Call It Gravity", 64, 10)
	if !ok || len(lines) != 8 {
		t.Fatalf("two lines of text are 8 rows, got %d (ok %v)", len(lines), ok)
	}
	for _, l := range lines {
		if w := utf8.RuneCountInString(l); w > 64 {
			t.Errorf("a row is %d wide, over 64", w)
		}
	}
}

func TestNameDoesNotFit(t *testing.T) {
	for _, tt := range []struct {
		text       string
		cols, rows int
	}{
		{"They Call It Gravity", 64, 7},  // two lines need 8 rows
		{"Supercalifragilistic", 40, 14}, // one word wider than the pane
		{"™ ©", 64, 14},                  // nothing the font can draw
		{"", 64, 14},
	} {
		if _, ok := Name(tt.text, tt.cols, tt.rows); ok {
			t.Errorf("%q in %dx%d should not fit", tt.text, tt.cols, tt.rows)
		}
	}
}

// Accents map to their letters, and marks the font lacks are left out.
func TestNameCleansText(t *testing.T) {
	a, _ := Name("Pokémon™", 64, 10)
	b, _ := Name("POKEMON", 64, 10)
	if strings.Join(a, "\n") != strings.Join(b, "\n") {
		t.Error("Pokémon™ should draw as POKEMON")
	}
}
