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
	visibleAlpha = 128  // a pixel at least half opaque is part of the shape
	solidShare   = 0.97 // an image this opaque is a rectangle, not a shape
	minVisible   = 16   // fewer visible pixels than this is nothing to draw
)

// Render draws img's shape in at most cols x rows cells, keeping its
// proportions (a cell is about twice as tall as it is wide). The shape is
// cropped to its visible part first. Every row has the same width, so the
// result can be centered as a block.
func Render(img image.Image, cols, rows int, b Blocks) ([]string, error) {
	if cols < 1 || rows < 1 {
		return nil, ErrUnusable
	}
	a := alphaOf(img)
	crop, ok := a.shape()
	if !ok {
		return nil, ErrUnusable
	}

	dx, dy := 2, 4
	if b == Sixths {
		dy = 3
	}
	c, r := fit(crop.Dx(), crop.Dy(), cols, rows)
	on := a.sample(crop, c*dx, r*dy)

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

// alpha is an image's opacity, one byte per pixel.
type alpha struct {
	w, h int
	pix  []uint8
}

func alphaOf(img image.Image) alpha {
	bounds := img.Bounds()
	a := alpha{bounds.Dx(), bounds.Dy(), make([]uint8, bounds.Dx()*bounds.Dy())}
	if n, ok := img.(*image.NRGBA); ok { // what most transparent PNGs decode to
		for y := range a.h {
			row := n.Pix[(y+bounds.Min.Y-n.Rect.Min.Y)*n.Stride+(bounds.Min.X-n.Rect.Min.X)*4:]
			for x := range a.w {
				a.pix[y*a.w+x] = row[x*4+3]
			}
		}
		return a
	}
	for y := range a.h {
		for x := range a.w {
			_, _, _, v := img.At(bounds.Min.X+x, bounds.Min.Y+y).RGBA()
			a.pix[y*a.w+x] = uint8(v >> 8)
		}
	}
	return a
}

// shape returns the box around the visible pixels, or false when the image
// is a solid rectangle or shows almost nothing.
func (a alpha) shape() (image.Rectangle, bool) {
	var box image.Rectangle
	visible, solid := 0, 0
	for y := range a.h {
		for x := range a.w {
			v := a.pix[y*a.w+x]
			if v >= 250 {
				solid++
			}
			if v >= visibleAlpha {
				visible++
				box = box.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	if visible < minVisible || float64(solid) >= solidShare*float64(len(a.pix)) {
		return image.Rectangle{}, false
	}
	return box, true
}

// sample shrinks the crop to pw x ph dots. A dot is on when the pixels it
// covers are, on average, at least half opaque.
func (a alpha) sample(crop image.Rectangle, pw, ph int) []bool {
	on := make([]bool, pw*ph)
	w, h := crop.Dx(), crop.Dy()
	for y := range ph {
		y0 := crop.Min.Y + y*h/ph
		y1 := max(crop.Min.Y+(y+1)*h/ph, y0+1)
		for x := range pw {
			x0 := crop.Min.X + x*w/pw
			x1 := max(crop.Min.X+(x+1)*w/pw, x0+1)
			sum := 0
			for sy := y0; sy < y1; sy++ {
				for _, v := range a.pix[sy*a.w+x0 : sy*a.w+x1] {
					sum += int(v)
				}
			}
			on[y*pw+x] = 2*sum >= 255*(y1-y0)*(x1-x0)
		}
	}
	return on
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
