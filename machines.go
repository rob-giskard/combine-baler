package main

// Machine extras: the truck boost and the tyre tracks it presses into the
// wheat, mud and the wash hose, horns, and the kite.

import "math"

const (
	tapX, tapY = 359.0 + OX, 92.0 + OY // stand here to take the hose
	nozX, nozY = 352.0 + OX, 84.0 + OY // where the hose leaves the reel
	HoseLen    = 72.0                  // about four truck lengths
	BoostTime  = 2.5                   // seconds of lingering speed-up
	DashTime   = 0.6                   // the first part of it: the dash
	BoostCD    = 4.0
	EngineMax  = 3
	EngineStep = 0.12 // top speed per engine level
	BigBedCap  = 14
	HornGap    = 0.4
	MudCombine = 90.0  // seconds of reaping to clog a combine
	MudTruck   = 150.0 // seconds of driving on the field to clog a truck
	WashTime   = 3.0   // seconds of spraying to clean a clogged machine
	SprayRange = 26.0
)

// bakeHoseStation puts a tap and a hose reel against the barn's left wall.
func (g *Game) bakeHoseStation() {
	g.shade(345, 86, 364, 88)
	g.oRect(356, 80, 364, 86, 0x8A8A94, true) // stone footing
	g.oRect(357, 81, 363, 85, 0xA8A8B4, true)
	g.oRect(359, 70, 362, 81, 0x7E7E8A, true) // standpipe up the barn wall
	g.oRect(359, 70, 360, 81, 0xB8B8C4, true)
	g.oRect(357, 74, 360, 76, 0xD8A830, true) // brass tap
	g.oRect(357, 76, 358, 78, 0xD8A830, true)
	g.oRect(358, 72, 361, 74, 0xC8402A, true) // red handle
	g.oDisc(350, 80, 5, 0x2E8A3A, true)       // hose reel
	g.oDisc(350, 80, 3, 0x45B055, true)
	g.oDisc(350, 80, 1, 0x1E5A26, true)
}

// ---------------------------------------------------------------- speed

func (g *Game) truckCap() int {
	if g.BigBed {
		return BigBedCap
	}
	return TruckCap
}

// mudFactor: no effect below half full, down to 55% top speed when clogged.
func mudFactor(v *Vehicle) float64 {
	return 1 - 0.45*math.Max(0, math.Min(1, (v.mud-0.5)/0.5))
}

func (g *Game) speedMult(v *Vehicle) float64 {
	m := (1 + EngineStep*float64(g.Engine)) * mudFactor(v)
	if v.driver >= 0 {
		p := &g.pl[v.driver]
		m *= 1 - 0.3*p.tired // sleepy farmers drive gently
		if p.bun {
			m *= 1.1
		}
	}
	return m
}

// ---------------------------------------------------------------- boost

func (g *Game) tryBoost(vi int) {
	v := &g.veh[vi]
	if v.kind != VTruck {
		return
	}
	if v.boostCD > 0 || v.mud >= 1 {
		if v.mud >= 1 {
			g.Snd.Play(SNope)
		}
		return
	}
	v.boost, v.boostCD = BoostTime, BoostCD
	g.Snd.Play(SBoost)
	s, c := math.Sin(v.ang), math.Cos(v.ang)
	ex, ey := v.x-3.5*c+9*s, v.y-3.5*s-9*c
	for k := 0; k < 18; k++ { // a big puff out of the exhaust and behind
		a := g.rnd() * 2 * math.Pi
		sp := 6 + g.rnd()*14
		g.parts = append(g.parts, Particle{x: ex + (g.rnd()-0.5)*3, y: ey + (g.rnd()-0.5)*3,
			vx: math.Cos(a)*sp - s*12, vy: math.Sin(a)*sp + c*12 - 3,
			life: 0.8 + g.rnd()*0.7, size: 2 + g.rnd()*2, c: []uint32{0xE8E8EC, 0xC8C8D0, 0xA8A8B4}[k%3]})
	}
}

func (v *Vehicle) boostFrac() float64 { return math.Max(0, v.boost) / BoostTime }

// tracks presses the wheat under a dashing truck's rear wheels flat.
func (g *Game) tracks(v *Vehicle) {
	s, c := math.Sin(v.ang), math.Cos(v.ang)
	for _, side := range [2]float64{-4.5, 4.5} {
		wx := v.x + side*c - 5.2*s
		wy := v.y + side*s + 5.2*c
		for dy := -1; dy <= 1; dy++ {
			for dx := -1; dx <= 1; dx++ {
				x, y := int(math.Floor(wx))+dx, int(math.Floor(wy))+dy
				if g.tileAt(x, y) != TWheat {
					continue
				}
				g.tile[y*WW+x] = TFlat
				g.Flat++
				for d := -2; d <= 2; d++ {
					g.recolor(x, y+d)
				}
			}
		}
	}
}

// ---------------------------------------------------------------- mud

// muddy builds mud on a machine working the field.
func (g *Game) muddy(v *Vehicle, dt float64, reaping bool) {
	if v.kind == VCombine {
		if reaping {
			v.mud += dt / MudCombine
		}
	} else if t := g.tileAt(int(v.x), int(v.y)); t == TWheat || t == TStubble || t == TFlat {
		v.mud += dt * math.Min(1, math.Abs(v.speed)/46) / MudTruck
	}
	v.mud = math.Min(1, v.mud)
}

func (g *Game) hoseFree() bool {
	for i := range g.pl {
		if g.pl[i].hose {
			return false
		}
	}
	return true
}

func (g *Game) hangHose(i int, snapped bool) {
	p := &g.pl[i]
	p.hose = false
	if snapped {
		g.Snd.Play(SSnap)
		g.say("THE HOSE SPRANG BACK TO THE TAP")
	} else {
		g.Snd.Play(SDoor)
	}
}

// spray sends water where the farmer faces and washes any machine it hits.
func (g *Game) spray(i int, dt float64) {
	p := &g.pl[i]
	hx, hy := p.x+p.fx*2, p.y-2+p.fy*2
	for k := 0; k < 3; k++ {
		sp := 34 + g.rnd()*18
		jx, jy := (g.rnd()-0.5)*0.35, (g.rnd()-0.5)*0.35
		g.parts = append(g.parts, Particle{x: hx, y: hy,
			vx: (p.fx + jx - p.fy*jx) * sp, vy: (p.fy + jy + p.fx*jy) * sp,
			life: 0.35 + g.rnd()*0.2, size: 1, c: []uint32{0x9FDDEE, 0xD8F4FA, 0x6FB8D8}[k]})
	}
	p.sprayT -= dt
	if p.sprayT <= 0 {
		p.sprayT = 0.3
		g.Snd.Play(SSpray)
	}
	for j := range g.veh {
		v := &g.veh[j]
		dx, dy := v.x-p.x, v.y-p.y
		along := dx*p.fx + dy*p.fy
		across := math.Abs(dx*p.fy - dy*p.fx)
		if along < 0 || along > SprayRange || across > 6+along*0.4 {
			continue
		}
		was := v.mud
		v.mud = math.Max(0, v.mud-dt/WashTime)
		if was > 0 && v.mud == 0 {
			v.sparkle = 1.2
			g.Snd.Play(SSparkle)
		}
	}
	for j := range g.balls { // a good soak also pushes the ball along
		b := &g.balls[j]
		dx, dy := b.x-p.x, b.y-p.y
		if along := dx*p.fx + dy*p.fy; along > 0 && along < SprayRange && math.Abs(dx*p.fy-dy*p.fx) < 5 {
			b.vx += p.fx * 40 * dt
			b.vy += p.fy * 40 * dt
		}
	}
}

// ---------------------------------------------------------------- horn

func (g *Game) honk(vi int) {
	v := &g.veh[vi]
	if v.hornCD > 0 {
		return
	}
	v.hornCD = HornGap
	if v.kind == VCombine {
		g.Snd.Play(SHornCombine)
	} else {
		g.Snd.Play(SHornTruck)
	}
	g.parts = append(g.parts, Particle{x: v.x, y: v.y, life: 0.6, size: 16, c: 0xFFF4D6, ring: true})
	for j := range g.pl { // farmers nearby jump
		q := &g.pl[j]
		if q.veh < 0 && math.Hypot(q.x-v.x, q.y-v.y) < 30 && q.hop <= 0 {
			q.hop = 0.4
		}
	}
}

// ---------------------------------------------------------------- per step

// machines advances the timers and effects of every vehicle.
func (g *Game) machines(dt float64) {
	g.wind += dt * 0.05
	for j := range g.veh {
		v := &g.veh[j]
		if v.driver < 0 && (v.speed != 0 || v.kx != 0 || v.ky != 0 || v.spin != 0) {
			g.drive(j, Input{}, dt) // still rolling, or knocked about
		}
		v.knock(dt)
		if v.boost > 0 {
			if v.boost > BoostTime-DashTime {
				g.tracks(v)
			}
			v.boost -= dt
		}
		if v.boostCD > 0 {
			v.boostCD -= dt
		}
		if v.hornCD > 0 {
			v.hornCD -= dt
		}
		if v.sparkle > 0 {
			v.sparkle -= dt
			if g.rnd() < dt*14 {
				g.parts = append(g.parts, Particle{x: v.x + (g.rnd()-0.5)*14, y: v.y + (g.rnd()-0.5)*14,
					vy: -3, life: 0.4, size: 1, c: 0xFFFFFF})
			}
		}
	}
	for i := range g.pl {
		p := &g.pl[i]
		if p.hose && math.Hypot(p.x-nozX, p.y-nozY) > HoseLen {
			g.hangHose(i, true)
		}
		if p.kite {
			g.flyKite(p, dt)
		}
	}
	g.treeSprings(dt)
}

// ---------------------------------------------------------------- kite

func (g *Game) flyKite(p *Player, dt float64) {
	ax, ay := g.pos(p)
	wx, wy := -math.Cos(g.wind), 0.35*math.Sin(g.wind*1.7)
	p.kiteT += dt
	tx := ax + wx*30 + math.Sin(p.kiteT*0.9)*6
	ty := ay + wy*30 + math.Cos(p.kiteT*1.3)*4
	k := math.Min(1, dt*1.5)
	p.kiteX += (tx - p.kiteX) * k
	p.kiteY += (ty - p.kiteY) * k
}

var kitePaint = [3][2]uint32{{0xE0402E, 0xFFD23E}, {0x3A7BD5, 0xF4F0E0}, {0x2FA07E, 0xF4842E}}

const kiteHigh = 24.0 // how far up the kite flies, drawn as a lift on screen

func (g *Game) drawKites(cm cam) {
	for i := range g.pl {
		p := &g.pl[i]
		if !p.kite {
			continue
		}
		ax, ay := g.pos(p)
		ay -= 3
		kx, ky := p.kiteX, p.kiteY-kiteHigh
		// the string, sagging a little
		for t := 0.0; t <= 1; t += 0.02 {
			x := ax + (kx-ax)*t
			y := ay + (ky-ay)*t + math.Sin(t*math.Pi)*4
			px, py := int(cm.sx(x)), int(cm.sy(y))
			g.rectA(px, py, px+1, py+1, 0xFFF4D6, 170)
		}
		// shadow on the ground
		sx, sy := cm.sx(p.kiteX), cm.sy(p.kiteY)
		g.rectA(int(sx)-2, int(sy)-1, int(sx)+2, int(sy)+1, 0x2A2A10, 70)
		// tail bows
		pt := kitePaint[i%3]
		for b := 1; b <= 4; b++ {
			x := kx + float64(b)*1.8 + math.Sin(p.kiteT*4+float64(b))*1.2
			y := ky + 3 + float64(b)*2.2
			px, py := int(cm.sx(x)), int(cm.sy(y))
			g.rect(px-1, py, px+2, py+1, pt[b%2])
		}
		// the diamond
		h := math.Max(3.2*cm.z, 5)
		cx, cy := cm.sx(kx), cm.sy(ky)
		for py := int(cy - h); py <= int(cy+h); py++ {
			w := h*0.7*(1-math.Abs(float64(py)-cy)/h) + 0.5
			for px := int(cx - w); px <= int(cx+w); px++ {
				c := pt[0]
				if (float64(px) < cx) != (float64(py) < cy) {
					c = pt[1]
				}
				g.rect(px, py, px+1, py+1, c)
			}
		}
	}
}

// ---------------------------------------------------------------- hose art

func (g *Game) drawHoses(cm cam) {
	for i := range g.pl {
		p := &g.pl[i]
		if !p.hose {
			continue
		}
		hx, hy := p.x+p.fx*1.5, p.y-1.5
		d := math.Hypot(hx-nozX, hy-nozY)
		sag := math.Max(0, HoseLen-d) * 0.12
		sz := max(1, int(cm.z*0.7))
		for t := 0.0; t <= 1; t += 0.5 / math.Max(d, 1) {
			x := nozX + (hx-nozX)*t
			y := nozY + (hy-nozY)*t + math.Sin(t*math.Pi)*sag
			px, py := int(cm.sx(x)), int(cm.sy(y))
			g.rect(px, py, px+sz, py+sz, 0x2E8A3A)
		}
	}
}

// ---------------------------------------------------------------- gauge

// mudClock draws the mud gauge: a brown disc filling clockwise from twelve.
func (g *Game) mudClock(cx, cy, r int, frac float64) {
	fr := float64(r)
	for py := cy - r - 1; py <= cy+r+1; py++ {
		for px := cx - r - 1; px <= cx+r+1; px++ {
			dx, dy := float64(px-cx)+0.5, float64(py-cy)+0.5
			d := math.Hypot(dx, dy)
			switch {
			case d > fr+1:
				continue
			case d > fr:
				g.rect(px, py, px+1, py+1, 0x3A2410)
				continue
			}
			a := math.Atan2(dx, -dy)
			if a < 0 {
				a += 2 * math.Pi
			}
			c := uint32(0xE8D8B8)
			if a <= frac*2*math.Pi {
				c = 0x8A5A2A
				if frac >= 1 { // clogged: blinks
					c = [2]uint32{0x5E3A18, 0xC8402A}[int(g.clock*3)%2]
				}
			}
			g.rect(px, py, px+1, py+1, c)
		}
	}
	g.rect(cx, cy-r+1, cx+1, cy, 0x3A2410) // the hand at twelve
}
