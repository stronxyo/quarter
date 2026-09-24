package library

import (
	"sort"
	"strings"
	"time"

	"github.com/stronxyo/quarter/internal/config"
)

// SortOrder is the column the library is ordered by.
type SortOrder int

const (
	SortName SortOrder = iota
	SortPlaytime
	SortLastPlayed
)

// DefaultDesc is the natural direction for each column: names A to Z, but most
// played and most recent first.
func (s SortOrder) DefaultDesc() bool { return s != SortName }

// Filter narrows which games are shown.
type Filter int

const (
	FilterAll Filter = iota
	FilterRecent
	filterCount
)

func (f Filter) String() string {
	return [...]string{"All", "Recent"}[f]
}

// Next cycles to the following filter.
func (f Filter) Next() Filter { return (f + 1) % filterCount }

// Filters lists every filter in display order.
func Filters() []Filter {
	fs := make([]Filter, filterCount)
	for i := range fs {
		fs[i] = Filter(i)
	}
	return fs
}

// View returns the games that pass f, ordered by s (reversed when desc),
// without changing games. Ties are always broken by name, A to Z.
func View(games []Game, f Filter, s SortOrder, desc bool, now time.Time) []Game {
	out := make([]Game, 0, len(games))
	for _, g := range games {
		if f == FilterRecent && (g.LastPlayed.IsZero() || now.Sub(g.LastPlayed) > config.RecentWindow) {
			continue
		}
		out = append(out, g)
	}

	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		var less, equal bool
		switch s {
		case SortPlaytime:
			less, equal = a.Playtime < b.Playtime, a.Playtime == b.Playtime
		case SortLastPlayed:
			less, equal = a.LastPlayed.Before(b.LastPlayed), a.LastPlayed.Equal(b.LastPlayed)
		default:
			an, bn := strings.ToLower(a.Name), strings.ToLower(b.Name)
			less, equal = an < bn, an == bn
		}
		if equal {
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
		return less != desc
	})
	return out
}
