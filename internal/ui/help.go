package ui

import (
	"slices"
	"strings"

	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"
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
	h.ShortSeparator = "   " // three: the mouse's hover mark keeps a space on both sides (Pablo, 2026-10-01)
	h.FullSeparator = "   "
	if !m.fullHelp {
		// Measured untruncated: help cuts an overlong line down to "…".
		_, state := m.stateOf()
		var lines [][]key.Binding
		switch {
		case m.histView != histOff:
			lines = historyHelp()
		case m.list.SettingFilter():
			lines = [][]key.Binding{searchKeys(state)}
		case m.detailOpen:
			lines = shortHelp(append([]key.Binding{helpClose}, m.gameKeys(state)...))
		default:
			lines = shortHelp(m.gameKeys(state))
		}
		for _, b := range lines {
			if line := h.ShortHelpView(b); lipgloss.Width(line) <= m.width-2 {
				return "\n  " + line
			}
		}
		// Still too wide: drop keys from the middle until it fits, the game's
		// keys first and q quit last kept. (bubbles v2.2.1 help doesn't cut
		// reliably: an item that overshoots with no room for "…" is added
		// anyway.)
		b := slices.Clone(lines[len(lines)-1])
		line := h.ShortHelpView(b)
		for len(b) > 2 && lipgloss.Width(line) > m.width-2 {
			b = slices.Delete(b, len(b)-2, len(b)-1)
			line = h.ShortHelpView(b)
		}
		return "\n  " + ansi.Truncate(line, max(m.width-2, 0), "…")
	}
	h.SetWidth(max(m.width-2, 0))
	return "\n  " + strings.ReplaceAll(h.FullHelpView(fullHelp(m.width)), "\n", "\n  ")
}
