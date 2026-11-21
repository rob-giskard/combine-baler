package main

// Combine-Baler - harvest prototype. Platform independent game core:
// everything renders into a 640x360 software framebuffer (0xRRGGBB).

import (
	"fmt"
	"math"
	"strings"
)

const (
	SW, SH = 640, 360   // internal resolution
	WW, WH = 1600, 1000 // world size in cells

	// the farm is a grid of fields separated by grass lanes
	GridW, GridH                 = 5, 4
	FieldW, FieldH, Lane, Margin = 304, 233, 14, 12
	YardC, YardR                 = 2, 1     // grid plot that holds the farmstead
	OX, OY                       = 400, 358 // farmstead art offset

	DayLen = 300.0 // seconds of real time per day (06:00 - 20:00)

	KgPerCell    = 0.8
	CellsPerBale = 900
	TruckCap     = 10

	PadX0, PadY0, PadX1, PadY1 = 380 + OX, 96 + OY, 420 + OX, 126 + OY // bale drop-off pad
)

const (
	TGrass = iota
	TWheat
	TStubble
	TDirt
)

const (
	StTitle = iota
	StPlay
	StPause
	StSummary
)

const (
	VCombine = iota
	VTruck
)

// Windows virtual key codes (the platform layer feeds a [256]bool of these)
const (
	KEnter = 0x0D
	KEsc   = 0x1B
	KSpace = 0x20
	KLeft  = 0x25
	KUp    = 0x26
	KRight = 0x27
	KDown  = 0x28
	KA     = 0x41
	KD     = 0x44
	KE     = 0x45
	KK     = 0x4B
	KL     = 0x4C
	KM     = 0x4D
	KN     = 0x4E
	KP     = 0x50
	KQ     = 0x51
	KR     = 0x52
	KS     = 0x53
	KW     = 0x57
	KF2    = 0x71
)

type Particle struct {
	x, y, vx, vy, life, size float64
	c                        uint32
	ch                       rune // when set, drawn as a letter (sleepy Z's)
}

type Bale struct{ x, y int }

type DayLog struct {
	Day    int
	Tonnes float64
	Bales  int
}

type Vehicle struct {
	kind             int
	x, y, ang, speed float64
	sx, sy           float64 // parking spot
	header           bool
	reel, tread      float64
	load             int
	driver           int // player index, -1 when empty
	tint             int
	baleAcc          int
	exhaustT, dropT  float64
}

type Player struct {
	x, y             float64
	veh              int // vehicle index, -1 on foot
	home             int // combine this player starts each day in
	faceL, moving    bool
	walk, stepT      float64
	zoom, camX, camY float64

	tired   float64 // 0..1, builds up while awake at night
	fizz    float64 // seconds of fizzy-water pep left
	yawnT   float64
	shop    bool // vending machine panel open
	shopSel int
	shake   float64
	last    Input
}

type Input struct{ up, down, left, right, act, use bool }

type Game struct {
	fb    []uint32
	tile  []uint8
	col   []uint32 // terrain colour per cell
	obj   []uint32 // baked object colour per cell (0 = none)
	solid []bool

	veh      []Vehicle
	pl       []Player
	NPlayers int
	menuSel  int
	mapView  bool
	spr      [40 * 38]uint32
	clip0    int
	clip1    int

	State       int
	Quit        bool
	Day         int
	T           float64 // seconds into the day
	CutToday    int
	CutTotal    int
	WheatTotal  int
	Stubble     int
	BaleCount   int // bales made
	Stored      int // bales delivered to the pad
	StoredToday int
	bales       []Bale // bales lying in the field
	Log         []DayLog

	Snd    *Audio
	parts  []Particle
	clock  float64
	toast  string
	toastT float64
	saveN  int

	Coins     int
	Cashed    int // bales sold at today's sundown
	Lamps     bool
	balls     []Ball
	cashFlash float64
	cashN     int
	light     []float32
	hintT     float64

	prev [256]bool
	rng  uint32
}

func hash(x, y int) uint32 {
	h := uint32(x)*374761393 + uint32(y)*668265263
	h = (h ^ (h >> 13)) * 1274126177
	return h ^ (h >> 16)
}

func (g *Game) rnd() float64 {
	g.rng ^= g.rng << 13
	g.rng ^= g.rng >> 17
	g.rng ^= g.rng << 5
	return float64(g.rng%100000) / 100000.0
}

func NewGame() *Game {
	g := &Game{
		fb:    make([]uint32, SW*SH),
		tile:  make([]uint8, WW*WH),
		col:   make([]uint32, WW*WH),
		obj:   make([]uint32, WW*WH),
		solid: make([]bool, WW*WH),
		rng:   0x9E3779B9,
		Day:   1,
		State: StTitle,
		Snd:   NewAudio(),
		clip1: SW,
	}
	g.buildWorld()
	g.setup(1)
	return g
}

// setup places the machines and farmers for a 1 or 2 player game.
func (g *Game) setup(n int) {
	g.NPlayers = n
	g.veh = []Vehicle{
		{kind: VCombine, sx: 340 + OX, sy: 115 + OY},
		{kind: VTruck, sx: 452 + OX, sy: 115 + OY},
	}
	g.pl = []Player{{home: 0}}
	if n == 2 {
		g.veh = append(g.veh,
			Vehicle{kind: VCombine, sx: 300 + OX, sy: 115 + OY, tint: 1},
			Vehicle{kind: VTruck, sx: 476 + OX, sy: 115 + OY, tint: 1})
		g.pl = append(g.pl, Player{home: 2})
	}
	g.park()
}

// park puts every machine back in the yard with the farmers in their combines.
func (g *Game) park() {
	for i := range g.veh {
		v := &g.veh[i]
		v.x, v.y, v.ang, v.speed, v.header, v.driver = v.sx, v.sy, math.Pi, 0, false, -1
	}
	for i := range g.pl {
		p := &g.pl[i]
		p.veh = p.home
		g.veh[p.home].driver = i
		p.zoom = g.zoomTarget(i)
		g.updateCamera(i)
	}
	g.mapView = false
}

func (g *Game) split() bool { return g.NPlayers == 2 && g.State != StTitle }

func (g *Game) view(i int) (x, w int) {
	if !g.split() {
		return 0, SW
	}
	if i == 0 {
		return 0, SW/2 - 1
	}
	return SW/2 + 1, SW/2 - 1
}

// ---------------------------------------------------------------- update

var probes = [2][][2]float64{
	{{0, 0}, {-5, -8}, {5, -8}, {-9, 10}, {9, 10}, {-5, 5}, {5, 5}, {0, 10}, {-4.5, 10}, {4.5, 10}, {0, -8}},
	{{0, 0}, {-5, 9}, {5, 9}, {-5, -9}, {5, -9}, {0, 9}, {0, -9}, {-5, 0}, {5, 0}},
}
var vehRadius = [2]float64{9, 7.5}

func (g *Game) solidAt(wx, wy float64) bool {
	ix, iy := int(math.Floor(wx)), int(math.Floor(wy))
	return ix < 1 || iy < 1 || ix >= WW-1 || iy >= WH-1 || g.solid[iy*WW+ix]
}

func (g *Game) blocked(vi int, x, y, a float64) bool {
	v := &g.veh[vi]
	s, c := math.Sin(a), math.Cos(a)
	for _, p := range probes[v.kind] {
		if g.solidAt(x+p[0]*c+p[1]*s, y+p[0]*s-p[1]*c) {
			return true
		}
	}
	for j := range g.veh {
		if j == vi {
			continue
		}
		o := &g.veh[j]
		dn := math.Hypot(x-o.x, y-o.y)
		if dn < vehRadius[v.kind]+vehRadius[o.kind] && dn < math.Hypot(v.x-o.x, v.y-o.y) {
			return true
		}
	}
	return false
}

func (g *Game) walkBlocked(x, y float64) bool {
	return g.solidAt(x-1.5, y) || g.solidAt(x+1.5, y) || g.solidAt(x-1.5, y+2) || g.solidAt(x+1.5, y+2)
}

func (g *Game) Tonnes(cells int) float64 { return float64(cells) * KgPerCell / 1000 }

func (g *Game) FieldPct() float64 {
	return 100 * float64(g.Stubble) / float64(g.WheatTotal)
}

func (g *Game) ClockString() string {
	m := (6*60 + int(g.T/DayLen*14*60)) % (24 * 60)
	return fmt.Sprintf("%02d:%02d", m/60, m%60)
}

func (g *Game) say(s string) { g.toast, g.toastT = strings.ToUpper(s), 3.5 }

func (g *Game) input(i int, keys *[256]bool) Input {
	hit := func(k int) bool { return keys[k] && !g.prev[k] }
	var in Input
	if i == 0 || g.NPlayers == 1 {
		in.up, in.down, in.left, in.right = keys[KW], keys[KS], keys[KA], keys[KD]
		in.act, in.use = hit(KSpace), hit(KE)
	}
	if i == 1 || g.NPlayers == 1 {
		in.up = in.up || keys[KUp]
		in.down = in.down || keys[KDown]
		in.left = in.left || keys[KLeft]
		in.right = in.right || keys[KRight]
		in.act = in.act || hit(KL)
		in.use = in.use || hit(KK)
	}
	return in
}

// nearVehicle returns the closest empty machine a farmer on foot can climb into.
func (g *Game) nearVehicle(p *Player) int {
	best, bd := -1, 13.0
	for j := range g.veh {
		v := &g.veh[j]
		if d := math.Hypot(v.x-p.x, v.y-p.y); v.driver < 0 && d < bd {
			best, bd = j, d
		}
	}
	return best
}

func (g *Game) Update(dt float64, keys *[256]bool) {
	hit := func(k int) bool { return keys[k] && !g.prev[k] }
	defer func() { g.prev = *keys }()
	g.clock += dt
	if g.toastT > 0 {
		g.toastT -= dt
	}
	if hit(KP) {
		if name, err := g.SavePicture(""); err == nil {
			g.say("SAVED " + name)
		} else {
			g.say("COULD NOT SAVE PICTURE")
		}
	}
	if hit(KF2) {
		g.Snd.ToggleMusic()
	}
	g.ambient(dt)
	if g.State != StPlay {
		g.Snd.SetState(false, 0, false, 0, false)
	}

	switch g.State {
	case StTitle:
		if hit(KUp) || hit(KDown) || hit(KW) || hit(KS) {
			g.menuSel = 1 - g.menuSel
			g.Snd.Play(SMenu)
		}
		if hit(KEnter) || hit(KSpace) {
			g.setup(g.menuSel + 1)
			g.State = StPlay
			g.park()
			g.Snd.Play(SSelect)
		}
		if hit(KEsc) {
			g.Quit = true
		}
		return
	case StPause:
		switch {
		case hit(KEsc) || hit(KEnter):
			g.State = StPlay
		case hit(KN):
			g.State = StPlay
			g.sleep()
		case hit(KM):
			snd, fb, sel := g.Snd, g.fb, g.NPlayers-1
			*g = *NewGame()
			g.Snd, g.fb, g.menuSel = snd, fb, sel
		case hit(KQ):
			g.Quit = true
		}
		return
	case StSummary:
		if hit(KEnter) || hit(KR) {
			if hit(KR) {
				g.Replant()
				g.say("FIELDS REPLANTED")
			}
			g.State = StPlay
			g.newDay(true)
			g.Snd.Play(SSelect)
		}
		if hit(KEsc) {
			g.Quit = true
		}
		return
	}

	// ---- playing
	if hit(KEsc) {
		g.State = StPause
		return
	}
	if hit(KM) {
		g.mapView = !g.mapView
	}

	for i := range g.pl {
		p := &g.pl[i]
		in := g.input(i, keys)
		last := p.last
		p.last = in
		if p.shop {
			g.shopInput(i, in, last)
			continue
		}
		if p.veh < 0 {
			g.walk(p, in, dt)
			if in.use {
				g.interact(i)
				if g.State != StPlay {
					return
				}
			}
			continue
		}
		v := &g.veh[p.veh]
		if in.use {
			g.exit(i)
			continue
		}
		if in.act && v.kind == VCombine {
			v.header = !v.header
			if v.header {
				g.Snd.Play(SReapOn)
			} else {
				g.Snd.Play(SReapOff)
			}
		}
		g.drive(p.veh, in, dt)
	}

	reaping := false
	var cOn, tOn bool
	var cSpd, tSpd float64
	for j := range g.veh {
		v := &g.veh[j]
		if v.driver >= 0 {
			f := math.Abs(v.speed) / 32
			if v.kind == VCombine {
				cOn, cSpd = true, math.Max(cSpd, f)
			} else {
				tOn, tSpd = true, math.Max(tSpd, f)
			}
		}
		if v.kind == VCombine {
			if g.work(v, dt) > 0 {
				reaping = true
			}
		} else {
			g.haul(v, dt)
		}
	}
	g.Snd.SetState(cOn, cSpd, tOn, tSpd, reaping)

	k := 1 - math.Exp(-dt*2.6)
	for i := range g.pl {
		p := &g.pl[i]
		target := g.zoomTarget(i)
		p.zoom = math.Exp(math.Log(p.zoom) + (math.Log(target)-math.Log(p.zoom))*k)
		if math.Abs(p.zoom-target) < 0.004 {
			p.zoom = target
		}
		g.updateCamera(i)
	}

	g.cosy(dt)
}

func (g *Game) zoomTarget(i int) float64 {
	p := &g.pl[i]
	two := g.NPlayers == 2
	pick := func(one, split float64) float64 {
		if two {
			return split
		}
		return one
	}
	switch {
	case g.mapView:
		return pick(SH/float64(WH), (SW/2-1)/float64(WW))
	case p.veh < 0:
		return 4
	case g.veh[p.veh].kind == VTruck:
		return pick(2.6, 2.2)
	case g.veh[p.veh].header:
		return pick(1.6, 1.3)
	}
	return pick(4, 3)
}

func (g *Game) walk(p *Player, in Input, dt float64) {
	dx, dy := 0.0, 0.0
	if in.left {
		dx--
	}
	if in.right {
		dx++
	}
	if in.up {
		dy--
	}
	if in.down {
		dy++
	}
	p.moving = dx != 0 || dy != 0
	if !p.moving {
		return
	}
	if dx != 0 {
		p.faceL = dx < 0
	}
	l := math.Hypot(dx, dy)
	sp := 17 * (1 - 0.25*p.tired)
	if p.fizz > 0 {
		sp *= 1.2
	}
	dx, dy = dx/l*sp*dt, dy/l*sp*dt
	if !g.walkBlocked(p.x+dx, p.y) {
		p.x += dx
	}
	if !g.walkBlocked(p.x, p.y+dy) {
		p.y += dy
	}
	p.walk += dt * 9
	p.stepT -= dt
	if p.stepT <= 0 {
		p.stepT = 0.27
		g.Snd.Play(SStep)
	}
}

func (g *Game) exit(i int) {
	p := &g.pl[i]
	v := &g.veh[p.veh]
	spots := [2][][2]float64{
		{{-8.5, -1}, {8.5, -1}, {0, -12}, {-8.5, -6}, {8.5, -6}},
		{{-8.5, 0}, {8.5, 0}, {0, -13}, {0, 13}},
	}
	s, c := math.Sin(v.ang), math.Cos(v.ang)
	for _, sp := range spots[v.kind] {
		x := v.x + sp[0]*c + sp[1]*s
		y := v.y + sp[0]*s - sp[1]*c
		if g.walkBlocked(x, y) {
			continue
		}
		p.x, p.y, p.veh = x, y, -1
		v.driver, v.speed, v.header = -1, 0, false
		g.Snd.Play(SDoor)
		return
	}
}

func (g *Game) drive(vi int, in Input, dt float64) {
	v := &g.veh[vi]
	maxF, maxR, acc, turn := 32.0, 14.0, 45.0, 1.7
	if v.header {
		maxF = 25
	}
	if v.kind == VTruck {
		maxF, maxR, acc, turn = 46, 18, 60, 2.1
	}
	if v.driver >= 0 {
		maxF *= 1 - 0.3*g.pl[v.driver].tired // sleepy farmers drive gently
	}
	switch {
	case in.up && !in.down:
		v.speed += acc * dt
	case in.down && !in.up:
		v.speed -= (acc + 10) * dt
	default:
		d := 40 * dt
		if math.Abs(v.speed) <= d {
			v.speed = 0
		} else if v.speed > 0 {
			v.speed -= d
		} else {
			v.speed += d
		}
	}
	v.speed = math.Max(-maxR, math.Min(maxF, v.speed))

	steer := 0.0
	if in.left {
		steer--
	}
	if in.right {
		steer++
	}
	grip := math.Min(1, math.Abs(v.speed)/7)
	if v.speed < 0 {
		steer = -steer
	}
	na := v.ang + steer*turn*grip*dt
	nx := v.x + math.Sin(na)*v.speed*dt
	ny := v.y - math.Cos(na)*v.speed*dt
	if !g.blocked(vi, nx, ny, na) {
		v.x, v.y, v.ang = nx, ny, na
	} else if !g.blocked(vi, nx, ny, v.ang) {
		v.x, v.y = nx, ny
	} else {
		if math.Abs(v.speed) > 8 {
			g.Snd.Play(SBump)
		}
		v.speed = -v.speed * 0.25
	}
	v.tread += v.speed * dt

	v.exhaustT -= dt
	if v.exhaustT <= 0 && (in.up || in.down) {
		v.exhaustT = 0.09
		s, c := math.Sin(v.ang), math.Cos(v.ang)
		r, f := 2.5, -5.5
		if v.kind == VTruck {
			r, f = -3.5, -9
		}
		g.parts = append(g.parts, Particle{
			x: v.x + r*c + f*s, y: v.y + r*s - f*c,
			vx: (g.rnd() - 0.5) * 3, vy: -4 - g.rnd()*3,
			life: 0.7 + g.rnd()*0.4, size: 1.5, c: 0x8A8A90})
	}
}

// work runs a combine's reaper for one step and returns the cells cut.
func (g *Game) work(v *Vehicle, dt float64) int {
	if !v.header {
		return 0
	}
	v.reel += dt * 9
	s, c := math.Sin(v.ang), math.Cos(v.ang)
	cut := 0
	for f := 6.5; f <= 10.0; f += 0.5 {
		for r := -9.0; r <= 9.0; r += 0.5 {
			x := int(math.Floor(v.x + r*c + f*s))
			y := int(math.Floor(v.y + r*s - f*c))
			if x < 0 || y < 0 || x >= WW || y >= WH || g.tile[y*WW+x] != TWheat {
				continue
			}
			g.tile[y*WW+x] = TStubble
			cut++
			for d := -2; d <= 2; d++ {
				g.recolor(x, y+d)
			}
		}
	}
	if cut == 0 {
		return 0
	}
	g.CutToday += cut
	g.CutTotal += cut
	g.Stubble += cut
	v.baleAcc += cut
	if v.baleAcc >= CellsPerBale {
		v.baleAcc -= CellsPerBale
		b := Bale{int(v.x - s*11 - 3), int(v.y + c*11 - 2)}
		g.bales = append(g.bales, b)
		g.bakeBale(b, true)
		g.BaleCount++
		g.Snd.Play(SBale)
	}
	// chaff flying out of the back
	cols := [...]uint32{0xFFD24A, 0xF5B82A, 0xFFE9A0, 0xD8B56A}
	for i := 0; i < 2; i++ {
		r := (g.rnd() - 0.5) * 6
		g.parts = append(g.parts, Particle{
			x: v.x + r*c - 8.5*s, y: v.y + r*s + 8.5*c,
			vx: -s*8 + (g.rnd()-0.5)*10, vy: c*8 + (g.rnd()-0.5)*10,
			life: 0.5 + g.rnd()*0.6, size: 1, c: cols[int(g.rnd()*4)%4]})
	}
	return cut
}

// haul lets a truck scoop up bales it drives over and unload them on the pad.
func (g *Game) haul(v *Vehicle, dt float64) {
	if v.driver < 0 {
		return
	}
	s, c := math.Sin(v.ang), math.Cos(v.ang)
	var gone []Bale
	for i := len(g.bales) - 1; i >= 0 && v.load < TruckCap; i-- {
		b := g.bales[i]
		dx, dy := float64(b.x)+3-v.x, float64(b.y)+2.5-v.y
		if math.Abs(dx*c+dy*s) < 7.5 && math.Abs(dx*s-dy*c) < 11 {
			gone = append(gone, b)
			g.bales = append(g.bales[:i], g.bales[i+1:]...)
			v.load++
			g.Snd.Play(SPick)
			for k := 0; k < 5; k++ {
				g.parts = append(g.parts, Particle{x: float64(b.x) + 3, y: float64(b.y) + 2,
					vx: (g.rnd() - 0.5) * 16, vy: (g.rnd() - 0.5) * 16, life: 0.4 + g.rnd()*0.3, size: 1, c: 0xFFE07A})
			}
		}
	}
	if gone != nil {
		g.rebake(gone)
	}
	v.dropT -= dt
	if v.load > 0 && v.dropT <= 0 && v.x > PadX0-4 && v.x < PadX1+4 && v.y > PadY0-4 && v.y < PadY1+4 {
		v.load--
		v.dropT = 0.2
		g.Stored++
		g.StoredToday++
		g.Snd.Play(SDrop)
		g.rebake(nil)
	}
}

func (g *Game) endDay() {
	g.Log = append(g.Log, DayLog{g.Day, g.Tonnes(g.CutToday), g.Cashed})
	for i := range g.veh {
		g.veh[i].header, g.veh[i].speed = false, 0
	}
	g.State = StSummary
	g.Snd.Play(SDayEnd)
}

func (g *Game) ambient(dt float64) {
	n := 0
	for _, p := range g.parts {
		p.life -= dt
		if p.life <= 0 {
			continue
		}
		p.x += p.vx * dt
		p.y += p.vy * dt
		p.vx *= 1 - 1.5*dt
		p.vy *= 1 - 1.5*dt
		g.parts[n] = p
		n++
	}
	g.parts = g.parts[:n]
}

func (g *Game) updateCamera(i int) {
	p := &g.pl[i]
	fx, fy := p.x, p.y
	if p.veh >= 0 {
		fx, fy = g.veh[p.veh].x, g.veh[p.veh].y
	}
	vw := SW
	if g.NPlayers == 2 {
		vw = SW/2 - 1
	}
	hw, hh := float64(vw)/2/p.zoom, SH/2/p.zoom
	p.camX, p.camY = fx, fy
	if 2*hw >= WW {
		p.camX = WW / 2
	} else {
		p.camX = math.Max(hw, math.Min(WW-hw, fx))
	}
	if 2*hh >= WH {
		p.camY = WH / 2
	} else {
		p.camY = math.Max(hh, math.Min(WH-hh, fy))
	}
}

// ---------------------------------------------------------------- render

func mul(c uint32, r, gg, b uint32) uint32 {
	return ((c>>16&255)*r/255)<<16 | ((c>>8&255)*gg/255)<<8 | (c&255)*b/255
}

func blend(a, b uint32, t uint32) uint32 { // t 0..255 of b
	r := ((a>>16&255)*(255-t) + (b>>16&255)*t) / 255
	gg := ((a>>8&255)*(255-t) + (b>>8&255)*t) / 255
	bl := ((a&255)*(255-t) + (b&255)*t) / 255
	return r<<16 | gg<<8 | bl
}

// body paint: main, light, dark, neck, grill
var combinePaint = [2][5]uint32{
	{0x2FA07E, 0x56C9A2, 0x1C6B55, 0x1F7A63, 0x24876B},
	{0x3A7BD5, 0x74ABF2, 0x24509A, 0x2B5DAE, 0x2F68BE},
}
var truckPaint = [2][3]uint32{
	{0xD8452E, 0xF2775A, 0x9A2A1A},
	{0xE8A52A, 0xFFCB5E, 0xA86E12},
}

// buildSprite draws vehicle v into g.spr and returns its size in texels
// (front of the machine at the top).
func (g *Game) buildSprite(v *Vehicle) (w, h int) {
	w, h = 40, 38
	if v.kind == VTruck {
		w, h = 16, 28
	}
	sp := g.spr[:]
	for i := range sp {
		sp[i] = 0
	}
	fill := func(u0, v0, u1, v1 int, c uint32) {
		for y := v0; y < v1; y++ {
			for x := u0; x < u1; x++ {
				sp[y*w+x] = c
			}
		}
	}
	wheel := func(u0, v0, u1, v1 int) {
		fill(u0, v0, u1, v1, 0x2B2B33)
		t := int(math.Floor(v.tread*2)) % 3
		if t < 0 {
			t += 3
		}
		for y := v0; y < v1; y++ {
			if (y+t)%3 == 0 {
				fill(u0, y, u1, y+1, 0x50505C)
			}
		}
	}
	if v.kind == VTruck {
		pt := truckPaint[v.tint]
		wheel(0, 3, 2, 9)
		wheel(14, 3, 16, 9)
		wheel(0, 19, 2, 25)
		wheel(14, 19, 16, 25)
		fill(1, 0, 15, 11, pt[0]) // cab and bonnet
		fill(1, 0, 15, 1, 0xD8D8E0)
		fill(2, 0, 4, 1, 0xFFE06A)
		fill(12, 0, 14, 1, 0xFFE06A)
		fill(2, 1, 14, 2, pt[1])
		fill(1, 1, 2, 11, pt[2])
		fill(14, 1, 15, 11, pt[2])
		fill(3, 4, 13, 6, 0x9FDDEE)
		fill(4, 4, 6, 5, 0xE6FAFF)
		fill(3, 6, 13, 10, pt[1])
		fill(1, 11, 15, 28, 0x7A4A2A) // bed
		fill(2, 12, 14, 27, 0xB5773A)
		for y := 14; y < 27; y += 3 {
			fill(2, y, 14, y+1, 0xA3682F)
		}
		for n := 0; n < v.load; n++ {
			u0, v0 := 2+(n%2)*6, 12+(n/2)*3
			fill(u0, v0, u0+6, v0+3, 0xF2C24A)
			fill(u0, v0, u0+6, v0+1, 0xFFE07A)
			fill(u0+5, v0, u0+6, v0+3, 0xC8922A)
		}
		fill(1, 27, 3, 28, 0xE02A2A)
		fill(13, 27, 15, 28, 0xE02A2A)
		return
	}
	pt := combinePaint[v.tint]
	// header
	fill(1, 0, 39, 2, 0x55555E)
	for u := 3; u < 37; u += 2 {
		sp[u] = 0xB8B8C4
	}
	fill(1, 2, 39, 9, 0xC8402A)
	fill(3, 3, 37, 8, 0x6E2618)
	ph := int(v.reel) % 5
	for k := 0; k < 2; k++ {
		y := 3 + (ph+k*3)%5
		fill(3, y, 37, y+1, 0xF2E3B0)
	}
	fill(19, 3, 21, 8, 0xC8402A)
	fill(1, 0, 3, 9, 0xA8321E)
	fill(37, 0, 39, 9, 0xA8321E)
	fill(15, 9, 25, 13, pt[3]) // feeder neck
	wheel(5, 12, 10, 24)
	wheel(30, 12, 35, 24)
	wheel(7, 28, 10, 35)
	wheel(30, 28, 33, 35)
	// body
	fill(10, 12, 30, 35, pt[0])
	fill(10, 12, 30, 13, pt[1])
	fill(11, 13, 12, 34, pt[1])
	fill(10, 12, 11, 35, pt[2])
	fill(29, 12, 30, 35, pt[2])
	fill(10, 34, 30, 35, pt[2])
	// cabin
	fill(13, 13, 27, 21, 0xF4F0E0)
	fill(14, 13, 26, 16, 0x9FDDEE)
	fill(15, 14, 18, 15, 0xE6FAFF)
	fill(15, 17, 25, 20, 0xE2DCC8)
	if v.header && int(g.clock*4)%2 == 0 {
		fill(19, 18, 21, 19, 0xFF9A1E)
	} else {
		fill(19, 18, 21, 19, 0xB86A1A)
	}
	// grain tank
	fill(13, 22, 27, 31, pt[2])
	fill(14, 23, 26, 30, 0xF0B429)
	for i := 0; i < 14; i++ {
		hh := hash(i, 7)
		sp[(23+int(hh%7))*w+14+int(hh>>8%12)] = 0xFFD860
	}
	// engine
	fill(13, 32, 27, 34, pt[2])
	for u := 14; u < 27; u += 2 {
		sp[32*w+u] = pt[4]
		sp[33*w+u] = pt[4]
	}
	fill(24, 31, 26, 33, 0x333338)
	return
}

// view transform for the viewport being drawn
type cam struct {
	x, y, z float64
	vx, vw  int
}

func (c cam) sx(wx float64) float64 { return (wx-c.x)*c.z + float64(c.vx) + float64(c.vw)/2 }
func (c cam) sy(wy float64) float64 { return (wy-c.y)*c.z + SH/2 }

func (g *Game) drawVehicle(v *Vehicle, cm cam, shadow bool) {
	w, h := g.buildSprite(v)
	hw, ff, ts := 10.0, 10.5, 0.5 // half width, front, cells per texel
	if v.kind == VTruck {
		hw, ff, ts = 5.2, 9.1, 0.65
	}
	ox, oy := 0.0, 0.0
	if shadow {
		ox, oy = 1.2, 1.8
	}
	z := cm.z
	cx, cy := cm.sx(v.x+ox), cm.sy(v.y+oy)
	rad := int(15*z) + 2
	s, c := math.Sin(v.ang), math.Cos(v.ang)
	for py := max(int(cy)-rad, 0); py <= min(int(cy)+rad, SH-1); py++ {
		for px := max(int(cx)-rad, g.clip0); px <= min(int(cx)+rad, g.clip1-1); px++ {
			dx := (float64(px) + 0.5 - cx) / z
			dy := (float64(py) + 0.5 - cy) / z
			u := int(math.Floor((dx*c + dy*s + hw) / ts))
			t := int(math.Floor((ff - (dx*s - dy*c)) / ts))
			if u < 0 || t < 0 || u >= w || t >= h {
				continue
			}
			col := g.spr[t*w+u]
			if col == 0 {
				continue
			}
			if shadow {
				g.fb[py*SW+px] = mul(g.fb[py*SW+px], 170, 165, 185)
				continue
			}
			if !v.header { // standing wheat in front of the machine leans over it
				wx, wy := int(math.Floor(v.x+dx)), int(math.Floor(v.y+dy))
				hb := 1 + tuftArc[wx&7]
				for k := 1; k <= hb; k++ {
					if g.tileAt(wx, wy+k) != TWheat {
						continue
					}
					dk := dy + float64(k)
					u2 := int(math.Floor((dx*c + dk*s + hw) / ts))
					t2 := int(math.Floor((ff - (dx*s - dk*c)) / ts))
					if u2 >= 0 && t2 >= 0 && u2 < w && t2 < h {
						continue // that spot is under the machine itself
					}
					col = wheatTex(wx, wy)
					if k == hb {
						col = 0xFFDC62
					}
					break
				}
			}
			g.fb[py*SW+px] = col
		}
	}
}

var farmerArt = [2][16]string{{
	"....HHHH....",
	"...HHHHHH...",
	"...BBBBBB...",
	"HHHHHHHHHHHH",
	".hhhhhhhhhh.",
	"..FFFFFFFF..",
	"..FKFFFFKF..",
	"..FFFFFFFF..",
	"...SSSSSS...",
	".SSOSSSSOSS.",
	".SSOOOOOOSS.",
	".FFOOOOOOFF.",
	"..OOOOOOOO..",
	"..OOO..OOO..",
	"..OOO..OOO..",
	"..GGG..GGG..",
}, {
	"....HHHH....",
	"...HHHHHH...",
	"...BBBBBB...",
	"HHHHHHHHHHHH",
	".hhhhhhhhhh.",
	"..FFFFFFFF..",
	"..FKFFFFKF..",
	"..FFFFFFFF..",
	"...SSSSSS...",
	".SSOSSSSOSS.",
	".FSOOOOOOSF.",
	"..FOOOOOOF..",
	"..OOOOOOOO..",
	"...OOOOOO...",
	"...OO.OOO...",
	"...GG.GGG...",
}}

// drawFarmer draws a little farmer in a straw hat; standing wheat hides
// everything below the waist.
func (g *Game) drawFarmer(i int, cm cam) {
	p := &g.pl[i]
	band, shirt, dungarees := uint32(0xD8402A), uint32(0xE8563A), uint32(0x3A6BC5)
	if i == 1 {
		band, shirt, dungarees = 0x3A7BD5, 0xF4F0E0, 0x3E9A5A
	}
	ts := cm.z / 2
	x0, y0 := cm.sx(p.x-3), cm.sy(p.y-5.5)
	fr := 0
	if p.moving && int(p.walk)%2 == 1 {
		fr = 1
	}
	feet := int(math.Floor(p.y + 2.5))
	if g.tileAt(int(p.x), feet) != TWheat {
		g.rectA(int(cm.sx(p.x-2.5)), int(cm.sy(p.y+2)), int(cm.sx(p.x+2.5)), int(cm.sy(p.y+3)), 0x3A2A10, 90)
	}
	for t, row := range farmerArt[fr] {
		for u := 0; u < 12; u++ {
			ch := row[u]
			if p.faceL {
				ch = row[11-u]
			}
			var c uint32
			switch ch {
			case 'H':
				c = 0xF6DC8A
			case 'h':
				c = 0xB8862E
			case 'B':
				c = band
			case 'F':
				c = 0xF6C9A0
			case 'K':
				c = 0x33333A
			case 'S':
				c = shirt
			case 'O':
				c = dungarees
			case 'G':
				c = 0x5A3418
			default:
				continue
			}
			wx := int(math.Floor(p.x - 3 + (float64(u)+0.5)/2))
			wy := p.y - 5.5 + (float64(t)+0.5)/2
			if g.tileAt(wx, feet) == TWheat || g.tileAt(wx, feet+1) == TWheat {
				top := p.y + 2.5 - 3 - float64(tuftArc[wx&7])*0.75
				if wy >= top+0.5 {
					c = wheatTex(wx, int(math.Floor(wy)))
				} else if wy >= top {
					c = 0xFFDC62
				}
			}
			g.rect(int(x0+float64(u)*ts), int(y0+float64(t)*ts), int(x0+float64(u+1)*ts), int(y0+float64(t+1)*ts), c)
		}
	}
}

func (g *Game) rect(x0, y0, x1, y1 int, c uint32) {
	for y := max(y0, 0); y < min(y1, SH); y++ {
		for x := max(x0, g.clip0); x < min(x1, g.clip1); x++ {
			g.fb[y*SW+x] = c
		}
	}
}

func (g *Game) rectA(x0, y0, x1, y1 int, c uint32, a uint32) {
	for y := max(y0, 0); y < min(y1, SH); y++ {
		for x := max(x0, g.clip0); x < min(x1, g.clip1); x++ {
			g.fb[y*SW+x] = blend(g.fb[y*SW+x], c, a)
		}
	}
}

func (g *Game) text(x, y int, s string, sc int, c uint32) {
	for _, ch := range s {
		gl, ok := font[ch]
		if ok {
			for i := 0; i < 35; i++ {
				if gl[i] == '#' {
					g.rect(x+(i%5)*sc, y+(i/5)*sc, x+(i%5+1)*sc, y+(i/5+1)*sc, c)
				}
			}
		}
		x += 6 * sc
	}
}

func textW(s string, sc int) int { return len(s)*6*sc - sc }

// textO draws text with a dark outline so it reads on top of wheat.
func (g *Game) textO(x, y int, s string, sc int, c uint32) {
	o := min(sc, 2)
	for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}, {1, 1}, {-1, 1}, {1, -1}, {-1, -1}, {0, 2}, {1, 2}, {-1, 2}} {
		g.text(x+d[0]*o, y+d[1]*o, s, sc, 0x5A3418)
	}
	g.text(x, y, s, sc, c)
}

func (g *Game) textC(y int, s string, sc int, c uint32) {
	g.textO((SW-textW(s, sc))/2, y, s, sc, c)
}

func (g *Game) cellColor(x, y int) uint32 {
	i := y*WW + x
	if c := g.obj[i]; c != 0 {
		return c
	}
	return g.col[i]
}

func (g *Game) renderView(i int) {
	p := &g.pl[i]
	vx, vw := g.view(i)
	g.clip0, g.clip1 = vx, vx+vw
	cm := cam{p.camX, p.camY, p.zoom, vx, vw}
	inv := 1 / cm.z
	x0 := cm.x - (float64(vw)/2-0.5)*inv
	y0 := cm.y - (SH/2-0.5)*inv
	for sy := 0; sy < SH; sy++ {
		wy := int(math.Floor(y0 + float64(sy)*inv))
		row := g.fb[sy*SW+vx : sy*SW+vx+vw]
		for sx := range row {
			wx := int(math.Floor(x0 + float64(sx)*inv))
			if wx < 0 || wy < 0 || wx >= WW || wy >= WH {
				c := uint32(0x6E8E2A)
				if hash(wx/2, wy/2)%3 == 0 {
					c = 0x5E7E24
				}
				row[sx] = c
				continue
			}
			c := g.obj[wy*WW+wx]
			if c == 0 {
				c = g.col[wy*WW+wx]
			}
			row[sx] = c
		}
	}
	for j := range g.veh {
		g.drawVehicle(&g.veh[j], cm, true)
	}
	for j := range g.veh {
		g.drawVehicle(&g.veh[j], cm, false)
	}
	for j := range g.pl {
		if g.pl[j].veh < 0 {
			g.drawFarmer(j, cm)
		}
	}
	g.drawBalls(cm)
	for _, pt := range g.parts {
		px, py := cm.sx(pt.x), cm.sy(pt.y)
		if pt.ch != 0 {
			g.textO(int(px), int(py), string(pt.ch), 1, pt.c)
			continue
		}
		sz := max(int(pt.size*cm.z), 1)
		a := uint32(230)
		if pt.life < 0.5 {
			a = uint32(230 * pt.life / 0.5)
		}
		g.rectA(int(px)-sz/2, int(py)-sz/2, int(px)-sz/2+sz, int(py)-sz/2+sz, pt.c, a)
	}
	g.prompt(i, cm)
	if cm.z < 0.7 && int(g.clock*3)%2 == 0 { // who is where on the map
		for j := range g.pl {
			q := &g.pl[j]
			x, y := q.x, q.y
			if q.veh >= 0 {
				x, y = g.veh[q.veh].x, g.veh[q.veh].y
			}
			px, py := int(cm.sx(x)), int(cm.sy(y))
			g.rect(px-4, py-4, px+4, py+4, 0x5A3418)
			g.rect(px-3, py-3, px+3, py+3, [2]uint32{0xFFFFFF, 0x74ABF2}[j])
		}
	}
	g.clip0, g.clip1 = 0, SW
}

func (g *Game) Render() {
	for i := range g.pl {
		g.renderView(i)
	}
	g.lighting()
	if g.split() {
		g.rect(SW/2-1, 0, SW/2+1, SH, 0x5A3418)
	}
	g.hud()
}
