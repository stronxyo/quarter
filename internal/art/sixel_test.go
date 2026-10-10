package art

import (
	"errors"
	"image"
	"image/color"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

var accent = color.RGBA{0xFF, 0xAF, 0x63, 0xFF}

// unsixel decodes what sixel writes: the picture's size, its colours
// (0 to 100 per channel) and each pixel's colour index, -1 for none.
func unsixel(t *testing.T, s string) (w, h int, pal map[int][3]int, pix []int) {
	t.Helper()
	m := regexp.MustCompile(`^\x1bP0;1;0q"1;1;(\d+);(\d+)`).FindStringSubmatch(s)
	if m == nil || !strings.HasSuffix(s, "\x1b\\") {
		t.Fatalf("not a Sixel picture: %q", s[:min(len(s), 40)])
	}
	w, _ = strconv.Atoi(m[1])
	h, _ = strconv.Atoi(m[2])
	pix = make([]int, w*h)
	for i := range pix {
		pix[i] = -1
	}
	body := s[len(m[0]) : len(s)-2]
	pal = map[int][3]int{}
	defs := regexp.MustCompile(`#(\d+);2;(\d+);(\d+);(\d+)`)
	for _, d := range defs.FindAllStringSubmatch(body, -1) {
		n, _ := strconv.Atoi(d[1])
		r, _ := strconv.Atoi(d[2])
		g, _ := strconv.Atoi(d[3])
		b, _ := strconv.Atoi(d[4])
		pal[n] = [3]int{r, g, b}
	}
	body = defs.ReplaceAllString(body, "")
	x, y, cur := 0, 0, 0
	put := func(v byte) {
		for d := range 6 {
			if v>>d&1 == 1 && x < w && y+d < h {
				pix[(y+d)*w+x] = cur
			}
		}
		x++
	}
	for i := 0; i < len(body); i++ {
		switch c := body[i]; {
		case c == '#' || c == '!':
			j := i + 1
			for j < len(body) && body[j] >= '0' && body[j] <= '9' {
				j++
			}
			n, _ := strconv.Atoi(body[i+1 : j])
			if c == '#' {
				cur, i = n, j-1
				continue
			}
			for range n {
				put(body[j] - 63)
			}
			i = j
		case c == '$':
			x = 0
		case c == '-':
			x, y = 0, y+6
		default:
			put(c - 63)
		}
	}
	return w, h, pal, pix
}

// A one-colour logo is drawn in the accent alone, the size asked for,
// transparent around the shape.
func TestRenderImageOneColour(t *testing.T) {
	s, err := RenderImage(ellipse(320, 120), 300, 100, accent)
	if err != nil {
		t.Fatal(err)
	}
	w, h, pal, pix := unsixel(t, s)
	if w != 300 || h != 100 {
		t.Errorf("size %dx%d, want 300x100", w, h)
	}
	if len(pal) != 1 || pal[0] != [3]int{100, 69, 39} {
		t.Errorf("a one-colour logo should use the accent alone, got %v", pal)
	}
	if pix[0] != -1 || pix[50*w+150] != 0 {
		t.Error("the corner should be transparent and the middle drawn")
	}
}

// Two colours: the one covering more of the logo becomes the accent, the
// other a darker amber.
func TestRenderImageMainColourIsAccent(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 300, 100))
	for y := range 100 {
		for x := range 300 {
			c := color.NRGBA{40, 90, 220, 255} // blue: two thirds
			if x >= 200 {
				c = color.NRGBA{250, 250, 250, 255} // white: one third
			}
			img.SetNRGBA(x, y, c)
		}
	}
	for y := range 100 { // a transparent border, or it's a plain rectangle
		for x := range 300 {
			if x < 10 || y < 10 || x >= 290 || y >= 90 {
				img.SetNRGBA(x, y, color.NRGBA{})
			}
		}
	}
	s, err := RenderImage(img, 300, 100, accent)
	if err != nil {
		t.Fatal(err)
	}
	w, _, pal, pix := unsixel(t, s)
	blue, white := pal[pix[50*w+60]], pal[pix[50*w+260]]
	if blue != [3]int{100, 69, 39} {
		t.Errorf("the main colour should be the accent, got %v", blue)
	}
	if white[0]+white[1]+white[2] >= blue[0]+blue[1]+blue[2] {
		t.Errorf("the other colour should be a darker amber, got %v", white)
	}
}

func TestRenderImageUnusable(t *testing.T) {
	for name, img := range map[string]image.Image{
		"solid":       canvas(100, 50, image.Rect(0, 0, 100, 50)),
		"transparent": canvas(100, 50),
	} {
		if _, err := RenderImage(img, 300, 100, accent); !errors.Is(err, ErrUnusable) {
			t.Errorf("%s: got %v, want ErrUnusable", name, err)
		}
	}
	if _, err := RenderImage(ellipse(100, 50), 4, 4, accent); !errors.Is(err, ErrUnusable) {
		t.Error("no room should be unusable")
	}
}

// A name in text blocks becomes a picture of the same blocks, one gradient
// colour per row.
func TestLinesImage(t *testing.T) {
	s := LinesImage([]string{"█▀", "▄ "}, 40, 40, color.White, color.Black)
	w, _, pal, pix := unsixel(t, s)
	if len(pal) != 2 || pal[0] == pal[1] {
		t.Fatalf("each row should have its own colour, got %v", pal)
	}
	// Cells are 10 x 20 here, the two columns centred: x 10 to 30.
	at := func(x, y int) int { return pix[y*w+x] }
	if at(12, 2) != 0 || at(22, 2) != 0 || at(22, 15) != -1 || at(12, 35) != 1 || at(12, 22) != -1 {
		t.Error("the blocks should be drawn where the characters' dots are")
	}
}
