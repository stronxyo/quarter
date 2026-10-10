package ui

import (
	"strings"
	"testing"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
)

// Every logo row is as wide as the others, so the box centers them as one.
func TestLogoRowsAlign(t *testing.T) {
	for i, row := range logoArt {
		if w, want := ansi.StringWidth(row), ansi.StringWidth(logoArt[0]); w != want {
			t.Errorf("row %d is %d wide, want %d", i, w, want)
		}
	}
}

// The letters take the row's gradient color and the shadow the dim one, and
// the text itself comes through unchanged.
func TestTwoTone(t *testing.T) {
	letter := lipgloss.NewStyle().Foreground(lipgloss.Color("#FF0000"))
	row := " ██┐ ██│"
	got := twoTone(row, letter)
	if ansi.Strip(got) != row {
		t.Fatalf("text changed: %q", ansi.Strip(got))
	}
	if want := letter.Render("██") + dimStyle.Render("┐") + " "; !strings.HasPrefix(got, " "+want) {
		t.Errorf("got %q, want letters then a dim shadow", got)
	}
}

// The quarter_ box's letters sit in its middle: as many rows between the
// top edge and the t's top as between the q's bottom and the bottom edge.
func TestLogoCentredInBox(t *testing.T) {
	lines := strings.Split(ansi.Strip(logoBox()), "\n")
	first, last := -1, -1
	for i, l := range lines {
		if strings.Contains(l, "█") {
			if first < 0 {
				first = i
			}
			last = i
		}
	}
	if above, below := first-1, len(lines)-2-last; above != below {
		t.Errorf("%d rows above the letters, %d below; want equal", above, below)
	}
}
