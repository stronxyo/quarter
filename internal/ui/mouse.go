package ui

// The mouse (Pablo, 2026-10-01, from the Mouse lab: "lets add it now v0.1"),
// for people who don't live in a terminal. The pointer lights up what it is
// over: its text in amber with a small square before it. A click acts on it:
// a game is selected, a double-click plays or installs it; tabs, keys on the
// key line, column names (sort, again to flip), the game's logo (the game in
// the Steam library; s is its store page), the quarter_ box (an intro) and
// the website in the details pane (2026-10-02) all work. A right-click
// stars a game.
// The wheel moves the selection, or over the tabs, turns them. On the HIGH
// SCORES calendar, a day under the pointer says what was played then.
// config.json's "mouse": "off" leaves the mouse to the terminal (plain text
// selection without Shift).

import (
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	"charm.land/bubbles/v2/list"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/steam"
)

const (
	doubleClick = 400 * time.Millisecond // two clicks this close on one target play it
	mouseOff    = "off"                  // config.json's "mouse" for leaving it to the terminal
	dayGames    = 3                      // games named for a calendar day, the rest counted

	refusalPoll   = 250 * time.Millisecond // how often Steam's log is looked at after a library link
	refusalMemory = 30 * 24 * time.Hour    // a game Steam refused is tried again after this
)

// refusalWatch is how long Steam's log is watched after a library link
// (tests shorten it).
var refusalWatch = 3 * time.Second

type targetKind int

const (
	tNone    targetKind = iota
	tItem               // a game in the list
	tRank               // a game in the history ranking
	tFilter             // a filter tab
	tPeriod             // a history period tab
	tKey                // a key on the key line, or the full list after ?
	tSearch             // the search box: typing again
	tSort               // a column's name
	tBand               // the game's logo
	tQuarter            // the quarter_ box
	tDay                // a day on the HIGH SCORES calendar
	tWebsite            // the website in the details pane
)

// target is something on screen the pointer can act on: row y, columns x0
// to x1.
type target struct {
	kind      targetKind
	y, x0, x1 int
	n         int    // which game, tab, column or day (days from the calendar's first)
	key       string // the key a key-line button presses
}

// pointer is the mouse's state.
type pointer struct {
	hover      target    // what the pointer is over
	lastClick  time.Time // for double-clicks
	lastTarget target
	cache      *targetCache
	layoutGen  uint64 // changes with every message but the pointer moving
	fromMouse  bool   // the key being handled was pressed by a click
}

// targetCache keeps the targets worked out last: the pointer moves far more
// often than anything moves on screen.
type targetCache struct {
	gen uint64
	t   []target
}

// mouseOn reports whether Quarter takes the mouse.
func (m Model) mouseOn() bool { return m.mouseSetting != mouseOff }

// targets lists what the pointer can act on, laid out as View draws it.
// They are worked out again only after something other than the pointer
// moving: a hover changes looks, never places.
func (m Model) targets() []target {
	if c := m.pointer.cache; c != nil && m.pointer.layoutGen != 0 && c.gen == m.pointer.layoutGen && c.t != nil {
		return c.t
	}
	t := m.workOutTargets()
	if c := m.pointer.cache; c != nil && m.pointer.layoutGen != 0 {
		*c = targetCache{gen: m.pointer.layoutGen, t: append(t, target{})[:len(t)]} // never nil: worked out, maybe none
	}
	return t
}

func (m Model) workOutTargets() []target {
	if m.splash || m.state != stateReady {
		return nil
	}
	top := 0
	if m.showLogo() {
		top = logoHeight()
	}
	body := m.body()
	lines := strings.Split(ansi.Strip(body), "\n")
	var out []target
	// label finds text on a line of the body and adds it as a target, with
	// the cell before and after.
	label := func(kind targetKind, row, n int, text string, extra int) {
		if x := strings.Index(lines[row], text); x >= 0 {
			x = ansi.StringWidth(lines[row][:x])
			out = append(out, target{kind: kind, y: top + row, x0: x - 1, x1: x + ansi.StringWidth(text) + 1 + extra, n: n})
		}
	}
	if m.showLogo() {
		for r := logoPadTop; r < logoHeight()-logoPadBottom; r++ {
			out = append(out, target{kind: tQuarter, y: r, x0: logoIndent, x1: logoIndent + logoBoxWidth})
		}
		if x, cols, rows, ok := m.logoBand(); ok && m.showsLogos() {
			for r := logoPadTop; r < logoPadTop+rows; r++ {
				out = append(out, target{kind: tBand, y: r, x0: x, x1: x + cols})
			}
		}
	}
	if m.histView == histOff {
		for i, f := range library.Filters() {
			label(tFilter, 0, i, f.String(), 0)
		}
		if prompt := strings.TrimSpace(m.list.FilterInput.Prompt); prompt != "" {
			if x := strings.Index(lines[0], prompt); x >= 0 {
				end := strings.Index(lines[0][x:], " ──")
				if end < 0 {
					end = len(lines[0]) - x
				}
				out = append(out, target{kind: tSearch, y: top, x0: ansi.StringWidth(lines[0][:x]), x1: ansi.StringWidth(lines[0][:x+end])})
			}
		}
		for i, c := range sortColumns {
			label(tSort, 1, i, c.name, 2) // and the ▲▼ after it
		}
		items := m.list.VisibleItems()
		per, page := m.list.Paginator.PerPage, m.list.Paginator.Page
		for i := range m.list.Paginator.ItemsOnPage(len(items)) {
			out = append(out, target{kind: tItem, y: top + 2 + i, x0: 0, x1: m.tableWidth(), n: page*per + i})
		}
		// First, so it wins over the game row under the pop-up pane.
		out = append(m.websiteTarget(lines, top), out...)
	} else {
		for i, p := range histPeriods {
			label(tPeriod, 0, i, p.label, 0)
		}
		r := m.ranking()
		per := m.rankPerPage()
		first := min(m.histSel, max(len(r)-1, 0)) / per * per
		for i := first; i < min(first+per, len(r)); i++ {
			out = append(out, target{kind: tRank, y: top + 2 + i - first, x0: 0, x1: m.tableWidth(), n: i})
		}
		out = append(out, m.dayTargets(lines, top)...)
	}
	return append(out, m.keyTargets(top+len(lines))...)
}

// websiteTarget is the website's name in the details pane, beside the list
// or in its pop-up, found on the row the pane labels Website: a click opens
// it, as w does, in any terminal (Quarter has the mouse, so the terminal's
// own link click doesn't reach it).
func (m Model) websiteTarget(lines []string, top int) []target {
	if m.selectedWebsite() == "" || (!m.sideBySide() && !m.detailOpen) {
		return nil
	}
	label := fmt.Sprintf("%-*s", paneLabelWidth, "Website") // the label as the pane pads it
	for i, l := range lines {
		x := strings.Index(l, label)
		if x < 0 || !strings.Contains(l[:x], "│") {
			continue // not the pane's row: the label sits inside its frame
		}
		start := x + len(label)
		value := l[start:]
		if end := strings.Index(value, "│"); end >= 0 {
			value = value[:end]
		}
		value = strings.TrimRight(value, " ")
		if value == "" {
			continue
		}
		x0 := ansi.StringWidth(l[:start])
		return []target{{kind: tWebsite, y: top + i, x0: x0 - 1, x1: x0 + ansi.StringWidth(value) + 1}}
	}
	return nil
}

// dayTargets are the board's calendar days: its Mon row found where the
// pane draws it, a cell a day, a column a week.
func (m Model) dayTargets(lines []string, top int) []target {
	if m.histView != histBoard || !m.sideBySide() {
		return nil
	}
	cx := m.paneX() + paneStyle.GetHorizontalFrameSize()/2
	for i, l := range lines {
		if ansi.StringWidth(l) > cx+4 && ansi.Cut(l, cx, cx+4) == "Mon " {
			var out []target
			for r := range 7 {
				for w := range calendarWeeks {
					out = append(out, target{kind: tDay, y: top + i + r, x0: cx + 4 + w, x1: cx + 5 + w, n: 7*w + r})
				}
			}
			return out
		}
	}
	return nil
}

// keyTargets are the keys on the key line, or on the full list after ?,
// starting at screen row y: each key with its words is a button.
func (m Model) keyTargets(y int) []target {
	_, state := m.stateOf()
	var all []key.Binding
	for _, set := range [][][]key.Binding{historyHelp(), {searchKeys(state)}, shortHelp(append([]key.Binding{helpClose}, m.gameKeys(state)...)), fullHelp(m.width)} {
		for _, line := range set {
			all = append(all, line...)
		}
	}
	var out []target
	for i, line := range strings.Split(ansi.Strip(m.helpView()), "\n") {
		found := map[int]target{} // by the key's place: the longest words win ("uninstall x2" over "uninstall")
		for _, b := range all {
			h := b.Help()
			if h.Key == "" || len(b.Keys()) == 0 {
				continue
			}
			k := wordAt(line, h.Key)
			if k < 0 {
				continue
			}
			// The key's words come next, after any gap: one space on the key
			// line, a column's padding on the full list.
			after := line[k+len(h.Key):]
			gap := len(after) - len(strings.TrimLeft(after, " "))
			if gap == 0 || !strings.HasPrefix(after[gap:], h.Desc) {
				continue
			}
			x0 := ansi.StringWidth(line[:k])
			x1 := ansi.StringWidth(line[:k+len(h.Key)+gap+len(h.Desc)])
			if prev, ok := found[k]; !ok || x1 > prev.x1 {
				found[k] = target{kind: tKey, y: y + i, x0: x0 - 1, x1: x1, key: b.Keys()[0]}
			}
		}
		for _, t := range found {
			out = append(out, t)
		}
	}
	return out
}

// wordAt is where w stands as a word of its own in line ("i" isn't the i in
// "uninstall"), or -1.
func wordAt(line, w string) int {
	for from := 0; from < len(line); {
		i := strings.Index(line[from:], w)
		if i < 0 {
			return -1
		}
		i += from
		end := i + len(w)
		if (i == 0 || line[i-1] == ' ') && (end == len(line) || line[end] == ' ') {
			return i
		}
		from = i + 1
	}
	return -1
}

func (m Model) targetAt(x, y int) target {
	for _, t := range m.targets() {
		if t.y == y && x >= t.x0 && x < t.x1 {
			return t
		}
	}
	return target{}
}

// keyFor is the key a binding names, as the terminal sends it.
func keyFor(k string) tea.KeyPressMsg {
	codes := map[string]rune{"enter": tea.KeyEnter, "tab": tea.KeyTab, "esc": tea.KeyEscape, "up": tea.KeyUp, "down": tea.KeyDown,
		"left": tea.KeyLeft, "right": tea.KeyRight, "pgup": tea.KeyPgUp, "pgdown": tea.KeyPgDown, "home": tea.KeyHome, "end": tea.KeyEnd, " ": tea.KeySpace}
	if c, ok := codes[k]; ok {
		return tea.KeyPressMsg{Code: c}
	}
	return tea.KeyPressMsg{Code: []rune(k)[0], Text: k}
}

// pressFor handles k as if it had been pressed, for a click.
func (m Model) pressFor(k string) (Model, tea.Cmd) {
	m.pointer.fromMouse = true
	tm, cmd := m.Update(keyFor(k))
	out := tm.(Model)
	out.pointer.fromMouse = false
	return out, cmd
}

// stopTyping leaves the search box, keeping its results and the game under
// the cursor: a click elsewhere while typing acts on what it clicks instead
// of typing its key.
func (m Model) stopTyping() Model {
	if !m.list.SettingFilter() {
		return m
	}
	if m.list.FilterInput.Value() == "" {
		m.list.ResetFilter()
		return m
	}
	i := m.list.Index()
	m.list.SetFilterState(list.FilterApplied)
	m.list.FilterInput.Blur()
	m.list.Select(i)
	return m
}

// searchKey reports whether k is one of the search box's own keys (enter,
// esc), which a click presses as is while typing.
func (m Model) searchKey(k string) bool {
	_, state := m.stateOf()
	return slices.ContainsFunc(searchKeys(state), func(b key.Binding) bool { return slices.Contains(b.Keys(), k) })
}

// dayText is a calendar day: its date, time played and games, most played
// first; "" for a day still to come.
func (m Model) dayText(n int) string {
	first, today := m.calendarDays(m.now())
	d := first.AddDate(0, 0, n)
	if d.After(today) {
		return ""
	}
	games := m.hist.DayGames(d)
	if len(games) == 0 {
		return d.Format("Mon 2 Jan 2006") + " · nothing played"
	}
	var total time.Duration
	var names []string
	for _, g := range games {
		total += g.Played
		if len(names) < dayGames {
			names = append(names, m.gameName(g.App))
		}
	}
	more := ""
	if len(games) > dayGames {
		more = fmt.Sprintf(" +%d", len(games)-dayGames)
	}
	return d.Format("Mon 2 Jan 2006") + " · " + playedText(total) + " · " + strings.Join(names, ", ") + more
}

// replayIntro plays a random intro, then the start screen: a click on the
// quarter_ box.
func (m Model) replayIntro() (Model, tea.Cmd) {
	m.splash, m.introDone, m.introRun = true, false, nil
	m.pickIntro(rand.New(rand.NewPCG(uint64(m.now().UnixNano()), 3)))
	m.startIntro()
	return m, tea.Batch(m.introTick(), m.blink())
}

// notInLibraryMsg says a game's library page can't be shown.
type notInLibraryMsg struct {
	id      int // the status it answers
	appID   uint32
	name    string
	refused bool // Steam's log said so: remembered
}

// refusalsMsg is every game Steam's log says it couldn't show in the library.
type refusalsMsg struct{ refused map[uint32]time.Time }

// loadRefusals reads, once, the games Steam has refused to show in the
// library, off the UI goroutine.
func (m *Model) loadRefusals() tea.Cmd {
	if m.refusalsRead || m.steamRoot == "" {
		return nil
	}
	m.refusalsRead = true
	root := m.steamRoot
	return func() tea.Msg {
		r, _ := steam.LibraryRefusals(root, time.Local) // no log: none known
		return refusalsMsg{r}
	}
}

// knownRefused reports whether Steam refused to show g in the library lately.
// An installed game is in the library, whatever the log says; and after
// refusalMemory the game is tried again (bought since, maybe).
func (m Model) knownRefused(g library.Game) bool {
	at, ok := m.refused[g.AppID]
	return ok && !g.Installed && m.now().Sub(at) < refusalMemory
}

// notInLibraryText says why a game's library page isn't shown.
func notInLibraryText(name string) string {
	return name + " isn't in your Steam library (a demo, a playtest, or a game no longer sold)"
}

// openInLibrary opens the game's page in the Steam library, off the UI
// goroutine (Pablo, 2026-10-01: some games didn't open, and Steam showed the
// last game instead). A game Steam keeps no library artwork for isn't opened
// at all, and the message line says so in red; any other gets the one link,
// then Steam's log is watched a moment: when Steam says it couldn't find the
// game, the message says so and the game is remembered, so it isn't opened
// again. (Opening the library's home first was tried and dropped: Steam
// flashed the home, then the last game, then the game.)
func openInLibrary(open func(string) error, steamPath string, g library.Game, id int, at time.Time) tea.Cmd {
	return func() tea.Msg {
		if steamPath != "" && !steam.InLibraryCache(steamPath, g.AppID) {
			return notInLibraryMsg{id: id, appID: g.AppID, name: g.Label()}
		}
		if open == nil {
			return openDoneMsg{id: id, err: fmt.Errorf("opening links is not available")}
		}
		if err := open(steam.LibraryURI(g.AppID)); err != nil || steamPath == "" {
			return openDoneMsg{id: id, err: err}
		}
		for waited := time.Duration(0); waited < refusalWatch; waited += refusalPoll {
			time.Sleep(refusalPoll)
			if steam.RefusedSince(steamPath, g.AppID, at, time.Local) {
				return notInLibraryMsg{id: id, appID: g.AppID, name: g.Label(), refused: true}
			}
		}
		return openDoneMsg{id: id}
	}
}

// tabsAt reports whether row y holds the filter or period tabs.
func (m Model) tabsAt(y int) bool {
	return slices.ContainsFunc(m.targets(), func(t target) bool { return (t.kind == tFilter || t.kind == tPeriod) && t.y == y })
}

// handleMouse handles the pointer moving, the wheel and clicks.
func (m Model) handleMouse(msg tea.Msg) (Model, tea.Cmd) {
	if !m.mouseOn() {
		return m, nil
	}
	switch msg := msg.(type) {
	case tea.MouseMotionMsg:
		t := m.targetAt(msg.X, msg.Y)
		if t == m.pointer.hover {
			m.viewQuiet = true // nothing new under the pointer
			return m, nil
		}
		m.pointer.hover = t
		if t.kind == tDay {
			if text := m.dayText(t.n); text != "" {
				return m, m.note(text)
			}
		}
		return m, nil

	case tea.MouseWheelMsg:
		up := msg.Button == tea.MouseWheelUp
		if !up && msg.Button != tea.MouseWheelDown {
			return m, nil
		}
		if m.tabsAt(msg.Y) { // over the tabs: the next or the one before
			steps := 1
			if up {
				steps = len(library.Filters()) - 1
				if m.histView != histOff {
					steps = len(histPeriods) - 1
				}
			}
			var cmd tea.Cmd
			for range steps {
				m, cmd = m.pressFor(keys.Filter.Keys()[0])
			}
			return m, cmd
		}
		if up {
			return m.pressFor("up")
		}
		return m.pressFor("down")

	case tea.MouseClickMsg:
		if m.splash {
			return m.pressFor(" ") // any key
		}
		t := m.targetAt(msg.X, msg.Y)
		if msg.Button == tea.MouseRight {
			if t.kind == tItem { // a favourite, without the f key
				m = m.stopTyping()
				m.list.Select(t.n)
				return m.pressFor(keys.Favourite.Keys()[0])
			}
			return m, nil
		}
		if msg.Button != tea.MouseLeft {
			return m, nil
		}
		if t.kind != tSearch && !(t.kind == tKey && m.searchKey(t.key)) {
			m = m.stopTyping()
		}
		double := t == m.pointer.lastTarget && m.now().Sub(m.pointer.lastClick) < doubleClick
		m.pointer.lastClick, m.pointer.lastTarget = m.now(), t
		switch t.kind {
		case tItem:
			if m.list.Index() == t.n && double {
				return m.pressFor(keys.Play.Keys()[0])
			}
			m.list.Select(t.n)
		case tRank:
			if m.histSel == t.n && double {
				return m.pressFor(keys.Play.Keys()[0])
			}
			m.histSel = t.n
		case tFilter, tPeriod:
			at := func() int { return int(m.filterBy) }
			if t.kind == tPeriod {
				at = func() int { return m.histPeriod }
			}
			var cmd tea.Cmd
			for range max(len(library.Filters()), len(histPeriods)) {
				if at() == t.n {
					break
				}
				m, cmd = m.pressFor(keys.Filter.Keys()[0])
			}
			return m, cmd
		case tSearch:
			if !m.list.SettingFilter() {
				i := m.list.Index()
				m.list.SetFilterState(list.Filtering)
				m.list.Select(i)
			}
		case tSort:
			for _, sk := range sortKeys {
				if sk.order == sortColumns[t.n].order {
					return m.pressFor(sk.binding.Keys()[0])
				}
			}
		case tBand: // the game in the Steam library (s is its store page)
			if g, ok := m.selected(); ok {
				if m.knownRefused(g) {
					return m, m.warn(notInLibraryText(g.Label()))
				}
				m.statusID++
				m.status, m.statusWrong = "Opening "+g.Label()+" in your Steam library", false
				return m, openInLibrary(m.open, m.steamRoot, g, m.statusID, m.now())
			}
		case tQuarter:
			return m.replayIntro()
		case tWebsite:
			return m.pressFor(keys.Website.Keys()[0])
		case tKey:
			return m.pressFor(t.key)
		}
	}
	return m, nil
}

// settleHover drops the hover when it no longer fits the screen: a key from
// the keyboard and a window resize clear it; after anything else it stays
// only while the same target is still under the pointer (a filter shortening
// the list). Moving the pointer brings it back.
func (m *Model) settleHover(msg tea.Msg) {
	if m.pointer.hover.kind == tNone {
		return
	}
	switch msg.(type) {
	case tea.MouseMotionMsg:
		return
	case tea.WindowSizeMsg:
		m.pointer.hover = target{}
		return
	case tea.KeyPressMsg:
		if !m.pointer.fromMouse {
			m.pointer.hover = target{}
			return
		}
	}
	if !slices.Contains(m.targets(), m.pointer.hover) {
		m.pointer.hover = target{}
	}
}

// withHover lights up what the pointer is over on the drawn screen: its text
// in amber with a small square in the cell before, keeping a space between
// (on the key line, the cell before that). Not the selected game or the
// active tab, already lit, and not the logos, pictures.
func (m Model) withHover(content string) string {
	t := m.pointer.hover
	switch {
	case t.kind == tNone || t.kind == tBand || t.kind == tQuarter,
		t.kind == tItem && t.n == m.list.Index(), t.kind == tRank && t.n == m.histSel,
		t.kind == tFilter && t.n == int(m.filterBy), t.kind == tPeriod && t.n == m.histPeriod:
		return content
	}
	lines := strings.Split(content, "\n")
	if t.y < 0 || t.y >= len(lines) {
		return content
	}
	line := lines[t.y]
	w := ansi.StringWidth(line)
	x0, x1 := max(t.x0, 0), min(t.x1, w)
	if x0 >= x1 {
		return content
	}
	plain := ansi.Strip(ansi.Cut(line, x0, x1))
	switch {
	case t.kind == tDay: // a calendar day: its square, bright
		lines[t.y] = ansi.Cut(line, 0, x0) + hoverDayStyle.Render(plain) + ansi.Cut(line, x1, w)
	case t.kind == tItem || t.kind == tRank: // the row's first cell, a space, is the mark's
		lines[t.y] = ansi.Cut(line, 0, x0) + hoverStyle.Render(markerHover+string([]rune(plain)[1:])) + ansi.Cut(line, x1, w)
	case x0 > 0 && ansi.Strip(ansi.Cut(line, x0-1, x0)) == " ": // keys and tabs: the cell before the space
		lines[t.y] = ansi.Cut(line, 0, x0-1) + hoverStyle.Render(markerHover+plain) + ansi.Cut(line, x1, w)
	default:
		lines[t.y] = ansi.Cut(line, 0, x0) + hoverStyle.Render(markerHover+string([]rune(plain)[1:])) + ansi.Cut(line, x1, w)
	}
	return strings.Join(lines, "\n")
}
