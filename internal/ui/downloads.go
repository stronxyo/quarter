package ui

// Downloads: games Steam is installing or updating show their progress,
// estimated from Steam's download log (see steam.ReadDownloads). Every
// downloadCheck Quarter looks whether the log changed and reads it only
// then; while a game downloads it also reads that game's install file (a
// pause shows only there), and the arrow blinks while Steam works on it.

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/steam"
)

const (
	downloadCheck = 2 * time.Second        // how often the download log is looked at
	blinkStep     = 500 * time.Millisecond // the arrow's blink, and the estimate's steps
	dlBarWidth    = 40                     // the pane's progress bar, at most
)

type (
	// downloadsMsg is one look at the downloads. The log is re-read only
	// when its size changed (reread); paused comes from the install files of
	// the games downloading.
	downloadsMsg struct {
		reread bool
		d      map[uint32]steam.Download
		size   int64
		paused map[uint32]bool
	}
	downloadCheckMsg struct{}
	dlBlinkMsg       struct{}
)

// downloading reports whether any game in the library is being downloaded.
func (m Model) downloading() bool {
	return slices.ContainsFunc(m.installed, func(g library.Game) bool { return g.Updating })
}

// watchDownloads starts looking at the download log, once the Steam folder
// is known, and keeps looking for as long as Quarter runs.
func (m *Model) watchDownloads() tea.Cmd {
	if m.dlWatch || m.steamRoot == "" {
		return nil
	}
	m.dlWatch = true
	return m.checkDownloads()
}

// checkDownloads looks at the log's size, reads it when it changed, and
// reads the install files of the games downloading, off the UI goroutine.
func (m Model) checkDownloads() tea.Cmd {
	root, seen := m.steamRoot, m.logSize
	type game struct {
		id  uint32
		lib string
	}
	var games []game
	for _, g := range m.installed {
		if g.Updating {
			games = append(games, game{g.AppID, g.LibraryPath})
		}
	}
	return func() tea.Msg {
		msg := downloadsMsg{size: seen, paused: map[uint32]bool{}}
		if info, err := os.Stat(filepath.Join(root, "logs", "content_log.txt")); err == nil && info.Size() != seen {
			if d, err := steam.ReadDownloads(root, time.Local); err == nil {
				msg.reread, msg.d, msg.size = true, d, info.Size()
			}
		}
		for _, g := range games {
			if mf, err := steam.ReadManifest(g.lib, g.id); err == nil && mf.Paused() {
				msg.paused[g.id] = true
			}
		}
		return msg
	}
}

// applyDownloads takes a look at the downloads. A download that ended, or
// one the library doesn't know as downloading (started while Quarter is
// open, or a new install), has its install file just written: the library
// is read at once.
func (m Model) applyDownloads(msg downloadsMsg) (Model, tea.Cmd) {
	cmds := []tea.Cmd{tea.Tick(downloadCheck, func(time.Time) tea.Msg { return downloadCheckMsg{} })}
	m.dlPaused = msg.paused
	if msg.reread {
		changed := false
		for id := range m.downloads {
			if _, ok := msg.d[id]; !ok {
				changed = true // ended
			}
		}
		// Only a download seen for the first time counts as started: one for
		// an app the library never lists (a redistributable, a DLC) would
		// otherwise read the library at every change of the log.
		for id, d := range msg.d {
			prev, seen := m.downloads[id]
			fresh := !seen || prev.Phase == steam.PhaseNone
			if g, ok := m.gameByID(id); fresh && d.Phase != steam.PhaseNone && (!ok || !g.Updating) {
				changed = true // started
			}
		}
		m.downloads, m.logSize = msg.d, msg.size
		if changed && !m.scanning {
			m.scanning = true
			cmds = append(cmds, scan(m.steamPath))
		}
	}
	m.list.SetDelegate(m.delegate())
	return m, tea.Batch(append(cmds, m.blinkDownloads())...)
}

// blinkDownloads starts the arrow's blink while a game downloads, once.
func (m *Model) blinkDownloads() tea.Cmd {
	if m.dlBlink || !m.downloading() {
		return nil
	}
	m.dlBlink = true
	return tea.Tick(blinkStep, func(time.Time) tea.Msg { return dlBlinkMsg{} })
}

// dlBlinkStep moves the blink on, and the estimates with it.
func (m Model) dlBlinkStep() (Model, tea.Cmd) {
	if !m.downloading() {
		m.dlBlink = false
		return m, nil
	}
	m.list.SetDelegate(m.delegate())
	return m, tea.Tick(blinkStep, func(time.Time) tea.Msg { return dlBlinkMsg{} })
}

// dlState is what the screen says about a game's download.
type dlState struct {
	phase string // "downloading", "applying", "verifying", "paused", "queued", or "" before the log says
	pct   int    // -1 when unknown
	d     steam.Download
}

// working reports whether Steam is working on the download now (the arrow
// blinks), rather than holding it.
func (s dlState) working() bool {
	return s.phase == "downloading" || s.phase == "applying" || s.phase == "verifying"
}

// dlView is what the download text needs, apart from the Model: the list's
// delegate keeps it, and a Model kept there would keep every older one.
type dlView struct {
	downloads map[uint32]steam.Download
	paused    map[uint32]bool
	now       func() time.Time
}

func (m Model) dlView() dlView {
	return dlView{m.downloads, m.dlPaused, m.now}
}

func (v dlView) stateOf(g library.Game) dlState {
	d, ok := v.downloads[g.AppID]
	s := dlState{pct: -1, d: d}
	switch {
	case g.Paused || v.paused[g.AppID]:
		s.phase = "paused"
	case !ok || d.Phase == steam.PhaseNone:
		if v.someoneDownloading() {
			s.phase = "queued"
		}
	case d.Phase == steam.PhaseApplying:
		s.phase = "applying"
	case d.Phase == steam.PhaseVerifying:
		s.phase = "verifying"
	default:
		s.phase = "downloading"
	}
	if ok && d.Total > 0 && (s.phase == "downloading" || s.phase == "paused") {
		s.pct = int(100 * d.Estimate(v.now()) / d.Total)
	}
	return s
}

// someoneDownloading reports whether the log shows a download under way.
func (v dlView) someoneDownloading() bool {
	for _, d := range v.downloads {
		if d.Phase != steam.PhaseNone {
			return true
		}
	}
	return false
}

// dlArrow is the download's marker: blinking while Steam works on it.
func (v dlView) arrow(s dlState) string {
	if s.working() && (v.now().UnixMilli()/blinkStep.Milliseconds())%2 == 1 {
		return " "
	}
	return markerUpdating
}

// dlCell is a downloading game's text in the list's last column.
func (v dlView) cell(g library.Game) string {
	s := v.stateOf(g)
	arrow := v.arrow(s)
	var text string
	switch {
	case s.phase == "downloading" && s.pct >= 0:
		text = fmt.Sprintf("%s ~%d%%", arrow, s.pct)
	case s.phase == "" || s.phase == "downloading":
		text = arrow + " " + dlWord(g)
	default:
		text = arrow + " " + s.phase
	}
	return ansi.Truncate(text, agoWidth, "")
}

// dlWord is what Steam is doing to the game: installing it or updating it.
func dlWord(g library.Game) string {
	if g.Installing {
		return "installing"
	}
	return "updating"
}

// dlLines are the pane's lines about a game's download.
func (v dlView) lines(g library.Game, inner int) []string {
	s := v.stateOf(g)
	title := v.arrow(s) + " " + strings.ToUpper(dlWord(g))
	if s.pct >= 0 {
		title += fmt.Sprintf(" ~%d%%", s.pct)
	}
	if s.phase != "" && s.phase != "downloading" {
		title += " · " + strings.ToUpper(s.phase)
	}
	lines := []string{updatingLabelStyle.Render(title)}
	if s.pct >= 0 {
		lines = append(lines, dlBar(s.pct, min(inner, dlBarWidth)))
	}
	var info string
	switch s.phase {
	case "downloading":
		info = "Steam is getting it ready."
		if s.d.Total > 0 {
			done := s.d.Estimate(v.now())
			info = fmt.Sprintf("%s of %s", sizeText(done), sizeText(s.d.Total))
			if s.d.Rate > 0 {
				left := time.Duration(float64(s.d.Total-done) / s.d.Rate * float64(time.Second))
				info += fmt.Sprintf(" · %.0f MB/s · about %s left", s.d.Rate/1e6, timeLeft(left))
			}
		}
	case "applying":
		info = "Downloaded. Steam is putting the files in place."
	case "verifying":
		info = "Steam is checking the files."
	case "paused":
		info = "Paused in Steam."
	case "queued":
		info = "Waiting in Steam's download queue."
	default:
		info = "Steam is getting it ready."
	}
	return append(lines, dimStyle.Render(info))
}

// dlBar is the progress bar: done in blue, the rest the thin dim line.
func dlBar(pct, width int) string {
	filled := min(width*pct/100, width)
	return updatingFgStyle.Render(strings.Repeat(barFilled, filled)) + dimStyle.Render(strings.Repeat(barEmpty, width-filled))
}

// sizeText is a size in GB with one decimal, or MB under a GB.
func sizeText(n int64) string {
	if n < 1e9 {
		return fmt.Sprintf("%.0f MB", float64(n)/1e6)
	}
	return fmt.Sprintf("%.1f GB", float64(n)/1e9)
}

// timeLeft is a time left, rounded the way people say it.
func timeLeft(d time.Duration) string {
	switch {
	case d < time.Minute:
		return "a minute"
	case d < time.Hour:
		return fmt.Sprintf("%d min", int(d.Minutes()+0.5))
	}
	return fmt.Sprintf("%.1f h", d.Hours())
}
