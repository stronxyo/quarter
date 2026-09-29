package ui

import (
	"errors"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/steam"
)

func TestSteamNotFoundScreen(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want []string
	}{
		{
			name: "searched",
			err:  &steam.NotFoundError{Looked: []string{`C:\Program Files (x86)\Steam`, `D:\Steam`}},
			want: []string{"Steam not found", "Quarter looked in", `C:\Program Files (x86)\Steam`, `D:\Steam`, "--steam-path", "QUARTER_STEAM_PATH", "r look again", "q quit"},
		},
		{
			name: "given folder isn't Steam",
			err:  &steam.NotFoundError{Given: `E:\Games`},
			want: []string{"Steam not found", `E:\Games is not a Steam folder`, "no steamapps folder", "--steam-path", "r look again"},
		},
		{
			name: "other failure",
			err:  errors.New("disk on fire"),
			want: []string{"Could not read Steam", "disk on fire", "q quit"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var m tea.Model = New(Options{})
			m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
			m, _ = m.Update(scanDoneMsg{err: tt.err})
			view := m.(Model).View().Content
			for _, w := range tt.want {
				if !strings.Contains(view, w) {
					t.Errorf("screen missing %q:\n%s", w, view)
				}
			}
		})
	}
}

// Steam installed while Quarter shows the screen: r looks again and the
// library opens.
func TestSteamNotFoundLookAgain(t *testing.T) {
	var m tea.Model = New(Options{})
	m, _ = m.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	m, _ = m.Update(scanDoneMsg{err: &steam.NotFoundError{Looked: []string{`C:\Steam`}}})
	m, cmd := m.Update(press("r"))
	if cmd == nil || !m.(Model).scanning {
		t.Fatal("r should start a new search")
	}
	if view := m.(Model).View().Content; !strings.Contains(view, "looking again") {
		t.Errorf("no sign of the search:\n%s", view)
	}
	m, _ = m.Update(scanDoneMsg{res: library.Result{Games: []library.Game{{AppID: 1, Name: "Alpha", Installed: true}}}})
	if m.(Model).state != stateReady || !strings.Contains(m.(Model).View().Content, "Alpha") {
		t.Error("the library should open once Steam is found")
	}
}
