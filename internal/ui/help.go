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
		// The donate link always shows, at the right edge (Pablo, 2026-10-02:
		// "must always be visible no matter the wide"); the keys get the
		// rest: the widest version that fits.
		support := h.ShortHelpView([]key.Binding{keys.Support})
		room := m.width - 2 - lipgloss.Width(support) - supportGap
		for _, b := range lines {
			if line := h.ShortHelpView(b); lipgloss.Width(line) <= room {
				return "\n  " + m.withSupport(line, support)
			}
		}
		// Still too wide: drop keys from the middle until it fits, the game's
		// keys first and q quit last kept. (bubbles v2.2.1 help doesn't cut
		// reliably: an item that overshoots with no room for "…" is added
		// anyway.)
		b := slices.Clone(lines[len(lines)-1])
		line := h.ShortHelpView(b)
		for len(b) > 2 && lipgloss.Width(line) > room {
			b = slices.Delete(b, len(b)-2, len(b)-1)
			line = h.ShortHelpView(b)
		}
		return "\n  " + m.withSupport(ansi.Truncate(line, max(room, 0), "…"), support)
	}
	h.SetWidth(max(m.width-2, 0))
	full := "\n  " + strings.ReplaceAll(h.FullHelpView(fullHelp(m.width)), "\n", "\n  ")
	return full + "\n  " + m.withSupport("", h.ShortHelpView([]key.Binding{keys.Support})) // in ? too, on a line of its own
}

// supportText is the donate link at the right end of the key line (Pablo,
// 2026-10-02, his pick of four: "buy me a coffee ↗"), with its key, c; the
// key or a click opens Ko-fi.
var supportText = "buy me a coffee " + markerLink // set again by applyGlyphs

// supportGap is the least room kept between the keys and the link: the
// key line's own gap between keys.
const supportGap = 3

// withSupport ends a line (drawn after a two-space indent) with the donate
// link, its key and words drawn as the other keys' (support), flush with the
// window's right edge. On a window too narrow for anything else, the link
// alone, cut to fit.
func (m Model) withSupport(line, support string) string {
	room := m.width - 2 - lipgloss.Width(line)
	if room < lipgloss.Width(support) {
		return ansi.Truncate(support, max(m.width-2, 0), "…")
	}
	return line + strings.Repeat(" ", room-lipgloss.Width(support)) + support
}
