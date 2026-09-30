package history

import (
	"slices"
	"testing"
	"time"

	"github.com/stronxyo/quarter/internal/replay"
	"github.com/stronxyo/quarter/internal/steam"
)

// loc is a zone away from UTC, so day boundaries are checked in local time.
var loc = time.FixedZone("UTC-3", -3*60*60)

// now is Wednesday 2026-09-30, 12:00 local.
var now = time.Date(2026, 9, 30, 12, 0, 0, 0, loc)

// at is a local time on day d of September 2026.
func at(d, hour, minute int) time.Time { return time.Date(2026, 9, d, hour, minute, 0, 0, loc) }

func played(app uint32, start time.Time, length time.Duration) steam.Session {
	return steam.Session{AppID: app, Start: start.UTC(), End: start.Add(length).UTC(), EndKnown: true}
}

var sample = New([]steam.Session{
	played(10, at(30, 9, 0), 2*time.Hour),                             // today, Wed
	played(10, at(29, 22, 0), time.Hour),                              // yesterday, Tue
	played(20, at(28, 20, 0), 3*time.Hour),                            // Mon
	played(10, at(27, 20, 0), 30*time.Minute),                         // Sun, last week
	played(20, at(1, 20, 0), 5*time.Hour),                             // Sep 1, a Tuesday
	played(30, at(30, 10, 0), 30*time.Second),                         // too short: not counted
	{AppID: 40, Start: at(29, 10, 0).UTC(), End: at(29, 10, 0).UTC()}, // no known end
	played(10, at(26, 23, 30), 20*time.Minute),                        // Sat 23:30 local is Sun in UTC: counts on Sat
}, loc)

func TestPlayed(t *testing.T) {
	tests := []struct {
		app  uint32
		from time.Time
		want time.Duration
	}{
		{10, sample.DaysBack(1, now), 2 * time.Hour},                                             // today
		{10, sample.DaysBack(7, now), 2*time.Hour + time.Hour + 30*time.Minute + 20*time.Minute}, // Sep 24 on
		{0, sample.DaysBack(7, now), 6*time.Hour + 50*time.Minute},
		{0, sample.DaysBack(30, now), 11*time.Hour + 50*time.Minute},
		{30, time.Time{}, 0}, // a 30 s session doesn't count
		{40, time.Time{}, 0}, // an unknown end doesn't count
	}
	for _, tt := range tests {
		if got := sample.Played(tt.app, tt.from); got != tt.want {
			t.Errorf("app %d from %v: got %v, want %v", tt.app, tt.from, got, tt.want)
		}
	}
}

func TestDaysBack(t *testing.T) {
	if got, want := sample.DaysBack(7, now), time.Date(2026, 9, 24, 0, 0, 0, 0, loc); !got.Equal(want) {
		t.Errorf("7 days back: got %v, want %v", got, want)
	}
}

func TestTop(t *testing.T) {
	got := sample.Top(time.Time{})
	want := []Entry{{20, 8 * time.Hour}, {10, 3*time.Hour + 50*time.Minute}}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if got := sample.Top(sample.DaysBack(1, now)); len(got) != 1 || got[0].App != 10 {
		t.Errorf("today: got %v, want only 10", got)
	}
}

func TestWeeks(t *testing.T) {
	// This week (Mon 28 to today) and the one before (Mon 21 to Sun 27).
	got := sample.Weeks(0, 2, now)
	want := []time.Duration{30*time.Minute + 20*time.Minute, 6 * time.Hour}
	if !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}

func TestDaysAndWeekdays(t *testing.T) {
	days := sample.Days(10)
	if got := days[time.Date(2026, 9, 26, 0, 0, 0, 0, loc)]; got != 20*time.Minute {
		t.Errorf("Sat 23:30 local should count on Saturday: %v", got)
	}
	wd := sample.Weekdays(0)
	if wd[1] != time.Hour+5*time.Hour || wd[2] != 2*time.Hour || wd[0] != 3*time.Hour {
		t.Errorf("weekdays (Mon first): got %v", wd)
	}
}

func TestLastAndLongest(t *testing.T) {
	last := sample.Last(10, 2)
	if len(last) != 2 || !last[0].Start.Equal(at(30, 9, 0)) || !last[1].Start.Equal(at(29, 22, 0)) {
		t.Errorf("last two of 10, newest first: got %v", last)
	}
	if s, ok := sample.Longest(0); !ok || s.AppID != 20 || s.Length() != 5*time.Hour {
		t.Errorf("longest: got %v %v", s, ok)
	}
	if _, ok := New(nil, loc).Longest(0); ok {
		t.Error("no sessions, no longest")
	}
}

func TestStreak(t *testing.T) {
	// Days with play: Sep 1, 26, 27, 28, 29, 30.
	cur, best := sample.Streak(0, now)
	if cur != 5 || best != 5 {
		t.Errorf("got current %d best %d, want 5 and 5", cur, best)
	}
	// Two days later the streak has ended; the best stays.
	if cur, best := sample.Streak(0, now.AddDate(0, 0, 2)); cur != 0 || best != 5 {
		t.Errorf("after a gap: got %d and %d, want 0 and 5", cur, best)
	}
	// Played yesterday but not yet today: still going.
	if cur, _ := sample.Streak(0, now.AddDate(0, 0, 1)); cur != 5 {
		t.Errorf("the day after: got %d, want 5", cur)
	}
}

// Past years (Steam Replay) fill the months before the first recorded
// session's month, and only those, so no time counts twice.
func TestWithReplay(t *testing.T) {
	month := func(y int, m time.Month, played time.Duration, games map[uint32]time.Duration) replay.Month {
		return replay.Month{Start: time.Date(y, m, 1, 0, 0, 0, 0, time.UTC), Played: played, Games: games}
	}
	years := []replay.Year{
		{Year: 2025, Months: []replay.Month{
			month(2025, time.July, 10*time.Hour, map[uint32]time.Duration{10: 6 * time.Hour, 50: 4 * time.Hour}),
			month(2025, time.August, 5*time.Hour, map[uint32]time.Duration{10: 5 * time.Hour}),
		}},
		{Year: 2026, Months: []replay.Month{
			month(2026, time.August, 99*time.Hour, nil),                                             // before the sample's first session (Sep 1, 2026): kept
			month(2026, time.September, 50*time.Hour, map[uint32]time.Duration{10: 50 * time.Hour}), // the first recorded month: left out
		}},
	}
	h := sample.WithReplay(years)
	if !h.HasReplay() || sample.HasReplay() {
		t.Error("WithReplay returns a history with the months, leaving the first as it was")
	}
	if got := h.Months(0); len(got) != 3 || got[time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)] != 0 {
		t.Errorf("months kept: %v (September 2026 has sessions, so it isn't from Replay)", got)
	}
	// All of it: the Replay months plus the sessions (11h 50m).
	if got, want := h.Played(0, time.Time{}), 114*time.Hour+11*time.Hour+50*time.Minute; got != want {
		t.Errorf("all time: got %v, want %v", got, want)
	}
	// A game over a period starting in August 2025: August's Replay and its sessions.
	if got, want := h.Played(10, time.Date(2025, 8, 1, 0, 0, 0, 0, loc)), 5*time.Hour+3*time.Hour+50*time.Minute; got != want {
		t.Errorf("app 10 since August 2025: got %v, want %v", got, want)
	}
	// A month that begins before the period isn't counted in part.
	if got := h.Played(50, time.Date(2025, 7, 15, 0, 0, 0, 0, loc)); got != 0 {
		t.Errorf("July began before July 15: got %v", got)
	}
	top := h.Top(time.Time{})
	if top[0].App != 10 || top[0].Played != 11*time.Hour+3*time.Hour+50*time.Minute {
		t.Errorf("top: %v", top)
	}
	years2 := h.Years()
	if years2[2025] != 15*time.Hour || years2[2026] != 99*time.Hour+11*time.Hour+50*time.Minute {
		t.Errorf("years: %v", years2)
	}
	if h.WithReplay(nil).HasReplay() {
		t.Error("WithReplay(nil) removes them")
	}
}

func TestSince(t *testing.T) {
	if got, want := sample.Since(), time.Date(2026, 9, 1, 0, 0, 0, 0, loc); !got.Equal(want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if !New(nil, loc).Since().IsZero() {
		t.Error("no sessions: no start")
	}
}
