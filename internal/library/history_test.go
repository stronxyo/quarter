package library

import (
	"testing"
	"time"

	"github.com/stronxyo/quarter/internal/steam"
)

func TestMergeSessions(t *testing.T) {
	loc := time.FixedZone("UTC-3", -3*60*60)
	day := func(d int) time.Time { return time.Date(2026, 3, d, 20, 0, 0, 0, loc) }
	session := func(app uint32, d int) steam.Session {
		return steam.Session{AppID: app, Start: day(d), End: day(d).Add(time.Hour), EndKnown: true}
	}
	saved := []steam.Session{session(10, 1), session(20, 3)}
	saved[0].Start, saved[0].End = saved[0].Start.UTC(), saved[0].End.UTC()

	// 10 on day 1 again (in local time), 20 on day 2 (new), 10 on day 1 at
	// the same time as another game (new: different game).
	found := []steam.Session{session(10, 1), session(20, 2), session(30, 1)}
	got, added := MergeSessions(saved, found)
	if added != 2 {
		t.Errorf("added: got %d, want 2", added)
	}
	want := []struct {
		app uint32
		day int
	}{{10, 1}, {30, 1}, {20, 2}, {20, 3}}
	if len(got) != len(want) {
		t.Fatalf("got %d sessions, want %d: %+v", len(got), len(want), got)
	}
	for i, w := range want {
		if got[i].AppID != w.app || !got[i].Start.Equal(day(w.day)) {
			t.Errorf("%d: got app %d at %v, want app %d on day %d", i, got[i].AppID, got[i].Start, w.app, w.day)
		}
	}
	for _, s := range got {
		if s.Start.Location() != time.UTC {
			t.Errorf("app %d: start kept in %v, want UTC", s.AppID, s.Start.Location())
		}
	}

	again, added := MergeSessions(got, found)
	if added != 0 || len(again) != len(got) {
		t.Errorf("reading the same log again added %d", added)
	}
}
