package ui

import (
	"image/color"

	"charm.land/lipgloss/v2"
)

// Arcade palette: the logo's gold-to-orange gradient on the terminal's own
// background; everything else is accented with the gradient's middle color.
var (
	colorText    = lipgloss.Color("#F0E4D2") // ivory: softer than the terminal's white
	colorAccent  = lipgloss.Color("#FFAF63")
	colorDim     = lipgloss.Color("#B07A45") // dim amber, like a CRT's fading phosphor
	colorKeycap  = lipgloss.Color("#2E2A26") // the dark key behind each key letter
	colorError   = lipgloss.Color("#FF7B72") // light red: something is wrong
	colorRunning = lipgloss.Color("#A6E3A1")
	colorUpdate  = lipgloss.Color("#89B4FA") // blue, the running green's neighbour in the same palette

	// colorHeat shades the history calendar's days, least played first:
	// the dim amber up to the logo's gold.
	colorHeat = []color.Color{lipgloss.Color("#7A5530"), colorDim, lipgloss.Color("#E09A55"), lipgloss.Color("#FFC070"), colorLogoTop}

	// The logo gradient runs from colorLogoTop to colorLogoBottom.
	colorLogoTop    = lipgloss.Color("#FFE08A")
	colorLogoBottom = lipgloss.Color("#FF7A3D")
	colorLogoBorder = colorAccent

	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#1E1E2E")).Background(colorAccent).Padding(0, 1)
	rowStyle      = lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle = lipgloss.NewStyle().PaddingLeft(1).Bold(true).Foreground(colorAccent).
			Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(colorAccent)
	textStyle      = lipgloss.NewStyle().Foreground(colorText)
	dimStyle       = lipgloss.NewStyle().Foreground(colorDim)
	activeTabStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	starStyle      = lipgloss.NewStyle().Foreground(colorAccent)

	runningStyle         = rowStyle.Foreground(colorRunning)
	selectedRunningStyle = selectedStyle.Foreground(colorRunning).BorderForeground(colorRunning)
	runningLabelStyle    = lipgloss.NewStyle().Bold(true).Foreground(colorRunning)
	updatingStyle        = rowStyle.Foreground(colorUpdate)
	selectedUpdateStyle  = selectedStyle.Foreground(colorUpdate).BorderForeground(colorUpdate)
	updatingLabelStyle   = lipgloss.NewStyle().Bold(true).Foreground(colorUpdate)
	updatingFgStyle      = lipgloss.NewStyle().Foreground(colorUpdate)             // a download's progress bar
	hoverStyle           = lipgloss.NewStyle().Foreground(colorAccent)             // under the mouse pointer (mouse.go)
	hoverDayStyle        = lipgloss.NewStyle().Bold(true).Foreground(colorLogoTop) // a calendar day under the pointer
	errorStyle           = lipgloss.NewStyle().Bold(true).Foreground(colorError)
	alertStyle           = lipgloss.NewStyle().Foreground(colorError)
	messageStyle         = lipgloss.NewStyle().Padding(1, 2)
)

// markerUpdating marks a game Steam is updating: U+2193, which Cascadia Mono
// and JetBrains Mono both have (U+2B07, ⬇, is in neither).
const markerUpdating = "↓"

// markerHover marks what the mouse pointer is over: U+25AA, in both fonts
// (Pablo's pick in the Mouse lab, 2026-10-01).
const markerHover = "▪"

// markerLink follows the website's name in the details pane: it opens
// (Pablo, 2026-10-02). U+2197 is in JetBrains Mono; on Cascadia and Consolas
// Windows draws it from Segoe UI Symbol, as it does the favourite star.
const markerLink = "↗"

const (
	markerInstalled            = "▸"
	markerUninstalled          = "▹"
	markerFavourite            = "✦"
	markerFavouriteUninstalled = "✧"
	markerRunning              = "▶"
	markerWidth                = 1  // one marker: the favourite star, else the installed dot
	hoursWidth                 = 9  // fits "99999.9h"
	agoWidth                   = 13 // fits "11 months ago"
	maxTableWidth              = 80 // keeps the numbers near the names on wide terminals
	minListRows                = 8  // the big logo gives way before the list gets shorter than this
	listChromeRows             = 1  // the row the list keeps for its page counter
	logoIndent                 = 2  // lines the box up with the section line
	paneMinWidth               = 40 // the detail pane beside the list, at its narrowest
	paneMaxWidth               = 80 // the pop-up pane on a narrow window
	paneTextWidth              = 76 // the pane's contents, a centred column: readable lines stop at about 75 (Batch 8, A2)
	paneGap                    = 2
	minTableWidth              = 56 // narrower than this, the pane becomes a pop-up
	logoPadTop                 = 1
	logoPadBottom              = 1
)
