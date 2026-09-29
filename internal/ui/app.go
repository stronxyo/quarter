// Package ui is Quarter's terminal interface, built on Bubbletea.
package ui

import (
	"context"
	"fmt"
	"log/slog"
	"maps"
	"math/rand/v2"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	uv "github.com/charmbracelet/ultraviolet"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/art"
	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/intro"
	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/webapi"
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

	detailOpen    bool // d pressed on a narrow window: the pane covers the list
	paneHidden    bool // d pressed on a wide window: the list takes the whole width
	playtimeKnown bool // false when Steam's playtime file couldn't be read

	fetchStore func(context.Context, uint32) (webapi.StoreInfo, error) // may be nil: no store details
	saveStore  func(map[uint32]webapi.StoreEntry) error
	storeInfo  map[uint32]webapi.StoreEntry
	offline    map[uint32]bool // store requests that failed this run
	fetching   uint32          // the game whose store request is out, 0 for none
	wanted     uint32          // the selected game, waiting for its turn
	storeDown  bool            // the last request failed on the network: stop the background fill
	debounce   time.Duration   // how long the selection must settle before fetching

	runningApp func() (uint32, bool) // asks Steam which game is running; may be nil
	running    uint32                // the game Steam is running, 0 for none
	afterPlay  afterPlay             // a game that stopped, waiting for Steam to save its playtime

	newLogos     func(steamPath string) LogoSource // may be nil: no game logos
	logos        LogoSource                        // set once the Steam folder is known
	logosHidden  bool                              // o pressed: the pane shows names only
	logoBlocks   art.Blocks
	logoCache    map[logoKey]drawnLogo // drawn logos, by game and size
	logoPending  map[logoKey]bool      // being drawn, or waiting for a download
	logoNone     map[uint32]bool       // no logo anywhere, this run
	logoFetching uint32                // the game whose logo is downloading, 0 for none

	sixel        bool      // the terminal draws Sixel: logos are pictures
	cellW, cellH int       // a cell in pixels, as the terminal reports it
	pictureWant  sixelSig  // the picture the band should show
	pictureShown sixelSig  // the picture drawn last, erased before the next
	resizedAt    time.Time // the window's last size change: pictures wait for it to settle

	splash     bool // the start screen is up
	splashSeen bool // it has been shown once (saved)
	blinkOn    bool // insert quarter is lit, in its blink

	introSetting string     // config.json's "intro": "" random, "off", or an intro's name
	lastIntro    string     // the intro played last, never played twice in a row (saved)
	introIndex   int        // this start's intro, -1 for none
	introRun     *intro.Run // it playing, once the window's size is known
	introStart   time.Time  // when it started
	introDone    bool       // it has ended, or a key skipped it

	goodbyeSetting string        // config.json's "goodbye": "" random, "off", or a goodbye's name
	lastGoodbye    string        // the goodbye shown last, never shown twice in a row (saved)
	played         time.Duration // time in games this session, finished runs
	playingSince   time.Time     // when the game running now started, zero for none

	favs         map[uint32]bool             // starred games, by appid
	saveSettings func(config.Settings) error // remembers favourites, sort and filter; may be nil

	games       []library.Game // what the list can show: installed games plus resolved candidates
	installed   []library.Game // from the scan
	uninstalled []library.Game // from the scan: other apps Steam knows, unnamed until the store says what they are
	filterBy    library.Filter
	sortBy      library.SortOrder
	sortDesc    bool
	now         func() time.Time

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

	FetchStore func(context.Context, uint32) (webapi.StoreInfo, error) // store details for one game
	StoreCache map[uint32]webapi.StoreEntry                            // store details from earlier runs
	SaveStore  func(map[uint32]webapi.StoreEntry) error                // remembers them

	RunningApp func() (uint32, bool) // which game Steam is running, if it says

	Logos func(steamPath string) LogoSource // game logos, once the Steam folder is known

	Splash bool // open on the INSERT QUARTER screen (the first run)
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

		fetchStore: opts.FetchStore,
		saveStore:  opts.SaveStore,
		storeInfo:  make(map[uint32]webapi.StoreEntry),
		offline:    make(map[uint32]bool),
		debounce:   storeDebounce,
		runningApp: opts.RunningApp,

		newLogos:    opts.Logos,
		logoCache:   make(map[logoKey]drawnLogo),
		logoPending: make(map[logoKey]bool),
		logoNone:    make(map[uint32]bool),

		splash: opts.Splash,
	}
	maps.Copy(m.storeInfo, opts.StoreCache)
	m.applySettings(opts.Settings)
	m.pickIntro(rand.New(rand.NewPCG(uint64(time.Now().UnixNano()), 1)))
	if opts.Cached != nil {
		m.state = stateReady
		m.installed, m.uninstalled = opts.Cached.Games, opts.Cached.Uninstalled
		m.setSteamPath(opts.Cached.SteamPath)
		m.rebuild()
		m.playtimeKnown = opts.Cached.PlaytimeKnown
		m.list.SetDelegate(m.delegate())
		m.refreshList()
	}
	return m
}

func (m Model) Init() tea.Cmd {
	cmd := tea.Batch(scan(m.steamPath), checkRunning(m.runningApp, 0), askPictures(), m.blink(), m.introTick())
	if m.introIndex >= 0 {
		cmd = tea.Batch(cmd, m.persist()) // remember it, so the next start picks another
	}
	return cmd
}

// delegate is the row renderer for the current state.
func (m Model) delegate() rowDelegate {
	return rowDelegate{playtimeKnown: m.playtimeKnown, now: m.now, running: m.running}
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
	before, _ := m.selected()
	m, cmd := m.update(msg)
	m.layout()
	if after, ok := m.selected(); ok && after.AppID != before.AppID && m.needsStore(after.AppID) {
		cmd = tea.Batch(cmd, wantStore(after.AppID, m.debounce))
	}
	if logo := m.ensureLogo(); logo != nil {
		cmd = tea.Batch(cmd, logo)
	}
	if pic := m.wantPicture(); pic != nil {
		cmd = tea.Batch(cmd, pic)
	}
	return m, cmd
}

func (m Model) update(msg tea.Msg) (Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Bubble Tea sends the size twice at startup: only a real change of
		// size ends an intro laid out for the old one.
		if m.introRun != nil && (msg.Width != m.width || msg.Height != m.height) {
			m.introDone = true
		}
		m.width, m.height = msg.Width, msg.Height
		m.startIntro()
		return m, m.resized()

	case uv.PrimaryDeviceAttributesEvent, uv.CellSizeEvent, pictureMsg, sizeSettledMsg:
		return m.picturesMsg(msg)

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

	case storeWantMsg:
		if g, ok := m.selected(); ok && g.AppID == msg.appID {
			m.wanted = msg.appID
			return m, m.nextStoreFetch()
		}
		return m, nil

	case storeDoneMsg:
		return m.applyStore(msg)

	case logoDoneMsg:
		return m.applyLogo(msg)

	case logoWantMsg:
		return m.wantLogo(msg)

	case runningMsg:
		var reread tea.Cmd
		if msg.appID != m.running {
			m.countPlayed(msg.appID)
			if m.running != 0 {
				reread = m.gameStopped(m.running)
			}
			m.running = msg.appID
			m.list.SetDelegate(m.delegate())
		}
		return m, tea.Batch(checkRunning(m.runningApp, runningPoll), reread)

	case afterPlayMsg:
		return m.afterPlayTick(msg)

	case introTickMsg:
		return m.stepIntro()

	case blinkMsg:
		m.blinkOn = !m.blinkOn
		return m, m.blink()

	case clearStatusMsg:
		if msg.id == m.statusID {
			m.status = ""
		}
		return m, nil

	case tea.KeyPressMsg:
		if m.splash {
			return m.splashKey(msg)
		}
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
			if m.detailOpen && (key.Matches(msg, keys.Details) || msg.String() == "esc") {
				m.detailOpen = false
				return m, nil
			}
			if key.Matches(msg, keys.Details) {
				if m.roomForPane() {
					m.paneHidden = !m.paneHidden
					return m, m.persist()
				}
				m.detailOpen = true
				return m, nil
			}
			if key.Matches(msg, keys.Logos) {
				m.logosHidden = !m.logosHidden
				return m, m.persist()
			}
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
			if m.wrapMove(msg) {
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	m.list, cmd = m.list.Update(msg)
	return m, cmd
}

func (m Model) applyScan(msg scanDoneMsg) (Model, tea.Cmd) {
	m, cmd := m.applyScanResult(msg)
	next := m.afterPlayScanned()
	return m, tea.Batch(cmd, next)
}

func (m Model) applyScanResult(msg scanDoneMsg) (Model, tea.Cmd) {
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
	m.installed, m.uninstalled = msg.res.Games, msg.res.Uninstalled
	m.setSteamPath(msg.res.SteamPath)
	m.rebuild()
	m.playtimeKnown = msg.res.PlaytimeKnown
	m.list.SetDelegate(m.delegate())
	cmd := tea.Batch(m.refreshList(), m.nextStoreFetch())
	if m.save != nil {
		cmd = tea.Batch(cmd, saveScan(m.save, msg.res))
	}
	return m, cmd
}

// setSteamPath points the logo sources at the Steam folder the scan found.
func (m *Model) setSteamPath(path string) {
	if m.newLogos != nil && m.logos == nil {
		m.logos = m.newLogos(path)
	}
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
	// Twice: bubbles v2.2.1 sizes a page by whether the page counter showed
	// before, so the resize that brings the counter in draws one row too
	// many. The second call sees the counter and gets it right.
	m.list.SetSize(m.tableWidth(), max(m.height-used, 0))
	m.list.SetSize(m.tableWidth(), max(m.height-used, 0))
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

// header is the section line, led by the compact badge when the big logo
// doesn't fit above it.
func (m Model) header() string {
	if m.showLogo() {
		return m.sectionLine("  ")
	}
	return m.sectionLine(" " + titleStyle.Render("quarter_") + " ")
}

// sectionLine is the rule above the columns: filters (active one
// highlighted), the search box while searching, and the game count, with a
// dim line running out to the table's edge, or on to the pane beside it.
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
	// Beside the pane, the line runs on to it across any gap.
	end := m.tableWidth()
	if m.sideBySide() {
		end = m.paneX() - paneGap
	}
	// On a narrow window the count goes first, then the line is cut, so it
	// never pushes the pane past the window's edge. At least one ─ ends it.
	if full := line + dimStyle.Render(count+" "); ansi.StringWidth(full) < end {
		line = full
	} else if ansi.StringWidth(line) >= end {
		line = ansi.Truncate(line, max(end-1, 0), "")
	}
	fill := end - ansi.StringWidth(line)
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
	line := columns(m.tableWidth()-rowStyle.GetHorizontalFrameSize(), strings.Repeat(" ", markerWidth),
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

// body is the list with its section line and column header, with the detail
// pane beside it, or over it when d opened it on a narrow window.
func (m Model) body() string {
	list := strings.Join([]string{m.header(), m.columnHeader(), m.list.View()}, "\n")
	height := lipgloss.Height(list)
	switch {
	case m.sideBySide():
		gap := strings.Repeat(" ", max(m.paneX()-lipgloss.Width(list), 0))
		return lipgloss.JoinHorizontal(lipgloss.Top, list, gap, m.detailView(m.paneWidth(), height))
	case m.detailOpen:
		return m.header() + "\n" + m.detailView(m.paneWidth(), height-1)
	}
	return list
}

func (m Model) View() tea.View {
	var content string
	switch {
	case m.splash:
		content = m.splashView()
	case m.state == stateScanning:
		content = messageStyle.Render("Scanning Steam libraries...")
	case m.state == stateFailed:
		content = messageStyle.Render(errorStyle.Render("Steam not found.") +
			"\n\nIf Steam is installed somewhere unusual, start Quarter with:\n  quarter --steam-path <steam folder>\n\n" +
			dimStyle.Render("q quit"))
	default:
		content = m.body()
		if m.showLogo() {
			content = m.withLogoBand(renderLogo()) + "\n" + content
		}
		content += "\n" + m.helpView()
		if f := m.footer(); f != "" {
			content += "\n" + f
		}
	}
	v := tea.NewView(content)
	v.AltScreen = true
	return v
}
