package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/config"
	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/steam"
	"github.com/stronxyo/quarter/internal/webapi"
)

// mouseGames are 40 games, a third installed.
func mouseGames() []library.Game {
	var games []library.Game
	for i := range 40 {
		games = append(games, library.Game{AppID: uint32(i + 1), Name: fmt.Sprintf("Game %02d", i+1), Installed: i%3 == 0})
	}
	return games
}

// mouseModel is the 40 games at w x h, on a clock the test moves.
func mouseModel(t *testing.T, w, h int) (Model, *time.Time) {
	t.Helper()
	m, _ := withOpener(t, nil, mouseGames()...)
	clock := time.Date(2026, 10, 1, 12, 0, 0, 0, time.Local)
	m.now = func() time.Time { return clock }
	tm, _ := tea.Model(m).Update(tea.WindowSizeMsg{Width: w, Height: h})
	return tm.(Model), &clock
}

// drawnLines is the drawn screen, without the hover or the kept-screen shortcut.
func drawnLines(m Model) []string {
	m.reuseView = false
	m.pointer.hover = target{}
	return strings.Split(ansi.Strip(m.View().Content), "\n")
}

func findTarget(t *testing.T, m Model, kind targetKind, n int, k string) target {
	t.Helper()
	for _, tg := range m.targets() {
		if tg.kind == kind && tg.n == n && tg.key == k {
			return tg
		}
	}
	t.Fatalf("no target %d/%d/%q among %v", kind, n, k, m.targets())
	return target{}
}

func clickOn(m Model, tg target, b tea.MouseButton) (Model, tea.Cmd) {
	tm, cmd := m.Update(tea.MouseClickMsg{X: (tg.x0 + tg.x1) / 2, Y: tg.y, Button: b})
	return tm.(Model), cmd
}

func pointAt(m Model, tg target) Model {
	tm, _ := m.Update(tea.MouseMotionMsg{X: (tg.x0 + tg.x1) / 2, Y: tg.y})
	return tm.(Model)
}

func typed(m Model, s string) Model {
	var tm tea.Model = m
	for _, r := range s {
		tm, _ = tm.Update(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
	return tm.(Model)
}

// Every target sits where the screen shows it: games, tabs, column names,
// keys, at a wide and a narrow window.
func TestTargetsMatchTheScreen(t *testing.T) {
	for _, size := range [][2]int{{150, 40}, {90, 30}} {
		m, _ := mouseModel(t, size[0], size[1])
		lines := drawnLines(m)
		var kinds = map[targetKind]int{}
		for _, tg := range m.targets() {
			kinds[tg.kind]++
			cell := ansi.Cut(lines[tg.y], max(tg.x0, 0), tg.x1)
			var want string
			switch tg.kind {
			case tItem:
				want = m.list.VisibleItems()[tg.n].(gameItem).Name
			case tFilter:
				want = library.Filters()[tg.n].String()
			case tSort:
				want = sortColumns[tg.n].name
			case tKey:
				want = strings.Fields(strings.TrimSpace(cell))[0]
				if b := keyFor(tg.key); b.Code == 0 {
					t.Errorf("key %q has no key press", tg.key)
				}
			}
			if !strings.Contains(cell, want) {
				t.Errorf("%dx%d: target %d/%d shows %q, want %q", size[0], size[1], tg.kind, tg.n, cell, want)
			}
		}
		if kinds[tItem] == 0 || kinds[tFilter] != 4 || kinds[tSort] != 3 || kinds[tKey] < 5 {
			t.Errorf("%dx%d: targets by kind %v", size[0], size[1], kinds)
		}
	}
}

// A click selects a game; a second click on it within doubleClick plays it;
// two slower clicks don't.
func TestDoubleClickPlays(t *testing.T) {
	m, clock := mouseModel(t, 150, 40)
	game := findTarget(t, m, tItem, 3, "") // Game 04, installed
	m, _ = clickOn(m, game, tea.MouseLeft)
	if m.list.Index() != 3 || m.status != "" {
		t.Fatalf("one click selects, at %d, status %q", m.list.Index(), m.status)
	}
	*clock = clock.Add(doubleClick + time.Millisecond)
	m, _ = clickOn(m, game, tea.MouseLeft)
	if m.status != "" {
		t.Errorf("a slow second click shouldn't play: %q", m.status)
	}
	*clock = clock.Add(doubleClick / 2)
	m, _ = clickOn(m, game, tea.MouseLeft)
	if !strings.Contains(m.status, "Launching Game 04") {
		t.Errorf("a double-click should play: %q", m.status)
	}
	other := findTarget(t, m, tItem, 1, "") // Game 02, not installed: a double-click installs
	m, _ = clickOn(m, other, tea.MouseLeft)
	m, _ = clickOn(m, other, tea.MouseLeft)
	if !strings.Contains(m.status, "Opening install for Game 02") {
		t.Errorf("a double-click on a game not installed should install it: %q", m.status)
	}
}

func TestClickTabsSortAndLogos(t *testing.T) {
	m, _ := mouseModel(t, 150, 40)
	m, _ = clickOn(m, findTarget(t, m, tFilter, int(library.FilterInstalled), ""), tea.MouseLeft)
	if m.filterBy != library.FilterInstalled {
		t.Errorf("Installed: got %v", m.filterBy)
	}
	m, _ = clickOn(m, findTarget(t, m, tFilter, int(library.FilterAll), ""), tea.MouseLeft)
	if m.filterBy != library.FilterAll {
		t.Errorf("All, round the tabs: got %v", m.filterBy)
	}
	m, _ = clickOn(m, findTarget(t, m, tSort, 0, ""), tea.MouseLeft)
	if m.sortBy != library.SortName {
		t.Fatalf("Name sorts by name: got %v", m.sortBy)
	}
	desc := m.sortDesc
	m, _ = clickOn(m, findTarget(t, m, tSort, 0, ""), tea.MouseLeft)
	if m.sortDesc == desc {
		t.Error("Name again flips the order")
	}
	if !m.showLogo() {
		t.Fatal("the big logo should show at 150x40")
	}
	m, _ = clickOn(m, findTarget(t, m, tQuarter, 0, ""), tea.MouseLeft)
	if !m.splash || m.introDone && m.introIndex >= 0 {
		t.Errorf("the quarter_ box brings an intro and the start screen: splash %v", m.splash)
	}
	tm, _ := m.Update(tea.MouseClickMsg{X: 1, Y: 1, Button: tea.MouseLeft}) // any click, like any key
	tm, _ = tm.Update(tea.MouseClickMsg{X: 1, Y: 1, Button: tea.MouseLeft})
	if tm.(Model).splash {
		t.Error("clicks should leave the start screen, as keys do")
	}
}

// The game's logo opens the game in the Steam library with one link (s is
// its store page); a game Steam keeps no library artwork for isn't opened,
// and the message line says so in red.
func TestClickLogoOpensLibrary(t *testing.T) {
	defer func(w time.Duration) { refusalWatch = w }(refusalWatch)
	refusalWatch = 2 * refusalPoll
	for _, inLibrary := range []bool{true, false} {
		logos := &fakeLogos{paths: map[uint32][]string{1: {logoPNG(t)}}}
		tm, _ := withLogos(t, logos, config.Settings{})
		m := tm.(Model)
		m.steamRoot = t.TempDir()
		if inLibrary {
			if err := os.MkdirAll(steam.LibraryCacheDir(m.steamRoot, 1), 0o755); err != nil {
				t.Fatal(err)
			}
		}
		var opened []string
		m.open = func(uri string) error { opened = append(opened, uri); return nil }
		band := findTarget(t, m, tBand, 0, "")
		m, cmd := m.handleMouse(tea.MouseClickMsg{X: band.x0 + 1, Y: band.y, Button: tea.MouseLeft})
		if cmd == nil {
			t.Fatal("the logo click should do something")
		}
		tm, _ = m.Update(cmd()) // the check and the links, as the program runs them
		m = tm.(Model)
		switch {
		case inLibrary && (len(opened) != 1 || opened[0] != steam.LibraryURI(1)):
			t.Errorf("in the library: want the game's page, once, got %v", opened)
		case inLibrary && m.status != "Opening Alpha in your Steam library":
			t.Errorf("in the library: status %q", m.status)
		case !inLibrary && len(opened) != 0:
			t.Errorf("not in the library: Steam shouldn't be opened, got %v", opened)
		case !inLibrary && (m.status != "Alpha isn't in your Steam library (a demo, a playtest, or a game no longer sold)" || !m.statusWrong):
			t.Errorf("not in the library: want the red message, got %q (red %v)", m.status, m.statusWrong)
		}
	}
}

// Steam couldn't find a game it keeps artwork for (Astro Burn): its log says
// so after the link, the message says so in red, and the next click doesn't
// open Steam at all.
func TestLogoClickLearnsRefusal(t *testing.T) {
	logos := &fakeLogos{paths: map[uint32][]string{1: {logoPNG(t)}}}
	tm, _ := withLogos(t, logos, config.Settings{})
	tm, _ = tm.Update(scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: []library.Game{{AppID: 1, Name: "Alpha"}}}}) // not installed: installed games are always in the library
	m := tm.(Model)
	m.steamRoot = t.TempDir()
	if err := os.MkdirAll(steam.LibraryCacheDir(m.steamRoot, 1), 0o755); err != nil {
		t.Fatal(err)
	}
	logPath := filepath.Join(m.steamRoot, "logs", "webhelper_js.txt")
	writeFile(t, logPath, "")
	var opened []string
	m.open = func(uri string) error { // Steam, refusing: its log line
		opened = append(opened, uri)
		f, _ := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY, 0o644)
		fmt.Fprintf(f, "[%s] SteamUI: ERROR: Unrecognized request details 1\n", time.Now().Format("2006-01-02 15:04:05"))
		return f.Close()
	}
	band := findTarget(t, m, tBand, 0, "")
	click := tea.MouseClickMsg{X: band.x0 + 1, Y: band.y, Button: tea.MouseLeft}
	m, cmd := m.handleMouse(click)
	tm, _ = m.Update(cmd())
	m = tm.(Model)
	if !m.statusWrong || m.status != notInLibraryText("Alpha") || len(opened) != 1 {
		t.Fatalf("the refusal should show in red: %q (red %v), opened %v", m.status, m.statusWrong, opened)
	}
	m, _ = m.handleMouse(click)
	if len(opened) != 1 || m.status != notInLibraryText("Alpha") {
		t.Errorf("the second click shouldn't open Steam: opened %v, status %q", opened, m.status)
	}
}

// The games Steam has refused before are read at start: not opened, unless
// installed since, or refused long ago.
func TestRefusalsReadAtStart(t *testing.T) {
	m, clock := mouseModel(t, 150, 40)
	m.steamRoot = t.TempDir()
	stamp := func(d time.Time) string { return d.Format("2006-01-02 15:04:05") }
	writeFile(t, filepath.Join(m.steamRoot, "logs", "webhelper_js.txt"),
		"["+stamp(clock.Add(-time.Hour))+"] SteamUI: ERROR: Unrecognized request details 2\n"+
			"["+stamp(clock.Add(-time.Hour))+"] SteamUI: ERROR: Unrecognized request details 1\n"+
			"["+stamp(clock.Add(-refusalMemory-time.Hour))+"] SteamUI: ERROR: Unrecognized request details 3\n")
	tm, _ := m.Update(m.loadRefusals()())
	m = tm.(Model)
	games := map[uint32]library.Game{}
	for _, g := range mouseGames() {
		games[g.AppID] = g
	}
	if !m.knownRefused(games[2]) {
		t.Error("game 2, refused an hour ago, shouldn't be opened")
	}
	if m.knownRefused(games[1]) {
		t.Error("game 1 is installed: in the library, whatever the log says")
	}
	if m.knownRefused(games[3]) {
		t.Error("game 3 was refused long ago: tried again")
	}
	if m.loadRefusals() != nil {
		t.Error("the log is read once")
	}
}

// A right-click stars a game.
func TestRightClickFavourite(t *testing.T) {
	m, _ := mouseModel(t, 150, 40)
	m, _ = clickOn(m, findTarget(t, m, tItem, 4, ""), tea.MouseRight)
	g, _ := m.selected()
	if m.list.Index() != 4 || !g.Favourite {
		t.Errorf("game 5 should be selected and starred: at %d, %v", m.list.Index(), g.Favourite)
	}
}

// After ?, all 25 keys of the full list click, each like its key; history's
// key line and periods click too.
func TestEveryKeyClicks(t *testing.T) {
	for _, w := range []int{150, 90} {
		fresh := func() Model { // favourites and settings are shared by copies: each try starts anew
			m, _ := mouseModel(t, w, 40)
			tm, _ := m.Update(press("?"))
			return tm.(Model)
		}
		var keysShown []target
		for _, tg := range fresh().targets() {
			if tg.kind == tKey {
				keysShown = append(keysShown, tg)
			}
		}
		if len(keysShown) != 25 {
			t.Errorf("%d wide: %d keys clickable on the full list, want 25", w, len(keysShown))
		}
		for _, tg := range keysShown {
			clicked, _ := clickOn(fresh(), tg, tea.MouseLeft)
			pressed, _ := fresh().Update(keyFor(tg.key))
			if a, b := drawnLines(clicked), drawnLines(pressed.(Model)); strings.Join(a, "\n") != strings.Join(b, "\n") {
				t.Errorf("%d wide: clicking %q differs from pressing it", w, tg.key)
			}
		}
	}

	m, _ := mouseModel(t, 150, 40)
	tm, _ := m.Update(press("h"))
	m = tm.(Model)
	var periods, histKeys int
	for _, tg := range m.targets() {
		switch tg.kind {
		case tPeriod:
			periods++
		case tKey:
			histKeys++
		}
	}
	if periods != len(histPeriods) || histKeys != 9 { // 8 keys and c buy me a coffee in the corner
		t.Errorf("history: %d periods and %d keys clickable, want %d and 9", periods, histKeys, len(histPeriods))
	}
	m, _ = clickOn(m, findTarget(t, m, tPeriod, 2, ""), tea.MouseLeft)
	if m.histPeriod != 2 {
		t.Errorf("3 months: at %d", m.histPeriod)
	}
}

// While typing a search, a click elsewhere ends the typing and keeps the
// results instead of typing its key; the box takes the typing back; esc
// clear, clicked, clears.
func TestSearchClicks(t *testing.T) {
	m, _ := mouseModel(t, 150, 40)
	m = typed(m, "/game 0")
	m, _ = clickOn(m, findTarget(t, m, tFilter, int(library.FilterInstalled), ""), tea.MouseLeft)
	if m.list.SettingFilter() || m.list.FilterInput.Value() != "game 0" || m.filterBy != library.FilterInstalled {
		t.Errorf("typing %v, search %q, filter %v", m.list.SettingFilter(), m.list.FilterInput.Value(), m.filterBy)
	}
	m, _ = clickOn(m, findTarget(t, m, tSearch, 0, ""), tea.MouseLeft)
	if !m.list.SettingFilter() {
		t.Error("the search box takes the typing back")
	}
	m, _ = clickOn(m, findTarget(t, m, tKey, 0, "esc"), tea.MouseLeft)
	if m.list.SettingFilter() || m.list.IsFiltered() {
		t.Error("esc clear, clicked, clears the search")
	}
}

// The wheel moves the selection; over the tabs, it turns them.
func TestWheel(t *testing.T) {
	m, _ := mouseModel(t, 150, 40)
	tabs := findTarget(t, m, tFilter, 0, "")
	tm, _ := m.Update(tea.MouseWheelMsg{X: 3, Y: tabs.y, Button: tea.MouseWheelDown})
	if tm.(Model).filterBy != library.FilterInstalled {
		t.Errorf("down over the tabs: the next one, got %v", tm.(Model).filterBy)
	}
	tm, _ = tm.Update(tea.MouseWheelMsg{X: 3, Y: tabs.y, Button: tea.MouseWheelUp})
	if tm.(Model).filterBy != library.FilterAll {
		t.Errorf("up over the tabs: back, got %v", tm.(Model).filterBy)
	}
	row := findTarget(t, tm.(Model), tItem, 0, "")
	tm, _ = tm.Update(tea.MouseWheelMsg{X: 3, Y: row.y, Button: tea.MouseWheelDown})
	if tm.(Model).list.Index() != 1 {
		t.Errorf("down over the list moves the selection, at %d", tm.(Model).list.Index())
	}
}

// A day on the HIGH SCORES calendar says what was played then.
func TestCalendarDayUnderPointer(t *testing.T) {
	m, clock := mouseModel(t, 170, 45)
	day := time.Date(clock.Year(), clock.Month(), clock.Day(), 9, 0, 0, 0, time.Local)
	sessions := []steam.Session{
		{AppID: 1, Start: day, End: day.Add(90 * time.Minute), EndKnown: true},
		{AppID: 4, Start: day.Add(2 * time.Hour), End: day.Add(150 * time.Minute), EndKnown: true},
	}
	tm, _ := m.Update(sessionsMsg{all: sessions})
	tm, _ = tm.Update(press("h"))
	tm, _ = tm.Update(press("h"))
	m = tm.(Model)
	first, _ := m.calendarDays(m.now())
	today := findTarget(t, m, tDay, int(day.Sub(first).Hours()/24+0.5), "")
	m = pointAt(m, today)
	if want := day.Format("Mon 2 Jan 2006") + " · 2.0h · Game 01, Game 04"; m.status != want {
		t.Errorf("the day under the pointer: %q, want %q", m.status, want)
	}
}

// The hover: the text in amber with ▪ before it, a space between, on a
// game, a tab and a key; the line keeps its width; not the selected game
// or the active tab.
func TestHoverLook(t *testing.T) {
	m, _ := mouseModel(t, 150, 40)
	for _, tg := range []target{findTarget(t, m, tItem, 5, ""), findTarget(t, m, tFilter, 2, ""), findTarget(t, m, tKey, 0, "/")} {
		lit := pointAt(m, tg)
		before, after := drawnLines(m)[tg.y], strings.Split(ansi.Strip(lit.View().Content), "\n")[tg.y]
		if ansi.StringWidth(after) != ansi.StringWidth(before) {
			t.Errorf("target %d: the line's width changed", tg.kind)
		}
		if !strings.Contains(after, markerHover+" ") || !strings.Contains(lit.View().Content, "\x1b[38;2;255;175;99m") {
			t.Errorf("target %d: want %q and a space, in amber: %q", tg.kind, markerHover, after)
		}
	}
	for _, tg := range []target{findTarget(t, m, tItem, 0, ""), findTarget(t, m, tFilter, int(m.filterBy), "")} {
		if lit := pointAt(m, tg); strings.Contains(ansi.Strip(lit.View().Content), markerHover) {
			t.Errorf("the selected game and the active tab are lit already: no hover on %d", tg.kind)
		}
	}
}

// The hover goes when it no longer fits: the arrow keys, a resize, a list
// shorter than the row under the pointer.
func TestHoverClears(t *testing.T) {
	m, _ := mouseModel(t, 150, 40)
	m = pointAt(m, findTarget(t, m, tItem, 15, ""))
	if m.pointer.hover.kind != tItem {
		t.Fatal("the pointer on game 16 lights it")
	}
	for name, msg := range map[string]tea.Msg{"an arrow key": tea.KeyPressMsg{Code: tea.KeyDown}, "a resize": tea.WindowSizeMsg{Width: 120, Height: 35}} {
		if tm, _ := m.Update(msg); tm.(Model).pointer.hover.kind != tNone {
			t.Errorf("%s should clear the hover", name)
		}
	}
	shorter, _ := clickOn(m, findTarget(t, m, tFilter, int(library.FilterInstalled), ""), tea.MouseLeft) // 14 games: row 16 is empty
	if shorter.pointer.hover.kind != tNone {
		t.Errorf("a list shorter than the hovered row should clear it: %v", shorter.pointer.hover)
	}
}

// "mouse": "off" leaves the mouse to the terminal: no reports asked for,
// clicks ignored, and the setting kept.
func TestMouseOff(t *testing.T) {
	saved := &[]config.Settings{}
	m := New(Options{Settings: config.Settings{Mouse: mouseOff}, SaveSettings: func(s config.Settings) error { *saved = append(*saved, s); return nil }})
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 150, Height: 40})
	tm, _ = tm.Update(scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: mouseGames()}})
	if mode := tm.(Model).View().MouseMode; mode != tea.MouseModeNone {
		t.Errorf("off: mouse mode %v, want none", mode)
	}
	if on, _ := mouseModel(t, 150, 40); on.View().MouseMode != tea.MouseModeAllMotion {
		t.Error("on, by default: the pointer's moves are asked for")
	}
	tm, _ = tm.Update(tea.MouseClickMsg{X: 5, Y: 20, Button: tea.MouseLeft})
	if tm.(Model).list.Index() != 0 {
		t.Error("off: a click does nothing")
	}
	if s := tm.(Model).settings(); s.Mouse != mouseOff {
		t.Errorf("the setting should be kept: %q", s.Mouse)
	}
}

// The website in the details pane clicks like w and lights up under the
// pointer; without a website there is nothing to click.
func TestWebsiteClicks(t *testing.T) {
	for _, w := range []int{150, 90} {
		m, opened := withOpener(t, nil, mouseGames()...)
		g, _ := m.selected()
		m.storeInfo[g.AppID] = webapi.StoreEntry{Info: webapi.StoreInfo{Publishers: []string{"Valve"}, Website: "https://www.example.com/"}, Fetched: time.Now()}
		tm, _ := tea.Model(m).Update(tea.WindowSizeMsg{Width: w, Height: 40})
		m = tm.(Model)
		if !m.sideBySide() {
			tm, _ = m.Update(press("d")) // the pop-up pane
			m = tm.(Model)
		}
		tg := findTarget(t, m, tWebsite, 0, "")
		lines := drawnLines(m)
		if got := ansi.Cut(lines[tg.y], tg.x0+1, tg.x1-1); got != "example.com "+markerLink { // the name and its arrow
			t.Errorf("%d wide: the target covers %q, want the website's name", w, got)
		}
		if lit := ansi.Strip(pointAt(m, tg).View().Content); !strings.Contains(strings.Split(lit, "\n")[tg.y], markerHover+" example.com") { // the mark, then the space before it, as on tabs
			t.Errorf("%d wide: the website should light up under the pointer", w)
		}
		clicked, cmd := clickOn(m, tg, tea.MouseLeft)
		for _, msg := range messages(cmd) {
			clicked.Update(msg)
		}
		if len(*opened) != 1 || (*opened)[0] != "https://www.example.com/" || !strings.Contains(clicked.status, "Opening example.com") {
			t.Errorf("%d wide: a click should open the website: opened %v, status %q", w, *opened, clicked.status)
		}

		delete(m.storeInfo, g.AppID)
		m.storeInfo[g.AppID] = webapi.StoreEntry{Info: webapi.StoreInfo{Publishers: []string{"Valve"}}, Fetched: time.Now()}
		m.pointer.cache = &targetCache{}
		for _, tg := range m.targets() {
			if tg.kind == tWebsite {
				t.Errorf("%d wide: no website, nothing to click", w)
			}
		}
	}
}

// "c buy me a coffee ↗" ends the key line, flush with the right edge, when
// there is room after the keys; for it the arrows' hints give way, never a
// key. A click opens Ko-fi; it lights up under the pointer.
func TestSupportLink(t *testing.T) {
	for _, tc := range []struct {
		w            int
		link, arrows bool
	}{{100, false, false}, {141, false, true}, {142, true, false}, {161, true, false}, {162, true, true}, {200, true, true}} {
		m, opened := withOpener(t, nil, mouseGames()...)
		g, _ := m.selected()
		m.storeInfo[g.AppID] = webapi.StoreEntry{Info: webapi.StoreInfo{Website: "https://www.example.com/"}, Fetched: time.Now()}
		tm, _ := tea.Model(m).Update(tea.WindowSizeMsg{Width: tc.w, Height: 40})
		m = tm.(Model)
		lines := drawnLines(m)
		bar := lines[len(lines)-1]
		if got := strings.HasSuffix(bar, "c "+supportText); got != tc.link {
			t.Errorf("%d wide: link at the right edge %v, want %v: %q", tc.w, got, tc.link, bar)
		}
		if got := strings.Contains(bar, "move"); got != tc.arrows {
			t.Errorf("%d wide: arrows' hints %v, want %v: %q", tc.w, got, tc.arrows, bar)
		}
		if ansi.StringWidth(bar) > tc.w {
			t.Errorf("%d wide: the key line is %d wide", tc.w, ansi.StringWidth(bar))
		}
		if tc.link {
			for _, k := range []string{"enter play", "u uninstall", "w website", "b steamdb", "/ search", "n/p/l sort", "tab filter", "h history", "? more", "q quit"} {
				if !strings.Contains(bar, k) {
					t.Errorf("%d wide: the link took the place of %q", tc.w, k)
				}
			}
			tg := findTarget(t, m, tKey, 0, "c")
			if lit := ansi.Strip(pointAt(m, tg).View().Content); !strings.Contains(lit, markerHover+" c "+supportText) { // the mark before the key, as on every key
				t.Errorf("%d wide: the link should light up under the pointer", tc.w)
			}
			clicked, cmd := clickOn(m, tg, tea.MouseLeft)
			for _, msg := range messages(cmd) {
				clicked.Update(msg)
			}
			if len(*opened) != 1 || (*opened)[0] != config.SupportURL || !strings.Contains(clicked.status, "Ko-fi") {
				t.Errorf("%d wide: a click should open Ko-fi: opened %v, status %q", tc.w, *opened, clicked.status)
			}
		}
	}
}

// c opens Ko-fi on the list and in history, with or without the link on
// show; typed into a search it is a letter.
func TestSupportKey(t *testing.T) {
	for _, keys := range [][]string{{"c"}, {"h", "c"}} {
		m, opened := withOpener(t, nil, mouseGames()...)
		var tm tea.Model = m
		for _, k := range keys {
			tm = act(tm, press(k))
		}
		if len(*opened) != 1 || (*opened)[0] != config.SupportURL || !strings.Contains(tm.(Model).status, "Ko-fi") {
			t.Errorf("%v: c should open Ko-fi: opened %v, status %q", keys, *opened, tm.(Model).status)
		}
	}
	m, opened := withOpener(t, nil, mouseGames()...)
	m = typed(m, "/c")
	if len(*opened) != 0 || m.list.FilterInput.Value() != "c" {
		t.Errorf("c in a search is a letter: opened %v, search %q", *opened, m.list.FilterInput.Value())
	}
}
