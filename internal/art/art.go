// Package art redraws an image's shape as text. Each character cell holds a
// small grid of dots, and what shows of the image on a dark terminal decides
// which are on.
package art

import (
	"errors"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
)

// Blocks picks the characters a picture is drawn with.
type Blocks int

const (
	// Quadrants draw 2x2 dots per cell with the block elements (U+2580 to
	// U+259F), which every common monospace font has. The default.
	Quadrants Blocks = iota
	// Eighths draw 2x4 dots per cell with Unicode 16's block octants: twice
	// the detail, but only some fonts have them (JetBrains Mono and Cascadia
	// Mono don't; Cascadia Code does). Chosen in config.json.
	Eighths
)

// quadrantRunes maps each 2x2 dot pattern (bit i is dot i+1, dots numbered
// 1 2 / 3 4) to its block element.
var quadrantRunes = []rune(" ▘▝▀▖▌▞▛▗▚▐▜▄▙▟█")

// ParseBlocks reads a Blocks setting; empty means the default, Quadrants.
func ParseBlocks(s string) (Blocks, bool) {
	switch s {
	case "", "quadrants":
		return Quadrants, true
	case "octants", "eighths":
		return Eighths, true
	}
	return Quadrants, false
}

// Key is the setting's saved name: empty for the default.
func (b Blocks) Key() string {
	if b == Eighths {
		return "octants"
	}
	return ""
}

// ErrUnusable means the image has no shape worth drawing: no transparency
// (a solid rectangle), or almost nothing visible.
var ErrUnusable = errors.New("art: image has no usable shape")

const (
	solidShare = 0.97 // an image this opaque is a rectangle, not a shape
	minVisible = 16   // fewer visible pixels than this is nothing to draw
)

// Render draws img's shape in at most cols x rows cells, keeping its
// proportions (a cell is about twice as tall as it is wide). The shape is
// what shows on the terminal's dark background (see shape.go), cropped to
// its visible part. Every row has the same width, so the result can be
// centered as a block.
func Render(img image.Image, cols, rows int, b Blocks) ([]string, error) {
	if cols < 1 || rows < 1 {
		return nil, ErrUnusable
	}
	p := read(shrinkLarge(img))
	crop, ok := p.shape()
	if !ok {
		return nil, ErrUnusable
	}
	// Work on the logo and a small margin only, not the empty space around it.
	p, crop = p.sub(crop.Inset(-cropMargin))

	dx, dy := 2, 2
	if b == Eighths {
		dy = 4
	}
	c, r := fit(crop.Dx(), crop.Dy(), cols, rows)
	on := p.sample(crop, c*dx, r*dy)

	lines := make([][]rune, r)
	for y := range r {
		lines[y] = make([]rune, c)
		for x := range c {
			n := 0
			for i := range dx * dy {
				if on[(y*dy+i/dx)*c*dx+x*dx+i%dx] {
					n |= 1 << i
				}
			}
			if b == Eighths {
				lines[y][x] = octants[n]
			} else {
				lines[y][x] = quadrantRunes[n]
			}
		}
	}
	return trim(lines), nil
}

// RenderFile reads a PNG and draws it as Render does.
func RenderFile(path string, cols, rows int, b Blocks) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return Render(img, cols, rows, b)
}

// fit picks the cell grid for a w x h shape: as wide as cols allows, unless
// that makes it taller than rows.
func fit(w, h, cols, rows int) (c, r int) {
	r = (cols*h + w) / (2 * w) // rounded cols*h/(2w)
	if r <= rows {
		return cols, max(r, 1)
	}
	c = (rows*2*w + h/2) / h
	return max(min(c, cols), 1), rows
}

// trim drops blank rows at the top and bottom and blank columns at the sides,
// the same columns from every row so the rows stay aligned.
func trim(lines [][]rune) []string {
	blankRow := func(l []rune) bool {
		for _, r := range l {
			if r != ' ' {
				return false
			}
		}
		return true
	}
	for len(lines) > 0 && blankRow(lines[0]) {
		lines = lines[1:]
	}
	for len(lines) > 0 && blankRow(lines[len(lines)-1]) {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return nil
	}
	blankCol := func(x int) bool {
		for _, l := range lines {
			if l[x] != ' ' {
				return false
			}
		}
		return true
	}
	left, right := 0, len(lines[0])
	for left < right && blankCol(left) {
		left++
	}
	for right > left && blankCol(right-1) {
		right--
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		out[i] = string(l[left:right])
	}
	return out
}
