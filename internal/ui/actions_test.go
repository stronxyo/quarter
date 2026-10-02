package ui

import (
	"errors"
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/webapi"
)

// withOpener returns a ready model whose link opener records what it was
// given, or fails with err when err is set.
func withOpener(t *testing.T, err error, games ...library.Game) (Model, *[]string) {
	t.Helper()
	opened := &[]string{}
	m := New(Options{Open: func(uri string) error {
		*opened = append(*opened, uri)
		return err
	}})
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 80, Height: 30})
	tm, _ = tm.Update(scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: games}})
	return tm.(Model), opened
}

// act presses a key and feeds the result of the open back in, the way the
// program would, without waiting for the message's clear timer.
func act(m tea.Model, k tea.KeyPressMsg) tea.Model {
	m, cmd := m.Update(k)
	if cmd == nil {
		return m
	}
	if msg, ok := cmd().(openDoneMsg); ok {
		m, _ = m.Update(msg) // its clear timer is returned, not run
	}
	return m
}

var portal2 = library.Game{AppID: 620, Name: "Portal 2", Installed: true}

func TestActionKeys(t *testing.T) {
	tests := []struct {
		name    string
		game    library.Game
		key     tea.KeyPressMsg
		wantURI string
		status  string
	}{
		{"enter plays", portal2, tea.KeyPressMsg{Code: tea.KeyEnter}, "steam://rungameid/620", "Launching Portal 2"},
		{"i installs a game that isn't installed", library.Game{AppID: 400, Name: "Portal"}, press("i"), "steam://install/400", "install for Portal"},
		{"s opens the store", portal2, press("s"), "steam://store/620", "store page for Portal 2"},
		{"enter on an uninstalled game installs", library.Game{AppID: 400, Name: "Portal"}, tea.KeyPressMsg{Code: tea.KeyEnter}, "steam://install/400", "install for Portal"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m, opened := withOpener(t, nil, tt.game)
			got := act(m, tt.key)
			if len(*opened) != 1 || (*opened)[0] != tt.wantURI {
				t.Errorf("opened %v, want [%s]", *opened, tt.wantURI)
			}
			if view := ansi.Strip(got.(Model).View().Content); !strings.Contains(view, tt.status) {
				t.Errorf("footer missing %q", tt.status)
			}
		})
	}
}

func TestActionKeysIgnoredWhileTypingSearch(t *testing.T) {
	m, opened := withOpener(t, nil, portal2)
	var tm tea.Model = m
	tm = act(tm, press("/"))
	tm = act(tm, press("s"))
	act(tm, press("i"))
	if len(*opened) != 0 {
		t.Errorf("keys typed into the search box opened %v", *opened)
	}
}

func TestFailedOpenShowsError(t *testing.T) {
	m, _ := withOpener(t, errors.New("no handler"), portal2)
	got := act(m, tea.KeyPressMsg{Code: tea.KeyEnter})
	if view := ansi.Strip(got.(Model).View().Content); !strings.Contains(view, "Could not open Steam: no handler") {
		t.Errorf("error not shown:\n%s", view)
	}
}

func TestStatusClears(t *testing.T) {
	m, _ := withOpener(t, nil, portal2)
	var tm tea.Model = act(m, press("s"))
	first := tm.(Model).statusID
	tm = act(tm, press("s"))

	tm, _ = tm.Update(clearStatusMsg{id: first}) // the older timer fires
	if tm.(Model).status == "" {
		t.Error("an older timer must not clear a newer message")
	}
	tm, _ = tm.Update(clearStatusMsg{id: tm.(Model).statusID})
	if tm.(Model).status != "" {
		t.Error("the current timer should clear the message")
	}
}

func TestHelpToggle(t *testing.T) {
	m, _ := withOpener(t, nil, portal2)
	short := ansi.Strip(m.View().Content)
	if !strings.Contains(short, "enter play") || !strings.Contains(short, "u uninstall") || strings.Contains(short, "i install") || strings.Contains(short, "store page") {
		t.Errorf("short help for an installed game should show enter and u, not i or s:\n%s", short)
	}
	var tm tea.Model = act(m, press("?"))
	full := ansi.Strip(tm.(Model).View().Content)
	for _, want := range []string{"i", "install", "store page", "rescan"} {
		if !strings.Contains(full, want) {
			t.Errorf("full help missing %q", want)
		}
	}
}

// The key line always fits: on narrow windows the arrows give way, move
// first, then h history, the sort keys, and w website and b steamdb (all
// still listed under ?).
func TestShortHelpFits(t *testing.T) {
	for _, tc := range []struct {
		w                       int
		move, page, hist, links bool
	}{ // three spaces between keys since the mouse (2026-10-01); w and b since 2026-10-02
		{60, false, false, false, false}, {80, false, false, false, false}, {91, false, false, false, false},
		{92, false, false, false, true}, {116, false, false, false, true}, {117, false, false, true, true},
		{126, false, false, true, true}, {127, false, true, true, true}, {137, true, true, true, true},
	} {
		w := tc.w
		m, _ := withOpener(t, nil, portal2)
		m.storeInfo[620] = webapi.StoreEntry{Info: webapi.StoreInfo{Website: "https://www.thinkwithportals.com/"}, Fetched: time.Now()}
		var tm tea.Model = m
		tm, _ = tm.Update(tea.WindowSizeMsg{Width: w, Height: 30})
		line := lineWith(ansi.Strip(tm.(Model).View().Content), "q quit")
		if !strings.Contains(line, "enter play") || !strings.Contains(line, "u uninstall") {
			t.Fatalf("width %d: the key line should hold the game's keys through q quit: %q", w, line)
		}
		if got := ansi.StringWidth(line); got > w {
			t.Errorf("width %d: key line is %d wide", w, got)
		}
		links := strings.Contains(line, "w website") && strings.Contains(line, "b steamdb")
		if strings.Contains(line, "move") != tc.move || strings.Contains(line, "page") != tc.page || strings.Contains(line, "h history") != tc.hist || links != tc.links {
			t.Errorf("width %d: want move %v, page %v, history %v, links %v: %q", w, tc.move, tc.page, tc.hist, tc.links, line)
		}
	}
}

// w opens the game's website once the store has given it, b its SteamDB
// page; both for a running game too. Without a website, w says why.
func TestWebsiteAndSteamDBKeys(t *testing.T) {
	m, opened := withOpener(t, nil, portal2)
	m.storeInfo[620] = webapi.StoreEntry{Info: webapi.StoreInfo{Website: "https://www.thinkwithportals.com/"}, Fetched: time.Now()}
	m.running = 620
	got := act(m, press("w")).(Model)
	got = act(got, press("b")).(Model)
	if want := []string{"https://www.thinkwithportals.com/", "https://steamdb.info/app/620/"}; strings.Join(*opened, " ") != strings.Join(want, " ") {
		t.Errorf("opened %v, want %v", *opened, want)
	}
	if !strings.Contains(got.status, "SteamDB for Portal 2") {
		t.Errorf("status %q", got.status)
	}

	bar := lineWith(ansi.Strip(m.View().Content), "q quit")
	if !strings.Contains(bar, "w website") || !strings.Contains(bar, "b steamdb") {
		t.Errorf("with a website, the key line has w and b: %q", bar)
	}

	// No website listed, not on the store, store info not here yet: w is
	// off the key line and does nothing; b stays.
	for _, entry := range []*webapi.StoreEntry{
		{Info: webapi.StoreInfo{Name: "Portal 2"}, Fetched: time.Now()},
		{Failed: true, Fetched: time.Now()},
		nil,
	} {
		m, opened := withOpener(t, nil, portal2)
		if entry != nil {
			m.storeInfo[620] = *entry
		}
		bar := lineWith(ansi.Strip(m.View().Content), "q quit")
		if strings.Contains(bar, "website") || !strings.Contains(bar, "b steamdb") {
			t.Errorf("without a website, w leaves the key line, b stays: %q", bar)
		}
		got := act(m, press("w")).(Model)
		if len(*opened) != 0 || got.status != "" {
			t.Errorf("w without a website should do nothing: opened %v, status %q", *opened, got.status)
		}
	}
}

// A web page that fails to open blames the browser, not Steam.
func TestWebsiteOpenFails(t *testing.T) {
	m, _ := withOpener(t, errors.New("no browser"), portal2)
	got := act(m, press("b")).(Model)
	if !strings.Contains(got.status, "Could not open the browser") {
		t.Errorf("status %q", got.status)
	}
	got = act(m, press("s")).(Model)
	if !strings.Contains(got.status, "Could not open Steam") {
		t.Errorf("status %q", got.status)
	}
}
