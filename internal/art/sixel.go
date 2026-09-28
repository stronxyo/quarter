package art

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

// Logos as real pictures, for terminals that draw Sixel graphics (Windows
// Terminal 1.22 and later, among others): many times the detail of text
// blocks, so small print and brush lettering stay readable. Pablo's pick in
// Batch 8, "A. real image, amber tones".

const (
	imageAlpha = 64  // a pixel this opaque counts toward the logo's box
	imageSolid = 128 // a pixel this opaque is drawn; the rest is transparent
	imageTones = 5   // colour groups per logo, the accent included
)

// RenderImage draws a logo as a Sixel picture, fitted inside w x h pixels
// and centred there, in amber tones (see amberTones). Transparent parts stay
// transparent, so the terminal's background shows through.
func RenderImage(img image.Image, w, h int, accent color.Color) (string, error) {
	if w < 8 || h < 8 {
		return "", ErrUnusable
	}
	src := toNRGBA(shrinkLarge(img))
	b := src.Bounds()
	var box image.Rectangle
	opaque := 0
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			if src.Pix[src.PixOffset(x, y)+3] >= imageAlpha {
				box = box.Union(image.Rect(x, y, x+1, y+1))
				opaque++
			}
		}
	}
	if opaque < minVisible || float64(opaque) >= solidShare*float64(b.Dx()*b.Dy()) {
		return "", ErrUnusable
	}
	s := math.Min(float64(w)/float64(box.Dx()), float64(h)/float64(box.Dy()))
	lw, lh := max(1, int(float64(box.Dx())*s)), max(1, int(float64(box.Dy())*s))
	logo := resample(src, box, lw, lh)

	var pts []okl
	for _, p := range logo {
		if p[3] >= imageSolid {
			pts = append(pts, toOK(unpremul(p)))
		}
	}
	if len(pts) == 0 {
		return "", ErrUnusable
	}
	g := groups(pts, imageTones)
	count := make([]int, len(g))
	label := make([]int, len(logo))
	k := 0
	for i, p := range logo {
		label[i] = -1
		if p[3] >= imageSolid {
			label[i] = nearestIdx(g, pts[k])
			count[label[i]]++
			k++
		}
	}
	tones := amberTones(g, count, accent)
	colours := make([][3]float64, len(tones))
	for i, t := range tones {
		colours[i] = fromOK(t)
	}

	pix := make([]int, w*h)
	for i := range pix {
		pix[i] = -1
	}
	ox, oy := (w-lw)/2, (h-lh)/2
	for y := range lh {
		copy(pix[(y+oy)*w+ox:], label[y*lw:(y+1)*lw])
	}
	return sixel(pix, w, h, colours), nil
}

// RenderImageFile reads a PNG and draws it as RenderImage does.
func RenderImageFile(path string, w, h int, accent color.Color) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	img, err := png.Decode(f)
	if err != nil {
		return "", fmt.Errorf("%s: %w", filepath.Base(path), err)
	}
	return RenderImage(img, w, h, accent)
}

// resample scales the box of src to w x h pixels, premultiplied: an area
// average when shrinking, bilinear when growing.
func resample(src *image.NRGBA, box image.Rectangle, w, h int) []rgba {
	px := func(x, y int) rgba {
		x = min(max(x, box.Min.X), box.Max.X-1)
		y = min(max(y, box.Min.Y), box.Max.Y-1)
		i := src.PixOffset(x, y)
		a := float64(src.Pix[i+3])
		return rgba{float64(src.Pix[i]) * a / 255, float64(src.Pix[i+1]) * a / 255, float64(src.Pix[i+2]) * a / 255, a}
	}
	sx, sy := float64(box.Dx())/float64(w), float64(box.Dy())/float64(h)
	out := make([]rgba, w*h)
	for y := range h {
		for x := range w {
			var s rgba
			if sx >= 1 && sy >= 1 {
				x0, x1 := box.Min.X+int(float64(x)*sx), box.Min.X+max(int(float64(x+1)*sx), int(float64(x)*sx)+1)
				y0, y1 := box.Min.Y+int(float64(y)*sy), box.Min.Y+max(int(float64(y+1)*sy), int(float64(y)*sy)+1)
				n := 0.0
				for yy := y0; yy < y1; yy++ {
					for xx := x0; xx < x1; xx++ {
						p := px(xx, yy)
						for c := range 4 {
							s[c] += p[c]
						}
						n++
					}
				}
				for c := range 4 {
					s[c] /= n
				}
			} else {
				fx := float64(box.Min.X) + (float64(x)+0.5)*sx - 0.5
				fy := float64(box.Min.Y) + (float64(y)+0.5)*sy - 0.5
				x0, y0 := int(math.Floor(fx)), int(math.Floor(fy))
				ax, ay := fx-float64(x0), fy-float64(y0)
				p00, p10, p01, p11 := px(x0, y0), px(x0+1, y0), px(x0, y0+1), px(x0+1, y0+1)
				for c := range 4 {
					s[c] = (p00[c]*(1-ax)+p10[c]*ax)*(1-ay) + (p01[c]*(1-ax)+p11[c]*ax)*ay
				}
			}
			out[y*w+x] = s
		}
	}
	return out
}

// LinesImage draws text-block lines (quarter blocks, as Name returns them)
// as a Sixel picture in the logo gradient, top to bottom, fitted in w x h
// pixels: a game's name, when it has no logo, beside the other pictures.
func LinesImage(lines []string, w, h int, top, bottom color.Color) string {
	rows, cols := len(lines), 0
	for _, l := range lines {
		cols = max(cols, len([]rune(l)))
	}
	if rows == 0 || cols == 0 {
		return ""
	}
	cw := min(w/cols, h/rows/2) // a cell twice as tall as wide
	ch := cw * 2
	if cw < 2 {
		return ""
	}
	ox, oy := (w-cols*cw)/2, (h-rows*ch)/2
	pix := make([]int, w*h)
	for i := range pix {
		pix[i] = -1
	}
	t0, t1 := toOKColor(top), toOKColor(bottom)
	colours := make([][3]float64, rows)
	for y, l := range lines {
		f := 0.0
		if rows > 1 {
			f = float64(y) / float64(rows-1)
		}
		colours[y] = fromOK(okl{t0.L + f*(t1.L-t0.L), t0.A + f*(t1.A-t0.A), t0.B + f*(t1.B-t0.B)})
		for x, r := range []rune(l) {
			n := slices.Index(quadrantRunes, r)
			for q := range 4 {
				if n < 0 || n>>q&1 == 0 {
					continue
				}
				x0, y0 := ox+x*cw+(q%2)*cw/2, oy+y*ch+(q/2)*ch/2
				for yy := y0; yy < y0+ch/2; yy++ {
					for xx := x0; xx < x0+cw/2; xx++ {
						pix[yy*w+xx] = y
					}
				}
			}
		}
	}
	return sixel(pix, w, h, colours)
}

// sixel encodes a w x h picture of palette indexes (-1: transparent) as
// Sixel, with a transparent background.
func sixel(pix []int, w, h int, colours [][3]float64) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\x1bP0;1;0q\"1;1;%d;%d", w, h)
	for i, c := range colours {
		fmt.Fprintf(&b, "#%d;2;%d;%d;%d", i, int(c[0]*100/255+0.5), int(c[1]*100/255+0.5), int(c[2]*100/255+0.5))
	}
	bits := make([]byte, w)
	for y0 := 0; y0 < h; y0 += 6 {
		var present []int
		for dy := 0; dy < 6 && y0+dy < h; dy++ {
			for _, c := range pix[(y0+dy)*w : (y0+dy+1)*w] {
				if c >= 0 && !slices.Contains(present, c) {
					present = append(present, c)
				}
			}
		}
		slices.Sort(present)
		for n, c := range present {
			for x := range w {
				v := byte(0)
				for dy := 0; dy < 6 && y0+dy < h; dy++ {
					if pix[(y0+dy)*w+x] == c {
						v |= 1 << dy
					}
				}
				bits[x] = 63 + v
			}
			fmt.Fprintf(&b, "#%d", c)
			for i := 0; i < w; {
				j := i
				for j < w && bits[j] == bits[i] {
					j++
				}
				if j-i > 3 {
					fmt.Fprintf(&b, "!%d%c", j-i, bits[i])
				} else {
					b.Write(bits[i:j])
				}
				i = j
			}
			if n < len(present)-1 {
				b.WriteByte('$') // back to the start of this row of six
			}
		}
		b.WriteByte('-') // the next row of six
	}
	b.WriteString("\x1b\\")
	return b.String()
}
