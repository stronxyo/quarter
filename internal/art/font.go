package art

import (
	"strings"
	"unicode"
)

// glyphs is Quarter's own pixel font: capitals, digits and common
// punctuation, 7 pixels tall, 5 wide (narrow marks less). A game with no
// logo anywhere shows its name in it, in the same block style as logos.
var glyphs = map[rune][7]string{
	'A':  {".###.", "#...#", "#...#", "#####", "#...#", "#...#", "#...#"},
	'B':  {"####.", "#...#", "#...#", "####.", "#...#", "#...#", "####."},
	'C':  {".###.", "#...#", "#....", "#....", "#....", "#...#", ".###."},
	'D':  {"####.", "#...#", "#...#", "#...#", "#...#", "#...#", "####."},
	'E':  {"#####", "#....", "#....", "####.", "#....", "#....", "#####"},
	'F':  {"#####", "#....", "#....", "####.", "#....", "#....", "#...."},
	'G':  {".###.", "#...#", "#....", "#.###", "#...#", "#...#", ".####"},
	'H':  {"#...#", "#...#", "#...#", "#####", "#...#", "#...#", "#...#"},
	'I':  {"###", ".#.", ".#.", ".#.", ".#.", ".#.", "###"},
	'J':  {"..###", "...#.", "...#.", "...#.", "...#.", "#..#.", ".##.."},
	'K':  {"#...#", "#..#.", "#.#..", "##...", "#.#..", "#..#.", "#...#"},
	'L':  {"#....", "#....", "#....", "#....", "#....", "#....", "#####"},
	'M':  {"#...#", "##.##", "#.#.#", "#.#.#", "#...#", "#...#", "#...#"},
	'N':  {"#...#", "#...#", "##..#", "#.#.#", "#..##", "#...#", "#...#"},
	'O':  {".###.", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
	'P':  {"####.", "#...#", "#...#", "####.", "#....", "#....", "#...."},
	'Q':  {".###.", "#...#", "#...#", "#...#", "#.#.#", "#..#.", ".##.#"},
	'R':  {"####.", "#...#", "#...#", "####.", "#.#..", "#..#.", "#...#"},
	'S':  {".####", "#....", "#....", ".###.", "....#", "....#", "####."},
	'T':  {"#####", "..#..", "..#..", "..#..", "..#..", "..#..", "..#.."},
	'U':  {"#...#", "#...#", "#...#", "#...#", "#...#", "#...#", ".###."},
	'V':  {"#...#", "#...#", "#...#", "#...#", "#...#", ".#.#.", "..#.."},
	'W':  {"#...#", "#...#", "#...#", "#.#.#", "#.#.#", "#.#.#", ".#.#."},
	'X':  {"#...#", "#...#", ".#.#.", "..#..", ".#.#.", "#...#", "#...#"},
	'Y':  {"#...#", "#...#", ".#.#.", "..#..", "..#..", "..#..", "..#.."},
	'Z':  {"#####", "....#", "...#.", "..#..", ".#...", "#....", "#####"},
	'0':  {".###.", "#...#", "#..##", "#.#.#", "##..#", "#...#", ".###."},
	'1':  {"..#..", ".##..", "..#..", "..#..", "..#..", "..#..", ".###."},
	'2':  {".###.", "#...#", "....#", "...#.", "..#..", ".#...", "#####"},
	'3':  {"####.", "....#", "....#", ".###.", "....#", "....#", "####."},
	'4':  {"...#.", "..##.", ".#.#.", "#..#.", "#####", "...#.", "...#."},
	'5':  {"#####", "#....", "####.", "....#", "....#", "#...#", ".###."},
	'6':  {".###.", "#....", "#....", "####.", "#...#", "#...#", ".###."},
	'7':  {"#####", "....#", "...#.", "..#..", ".#...", ".#...", ".#..."},
	'8':  {".###.", "#...#", "#...#", ".###.", "#...#", "#...#", ".###."},
	'9':  {".###.", "#...#", "#...#", ".####", "....#", "....#", ".###."},
	':':  {".", "#", "#", ".", "#", "#", "."},
	'.':  {".", ".", ".", ".", ".", ".", "#"},
	',':  {"..", "..", "..", "..", "..", ".#", "#."},
	'\'': {"#", "#", ".", ".", ".", ".", "."},
	'!':  {"#", "#", "#", "#", "#", ".", "#"},
	'?':  {".###.", "#...#", "....#", "...#.", "..#..", ".....", "..#.."},
	'-':  {"....", "....", "....", "####", "....", "....", "...."},
	'+':  {".....", "..#..", "..#..", "#####", "..#..", "..#..", "....."},
	'/':  {"....#", "...#.", "...#.", "..#..", ".#...", ".#...", "#...."},
	'&':  {".##..", "#..#.", "#.#..", ".#...", "#.#.#", "#..#.", ".##.#"},
	'(':  {"..#", ".#.", "#..", "#..", "#..", ".#.", "..#"},
	')':  {"#..", ".#.", "..#", "..#", "..#", ".#.", "#.."},
	'_':  {".....", ".....", ".....", ".....", ".....", ".....", "#####"},
}

// accents maps accented capitals to the letters the font has.
var accents = map[rune]rune{
	'À': 'A', 'Á': 'A', 'Â': 'A', 'Ã': 'A', 'Ä': 'A', 'Å': 'A', 'Ç': 'C',
	'È': 'E', 'É': 'E', 'Ê': 'E', 'Ë': 'E', 'Ì': 'I', 'Í': 'I', 'Î': 'I', 'Ï': 'I',
	'Ñ': 'N', 'Ò': 'O', 'Ó': 'O', 'Ô': 'O', 'Õ': 'O', 'Ö': 'O', 'Ø': 'O',
	'Ù': 'U', 'Ú': 'U', 'Û': 'U', 'Ü': 'U', 'Ý': 'Y',
}

const (
	glyphRows = 7
	spaceCols = 3 // a space is narrower than a letter
)

// Name draws text in Quarter's pixel font as half blocks (two pixel rows per
// cell), wrapped at spaces, in at most cols x rows cells. Characters the font
// lacks are left out. ok is false when it doesn't fit, or nothing is left.
func Name(text string, cols, rows int) ([]string, bool) {
	words := fontWords(text)
	if len(words) == 0 {
		return nil, false
	}
	var lines [][]string
	var line []string
	width := func(ws []string) int {
		w := 0
		for i, word := range ws {
			if i > 0 {
				w += spaceCols + 2 // a letter gap on each side of the space
			}
			w += wordCols(word)
		}
		return w
	}
	for _, word := range words {
		if wordCols(word) > cols {
			return nil, false // a word too long for the pane
		}
		if try := append(append([]string{}, line...), word); width(try) <= cols {
			line = try
			continue
		}
		lines = append(lines, line)
		line = []string{word}
	}
	lines = append(lines, line)

	// One blank pixel row between lines; two pixel rows per cell.
	px := len(lines)*(glyphRows+1) - 1
	if (px+1)/2 > rows {
		return nil, false
	}
	var out []string
	for i, l := range lines {
		bits := pixels(strings.Join(l, " "))
		if i > 0 {
			bits = append([][]bool{make([]bool, len(bits[0]))}, bits...) // the gap row
		}
		out = append(out, halfBlocks(bits)...)
	}
	return out, true
}

// fontWords uppercases text, maps accents, drops what the font can't draw,
// and splits it into words.
func fontWords(text string) []string {
	var b strings.Builder
	for _, r := range strings.ToUpper(text) {
		if a, ok := accents[r]; ok {
			r = a
		}
		if _, ok := glyphs[r]; ok || unicode.IsSpace(r) {
			b.WriteRune(r)
		}
	}
	return strings.Fields(b.String())
}

func wordCols(word string) int {
	w := 0
	for i, r := range word {
		if i > 0 {
			w++ // the gap between letters
		}
		w += len(glyphs[r][0])
	}
	return w
}

// pixels lays a line of text out as rows of pixels.
func pixels(line string) [][]bool {
	rows := make([][]bool, glyphRows)
	for i, r := range []rune(line) {
		for y := range glyphRows {
			if i > 0 {
				rows[y] = append(rows[y], false)
			}
			if r == ' ' {
				rows[y] = append(rows[y], make([]bool, spaceCols)...)
				continue
			}
			for _, c := range glyphs[r][y] {
				rows[y] = append(rows[y], c == '#')
			}
		}
	}
	return rows
}

// halfBlocks packs two pixel rows into each row of characters.
func halfBlocks(bits [][]bool) []string {
	var out []string
	for y := 0; y < len(bits); y += 2 {
		var b strings.Builder
		for x := range bits[y] {
			top, bottom := bits[y][x], y+1 < len(bits) && bits[y+1][x]
			switch {
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
