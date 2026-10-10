package ui

import (
	"strings"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/steam"
	"github.com/stronxyo/quarter/internal/webapi"
)

func sized(t *testing.T, width, height int) tea.Model {
	t.Helper()
	var m tea.Model = New(Options{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: width, Height: height})
	m, _ = m.Update(scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: []library.Game{
		{AppID: 1, Name: "Alpha", Installed: true, Playtime: 3 * time.Hour, SizeOnDisk: 13_314_398_618, LibraryPath: `D:\SteamLibrary`},
		{AppID: 2, Name: "Bravo Demo", Installed: true},
	}}})
	return m
}

// A description too long for the pane is cut inside the frame: the bottom
// border stays.
func TestLongDescriptionKeepsBorder(t *testing.T) {
	m := sized(t, 150, 30).(Model)
	m.storeInfo[1] = webapi.StoreEntry{Info: webapi.StoreInfo{Description: strings.Repeat("word ", 400)}, Fetched: time.Now()}
	lines := strings.Split(ansi.Strip(m.detailView(50, 15)), "\n")
	if len(lines) != 15 || !strings.HasPrefix(lines[len(lines)-1], "╰") {
		t.Errorf("want 15 lines ending in the bottom border, got %d ending %q", len(lines), lines[len(lines)-1])
	}
}

// fits fails the test if any line is wider, or the view taller, than the window.
func fits(t *testing.T, m tea.Model, width, height int) string {
	t.Helper()
	view := ansi.Strip(m.(Model).View().Content)
	lines := strings.Split(view, "\n")
	if len(lines) > height {
		t.Errorf("view is %d lines, window is %d", len(lines), height)
	}
	for _, l := range lines {
		if w := ansi.StringWidth(l); w > width {
			t.Errorf("line is %d wide, window is %d: %q", w, width, l)
		}
	}
	return view
}

func TestPaneBesideListOnWideWindow(t *testing.T) {
	m := sized(t, 140, 30)
	view := fits(t, m, 140, 30)
	for _, want := range []string{"ALPHA", "Played       3.0h", "Size         12.4 GB", `Library      D:\SteamLibrary`} {
		if !strings.Contains(view, want) {
			t.Errorf("pane missing %q", want)
		}
	}

	m, _ = m.Update(press("j"))
	view = fits(t, m, 140, 30)
	if !strings.Contains(view, "BRAVO (DEMO)") {
		t.Error("pane should follow the selection to Bravo, with its demo tag")
	}
}

func TestPanePopUpOnNarrowWindow(t *testing.T) {
	m := sized(t, 80, 30)
	if view := fits(t, m, 80, 30); strings.Contains(view, "ALPHA") {
		t.Fatal("narrow windows should not show the pane until d is pressed")
	}

	m, _ = m.Update(press("d"))
	view := fits(t, m, 80, 30)
	if !strings.Contains(view, "ALPHA") || strings.Contains(view, "Bravo") {
		t.Errorf("d should open the pane over the list:\n%s", view)
	}

	m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if view := fits(t, m, 80, 30); strings.Contains(view, "ALPHA") || !strings.Contains(view, "Bravo") {
		t.Error("esc should close the pane and show the list again")
	}
}

func TestPaneGrowsWithWindow(t *testing.T) {
	tests := []struct {
		width, wantTable, wantPane int
	}{
		{98, 56, 40},   // just wide enough for both
		{122, 80, 40},  // the table reaches its full width first
		{140, 80, 58},  // then the pane grows
		{162, 80, 80},  // out to the right edge
		{240, 80, 158}, // at any width: no empty centre
		{80, 80, 80},   // narrow: the pop-up
	}
	for _, tt := range tests {
		m := sized(t, tt.width, 30).(Model)
		if m.tableWidth() != tt.wantTable || m.paneWidth() != tt.wantPane {
			t.Errorf("width %d: table %d pane %d, want %d and %d", tt.width, m.tableWidth(), m.paneWidth(), tt.wantTable, tt.wantPane)
		}
		if m.sideBySide() && m.paneX()+m.paneWidth() != tt.width {
			t.Errorf("width %d: the pane should end at the right edge", tt.width)
		}
		fits(t, m, tt.width, 30)
	}
}

func TestDetailsKeyIgnoredOnWideWindow(t *testing.T) {
	m := sized(t, 140, 30)
	m, _ = m.Update(press("d"))
	if m.(Model).detailOpen {
		t.Error("d should do nothing when the pane is already beside the list")
	}
}

func TestAchievementBar(t *testing.T) {
	tests := []struct {
		achieved, total int
		want            string
	}{
		{28, 49, "28/49   ━━━━━━━━──────"},
		{0, 10, "0/10    ──────────────"},
		{1, 200, "1/200   ━─────────────"},
		{199, 200, "199/200 ━━━━━━━━━━━━━─"},
		{51, 51, "51/51   ━━━━━━━━━━━━━━"},
	}
	for _, tt := range tests {
		if got := ansi.Strip(achievementBar(tt.achieved, tt.total)); got != tt.want {
			t.Errorf("%d/%d: got %q, want %q", tt.achieved, tt.total, got, tt.want)
		}
	}
}

func TestPaneShowsAchievements(t *testing.T) {
	var m tea.Model = New(Options{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 150, Height: 30})
	m, _ = m.Update(scanDoneMsg{res: library.Result{PlaytimeKnown: true, Games: []library.Game{
		{AppID: 1, Name: "Alpha", Installed: true, Achievements: steam.Achievements{Achieved: 28, Total: 49}},
		{AppID: 2, Name: "Bravo", Installed: true},
	}}})
	if view := ansi.Strip(m.(Model).View().Content); !strings.Contains(view, "Achievements 28/49   ━━━━━━━━──────") {
		t.Errorf("pane should show Alpha's achievements:\n%s", view)
	}
	m, _ = m.Update(press("j"))
	if strings.Contains(ansi.Strip(m.(Model).View().Content), "Achievements") {
		t.Error("Bravo has no achievements; the row should not show")
	}
}

// Publisher and website sit in a block of their own under the table, after
// a blank line; the website shows as its name and links to the full
// address. Without either, there is no block.
func TestPublisherAndWebsite(t *testing.T) {
	m := sized(t, 150, 30).(Model)
	m.storeInfo[1] = webapi.StoreEntry{Info: webapi.StoreInfo{
		Developers: []string{"Psyonix LLC"}, Publishers: []string{"Psyonix LLC"}, Website: "http://www.rocketleague.com/",
	}, Fetched: time.Now()}
	raw := m.detailView(60, 25)
	lines := strings.Split(ansi.Strip(raw), "\n")
	at := -1
	for i, l := range lines {
		if strings.Contains(l, "Publisher") {
			at = i
		}
	}
	if at < 2 || !strings.Contains(lines[at-2], "Library") {
		t.Fatalf("Publisher should come after Library and a blank line:\n%s", strings.Join(lines, "\n"))
	}
	if strings.Trim(lines[at-1], "│ ") != "" || !strings.Contains(lines[at], "Psyonix LLC") || !strings.Contains(lines[at+1], "Website") || !strings.Contains(lines[at+1], "rocketleague.com") {
		t.Errorf("want a blank line, Publisher, then Website:\n%s", strings.Join(lines, "\n"))
	}
	if strings.Contains(lines[at+1], "http") || strings.Contains(lines[at+1], "www.") {
		t.Errorf("the website shows as its name: %q", lines[at+1])
	}
	if !strings.Contains(lines[at+1], "rocketleague.com "+markerLink) {
		t.Errorf("the website's name ends in the arrow: %q", lines[at+1])
	}
	if strings.Contains(raw, "\x1b]8;") {
		t.Error("no terminal link codes: Windows Terminal underlines them with dots")
	}

	// A name too long for the pane is cut, and the arrow stays.
	m.storeInfo[1] = webapi.StoreEntry{Info: webapi.StoreInfo{Website: "https://www." + strings.Repeat("long", 20) + ".com/"}, Fetched: time.Now()}
	for _, l := range strings.Split(ansi.Strip(m.detailView(40, 25)), "\n") {
		if strings.Contains(l, "Website") && !strings.Contains(l, "… "+markerLink) {
			t.Errorf("a cut name keeps its arrow: %q", l)
		}
	}

	m.storeInfo[1] = webapi.StoreEntry{Info: webapi.StoreInfo{Developers: []string{"Psyonix LLC"}}, Fetched: time.Now()}
	if v := ansi.Strip(m.detailView(60, 25)); strings.Contains(v, "Publisher") || strings.Contains(v, "Website") {
		t.Errorf("without a publisher or website, no block:\n%s", v)
	}
}

func TestSiteName(t *testing.T) {
	for in, want := range map[string]string{
		"http://www.rocketleague.com":        "rocketleague.com",
		"https://www.teamfortress.com/":      "teamfortress.com",
		"https://example.com/games/thing/":   "example.com/games/thing",
		"https://store.example.com/game?x=1": "store.example.com/game",
	} {
		if got := siteName(in); got != want {
			t.Errorf("siteName(%q) = %q, want %q", in, got, want)
		}
	}
}
