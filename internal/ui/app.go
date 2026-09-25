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
	"charm.land/lipgloss/v2"
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
	scanning  bool                       // a background scan is running
	scanErr   error                      // a rescan failed while a cached library is shown
	save      func(library.Result) error // stores a fresh scan for the next launch; may be nil
	open      func(uri string) error     // hands a steam:// link to the OS
	status    string                     // short-lived footer message after an action
	statusID  int                        // tells a stale clear timer from the current message
	fullHelp  bool                       // ? pressed: every key is listed

	favs         map[uint32]bool             // starred games, by appid
	saveSettings func(config.Settings) error // remembers favourites, sort and filter; may be nil

	games    []library.Game // everything scanned; the list shows a filtered, sorted view
	filterBy library.Filter
	sortBy   library.SortOrder
	sortDesc bool
	now      func() time.Time

	list   list.Model
	width  int
	height int
}

// Options wires the UI to the rest of Quarter. Every field may be left empty.
type Options struct {
	SteamPath string                     // --steam-path override; empty means detect
	Cached    *library.Result            // shown immediately while a fresh scan runs
	Save      func(library.Result) error // stores each fresh scan for the next launch
	Open      func(uri string) error     // hands steam:// links to the OS

	Settings     config.Settings             // favourites, sort and filter from the last run
	SaveSettings func(config.Settings) error // remembers them for the next run
}

// New returns the UI model.
func New(opts Options) Model {
	m := Model{
		steamPath: opts.SteamPath,
		save:      opts.Save,
		open:      opts.Open,
		list:      newList(),
		now:       time.Now,
		scanning:  true,

		favs:         make(map[uint32]bool),
		saveSettings: opts.SaveSettings,
	}
	m.applySettings(opts.Settings)
	if opts.Cached != nil {
		m.state = stateReady
		m.games = opts.Cached.Games
		m.markFavourites()
		m.list.SetDelegate(rowDelegate{playtimeKnown: opts.Cached.PlaytimeKnown, now: m.now})
		m.refreshList()
	}
	return m
}

func (m Model) Init() tea.Cmd {
	return scan(m.steamPath)
}

// saveScan stores a scan off the UI goroutine; a failure only costs speed on
// the next launch, so it is logged, not shown.
func saveScan(save func(library.Result) error, res library.Result) tea.Cmd {
	return func() tea.Msg {
		if err := save(res); err != nil {
			slog.Warn("cache", "err", err)
		}
		return nil
	}
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

	case openDoneMsg:
		if msg.id != m.statusID {
			return m, nil // a newer action replaced this one
		}
		if msg.err != nil {
			slog.Warn("open", "err", msg.err)
			m.status = "Could not open Steam: " + msg.err.Error()
		}
		return m, clearStatusAfter(msg.id)

	case clearStatusMsg:
		if msg.id == m.statusID {
			m.status = ""
		}
		return m, nil

	case tea.KeyPressMsg:
		if key.Matches(msg, keys.Refresh) && !m.list.SettingFilter() && m.state != stateScanning {
			if m.scanning {
				return m, nil
			}
			m.scanning = true
			return m, scan(m.steamPath)
		}
		if m.state != stateReady {
			if s := msg.String(); s == "q" || s == "ctrl+c" {
				return m, tea.Quit
			}
			return m, nil
		}
		// While typing a search, every key belongs to the search box.
		if !m.list.SettingFilter() {
			if key.Matches(msg, keys.Help) {
				m.fullHelp = !m.fullHelp
				return m, nil
			}
			if m, cmd, ok := m.action(msg); ok {
				return m, cmd
			}
			for _, sk := range sortKeys {
				if key.Matches(msg, sk.binding) {
					m.sortBy, m.sortDesc = nextSort(m.sortBy, m.sortDesc, sk.order)
					return m, tea.Batch(m.refreshList(), m.persist())
				}
			}
			if key.Matches(msg, keys.Filter) {
				m.filterBy = m.filterBy.Next()
				return m, tea.Batch(m.refreshList(), m.persist())
			}
			if key.Matches(msg, keys.Favourite) {
				return m.toggleFavourite()
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) applyScan(msg scanDoneMsg) (Model, tea.Cmd) {
	m.scanning = false
	if msg.err != nil {
		slog.Warn("scan", "err", msg.err)
		if m.state == stateReady {
			// Keep showing the cached library rather than an error screen.
			m.scanErr = msg.err
			return m, nil
		}
		m.state, m.err = stateFailed, msg.err
		return m, nil
	}
	m.scanErr = nil
	for _, w := range msg.res.Warnings {
		slog.Warn("scan", "err", w)
	}
	m.state = stateReady
	m.warnings = len(msg.res.Warnings)
	m.games = msg.res.Games
	m.markFavourites()
	m.list.SetDelegate(rowDelegate{playtimeKnown: msg.res.PlaytimeKnown, now: m.now})
	cmd := m.refreshList()
	if m.save != nil {
		cmd = tea.Batch(cmd, saveScan(m.save, msg.res))
	}
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

// layout gives the list every row except the header, section line, column
// header and footer.
func (m *Model) layout() {
	used := m.chromeRows()
	if m.showLogo() {
		used += logoHeight()
	}
	m.list.SetSize(m.width, max(m.height-used, 0))
}

// chromeRows counts the rows outside the list besides the logo: the section
// line, the column header, the key help, and the footer when there is one.
func (m Model) chromeRows() int {
	rows := 2 + lipgloss.Height(m.helpView())
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
	if m.scanning {
		count += " · updating…"
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
	line := columns(m.width-rowStyle.GetHorizontalFrameSize(), strings.Repeat(" ", markerWidth),
		label("Name", library.SortName), label("Played", library.SortPlaytime), label("Last played", library.SortLastPlayed))
	return rowStyle.Render(dimStyle.Render(line))
}

// footer shows, most important first: an action's message, a failed rescan,
// or files that could not be read.
func (m Model) footer() string {
	if m.status != "" {
		return "  " + activeTabStyle.Render(m.status)
	}
	if m.scanErr != nil {
		return dimStyle.Render("  Could not rescan Steam, showing the last saved library. Press r to retry.")
	}
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
		content = strings.Join([]string{m.header(), m.columnHeader(), m.list.View(), m.helpView()}, "\n")
		if f := m.footer(); f != "" {
			content += "\n" + f
		}
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
