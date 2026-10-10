package art

import (
	"image"
	"image/color"
	"math"
	"sort"
)

// Colour for logo pictures: OKLab (Ottosson 2020), where distance follows
// what the eye sees; grouping a logo's colours; and Pablo's rule for turning
// them into amber tones.

// okl is a colour in OKLab.
type okl struct{ L, A, B float64 }

func (a okl) dist2(b okl) float64 {
	dl, da, db := a.L-b.L, a.A-b.A, a.B-b.B
	return dl*dl + da*da + db*db
}

func lin(c float64) float64 {
	c /= 255
	if c <= 0.04045 {
		return c / 12.92
	}
	return math.Pow((c+0.055)/1.055, 2.4)
}

func unlin(c float64) float64 {
	if c <= 0.0031308 {
		c *= 12.92
	} else {
		c = 1.055*math.Pow(c, 1/2.4) - 0.055
	}
	return math.Max(0, math.Min(255, c*255))
}

func toOK(c [3]float64) okl {
	r, g, b := lin(c[0]), lin(c[1]), lin(c[2])
	l := math.Cbrt(0.4122214708*r + 0.5363325363*g + 0.0514459929*b)
	m := math.Cbrt(0.2119034982*r + 0.6806995451*g + 0.1073969566*b)
	s := math.Cbrt(0.0883024619*r + 0.2817188376*g + 0.6299787005*b)
	return okl{0.2104542553*l + 0.7936177850*m - 0.0040720468*s,
		1.9779984951*l - 2.4285922050*m + 0.4505937099*s,
		0.0259040371*l + 0.7827717662*m - 0.8086757660*s}
}

func fromOK(o okl) [3]float64 {
	l := o.L + 0.3963377774*o.A + 0.2158037573*o.B
	m := o.L - 0.1055613458*o.A - 0.0638541728*o.B
	s := o.L - 0.0894841775*o.A - 1.2914855480*o.B
	l, m, s = l*l*l, m*m*m, s*s*s
	return [3]float64{unlin(4.0767416621*l - 3.3077115913*m + 0.2309699292*s),
		unlin(-1.2684380046*l + 2.6097574011*m - 0.3413193965*s),
		unlin(-0.0041960863*l - 0.7034186147*m + 1.7076147010*s)}
}

func toOKColor(c color.Color) okl {
	r, g, b, _ := c.RGBA()
	return toOK([3]float64{float64(r >> 8), float64(g >> 8), float64(b >> 8)})
}

// rgba is a premultiplied colour with alpha, 0 to 255.
type rgba [4]float64

func unpremul(s rgba) [3]float64 {
	a := s[3] / 255
	return [3]float64{s[0] / a, s[1] / a, s[2] / a}
}

// groups finds up to k colour groups among pts: k-means in OKLab, seeded by
// the farthest points, on an even sample of at most 20000 points.
func groups(pts []okl, k int) []okl {
	sample := pts
	if len(pts) > 20000 {
		step := len(pts) / 20000
		sample = make([]okl, 0, 20000)
		for i := 0; i < len(pts); i += step {
			sample = append(sample, pts[i])
		}
	}
	cent := []okl{sample[0]}
	for len(cent) < k {
		far, fd := sample[0], -1.0
		for _, p := range sample {
			if d := p.dist2(cent[nearestIdx(cent, p)]); d > fd {
				far, fd = p, d
			}
		}
		if fd < 0.002 { // no more distinct colours
			break
		}
		cent = append(cent, far)
	}
	for range 10 {
		sum := make([]okl, len(cent))
		n := make([]float64, len(cent))
		for _, p := range sample {
			j := nearestIdx(cent, p)
			sum[j].L, sum[j].A, sum[j].B = sum[j].L+p.L, sum[j].A+p.A, sum[j].B+p.B
			n[j]++
		}
		for j := range cent {
			if n[j] > 0 {
				cent[j] = okl{sum[j].L / n[j], sum[j].A / n[j], sum[j].B / n[j]}
			}
		}
	}
	return cent
}

func nearestIdx(pal []okl, c okl) int {
	best, bd := 0, math.Inf(1)
	for j, p := range pal {
		if d := p.dist2(c); d < bd {
			best, bd = j, d
		}
	}
	return best
}

const (
	toneStep   = 0.11 // OKLab lightness between neighbouring tones, at most
	toneFloor  = 0.40 // the darkest tone, still clear on a dark terminal
	toneChroma = 0.95 // the tones below the accent, a touch less saturated
)

// amberTones gives each colour group its tone, given how many pixels each
// covers: the biggest is the accent, the brightest tone; the others step
// down from it in lightness, nearest in the logo's own lightness first, so
// nothing outshines it.
func amberTones(groups []okl, count []int, accent color.Color) []okl {
	acc := toOKColor(accent)
	main := 0
	for i := range groups {
		if count[i] > count[main] {
			main = i
		}
	}
	rest := make([]int, 0, len(groups))
	for i := range groups {
		if i != main {
			rest = append(rest, i)
		}
	}
	sort.SliceStable(rest, func(a, b int) bool {
		return math.Abs(groups[rest[a]].L-groups[main].L) < math.Abs(groups[rest[b]].L-groups[main].L)
	})
	step := math.Min(toneStep, (acc.L-toneFloor)/float64(max(len(rest), 1)))
	tones := make([]okl, len(groups))
	tones[main] = acc
	for n, i := range rest {
		tones[i] = okl{acc.L - step*float64(n+1), acc.A * toneChroma, acc.B * toneChroma}
	}
	return tones
}

func toNRGBA(img image.Image) *image.NRGBA {
	if n, ok := img.(*image.NRGBA); ok {
		return n
	}
	b := img.Bounds()
	out := image.NewNRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			out.Set(x, y, color.NRGBAModel.Convert(img.At(x, y)))
		}
	}
	return out
}
