package ui

// The past years from Steam (Steam Replay, see internal/replay), brought
// into history on request only: the Steam Web API terms allow fetching a
// user's data only as the user asks. Asked for from the history views, never
// at start: a asks, y confirms, esc declines; a again offers to remove them.

import (
	"context"

	"fmt"
	"log/slog"
	"slices"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"

	"github.com/stronxyo/quarter/internal/replay"
)

type replayState int

const (
	replayOff      replayState = iota // not brought in
	replayAsking                      // the pane asks to bring them in
	replayFetching                    // asking Steam
	replayOn                          // in the history
	replayRemoving                    // the pane asks to remove them
)

// replayDoneMsg carries the years Steam gave, and the years it didn't.
type replayDoneMsg struct {
	years   []replay.Year
	missing []int
	err     error
}

// replayKey handles the past years' keys while history is open; ok is
// false for other keys, and for all of them when Quarter can't fetch.
func (m Model) replayKey(msg tea.KeyPressMsg) (Model, tea.Cmd, bool) {
	if m.fetchReplay == nil {
		return m, nil, false
	}
	asking := m.replay == replayAsking || m.replay == replayRemoving
	switch {
	case asking && key.Matches(msg, keyNo):
		if m.replay == replayRemoving {
			m.replay = replayOn
		} else {
			m.replay = replayOff
		}
	case m.replay == replayAsking && key.Matches(msg, keyYes):
		m.replay = replayFetching
		fetch := m.fetchReplay
		return m, func() tea.Msg {
			years, missing, err := fetch(context.Background())
			return replayDoneMsg{years: years, missing: missing, err: err}
		}, true
	case m.replay == replayRemoving && key.Matches(msg, keyYes):
		m.replay, m.replayYears = replayOff, nil
		m.hist = m.hist.WithReplay(nil)
		return m, tea.Batch(m.note("Past years removed from your history"), m.forgetReplay()), true
	case asking || m.replay == replayFetching:
		// The question waits for y or esc; the fetch, for Steam.
	case key.Matches(msg, keys.PastYears) && m.replay == replayOff:
		m.replay = replayAsking
	case key.Matches(msg, keys.PastYears) && m.replay == replayOn:
		m.replay = replayRemoving
	default:
		return m, nil, false
	}
	return m, nil, true
}

// applyReplay takes the years Steam gave, and keeps them.
func (m Model) applyReplay(msg replayDoneMsg) (Model, tea.Cmd) {
	switch {
	case msg.err != nil:
		slog.Warn("past years", "err", msg.err)
		m.replay = replayOff
		return m, m.warn("Could not reach Steam for your past years. Press a to try again.")
	case len(msg.years) == 0:
		m.replay = replayOff
		return m, m.warn("Steam gave no past years: set each year to Public in Steam Replay, then press a again.")
	}
	m.replay, m.replayYears = replayOn, msg.years
	m.hist = m.hist.WithReplay(msg.years)
	text := "Past years added: " + yearList(msg.years)
	if len(msg.missing) > 0 {
		text += " (not public: " + strings.Join(intStrings(msg.missing), ", ") + ")"
	}
	return m, tea.Batch(m.note(text), m.keepReplay(msg.years))
}

// keepReplay saves the past years off the UI goroutine; failing only means
// asking again next run, so it is logged.
func (m Model) keepReplay(years []replay.Year) tea.Cmd {
	if m.saveReplay == nil {
		return nil
	}
	save := m.saveReplay
	return func() tea.Msg {
		if err := save(years); err != nil {
			slog.Warn("past years", "err", err)
		}
		return nil
	}
}

// forgetReplay deletes the saved past years off the UI goroutine.
func (m Model) forgetReplay() tea.Cmd {
	if m.deleteReplay == nil {
		return nil
	}
	del := m.deleteReplay
	return func() tea.Msg {
		if err := del(); err != nil {
			slog.Warn("past years", "err", err)
		}
		return nil
	}
}

func yearList(years []replay.Year) string {
	out := make([]string, len(years))
	for i, y := range years {
		out[i] = fmt.Sprint(y.Year)
	}
	return strings.Join(out, ", ")
}

func intStrings(ns []int) []string {
	out := make([]string, len(ns))
	for i, n := range ns {
		out[i] = fmt.Sprint(n)
	}
	return out
}

// warn shows a message in the alert colour.
func (m *Model) warn(text string) tea.Cmd {
	cmd := m.note(text)
	m.statusWrong = true
	return cmd
}

// replayPane is the question in the pane while the past years are asked
// about, or fetched.
func (m Model) replayPane(width, height int) string {
	inner := width - paneStyle.GetHorizontalFrameSize()
	choice := func(yes, no string) string {
		return activeTabStyle.Render("y") + dimStyle.Render(" "+yes+"   ") + activeTabStyle.Render("esc") + dimStyle.Render(" "+no)
	}
	var lines []string
	switch m.replay {
	case replayFetching:
		lines = []string{paneTitleStyle.Render("PAST YEARS"), "", dimStyle.Render("Asking Steam…")}
	case replayRemoving:
		lines = []string{
			paneTitleStyle.Render("REMOVE YOUR PAST YEARS?"), "",
			textStyle.Render("Your history goes back to recorded sessions only,"),
			textStyle.Render("and Quarter deletes the past years it saved."), "",
			choice("remove", "keep"),
		}
	default:
		about := textStyle.Width(inner).Render(fmt.Sprintf(
			"Quarter asks Steam for your yearly recap (Steam Replay, %d to %d): the time you played each game, month by month.",
			replay.FirstYear, m.now().Year()-1))
		lines = []string{paneTitleStyle.Render("BRING IN YOUR PAST YEARS?"), ""}
		lines = append(lines, strings.Split(about, "\n")...)
		lines = append(lines, "",
			dimStyle.Render("· Nothing else, and no login: only your public SteamID."),
			dimStyle.Render("· Saved only on this PC (replay.json)."),
			dimStyle.Render("· Each year must be set to Public in Steam Replay."),
			dimStyle.Render("· Press a later to remove them."), "",
			choice("bring them in", "not now"),
		)
	}
	return paneBox(lines, width, height, inner)
}

// replayHint is the game history's last line until the past years are in.
func (m Model) replayHint() string {
	if m.replay != replayOff || m.fetchReplay == nil {
		return ""
	}
	return dimStyle.Render("Older years? ") + activeTabStyle.Render("a") + dimStyle.Render(" add your past years from Steam")
}

// yearsLine is the board's line of time played in each year.
func (m Model) yearsLine() string {
	years := m.hist.Years()
	if len(years) < 2 {
		return ""
	}
	keys := make([]int, 0, len(years))
	for y := range years {
		keys = append(keys, y)
	}
	slices.Sort(keys)
	parts := make([]string, len(keys))
	for i, y := range keys {
		parts[i] = dimStyle.Render(fmt.Sprint(y)+" ") + activeTabStyle.Render(fmt.Sprintf("%.0fh", years[y].Hours()))
	}
	return dimStyle.Render(fmt.Sprintf("%-9s", "YEARS")) + strings.Join(parts, dimStyle.Render(" · "))
}

// monthDay is a day's share of its month's past-years time (months keyed
// by their first day, UTC): the month spread evenly over its days, for the
// calendar before the recorded sessions.
func monthDay(months map[time.Time]time.Duration, d time.Time) time.Duration {
	start := time.Date(d.Year(), d.Month(), 1, 0, 0, 0, 0, time.UTC)
	played := months[start]
	if played <= 0 {
		return 0
	}
	return played / time.Duration(start.AddDate(0, 1, -1).Day())
}
