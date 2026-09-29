// Package intro draws Quarter's start screen: the quarter_ logo in its box,
// "insert quarter" under it and "press any key", with a few lines of header
// above (the system check), and the intro animations that lead into it
// (Batch 8, A3; designed in Pablo's Intro lab). It draws whole frames as text
// with 24-bit colour; the UI decides when.
package intro

import (
	"fmt"
	"math"
	"slices"
	"strings"
)

// rgb is a colour, 0 to 255 per channel.
type rgb struct{ r, g, b float64 }

// Quarter's palette, as in internal/ui/styles.go.
var (
	colTop    = rgb{0xFF, 0xE0, 0x8A} // the logo gradient's top
	colBottom = rgb{0xFF, 0x7A, 0x3D} // and bottom
	colAccent = rgb{0xFF, 0xAF, 0x63}
	colDim    = rgb{0xB0, 0x7A, 0x45} // the shadow, dim text
	colHot    = rgb{0xFF, 0xF4, 0xC8} // overbright, for leading edges
	colText   = rgb{0xF0, 0xE4, 0xD2}
	colBG     = rgb{0x0C, 0x0C, 0x0C} // the dark terminal the effects fade from
)

func mix(a, b rgb, t float64) rgb {
	t = clamp(t)
	return rgb{a.r + (b.r-a.r)*t, a.g + (b.g-a.g)*t, a.b + (b.b-a.b)*t}
}

// Tone is how a header span is coloured.
type Tone int

const (
	Faint  Tone = iota // dim amber: titles, leaders
	Plain              // ivory: labels
	Bright             // the accent: results
)

func (t Tone) colour() rgb {
	switch t {
	case Plain:
		return colText
	case Bright:
		return colAccent
	}
	return colDim
}

// Span is a run of header text in one tone; a Line is a row of them.
type Span struct {
	Text string
	Tone Tone
}

type Line []Span

func (l Line) width() int {
	n := 0
	for _, s := range l {
		n += len([]rune(s.Text))
	}
	return n
}

// cell is one character cell: a rune in a colour, on an optional background.
type cell struct {
	r    rune
	fg   rgb
	bg   rgb
	hasB bool
}

// grid is a w x h screen.
type grid struct {
	w, h  int
	cells []cell
}

func newGrid(w, h int) *grid {
	g := &grid{w: w, h: h, cells: make([]cell, w*h)}
	for i := range g.cells {
		g.cells[i].r = ' '
	}
	return g
}

func (g *grid) set(x, y int, c cell) {
	if x >= 0 && y >= 0 && x < g.w && y < g.h {
		g.cells[y*g.w+x] = c
	}
}

func (g *grid) text(x, y int, s string, c rgb) {
	for i, r := range []rune(s) {
		g.set(x+i, y, cell{r: r, fg: c})
	}
}

// String renders the grid with 24-bit colour, codes only where they change.
// Trailing blanks are left out.
func (g *grid) String() string {
	var b strings.Builder
	for y := range g.h {
		row := g.cells[y*g.w : (y+1)*g.w]
		end := len(row)
		for end > 0 && row[end-1].r == ' ' && !row[end-1].hasB {
			end--
		}
		var fg, bg rgb
		hasFG, hasBG := false, false
		for _, c := range row[:end] {
			if c.r != ' ' && (!hasFG || c.fg != fg) {
				fmt.Fprintf(&b, "\x1b[38;2;%d;%d;%dm", int(c.fg.r), int(c.fg.g), int(c.fg.b))
				fg, hasFG = c.fg, true
			}
			switch {
			case c.hasB && (!hasBG || c.bg != bg):
				fmt.Fprintf(&b, "\x1b[48;2;%d;%d;%dm", int(c.bg.r), int(c.bg.g), int(c.bg.b))
				bg, hasBG = c.bg, true
			case !c.hasB && hasBG:
				b.WriteString("\x1b[49m")
				hasBG = false
			}
			b.WriteRune(c.r)
		}
		if hasFG || hasBG {
			b.WriteString("\x1b[m")
		}
		if y < g.h-1 {
			b.WriteByte('\n')
		}
	}
	return b.String()
}

// logoArt is Quarter's quarter_ wordmark, as in internal/ui/logo.go.
var logoArt = []string{
	"                                ██┐                          ",
	" ██████┐██┐  ██┐ █████┐  █████┐██████┐ █████┐  █████┐        ",
	"██┌──██│██│  ██│ └───██┐██┌───┘└██┌──┘██┌──██┐██┌───┘        ",
	"██│  ██│██│  ██│ ██████│██│     ██│   ███████│██│            ",
	"██│  ██│██│  ██│██┌──██│██│     ██│   ██┌────┘██│            ",
	"└██████│└██████│└██████│██│     └████┐└██████┐██│    ███████┐",
	" └───██│ └─────┘ └─────┘└─┘      └───┘ └─────┘└─┘    └──────┘",
	"     ██│                                                     ",
	"     └─┘                                                     ",
}

const (
	artPadX   = 15 // columns of padding inside the box, both sides together
	inkPad    = 2  // rows between the box's edge and the letters' solid part, above and below
	headerGap = 2  // rows between the header and the box
)

// coinGap is the rows from the box's bottom edge to "insert quarter": 3
// beside the big logo, 2 beside the normal one.
func coinGap(scale int) int { return 1 + scale }

// logoFont is the quarter_ wordmark's lowercase letters as pixels, one pixel
// per two-cell stroke of the logo (q u a r t e as drawn there; the rest
// built the same way: round letters lose a corner pixel, like the q and e).
// Rows: an ascender, five of x-height, two of descender.
var logoFont = map[rune][8]string{
	'q':  {"....", ".###", "#..#", "#..#", "#..#", ".###", "...#", "...#"},
	'u':  {"....", "#..#", "#..#", "#..#", "#..#", ".###", "....", "...."},
	'a':  {"....", "###.", "...#", ".###", "#..#", ".###", "....", "...."},
	'r':  {"....", ".###", "#...", "#...", "#...", "#...", "....", "...."},
	't':  {".#..", "####", ".#..", ".#..", ".#..", "..##", "....", "...."},
	'e':  {"....", ".##.", "#..#", "####", "#...", ".###", "....", "...."},
	'i':  {"#", ".", "#", "#", "#", "#", ".", "."},
	'n':  {"....", "###.", "#..#", "#..#", "#..#", "#..#", "....", "...."},
	's':  {"....", ".###", "#...", ".##.", "...#", "###.", "....", "...."},
	'c':  {"....", ".###", "#...", "#...", "#...", ".###", "....", "...."},
	'd':  {"...#", ".###", "#..#", "#..#", "#..#", ".###", "....", "...."},
	'f':  {"..##", ".#..", "####", ".#..", ".#..", ".#..", "....", "...."},
	'g':  {"....", ".###", "#..#", "#..#", "#..#", ".###", "...#", "###."},
	'h':  {"#...", "###.", "#..#", "#..#", "#..#", "#..#", "....", "...."},
	'k':  {"#...", "#..#", "#.#.", "##..", "#.#.", "#..#", "....", "...."},
	'l':  {"#", "#", "#", "#", "#", "#", ".", "."},
	'm':  {".....", "####.", "#.#.#", "#.#.#", "#.#.#", "#.#.#", ".....", "....."},
	'o':  {"....", ".##.", "#..#", "#..#", "#..#", ".##.", "....", "...."},
	'p':  {"....", "###.", "#..#", "#..#", "#..#", "###.", "#...", "#..."},
	'v':  {"....", "#..#", "#..#", "#..#", ".##.", ".##.", "....", "...."},
	'w':  {".....", "#...#", "#...#", "#.#.#", "#.#.#", ".#.#.", ".....", "....."},
	'x':  {"....", "#..#", "#..#", ".##.", "#..#", "#..#", "....", "...."},
	'y':  {"....", "#..#", "#..#", "#..#", "#..#", ".###", "...#", "###."},
	'?':  {".##.", "#..#", "..#.", ".#..", "....", ".#..", "....", "...."},
	'.':  {".", ".", ".", ".", ".", "#", ".", "."},
	'\'': {"#", "#", ".", ".", ".", ".", ".", "."},
}

// logoText draws lowercase text in logoFont at half the logo's size: each
// pixel a half block, so the letters keep the logo's proportions. A pixel
// between letters, three between words.
func logoText(text string) []string {
	px := make([][]bool, 8)
	for i, r := range text {
		for y := range px {
			switch {
			case r == ' ':
				px[y] = append(px[y], false, false, false)
			case i > 0 && text[i-1] != ' ':
				px[y] = append(px[y], false)
			}
			if r != ' ' {
				for _, c := range logoFont[r][y] {
					px[y] = append(px[y], c == '#')
				}
			}
		}
	}
	var out []string
	for y := 0; y < len(px); y += 2 {
		var b strings.Builder
		for x := range px[y] {
			switch top, bottom := px[y][x], px[y+1][x]; {
			case top && bottom:
				b.WriteRune('█')
			case top:
				b.WriteRune('▀')
			case bottom:
				b.WriteRune('▄')
			default:
				b.WriteRune(' ')
			}
		}
		out = append(out, b.String())
	}
	return out
}

// big2x maps each character of the art to its 2 x 2 cells at double size:
// letters stay solid; shadow lines keep to the top row and right column of
// their square, so they still meet their neighbours.
var big2x = map[rune][2]string{
	'█': {"██", "██"},
	'─': {"──", "  "},
	'│': {" │", " │"},
	'┌': {" ┌", " │"},
	'┐': {"─┐", " │"},
	'└': {" └", "  "},
	'┘': {"─┘", "  "},
	' ': {"  ", "  "},
}

// scaleArt draws the art at scale 1 or 2.
func scaleArt(rows []string, s int) []string {
	if s == 1 {
		return rows
	}
	var out []string
	for _, row := range rows {
		var top, bottom strings.Builder
		for _, r := range row {
			q := big2x[r]
			top.WriteString(q[0])
			bottom.WriteString(q[1])
		}
		out = append(out, top.String(), bottom.String())
	}
	return out
}

// inkRows is how many rows the letters' solid part spans at a scale: from
// the first art row with a block to the last (the t's top to the q's
// bottom); the rows after it are only shadow.
func inkRows(scale int) int {
	last := 0
	for y, row := range scaleArt(logoArt, scale) {
		if strings.ContainsRune(row, '█') {
			last = y
		}
	}
	return last + 1
}

// piece is one visible cell of the screen: where it goes, what it is, and
// whether it belongs to the logo's letters, its shadow or its box.
type piece struct {
	x, y int
	c    cell
	kind int
}

const (
	kindLetter = iota
	kindShadow
	kindBorder
)

// scene is the start screen for a w x h window, as the intros see it.
type scene struct {
	w, h    int
	scale   int // the logo's size: 1, or 2 on big windows
	bw, bh  int // the box, border included
	lx, ly  int // the box's top-left corner
	logo    []piece
	reading []piece // the logo, top to bottom, left to right
	at      map[[2]int]cell
	coin    []piece // insert quarter
	press   []piece // press any key
	header  []piece
}

// Screen is the start screen laid out for one window size and header.
type Screen struct{ s scene }

// NewScreen lays the start screen out in a w x h window. Above the logo, the
// title centred, a blank row, then the checks: their lines share a left
// edge, the block centred. ok is false when even the normal-size logo
// doesn't fit; the UI then shows its compact screen.
func NewScreen(w, h int, title string, checks []Line) (Screen, bool) {
	var header []Line
	if title != "" || len(checks) > 0 {
		header = append([]Line{{{Text: title, Tone: Faint}}, {}}, checks...)
	}
	artW := len([]rune(logoArt[0]))
	// The letters are centred in the box by their solid part, the t's top
	// to the q's bottom (the shadow line under it sits in the gap below).
	box := func(sc int) (int, int) { return artW*sc + 2 + artPadX + (sc - 1), 2 + inkPad + inkRows(sc) + inkPad }
	coinRows := func(sc int) int {
		if sc == 2 {
			return len(logoText("insert quarter"))
		}
		return 1
	}
	headH := 0
	if len(header) > 0 {
		headH = len(header) + headerGap
	}
	// The rows it all needs, press any key and a row under it included.
	need := func(sc int) (int, int) {
		bw, bh := box(sc)
		return bw + 2, headH + bh + coinGap(sc) + coinRows(sc) + 3
	}
	scale := 0
	for _, sc := range []int{2, 1} {
		if nw, nh := need(sc); w >= nw && h >= nh {
			scale = sc
			break
		}
	}
	if scale == 0 {
		return Screen{}, false
	}

	s := scene{w: w, h: h, scale: scale}
	s.bw, s.bh = box(scale)
	var coin []string
	if scale == 2 {
		coin = logoText("insert quarter")
	} else {
		coin = []string{"insert quarter"}
	}
	// The header, the logo and insert quarter are centred as one block;
	// press any key sits halfway between insert quarter and the bottom edge.
	total := headH + s.bh + coinGap(scale) + len(coin)
	top := max((h-total)/2, 0)
	s.lx, s.ly = (w-s.bw)/2, top+headH

	// The header: the title centred; the checks share a left edge, the block
	// centred.
	hw := 0
	for _, l := range checks {
		hw = max(hw, l.width())
	}
	for i, l := range header {
		x := (w - hw) / 2
		if i == 0 {
			x = (w - l.width()) / 2
		}
		for _, sp := range l {
			for _, r := range sp.Text {
				if r != ' ' {
					s.header = append(s.header, piece{x, top + i, cell{r: r, fg: sp.Tone.colour()}, kindLetter})
				}
				x++
			}
		}
	}

	// The box.
	for x := range s.bw {
		topC, bottomC := '─', '─'
		switch x {
		case 0:
			topC, bottomC = '╭', '╰'
		case s.bw - 1:
			topC, bottomC = '╮', '╯'
		}
		s.logo = append(s.logo,
			piece{s.lx + x, s.ly, cell{r: topC, fg: colAccent}, kindBorder},
			piece{s.lx + x, s.ly + s.bh - 1, cell{r: bottomC, fg: colAccent}, kindBorder})
	}
	for y := 1; y < s.bh-1; y++ {
		s.logo = append(s.logo,
			piece{s.lx, s.ly + y, cell{r: '│', fg: colAccent}, kindBorder},
			piece{s.lx + s.bw - 1, s.ly + y, cell{r: '│', fg: colAccent}, kindBorder})
	}
	// The wordmark, centred inside, letters in the gradient, shadow dim.
	art := scaleArt(logoArt, scale)
	ax, ay := s.lx+1+(s.bw-2-artW*scale)/2, s.ly+1+inkPad
	for y, row := range art {
		fg := mix(colTop, colBottom, float64(y)/float64(len(art)-1))
		for x, r := range []rune(row) {
			switch r {
			case ' ':
			case '█':
				s.logo = append(s.logo, piece{ax + x, ay + y, cell{r: r, fg: fg}, kindLetter})
			default:
				s.logo = append(s.logo, piece{ax + x, ay + y, cell{r: r, fg: colDim}, kindShadow})
			}
		}
	}
	s.reading = slices.Clone(s.logo)
	slices.SortFunc(s.reading, func(a, b piece) int {
		if a.y != b.y {
			return a.y - b.y
		}
		return a.x - b.x
	})
	s.at = make(map[[2]int]cell, len(s.logo))
	for _, p := range s.logo {
		s.at[[2]int{p.x, p.y}] = p.c
	}

	// insert quarter in the logo's gradient, and press any key, centred.
	line := func(text string, y int, c rgb) []piece {
		var out []piece
		x0 := (w - len([]rune(text))) / 2
		for x, r := range []rune(text) {
			if r != ' ' {
				out = append(out, piece{x0 + x, y, cell{r: r, fg: c}, kindLetter})
			}
		}
		return out
	}
	coinY := s.ly + s.bh + coinGap(scale)
	for i, row := range coin {
		fg := colAccent
		if len(coin) > 1 {
			fg = mix(colTop, colBottom, float64(i)/float64(len(coin)-1))
		}
		s.coin = append(s.coin, line(row, coinY+i, fg)...)
	}
	below := coinY + len(coin)
	s.press = line("press any key", below+(h-below)/2, colDim)
	return Screen{s}, true
}

// Rest is the start screen as it waits: everything, insert quarter lit or
// dark as it blinks.
func (sc Screen) Rest(coinLit bool) string {
	g := newGrid(sc.s.w, sc.s.h)
	for _, p := range sc.s.logo {
		g.set(p.x, p.y, p.c)
	}
	if coinLit {
		put(g, sc.s.coin)
	}
	sc.frame(g)
	return g.String()
}

// frame draws what every frame has, intro or rest: the header and press any
// key.
func (sc Screen) frame(g *grid) {
	put(g, sc.s.header)
	put(g, sc.s.press)
}

func put(g *grid, ps []piece) {
	for _, p := range ps {
		g.set(p.x, p.y, p.c)
	}
}

// Easing curves (Penner), t from 0 to 1.
func clamp(t float64) float64 { return math.Max(0, math.Min(1, t)) }

func outCubic(t float64) float64 { t = clamp(t); return 1 - math.Pow(1-t, 3) }

func inOutSine(t float64) float64 {
	t = clamp(t)
	return -(math.Cos(math.Pi*t) - 1) / 2
}

func outExpo(t float64) float64 {
	t = clamp(t)
	if t == 1 {
		return 1
	}
	return 1 - math.Pow(2, -10*t)
}

// outBack overshoots once, by about 10%, then settles.
func outBack(t float64) float64 {
	t = clamp(t)
	const c1 = 1.70158
	const c3 = c1 + 1
	return 1 + c3*math.Pow(t-1, 3) + c1*math.Pow(t-1, 2)
}

func outBounce(t float64) float64 {
	t = clamp(t)
	const n1, d1 = 7.5625, 2.75
	switch {
	case t < 1/d1:
		return n1 * t * t
	case t < 2/d1:
		t -= 1.5 / d1
		return n1*t*t + 0.75
	case t < 2.5/d1:
		t -= 2.25 / d1
		return n1*t*t + 0.9375
	default:
		t -= 2.625 / d1
		return n1*t*t + 0.984375
	}
}

// span maps t from [a, b] to [0, 1].
func span(t, a, b float64) float64 { return clamp((t - a) / (b - a)) }
