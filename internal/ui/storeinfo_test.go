package ui

import (
	"context"
	"errors"
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
		case storeWantMsg, storeDoneMsg:
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
	var tm tea.Model = m
	tm, _ = tm.Update(tea.WindowSizeMsg{Width: 150, Height: 30})
	tm, cmd := tm.Update(scanDoneMsg{res: threeGames})
	return drive(tm, cmd)
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
		errs: map[uint32]error{1: webapi.ErrNotOnStore, 2: errors.New("no network")},
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
	if view := ansi.Strip(m.(Model).View().Content); !strings.Contains(view, "store info unavailable") {
		t.Errorf("app 2 failed on the network:\n%s", view)
	}
	if n := len(src.asked); n != 3 {
		t.Errorf("each game asked once this run, got %d requests", n)
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
