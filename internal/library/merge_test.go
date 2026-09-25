package library

import (
	"testing"
	"time"

	"github.com/stronxyo/quarter/internal/steam"
)

func TestFromSteam(t *testing.T) {
	installed := []steam.Manifest{
		{AppID: 620, Name: "portal 2", StateFlags: 4, LibraryPath: "C"},
		{AppID: 400, Name: "Portal", StateFlags: 4, LibraryPath: "D"},
		{AppID: 10, Name: "Counter-Strike", StateFlags: 4},
	}
	stats := map[uint32]steam.PlayStats{
		620: {Playtime: 754 * time.Minute, LastPlayed: time.Unix(1726000000, 0)},
		999: {Playtime: time.Hour}, // not installed, ignored
	}

	games := FromSteam(installed, stats, map[uint32]steam.Achievements{620: {Achieved: 20, Total: 51}})

	wantOrder := []uint32{10, 400, 620} // case-insensitive by name
	if len(games) != len(wantOrder) {
		t.Fatalf("got %d games, want %d", len(games), len(wantOrder))
	}
	for i, id := range wantOrder {
		if games[i].AppID != id {
			t.Errorf("position %d: got appid %d, want %d", i, games[i].AppID, id)
		}
	}

	p2 := games[2]
	if p2.Achievements != (steam.Achievements{Achieved: 20, Total: 51}) {
		t.Errorf("Portal 2 achievements: got %+v", p2.Achievements)
	}
	if p2.Playtime != 754*time.Minute || !p2.LastPlayed.Equal(time.Unix(1726000000, 0)) || !p2.Installed || p2.LibraryPath != "C" {
		t.Errorf("Portal 2 merged wrong: %+v", p2)
	}
	if games[1].Playtime != 0 || !games[1].LastPlayed.IsZero() {
		t.Errorf("Portal has no stats, got %+v", games[1])
	}
}

func TestFromSteamNilStats(t *testing.T) {
	games := FromSteam([]steam.Manifest{{AppID: 620, Name: "Portal 2", StateFlags: 4}}, nil, nil)
	if len(games) != 1 || games[0].Playtime != 0 {
		t.Errorf("got %+v, want one game with zero playtime", games)
	}
}
