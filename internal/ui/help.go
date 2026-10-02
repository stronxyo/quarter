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
		// The widest version that fits, with the donate link when there is
		// room; for the link, the arrows' hints give way, never a key.
		widest := ""
		for _, b := range lines {
			line := h.ShortHelpView(b)
			if lipgloss.Width(line) > m.width-2 {
				continue
			}
			if widest == "" {
				widest = line
			} else if !sameKeys(b, lines[0]) {
				break // this version drops keys: the link isn't worth one
			}
			if with, ok := m.withSupport(line, h.ShortHelpView([]key.Binding{keys.Support})); ok {
				return "\n  " + with
			}
		}
		if widest != "" {
			return "\n  " + widest
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

// supportText is the donate link at the right end of the key line (Pablo,
// 2026-10-02, his pick of four: "buy me a coffee ↗"), with its key, c; the
// key or a click opens Ko-fi.
const supportText = "buy me a coffee " + markerLink

// supportGap is the least room kept between the keys and the link.
const supportGap = 6

// withSupport ends the key line (drawn after a two-space indent) with the
// donate link, its key and words drawn as the other keys' (support), flush
// with the window's right edge, if there is room for it after the keys.
func (m Model) withSupport(line, support string) (string, bool) {
	room := m.width - 2 - lipgloss.Width(line)
	if room < supportGap+lipgloss.Width(support) {
		return line, false
	}
	return line + strings.Repeat(" ", room-lipgloss.Width(support)) + support, true
}

// arrowHints are the key line's hints for keys anyone guesses; they are the
// first to give way.
var arrowHints = []key.Binding{helpMove, helpPage, helpEnds}

// sameKeys reports whether two versions of the key line hold the same keys,
// leaving the arrows' hints aside.
func sameKeys(a, b []key.Binding) bool {
	keysOf := func(bs []key.Binding) []string {
		var out []string
		for _, k := range bs {
			if !slices.ContainsFunc(arrowHints, func(h key.Binding) bool { return h.Help() == k.Help() }) {
				out = append(out, k.Help().Key)
			}
		}
		return out
	}
	return slices.Equal(keysOf(a), keysOf(b))
}
