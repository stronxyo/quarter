package ui

import (
	"fmt"
	"math"
	"strings"
	"time"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/history"
	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/steam"
)

// Play history (see internal/history): h cycles the games, the ranking of
// the games played most (the list's place, the ranked game's history in
// the pane), the HIGH SCORES board (in the pane, the ranking staying), and
// back to the games.

// histView is which history view is open.
type histView int

const (
	histOff     histView = iota // the games
	histRanking                 // MOST PLAYED, the game's history in the pane
	histBoard                   // MOST PLAYED, HIGH SCORES in the pane
)

// sessionsMsg carries every recorded session, after new ones were saved.
type sessionsMsg struct{ all []steam.Session }

// histPeriods are the periods to rank by, in days ending today.
var histPeriods = []struct {
	label string
	days  int
}{{"7 days", 7}, {"1 month", 30}, {"3 months", 91}, {"6 months", 182}, {"1 year", 365}}

const (
	histWeeks     = 16 // bars in the game's history, a week each
	histRecent    = 6  // the game's latest sessions listed
	boardPlaces   = 8  // rows on the HIGH SCORES board
	calendarWeeks = 53 // a year of weeks in the board's calendar
	rankBarWidth  = 20 // the ranking's bars
	boardBarWidth = 12 // the board's bars
)

// sparkSteps are the bar heights for weekly and weekday bars.
const sparkSteps = "▁▂▃▄▅▆▇█"

// barParts are a horizontal bar's last cell, in eighths.
const barParts = " ▏▎▍▌▋▊▉"

// histKey handles a key while history is open, and h from the games. ok is
// false for a key history leaves to the rest of the app.
func (m Model) histKey(msg tea.KeyPressMsg) (_ Model, _ tea.Cmd, ok bool) {
	if key.Matches(msg, keys.History) {
		m.histView = (m.histView + 1) % 3
		if m.histView == histRanking {
			m.histSel = 0
		}
		return m, nil, true
	}
	if m.histView == histOff {
		return m, nil, false
	}
	n := len(m.ranking())
	switch s := msg.String(); s {
	case "esc":
		m.histView = histOff
	case "left":
		m.histPeriod = (m.histPeriod + len(histPeriods) - 1) % len(histPeriods)
		m.histSel = 0
	case "right":
		m.histPeriod = (m.histPeriod + 1) % len(histPeriods)
		m.histSel = 0
	// As in the game list: up and down run on across pages and wrap round
	// past the ends; g and G jump to the first and the last.
	case "up", "k", "down", "j", "g", "home", "G", "end":
		if n == 0 {
			break
		}
		switch s {
		case "up", "k":
			m.histSel = (m.histSel - 1 + n) % n
		case "down", "j":
			m.histSel = (m.histSel + 1) % n
		case "g", "home":
			m.histSel = 0
		default:
			m.histSel = n - 1
		}
	default:
		// Refresh, help and quitting go on as usual; every other key waits
		// for the games view, so it never acts on a game not on show.
		passes := key.Matches(msg, keys.Refresh, keys.Help, helpQuit) || s == "ctrl+c"
		return m, nil, !passes
	}
	return m, nil, true
}

// ranking is the period's games by time played, Steam tools left out.
func (m Model) ranking() []history.Entry {
	var out []history.Entry
	for _, e := range m.hist.Top(m.periodStart()) {
		if steam.IsGame(e.App) {
			out = append(out, e)
		}
	}
	return out
}

func (m Model) periodStart() time.Time {
	return m.hist.DaysBack(histPeriods[m.histPeriod].days, m.now())
}

// rankedGame is the game selected in the ranking: from the library, or
// just its appid and store name when it isn't in the list.
func (m Model) rankedGame() (library.Game, bool) {
	r := m.ranking()
	if len(r) == 0 {
		return library.Game{}, false
	}
	id := r[min(m.histSel, len(r)-1)].App
	if g, ok := m.gameByID(id); ok {
		return g, true
	}
	return library.Game{AppID: id, Name: m.storeInfo[id].Info.Name}, true
}

// gameName is how a game is named in the ranking and on the board.
func (m Model) gameName(id uint32) string {
	if g, ok := m.gameByID(id); ok && g.Name != "" {
		return g.Label()
	}
	if name := m.storeInfo[id].Info.Name; name != "" {
		return name
	}
	return fmt.Sprintf("App %d", id)
}

// playedText is time played: minutes under an hour, "-" for none.
func playedText(d time.Duration) string {
	switch {
	case d <= 0:
		return "-"
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	return fmt.Sprintf("%.1fh", d.Hours())
}

// lengthText is a session's length: "45m", "2h", "1h 20m".
func lengthText(d time.Duration) string {
	h, mins := int(d.Hours()), int(d.Minutes())%60
	switch {
	case h == 0:
		return fmt.Sprintf("%dm", mins)
	case mins == 0:
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dh %dm", h, mins)
}

// emptyMark is n cells of nothing played: the achievement bar's thin line.
func emptyMark(n int) string { return dimStyle.Render(strings.Repeat(barEmpty, n)) }

// sparkRow draws each value as a bar width cells wide, its height its share
// of the largest, with sep between bars.
func sparkRow(values []time.Duration, width int, sep string) string {
	var top time.Duration
	for _, v := range values {
		top = max(top, v)
	}
	parts := make([]string, len(values))
	for i, v := range values {
		if v <= 0 {
			parts[i] = emptyMark(width)
			continue
		}
		step := int(math.Ceil(float64(v)/float64(top)*8)) - 1
		parts[i] = starStyle.Render(strings.Repeat(string([]rune(sparkSteps)[max(step, 0)]), width))
	}
	return strings.Join(parts, sep)
}

// hbar is a horizontal bar width cells long, filled to share, in eighths.
func hbar(share float64, width int) string {
	eighths := int(math.Round(share * float64(width*8)))
	full, rest := eighths/8, eighths%8
	s := strings.Repeat("█", full)
	if rest > 0 && full < width {
		s += string([]rune(barParts)[rest])
	}
	return starStyle.Render(s)
}

// periodTabs are the periods, the chosen one in brackets.
func (m Model) periodTabs() string {
	tabs := make([]string, len(histPeriods))
	for i, p := range histPeriods {
		if i == m.histPeriod {
			tabs[i] = activeTabStyle.Render("[" + p.label + "]")
		} else {
			tabs[i] = dimStyle.Render(" " + p.label + " ")
		}
	}
	return strings.Join(tabs, " ")
}

// rankHeader is the section line over the ranking: MOST PLAYED and the
// periods, running on to the pane like the games' section line.
func (m Model) rankHeader() string {
	line := "  " + dimStyle.Render("▸─ ") + activeTabStyle.Render("MOST PLAYED") + dimStyle.Render(" ── ") + m.periodTabs() + dimStyle.Render(" ──")
	end := m.tableWidth()
	if m.sideBySide() {
		end = m.paneX() - paneGap
	}
	line = ansi.Truncate(line, max(end-1, 0), "")
	return line + dimStyle.Render(strings.Repeat("─", max(end-ansi.StringWidth(line), 0)))
}

// rankRows fills height rows with the ranking: a column header, a page of
// games as long as the game list's pages, and the page counter.
func (m Model) rankRows(height int) string {
	width := m.tableWidth() - rowStyle.GetHorizontalFrameSize()
	nameWidth := max(width-(4+1+1+8+2+rankBarWidth), 8)
	row := func(rank, name, played string) string {
		return fmt.Sprintf("%-4s %-*s %8s  ", rank, nameWidth, ansi.Truncate(name, nameWidth, "…"), played)
	}
	rows := []string{rowStyle.Render(dimStyle.Render(row("#", "Game", "Played") + "Share"))}
	r := m.ranking()
	per := max(min(m.list.Paginator.PerPage, height-2), 1)
	first := min(m.histSel, max(len(r)-1, 0)) / per * per
	for i := first; i < min(first+per, len(r)); i++ {
		text := row(fmt.Sprintf("%d.", i+1), m.gameName(r[i].App), playedText(r[i].Played))
		bar := hbar(float64(r[i].Played)/float64(r[0].Played), rankBarWidth)
		if i == m.histSel {
			rows = append(rows, selectedStyle.Render(text)+bar)
		} else {
			rows = append(rows, rowStyle.Render(textStyle.Render(text))+bar)
		}
	}
	if len(r) == 0 {
		rows = append(rows, rowStyle.Render(dimStyle.Render("Nothing played in this period.")))
	}
	for len(rows) < height-1 {
		rows = append(rows, "")
	}
	if pages := (len(r) + per - 1) / per; pages > 1 {
		rows = append(rows, rowStyle.Render(dimStyle.Render(fmt.Sprintf("%d/%d", first/per+1, pages))))
	}
	for len(rows) < height {
		rows = append(rows, "")
	}
	return strings.Join(rows[:height], "\n")
}

// historyPane is the ranked game's history, width by height cells.
func (m Model) historyPane(g library.Game, width, height int) string {
	now := m.now()
	inner := min(width-paneStyle.GetHorizontalFrameSize(), paneTextWidth)
	row := func(k, v string) string { return dimStyle.Render(fmt.Sprintf("%-14s", k)) + textStyle.Render(v) }
	lines := []string{paneTitleStyle.Width(inner).Render(strings.ToUpper(g.Label())), ""}
	for _, p := range histPeriods {
		lines = append(lines, row(p.label, playedText(m.hist.Played(g.AppID, m.hist.DaysBack(p.days, now)))))
	}
	lines = append(lines, row("Total", library.FormatHours(g.Playtime, m.playtimeKnown)), "")
	if s, ok := m.hist.Longest(g.AppID); ok {
		lines = append(lines, row("Longest", lengthText(s.Length())+" · "+s.Start.In(time.Local).Format("Jan 2")))
	}
	cur, best := m.hist.Streak(g.AppID, now)
	lines = append(lines, row("Streak", fmt.Sprintf("%d days · best %d", cur, best)))
	lines = append(lines, "", dimStyle.Render(fmt.Sprintf("Last %d weeks", histWeeks)), sparkRow(m.hist.Weeks(g.AppID, histWeeks, now), 2, ""))
	wd := m.hist.Weekdays(g.AppID)
	lines = append(lines, "", dimStyle.Render("Mo Tu We Th Fr Sa Su"), sparkRow(wd[:], 2, " "))
	if last := m.hist.Last(g.AppID, histRecent); len(last) > 0 {
		lines = append(lines, "", dimStyle.Render("Recent sessions"))
		for _, s := range last {
			lines = append(lines, textStyle.Render(s.Start.In(time.Local).Format("Mon Jan 2  15:04"))+"   "+starStyle.Render(lengthText(s.Length())))
		}
	} else {
		lines = append(lines, "", dimStyle.Render("No sessions recorded yet."))
	}
	return paneBox(lines, width, height, inner)
}

// boardPane is the HIGH SCORES board, width by height cells: the period's
// top 8, the year's calendar of played days, and the records.
func (m Model) boardPane(width, height int) string {
	now := m.now()
	inner := width - paneStyle.GetHorizontalFrameSize()
	nameWidth := max(min(inner-(5+1+1+8+2+boardBarWidth), 30), 10)
	calendar := m.calendar(now)
	var records []string
	for _, kv := range m.boardRecords(now) {
		records = append(records, dimStyle.Render(fmt.Sprintf("%-9s", kv[0]))+activeTabStyle.Render(kv[1]))
	}
	// The calendar and the records always show; the places give way, down
	// to a podium of 3, when the pane is short.
	room := height - paneStyle.GetVerticalFrameSize()
	places := max(min(boardPlaces, room-(len(calendar)+len(records)+4)), 3)
	lines := []string{
		activeTabStyle.Render("H I G H   S C O R E S") + dimStyle.Render("   "+histPeriods[m.histPeriod].label),
		"",
	}
	lines = append(lines, m.scoreRows(nameWidth)[:places]...)
	lines = append(lines, "")
	lines = append(lines, calendar...)
	lines = append(lines, "")
	lines = append(lines, records...)
	return paneBox(lines, width, height, inner)
}

// paneBox frames lines as the pane, cut to fit so its border always shows.
func paneBox(lines []string, width, height, inner int) string {
	if room := max(height-paneStyle.GetVerticalFrameSize(), 0); len(lines) > room {
		lines = lines[:room]
	}
	for i, l := range lines {
		lines[i] = ansi.Truncate(l, inner, "…")
	}
	return paneStyle.Width(width).Height(height).MaxHeight(height).Render(strings.Join(lines, "\n"))
}

// ordinal is 1ST, 2ND, 3RD, 4TH... as on an arcade board.
func ordinal(n int) string {
	switch {
	case n%100 >= 11 && n%100 <= 13:
		return fmt.Sprintf("%dTH", n)
	case n%10 == 1:
		return fmt.Sprintf("%dST", n)
	case n%10 == 2:
		return fmt.Sprintf("%dND", n)
	case n%10 == 3:
		return fmt.Sprintf("%dRD", n)
	}
	return fmt.Sprintf("%dTH", n)
}

// scoreRows are the board's places: rank, name dotted out, time and bar;
// places nobody reached are dotted.
func (m Model) scoreRows(nameWidth int) []string {
	r := m.ranking()
	out := make([]string, boardPlaces)
	for i := range out {
		if i >= len(r) {
			out[i] = dimStyle.Render(fmt.Sprintf("%-5s %s %8s", ordinal(i+1), strings.Repeat("·", nameWidth), "---"))
			continue
		}
		name := strings.ToUpper(ansi.Truncate(m.gameName(r[i].App), nameWidth, ""))
		style := textStyle
		if i == 0 {
			style = activeTabStyle
		}
		out[i] = style.Render(fmt.Sprintf("%-5s %s", ordinal(i+1), name)) +
			dimStyle.Render(strings.Repeat(".", max(nameWidth-ansi.StringWidth(name), 0))) +
			style.Render(fmt.Sprintf(" %8s  ", strings.ToUpper(playedText(r[i].Played)))) +
			hbar(float64(r[i].Played)/float64(r[0].Played), boardBarWidth)
	}
	return out
}

// boardRecords are the board's facts: the period's total, and all time's
// longest session, streak, and the weekday played most.
func (m Model) boardRecords(now time.Time) [][2]string {
	out := [][2]string{{"TOTAL", playedText(m.hist.Played(0, m.periodStart()))}}
	if s, ok := m.hist.Longest(0); ok {
		out = append(out, [2]string{"LONGEST", lengthText(s.Length()) + " " + strings.ToUpper(m.gameName(s.AppID))})
	}
	cur, best := m.hist.Streak(0, now)
	out = append(out, [2]string{"STREAK", fmt.Sprintf("%d DAYS (BEST %d)", cur, best)})
	wd := m.hist.Weekdays(0)
	busiest := 0
	for i := range wd {
		if wd[i] > wd[busiest] {
			busiest = i
		}
	}
	if wd[busiest] > 0 {
		days := []string{"MONDAYS", "TUESDAYS", "WEDNESDAYS", "THURSDAYS", "FRIDAYS", "SATURDAYS", "SUNDAYS"}
		out = append(out, [2]string{"MOSTLY", days[busiest]})
	}
	return out
}

// calendar is a year of days, GitHub-style: a column per week, Monday on
// top, a day's shade how long was played; empty days the thin line.
func (m Model) calendar(now time.Time) []string {
	days := m.hist.Days(0)
	shade := func(d time.Duration) int {
		switch {
		case d < 30*time.Minute:
			return 0
		case d < time.Hour:
			return 1
		case d < 2*time.Hour:
			return 2
		case d < 4*time.Hour:
			return 3
		}
		return 4
	}
	today := m.hist.DaysBack(1, now)
	first := today.AddDate(0, 0, -((int(today.Weekday())+6)%7)-7*(calendarWeeks-1)) // the Monday a year of weeks back

	// A month's name goes over its first week, when there's room for it
	// and a space before the next.
	months := []rune(strings.Repeat(" ", calendarWeeks))
	lastMonth, free := time.Month(0), 0
	for w := range calendarWeeks {
		d := first.AddDate(0, 0, 7*w)
		if d.Month() == lastMonth {
			continue
		}
		lastMonth = d.Month()
		if w >= free && w+3 <= calendarWeeks {
			copy(months[w:], []rune(d.Format("Jan")))
			free = w + 4
		}
	}
	out := []string{dimStyle.Render("    " + string(months))}
	for r, name := range []string{"Mon", "", "Wed", "", "Fri", "", "Sun"} {
		var b strings.Builder
		b.WriteString(dimStyle.Render(fmt.Sprintf("%-4s", name)))
		for w := range calendarWeeks {
			d := first.AddDate(0, 0, 7*w+r)
			switch played := days[d]; {
			case d.After(today):
				b.WriteString(" ")
			case played <= 0:
				b.WriteString(emptyMark(1))
			default:
				b.WriteString(lipgloss.NewStyle().Foreground(colorHeat[shade(played)]).Render("■"))
			}
		}
		out = append(out, b.String())
	}
	legend := dimStyle.Render("    less ") + emptyMark(1)
	for _, c := range colorHeat {
		legend += lipgloss.NewStyle().Foreground(c).Render("■")
	}
	return append(out, legend+dimStyle.Render(" more"))
}
