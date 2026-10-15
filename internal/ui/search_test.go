package ui

import (
	"slices"
	"testing"
)

// filtered is what the list's search shows for query, in order.
func filtered(query string, names ...string) []string {
	var got []string
	for _, r := range newList().Filter(query, names) {
		got = append(got, names[r.Index])
	}
	return got
}

func TestSearchMatchesWords(t *testing.T) {
	names := []string{
		"Cyberpunk 2077",
		"Black Myth: Wukong Benchmark Tool",
		"Marvel's Spider-Man Remastered",
		"S.T.A.L.K.E.R. 2: Heart of Chornobyl",
		"Counter-Strike 2",
		"Rocket League",
	}
	for _, c := range []struct {
		query string
		want  []string
	}{
		{"cyber", []string{"Cyberpunk 2077"}}, // not blaCk mYth ... BEnchmaRk
		{"CYBER", []string{"Cyberpunk 2077"}},
		{"wukong black", []string{"Black Myth: Wukong Benchmark Tool"}}, // any order
		{"spiderman", []string{"Marvel's Spider-Man Remastered"}},       // punctuation ignored
		{"stalker", []string{"S.T.A.L.K.E.R. 2: Heart of Chornobyl"}},
		{"counterstrike", []string{"Counter-Strike 2"}},
		{"rl", nil},  // no letters scattered across a name
		{"-", names}, // only punctuation: everything, like an empty search
	} {
		if got := filtered(c.query, names...); !slices.Equal(got, c.want) {
			t.Errorf("search %q: got %q, want %q", c.query, got, c.want)
		}
	}
}

// The best matches come first: the start of the name, then the start of a
// word in it, then inside a word. Games that tie keep the list's order.
func TestSearchBestFirst(t *testing.T) {
	for _, c := range []struct {
		query       string
		names, want []string
	}{
		{"war",
			[]string{"Alien Swarm: Reactive Drop", "War Thunder", "Warhammer 40,000: Speed Freeks", "World of Warships"},
			[]string{"War Thunder", "Warhammer 40,000: Speed Freeks", "World of Warships", "Alien Swarm: Reactive Drop"}},
		{"man", // after a hyphen is a word's start
			[]string{"HITMAN", "Marvel's Spider-Man", "Manor Lords"},
			[]string{"Manor Lords", "Marvel's Spider-Man", "HITMAN"}},
		{"mania", // so is a capital inside a name
			[]string{"Romania Rally", "ShootMania Elite"},
			[]string{"ShootMania Elite", "Romania Rally"}},
		{"war", // a tie keeps the list's order, whatever its sort
			[]string{"Warhammer 40,000: Speed Freeks", "War Thunder"},
			[]string{"Warhammer 40,000: Speed Freeks", "War Thunder"}},
	} {
		if got := filtered(c.query, c.names...); !slices.Equal(got, c.want) {
			t.Errorf("search %q in %q: got %q, want %q", c.query, c.names, got, c.want)
		}
	}
}
