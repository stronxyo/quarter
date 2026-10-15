package ui

import (
	"slices"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/list"
)

// searchFilter is the list's search. Every word typed must be in the game's
// name, anywhere and in any order, with capitals and punctuation ignored
// ("spiderman" finds "Marvel's Spider-Man"). The best matches come first: a
// word at the start of the name, then at the start of a word in it, then
// inside a word; games that tie keep the list's order, its sort. Bubbles'
// own filter is fuzzy, letters in order anywhere, so "cyber" also found
// "Black Myth: Wukong Benchmark Tool" (specs/BATCH_08_SPEC.md, "Search by
// words").
func searchFilter(term string, targets []string) []list.Rank {
	query, _ := searchForm(term)
	words := strings.Fields(query)
	type hit struct{ index, score int }
	hits := make([]hit, 0, len(targets))
	for i, t := range targets {
		if score, ok := searchScore(t, words); ok {
			hits = append(hits, hit{i, score})
		}
	}
	slices.SortStableFunc(hits, func(a, b hit) int { return a.score - b.score })
	ranks := make([]list.Rank, len(hits))
	for i, h := range hits {
		ranks[i] = list.Rank{Index: h.index}
	}
	return ranks
}

// searchScore is how well name matches the words, lower being better: per
// word 0 at the start of the name, 1 at the start of a word, 2 inside one.
// ok is false when a word isn't in the name.
func searchScore(name string, words []string) (score int, ok bool) {
	text, starts := searchForm(name)
	for _, w := range words {
		best := -1
		for from := 0; best != 0; {
			i := strings.Index(text[from:], w)
			if i < 0 {
				break
			}
			at := from + i
			s := 2
			if at == 0 {
				s = 0
			} else if _, found := slices.BinarySearch(starts, at); found {
				s = 1
			}
			if best < 0 || s < best {
				best = s
			}
			from = at + 1
		}
		if best < 0 {
			return 0, false
		}
		score += best
	}
	return score, true
}

// searchForm is s as search compares it: lower case, letters, digits and
// spaces only, so "Spider-Man" is "spiderman". starts are the byte offsets
// where its words start: after a space or dropped punctuation (the Man in
// Spider-Man), or at a capital after a small letter (the Mania in
// ShootMania).
func searchForm(s string) (text string, starts []int) {
	var b strings.Builder
	b.Grow(len(s))
	boundary := true
	var prev rune
	for _, r := range s {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r):
			if boundary || unicode.IsUpper(r) && unicode.IsLower(prev) {
				starts = append(starts, b.Len())
			}
			b.WriteRune(unicode.ToLower(r))
			boundary = false
		case unicode.IsSpace(r):
			b.WriteByte(' ')
			boundary = true
		default:
			boundary = true
		}
		prev = r
	}
	return b.String(), starts
}
