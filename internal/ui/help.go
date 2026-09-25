package ui

import (
	"strings"

	"charm.land/bubbles/v2/help"
)

// helpView is the key help under the list: one line, or every key after ?.
func (m Model) helpView() string {
	h := help.New()
	h.SetWidth(max(m.width-2, 0))
	if m.fullHelp {
		return "\n  " + strings.ReplaceAll(h.FullHelpView(fullHelp()), "\n", "\n  ")
	}
	return "\n  " + h.ShortHelpView(shortHelp())
}
