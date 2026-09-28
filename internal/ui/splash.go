package ui

import (
	"fmt"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

// The first-run screen (Batch 8, A3): an old PC checking itself as it boots,
// then the quarter_ logo and INSERT QUARTER blinking, like an arcade cabinet
// waiting for a coin. It shows the real scan as it happens, so it never adds
// waiting of its own, and any key starts. It is shown once; the setting
// splash_seen remembers that.

// blinkEvery is how long INSERT QUARTER stays lit, and stays dark.
const blinkEvery = 530 * time.Millisecond

// splashLeader is the width of a boot line, dots included, before its result.
const splashLeader = 30

// blinkMsg turns INSERT QUARTER on or off.
type blinkMsg struct{}

// blink schedules the next blink while the splash is up.
func (m Model) blink() tea.Cmd {
	if !m.splash {
		return nil
	}
	return tea.Tick(blinkEvery, func(time.Time) tea.Msg { return blinkMsg{} })
}

// leaveSplash handles a key on the splash: q and ctrl+c quit, any other key
// starts, and the splash is remembered as seen.
func (m Model) leaveSplash(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	if s := msg.String(); s == "q" || s == "ctrl+c" {
		return m, tea.Quit
	}
	m.splash, m.splashSeen = false, true
	return m, m.persist()
}

// splashView is the whole screen while the splash is up.
func (m Model) splashView() string {
	steam, games := "...", "..."
	switch m.state {
	case stateReady:
		steam, games = "OK", fmt.Sprintf("%d games", len(m.games))
	case stateFailed:
		steam, games = "NOT FOUND", "-"
	}
	lines := []string{
		dimStyle.Render("QUARTER SYSTEM CHECK"),
		"",
		bootLine("Detecting Steam", steam),
		bootLine("Reading libraries", games),
	}
	// One left edge for the boot lines, the block centred as a whole.
	boot := lipgloss.NewStyle().Width(lipgloss.Width(strings.Join(lines, "\n"))).Render(strings.Join(lines, "\n"))

	// The big logo when it fits, else the badge.
	logo := titleStyle.Render("quarter_")
	if m.width >= logoBoxWidth && m.height >= lipgloss.Height(logoBox())+8 {
		logo = logoBox()
	}

	coin := strings.Repeat(" ", len("INSERT QUARTER"))
	if m.blinkOn {
		coin = activeTabStyle.Render("INSERT QUARTER")
	}
	prompt := lipgloss.JoinVertical(lipgloss.Center, coin, dimStyle.Render("press any key"))

	screen := lipgloss.JoinVertical(lipgloss.Center, boot, "", logo, "", prompt)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, screen)
}

// bootLine is "Detecting Steam ........... OK": the check, dots out to
// splashLeader, and its result.
func bootLine(check, result string) string {
	dots := strings.Repeat(".", max(splashLeader-len(check)-2, 3))
	return textStyle.Render(check) + " " + dimStyle.Render(dots) + " " + activeTabStyle.Render(result)
}
