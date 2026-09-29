package ui

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/stronxyo/quarter/internal/library"
	"github.com/stronxyo/quarter/internal/webapi"
)

// fakeStoreSource answers from a fixed table and records what was asked.
type fakeStoreSource struct {
	mu    sync.Mutex
	asked []uint32
	info  map[uint32]webapi.StoreInfo
	errs  map[uint32]error
}

func (f *fakeStoreSource) fetch(_ context.Context, id uint32) (webapi.StoreInfo, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.asked = append(f.asked, id)
	if err := f.errs[id]; err != nil {
		return webapi.StoreInfo{}, err
	}
	return f.info[id], nil
}

// drive runs cmd and every command that follows from store messages, the
// way the program would, ignoring everything else.
func drive(m tea.Model, cmd tea.Cmd) tea.Model {
	queue := []tea.Cmd{cmd}
	for steps := 0; len(queue) > 0 && steps < 200; steps++ {
		c := queue[0]
		queue = queue[1:]
		if c == nil {
			continue
		}
		switch msg := c().(type) {
		case tea.BatchMsg:
			queue = append(queue, msg...)
		case storeWantMsg, storeDoneMsg, resumeMsg:
			var next tea.Cmd
			m, next = m.Update(msg)
			queue = append(queue, next)
		}
	}
	return m
}

var threeGames = library.Result{PlaytimeKnown: true, Games: []library.Game{
	{AppID: 1, Name: "Alpha", Installed: true},
	{AppID: 2, Name: "Bravo", Installed: true},
	{AppID: 3, Name: "Charlie", Installed: true},
}}

func withStore(t *testing.T, src *fakeStoreSource, cache map[uint32]webapi.StoreEntry, saved *[]map[uint32]webapi.StoreEntry) tea.Model {
	t.Helper()
	m := New(Options{
		FetchStore: src.fetch,
		StoreCache: cache,
		SaveStore: func(c map[uint32]webapi.StoreEntry) error {
			*saved = append(*saved, c)
			return nil
		},
	})
	m.debounce = 0
	// A still clock, and waits of a millisecond: a pause after a failure
	// lasts until the test moves the clock on.
	clock := time.Now()
	m.now = func() time.Time { return clock }
	m.jitter = func(time.Duration) time.Duration { return time.Millisecond }
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 150, Height: 30})
	tm, cmd := tm.Update(scanDoneMsg{res: threeGames})
	return drive(tm, cmd)
}

// later moves the model's clock on by d.
func later(m tea.Model, d time.Duration) tea.Model {
	mm := m.(Model)
	at := mm.now().Add(d)
	mm.now = func() time.Time { return at }
	return mm
}

func TestStoreDetailsFetchedAndShown(t *testing.T) {
	src := &fakeStoreSource{info: map[uint32]webapi.StoreInfo{
		1: {Type: "game", Developers: []string{"Alpha Studio"}, ReleaseDate: "6 Jul, 2015", Genres: []string{"Action", "Racing"}, Description: "Cars and a ball."},
		2: {Type: "game"},
		3: {Type: "game"},
	}}
	var saved []map[uint32]webapi.StoreEntry
	m := withStore(t, src, nil, &saved)

	view := ansi.Strip(m.(Model).View().Content)
	for _, want := range []string{"Alpha Studio · 2015", "Action, Racing", "Cars and a ball."} {
		if !strings.Contains(view, want) {
			t.Errorf("pane missing %q", want)
		}
	}
	if len(src.asked) != 3 || src.asked[0] != 1 {
		t.Errorf("want the selected game first, then the rest: asked %v", src.asked)
	}
	if last := saved[len(saved)-1]; len(last) != 3 {
		t.Errorf("cache should hold all 3 games, got %d", len(last))
	}
}

func TestFreshCacheIsNotFetchedAgain(t *testing.T) {
	now := time.Now()
	cache := map[uint32]webapi.StoreEntry{
		1: {Info: webapi.StoreInfo{Developers: []string{"Cached Dev"}}, Fetched: now},
		2: {Fetched: now},
		3: {Failed: true, Fetched: now},
	}
	src := &fakeStoreSource{}
	var saved []map[uint32]webapi.StoreEntry
	m := withStore(t, src, cache, &saved)
	if len(src.asked) != 0 {
		t.Errorf("fresh cache should mean no requests, asked %v", src.asked)
	}
	if !strings.Contains(ansi.Strip(m.(Model).View().Content), "Cached Dev") {
		t.Error("cached details should show")
	}
}

func TestStoreFailures(t *testing.T) {
	src := &fakeStoreSource{
		info: map[uint32]webapi.StoreInfo{3: {Type: "game"}},
		errs: map[uint32]error{1: webapi.ErrNotOnStore, 2: webapi.Unreachable("store 2", errors.New("dial tcp: no route"))},
	}
	var saved []map[uint32]webapi.StoreEntry
	m := withStore(t, src, nil, &saved)

	if view := ansi.Strip(m.(Model).View().Content); !strings.Contains(view, "not on the Steam store") {
		t.Errorf("app 1 is not on the store:\n%s", view)
	}
	last := saved[len(saved)-1]
	if !last[1].Failed {
		t.Error("'not on the store' should be cached as a failure")
	}
	if _, ok := last[2]; ok {
		t.Error("a network error must not be cached")
	}

	m, _ = m.Update(press("j")) // Bravo, whose request failed on the network
	view := ansi.Strip(m.(Model).View().Content)
	if !strings.Contains(view, "store info unavailable, retrying") {
		t.Errorf("app 2 failed on the network:\n%s", view)
	}
	if !strings.Contains(view, "3 games · offline") {
		t.Errorf("offline should show beside the game count:\n%s", view)
	}
	// The store pauses after the failure, so app 3 isn't asked yet.
	if n := len(src.asked); n != 2 {
		t.Errorf("want 2 requests (paused after a network failure), got %d: %v", n, src.asked)
	}

	// The network is back when the pause ends: the selected game is asked
	// first, then the rest, and offline goes.
	delete(src.errs, 2)
	src.info[2] = webapi.StoreInfo{Type: "game", Developers: []string{"Bravo Dev"}}
	m = later(m, time.Hour)
	m = drive(m, func() tea.Msg { return resumeMsg{} })
	if got := src.asked[2:]; len(got) != 2 || got[0] != 2 || got[1] != 3 {
		t.Errorf("after resuming: asked %v, want [2 3]", got)
	}
	view = ansi.Strip(m.(Model).View().Content)
	if strings.Contains(view, "offline") || !strings.Contains(view, "Bravo Dev") {
		t.Errorf("recovered: want Bravo's details and no offline marker:\n%s", view)
	}
}

func TestStoreRateLimited(t *testing.T) {
	src := &fakeStoreSource{}
	var saved []map[uint32]webapi.StoreEntry
	m := withStore(t, src, map[uint32]webapi.StoreEntry{}, &saved).(Model)
	m.storeInfo = map[uint32]webapi.StoreEntry{} // nothing known yet
	m.fetching = 1

	// The store asks for two minutes: it is paused for exactly that, and
	// the pane says it's busy, not offline.
	limited := &webapi.RateLimitError{Status: 429, RetryAfter: 2 * time.Minute}
	tm, _ := m.Update(storeDoneMsg{appID: 1, err: fmt.Errorf("store 1: %w", limited), at: m.now()})
	m = tm.(Model)
	if got := m.storePause.until.Sub(m.now()); got != 2*time.Minute {
		t.Errorf("paused for %v, want the 2m the store asked", got)
	}
	view := ansi.Strip(m.View().Content)
	if !strings.Contains(view, "store busy, retrying") || strings.Contains(view, "offline") {
		t.Errorf("want the busy message and no offline marker:\n%s", view)
	}

	// Without Retry-After: 60 s, then doubling, at most 10 minutes.
	m.jitter = func(d time.Duration) time.Duration { return d }
	m.storePause = pause{}
	var waits []time.Duration
	for range 6 {
		m.fetching = 1
		tm, _ = m.Update(storeDoneMsg{appID: 1, err: &webapi.RateLimitError{Status: 429}, at: m.now()})
		m = tm.(Model)
		waits = append(waits, m.storePause.until.Sub(m.now()))
	}
	want := []time.Duration{time.Minute, 2 * time.Minute, 4 * time.Minute, 8 * time.Minute, 10 * time.Minute, 10 * time.Minute}
	if !slices.Equal(waits, want) {
		t.Errorf("waits: got %v, want %v", waits, want)
	}
}

// A game the server fails on doesn't hold up the rest of the fill.
func TestStoreServerErrorSkipsGame(t *testing.T) {
	src := &fakeStoreSource{
		info: map[uint32]webapi.StoreInfo{1: {Type: "game"}, 3: {Type: "game"}},
		errs: map[uint32]error{2: fmt.Errorf("store 2: HTTP 500: %w", webapi.ErrServer)},
	}
	var saved []map[uint32]webapi.StoreEntry
	m := withStore(t, src, nil, &saved)
	m = later(m, time.Hour)
	m = drive(m, func() tea.Msg { return resumeMsg{} })
	if got := src.asked; len(got) != 3 || got[2] != 3 {
		t.Errorf("asked %v, want 1, 2, then 3 after the pause", got)
	}
	if view := ansi.Strip(m.(Model).View().Content); strings.Contains(view, "offline") {
		t.Errorf("a server error isn't offline:\n%s", view)
	}
	if _, ok := saved[len(saved)-1][2]; ok {
		t.Error("a server error must not be cached")
	}
}

func TestStoreDemoTagsTheName(t *testing.T) {
	src := &fakeStoreSource{info: map[uint32]webapi.StoreInfo{2: {Type: "demo"}, 1: {Type: "game"}, 3: {Type: "game"}}}
	var saved []map[uint32]webapi.StoreEntry
	m := withStore(t, src, nil, &saved)
	if !strings.Contains(ansi.Strip(m.(Model).View().Content), "Bravo (Demo)") {
		t.Error("the store says Bravo is a demo; its row should say so")
	}
}

func TestStaleWantIsIgnored(t *testing.T) {
	src := &fakeStoreSource{}
	m := New(Options{FetchStore: src.fetch})
	m.debounce = 0
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 150, Height: 30})
	tm, _ = tm.Update(scanDoneMsg{res: threeGames}) // don't run the backfill
	mm := tm.(Model)
	mm.fetching = 0

	// A want for Charlie arrives, but Alpha is selected: the user moved on.
	if _, cmd := mm.Update(storeWantMsg{appID: 3}); cmd != nil {
		drive(mm, cmd)
	}
	for _, id := range src.asked {
		if id == 3 {
			t.Error("a want for a game no longer selected must not jump the queue")
		}
	}
}

func TestAchievementTotalFromStoreWhenCacheIsEmpty(t *testing.T) {
	src := &fakeStoreSource{info: map[uint32]webapi.StoreInfo{1: {Type: "game", Achievements: 33}, 2: {Type: "game"}, 3: {Type: "game"}}}
	var saved []map[uint32]webapi.StoreEntry
	m := withStore(t, src, nil, &saved)
	if !strings.Contains(ansi.Strip(m.(Model).View().Content), "0/33    ──────────────") {
		t.Error("with no local progress, the store's total should show as 0/33")
	}
}
