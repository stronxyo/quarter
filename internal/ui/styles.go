package ui

import "charm.land/lipgloss/v2"

// Arcade palette: coin gold on the terminal's own background.
var (
	colorAccent = lipgloss.Color("#FFC857")
	colorDim    = lipgloss.Color("#7F849C")
	colorError  = lipgloss.Color("#F38BA8")

	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#1E1E2E")).Background(colorAccent).Padding(0, 1)
	rowStyle      = lipgloss.NewStyle().PaddingLeft(2)
	selectedStyle = lipgloss.NewStyle().PaddingLeft(1).Bold(true).Foreground(colorAccent).
			Border(lipgloss.NormalBorder(), false, false, false, true).BorderForeground(colorAccent)
	dimStyle       = lipgloss.NewStyle().Foreground(colorDim)
	activeTabStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent)
	errorStyle     = lipgloss.NewStyle().Bold(true).Foreground(colorError)
	messageStyle   = lipgloss.NewStyle().Padding(1, 2)
)

const (
	markerInstalled = "●"
	hoursWidth      = 9  // fits "99999.9h"
	agoWidth        = 13 // fits "11 months ago"
	maxTableWidth   = 80 // keeps the numbers near the names on wide terminals
)
