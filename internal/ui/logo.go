package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// logoArt is "quarter_", a prompt with its cursor, in lowercase letters drawn
// in the style of the "ANSI Shadow" figlet font (which has no lowercase):
// strokes two cells wide and a single-line shadow down and to the right.
var logoArt = []string{
	"                                ██┐                          ",
	" ██████┐██┐  ██┐ █████┐  █████┐██████┐ █████┐  █████┐        ",
	"██┌──██│██│  ██│ └───██┐██┌───┘└██┌──┘██┌──██┐██┌───┘        ",
	"██│  ██│██│  ██│ ██████│██│     ██│   ███████│██│            ",
	"██│  ██│██│  ██│██┌──██│██│     ██│   ██┌────┘██│            ",
	"└██████│└██████│└██████│██│     └████┐└██████┐██│    ███████┐",
	" └───██│ └─────┘ └─────┘└─┘      └───┘ └─────┘└─┘    └──────┘",
	"     ██│                                                     ",
	"     └─┘                                                     ",
}

// logoBoxWidth makes the box as wide as the table, so its right edge lines up
// with the end of the section line.
const logoBoxWidth = maxTableWidth - logoIndent

var logoBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(colorLogoBorder).
	Padding(1, 0).
	Width(logoBoxWidth).
	Align(lipgloss.Center)

// logoFits reports whether the big logo fits and still leaves the list
// minListRows rows, or every game when there are fewer than that.
func logoFits(width, height, chromeRows, games int) bool {
	free := height - logoHeight() - chromeRows - listChromeRows
	return width >= logoBoxWidth+logoIndent && free >= min(games, minListRows)
}

// logoHeight is the rows the logo block takes, spacing included.
func logoHeight() int {
	return logoPadTop + lipgloss.Height(logoBox()) + logoPadBottom
}

// logoBox is the logo in its box: the letters in the top-to-bottom gradient,
// the shadow in the dim color so it sits behind them.
func logoBox() string {
	colors := lipgloss.Blend1D(len(logoArt), colorLogoTop, colorLogoBottom)
	rows := make([]string, len(logoArt))
	for i, row := range logoArt {
		rows[i] = twoTone(row, lipgloss.NewStyle().Foreground(colors[i]))
	}
	return logoBoxStyle.Render(strings.Join(rows, "\n"))
}

// twoTone colors a logo row run by run: blocks with letter, shadow lines dim,
// spaces plain. Styling whole runs keeps one color's reset from cutting the
// next short.
func twoTone(row string, letter lipgloss.Style) string {
	var b, run strings.Builder
	style := func(r rune) *lipgloss.Style {
		switch {
		case r == ' ':
			return nil
		case r == '█':
			return &letter
		}
		return &dimStyle
	}
	var current *lipgloss.Style
	flush := func() {
		if current == nil {
			b.WriteString(run.String())
		} else {
			b.WriteString(current.Render(run.String()))
		}
		run.Reset()
	}
	for _, r := range row {
		if s := style(r); s != current {
			flush()
			current = s
		}
		run.WriteRune(r)
	}
	flush()
	return b.String()
}

// renderLogo draws the logo block: empty rows above and below, and the boxed
// logo indented to line up with the section line.
func renderLogo() string {
	indent := strings.Repeat(" ", logoIndent)
	lines := make([]string, 0, logoHeight())
	for range logoPadTop {
		lines = append(lines, "")
	}
	for _, row := range strings.Split(logoBox(), "\n") {
		lines = append(lines, indent+row)
	}
	for range logoPadBottom {
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}
