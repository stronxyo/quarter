// Package ui is Quarter's terminal interface, built on Bubbletea.
package ui

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
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

	games    []library.Game // everything scanned; the list shows a filtered, sorted view
	filterBy library.Filter
	sortBy   library.SortOrder
	sortDesc bool
	now      func() time.Time

	list   list.Model
	width  int
	height int
}

// New returns the UI model. steamPath may be empty to detect Steam.
func New(steamPath string) Model {
	return Model{steamPath: steamPath, list: newList(), now: time.Now}
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
		// While typing a search, every key belongs to the search box.
		if !m.list.SettingFilter() {
			for _, sk := range sortKeys {
				if key.Matches(msg, sk.binding) {
					m.sortBy, m.sortDesc = nextSort(m.sortBy, m.sortDesc, sk.order)
					return m, m.refreshList()
				}
			}
			if key.Matches(msg, keys.Filter) {
				m.filterBy = m.filterBy.Next()
				return m, m.refreshList()
			}
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
	m.games = msg.res.Games
	m.list.SetDelegate(rowDelegate{playtimeKnown: msg.res.PlaytimeKnown, now: m.now})
	cmd := m.refreshList()
	m.layout()
	return m, cmd
}

// refreshList rebuilds the list from the current filter and sort, keeping the
// selected game selected when it is still shown. During a search the list
// ranks matches itself, so the selection starts at the best match.
func (m *Model) refreshList() tea.Cmd {
	var selected uint32
	if it, ok := m.list.SelectedItem().(gameItem); ok {
		selected = it.AppID
	}

	view := library.View(m.games, m.filterBy, m.sortBy, m.sortDesc, m.now())
	cmd := m.list.SetItems(toItems(view))

	m.list.ResetSelected()
	if !m.list.IsFiltered() {
		for i, g := range view {
			if g.AppID == selected {
				m.list.Select(i)
				break
			}
		}
	}
	return cmd
}

// nextSort picks a column: the same column again reverses it, a new column
// starts in its natural direction.
func nextSort(cur library.SortOrder, desc bool, picked library.SortOrder) (library.SortOrder, bool) {
	if picked == cur {
		return cur, !desc
	}
	return picked, picked.DefaultDesc()
}

// layout gives the list every row except the header, search line, column
// header and footer.
func (m *Model) layout() {
	used := 3
	if m.footer() != "" {
		used++
	}
	m.list.SetSize(m.width, max(m.height-used, 0))
	// bubbles v2.2.1 sizes the help to the full width, then pads it.
	m.list.Help.SetWidth(max(m.width-m.list.Styles.HelpStyle.GetHorizontalFrameSize(), 0))
}

// header shows the title, the filters (active one highlighted) and the count.
func (m Model) header() string {
	var parts []string
	for _, f := range library.Filters() {
		if f == m.filterBy {
			parts = append(parts, activeTabStyle.Render("["+f.String()+"]"))
		} else {
			parts = append(parts, dimStyle.Render(" "+f.String()+" "))
		}
	}
	count := fmt.Sprintf("%d games", len(m.list.Items()))
	if m.list.IsFiltered() || m.list.SettingFilter() {
		count = fmt.Sprintf("%d of %d games", len(m.list.VisibleItems()), len(m.list.Items()))
	}
	return " " + titleStyle.Render("QUARTER") + "  " + strings.Join(parts, " ") + dimStyle.Render("   "+count)
}

// searchLine shows the search box while searching, and is blank otherwise.
func (m Model) searchLine() string {
	if m.list.SettingFilter() || m.list.IsFiltered() {
		return "  " + m.list.FilterInput.View()
	}
	return ""
}

// columnHeader labels the columns and marks the sorted one with an arrow.
func (m Model) columnHeader() string {
	label := func(name string, order library.SortOrder) string {
		if order != m.sortBy {
			return name
		}
		if m.sortDesc {
			return name + " ▼"
		}
		return name + " ▲"
	}
	line := columns(m.width-rowStyle.GetHorizontalFrameSize(), " ",
		label("Name", library.SortName), label("Played", library.SortPlaytime), label("Last played", library.SortLastPlayed))
	return rowStyle.Render(dimStyle.Render(line))
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
		content = strings.Join([]string{m.header(), m.searchLine(), m.columnHeader(), m.list.View()}, "\n")
		if f := m.footer(); f != "" {
			content += "\n" + f
		}
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
