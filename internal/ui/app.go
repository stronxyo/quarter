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
	"github.com/stronxyo/quarter/internal/steam"
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
	steamPath   string // --steam-path override; empty means detect
	state       state
	err         error
	warnings    []error                      // Steam files the last scan couldn't read
	missing     []steam.LibraryFolder        // libraries whose drive isn't connected
	noAccount   bool                         // no one has signed in to Steam here: no playtime
	scanning    bool                         // a background scan is running
	scanErr     error                        // a rescan failed while a cached library is shown
	save        func(library.Result) error   // stores a fresh scan for the next launch; may be nil
	record      func(steamPath string) error // saves the game sessions in Steam's log; may be nil
	open        func(uri string) error       // hands a steam:// link to the OS
	status      string                       // short-lived footer message after an action
	statusID    int                          // tells a stale clear timer from the current message
	statusWrong bool                         // the status says an action failed
	fullHelp    bool                         // ? pressed: every key is listed

	detailOpen    bool // d pressed on a narrow window: the pane covers the list
	paneHidden    bool // d pressed on a wide window: the list takes the whole width
	playtimeKnown bool // false when Steam's playtime file couldn't be read

	fetchStore   func(context.Context, uint32) (webapi.StoreInfo, error) // may be nil: no store details
	saveStore    func(map[uint32]webapi.StoreEntry) error
	storeInfo    map[uint32]webapi.StoreEntry
	fetching     uint32          // the game whose store request is out, 0 for none
	wanted       uint32          // the selected game, waiting for its turn
	storePause   pause           // store requests wait after a network failure
	serverFailed map[uint32]bool // the store failed on these this run: the background fill skips them
	debounce     time.Duration   // how long the selection must settle before fetching

	netDown bool                              // the last request got no answer: "offline" shows
	jitter  func(time.Duration) time.Duration // spreads the waits after failures

	runningApp func() (uint32, bool) // asks Steam which game is running; may be nil
	running    uint32                // the game Steam is running, 0 for none
	afterPlay  afterPlay             // a game that stopped, waiting for Steam to save its playtime

	updateWatch bool // a read of the library is scheduled while Steam updates a game

	newLogos     func(steamPath string) LogoSource // may be nil: no game logos
	logos        LogoSource                        // set once the Steam folder is known
	logosHidden  bool                              // o pressed: the pane shows names only
	logoBlocks   art.Blocks
	logoCache    map[logoKey]drawnLogo // drawn logos, by game and size
	logoPending  map[logoKey]bool      // being drawn, or waiting for a download
	logoNone     map[uint32]bool       // no logo anywhere, this run
	logoPause    pause                 // downloads wait after a network failure
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

	RecordSessions func(steamPath string) error // saves the game sessions in Steam's log, after each scan

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
		record:    opts.RecordSessions,
		open:      opts.Open,
		list:      newList(),
		now:       time.Now,
		scanning:  true,

		favs:         make(map[uint32]bool),
		saveSettings: opts.SaveSettings,

		fetchStore:   opts.FetchStore,
		saveStore:    opts.SaveStore,
		storeInfo:    make(map[uint32]webapi.StoreEntry),
		serverFailed: make(map[uint32]bool),
		jitter:       halfJitter,
		debounce:     storeDebounce,
		runningApp:   opts.RunningApp,

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
			m.statusWrong = true
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

	case resumeMsg:
		return m.resume(msg)

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

	case updatePollMsg:
		return m.updateTick()

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
	m.warnings, m.missing, m.noAccount = msg.res.Warnings, msg.res.Missing, msg.res.NoAccount
	m.installed, m.uninstalled = msg.res.Games, msg.res.Uninstalled
	m.setSteamPath(msg.res.SteamPath)
	m.rebuild()
	m.playtimeKnown = msg.res.PlaytimeKnown
	m.list.SetDelegate(m.delegate())
	cmd := tea.Batch(m.refreshList(), m.nextStoreFetch(), m.watchUpdates())
	if m.save != nil {
		cmd = tea.Batch(cmd, saveScan(m.save, msg.res))
	}
	if m.record != nil {
		cmd = tea.Batch(cmd, recordSessions(m.record, msg.res.SteamPath))
	}
	return m, cmd
}

// recordSessions saves the sessions in Steam's log off the UI goroutine. A
// failure only delays the history until the next scan, so it is logged.
func recordSessions(record func(string) error, steamPath string) tea.Cmd {
	return func() tea.Msg {
		if err := record(steamPath); err != nil {
			slog.Warn("sessions", "err", err)
		}
		return nil
	}
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
	counted := dimStyle.Render(count + " ")
	if m.netDown {
		counted = dimStyle.Render(count+" · ") + alertStyle.Render("offline") + " "
	}
	// Beside the pane, the line runs on to it across any gap.
	end := m.tableWidth()
	if m.sideBySide() {
		end = m.paneX() - paneGap
	}
	// On a narrow window the count goes first, then the line is cut, so it
	// never pushes the pane past the window's edge. At least one ─ ends it.
	if full := line + counted; ansi.StringWidth(full) < end {
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

// footer is the message line under the key help, when the pane isn't beside
// the list to carry it (see paneNotice). One line, cut to the window.
func (m Model) footer() string {
	text, style := m.notice()
	if text == "" || m.sideBySide() {
		return ""
	}
	return m.fitLine("  " + style.Render(text))
}

// notice is the message to show, most important first: an action's message,
// a failed rescan, libraries not connected, Steam files that could not be
// read, or why there's no playtime; with its style: light red when it says
// something is wrong.
func (m Model) notice() (string, lipgloss.Style) {
	text, wrong := m.noticeText()
	switch {
	case wrong:
		return text, alertStyle
	case m.status != "":
		return text, activeTabStyle
	}
	return text, dimStyle
}

func (m Model) noticeText() (text string, wrong bool) {
	switch {
	case m.status != "":
		return m.status, m.statusWrong
	case m.scanErr != nil:
		return "Could not rescan Steam, showing the last saved library. Press r to retry.", true
	case len(m.missing) > 0:
		return missingText(m.missing), true
	case len(m.warnings) > 0:
		return warningText(m.warnings), true
	case m.noAccount:
		return "Sign in to Steam once to see playtime.", false
	}
	return "", false
}

// paneNotice is the message under the pane, wrapped to its width, at most
// three lines.
func (m Model) paneNotice(width int) string {
	text, style := m.notice()
	if text == "" {
		return ""
	}
	lines := strings.Split(lipgloss.NewStyle().Width(max(width-1, 1)).Render(text), "\n")
	if len(lines) > 3 {
		lines = lines[:3]
		lines[2] = ansi.Truncate(lines[2], max(width-2, 1), "…")
	}
	for i, l := range lines {
		lines[i] = " " + style.Render(strings.TrimRight(l, " "))
	}
	return strings.Join(lines, "\n")
}

// fitLine cuts a line to the window's width, once the width is known.
func (m Model) fitLine(s string) string {
	if m.width <= 0 {
		return s
	}
	return ansi.Truncate(s, m.width, "…")
}

// listView is the list, or when it shows nothing, why, in the rows it takes.
func (m Model) listView() string {
	view := m.list.View()
	if len(m.list.VisibleItems()) > 0 {
		return view
	}
	msg := rowStyle.Render(dimStyle.Render(ansi.Truncate(m.emptyMessage(), max(m.tableWidth()-rowStyle.GetHorizontalFrameSize(), 1), "…")))
	return msg + strings.Repeat("\n", max(lipgloss.Height(view)-1, 0))
}

// emptyMessage says why the list is empty and what to do about it.
func (m Model) emptyMessage() string {
	switch {
	case m.list.SettingFilter() || m.list.IsFiltered():
		return "No games match."
	case len(m.games) == 0 || m.filterBy == library.FilterInstalled:
		return "No games installed yet. Install one in Steam, then press r."
	case m.filterBy == library.FilterRecent:
		return fmt.Sprintf("Nothing played in the last %d days.", int(config.RecentWindow.Hours()/24))
	case m.filterBy == library.FilterFavourites:
		return "No favourites yet. Press f on a game to star it."
	}
	return "No games to show."
}

// body is the list with its section line and column header, with the detail
// pane beside it, or over it when d opened it on a narrow window.
func (m Model) body() string {
	list := strings.Join([]string{m.header(), m.columnHeader(), m.listView()}, "\n")
	height := lipgloss.Height(list)
	switch {
	case m.sideBySide():
		gap := strings.Repeat(" ", max(m.paneX()-lipgloss.Width(list), 0))
		pane := m.detailView(m.paneWidth(), height)
		if n := m.paneNotice(m.paneWidth()); n != "" {
			pane = m.detailView(m.paneWidth(), height-lipgloss.Height(n)) + "\n" + n
		}
		return lipgloss.JoinHorizontal(lipgloss.Top, list, gap, pane)
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
		content = m.failedView()
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
