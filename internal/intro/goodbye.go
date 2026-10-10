package intro

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"time"
)

// The goodbye (Batch 8, A3): printed into the shell as Quarter closes, the
// start screen's "insert quarter" answered. One line, at random and never
// the same twice in a row, in the logo's lowercase letters at the size of
// "insert quarter", one column in from the left edge; under it the
// time played in games this session, when there was any. Pablo's pick,
// 2026-09-28; moved left 2026-09-29 (a smaller cut was tried and not taken).
var goodbyes = []string{
	"keep the change",
	"quarter returned",
	"out of quarters already?",
	"thanks for the quarter",
	"shutting down...",
	"session saved",
	"it's now safe to close your terminal",
	"thanks for playing",
	"high score saved",
	"powering down...",
	"see you next round",
}

// GoodbyeCount is how many goodbyes there are.
func GoodbyeCount() int { return len(goodbyes) }

// GoodbyeKey is goodbye i's name for config.json: its words joined by
// dashes, punctuation dropped ("it's now safe..." is "its-now-safe-...").
func GoodbyeKey(i int) string {
	var b strings.Builder
	for _, r := range goodbyes[i] {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
		case r == ' ':
			b.WriteRune('-')
		}
	}
	return b.String()
}

// FindGoodbye is the goodbye with the given key.
func FindGoodbye(key string) (int, bool) {
	for i := range goodbyes {
		if GoodbyeKey(i) == key {
			return i, true
		}
	}
	return 0, false
}

// PickGoodbye chooses a goodbye at random, never last (the previous one's
// index, or -1).
func PickGoodbye(last int, r *rand.Rand) int { return pick(len(goodbyes), last, r) }

// pick is a random index below n, never last, every other one equally
// likely.
func pick(n, last int, r *rand.Rand) int {
	if n < 2 {
		return 0
	}
	i := r.IntN(n - 1)
	if last >= 0 && i >= last {
		i++
	}
	return i
}

// Goodbye is goodbye i as printed in a terminal width columns wide, with
// the time played under it when there was any. A line too wide for the
// terminal in big letters is printed as plain text instead.
func Goodbye(i, width int, played time.Duration) string {
	text := goodbyes[i]
	rows := logoText(text)
	if len([]rune(rows[0])) > width-2 {
		rows = []string{text}
	}
	var b strings.Builder
	b.WriteString("\n")
	for y, row := range rows {
		c := colAccent
		if len(rows) > 1 {
			c = mix(colTop, colBottom, float64(y)/float64(len(rows)-1))
		}
		line(&b, strings.TrimRight(row, " "), c)
	}
	if text := playedText(played); text != "" {
		b.WriteString("\n")
		line(&b, text, colDim)
	}
	b.WriteString("\n")
	return b.String()
}

// line writes one line of text one column in from the left edge, in c.
func line(b *strings.Builder, text string, c rgb) {
	fmt.Fprintf(b, " \x1b[38;2;%d;%d;%dm%s\x1b[m\n", int(c.r), int(c.g), int(c.b), text)
}

// playedText is "played this session: 2h 10m", or "" when nothing was
// played. Under a minute counts as a minute.
func playedText(d time.Duration) string {
	if d <= 0 {
		return ""
	}
	m := max(int(d.Round(time.Minute)/time.Minute), 1)
	switch {
	case m < 60:
		return fmt.Sprintf("played this session: %dm", m)
	case m%60 == 0:
		return fmt.Sprintf("played this session: %dh", m/60)
	}
	return fmt.Sprintf("played this session: %dh %dm", m/60, m%60)
}
