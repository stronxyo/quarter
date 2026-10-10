// Package history turns recorded game sessions into play history: time per
// period, per week, per day and per weekday, streaks and records. Sessions
// with no known end, and sessions under a minute, count no time.
package history

import (
	"cmp"
	"slices"
	"time"

	"github.com/stronxyo/quarter/internal/replay"
	"github.com/stronxyo/quarter/internal/steam"
)

// MinSession is the shortest session that counts: shorter ones are a game
// opened and closed, not play.
const MinSession = time.Minute

// History is a set of sessions, read in one time zone, and optionally the
// months of Steam Replay before them.
type History struct {
	sessions []steam.Session // counted ones, by start
	loc      *time.Location
	months   []replay.Month // Replay months before the first recorded session's month
}

// WithReplay adds the Replay months that come before the recorded
// sessions: a month counts from Replay only when it ended before the first
// recorded session's month began, so no time is counted twice.
func (h History) WithReplay(years []replay.Year) History {
	cut := h.firstMonth()
	h.months = nil
	for _, y := range years {
		for _, m := range y.Months {
			if cut.IsZero() || m.Start.Before(cut) {
				h.months = append(h.months, m)
			}
		}
	}
	return h
}

// firstMonth is the month of the first recorded session, as a UTC date
// like Replay's months; zero without sessions.
func (h History) firstMonth() time.Time {
	if len(h.sessions) == 0 {
		return time.Time{}
	}
	t := h.sessions[0].Start.In(h.loc)
	return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// replayPlayed is the Replay time of app (0: all games) in the months that
// begin on or after from.
func (h History) replayPlayed(app uint32, from time.Time) time.Duration {
	f := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	var d time.Duration
	for _, m := range h.months {
		switch {
		case m.Start.Before(f):
		case app == 0:
			d += m.Played
		default:
			d += m.Games[app]
		}
	}
	return d
}

// Months is the Replay time of each month before the recorded sessions,
// keyed by the month's first day (UTC), for app or all games (0).
func (h History) Months(app uint32) map[time.Time]time.Duration {
	out := map[time.Time]time.Duration{}
	for _, m := range h.months {
		if app == 0 {
			out[m.Start] += m.Played
		} else if d := m.Games[app]; d > 0 {
			out[m.Start] += d
		}
	}
	return out
}

// Years is the time played in each year, Replay months and recorded
// sessions together.
func (h History) Years() map[int]time.Duration {
	out := map[int]time.Duration{}
	for _, m := range h.months {
		out[m.Start.Year()] += m.Played
	}
	for _, s := range h.sessions {
		out[s.Start.In(h.loc).Year()] += s.Length()
	}
	return out
}

// HasReplay reports whether Replay months were added.
func (h History) HasReplay() bool { return len(h.months) > 0 }

// New keeps the sessions that count, in loc's days.
func New(sessions []steam.Session, loc *time.Location) History {
	var kept []steam.Session
	for _, s := range sessions {
		if s.Length() >= MinSession {
			kept = append(kept, s)
		}
	}
	slices.SortFunc(kept, func(a, b steam.Session) int { return a.Start.Compare(b.Start) })
	return History{sessions: kept, loc: loc}
}

// DaysBack is the start of a period of days ending today: midnight days-1
// days before now, so 7 is today and the 6 days before.
func (h History) DaysBack(days int, now time.Time) time.Time {
	return day(now.In(h.loc)).AddDate(0, 0, -(days - 1))
}

// Since is the day history starts: the first session's day.
func (h History) Since() time.Time {
	if len(h.sessions) == 0 {
		return time.Time{}
	}
	return day(h.sessions[0].Start.In(h.loc))
}

// Played is the time played since from, in one game (app 0: all games).
func (h History) Played(app uint32, from time.Time) time.Duration {
	d := h.replayPlayed(app, from)
	for _, s := range h.of(app) {
		if !s.Start.Before(from) {
			d += s.Length()
		}
	}
	return d
}

// Entry is one game's time in a ranking.
type Entry struct {
	App    uint32
	Played time.Duration
}

// Top ranks the games by time played since from, most first.
func (h History) Top(from time.Time) []Entry {
	by := map[uint32]time.Duration{}
	for _, s := range h.sessions {
		if !s.Start.Before(from) {
			by[s.AppID] += s.Length()
		}
	}
	f := time.Date(from.Year(), from.Month(), from.Day(), 0, 0, 0, 0, time.UTC)
	for _, m := range h.months {
		if !m.Start.Before(f) {
			for app, d := range m.Games {
				by[app] += d
			}
		}
	}
	out := make([]Entry, 0, len(by))
	for app, d := range by {
		out = append(out, Entry{app, d})
	}
	slices.SortFunc(out, func(a, b Entry) int { return cmp.Or(cmp.Compare(b.Played, a.Played), cmp.Compare(a.App, b.App)) })
	return out
}

// Weeks is the time played in each of the last n weeks (Monday to Sunday),
// oldest first, the current week last.
func (h History) Weeks(app uint32, n int, now time.Time) []time.Duration {
	out := make([]time.Duration, n)
	thisWeek := weekStart(now.In(h.loc))
	for _, s := range h.of(app) {
		w := int(thisWeek.Sub(weekStart(s.Start.In(h.loc))).Hours()/24) / 7
		if w >= 0 && w < n {
			out[n-1-w] += s.Length()
		}
	}
	return out
}

// Days is the time played on each day, keyed by the day's midnight in loc.
func (h History) Days(app uint32) map[time.Time]time.Duration {
	out := map[time.Time]time.Duration{}
	for _, s := range h.of(app) {
		out[day(s.Start.In(h.loc))] += s.Length()
	}
	return out
}

// DayGames is what was played on day d (a day as Days keys it), most played
// first: the HIGH SCORES calendar says it for the day under the pointer.
func (h History) DayGames(d time.Time) []Entry {
	per := map[uint32]time.Duration{}
	for _, s := range h.sessions {
		if day(s.Start.In(h.loc)).Equal(d) {
			per[s.AppID] += s.Length()
		}
	}
	out := make([]Entry, 0, len(per))
	for app, played := range per {
		out = append(out, Entry{App: app, Played: played})
	}
	slices.SortFunc(out, func(a, b Entry) int { return cmp.Or(cmp.Compare(b.Played, a.Played), cmp.Compare(a.App, b.App)) })
	return out
}

// Weekdays is the time played on each weekday, Monday first.
func (h History) Weekdays(app uint32) [7]time.Duration {
	var out [7]time.Duration
	for _, s := range h.of(app) {
		out[(int(s.Start.In(h.loc).Weekday())+6)%7] += s.Length()
	}
	return out
}

// Last returns the game's latest sessions, newest first, at most n.
func (h History) Last(app uint32, n int) []steam.Session {
	of := h.of(app)
	out := make([]steam.Session, 0, n)
	for i := len(of) - 1; i >= 0 && len(out) < n; i-- {
		out = append(out, of[i])
	}
	return out
}

// Longest is the longest session, in one game or all.
func (h History) Longest(app uint32) (steam.Session, bool) {
	var best steam.Session
	for _, s := range h.of(app) {
		if s.Length() > best.Length() {
			best = s
		}
	}
	return best, best.Length() > 0
}

// Streak is the current run of days in a row with play (today or yesterday
// counting as its end), and the longest run ever.
func (h History) Streak(app uint32, now time.Time) (current, best int) {
	days := h.Days(app)
	var list []time.Time
	for d := range days {
		list = append(list, d)
	}
	slices.SortFunc(list, func(a, b time.Time) int { return a.Compare(b) })
	run := 0
	for i, d := range list {
		if i > 0 && sameDay(list[i-1].AddDate(0, 0, 1), d) {
			run++
		} else {
			run = 1
		}
		best = max(best, run)
	}
	today := day(now.In(h.loc))
	if len(list) > 0 {
		last := list[len(list)-1]
		if sameDay(last, today) || sameDay(last, today.AddDate(0, 0, -1)) {
			current = run
		}
	}
	return current, best
}

func (h History) of(app uint32) []steam.Session {
	if app == 0 {
		return h.sessions
	}
	var out []steam.Session
	for _, s := range h.sessions {
		if s.AppID == app {
			out = append(out, s)
		}
	}
	return out
}

func day(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func sameDay(a, b time.Time) bool {
	ya, ma, da := a.Date()
	yb, mb, db := b.Date()
	return ya == yb && ma == mb && da == db
}

// weekStart is the Monday of t's week.
func weekStart(t time.Time) time.Time {
	d := day(t)
	return d.AddDate(0, 0, -((int(d.Weekday()) + 6) % 7))
}
