package intro

// The intros. Each draws the logo at time t (seconds) from the start; at
// t >= dur the screen is the final frame, identical to the resting start
// screen. Timings follow Intro lab/research/RESEARCH.md: 0.6 to 1.4 s in all,
// 200 to 400 ms per element, ease-out entrances, at most one overshoot, no
// flashes.

import (
	"math"
)

type intro struct {
	name string
	note string  // where it comes from, what it's for
	dur  float64 // seconds
	draw func(g *grid, s scene, t float64, fx *fxState)
}

// fxState holds per-intro randomness, fixed for a run so replays match.
type fxState struct {
	rnd   [][3]float64 // three random numbers per logo piece
	cache map[string][]float64
	frame int
}

// hash is a repeatable random number in [0, 1) for two integers.
func hash(a, b int) float64 {
	h := uint32(a)*374761393 + uint32(b)*668265263
	h = (h ^ (h >> 13)) * 1274126177
	return float64(h^(h>>16)) / float64(1<<32)
}

func (s scene) centre() (float64, float64) {
	return float64(s.lx) + float64(s.bw)/2, float64(s.ly) + float64(s.bh)/2
}

func inBox(s scene, x, y int) bool {
	return x > s.lx && x < s.lx+s.bw-1 && y > s.ly && y < s.ly+s.bh-1
}

var intros = []intro{
	{
		name: "CRT power-on",
		note: "a tube warming up: a dot, a line, then the picture opens out (arcade monitors, TVs)",
		dur:  0.8,
		draw: func(g *grid, s scene, t float64, _ *fxState) {
			p := t / 0.8
			cx, cy := s.centre()
			dot, line, open := span(p, 0, 0.2), span(p, 0.2, 0.5), span(p, 0.5, 1)
			if line == 0 {
				g.set(int(cx), int(cy), cell{r: '■', fg: mix(colBG, colHot, outCubic(dot))})
				return
			}
			if open == 0 {
				half := float64(s.bw) / 2 * outExpo(line)
				for x := int(cx - half); x <= int(cx+half); x++ {
					g.set(x, int(cy), cell{r: '━', fg: colHot})
				}
				return
			}
			reach := (float64(s.bh)/2 + 1) * outCubic(open)
			for _, pc := range s.logo {
				if math.Abs(float64(pc.y)-cy) <= reach {
					c := pc.c
					c.fg = mix(colHot, c.fg, open) // the picture cools from white-hot
					g.set(pc.x, pc.y, c)
				}
			}
		},
	},
	{
		name: "Implode (reverse explosion)",
		note: "every piece flies in from scattered points and locks into place",
		dur:  1.0,
		draw: func(g *grid, s scene, t float64, fx *fxState) {
			for i, pc := range s.logo {
				r := fx.rnd[i]
				delay := 0.15 * r[2]
				q := span(t, delay, delay+0.85)
				if q >= 1 {
					g.set(pc.x, pc.y, pc.c)
					continue
				}
				a, dist := 2*math.Pi*r[0], 0.6*float64(s.bw)*(0.5+0.5*r[1])
				e := outCubic(q)
				x := float64(pc.x) + math.Cos(a)*dist*(1-e)
				y := float64(pc.y) + math.Sin(a)*dist*0.5*(1-e) // cells are twice as tall
				g.set(int(math.Round(x)), int(math.Round(y)), inFlight(pc, e))
			}
		},
	},
	{
		name: "Coin drop",
		note: "a quarter falls into the slot, and the letters drop into the cabinet with a bounce",
		dur:  1.4,
		draw: func(g *grid, s scene, t float64, _ *fxState) {
			cx, _ := s.centre()
			// The box is the cabinet, there from the start.
			for _, pc := range s.logo {
				if pc.kind == kindBorder {
					c := pc.c
					c.fg = mix(colDim, colAccent, span(t, 0, 0.25))
					g.set(pc.x, pc.y, c)
				}
			}
			// The coin falls under gravity to the box's top edge.
			if t < 0.45 {
				q := t / 0.45
				y := int(-2 + float64(s.ly+1)*q*q)
				g.text(int(cx)-2, y, "▄██▄", colHot)
				g.text(int(cx)-2, y+1, "▀██▀", colAccent)
				return
			}
			// Then the letters drop in, bouncing to rest.
			q := span(t, 0.5, 1.4)
			drop := float64(s.bh) * (1 - outBounce(q))
			for _, pc := range s.logo {
				if pc.kind == kindBorder {
					continue
				}
				y := pc.y - int(math.Round(drop))
				if inBox(s, pc.x, y) {
					g.set(pc.x, y, pc.c)
				}
			}
		},
	},
	{
		name: "Staggered drop",
		note: "columns fall in left to right, 50 ms apart, each overshooting once",
		dur:  1.2,
		draw: func(g *grid, s scene, t float64, _ *fxState) {
			for _, pc := range s.logo {
				if pc.kind == kindBorder {
					c := pc.c
					c.fg = mix(colBG, colAccent, span(t, 0, 0.3))
					g.set(pc.x, pc.y, c)
					continue
				}
				delay := float64(pc.x-s.lx) / float64(s.bw) * 0.6
				q := span(t, delay, delay+0.5)
				if q == 0 {
					continue
				}
				y := pc.y - int(math.Round(8*(1-outBack(q))))
				if inBox(s, pc.x, y) {
					g.set(pc.x, y, pc.c)
				}
			}
		},
	},
	{
		name: "Raster reveal",
		note: "the picture is drawn top to bottom by a bright scanline, like a CRT's beam",
		dur:  0.7,
		draw: func(g *grid, s scene, t float64, _ *fxState) {
			row := float64(s.bh) * inOutSine(t/0.7)
			for _, pc := range s.logo {
				r := float64(pc.y - s.ly)
				switch {
				case r < row-1:
					g.set(pc.x, pc.y, pc.c)
				case r < row:
					c := pc.c
					c.fg = colHot // the beam's leading row
					g.set(pc.x, pc.y, c)
				}
			}
		},
	},
	{
		name: "Typewriter",
		note: "printed in reading order with a block cursor at the head, like the _ in the wordmark",
		dur:  1.1,
		draw: func(g *grid, s scene, t float64, _ *fxState) {
			n := int(float64(len(s.logo)) * clamp(t/1.1))
			for _, pc := range s.byReading()[:n] {
				g.set(pc.x, pc.y, pc.c)
			}
			if n < len(s.logo) {
				head := s.byReading()[n]
				g.set(head.x, head.y, cell{r: '█', fg: colHot})
			}
		},
	},
	{
		name: "Decode",
		note: "every cell cycles through random glyphs, then locks to the logo (a film-terminal decrypt)",
		dur:  1.2,
		draw: func(g *grid, s scene, t float64, fx *fxState) {
			const noise = "01#%@$&*+=<>/\\"
			for i, pc := range s.logo {
				lock := (0.25 + 0.7*fx.rnd[i][0]) * 1.2
				if t >= lock {
					g.set(pc.x, pc.y, pc.c)
					continue
				}
				k := int(hash(i, fx.frame/2) * float64(len(noise)))
				g.set(pc.x, pc.y, cell{r: rune(noise[k]), fg: mix(colDim, pc.c.fg, t/lock)})
			}
		},
	},
	{
		name: "Light sweep",
		note: "the logo fades up, then a band of light passes across it (the shine of a new cabinet)",
		dur:  1.0,
		draw: func(g *grid, s scene, t float64, _ *fxState) {
			fade := outCubic(span(t, 0, 0.3))
			sweep := -6 + float64(s.bw+s.bh+12)*inOutSine(span(t, 0.3, 1.0))
			for _, pc := range s.logo {
				d := float64(pc.x-s.lx) + 0.5*float64(pc.y-s.ly) - sweep
				c := pc.c
				c.fg = mix(mix(colBG, c.fg, fade), colHot, math.Max(0, 1-math.Abs(d)/6))
				g.set(pc.x, pc.y, c)
			}
		},
	},
	{
		name: "Dither dissolve",
		note: "cells appear in a 4x4 Bayer pattern, soft edges in shade blocks (8-bit fades)",
		dur:  0.9,
		draw: func(g *grid, s scene, t float64, _ *fxState) {
			bayer := [4][4]float64{{0, 8, 2, 10}, {12, 4, 14, 6}, {3, 11, 1, 9}, {15, 7, 13, 5}}
			e := outCubic(t / 0.9)
			for _, pc := range s.logo {
				thr := (bayer[pc.y%4][pc.x%4] + 0.5) / 16
				switch lead := e - thr; {
				case lead > 0.15:
					g.set(pc.x, pc.y, pc.c)
				case lead > 0:
					ramp := []rune("░▒▓")[int(lead/0.15*2.99)]
					g.set(pc.x, pc.y, cell{r: ramp, fg: pc.c.fg})
				}
			}
		},
	},
	{
		name: "Glitch settle",
		note: "rows jitter sideways and settle into place (a signal locking on)",
		dur:  0.9,
		draw: func(g *grid, s scene, t float64, fx *fxState) {
			amp := math.Pow(1-clamp(t/0.75), 2)
			for _, pc := range s.logo {
				dx := int(math.Round((hash(pc.y, fx.frame/2)*8 - 4) * amp))
				c := pc.c
				if dx != 0 {
					c.fg = mix(c.fg, colDim, 0.4)
				}
				g.set(pc.x+dx, pc.y, c)
			}
		},
	},
	{
		name: "Starfield warp",
		note: "stars stream out from the centre, then the logo gathers from the middle (a space-shooter title)",
		dur:  1.4,
		draw: func(g *grid, s scene, t float64, fx *fxState) {
			cx, cy := s.centre()
			fade := 1 - span(t, 0.8, 1.2)
			for j := range 90 {
				a := 2 * math.Pi * hash(j, 1)
				r := math.Mod(hash(j, 2)*30+t*60*(0.5+hash(j, 3)), 60)
				x, y := cx+math.Cos(a)*r*2, cy+math.Sin(a)*r
				star := []rune(".+*")[min(int(r/20), 2)]
				g.set(int(x), int(y), cell{r: star, fg: mix(colBG, mix(colDim, colAccent, r/60), fade)})
			}
			for i, pc := range s.logo {
				delay := 0.5 + 0.3*fx.rnd[i][0]
				q := span(t, delay, delay+0.6)
				if q == 0 {
					continue
				}
				e := outCubic(q)
				x := cx + (float64(pc.x)-cx)*e
				y := cy + (float64(pc.y)-cy)*e
				c := pc.c
				if q < 1 {
					c = inFlight(pc, e)
				}
				g.set(int(math.Round(x)), int(math.Round(y)), c)
			}
		},
	},
	{
		name: "Beams",
		note: "two searchlights sweep in from the sides and meet in the middle, lighting the logo as they pass",
		dur:  1.1,
		draw: func(g *grid, s scene, t float64, _ *fxState) {
			reach := float64(s.bw) / 2 * inOutSine(t/1.1)
			left, right := float64(s.lx)+reach, float64(s.lx+s.bw)-reach
			for _, pc := range s.logo {
				x := float64(pc.x)
				beam, lit := left, x <= left
				if x > float64(s.lx)+float64(s.bw)/2 {
					beam, lit = right, x >= right
				}
				glow := math.Exp(-math.Pow((x-beam)/4, 2))
				c := pc.c
				switch {
				case lit:
					c.fg = mix(c.fg, colHot, glow)
				case glow > 0.05:
					c.fg = mix(colBG, colHot, glow)
				default:
					continue
				}
				g.set(pc.x, pc.y, c)
			}
		},
	},
	{
		name: "Slice",
		note: "the top half slides in from the left, the bottom half from the right, and they meet",
		dur:  0.8,
		draw: func(g *grid, s scene, t float64, _ *fxState) {
			off := float64(s.w) * (1 - outCubic(t/0.8))
			for _, pc := range s.logo {
				dx := -off
				if pc.y >= s.ly+s.bh/2 {
					dx = off
				}
				g.set(pc.x+int(math.Round(dx)), pc.y, pc.c)
			}
		},
	},
	{
		name: "Stack up",
		note: "cells rain down and pile up from the bottom row, like blocks filling a well",
		dur:  1.3,
		draw: func(g *grid, s scene, t float64, fx *fxState) {
			for i, pc := range s.logo {
				rowUp := 1 - float64(pc.y-s.ly)/float64(s.bh)
				delay := rowUp*0.7 + fx.rnd[i][0]*0.15
				q := span(t, delay, delay+0.4)
				if q == 0 {
					continue
				}
				y := int(math.Round(-1 + float64(pc.y+1)*q*q)) // falling under gravity
				g.set(pc.x, y, pc.c)
			}
		},
	},
	{
		name: "Spiral in",
		note: "the pieces swirl in around the centre and unwind into the logo",
		dur:  1.2,
		draw: func(g *grid, s scene, t float64, fx *fxState) {
			cx, cy := s.centre()
			for i, pc := range s.logo {
				delay := 0.2 * fx.rnd[i][0]
				q := span(t, delay, delay+1.0)
				if q == 0 {
					continue
				}
				e := outCubic(q)
				dx, dy := (float64(pc.x)-cx)/2, float64(pc.y)-cy // in square units
				a := 2.5 * math.Pi * (1 - e)
				sc := 0.2 + 0.8*e
				x := cx + 2*sc*(dx*math.Cos(a)-dy*math.Sin(a))
				y := cy + sc*(dx*math.Sin(a)+dy*math.Cos(a))
				c := pc.c
				if q < 1 {
					c.fg = mix(colDim, c.fg, e)
				}
				g.set(int(math.Round(x)), int(math.Round(y)), c)
			}
		},
	},
	{
		name: "Ignite",
		note: "a glowing front burns across left to right and leaves the logo cooling behind it",
		dur:  1.1,
		draw: func(g *grid, s scene, t float64, _ *fxState) {
			front := -4 + float64(s.bw+8)*inOutSine(t/1.1)
			for _, pc := range s.logo {
				d := front - float64(pc.x-s.lx)
				switch {
				case d > 2:
					c := pc.c
					c.fg = mix(colHot, c.fg, (d-2)/10)
					g.set(pc.x, pc.y, c)
				case d > 0:
					g.set(pc.x, pc.y, cell{r: []rune("▓▒")[int(d)], fg: colHot})
				}
			}
		},
	},
}

// inFlight is a piece on its way home, e of the way there: a dim dot far
// out, its own shape once close, so the logo forms while still moving
// instead of popping in at the end.
func inFlight(pc piece, e float64) cell {
	if e < 0.55 {
		return cell{r: '·', fg: mix(colDim, pc.c.fg, e)}
	}
	return cell{r: pc.c.r, fg: mix(colDim, pc.c.fg, (e-0.55)/0.45)}
}

// byReading is the logo's pieces top to bottom, left to right.
func (s scene) byReading() []piece { return s.reading }
