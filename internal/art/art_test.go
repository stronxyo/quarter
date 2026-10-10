package art

import (
	"errors"
	"image"
	"image/color"
	"strings"
	"testing"
)

// canvas is a transparent w x h image with the given rectangles opaque.
func canvas(w, h int, opaque ...image.Rectangle) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for _, r := range opaque {
		for y := r.Min.Y; y < r.Max.Y; y++ {
			for x := r.Min.X; x < r.Max.X; x++ {
				img.SetNRGBA(x, y, color.NRGBA{255, 255, 255, 255})
			}
		}
	}
	return img
}

// ellipse is a transparent w x h image with an opaque ellipse filling it.
func ellipse(w, h int) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, w, h))
	for y := range h {
		for x := range w {
			fx, fy := (float64(x)+0.5)/float64(w)*2-1, (float64(y)+0.5)/float64(h)*2-1
			if fx*fx+fy*fy <= 1 {
				img.SetNRGBA(x, y, color.NRGBA{255, 255, 255, 255})
			}
		}
	}
	return img
}

func TestTables(t *testing.T) {
	seen := map[rune]bool{}
	for n, r := range octants {
		if r == 0 || seen[r] {
			t.Fatalf("octant pattern %08b: character %U missing or repeated", n, r)
		}
		seen[r] = true
	}
	for _, tt := range []struct {
		n    int
		want rune
	}{{0, ' '}, {0b11111111, '█'}, {0b01010101, '▌'}, {0b10101010, '▐'}, {0b00001111, '▀'}, {0b11110000, '▄'}, {0b00000001, 0x1CEA8}} {
		if got := octants[tt.n]; got != tt.want {
			t.Errorf("octant %08b = %U, want %U", tt.n, got, tt.want)
		}
	}

	// Quadrants: 16 distinct block elements, all in U+2580 to U+259F (or a
	// space), the range every common monospace font has.
	seen = map[rune]bool{}
	for n, r := range quadrantRunes {
		if seen[r] || (r != ' ' && (r < 0x2580 || r > 0x259F)) {
			t.Fatalf("quadrant pattern %04b: %U repeated or outside the block elements", n, r)
		}
		seen[r] = true
	}
	for _, tt := range []struct {
		n    int
		want rune
	}{{0, ' '}, {0b1111, '█'}, {0b0101, '▌'}, {0b0011, '▀'}, {0b1001, '▚'}, {0b0001, '▘'}} {
		if got := quadrantRunes[tt.n]; got != tt.want {
			t.Errorf("quadrant %04b = %c, want %c", tt.n, got, tt.want)
		}
	}
}

// A shape with a transparent border is cropped to the shape, keeps its
// proportions, and comes out the same size in quadrants and eighths. One big
// solid block is a fill, so it's drawn as its outline: full top and bottom
// rows, a hollow middle.
func TestRenderRectangle(t *testing.T) {
	img := canvas(600, 300, image.Rect(100, 100, 500, 200)) // 400 x 100 inside a border
	for _, b := range []Blocks{Quadrants, Eighths} {
		lines, err := Render(img, 64, 14, b)
		if err != nil {
			t.Fatal(err)
		}
		// 64 cells wide; 400:100 is 4:1, and a cell is twice as tall as wide.
		if len(lines) != 8 || len([]rune(lines[0])) != 64 {
			t.Fatalf("blocks %d: got %d rows of %d, want 8 of 64", b, len(lines), len([]rune(lines[0])))
		}
		for _, l := range []string{lines[0], lines[len(lines)-1]} {
			if strings.Contains(l, " ") {
				t.Errorf("blocks %d: the rim should run the full width: %q", b, l)
			}
		}
		if mid := []rune(lines[len(lines)/2]); mid[len(mid)/2] != ' ' {
			t.Errorf("blocks %d: the middle should be hollow:\n%s", b, strings.Join(lines, "\n"))
		}
	}
}

// Letters, however heavy, stay solid: several similar pieces, none of them a
// big share of the logo, are not a fill.
func TestRenderLettersStaySolid(t *testing.T) {
	var bars []image.Rectangle
	for i := range 6 { // six heavy "letters"
		bars = append(bars, image.Rect(20+i*100, 20, 100+i*100, 200))
	}
	lines, err := Render(canvas(640, 220, bars...), 60, 13, Quadrants)
	if err != nil {
		t.Fatal(err)
	}
	if mid := lines[len(lines)/2]; strings.Count(mid, "█") < 40 {
		t.Errorf("the letters should be solid, not outlines:\n%s", strings.Join(lines, "\n"))
	}
}

func TestRenderTallShapeFitsRows(t *testing.T) {
	lines, err := Render(canvas(200, 500, image.Rect(50, 50, 150, 450)), 64, 14, Eighths)
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 14 || len([]rune(lines[0])) != 7 {
		t.Errorf("got %d rows of %d, want 14 rows of 7", len(lines), len([]rune(lines[0])))
	}
}

// An ellipse's rows are all the same width, narrower at the top than in the
// middle, and use partial blocks at its edges.
func TestRenderEllipse(t *testing.T) {
	lines, err := Render(ellipse(640, 360), 64, 14, Eighths)
	if err != nil {
		t.Fatal(err)
	}
	width := len([]rune(lines[0]))
	for _, l := range lines {
		if len([]rune(l)) != width {
			t.Fatalf("rows differ in width:\n%s", strings.Join(lines, "\n"))
		}
	}
	// Measured by how far each row reaches: a big ellipse is a fill, drawn
	// as its outline, so counting solid blocks would say nothing.
	reach := func(l string) int { return len([]rune(strings.TrimSpace(l))) }
	if reach(lines[0]) >= reach(lines[len(lines)/2]) {
		t.Errorf("the top should be narrower than the middle:\n%s", strings.Join(lines, "\n"))
	}
	if !strings.ContainsFunc(strings.Join(lines, ""), func(r rune) bool { return r >= 0x1CD00 && r <= 0x1CDE5 }) {
		t.Error("the curved edge should use octant characters")
	}
}

func TestRenderUnusable(t *testing.T) {
	cases := map[string]image.Image{
		"solid":       canvas(100, 50, image.Rect(0, 0, 100, 50)),
		"transparent": canvas(100, 50),
		"a speck":     canvas(100, 50, image.Rect(10, 10, 12, 12)),
		"not NRGBA":   image.NewGray(image.Rect(0, 0, 100, 50)), // always opaque
	}
	for name, img := range cases {
		if _, err := Render(img, 64, 14, Eighths); !errors.Is(err, ErrUnusable) {
			t.Errorf("%s: got %v, want ErrUnusable", name, err)
		}
	}
	if _, err := Render(ellipse(100, 50), 0, 14, Eighths); !errors.Is(err, ErrUnusable) {
		t.Error("no room should be unusable")
	}
}

// Other image types go through the slower generic path and agree with NRGBA.
func TestRenderGenericImage(t *testing.T) {
	src := ellipse(320, 180)
	rgba := image.NewRGBA(src.Bounds())
	for y := range 180 {
		for x := range 320 {
			rgba.Set(x, y, src.At(x, y))
		}
	}
	a, _ := Render(src, 40, 10, Eighths)
	b, err := Render(rgba, 40, 10, Eighths)
	if err != nil || strings.Join(a, "\n") != strings.Join(b, "\n") {
		t.Errorf("RGBA and NRGBA should draw the same (err %v)", err)
	}
}

// Two white letters touching through a thin black outline, the way many
// Steam logos are drawn: the outline counts as background, and the gap
// survives even when it is thinner than a dot.
func TestRenderKeepsDarkGaps(t *testing.T) {
	img := canvas(440, 140, image.Rect(20, 20, 216, 120), image.Rect(224, 20, 420, 120))
	black := color.NRGBA{0, 0, 0, 255}
	for y := 20; y < 120; y++ {
		for x := 216; x < 224; x++ { // an 8-pixel outline: under one dot at 20 cells
			img.SetNRGBA(x, y, black)
		}
	}
	lines, err := Render(img, 20, 10, Eighths)
	if err != nil {
		t.Fatal(err)
	}
	// The gap may fall inside a cell (▌▐), so no row may be solid across.
	for _, l := range lines {
		if !strings.ContainsFunc(l, func(r rune) bool { return r != '█' }) {
			t.Fatalf("the two shapes should stay apart:\n%s", strings.Join(lines, "\n"))
		}
	}
}

// An oversized logo is shrunk before drawing and still comes out the same
// shape as a normal-sized one.
func TestShrinkLarge(t *testing.T) {
	small := ellipse(640, 360)
	if shrinkLarge(small) != image.Image(small) {
		t.Error("a normal logo should be left as it is")
	}
	big := ellipse(4000, 2250)
	s := shrinkLarge(big)
	if s.Bounds().Dx() > maxSourceWidth || s.Bounds().Dx() < maxSourceWidth/2 {
		t.Errorf("shrunk to %d wide", s.Bounds().Dx())
	}
	a, _ := Render(small, 40, 10, Quadrants)
	b, _ := Render(big, 40, 10, Quadrants)
	if len(a) != len(b) || len([]rune(a[0])) != len([]rune(b[0])) {
		t.Errorf("sizes differ: %dx%d and %dx%d", len([]rune(a[0])), len(a), len([]rune(b[0])), len(b))
	}
}

// Dark specks in a bright logo are texture and get filled; a long dark
// outline stays a gap.
func TestFillSpecks(t *testing.T) {
	const w, h = 40, 20
	lit := make([]bool, w*h)
	alpha := make([]uint8, w*h)
	for i := range lit {
		lit[i], alpha[i] = true, 255 // an opaque bright block
	}
	lit[5*w+5], lit[5*w+6] = false, false // a 2-pixel dark speck
	for y := range h {
		lit[y*w+20] = false // a dark outline right across
	}
	out := fillSpecks(lit, alpha, w, 10)
	if !out[5*w+5] || !out[5*w+6] {
		t.Error("the speck should be filled")
	}
	for y := range h { // every part of the outline, not only where the walk began
		if out[y*w+20] {
			t.Fatalf("the outline should stay a gap, row %d was filled", y)
		}
	}
}

// The fast max filter agrees with the obvious one everywhere.
func TestMaxFilter(t *testing.T) {
	const w, h = 37, 23
	v := make([]float32, w*h)
	for i := range v {
		v[i] = float32((i*7919 + 13) % 101) // spread-out values, no pattern
	}
	for _, r := range []int{0, 1, 2, 5, 11, 40} {
		got := maxFilter(v, w, h, r)
		for y := range h {
			for x := range w {
				want := float32(0)
				for yy := max(y-r, 0); yy <= min(y+r, h-1); yy++ {
					for xx := max(x-r, 0); xx <= min(x+r, w-1); xx++ {
						want = max(want, v[yy*w+xx])
					}
				}
				if got[y*w+x] != want {
					t.Fatalf("r %d at %d,%d: got %v, want %v", r, x, y, got[y*w+x], want)
				}
			}
		}
	}
}

// narrowGaps marks a gap narrower than the closing and a small hole, but
// not a wide gap or the open space around the shape.
func TestNarrowGaps(t *testing.T) {
	const w, h = 100, 40
	lit := make([]bool, w*h)
	set := func(x0, y0, x1, y1 int, on bool) {
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				lit[y*w+x] = on
			}
		}
	}
	set(5, 5, 30, 35, true)    // a letter
	set(34, 5, 55, 35, true)   // another, 4 pixels away
	set(75, 5, 95, 35, true)   // a third, 20 pixels away
	set(40, 15, 46, 21, false) // a 6-pixel hole in the second
	gap := narrowGaps(lit, w, h, 3)
	for _, c := range []struct {
		x, y int
		want bool
		what string
	}{
		{31, 20, true, "the 4-pixel gap"},
		{43, 18, true, "the small hole"},
		{65, 20, false, "the 20-pixel gap"},
		{2, 2, false, "the space around the shape"},
		{10, 10, false, "the letter itself"},
	} {
		if gap[c.y*w+c.x] != c.want {
			t.Errorf("%s: got %v, want %v", c.what, gap[c.y*w+c.x], c.want)
		}
	}
}

// A logo whose body is dark (like Cyberpunk 2077's, 74% near black) is drawn
// by its shape, not its colour: the terminal paints it in the gradient.
func TestRenderDarkLogo(t *testing.T) {
	img := canvas(300, 100)
	for y := 20; y < 80; y++ {
		for x := 20; x < 280; x++ {
			img.SetNRGBA(x, y, color.NRGBA{10, 10, 10, 255})
		}
	}
	lines, err := Render(img, 30, 10, Quadrants)
	if err != nil || !strings.Contains(strings.Join(lines, ""), "█") {
		t.Errorf("a dark logo should still draw, got %v:\n%s", err, strings.Join(lines, "\n"))
	}
}

// A hair-thin line, far narrower than a dot, still shows.
func TestRenderKeepsThinLines(t *testing.T) {
	img := canvas(400, 200, image.Rect(0, 98, 400, 101)) // 3 pixels tall, a dot is ~7
	lines, err := Render(img, 60, 10, Quadrants)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.ContainsFunc(strings.Join(lines, ""), func(r rune) bool { return r != ' ' }) || len([]rune(lines[0])) < 55 {
		t.Errorf("the thin line should run across:\n%s", strings.Join(lines, "\n"))
	}
}

func TestParseBlocks(t *testing.T) {
	for in, want := range map[string]Blocks{"": Quadrants, "quadrants": Quadrants, "octants": Eighths, "eighths": Eighths} {
		if got, ok := ParseBlocks(in); !ok || got != want {
			t.Errorf("ParseBlocks(%q) = %v, %v", in, got, ok)
		}
	}
	if _, ok := ParseBlocks("sixths"); ok {
		t.Error("unknown values should not parse")
	}
}

// A worst case: one big fill covering a whole 640x360 image, so nothing is
// cropped and it's outlined (about 19 ms). Pablo's 265 real logos: median
// 7.5 ms, slowest 39 ms, drawn off the UI thread and cached.
func BenchmarkRender(b *testing.B) {
	img := ellipse(640, 360)
	for b.Loop() {
		if _, err := Render(img, 64, 14, Eighths); err != nil {
			b.Fatal(err)
		}
	}
}
