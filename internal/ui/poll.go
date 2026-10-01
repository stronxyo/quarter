package ui

// Quarter's background checks (Phase C3, Pablo, 2026-09-30: "go 1 to 3
// now"): one timer asks Steam which game is running and looks at the
// download log, and brings both back in one message. Bubble Tea draws the
// screen again after every message, so when a check finds nothing new, the
// screen drawn last is handed back instead of drawn again: idle, Quarter
// was redrawing about 90 times a minute with nothing to show (C1, item 8).
//
// Each round also looks at when each library's steamapps folder last
// changed (Pablo, 2026-10-01: a game uninstalled stayed listed until
// Quarter restarted). Steam adds or removes a game's install file there for
// every install, uninstall or move, and the folder's time changes with it;
// a change reads the library at once.

import (
	"maps"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"

	tea "charm.land/bubbletea/v2"
)

const (
	pollEvery  = 2 * time.Second  // how often the running game and the download log are checked
	viewMaxAge = 30 * time.Second // a kept screen is drawn again after this, for time shown ("5 minutes ago")
)

// pollMsg is one round of the checks.
type pollMsg struct {
	running    uint32               // the game Steam is running, 0 for none
	asked      bool                 // Steam was asked and said
	stopAsking bool                 // Steam doesn't say on this platform: don't ask again
	downloads  *downloadsMsg        // the look at the download log, when it is watched
	folders    map[string]time.Time // each library's steamapps folder, when it last changed
}

// libraryFolders are the steamapps folders to look at: Steam's own and
// every library a game is installed in.
func (m Model) libraryFolders() []string {
	seen := map[string]bool{}
	var out []string
	add := func(lib string) {
		if d := filepath.Join(lib, "steamapps"); lib != "" && !seen[d] {
			seen[d] = true
			out = append(out, d)
		}
	}
	add(m.steamRoot)
	for _, g := range m.installed {
		add(g.LibraryPath)
	}
	return out
}

// pollCmd is the next round of the checks after a delay, or nil when there
// is nothing to check. The checks run off the UI goroutine.
func (m Model) pollCmd(after time.Duration) tea.Cmd {
	ask := m.runningApp
	var look func() downloadsMsg
	var folders []string
	if m.dlWatch {
		look = m.lookDownloads()
		folders = m.libraryFolders()
	}
	if ask == nil && look == nil {
		return nil
	}
	return tea.Tick(after, func(time.Time) tea.Msg {
		var msg pollMsg
		if ask != nil {
			id, known := ask()
			msg.running, msg.asked, msg.stopAsking = id, known, !known
		}
		if look != nil {
			d := look()
			msg.downloads = &d
		}
		if len(folders) > 0 {
			msg.folders = map[string]time.Time{}
			for _, d := range folders {
				if info, err := os.Stat(d); err == nil {
					msg.folders[d] = info.ModTime()
				}
			}
		}
		return msg
	})
}

// startPoll schedules the next round, unless one is on its way.
func (m *Model) startPoll(after time.Duration) tea.Cmd {
	if m.polling {
		return nil
	}
	cmd := m.pollCmd(after)
	m.polling = cmd != nil
	return cmd
}

// applyPoll takes a round of the checks, and notes whether anything on
// screen can have changed.
func (m Model) applyPoll(msg pollMsg) (Model, tea.Cmd) {
	m.polling = false
	quiet := true
	var cmds []tea.Cmd
	if msg.stopAsking {
		m.runningApp = nil
	}
	if msg.asked {
		quiet = msg.running == m.running
		var cmd tea.Cmd
		m, cmd = m.applyRunning(runningMsg{appID: msg.running})
		cmds = append(cmds, cmd)
	}
	if d := msg.downloads; d != nil {
		if d.reread || !maps.Equal(d.paused, m.dlPaused) || m.downloading() {
			quiet = false
		}
		var cmd tea.Cmd
		m, cmd = m.applyDownloads(*d)
		cmds = append(cmds, cmd)
	}
	if msg.folders != nil {
		// The first look only notes the times; a change after reads the
		// library (a game installed, uninstalled or moved). With a read
		// already running, which may have started before the change, the
		// old times are kept, so the next round looks again.
		changed := m.folders != nil && !maps.EqualFunc(m.folders, msg.folders, time.Time.Equal)
		switch {
		case changed && m.scanning:
		case changed:
			m.scanning, quiet = true, false
			cmds = append(cmds, scan(m.steamPath))
			m.folders = msg.folders
		default:
			m.folders = msg.folders
		}
	}
	m.viewQuiet = quiet
	return m, tea.Batch(append(cmds, m.startPoll(pollEvery))...)
}

// viewMemo is the screen drawn last. Models are copies, so it is shared
// through a pointer and tagged with the generation of the model that drew
// it; generations come from one counter, so two copies never share one.
type viewMemo struct {
	gen uint64
	at  time.Time
	v   tea.View
}

var viewGens atomic.Uint64

// settleView, after an update, marks the screen as needing a new drawing,
// unless the update was a check that found nothing new.
func (m *Model) settleView() {
	m.reuseView = m.viewQuiet
	if !m.viewQuiet {
		m.viewGen = viewGens.Add(1)
	}
	m.viewQuiet = false
}

// keptView is the screen drawn last, when it can be shown again.
func (m Model) keptView() (tea.View, bool) {
	if !m.reuseView || m.memo == nil || m.memo.gen != m.viewGen || m.now().Sub(m.memo.at) >= viewMaxAge {
		return tea.View{}, false
	}
	return m.memo.v, true
}

func (m Model) keepView(v tea.View) {
	if m.memo != nil {
		*m.memo = viewMemo{gen: m.viewGen, at: m.now(), v: v}
	}
}
