package ui

import (
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/lipgloss/v2"
)

// Keycaps: each key sits on a dark key with its letter in the accent color.
var (
	keycapStyle = lipgloss.NewStyle().Bold(true).Foreground(colorAccent).Background(colorKeycap).Padding(0, 1)
	helpStyles  = help.Styles{
		ShortKey:       keycapStyle,
		ShortDesc:      dimStyle,
		ShortSeparator: dimStyle,
		FullKey:        keycapStyle,
		FullDesc:       dimStyle,
		FullSeparator:  dimStyle,
		Ellipsis:       dimStyle,
	}
)

// helpView is the key help under the list: one line, or every key after ?.
func (m Model) helpView() string {
	h := help.New()
	h.Styles = helpStyles
	h.ShortSeparator = "  " // keycaps already set keys apart
	h.FullSeparator = "   "
	h.SetWidth(max(m.width-2, 0))
	if m.fullHelp {
		return "\n  " + strings.ReplaceAll(h.FullHelpView(fullHelp()), "\n", "\n  ")
	}
	return "\n  " + h.ShortHelpView(shortHelp())
}
