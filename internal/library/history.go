package library

import (
	"cmp"
	"slices"

	"github.com/stronxyo/quarter/internal/steam"
)

// MergeSessions adds the sessions in found that saved doesn't have yet (a
// session is known by its game and start) and returns them all by start
// time, in UTC, with how many were new. Saved sessions stay, so history
// outlives Steam's log.
func MergeSessions(saved, found []steam.Session) ([]steam.Session, int) {
	type key struct {
		app   uint32
		start int64
	}
	seen := make(map[key]bool, len(saved))
	out := make([]steam.Session, 0, len(saved)+len(found))
	add := func(s steam.Session) bool {
		k := key{s.AppID, s.Start.Unix()}
		if seen[k] {
			return false
		}
		seen[k] = true
		s.Start, s.End = s.Start.UTC(), s.End.UTC()
		out = append(out, s)
		return true
	}
	for _, s := range saved {
		add(s)
	}
	added := 0
	for _, s := range found {
		if add(s) {
			added++
		}
	}
	slices.SortStableFunc(out, func(a, b steam.Session) int {
		return cmp.Or(a.Start.Compare(b.Start), cmp.Compare(a.AppID, b.AppID))
	})
	return out, added
}
