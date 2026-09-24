// Package ui is Quarter's terminal interface, built on Bubbletea.
package ui

import (
	"fmt"
	"log/slog"

	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"

	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/library"
)

type state int

const (
	stateScanning state = iota
	stateReady
	stateFailed
)

// scanDoneMsg carries a finished background scan.
type scanDoneMsg struct {
	res library.Result
	err error
}

// Model is the root of the UI.
type Model struct {
	steamPath string // --steam-path override; empty means detect
	state     state
	err       error
	warnings  int
	list      list.Model
	width     int
	height    int
}

// New returns the UI model. steamPath may be empty to detect Steam.
func New(steamPath string) Model {
	return Model{steamPath: steamPath, list: newList()}
}

func (m Model) Init() tea.Cmd {
	return scan(m.steamPath)
}

// scan runs off the UI goroutine so the screen never freezes on disk reads.
func scan(steamPath string) tea.Cmd {
	return func() tea.Msg {
		res, err := library.Scan(steamPath)
		return scanDoneMsg{res: res, err: err}
	}
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
		m.layout()
		return m, nil

	case scanDoneMsg:
		return m.applyScan(msg)

	case tea.KeyPressMsg:
		if m.state != stateReady {
			if s := msg.String(); s == "q" || s == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) applyScan(msg scanDoneMsg) (tea.Model, tea.Cmd) {
	if msg.err != nil {
		m.state, m.err = stateFailed, msg.err
		return m, nil
	}
	for _, w := range msg.res.Warnings {
		slog.Warn("scan", "err", w)
	}
	m.state = stateReady
	m.warnings = len(msg.res.Warnings)
	m.list.SetDelegate(rowDelegate{playtimeKnown: msg.res.PlaytimeKnown})
	cmd := m.list.SetItems(toItems(msg.res.Games))
	m.layout()
	return m, cmd
}

// layout gives the list all rows except the footer.
func (m *Model) layout() {
	footerHeight := 0
	if m.footer() != "" {
		footerHeight = 1
	}
	m.list.SetSize(m.width, max(m.height-footerHeight, 0))
}

func (m Model) footer() string {
	if m.warnings == 0 {
		return ""
	}
	return dimStyle.Render(fmt.Sprintf("  %d file(s) could not be read, see %s", m.warnings, config.LogFile))
}

func (m Model) View() tea.View {
	var content string
	switch m.state {
	case stateScanning:
		content = messageStyle.Render("Scanning Steam libraries...")
	case stateFailed:
		content = messageStyle.Render(errorStyle.Render("Steam not found.") +
			"\n\nIf Steam is installed somewhere unusual, start Quarter with:\n  quarter --steam-path <steam folder>\n\n" +
			dimStyle.Render("q quit"))
	default:
		content = m.list.View()
		if f := m.footer(); f != "" {
			content += "\n" + f
		}
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
