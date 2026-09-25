package ui

import (
	"strings"

	"charm.land/lipgloss/v2"
)

// logoArt is QUARTER in the "ANSI Shadow" figlet font, with single-line shadows.
var logoArt = []string{
	" ██████┐ ██┐   ██┐ █████┐ ██████┐ ████████┐███████┐██████┐ ",
	"██┌───██┐██│   ██│██┌──██┐██┌──██┐└──██┌──┘██┌────┘██┌──██┐",
	"██│   ██│██│   ██│███████│██████┌┘   ██│   █████┐  ██████┌┘",
	"██│▄▄ ██│██│   ██│██┌──██│██┌──██┐   ██│   ██┌──┘  ██┌──██┐",
	"└██████┌┘└██████┌┘██│  ██│██│  ██│   ██│   ███████┐██│  ██│",
	" └──▀▀─┘  └─────┘ └─┘  └─┘└─┘  └─┘   └─┘   └──────┘└─┘  └─┘",
}

// logoBoxWidth makes the box as wide as the table, so its right edge lines up
// with the end of the section line.
const logoBoxWidth = maxTableWidth - logoIndent

var logoBoxStyle = lipgloss.NewStyle().
	Border(lipgloss.NormalBorder()).
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

// logoBox is the logo with its top-to-bottom gradient, framed in a rounded box.
func logoBox() string {
	colors := lipgloss.Blend1D(len(logoArt), colorLogoTop, colorLogoBottom)
	rows := make([]string, len(logoArt))
	for i, row := range logoArt {
		rows[i] = lipgloss.NewStyle().Foreground(colors[i]).Render(row)
	}
	return logoBoxStyle.Render(strings.Join(rows, "\n"))
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
