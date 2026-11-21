package main

// The cosy layer: night and tiredness, selling bales at sundown, sleeping
// in the barn, the vending machine and the bouncy balls.

import "math"

const (
	NightLen  = DayLen * 10 / 14 // 20:00 - 06:00
	BalePrice = 1                // coins per bale at sundown
	MaxBalls  = 6
)

const (
	doorX, doorY = 400.0 + OX, 94.0 + OY // barn door: go to sleep
	shopX, shopY = 444.0 + OX, 96.0 + OY // in front of the vending machine
)

const (
	UseNone = iota
	UseVehicle
	UseDoor
	UseShop
)

const (
	ItemCoffee = iota
	ItemFizzy
	ItemBall
	ItemLamps
)

type Item struct {
	name  string
	price int
}

var items = [4]Item{{"COFFEE", 2}, {"FIZZY WATER", 1}, {"BOUNCY BALL", 4}, {"HEADLAMPS", 12}}

type Ball struct{ x, y, vx, vy, z, vz float64 }

func (g *Game) Night() bool { return g.T >= DayLen }

func (g *Game) pos(p *Player) (x, y float64) {
	if p.veh >= 0 {
		return g.veh[p.veh].x, g.veh[p.veh].y
	}
	return p.x, p.y
}

// nearThing finds what a farmer on foot could use right now.
func (g *Game) nearThing(p *Player) (kind, idx int, x, y float64) {
	best := 1e9
	if j := g.nearVehicle(p); j >= 0 {
		v := &g.veh[j]
		kind, idx, x, y, best = UseVehicle, j, v.x, v.y, math.Hypot(v.x-p.x, v.y-p.y)
	}
	if d := math.Hypot(doorX-p.x, doorY-p.y); d < 9 && d < best {
		kind, x, y, best = UseDoor, doorX, doorY-6, d
	}
	if d := math.Hypot(shopX-p.x, shopY-p.y); d < 9 && d < best {
		kind, x, y = UseShop, shopX, shopY-12
	}
	return
}

func (g *Game) interact(i int) {
	p := &g.pl[i]
	switch kind, j, _, _ := g.nearThing(p); kind {
	case UseVehicle:
		p.veh = j
		g.veh[j].driver = i
		g.Snd.Play(SDoor)
	case UseDoor:
		g.sleep()
	case UseShop:
		p.shop = true
		g.Snd.Play(SMenu)
	}
}

func (g *Game) shopInput(i int, in, last Input) {
	p := &g.pl[i]
	p.moving = false
	switch {
	case in.left && !last.left:
		p.shopSel = (p.shopSel + len(items) - 1) % len(items)
		g.Snd.Play(SMenu)
	case in.right && !last.right:
		p.shopSel = (p.shopSel + 1) % len(items)
		g.Snd.Play(SMenu)
	case (in.up && !last.up) || (in.down && !last.down):
		p.shop = false
	case in.act || in.use:
		g.buy(i)
	}
}

func (g *Game) buy(i int) {
	p := &g.pl[i]
	it := items[p.shopSel]
	if g.Coins < it.price || (p.shopSel == ItemLamps && g.Lamps) || (p.shopSel == ItemBall && len(g.balls) >= MaxBalls) {
		p.shake = 0.3
		g.Snd.Play(SNope)
		return
	}
	g.Coins -= it.price
	switch p.shopSel {
	case ItemCoffee: // only matters to a tired farmer
		p.tired = 0
		for k := 0; k < 6; k++ {
			g.parts = append(g.parts, Particle{x: p.x + (g.rnd()-0.5)*3, y: p.y - 5, vx: (g.rnd() - 0.5) * 2,
				vy: -5 - g.rnd()*3, life: 1 + g.rnd(), size: 1.2, c: 0xF4EEE6})
		}
		g.Snd.Play(SSip)
	case ItemFizzy:
		p.fizz = 25
		p.tired = math.Max(0, p.tired-0.2)
		g.Snd.Play(SFizz)
	case ItemBall:
		g.balls = append(g.balls, Ball{x: shopX, y: shopY + 2, vx: (g.rnd() - 0.5) * 12, vy: 14, vz: 24})
		g.Snd.Play(SBoing)
	case ItemLamps:
		g.Lamps = true
		g.Snd.Play(SSelect)
	}
}

// cosy advances the clock and everything that hangs off it.
func (g *Game) cosy(dt float64) {
	was := g.T
	g.T += dt
	if was < DayLen && g.T >= DayLen {
		g.cashOut()
	}
	if g.T >= DayLen+NightLen { // worked right through the night
		g.Log = append(g.Log, DayLog{g.Day, g.Tonnes(g.CutToday), g.Cashed})
		g.newDay(false)
		g.Snd.Play(SSelect)
	}
	g.hintT += dt
	if g.cashFlash > 0 {
		g.cashFlash -= dt
	}
	for i := range g.pl {
		p := &g.pl[i]
		x, y := g.pos(p)
		if g.Night() {
			p.tired = math.Min(1, p.tired+dt/160)
		}
		if p.shake > 0 {
			p.shake -= dt
		}
		if p.fizz > 0 {
			p.fizz -= dt
			if g.rnd() < dt*9 {
				g.parts = append(g.parts, Particle{x: x + (g.rnd()-0.5)*6, y: y - g.rnd()*4, vx: (g.rnd() - 0.5) * 2,
					vy: -6 - g.rnd()*4, life: 0.6 + g.rnd()*0.5, size: 1, c: 0xD8F4FA})
			}
		}
		if p.tired > 0.35 {
			p.yawnT -= dt
			if p.yawnT <= 0 {
				p.yawnT = 3.4 - 1.6*p.tired
				g.parts = append(g.parts, Particle{x: x + 2, y: y - 9, vx: 2.5, vy: -4, life: 1.8, c: 0xDDE6FF, ch: 'Z'})
				g.Snd.Play(SYawn)
			}
		}
	}
	g.rollBalls(dt)
}

// cashOut sells every bale on the pad; it happens at sundown.
func (g *Game) cashOut() {
	n := g.Stored
	if n == 0 {
		return
	}
	var gone []Bale
	for k := 0; k < min(n, StackMax); k++ {
		gone = append(gone, stackSpot(k))
	}
	g.Stored = 0
	g.rebake(gone)
	g.Coins += n * BalePrice
	g.Cashed += n
	g.cashN, g.cashFlash = n*BalePrice, 4
	for k := 0; k < min(n*2, 40); k++ {
		g.parts = append(g.parts, Particle{
			x: float64(PadX0) + g.rnd()*float64(PadX1-PadX0), y: float64(PadY0) + g.rnd()*float64(PadY1-PadY0),
			vx: (g.rnd() - 0.5) * 6, vy: -10 - g.rnd()*10, life: 0.8 + g.rnd()*0.8, size: 1.5, c: 0xFFD23E})
	}
	g.Snd.Play(SCoin)
}

// sleep ends the day at the barn door. Turning in before sundown still
// sells the bales, as the sun goes down while the farmers sleep.
func (g *Game) sleep() {
	if !g.Night() {
		g.cashOut()
	}
	g.endDay()
}

func (g *Game) newDay(slept bool) {
	g.Day++
	g.T = 0
	g.CutToday, g.Cashed = 0, 0
	g.mapView = false
	if !slept {
		return
	}
	for i := range g.pl { // everyone wakes up at the barn door
		p := &g.pl[i]
		if p.veh >= 0 {
			v := &g.veh[p.veh]
			v.driver, v.speed, v.header = -1, 0, false
			p.veh = -1
		}
		p.x, p.y = doorX-4+8*float64(i), doorY+1
		p.tired, p.shop, p.faceL = 0, false, false
		p.zoom = g.zoomTarget(i)
		g.updateCamera(i)
	}
}

// ---------------------------------------------------------------- balls

func (g *Game) rollBalls(dt float64) {
	for i := range g.balls {
		b := &g.balls[i]
		for j := range g.pl {
			p := &g.pl[j]
			d := math.Hypot(b.x-p.x, b.y-p.y)
			if p.veh < 0 && p.moving && d < 3.4 && b.z < 3 && math.Hypot(b.vx, b.vy) < 14 {
				nx, ny := (b.x-p.x)/(d+0.01), (b.y-p.y)/(d+0.01)
				b.vx, b.vy, b.vz = nx*30, ny*30, 17
				g.Snd.Play(SBoing)
			}
		}
		for j := range g.veh {
			v := &g.veh[j]
			d := math.Hypot(b.x-v.x, b.y-v.y)
			if d < vehRadius[v.kind]+1.5 && math.Abs(v.speed) > 3 && b.z < 4 {
				sp := math.Abs(v.speed) + 10
				b.vx, b.vy, b.vz = (b.x-v.x)/(d+0.01)*sp, (b.y-v.y)/(d+0.01)*sp, 15
				b.x += b.vx * dt * 2
				b.y += b.vy * dt * 2
				g.Snd.Play(SBoing)
			}
		}
		if b.z == 0 {
			fr := 1.3
			if g.tileAt(int(b.x), int(b.y)) == TWheat {
				fr = 3.5 // wheat is soft
			}
			k := math.Max(0, 1-fr*dt)
			b.vx *= k
			b.vy *= k
		}
		if nx := b.x + b.vx*dt; g.solidAt(nx, b.y) {
			b.vx = -b.vx * 0.7
		} else {
			b.x = nx
		}
		if ny := b.y + b.vy*dt; g.solidAt(b.x, ny) {
			b.vy = -b.vy * 0.7
		} else {
			b.y = ny
		}
		if b.z > 0 || b.vz != 0 {
			b.vz -= 70 * dt
			b.z += b.vz * dt
			if b.z <= 0 {
				b.z = 0
				if b.vz < -8 {
					b.vz = -b.vz * 0.55
				} else {
					b.vz = 0
				}
			}
		}
	}
}

func (g *Game) drawBalls(cm cam) {
	for _, b := range g.balls {
		g.rectA(int(cm.sx(b.x-1.3)), int(cm.sy(b.y+0.6)), int(cm.sx(b.x+1.3)), int(cm.sy(b.y+1.4)), 0x3A2A10, 90)
		r := 1.4 * cm.z
		cx, cy := cm.sx(b.x), cm.sy(b.y-b.z-0.6)
		for py := int(cy - r); py <= int(cy+r); py++ {
			for px := int(cx - r); px <= int(cx+r); px++ {
				dx, dy := float64(px)+0.5-cx, float64(py)+0.5-cy
				if dx*dx+dy*dy > r*r {
					continue
				}
				c := uint32(0xE0402E)
				switch {
				case math.Abs(dy) < r*0.22:
					c = 0xFFF4D6
				case dx < -r*0.2 && dy < -r*0.3:
					c = 0xFF9A8A
				case dy > r*0.55:
					c = 0xA82A1E
				}
				g.rect(px, py, px+1, py+1, c)
			}
		}
	}
}

// ---------------------------------------------------------------- light

// ambient returns the colour of daylight for the time of day.
func (g *Game) ambient3() (r, gg, b float64) {
	lerp := func(a, b [3]float64, k float64) (float64, float64, float64) {
		return a[0] + (b[0]-a[0])*k, a[1] + (b[1]-a[1])*k, a[2] + (b[2]-a[2])*k
	}
	dawn, noon := [3]float64{1, 0.89, 0.765}, [3]float64{1, 1, 1}
	dusk, night := [3]float64{0.784, 0.569, 0.627}, [3]float64{0.30, 0.36, 0.58}
	if f := g.T / DayLen; f < 0.08 {
		return lerp(dawn, noon, f/0.08)
	} else if f < 0.72 {
		return 1, 1, 1
	} else if f < 1 {
		return lerp(noon, dusk, (f-0.72)/0.28)
	}
	n := (g.T - DayLen) / NightLen
	switch {
	case n < 0.15:
		return lerp(dusk, night, n/0.15)
	case n > 0.88:
		return lerp(night, dawn, math.Min(1, (n-0.88)/0.12))
	}
	return night[0], night[1], night[2]
}

func (g *Game) addLight(cm cam, wx, wy, radius, strength float64) {
	cx, cy, r := cm.sx(wx), cm.sy(wy), radius*cm.z
	for py := max(int(cy-r), 0); py <= min(int(cy+r), SH-1); py++ {
		for px := max(int(cx-r), cm.vx); px <= min(int(cx+r), cm.vx+cm.vw-1); px++ {
			dx, dy := float64(px)-cx, float64(py)-cy
			if d := (dx*dx + dy*dy) / (r * r); d < 1 {
				g.light[py*SW+px] += float32(strength * (1 - d))
			}
		}
	}
}

// lighting tints the frame for the hour, lets lamps cut through the dark
// and closes in the edges of a tired farmer's view.
func (g *Game) lighting() {
	ar, ag, ab := g.ambient3()
	if ag < 0.995 {
		dark := ag < 0.9
		if dark {
			if g.light == nil {
				g.light = make([]float32, SW*SH)
			}
			clear(g.light)
			for i := range g.pl {
				p := &g.pl[i]
				vx, vw := g.view(i)
				cm := cam{p.camX, p.camY, p.zoom, vx, vw}
				for j := range g.veh {
					v := &g.veh[j]
					if v.driver < 0 {
						continue
					}
					s, c := math.Sin(v.ang), -math.Cos(v.ang)
					g.addLight(cm, v.x, v.y, 13, 0.45)
					g.addLight(cm, v.x+s*16, v.y+c*16, 15, 0.75)
					if g.Lamps {
						g.addLight(cm, v.x+s*22, v.y+c*22, 21, 0.6)
						g.addLight(cm, v.x+s*44, v.y+c*44, 28, 0.7)
					}
				}
				for j := range g.pl {
					if q := &g.pl[j]; q.veh < 0 {
						g.addLight(cm, q.x, q.y-2, 13, 0.7)
					}
				}
				g.addLight(cm, doorX, doorY-8, 36, 0.55) // barn
				g.addLight(cm, shopX, shopY-12, 16, 0.9) // vending machine
			}
		}
		wr, wg, wb := math.Max(ar, 1.0), math.Max(ag, 0.95), math.Max(ab, 0.80)
		for i, c := range g.fb {
			r, gg, b := ar, ag, ab
			if dark {
				if l := float64(g.light[i]); l > 0 {
					l = math.Min(1, l)
					r, gg, b = ar+(wr-ar)*l, ag+(wg-ag)*l, ab+(wb-ab)*l
				}
			}
			g.fb[i] = mul(c, uint32(r*255), uint32(gg*255), uint32(b*255))
		}
	}
	for i := range g.pl {
		t := g.pl[i].tired
		if t < 0.03 {
			continue
		}
		vx, vw := g.view(i)
		hw := float64(vw) / 2
		for py := 0; py < SH; py++ {
			ny := (float64(py) - SH/2) / (SH / 2)
			for px := vx; px < vx+vw; px++ {
				nx := (float64(px) - float64(vx) - hw) / hw
				if d := nx*nx + ny*ny; d > 0.3 {
					k := uint32(255 * (1 - math.Min(1, (d-0.3)/1.1)*t*0.85))
					g.fb[py*SW+px] = mul(g.fb[py*SW+px], k, k, k)
				}
			}
		}
	}
}
