package main

// Collisions. Every hit gets a contact normal, so a machine can slide along
// whatever it touches instead of stopping dead. Trees (and other free
// standing map objects) bounce instead when the Bouncy setting is on, which
// is the default; the barn and other machines always slide.

import "math"

// what fills a cell, for movement
const (
	CNone    uint8 = iota
	CHard          // barn, vending machine, tap, the edge of the world
	CTree          // tree trunks and other free standing objects
	CMachine       // not a cell kind: another vehicle (contact only)
)

var vehRadius = [2]float64{9, 7.5}

var truckProbes = [][2]float64{{0, 0}, {-5, 9}, {5, 9}, {-5, -9}, {5, -9}, {0, 9}, {0, -9}, {-5, 0}, {5, 0}}

// probes are points on a machine's outline, in its own frame (y forward).
func (g *Game) probes(v *Vehicle) [][2]float64 {
	if v.kind == VTruck {
		return truckProbes
	}
	h := g.headerHalf()
	return [][2]float64{{0, 0}, {-5, -8}, {5, -8}, {-h, 10}, {h, 10}, {-5, 5}, {5, 5}, {0, 10},
		{-h / 2, 10}, {h / 2, 10}, {0, -8}}
}

// headerHalf is half the width of the combine's cutting header.
func (g *Game) headerHalf() float64 {
	if g.WideReap {
		return 12
	}
	return 9
}

func (g *Game) cellKind(wx, wy float64) uint8 {
	ix, iy := int(math.Floor(wx)), int(math.Floor(wy))
	if ix < 1 || iy < 1 || ix >= WW-1 || iy >= WH-1 {
		return CHard
	}
	return g.kind[iy*WW+ix]
}

func (g *Game) solidAt(wx, wy float64) bool { return g.cellKind(wx, wy) != CNone }

// normalAt points away from whatever is solid around (px, py).
func (g *Game) normalAt(px, py, cx, cy float64) (nx, ny float64) {
	for k := 0; k < 16; k++ {
		a := float64(k) * math.Pi / 8
		s, c := math.Sin(a), math.Cos(a)
		if !g.solidAt(px+c*2.5, py+s*2.5) {
			nx += c
			ny += s
		}
	}
	if l := math.Hypot(nx, ny); l > 0.5 {
		return nx / l, ny / l
	}
	nx, ny = cx-px, cy-py // buried: point back at the machine
	l := math.Hypot(nx, ny) + 1e-9
	return nx / l, ny / l
}

// contact tests vehicle vi at pose (x, y, a). It returns what it touches and
// a unit normal pointing away from it.
func (g *Game) contact(vi int, x, y, a float64) (k uint8, nx, ny float64) {
	v := &g.veh[vi]
	s, c := math.Sin(a), math.Cos(a)
	for _, p := range g.probes(v) {
		px, py := x+p[0]*c+p[1]*s, y+p[0]*s-p[1]*c
		if kk := g.cellKind(px, py); kk != CNone {
			if kk == CTree {
				// a round trunk: push the machine's centre away from the
				// trunk's, so a machine that clips it with a corner is
				// steered round it rather than held
				if t := g.nearestTree(px, py); t != nil {
					nx, ny = x-t.x, y-(t.baseY()-1)
					l := math.Hypot(nx, ny) + 1e-9
					return kk, nx / l, ny / l
				}
			}
			nx, ny = g.normalAt(px, py, x, y)
			return kk, nx, ny
		}
	}
	for j := range g.veh {
		if j == vi {
			continue
		}
		o := &g.veh[j]
		dn := math.Hypot(x-o.x, y-o.y)
		if dn < vehRadius[v.kind]+vehRadius[o.kind] && dn < math.Hypot(v.x-o.x, v.y-o.y) {
			return CMachine, (x - o.x) / (dn + 1e-9), (y - o.y) / (dn + 1e-9)
		}
	}
	return CNone, 0, 0
}

// blocked is kept for the tests and the parking check.
func (g *Game) blocked(vi int, x, y, a float64) bool {
	k, _, _ := g.contact(vi, x, y, a)
	return k != CNone
}

func (g *Game) bouncy() bool { return !g.Set.Smooth }

// move advances vehicle vi by one step of its own speed plus knockback,
// turning to na. It slides along walls and bounces off trees.
func (g *Game) move(vi int, na, dt float64) {
	v := &g.veh[vi]
	fx, fy := math.Sin(na), -math.Cos(na)
	mx, my := fx*v.speed+v.kx, fy*v.speed+v.ky
	nx, ny := v.x+mx*dt, v.y+my*dt
	k, cx, cy := g.contact(vi, nx, ny, na)
	if k == CNone {
		v.x, v.y, v.ang, v.contact = nx, ny, na, false
		return
	}
	if k2, _, _ := g.contact(vi, nx, ny, v.ang); k2 == CNone { // only the turn was in the way
		v.x, v.y, v.contact = nx, ny, false
		v.spin = 0
		return
	}
	vn := mx*cx + my*cy // speed into the obstacle (negative)
	if k == CTree && g.bouncy() {
		if vn < 0 {
			kick := -vn*1.9 + 14
			v.kx += cx * kick
			v.ky += cy * kick
			v.speed *= 0.3
			side := fx*cy - fy*cx
			v.spin = math.Copysign(2.6, side) * math.Min(1, -vn/25)
			if v.bumpT <= 0 && -vn > 4 {
				g.Snd.Play(SBoing)
				v.bumpT = 0.25
				g.leafBurst(v.x-cx*8, v.y-cy*8, 6)
			}
		}
		v.contact = true
		return
	}
	if vn < 0 {
		if !v.contact && -vn > 9 && v.bumpT <= 0 {
			g.Snd.Play(SBump)
			v.bumpT = 0.3
		}
		if !v.contact {
			v.speed *= 0.85
		}
		frac := -vn / (math.Hypot(mx, my) + 1e-9)
		v.speed *= math.Max(0, 1-frac*3*dt) // pushing straight at a wall stalls
		if kn := v.kx*cx + v.ky*cy; kn < 0 {
			v.kx -= kn * cx
			v.ky -= kn * cy
		}
		mx -= vn * cx
		my -= vn * cy
	}
	v.contact = true
	// slide along the surface; round things (trunks) curve away, so also
	// try easing out along the normal
	sx, sy := v.x+mx*dt, v.y+my*dt
	slide := math.Hypot(mx, my) * dt               // scale the easing by how far it slides,
	for _, push := range [3]float64{0, 0.5, 1.2} { // so a head-on push doesn't jitter
		tx, ty := sx+cx*push*slide, sy+cy*push*slide
		if k3, _, _ := g.contact(vi, tx, ty, v.ang); k3 == CNone {
			v.x, v.y = tx, ty
			return
		}
	}
}

// unstick nudges a machine out of anything it has ended up overlapping
// (after a spin, say), so it can never wedge for good.
func (g *Game) unstick(vi int) {
	v := &g.veh[vi]
	k, nx, ny := g.contact(vi, v.x, v.y, v.ang)
	if k == CNone || k == CMachine {
		return
	}
	if x, y := v.x+nx*0.6, v.y+ny*0.6; !g.solidAt(x, y) {
		v.x, v.y = x, y
	}
}

// knock decays knockback and spin.
func (v *Vehicle) knock(dt float64) {
	d := math.Exp(-4 * dt)
	v.kx *= d
	v.ky *= d
	v.spin *= math.Exp(-5 * dt)
	if math.Abs(v.kx) < 0.05 {
		v.kx = 0
	}
	if math.Abs(v.ky) < 0.05 {
		v.ky = 0
	}
	if v.bumpT > 0 {
		v.bumpT -= dt
	}
}
