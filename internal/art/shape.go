package art

import (
	"image"
	"math"
)

// How a logo's shape is read, tuned on real Steam logos (Batch 7): white
// letters with black outlines, thin brush strokes, emblems with small print.
const (
	darkCut    = 60   // brightness on black below this is background, outlines included
	erodeShare = 0.35 // erosion, as a share of the source pixels one dot covers
	coverThin  = 0.25 // a dot this covered is on, for strokes thinner than a dot
	coverThick = 0.30 // the same for thicker strokes
	restoreCut = 4    // a piece erosion kept less than 1/restoreCut of is put back
)

// picture is an image reduced to what matters on a dark terminal: how opaque
// each pixel is, and whether it shows against the dark at all.
type picture struct {
	w, h  int
	alpha []uint8
	lit   []bool
}

// read measures each pixel as it looks on black: opacity times brightness.
// Dark outlines and shadows count as background, so letters stay apart.
func read(img image.Image) picture {
	b := img.Bounds()
	p := picture{w: b.Dx(), h: b.Dy()}
	p.alpha = make([]uint8, p.w*p.h)
	p.lit = make([]bool, p.w*p.h)
	put := func(i int, r, g, bl, a uint32) { // premultiplied, 0 to 0xFFFF
		p.alpha[i] = uint8(a >> 8)
		p.lit[i] = (299*r+587*g+114*bl)/1000>>8 >= darkCut
	}
	if n, ok := img.(*image.NRGBA); ok { // what most transparent PNGs decode to
		for y := range p.h {
			row := n.Pix[(y+b.Min.Y-n.Rect.Min.Y)*n.Stride+(b.Min.X-n.Rect.Min.X)*4:]
			for x := range p.w {
				c := row[x*4 : x*4+4]
				a := uint32(c[3])
				put(y*p.w+x, uint32(c[0])*a*257/255, uint32(c[1])*a*257/255, uint32(c[2])*a*257/255, a*257)
			}
		}
		return p
	}
	for y := range p.h {
		for x := range p.w {
			r, g, bl, a := img.At(b.Min.X+x, b.Min.Y+y).RGBA()
			put(y*p.w+x, r, g, bl, a)
		}
	}
	return p
}

// shape returns the box around what shows, or false when the image is a
// solid rectangle or shows almost nothing.
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

// sample shrinks the crop to pw x ph dots. Heavy lettering is eroded first,
// so the thin gaps between letters survive the shrink; thin strokes are
// left alone and need less of a dot to show, so lines don't break up.
func (p picture) sample(crop image.Rectangle, pw, ph int) []bool {
	lit := p.lit
	perDot := float64(crop.Dx()) / float64(pw) // source pixels per dot
	stroke := p.stroke()
	if k := min(int(erodeShare*perDot+0.5), int((stroke-1.5*perDot)/2)); k > 0 {
		lit = keepSmallParts(lit, erode(lit, p.w, p.h, k), p.w)
	}
	cover := coverThick
	if stroke < perDot {
		cover = coverThin
	}

	on := make([]bool, pw*ph)
	w, h := crop.Dx(), crop.Dy()
	for y := range ph {
		y0 := crop.Min.Y + y*h/ph
		y1 := max(crop.Min.Y+(y+1)*h/ph, y0+1)
		for x := range pw {
			x0 := crop.Min.X + x*w/pw
			x1 := max(crop.Min.X+(x+1)*w/pw, x0+1)
			n := 0
			for sy := y0; sy < y1; sy++ {
				for _, v := range lit[sy*p.w+x0 : sy*p.w+x1] {
					if v {
						n++
					}
				}
			}
			on[y*pw+x] = float64(n) >= cover*float64((y1-y0)*(x1-x0))
		}
	}
	return on
}

// stroke estimates how thick the shape's strokes are, in source pixels:
// twice its area over the length of its edge.
func (p picture) stroke() float64 {
	area, edge := 0, 0
	for y := range p.h {
		for x := range p.w {
			i := y*p.w + x
			if !p.lit[i] {
				continue
			}
			area++
			if x == 0 || y == 0 || x == p.w-1 || y == p.h-1 || !p.lit[i-1] || !p.lit[i+1] || !p.lit[i-p.w] || !p.lit[i+p.w] {
				edge++
			}
		}
	}
	if edge == 0 {
		return math.Inf(1)
	}
	return 2 * float64(area) / float64(edge)
}

// erode shrinks the shape by k pixels on every side (a square window),
// using running sums so the cost doesn't grow with k.
func erode(on []bool, w, h, k int) []bool {
	pass := func(src []bool, n, stride, lines, step int) []bool {
		out := make([]bool, len(src))
		sums := make([]int, n+1)
		for line := range lines {
			base := line * step
			for i := range n {
				sums[i+1] = sums[i]
				if src[base+i*stride] {
					sums[i+1]++
				}
			}
			for i := range n {
				lo, hi := max(i-k, 0), min(i+k+1, n)
				out[base+i*stride] = sums[hi]-sums[lo] == hi-lo
			}
		}
		return out
	}
	return pass(pass(on, w, 1, h, w), h, w, w, 1)
}

// keepSmallParts puts back, as they were, the separate pieces of the shape
// that erosion wiped out or nearly: small print must not vanish.
func keepSmallParts(orig, eroded []bool, w int) []bool {
	// Label each piece (4-connected), counting its pixels and how many
	// erosion kept; label 0 is background.
	label := make([]int32, len(orig))
	restore := []bool{false}
	stack := make([]int32, 0, 1024)
	for start, on := range orig {
		if !on || label[start] != 0 {
			continue
		}
		id := int32(len(restore))
		size, kept := 0, 0
		label[start] = id
		stack = append(stack[:0], int32(start))
		for len(stack) > 0 {
			i := int(stack[len(stack)-1])
			stack = stack[:len(stack)-1]
			size++
			if eroded[i] {
				kept++
			}
			x := i % w
			for _, j := range [4]int{i - 1, i + 1, i - w, i + w} {
				switch {
				case j < 0 || j >= len(orig) || !orig[j] || label[j] != 0:
					continue
				case j == i-1 && x == 0, j == i+1 && x == w-1:
					continue // across the image's edge
				}
				label[j] = id
				stack = append(stack, int32(j))
			}
		}
		restore = append(restore, kept*restoreCut < size)
	}
	out := make([]bool, len(orig))
	for i := range out {
		out[i] = eroded[i] || restore[label[i]]
	}
	return out
}
