package ui

import "charm.land/lipgloss/v2"

// Arcade palette: the logo's gold-to-orange gradient on the terminal's own
// background; everything else is accented with the gradient's middle color.
var (
	colorAccent = lipgloss.Color("#FFAF63")
	colorDim    = lipgloss.Color("#7F849C")
	colorError  = lipgloss.Color("#F38BA8")

	// The logo gradient runs from colorLogoTop to colorLogoBottom.
	colorLogoTop    = lipgloss.Color("#FFE08A")
	colorLogoBottom = lipgloss.Color("#FF7A3D")
	colorLogoBorder = colorAccent

	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#1E1E2E")).Background(colorAccent).Padding(0, 1)
	rowStyle      = lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle = lipgloss.NewStyle().PaddingLeft(1).Bold(true).Foreground(colorAccent).
			Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(colorAccent)
	dimStyle       = lipgloss.NewStyle().Foreground(colorDim)
	activeTabStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	starStyle      = lipgloss.NewStyle().Foreground(colorAccent)
	errorStyle     = lipgloss.NewStyle().Bold(true).Foreground(colorError)
	messageStyle   = lipgloss.NewStyle().Padding(1, 2)
)

const (
	markerInstalled = "•"
	markerFavourite = "✦"
	markerWidth     = 1  // one marker: the favourite star, else the installed dot
	hoursWidth      = 9  // fits "99999.9h"
	agoWidth        = 13 // fits "11 months ago"
	maxTableWidth   = 80 // keeps the numbers near the names on wide terminals
	minListRows     = 8  // the big logo gives way before the list gets shorter than this
	listChromeRows  = 1  // the row the list keeps for its page counter
	logoIndent      = 2  // lines the box up with the section line
	paneMinWidth    = 40 // the detail pane beside the list, at its narrowest
	paneMaxWidth    = 68 // wide enough for a 64-column game logo
	paneGap         = 2
	minTableWidth   = 56 // narrower than this, the pane becomes a pop-up
	logoPadTop      = 1
	logoPadBottom   = 1
)
