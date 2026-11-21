package main

// Autumn trees. The trunk is baked into the ground layer and blocks; the
// crown is drawn over the machines as a canopy and fades when something is
// under it. With Bouncy collisions (the default) the crown is also a soft
// spring that pushes machines and farmers back out.

import "math"

type Tree struct {
	x, y float64 // crown centre
	r    float64 // crown radius
	pal  int
	fade float64 // 1 opaque .. 0.4 when someone is underneath
}

// baseY is where the trunk meets the ground.
func (t *Tree) baseY() float64 { return t.y + t.r*1.5 }

// bumper is the soft spring area under the crown (centre and radius).
func (t *Tree) bumper() (x, y, r float64) { return t.x, t.baseY() - t.r*0.55, t.r * 0.85 }

// leaf colours: main, light, highlight, dark
var leafPal = [4][4]uint32{
	{0xE8581C, 0xF47A2E, 0xFF9A48, 0xC8421A}, // orange
	{0xC8302A, 0xE0483A, 0xF47060, 0x9A2020}, // red
	{0xE8B020, 0xF6C83A, 0xFFE070, 0xC08A18}, // gold
	{0xA8552A, 0xC0703A, 0xD89050, 0x7E3C1E}, // russet
}

// plantTrees puts the yard trees back and lines the grass lanes with more.
// Trunks only ever stand on grass; crowns may hang over the wheat.
func (g *Game) plantTrees() {
	g.trees = nil
	add := func(x, y, r float64, pal int) {
		t := Tree{x: x, y: y, r: r, pal: pal, fade: 1}
		bx0, bx1 := int(x-r*0.22), int(x+r*0.25)+1
		by0, by1 := int(t.baseY()-r*0.35), int(t.baseY())+1
		for yy := by0 - 1; yy <= by1; yy++ {
			for xx := bx0 - 1; xx <= bx1; xx++ {
				if g.tileAt(xx, yy) != TGrass || g.kind[yy*WW+xx] != CNone {
					return // not on open grass: skip this one
				}
			}
		}
		for yy := int(y + r*0.75); yy < by1; yy++ {
			for xx := bx0; xx < bx1; xx++ {
				c := uint32(0x7A4A2A)
				if xx == bx0 {
					c = 0x5E3820
				} else if xx == bx1-1 {
					c = 0x96603A
				}
				g.obj[yy*WW+xx] = c
				if yy >= by0 {
					g.kind[yy*WW+xx] = CTree
				}
			}
		}
		g.trees = append(g.trees, t)
	}
	// the yard: one by the barn, a little orchard behind it
	for k, t := range [][2]float64{{460, 48}, {290, -40}, {335, -62}, {392, -48}, {455, -66}, {512, -38}, {268, 60}, {530, 70}} {
		add(t[0]+OX, t[1]+OY, 13, k%4)
	}
	// yard reports whether the plots in columns c0..c1, rows r0..r1 include the farmstead
	yard := func(c0, c1, r0, r1 int) bool { return YardC >= c0 && YardC <= c1 && YardR >= r0 && YardR <= r1 }
	size := [3]float64{12, 15, 18}
	plant := func(x, base float64, h uint32) {
		r := size[h%3]
		add(x, base-r*1.5, r, int(h>>4%4))
	}
	// where the grass lanes cross
	for r := 1; r < GridH; r++ {
		for c := 1; c < GridW; c++ {
			if yard(c-1, c, r-1, r) {
				continue
			}
			x := float64(Margin + c*(FieldW+Lane) - Lane/2)
			y := float64(Margin + r*(FieldH+Lane) - Lane/2)
			plant(x, y+3, hash(c, r*7))
		}
	}
	// along the lanes, every so often
	for r := 0; r < GridH; r++ {
		for c := 1; c < GridW; c++ { // lanes running north-south
			h := hash(c*13, r*5+1)
			if h%3 == 0 || yard(c-1, c, r, r) {
				continue
			}
			x := float64(Margin + c*(FieldW+Lane) - Lane/2)
			y := float64(Margin + r*(FieldH+Lane) + FieldH/3 + int(h>>8%uint32(FieldH/3)))
			plant(x, y, h>>3)
		}
	}
	for r := 1; r < GridH; r++ {
		for c := 0; c < GridW; c++ { // lanes running east-west
			h := hash(c*17+3, r*11)
			if h%3 == 0 || yard(c, c, r-1, r) {
				continue
			}
			x := float64(Margin + c*(FieldW+Lane) + FieldW/4 + int(h>>8%uint32(FieldW/2)))
			y := float64(Margin + r*(FieldH+Lane) - Lane/2)
			plant(x, y+3, h>>3)
		}
	}
}

func (g *Game) nearestTree(x, y float64) *Tree {
	var best *Tree
	bd := 1e18
	for i := range g.trees {
		t := &g.trees[i]
		if d := math.Hypot(x-t.x, y-t.baseY()); d < bd {
			best, bd = t, d
		}
	}
	return best
}

// treeSprings pushes machines and farmers out of the soft crowns (Bouncy
// only) and fades crowns over anything underneath.
func (g *Game) treeSprings(dt float64) {
	for i := range g.trees {
		t := &g.trees[i]
		under := false
		bx, by, br := t.bumper()
		for j := range g.veh {
			v := &g.veh[j]
			dx, dy := v.x-bx, v.y-by
			d := math.Hypot(dx, dy)
			if d < t.r+vehRadius[v.kind]+4 {
				under = true
			}
			if !g.bouncy() {
				continue
			}
			if pen := br + vehRadius[v.kind]*0.6 - d; pen > 0 && d > 0.01 {
				f := math.Min(pen, 12) * 34 * dt
				v.kx += dx / d * f
				v.ky += dy / d * f
				if v.bumpT <= 0 && math.Abs(v.speed) > 12 && pen < 3 {
					g.Snd.Play(SBoing)
					g.leafBurst(v.x-dx/d*5, v.y-dy/d*5, 4)
					v.bumpT = 0.6
				}
			}
		}
		for j := range g.pl {
			p := &g.pl[j]
			if p.veh >= 0 {
				continue
			}
			dx, dy := p.x-t.x, p.y-t.baseY()
			if math.Hypot(p.x-t.x, p.y-t.y) < t.r+3 {
				under = true
			}
			if d := math.Hypot(dx, dy); g.bouncy() && d < 5 && d > 0.01 && p.bounceT <= 0 {
				p.kx, p.ky = dx/d*38, dy/d*38
				p.hop, p.bounceT = 0.4, 0.5
				g.Snd.Play(SBoing)
				g.leafBurst(t.x, t.baseY()-t.r*0.4, 3)
			}
		}
		target := 1.0
		if under {
			target = 0.4
		}
		t.fade += (target - t.fade) * math.Min(1, dt*6)
	}
}

// leafBurst shakes a few leaves loose.
func (g *Game) leafBurst(x, y float64, n int) {
	for k := 0; k < n; k++ {
		pal := leafPal[int(g.rnd()*4)%4]
		g.parts = append(g.parts, Particle{x: x + (g.rnd()-0.5)*10, y: y + (g.rnd()-0.5)*6,
			vx: (g.rnd() - 0.5) * 14, vy: 4 + g.rnd()*8, life: 0.9 + g.rnd()*0.7, size: 1.2, c: pal[int(g.rnd()*3)]})
	}
}

// drawTreeShadows darkens the ground under the crowns (drawn before machines).
func (g *Game) drawTreeShadows(cm cam) {
	for i := range g.trees {
		t := &g.trees[i]
		cx, cy := cm.sx(t.x+2), cm.sy(t.baseY()-1)
		rx, ry := t.r*0.95*cm.z, t.r*0.45*cm.z
		if cx+rx < float64(g.clip0) || cx-rx > float64(g.clip1) || cy+ry < float64(g.clipY0) || cy-ry > float64(g.clipY1) {
			continue
		}
		for py := int(cy - ry); py <= int(cy+ry); py++ {
			for px := int(cx - rx); px <= int(cx+rx); px++ {
				dx, dy := (float64(px)+0.5-cx)/rx, (float64(py)+0.5-cy)/ry
				if dx*dx+dy*dy <= 1 {
					g.rectA(px, py, px+1, py+1, 0x2A2A10, 70)
				}
			}
		}
	}
}

// drawCanopy paints the crowns over everything on the ground.
func (g *Game) drawCanopy(cm cam) {
	for i := range g.trees {
		t := &g.trees[i]
		r := t.r * cm.z
		cx, cy := cm.sx(t.x), cm.sy(t.y)
		if cx+r < float64(g.clip0) || cx-r > float64(g.clip1) || cy+r < float64(g.clipY0) || cy-r > float64(g.clipY1) {
			continue
		}
		pal := leafPal[t.pal]
		a := uint32(255 * t.fade)
		type disc struct{ x, y, r float64 }
		ds := [4]disc{{0, 0, 1}, {-0.3, -0.3, 0.62}, {-0.55, -0.55, 0.24}, {0.55, 0.45, 0.38}}
		for py := max(int(cy-r), g.clipY0); py <= min(int(cy+r), g.clipY1-1); py++ {
			for px := max(int(cx-r), g.clip0); px <= min(int(cx+r), g.clip1-1); px++ {
				dx, dy := (float64(px)+0.5-cx)/r, (float64(py)+0.5-cy)/r
				col := uint32(0)
				for k, d := range ds {
					ex, ey := dx-d.x, dy-d.y
					if ex*ex+ey*ey <= d.r*d.r {
						col = pal[k]
					}
				}
				if col == 0 {
					continue
				}
				// a little leafy texture, fixed to the world
				wx, wy := int(math.Floor(t.x+dx*t.r)), int(math.Floor(t.y+dy*t.r))
				if h := hash(wx, wy) % 11; h == 0 {
					col = pal[3]
				} else if h == 1 {
					col = pal[1]
				}
				if a >= 250 {
					g.fb[py*SW+px] = col
				} else {
					g.fb[py*SW+px] = blend(g.fb[py*SW+px], col, a)
				}
			}
		}
	}
}
