package art

import (
	"image"
	"image/color"
	"math"
)

// How a logo's shape is read and shrunk. Tuned on Pablo's 262 local Steam
// logos (Batch 8): heavy lettering with thin gaps (THE FINALS), dark logos
// with bright highlights (Cyberpunk 2077), emblems with small print (Elite
// Dangerous), hair-thin letters (Half-Life: Alyx).
const (
	visibleAlpha = 128  // a pixel at least half opaque is part of the shape
	thinShare    = 1.0  // a stroke thinner than this share of a dot's smaller side is thin
	thinBoost    = 2.0  // a thin stroke's pixels count this much toward a dot
	gapShare     = 1.0  // a gap or hole narrower than this share of a dot's width is kept open
	gapCut       = 0.15 // a dot with this share of kept-open gap stays off
	cover        = 0.45 // a dot is on when this much of it is covered
	missWeight   = 1.5  // for grid fitting: a lost bit of logo costs more than a stray one
)

// picture is an image reduced to its shape: how opaque each pixel is, and
// whether it counts as part of the logo.
//
// Which pixels count depends on the logo. Most are bright (156 of Pablo's
// 262 logos are over 90% bright): their near-black pixels are outlines and
// shadows, so they count as gaps, which keeps letters apart (Worms
// Revolution, APB Reloaded). Only near black: a dark coloured letter, like
// Z1 Battle Royale's rust-red Z, is still a letter. Some logos are near
// black with bright highlights (74% of Cyberpunk 2077's): there the dark
// part is the logo itself, so every opaque pixel counts. The terminal draws
// all of them in the gradient.
type picture struct {
	w, h  int
	alpha []uint8
	lit   []bool
}

const (
	darkCut     = 28  // brightness (0 to 255) below this is near black: an outline or shadow
	brightLogos = 0.6 // a logo with this share of bright opaque pixels is a bright one
)

func read(img image.Image) picture {
	b := img.Bounds()
	p := picture{w: b.Dx(), h: b.Dy()}
	p.alpha = make([]uint8, p.w*p.h)
	bright := make([]bool, p.w*p.h)
	put := func(i int, r, g, bl, a uint32) { // 0 to 255, not premultiplied
		p.alpha[i] = uint8(a)
		bright[i] = (299*r+587*g+114*bl)/1000 >= darkCut
	}
	if n, ok := img.(*image.NRGBA); ok { // what most transparent PNGs decode to
		for y := range p.h {
			row := n.Pix[(y+b.Min.Y-n.Rect.Min.Y)*n.Stride+(b.Min.X-n.Rect.Min.X)*4:]
			for x := range p.w {
				c := row[x*4 : x*4+4]
				put(y*p.w+x, uint32(c[0]), uint32(c[1]), uint32(c[2]), uint32(c[3]))
			}
		}
	} else {
		for y := range p.h {
			for x := range p.w {
				c := color.NRGBAModel.Convert(img.At(b.Min.X+x, b.Min.Y+y)).(color.NRGBA)
				put(y*p.w+x, uint32(c.R), uint32(c.G), uint32(c.B), uint32(c.A))
			}
		}
	}

	opaque, lit := 0, 0
	for i, a := range p.alpha {
		if a >= visibleAlpha {
			opaque++
			if bright[i] {
				lit++
			}
		}
	}
	useBright := float64(lit) >= brightLogos*float64(max(opaque, 1))
	p.lit = make([]bool, p.w*p.h)
	for i, a := range p.alpha {
		p.lit[i] = a >= visibleAlpha && (bright[i] || !useBright)
	}
	return p
}

// shape returns the box around the logo, or false when the image is a solid
// rectangle or shows almost nothing.
func (p picture) shape() (image.Rectangle, bool) {
	var box image.Rectangle
	lit, solid := 0, 0
	for y := range p.h {
		for x := range p.w {
			i := y*p.w + x
			if p.alpha[i] >= 250 {
				solid++
			}
			if p.lit[i] {
				lit++
				box = box.Union(image.Rect(x, y, x+1, y+1))
			}
		}
	}
	if lit < minVisible || float64(solid) >= solidShare*float64(len(p.alpha)) {
		return image.Rectangle{}, false
	}
	return box, true
}

// maxSourceWidth caps the pixels worked on: nearly every Steam logo is
// 640x360, but one of Pablo's is 4731 wide and took 250 ms to draw. 1024 is
// still far more than a logo band's dots.
const maxSourceWidth = 1024

// shrinkLarge returns img shrunk by a whole factor to at most
// maxSourceWidth wide, averaging each block of pixels, or img itself when
// it's small enough.
func shrinkLarge(img image.Image) image.Image {
	b := img.Bounds()
	f := (b.Dx() + maxSourceWidth - 1) / maxSourceWidth
	if f <= 1 {
		return img
	}
	out := image.NewNRGBA(image.Rect(0, 0, b.Dx()/f, b.Dy()/f))
	src, fast := img.(*image.NRGBA) // what most transparent PNGs decode to
	for y := range out.Rect.Dy() {
		for x := range out.Rect.Dx() {
			var r, g, bl, a uint64 // premultiplied, so transparent pixels add no colour
			for sy := range f {
				for sx := range f {
					px, py := b.Min.X+x*f+sx, b.Min.Y+y*f+sy
					if fast {
						c := src.Pix[src.PixOffset(px, py):]
						ca := uint64(c[3])
						r, g, bl, a = r+uint64(c[0])*ca, g+uint64(c[1])*ca, bl+uint64(c[2])*ca, a+ca*255
						continue
					}
					cr, cg, cb, ca := img.At(px, py).RGBA()
					r, g, bl, a = r+uint64(cr>>8)*255, g+uint64(cg>>8)*255, bl+uint64(cb>>8)*255, a+uint64(ca>>8)*255
				}
			}
			// Back from premultiplied sums to plain 8-bit colour.
			if a == 0 {
				continue
			}
			i := out.PixOffset(x, y)
			out.Pix[i], out.Pix[i+1], out.Pix[i+2] = uint8(r*255/a), uint8(g*255/a), uint8(bl*255/a)
			out.Pix[i+3] = uint8(a / uint64(f*f) / 255)
		}
	}
	return out
}

// cropMargin keeps some empty space around the logo, in pixels, so strokes
// and gaps at its edge measure the same as anywhere else.
const cropMargin = 8

// sub returns the part of p inside r (clipped to p), and the logo's box in
// the new picture's coordinates.
func (p picture) sub(r image.Rectangle) (picture, image.Rectangle) {
	r = r.Intersect(image.Rect(0, 0, p.w, p.h))
	q := picture{w: r.Dx(), h: r.Dy()}
	q.alpha = make([]uint8, q.w*q.h)
	q.lit = make([]bool, q.w*q.h)
	for y := range q.h {
		src := (r.Min.Y+y)*p.w + r.Min.X
		copy(q.alpha[y*q.w:(y+1)*q.w], p.alpha[src:src+q.w])
		copy(q.lit[y*q.w:(y+1)*q.w], p.lit[src:src+q.w])
	}
	crop, _ := q.shape()
	return q, crop
}

// sample shrinks the crop to pw x ph dots, judging each part of the logo by
// its own shape rather than one rule for the whole logo:
//   - thin strokes (thinner than a dot) count thinBoost times, so a line
//     crossing a dot turns it on instead of vanishing;
//   - narrow gaps and small holes (narrower than a dot: between heavy
//     letters, inside an O or an A) keep the dots they cross off, so letters
//     stay apart and counters stay open. Outer edges are untouched.
//
// Then the dot grid is fitted, as font renderers do for sharp text: shifted
// by quarters of a dot to where the logo's edges land on whole dots.
func (p picture) sample(crop image.Rectangle, pw, ph int) []bool {
	dw := float64(crop.Dx()) / float64(pw) // source pixels per dot
	dh := float64(crop.Dy()) / float64(ph)

	// Dark specks smaller than a dot are texture (grunge, film grain), not
	// outlines: they count as logo, or they'd shatter the letter they're in.
	lit := fillSpecks(p.lit, p.alpha, p.w, int(dw*dh))

	// Big filled areas (banners, plates, discs) become a clean outline one
	// dot wide instead of a painted block; letters are thinner and stay solid.
	lit = hollowFills(lit, p.w, p.h, dw, dh)

	// A pixel's stroke is thin when no pixel near it lies deeper inside.
	dist := distances(lit, p.w, p.h)
	depth := maxFilter(dist, p.w, p.h, int(math.Ceil(min(dw, dh))))
	thinLimit := thinShare * min(dw, dh) / 2 // half of a thin stroke's width
	thick := make([]bool, len(lit))
	thin := make([]bool, len(lit))
	for i, on := range lit {
		if on {
			thin[i] = float64(depth[i]) < thinLimit
			thick[i] = !thin[i]
		}
	}

	gap := narrowGaps(lit, p.w, p.h, int(math.Ceil(gapShare*dw/2)))

	thickT, thinT, gapT, real := summed(thick, p.w, p.h), summed(thin, p.w, p.h), summed(gap, p.w, p.h), summed(lit, p.w, p.h)
	total := real.count(crop.Min.X, crop.Min.Y, crop.Max.X, crop.Max.Y)
	var best []bool
	bestErr := math.Inf(1)
	for _, fy := range fitShifts {
		for _, fx := range fitShifts {
			ox, oy := float64(crop.Min.X)+fx*dw, float64(crop.Min.Y)+fy*dh
			on, miss := grid(thickT, thinT, gapT, real, ox, oy, dw, dh, pw, ph)
			// What a shift pushes outside the grid is lost too.
			outside := total - real.count(round(ox), round(oy), round(ox+dw*float64(pw)), round(oy+dh*float64(ph)))
			if e := miss + missWeight*float64(outside); e < bestErr {
				best, bestErr = on, e
			}
		}
	}
	return best
}

// Which filled areas become outlines. Measured on Pablo's 12-logo test set
// (Batch 8): the fills to outline, Conan Exiles' banner and ShareX's disc,
// average 12.6 and 13.3 dots thick; the heaviest lettering, Worms
// Revolution's, 11.2. So a fill is a piece that is a big share of the logo
// and averages at least fillMinDots thick.
const (
	fillMinShare = 0.2  // of all the logo's pixels, in one connected piece
	fillMinDots  = 12.0 // average thickness: twice the area over the edge length
	outlineDots  = 1.2  // the rim kept, across and down
)

// hollowFills empties the inside of every fill, leaving its rim: a pixel
// goes when it lies in a fill and is more than a rim's width from the edge
// both across and down. Dots are dw x dh source pixels.
func hollowFills(lit []bool, w, h int, dw, dh float64) []bool {
	fill := fills(lit, w, fillMinShare, fillMinDots*dw)
	across, down := runs(lit, w, h)
	out := append([]bool(nil), lit...)
	for i, on := range lit {
		if on && fill[i] && float64(across[i]) > outlineDots*dw && float64(down[i]) > outlineDots*dh {
			out[i] = false
		}
	}
	return out
}

// fills marks the pixels of every connected piece (4-connected) that holds
// at least share of all lit pixels and averages at least minThick pixels
// thick. A letter, however heavy, is one of many similar pieces; a banner
// or a disc is one big, thick piece.
func fills(lit []bool, w int, share, minThick float64) []bool {
	total := 0
	for _, on := range lit {
		if on {
			total++
		}
	}
	label := make([]int32, len(lit))
	var sizes, edges []int
	sizes, edges = append(sizes, 0), append(edges, 0)
	stack := make([]int32, 0, 1024)
	for start, on := range lit {
		if !on || label[start] != 0 {
			continue
		}
		id := int32(len(sizes))
		size, edge := 0, 0
		label[start] = id
		stack = append(stack[:0], int32(start))
		for len(stack) > 0 {
			i := int(stack[len(stack)-1])
			stack = stack[:len(stack)-1]
			size++
			x := i % w
			onEdge := false
			for _, j := range [4]int{i - 1, i + 1, i - w, i + w} {
				switch {
				case j < 0 || j >= len(lit) || (j == i-1 && x == 0) || (j == i+1 && x == w-1):
					onEdge = true // the image's edge
				case !lit[j]:
					onEdge = true
				case label[j] == 0:
					label[j] = id
					stack = append(stack, int32(j))
				}
			}
			if onEdge {
				edge++
			}
		}
		sizes, edges = append(sizes, size), append(edges, edge)
	}
	isFill := make([]bool, len(sizes))
	for id := 1; id < len(sizes); id++ {
		thick := 2 * float64(sizes[id]) / float64(max(edges[id], 1))
		isFill[id] = float64(sizes[id]) >= share*float64(total) && thick >= minThick
	}
	out := make([]bool, len(lit))
	for i, l := range label {
		out[i] = isFill[l]
	}
	return out
}

// runs is, for each lit pixel, how far the nearest unlit pixel is along its
// row (across) and along its column (down).
func runs(lit []bool, w, h int) (across, down []int32) {
	across, down = make([]int32, len(lit)), make([]int32, len(lit))
	line := func(out []int32, n, stride, base int) {
		d := int32(0) // distance since the last unlit pixel, or the start
		for i := range n {
			if j := base + i*stride; lit[j] {
				d++
				out[j] = d
			} else {
				d = 0
			}
		}
		d = 0
		for i := n - 1; i >= 0; i-- {
			if j := base + i*stride; lit[j] {
				d++
				out[j] = min(out[j], d)
			} else {
				d = 0
			}
		}
	}
	for y := range h {
		line(across, w, 1, y*w)
	}
	for x := range w {
		line(down, h, w, x)
	}
	return across, down
}

// fillSpecks returns lit with every opaque dark patch smaller than maxArea
// pixels filled in. Outlines are long connected strokes and stay gaps.
func fillSpecks(lit []bool, alpha []uint8, w, maxArea int) []bool {
	out := append([]bool(nil), lit...)
	dark := func(i int) bool { return !lit[i] && alpha[i] >= visibleAlpha }
	seen := make([]bool, len(lit))
	var stack, patch []int
	for start := range lit {
		if seen[start] || !dark(start) {
			continue
		}
		seen[start] = true
		stack, patch = append(stack[:0], start), patch[:0]
		size := 0
		for len(stack) > 0 { // walk the whole patch, so no part of it is taken for a speck later
			i := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			if size++; size < maxArea {
				patch = append(patch, i)
			}
			x := i % w
			for _, j := range [4]int{i - 1, i + 1, i - w, i + w} {
				switch {
				case j < 0 || j >= len(lit) || seen[j] || !dark(j):
				case j == i-1 && x == 0, j == i+1 && x == w-1: // across the image's edge
				default:
					seen[j] = true
					stack = append(stack, j)
				}
			}
		}
		if size < maxArea {
			for _, i := range patch {
				out[i] = true
			}
		}
	}
	return out
}

// fitShifts are the grid shifts tried, in dots.
var fitShifts = []float64{-0.25, 0, 0.25, 0.5}

// grid samples pw x ph dots of dw x dh source pixels from (ox, oy). A dot is
// on when its thick pixels plus its thin ones, boosted, cover enough of it,
// and it doesn't cross a narrow gap. miss scores the fit against the real
// shape: stray pixels count once, lost ones missWeight times.
func grid(thick, thin, gap, real table, ox, oy, dw, dh float64, pw, ph int) (on []bool, miss float64) {
	on = make([]bool, pw*ph)
	for y := range ph {
		y0, y1 := round(oy+float64(y)*dh), round(oy+float64(y+1)*dh)
		for x := range pw {
			x0, x1 := round(ox+float64(x)*dw), round(ox+float64(x+1)*dw)
			area := float64(max((x1-x0)*(y1-y0), 1))
			seen := float64(thick.count(x0, y0, x1, y1)) + thinBoost*float64(thin.count(x0, y0, x1, y1))
			n := float64(real.count(x0, y0, x1, y1))
			if seen >= cover*area && float64(gap.count(x0, y0, x1, y1)) < gapCut*area {
				on[y*pw+x] = true
				miss += max(area-n, 0)
			} else {
				miss += missWeight * n
			}
		}
	}
	return on, miss
}

// narrowGaps finds the empty pixels a closing of radius r fills: gaps
// between strokes and holes inside them no wider than about 2r.
func narrowGaps(lit []bool, w, h, r int) []bool {
	empty := make([]bool, len(lit))
	for i, on := range lit {
		empty[i] = !on
	}
	// Grow the shape by r, then shrink the grown shape by r: what's added
	// and not taken back was a narrow gap or a small hole.
	toShape := distances(empty, w, h) // for empty pixels: how far to the shape
	grown := make([]bool, len(lit))
	for i, d := range toShape {
		grown[i] = lit[i] || d <= float32(r)
	}
	inGrown := distances(grown, w, h)
	gap := make([]bool, len(lit))
	for i, d := range inGrown {
		gap[i] = !lit[i] && d > float32(r)
	}
	return gap
}

// distances is, for each lit pixel, how far it is from the nearest unlit
// one, in pixels, measured square (chessboard distance): thinning by it keeps
// square corners square, where a round measure would bevel them.
func distances(lit []bool, w, h int) []float32 {
	const big = math.MaxInt32 / 2
	d := make([]int32, len(lit))
	for i, on := range lit {
		if on {
			d[i] = big
		}
	}
	at := func(x, y int) int32 {
		if x < 0 || y < 0 || x >= w || y >= h {
			return 0 // outside the image is background
		}
		return d[y*w+x]
	}
	for y := range h {
		for x := range w {
			if i := y*w + x; d[i] != 0 {
				d[i] = min(d[i], at(x-1, y)+1, at(x, y-1)+1, at(x-1, y-1)+1, at(x+1, y-1)+1)
			}
		}
	}
	for y := h - 1; y >= 0; y-- {
		for x := w - 1; x >= 0; x-- {
			if i := y*w + x; d[i] != 0 {
				d[i] = min(d[i], at(x+1, y)+1, at(x, y+1)+1, at(x+1, y+1)+1, at(x-1, y+1)+1)
			}
		}
	}
	out := make([]float32, len(d))
	for i, v := range d {
		out[i] = float32(v)
	}
	return out
}

// maxFilter is, for each pixel, the largest value within r pixels (a
// square window), in two passes. Each line uses van Herk/Gil-Werman: running
// maxima forwards and backwards within blocks of the window's size, so any
// window's maximum is two lookups, whatever r is.
func maxFilter(v []float32, w, h, r int) []float32 {
	k := 2*r + 1
	pass := func(src []float32, n, stride, lines, step int) []float32 {
		out := make([]float32, len(src))
		// Each line is padded with r zeros on both sides (every value is at
		// least 0), so every window is exactly k long.
		m := n + 2*r
		line, fwd, bwd := make([]float32, m), make([]float32, m), make([]float32, m)
		for l := range lines {
			base := l * step
			for i := range n {
				line[r+i] = src[base+i*stride]
			}
			for i := range m {
				if i%k == 0 {
					fwd[i] = line[i]
				} else {
					fwd[i] = max(fwd[i-1], line[i])
				}
			}
			for i := m - 1; i >= 0; i-- {
				if i == m-1 || (i+1)%k == 0 {
					bwd[i] = line[i]
				} else {
					bwd[i] = max(bwd[i+1], line[i])
				}
			}
			for i := range n {
				// Pixel i's window is line[i : i+k]: the tail of one block
				// (bwd from its start) and the head of the next (fwd to its end).
				out[base+i*stride] = max(bwd[i], fwd[i+k-1])
			}
		}
		return out
	}
	return pass(pass(v, w, 1, h, w), h, w, w, 1)
}

// table is a summed-area table: any rectangle's lit count in four lookups.
type table struct {
	w, h int
	sum  []int32
}

func summed(on []bool, w, h int) table {
	t := table{w, h, make([]int32, (w+1)*(h+1))}
	for y := range h {
		var row int32
		for x := range w {
			if on[y*w+x] {
				row++
			}
			t.sum[(y+1)*(w+1)+x+1] = t.sum[y*(w+1)+x+1] + row
		}
	}
	return t
}

// count is how many pixels are lit in [x0,x1) x [y0,y1), clipped to the image.
func (t table) count(x0, y0, x1, y1 int) int {
	x0, y0 = max(x0, 0), max(y0, 0)
	x1, y1 = min(x1, t.w), min(y1, t.h)
	if x1 <= x0 || y1 <= y0 {
		return 0
	}
	W := t.w + 1
	return int(t.sum[y1*W+x1] - t.sum[y0*W+x1] - t.sum[y1*W+x0] + t.sum[y0*W+x0])
}

func round(v float64) int { return int(math.Floor(v + 0.5)) }
