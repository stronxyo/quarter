// Package art redraws an image's shape as text. Each character cell holds a
// small grid of dots, and the image's transparency decides which are on.
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
	Eighths Blocks = iota // 2x4 dots per cell: Unicode 16 block octants
	Sixths                // 2x3 dots per cell, for fonts without octants
)

// ParseBlocks reads a Blocks setting; empty means the default, Eighths.
func ParseBlocks(s string) (Blocks, bool) {
	switch s {
	case "", "eighths":
		return Eighths, true
	case "sixths":
		return Sixths, true
	}
	return Eighths, false
}

// Key is the setting's saved name: empty for the default.
func (b Blocks) Key() string {
	if b == Sixths {
		return "sixths"
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
	p := read(img)
	crop, ok := p.shape()
	if !ok {
		return nil, ErrUnusable
	}

	dx, dy := 2, 4
	if b == Sixths {
		dy = 3
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
			if b == Sixths {
				lines[y][x] = sextant(n)
			} else {
				lines[y][x] = octants[n]
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
