package ui

import (
	"errors"
	"strings"

	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/steam"
)

// failedView is the screen when the scan couldn't start: Steam wasn't found.
// It says where Quarter looked and what to do, and r looks again (Steam may
// have been installed since).
func (m Model) failedView() string {
	var b strings.Builder
	var nf *steam.NotFoundError
	switch {
	case errors.As(m.err, &nf) && nf.Given != "":
		b.WriteString(errorStyle.Render("Steam not found.") + "\n\n")
		b.WriteString(alertStyle.Render(nf.Given+" is not a Steam folder: it has no steamapps folder.") + "\n\n")
		b.WriteString("Check the folder given with --steam-path or " + config.SteamPathEnv + ".\n")
	case errors.As(m.err, &nf):
		b.WriteString(errorStyle.Render("Steam not found.") + "\n\n")
		b.WriteString("Quarter looked in:\n")
		for _, dir := range nf.Looked {
			b.WriteString(dimStyle.Render("  "+dir) + "\n")
		}
		b.WriteString("\nInstall Steam, or tell Quarter where it is:\n")
		b.WriteString("  quarter --steam-path <steam folder>\n")
		b.WriteString("  or set " + config.SteamPathEnv + "\n")
	default:
		b.WriteString(errorStyle.Render("Could not read Steam.") + "\n\n")
		b.WriteString(alertStyle.Render(m.err.Error()) + "\n")
	}
	b.WriteString("\n")
	if m.scanning {
		b.WriteString(dimStyle.Render("looking again…"))
	} else {
		b.WriteString(dimStyle.Render("r look again · q quit"))
	}
	return messageStyle.Render(b.String())
}
