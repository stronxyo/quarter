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
	"github.com/charmbracelet/x/ansi"

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

// Update handles a message, then lays the screen out again: whether the logo
// fits depends on how many games are showing, which searches and filters change.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	m, cmd := m.update(msg)
	m.layout()
	return m, cmd
}

func (m Model) update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width, m.height = msg.Width, msg.Height
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

func (m Model) applyScan(msg scanDoneMsg) (Model, tea.Cmd) {
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
	return m, m.refreshList()
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

// layout gives the list every row except the header, section line, column
// header and footer.
func (m *Model) layout() {
	used := m.chromeRows()
	if m.showLogo() {
		used += logoHeight()
	}
	m.list.SetSize(m.width, max(m.height-used, 0))
	// bubbles v2.2.1 sizes the help to the full width, then pads it.
	m.list.Help.SetWidth(max(m.width-m.list.Styles.HelpStyle.GetHorizontalFrameSize(), 0))
}

// chromeRows counts the rows outside the list besides the logo: the section
// line, the column header, and the warnings footer when there is one.
func (m Model) chromeRows() int {
	rows := 2
	if m.footer() != "" {
		rows++
	}
	return rows
}

// showLogo reports whether the big logo fits alongside a usable list.
func (m Model) showLogo() bool {
	return logoFits(m.width, m.height, m.chromeRows(), len(m.list.VisibleItems()))
}

// header shows the logo above the section line, or the compact badge in
// front of it when space is short.
func (m Model) header() string {
	if m.showLogo() {
		return renderLogo() + "\n" + m.sectionLine("  ")
	}
	return m.sectionLine(" " + titleStyle.Render("QUARTER") + " ")
}

// sectionLine is the rule above the columns: filters (active one
// highlighted), the search box while searching, and the game count, with a
// dim line running out to the table's edge.
func (m Model) sectionLine(lead string) string {
	var tabs []string
	for _, f := range library.Filters() {
		if f == m.filterBy {
			tabs = append(tabs, activeTabStyle.Render("["+f.String()+"]"))
		} else {
			tabs = append(tabs, dimStyle.Render(" "+f.String()+" "))
		}
	}

	searching := m.list.SettingFilter() || m.list.IsFiltered()
	count := fmt.Sprintf("%d games", len(m.list.Items()))
	if searching {
		count = fmt.Sprintf("%d of %d games", len(m.list.VisibleItems()), len(m.list.Items()))
	}

	line := lead + dimStyle.Render("▸─ ") + strings.Join(tabs, " ") + dimStyle.Render(" ── ")
	if searching {
		// Size a copy of the input to its text so the line stays tight.
		input := m.list.FilterInput
		input.SetWidth(ansi.StringWidth(input.Value()) + 1)
		line += input.View() + dimStyle.Render(" ── ")
	}
	line += dimStyle.Render(count + " ")
	fill := min(m.width, maxTableWidth) - ansi.StringWidth(line)
	return line + dimStyle.Render(strings.Repeat("─", max(fill, 0)))
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
		content = strings.Join([]string{m.header(), m.columnHeader(), m.list.View()}, "\n")
		if f := m.footer(); f != "" {
			content += "\n" + f
		}
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
