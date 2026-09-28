package intro

// The second 16: arcade games, demoscene effects and modern motion design,
// mixed. Same rules as the first set: at most 2 s, ease-out entrances, no
// full-screen flashes, any key skips, the final frame is the resting screen.

import (
	"fmt"
	"math"
	"slices"
)

func init() { intros = append(intros, more...) }

// tri is a triangle wave in [0, 1]: a ball bouncing between two walls.
func tri(v float64) float64 {
	v = math.Mod(v, 2)
	if v > 1 {
		return 2 - v
	}
	return v
}

// ball is the Pong ball's position at time t, inside the box.
func (s scene) ball(t float64) (float64, float64) {
	return float64(s.lx+2) + tri(t*2.3)*float64(s.bw-5), float64(s.ly+1) + tri(t*3.1+0.3)*float64(s.bh-3)
}

// wave is the amber palette as a ramp from 0 (dark) to 1 (white-hot).
func wave(v float64) rgb {
	switch v = clamp(v); {
	case v < 0.33:
		return mix(colBG, colDim, v/0.33)
	case v < 0.66:
		return mix(colDim, colAccent, (v-0.33)/0.33)
	default:
		return mix(colAccent, colHot, (v-0.66)/0.34)
	}
}

var more = []intro{
	{
		name: "Pong",
		note: "a ball rallies between two paddles and paints the logo wherever it passes",
		dur:  1.8,
		draw: func(g *grid, s scene, t float64, fx *fxState) {
			// When the ball first came near each piece.
			first := fx.memo("pong", func() []float64 {
				out := make([]float64, len(s.logo))
				for i := range out {
					out[i] = 99
				}
				for k := range 300 {
					bt := float64(k) / 300 * 1.4
					bx, by := s.ball(bt)
					for i, pc := range s.logo {
						if dx, dy := (float64(pc.x)-bx)/2, float64(pc.y)-by; dx*dx+dy*dy < 9 && bt < out[i] {
							out[i] = bt
						}
					}
				}
				return out
			})
			rest := span(t, 1.4, 1.8) // what the ball missed fades in
			for i, pc := range s.logo {
				c := pc.c
				switch {
				case t >= first[i]:
					c.fg = mix(colHot, c.fg, (t-first[i])/0.4)
				case rest > 0:
					c.fg = mix(colBG, c.fg, rest)
				default:
					continue
				}
				g.set(pc.x, pc.y, c)
			}
			if t < 1.4 {
				bx, by := s.ball(t)
				g.set(int(bx), int(by), cell{r: '█', fg: colHot})
				for d := -1; d <= 1; d++ {
					g.set(s.lx-3, int(by)+d, cell{r: '▐', fg: colAccent})
					g.set(s.lx+s.bw+2, int(by)+d, cell{r: '▌', fg: colAccent})
				}
			}
		},
	},
	{
		name: "Pac-Man",
		note: "a chomping Pac-Man eats a row of dots across the box, a ghost on his tail, and leaves the logo behind",
		dur:  1.9,
		draw: func(g *grid, s scene, t float64, fx *fxState) {
			_, cy := s.centre()
			px := float64(s.lx-8) + float64(s.bw+16)*span(t, 0, 1.6)
			for _, pc := range s.logo {
				if d := px - float64(pc.x); d > 0 {
					c := pc.c
					c.fg = mix(colHot, c.fg, d/12)
					g.set(pc.x, pc.y, c)
				}
			}
			for x := int(px) + 4; x < s.lx+s.bw+6; x += 3 {
				g.set(x, int(cy), cell{r: '·', fg: colDim})
			}
			if t < 1.6 {
				open := [3]string{" ▄███▄", "████▀ ", " ▀███▀"}
				if fx.frame/3%2 == 0 {
					open = [3]string{" ▄███▄", "██████", " ▀███▀"}
				}
				for r, row := range open {
					g.text(int(px)-3, int(cy)-1+r, row, colTop)
				}
				for r, row := range [3]string{"▄██▄", "█▌▐█", "▀▀▀▀"} {
					g.text(int(px)-14, int(cy)-1+r, row, colDim)
				}
			}
		},
	},
	{
		name: "Tetris",
		note: "the logo falls in as blocks, one column band at a time, stepping down row by row and locking into the well",
		dur:  1.9,
		draw: func(g *grid, s scene, t float64, fx *fxState) {
			order := fx.memo("tetris", func() []float64 {
				bands := (s.bw + 5) / 6
				perm := make([]float64, bands)
				for i := range perm {
					perm[i] = float64(i)
				}
				for i := range perm { // shuffle, repeatable
					j := int(hash(i, 77) * float64(len(perm)))
					perm[i], perm[j] = perm[j], perm[i]
				}
				return perm
			})
			for _, pc := range s.logo {
				if pc.kind == kindBorder && (pc.y > s.ly || pc.x == s.lx || pc.x == s.lx+s.bw-1) {
					g.set(pc.x, pc.y, cell{r: pc.c.r, fg: colDim}) // the well: sides and floor
				}
			}
			for _, pc := range s.logo {
				band := (pc.x - s.lx) / 6
				start := order[min(band, len(order)-1)] / float64(len(order)) * 1.3
				q := span(t, start, start+0.35)
				if q == 0 {
					continue
				}
				fall := int(float64(s.bh+2) * (1 - q)) // whole rows: blocks step down
				c := pc.c
				if q == 1 {
					c.fg = mix(colHot, pc.c.fg, span(t, start+0.35, start+0.6))
				}
				g.set(pc.x, pc.y-fall, c)
			}
		},
	},
	{
		name: "Invaders",
		note: "the logo marches in as an invader formation, stepping side to side and down, then lands in place",
		dur:  1.7,
		draw: func(g *grid, s scene, t float64, fx *fxState) {
			step := int(t / 0.12) // invaders move in steps, not smoothly
			settle := 1 - span(t, 0.2, 1.4)
			dx := int(math.Round(float64([]int{-4, -2, 0, 2, 4, 2, 0, -2}[step%8]) * settle))
			dy := -int(math.Round(8 * settle))
			for _, pc := range s.logo {
				c := pc.c
				if settle > 0 {
					c.fg = mix(colDim, c.fg, 1-settle)
				}
				g.set(pc.x+dx, pc.y+dy, c)
			}
			if t < 1.4 {
				cx, _ := s.centre()
				ship := int(cx) + int(10*math.Sin(t*5))
				g.text(ship-1, s.ly+s.bh+2, "▄█▄", colAccent)
				if shot := math.Mod(t, 0.35) / 0.35; shot < 0.8 {
					g.set(ship, s.ly+s.bh+1-int(shot*float64(s.bh)), cell{r: '│', fg: colHot})
				}
			}
		},
	},
	{
		name: "Snake",
		note: "a snake winds through the logo column by column, lighting everything it has passed",
		dur:  1.8,
		draw: func(g *grid, s scene, t float64, fx *fxState) {
			path := slices.Clone(s.logo)
			slices.SortFunc(path, func(a, b piece) int { // down one column, up the next
				if a.x != b.x {
					return a.x - b.x
				}
				if (a.x-s.lx)%2 == 0 {
					return a.y - b.y
				}
				return b.y - a.y
			})
			n := int(float64(len(path)) * outCubic(span(t, 0, 1.6)))
			for k, pc := range path[:n] {
				c := pc.c
				if body := n - k; body < 30 {
					c = cell{r: '█', fg: mix(colHot, colAccent, float64(body)/30)}
				}
				g.set(pc.x, pc.y, c)
			}
		},
	},
	{
		name: "Plasma",
		note: "a demoscene plasma boils inside the box, then cools and leaves the logo standing in it",
		dur:  1.7,
		draw: func(g *grid, s scene, t float64, _ *fxState) {
			heat := 1 - span(t, 0.8, 1.6)
			show := span(t, 0.5, 1.2)
			for y := s.ly + 1; y < s.ly+s.bh-1; y++ {
				for x := s.lx + 1; x < s.lx+s.bw-1; x++ {
					fx, fy := float64(x), float64(y)
					v := (math.Sin(fx/6+t*3) + math.Sin(fy/2.5-t*2) + math.Sin((fx+fy*2)/9+t*1.5) + 3) / 6
					c, ok := s.at[[2]int{x, y}]
					if !ok || show == 0 {
						c = cell{r: ' '}
					} else {
						c.fg = mix(colBG, c.fg, show)
					}
					if heat > 0 {
						c.bg, c.hasB = mix(colBG, wave(v*0.8), heat), true
					}
					g.set(x, y, c)
				}
			}
			for _, pc := range s.logo {
				if pc.kind == kindBorder {
					g.set(pc.x, pc.y, pc.c)
				}
			}
		},
	},
	{
		name: "Fire",
		note: "the classic demoscene fire rises from the bottom of the box, then dies down to reveal the logo",
		dur:  1.8,
		draw: func(g *grid, s scene, t float64, fx *fxState) {
			flame := span(t, 0, 0.5) * (1 - span(t, 0.9, 1.7))
			for y := s.ly + 1; y < s.ly+s.bh-1; y++ {
				up := float64(s.ly+s.bh-1-y) / float64(s.bh-2) // 0 at the floor, 1 at the top
				for x := s.lx + 1; x < s.lx+s.bw-1; x++ {
					flicker := hash(x, y+fx.frame) * 0.35
					heat := flame*1.3 - up + flicker - 0.15
					c, isLogo := s.at[[2]int{x, y}]
					switch {
					case heat > 0.05:
						r := []rune(" ░▒▓█")[min(int(heat*5), 4)]
						g.set(x, y, cell{r: r, fg: wave(0.35 + heat*0.6)})
					case isLogo && t > 0.9:
						c.fg = mix(colHot, c.fg, span(t, 0.9, 1.6))
						g.set(x, y, c)
					}
				}
			}
			for _, pc := range s.logo {
				if pc.kind == kindBorder {
					g.set(pc.x, pc.y, pc.c)
				}
			}
		},
	},
	{
		name: "Code rain",
		note: "glyphs pour down every column; where a drop passes, the logo locks in behind it",
		dur:  1.7,
		draw: func(g *grid, s scene, t float64, fx *fxState) {
			const noise = "01#%$&*+=<>/|:;{}"
			glyphs := []rune(noise)
			for x := s.lx - 2; x < s.lx+s.bw+2; x++ {
				speed := 22 + 18*hash(x, 5)
				head := float64(s.ly-8) - 10*hash(x, 6) + speed*t
				for k := range 8 {
					y := int(head) - k
					if y < s.ly-1 || y > s.ly+s.bh {
						continue
					}
					r := glyphs[int(hash(x*31+y, fx.frame/2)*float64(len(glyphs)))]
					fg := mix(colHot, colDim, float64(k)/8)
					g.set(x, y, cell{r: r, fg: mix(colBG, fg, 1-span(t, 1.3, 1.7))})
				}
				for y := s.ly; y < s.ly+s.bh && float64(y) < head-2; y++ {
					if c, ok := s.at[[2]int{x, y}]; ok {
						g.set(x, y, c)
					}
				}
			}
			if t > 1.3 { // anything the rain missed
				for _, pc := range s.logo {
					g.set(pc.x, pc.y, pc.c)
				}
			}
		},
	},
	{
		name: "Card flip",
		note: "the logo is a card: its patterned back spins to face you, then the front turns into place",
		dur:  1.1,
		draw: func(g *grid, s scene, t float64, _ *fxState) {
			a := math.Pi * (1 - outCubic(t/1.1)) // 180 degrees to 0
			k := math.Abs(math.Cos(a))
			cx, _ := s.centre()
			if a > math.Pi/2 { // the back: a dim diamond pattern
				for y := s.ly; y < s.ly+s.bh; y++ {
					for x := s.lx; x < s.lx+s.bw; x++ {
						if (x+y*2)%6 < 2 {
							nx := int(math.Round(cx + (float64(x)-cx)*k))
							g.set(nx, y, cell{r: '◆', fg: colDim})
						}
					}
				}
				return
			}
			for _, pc := range s.logo {
				nx := int(math.Round(cx + (float64(pc.x)-cx)*k))
				g.set(nx, pc.y, pc.c)
			}
		},
	},
	{
		name: "Synthwave",
		note: "an 80s neon grid races toward you under a striped sun, and the logo drops onto the horizon",
		dur:  2.0,
		draw: func(g *grid, s scene, t float64, _ *fxState) {
			fade := 1 - span(t, 1.4, 2.0)
			horizon := s.ly + s.bh/2
			cx, _ := s.centre()
			// The sun: a striped half disc above the horizon.
			for y := horizon - 7; y < horizon; y++ {
				if (horizon-y)%2 == 0 && y > horizon-4 {
					continue // the stripes cut through its lower half
				}
				dy := float64(horizon - y)
				half := math.Sqrt(math.Max(0, 49-dy*dy)) * 2
				for x := int(cx - half); x <= int(cx+half); x++ {
					g.set(x, y, cell{r: '█', fg: mix(colBG, mix(colTop, colBottom, 1-dy/7), fade)})
				}
			}
			// The grid: lines that race toward the viewer, and rays to the vanishing point.
			q := span(t, 0.5, 1.3)
			// Once the logo has landed, the box's inside is kept clear.
			covered := func(x, y int) bool {
				return q == 1 && x >= s.lx && x < s.lx+s.bw && y >= s.ly && y < s.ly+s.bh
			}
			for i := range 12 {
				z := math.Mod(float64(i)-t*6, 12) + 0.8
				y := horizon + int(18/z)
				for x := 0; x < s.w; x++ {
					if !covered(x, y) {
						g.set(x, y, cell{r: '─', fg: mix(colBG, colDim, fade)})
					}
				}
			}
			for k := -12; k <= 12; k++ {
				for y := horizon + 1; y < s.h; y++ {
					if x := int(cx + float64(k*(y-horizon))*1.6); !covered(x, y) {
						g.set(x, y, cell{r: '·', fg: mix(colBG, colAccent, fade*0.8)})
					}
				}
			}
			if q > 0 {
				drop := int(math.Round(-float64(s.ly+s.bh) * (1 - outBack(q))))
				for _, pc := range s.logo {
					g.set(pc.x, pc.y+drop, pc.c)
				}
			}
		},
	},
	{
		name: "Installer",
		note: "a 90s setup program copies files with a progress bar, then the picture flicks on",
		dur:  1.8,
		draw: func(g *grid, s scene, t float64, fx *fxState) {
			files := []string{"quarter.exe", "arcade.dat", "coins.ini", "joystick.drv", "cabinet.vga", "hiscore.tbl"}
			p := outCubic(span(t, 0, 1.2))
			if t < 1.25 {
				cx, cy := s.centre()
				x := int(cx) - 20
				g.text(x, int(cy)-2, "QUARTER Setup", colAccent)
				g.text(x, int(cy)-1, "Copying "+files[int(p*float64(len(files)-1))]+"...", colDim)
				bar := int(p * 34)
				for i := range 34 {
					r := '░'
					if i < bar {
						r = '█'
					}
					g.set(x+i, int(cy)+1, cell{r: r, fg: colAccent})
				}
				g.text(x+36, int(cy)+1, fmt.Sprintf("%3d%%", int(p*100)), colText)
				return
			}
			rows := float64(s.bh) * outCubic(span(t, 1.25, 1.6))
			for _, pc := range s.logo {
				if float64(pc.y-s.ly) <= rows {
					g.set(pc.x, pc.y, pc.c)
				}
			}
		},
	},
	{
		name: "Shockwave",
		note: "a ring bursts out from the centre, and the logo appears in its wake, briefly pushed outward",
		dur:  1.1,
		draw: func(g *grid, s scene, t float64, _ *fxState) {
			cx, cy := s.centre()
			// The ring reaches the box's corners (in square units) as the intro ends.
			r := outCubic(t/1.1) * (math.Hypot(float64(s.bw)/4, float64(s.bh)/2) + 2)
			for _, pc := range s.logo {
				dx, dy := (float64(pc.x)-cx)/2, float64(pc.y)-cy
				d := math.Hypot(dx, dy)
				edge := math.Abs(d - r)
				switch {
				case d > r+1:
					continue
				case edge < 2: // on the ring: hot, pushed out a cell
					push := 1.0
					if d > 0 {
						g.set(pc.x+int(math.Round(dx/d*push*2)), pc.y+int(math.Round(dy/d*push)), cell{r: pc.c.r, fg: colHot})
					}
				default:
					g.set(pc.x, pc.y, pc.c)
				}
			}
		},
	},
	{
		name: "Liquid fill",
		note: "the box fills with molten amber, a wave on its surface, and drains away to leave the logo",
		dur:  1.6,
		draw: func(g *grid, s scene, t float64, _ *fxState) {
			level := float64(s.bh-2) * outCubic(span(t, 0, 1.0))
			drain := span(t, 1.0, 1.6)
			for y := s.ly + 1; y < s.ly+s.bh-1; y++ {
				for x := s.lx + 1; x < s.lx+s.bw-1; x++ {
					surface := float64(s.ly+s.bh-1) - level + math.Sin(float64(x)/5+t*9)*0.8*(1-span(t, 0.8, 1.0))
					if float64(y) < surface {
						continue
					}
					c, ok := s.at[[2]int{x, y}]
					if !ok {
						c = cell{r: ' '}
					}
					if drain < 1 {
						c.bg, c.hasB = mix(mix(colBottom, colBG, 0.55), colBG, drain), true
					}
					if float64(y) < surface+1 {
						c.bg = mix(colHot, colBG, drain) // the bright surface line
					}
					g.set(x, y, c)
				}
			}
			for _, pc := range s.logo {
				if pc.kind == kindBorder {
					g.set(pc.x, pc.y, pc.c)
				}
			}
		},
	},
	{
		name: "Neon",
		note: "each letter is a neon tube that stutters on, one after another; small areas, well under 3 flashes a second each",
		dur:  1.8,
		draw: func(g *grid, s scene, t float64, _ *fxState) {
			letters := 8
			for _, pc := range s.logo {
				if pc.kind == kindBorder {
					c := pc.c
					c.fg = mix(colBG, c.fg, span(t, 1.4, 1.8))
					g.set(pc.x, pc.y, c)
					continue
				}
				k := min((pc.x-s.lx)*letters/s.bw, letters-1)
				on := 0.1 + float64((k*5)%letters)*0.14 // a scattered order
				c := pc.c
				switch q := t - on; {
				case q < 0:
					c.fg = mix(colBG, colDim, 0.35) // the dark tube
				case q < 0.12, q > 0.2 && q < 0.26:
					// lit
				case q < 0.2:
					c.fg = mix(colBG, colDim, 0.5) // the stutter
				}
				g.set(pc.x, pc.y, c)
			}
		},
	},
	{
		name: "Firework",
		note: "a rocket climbs to the centre and bursts; its sparks arc out and land as the logo",
		dur:  1.8,
		draw: func(g *grid, s scene, t float64, fx *fxState) {
			cx, cy := s.centre()
			if t < 0.5 {
				y := float64(s.h) - (float64(s.h)-cy)*outCubic(t/0.5)
				g.set(int(cx), int(y), cell{r: '▲', fg: colHot})
				g.set(int(cx), int(y)+1, cell{r: '│', fg: colAccent})
				g.set(int(cx), int(y)+2, cell{r: '·', fg: colDim})
				return
			}
			for i, pc := range s.logo {
				delay := 0.5 + 0.2*fx.rnd[i][0]
				q := span(t, delay, delay+1.0)
				if q == 0 {
					continue
				}
				e := outCubic(q)
				x := cx + (float64(pc.x)-cx)*e
				y := cy + (float64(pc.y)-cy)*e - math.Sin(math.Pi*e)*4 // the arc
				c := pc.c
				if q < 1 {
					c = inFlight(pc, e)
					c.fg = mix(colHot, c.fg, e)
				}
				g.set(int(math.Round(x)), int(math.Round(y)), c)
			}
		},
	},
	{
		name: "Zoom punch",
		note: "the logo slams in from huge to its size with a small overshoot, like a modern title card",
		dur:  0.9,
		draw: func(g *grid, s scene, t float64, _ *fxState) {
			k := 4 - 3*outBack(t/0.9) // 4x down to 1x, overshooting a touch
			cx, cy := s.centre()
			for y := range s.h {
				for x := range s.w {
					sx := int(math.Round(cx + (float64(x)-cx)/k))
					sy := int(math.Round(cy + (float64(y)-cy)/k))
					if c, ok := s.at[[2]int{sx, sy}]; ok {
						c.fg = mix(colHot, c.fg, t/0.9)
						g.set(x, y, c)
					}
				}
			}
		},
	},
}

// memo computes a per-run value once and keeps it for the rest of the run.
func (fx *fxState) memo(key string, f func() []float64) []float64 {
	if v, ok := fx.cache[key]; ok {
		return v
	}
	if fx.cache == nil {
		fx.cache = map[string][]float64{}
	}
	v := f()
	fx.cache[key] = v
	return v
}
