package ui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/steam"
)

func TestEmptyListMessages(t *testing.T) {
	tests := []struct {
		name  string
		games []string
		keys  []string
		want  string
	}{
		{"nothing installed", nil, nil, "No games installed yet. Install one in Steam, then press r."},
		{"no favourites", []string{"Alpha"}, []string{"tab", "tab", "tab"}, "No favourites yet. Press f on a game to star it."},
		{"nothing recent", []string{"Alpha"}, []string{"tab", "tab"}, "Nothing played in the last 14 days."},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m tea.Model = ready(t, tt.games...)
			for _, k := range tt.keys {
				if k == "tab" {
					m, _ = m.Update(tea.KeyPressMsg{Code: tea.KeyTab})
				} else {
					m, _ = m.Update(press(k))
				}
			}
			view := m.(Model).View().Content
			if !strings.Contains(view, tt.want) {
				t.Errorf("missing %q:\n%s", tt.want, view)
			}
			if strings.Contains(view, "No items") {
				t.Errorf("bubbles' own message still shows:\n%s", view)
			}
			if got := strings.Count(view, "\n") + 1; got != 24 {
				t.Errorf("the empty list should keep the screen's height: %d rows, want 24", got)
			}
		})
	}
}

func TestFooterMessages(t *testing.T) {
	manifest := &steam.FileError{Name: "appmanifest_620.acf", Err: errors.New("no AppState block")}
	tests := []struct {
		name string
		res  library.Result
		want string
	}{
		{
			name: "one library not connected",
			res:  library.Result{Missing: []steam.LibraryFolder{{Path: `D:\SteamLibrary`, Games: 5}}},
			want: `Library D:\SteamLibrary not connected, 5 games hidden.`,
		},
		{
			name: "two libraries not connected",
			res:  library.Result{Missing: []steam.LibraryFolder{{Path: `D:\A`, Games: 1}, {Path: `E:\B`}}},
			want: "2 libraries not connected, 1 game hidden.",
		},
		{
			name: "a damaged install file",
			res:  library.Result{Warnings: []error{manifest}},
			want: "Steam's file for app 620 (appmanifest_620.acf) can't be read, so that game is hidden.",
		},
		{
			name: "the playtime file",
			res:  library.Result{Warnings: []error{&steam.FileError{Name: "localconfig.vdf", Err: errors.New("x")}}},
			want: "Steam's playtime file (localconfig.vdf) can't be read, so playtime is hidden.",
		},
		{
			name: "several files",
			res:  library.Result{Warnings: []error{manifest, manifest}},
			want: "2 Steam files can't be read, see quarter.log.",
		},
		{
			name: "never signed in",
			res:  library.Result{NoAccount: true},
			want: "Sign in to Steam once to see playtime.",
		},
		{
			name: "a missing drive comes before damaged files",
			res:  library.Result{Missing: []steam.LibraryFolder{{Path: `D:\A`, Games: 2}}, Warnings: []error{manifest}, NoAccount: true},
			want: `Library D:\A not connected, 2 games hidden.`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// 97 columns: no room for the pane beside the list, so the
			// message is the one line under the key help.
			var m tea.Model = New(Options{})
			m, _ = m.Update(tea.WindowSizeMsg{Width: 97, Height: 30})
			tt.res.Games = []library.Game{{AppID: 1, Name: "Alpha", Installed: true}}
			m, _ = m.Update(scanDoneMsg{res: tt.res})
			if view := m.(Model).View().Content; !strings.Contains(view, tt.want) {
				t.Errorf("footer missing %q:\n%s", tt.want, view)
			}
		})
	}
}

// Beside the list, the message goes under the pane instead, wrapped to the
// pane's width, and the screen keeps the window's height.
func TestNoticeUnderPane(t *testing.T) {
	var m tea.Model = New(Options{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 140, Height: 30})
	m, _ = m.Update(scanDoneMsg{res: library.Result{
		Games:    []library.Game{{AppID: 1, Name: "Alpha", Installed: true}},
		Warnings: []error{&steam.FileError{Name: "appmanifest_620.acf", Err: errors.New("x")}},
	}})
	mm := m.(Model)
	lines := strings.Split(ansi.Strip(mm.View().Content), "\n")
	if len(lines) != 30 {
		t.Errorf("the screen is %d rows, want the window's 30", len(lines))
	}
	help := -1
	for i, l := range lines {
		if strings.Contains(l, "q quit") {
			help = i
		}
	}
	var under []string
	bottom := -1
	for i, l := range lines {
		if strings.Contains(l, "╰") && ansi.StringWidth(l) > mm.paneX() {
			bottom = i
		}
	}
	if bottom < 0 {
		t.Fatal("no pane found")
	}
	for _, l := range lines[bottom+1 : help] {
		if ansi.StringWidth(l) > mm.paneX() {
			under = append(under, strings.TrimSpace(ansi.Cut(l, mm.paneX(), ansi.StringWidth(l))))
		}
	}
	if got := strings.Join(under, " "); got != "Steam's file for app 620 (appmanifest_620.acf) can't be read, so that game is hidden." {
		t.Errorf("under the pane: got %q", got)
	}
	if strings.Contains(lines[help], "can't be read") || help != len(lines)-1 {
		t.Error("the key help should be the last line, with no message under it")
	}
	for _, l := range lines {
		if w := ansi.StringWidth(l); w > 140 {
			t.Errorf("line wider than the window (%d): %q", w, l)
		}
	}
}

// Something wrong shows in light red; guidance stays dim.
func TestNoticeColours(t *testing.T) {
	red := "38;2;255;123;114m" // #FF7B72
	tests := []struct {
		res  library.Result
		text string
		red  bool
	}{
		{library.Result{Missing: []steam.LibraryFolder{{Path: `D:\A`, Games: 2}}}, `Library D:\A not connected`, true},
		{library.Result{NoAccount: true}, "Sign in to Steam once", false},
	}
	for _, tt := range tests {
		var m tea.Model = New(Options{})
		m, _ = m.Update(tea.WindowSizeMsg{Width: 97, Height: 30})
		tt.res.Games = []library.Game{{AppID: 1, Name: "Alpha", Installed: true}}
		m, _ = m.Update(scanDoneMsg{res: tt.res})
		for _, l := range strings.Split(m.(Model).View().Content, "\n") {
			if strings.Contains(ansi.Strip(l), tt.text) && strings.Contains(l, red) != tt.red {
				t.Errorf("%q: red %v, want %v: %q", tt.text, !tt.red, tt.red, l)
			}
		}
	}
}

func TestEmptySearchMessage(t *testing.T) {
	m := ready(t, "Alpha")
	m.list.SetFilterText("zzz")
	if view := m.View().Content; !strings.Contains(view, "No games match.") || strings.Contains(view, "No items") {
		t.Errorf("a search with no match should say so:\n%s", view)
	}
}

// A footer longer than the window is cut, so it never wraps and pushes the
// screen past the window.
func TestFooterFitsWindow(t *testing.T) {
	var m tea.Model = New(Options{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 60, Height: 24})
	m, _ = m.Update(scanDoneMsg{res: library.Result{
		Games:   []library.Game{{AppID: 1, Name: "Alpha", Installed: true}},
		Missing: []steam.LibraryFolder{{Path: `D:\` + strings.Repeat("Very long folder name ", 5), Games: 3}},
	}})
	for _, line := range strings.Split(m.(Model).View().Content, "\n") {
		if w := ansi.StringWidth(line); w > 60 {
			t.Errorf("line wider than the window (%d): %q", w, line)
		}
	}
}
