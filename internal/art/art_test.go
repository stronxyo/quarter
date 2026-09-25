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

	seen = map[rune]bool{}
	for n := range 64 {
		r := sextant(n)
		if seen[r] {
			t.Fatalf("sextant pattern %06b: character %U repeated", n, r)
		}
		seen[r] = true
	}
	for _, tt := range []struct {
		n    int
		want rune
	}{{0b000001, 0x1FB00}, {0b111110, 0x1FB3B}, {0b010101, '▌'}, {0b000011, 0x1FB02}} {
		if got := sextant(tt.n); got != tt.want {
			t.Errorf("sextant %06b = %U, want %U", tt.n, got, tt.want)
		}
	}
}

// A shape with a transparent border is cropped to the shape, keeps its
// proportions, and comes out the same size in eighths and sixths.
func TestRenderRectangle(t *testing.T) {
	img := canvas(600, 300, image.Rect(100, 100, 500, 200)) // 400 x 100 inside a border
	for _, b := range []Blocks{Eighths, Sixths} {
		lines, err := Render(img, 64, 14, b)
		if err != nil {
			t.Fatal(err)
		}
		// 64 cells wide; 400:100 is 4:1, and a cell is twice as tall as wide.
		if len(lines) != 8 {
			t.Errorf("blocks %d: got %d rows, want 8", b, len(lines))
		}
		for _, l := range lines {
			if l != strings.Repeat("█", 64) {
				t.Fatalf("blocks %d: want solid rows of 64, got %q", b, l)
			}
		}
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
	solid := func(l string) int { return strings.Count(l, "█") }
	if solid(lines[0]) >= solid(lines[len(lines)/2]) {
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

// Erosion opens gaps in heavy lettering, but a small piece it would wipe out
// is put back as it was.
func TestKeepSmallParts(t *testing.T) {
	const w, h = 60, 30
	orig := make([]bool, w*h)
	set := func(x0, y0, x1, y1 int) {
		for y := y0; y < y1; y++ {
			for x := x0; x < x1; x++ {
				orig[y*w+x] = true
			}
		}
	}
	set(2, 2, 40, 20)   // heavy lettering
	set(45, 22, 48, 26) // small print
	eroded := erode(orig, w, h, 2)
	if eroded[23*w+46] {
		t.Fatal("erosion by 2 should wipe out a 3-pixel stroke")
	}
	out := keepSmallParts(orig, eroded, w)
	for y := 22; y < 26; y++ {
		for x := 45; x < 48; x++ {
			if !out[y*w+x] {
				t.Fatalf("small print pixel %d,%d should be back", x, y)
			}
		}
	}
	if out[2*w+2] {
		t.Error("the heavy shape should stay eroded")
	}
}

func TestParseBlocks(t *testing.T) {
	for in, want := range map[string]Blocks{"": Eighths, "eighths": Eighths, "sixths": Sixths} {
		if got, ok := ParseBlocks(in); !ok || got != want {
			t.Errorf("ParseBlocks(%q) = %v, %v", in, got, ok)
		}
	}
	if _, ok := ParseBlocks("quarters"); ok {
		t.Error("unknown values should not parse")
	}
}

// The spec's target: a 640x360 logo drawn in under 10 ms.
func BenchmarkRender(b *testing.B) {
	img := ellipse(640, 360)
	for b.Loop() {
		if _, err := Render(img, 64, 14, Eighths); err != nil {
			b.Fatal(err)
		}
	}
}
