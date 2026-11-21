package main

// ---------------------------------------------------------------- world

func (g *Game) tileAt(x, y int) uint8 {
	if x < 0 || y < 0 || x >= WW || y >= WH {
		return TGrass
	}
	return g.tile[y*WW+x]
}

func (g *Game) fillTiles(x0, y0, x1, y1 int, t uint8) {
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			g.tile[y*WW+x] = t
		}
	}
}

// plot returns the bounds of the field in column c, row r of the farm grid.
func plot(c, r int) (x0, y0, x1, y1 int) {
	x0 = Margin + c*(FieldW+Lane)
	y0 = Margin + r*(FieldH+Lane)
	return x0, y0, x0 + FieldW, y0 + FieldH
}

func (g *Game) buildWorld() {
	g.fillTiles(0, 0, WW, WH, TGrass)
	for r := 0; r < GridH; r++ {
		for c := 0; c < GridW; c++ {
			if c == YardC && r == YardR {
				continue // the farmstead
			}
			x0, y0, x1, y1 := plot(c, r)
			g.fillTiles(x0, y0, x1, y1, TWheat)
		}
	}
	// a few fields are joined into bigger ones by planting over the lane
	for _, m := range [][3]int{{0, 2, 0}, {3, 0, 0}, {3, 2, 1}, {0, 0, 1}} {
		x0, y0, x1, y1 := plot(m[0], m[1])
		if m[2] == 0 {
			g.fillTiles(x1, y0, x1+Lane, y1, TWheat)
		} else {
			g.fillTiles(x0, y1, x1, y1+Lane, TWheat)
		}
	}
	_, _, _, yardBottom := plot(YardC, YardR)
	g.fillTiles(386+OX, 90+OY, 414+OX, yardBottom, TDirt) // drive from the house
	g.fillTiles(350+OX, 86+OY, 450+OX, 96+OY, TDirt)      // yard in front of the house
	g.fillTiles(PadX0, PadY0, PadX1, PadY1, TDirt)        // bale pad
	g.WheatTotal = 0
	for i := range g.tile {
		if g.tile[i] == TWheat {
			g.WheatTotal++
		}
	}
	for y := 0; y < WH; y++ {
		for x := 0; x < WW; x++ {
			g.recolor(x, y)
		}
	}
	g.bakeFarm()
	g.plantTrees()
}

// tuftArc is the rounded top of one clump of wheat, 8 cells wide.
var tuftArc = [8]int{0, 1, 2, 2, 2, 2, 1, 0}

// wheatTex paints standing wheat as rows of soft rounded clumps.
func wheatTex(x, y int) uint32 {
	row := y / 6
	xs := x + (row&1)*4 + int(hash(row, 3)%3)
	u, v := xs%8, y%6
	top := 2 - tuftArc[u]
	switch {
	case v < top:
		return 0xEAA923 // crease between two clumps
	case v == top:
		return 0xFFDC62 // lit rim
	case v == top+1 && u >= 2 && u <= 5:
		return 0xFACB45
	}
	switch hash(xs/8, row) % 5 {
	case 0:
		return 0xF0B027
	case 1:
		return 0xF7BE36
	}
	return 0xF4B82E
}

func (g *Game) recolor(x, y int) {
	if x < 0 || y < 0 || x >= WW || y >= WH {
		return
	}
	h := hash(x, y)
	var c uint32
	switch g.tile[y*WW+x] {
	case TWheat:
		c = wheatTex(x, y)
		if g.tileAt(x, y+1) != TWheat {
			c = 0xC07C16 // front face of the standing crop
		} else if g.tileAt(x, y+2) != TWheat {
			c = 0xDA961C
		} else if g.tileAt(x, y-1) != TWheat {
			c = 0xFFE06A
		}
	case TFlat: // pressed down: darker, streaked along the tyre track
		c = mul(wheatTex(x, y), 196, 178, 150)
		if h%5 == 0 {
			c = mul(c, 220, 215, 205)
		}
		if g.tileAt(x, y-1) == TWheat {
			c = 0xC07C16
		}
	case TStubble:
		c = 0xA87442
		if y%3 == 0 {
			c = 0x98663A
		}
		if h%29 == 0 {
			c = 0xC9A05C
		}
		if g.tileAt(x, y-1) == TWheat {
			c = 0x74492A
		} else if g.tileAt(x, y-2) == TWheat {
			c = 0x8A5A34
		}
	case TDirt:
		c = 0xD0A262
		if h%13 == 0 {
			c = 0xC79852
		}
		if h%31 == 0 {
			c = 0xE2BA7C
		}
	default:
		c = 0xA9C53B
		if h%13 == 0 {
			c = 0xBDD650
		} else if h%17 == 0 {
			c = 0x93B230
		}
		if h%211 == 0 {
			c = 0xFFF6D8 // daisy
		} else if h%263 == 0 {
			c = 0xFFD24A
		}
		if g.tileAt(x, y-1) == TWheat {
			c = 0x7F9C2A
		} else if g.tileAt(x, y-2) == TWheat {
			c = 0x93B230
		}
	}
	g.col[y*WW+x] = c
}

func (g *Game) oRect(x0, y0, x1, y1 int, c uint32, solid bool) {
	x0, x1, y0, y1 = x0+OX, x1+OX, y0+OY, y1+OY
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if x < 0 || y < 0 || x >= WW || y >= WH {
				continue
			}
			g.obj[y*WW+x] = c
			if solid {
				g.kind[y*WW+x] = CHard
			}
		}
	}
}

func (g *Game) oDisc(cx, cy, r int, c uint32, solid bool) {
	for y := cy - r; y <= cy+r; y++ {
		for x := cx - r; x <= cx+r; x++ {
			dx, dy := x-cx, y-cy
			if dx*dx+dy*dy <= r*r+r/2 {
				g.oRect(x, y, x+1, y+1, c, solid)
			}
		}
	}
}

func (g *Game) shade(x0, y0, x1, y1 int) {
	x0, x1, y0, y1 = x0+OX, x1+OX, y0+OY, y1+OY
	for y := y0; y < y1; y++ {
		for x := x0; x < x1; x++ {
			if g.obj[y*WW+x] == 0 {
				g.obj[y*WW+x] = mul(g.col[y*WW+x], 200, 200, 215)
			}
		}
	}
}

func (g *Game) bakeFarm() {
	// barn: gambrel gable front, red planks, white trim
	const cx, top = 400, 22
	white, roof := uint32(0xF4EEE6), uint32(0x3E3E46)
	g.shade(366, 90, 440, 93)
	g.shade(434, 60, 439, 90)
	for y := top; y < 90; y++ {
		hw := 34
		if y < top+14 {
			hw = 3 + (y-top)*22/14
		} else if y < top+36 {
			hw = 25 + (y-top-14)*12/22
		}
		for x := cx - hw - 2; x < cx+hw+2; x++ {
			d := x - cx
			if d < 0 {
				d = -d - 1
			}
			c := uint32(0xB3202A)
			if x%5 == 0 {
				c = 0x8E1620 // gaps between planks
			}
			if y < top+36 {
				switch {
				case y < top+2 || d == hw+1:
					c = 0x5A5A64
				case d >= hw-2:
					c = roof
				}
			} else if d >= 34 {
				continue
			}
			g.oRect(x, y, x+1, y+1, c, true)
		}
	}
	g.oRect(366, 58, 434, 60, 0x7A121A, true) // shade under the eaves
	g.oRect(366, 88, 434, 90, 0x7A121A, true)
	xbox := func(x0, y0, x1, y1 int, fill uint32) {
		g.oRect(x0, y0, x1, y1, white, true)
		g.oRect(x0+1, y0+1, x1-1, y1-1, fill, true)
		for y := y0 + 1; y < y1-1; y++ {
			t := (y - y0 - 1) * (x1 - x0 - 3) / (y1 - y0 - 3)
			g.oRect(x0+1+t, y, x0+2+t, y+1, white, true)
			g.oRect(x1-2-t, y, x1-1-t, y+1, white, true)
		}
	}
	xbox(388, 37, 398, 47, 0x9E1A24) // loft hatches
	xbox(402, 37, 412, 47, 0x9E1A24)
	g.oRect(380, 61, 420, 63, roof, true) // door rail
	for _, x0 := range []int{384, 400} {
		xbox(x0, 63, x0+16, 77, 0xA51C26)
		xbox(x0, 76, x0+16, 90, 0xA51C26)
	}
	for _, x0 := range []int{369, 421} { // windows
		g.oRect(x0, 66, x0+10, 78, white, true)
		g.oRect(x0+1, 67, x0+9, 77, 0x4A4A55, true)
		g.oRect(x0+4, 67, x0+6, 77, white, true)
		g.oRect(x0+1, 71, x0+9, 73, white, true)
	}
	g.bakeHoseStation()
	// coffee vending machine against the barn's right side
	g.shade(438, 92, 452, 94)
	g.oRect(438, 68, 450, 92, 0x1C6B55, true)
	g.oRect(439, 69, 449, 91, 0x2FA07E, true)
	g.oRect(439, 69, 449, 72, 0xFFE06A, true) // lit sign
	g.oRect(440, 73, 448, 83, 0xD8F4FA, true) // window with goods
	for i, c := range []uint32{0xD8402A, 0x3A7BD5, 0xF2C24A, 0x8A5A34, 0xF4EEE6, 0xD8402A} {
		g.oRect(441+i%3*2+i%3/2, 74+i/3*4, 443+i%3*2+i%3/2, 77+i/3*4, c, true)
	}
	g.oRect(440, 78, 448, 79, 0x9AC8D4, true)
	g.oRect(441, 85, 447, 89, 0x14493A, true) // tray
	g.oRect(447, 80, 449, 82, 0xFFF4D6, true) // coin slot
	// corner posts of the bale pad (walk/drive-through markers)
	for _, p := range [][2]int{{380, 96}, {418, 96}, {380, 124}, {418, 124}} {
		g.oRect(p[0], p[1], p[0]+2, p[1]+2, 0xE8702A, false)
	}
}

func (g *Game) bakeBale(b Bale, on bool) {
	for dy := 0; dy < 5; dy++ {
		for dx := 0; dx < 6; dx++ {
			x, y := b.x+dx, b.y+dy
			if x < 0 || y < 0 || x >= WW || y >= WH || g.kind[y*WW+x] != CNone {
				continue
			}
			if !on {
				g.obj[y*WW+x] = 0
				continue
			}
			if (dx == 0 || dx == 5) && (dy == 0 || dy == 4) {
				continue
			}
			c := uint32(0xF2C24A)
			switch {
			case dy == 4:
				c = 0xA8781E
			case dy == 0:
				c = 0xFFE07A
			case dx == 2 || dx == 4:
				c = 0xD8A02E
			}
			g.obj[y*WW+x] = c
		}
	}
}

func (g *Game) Replant() {
	old := g.bales
	g.bales = nil
	for i := range g.tile {
		if g.tile[i] == TStubble || g.tile[i] == TFlat {
			g.tile[i] = TWheat
		}
	}
	g.Stubble, g.Flat = 0, 0
	for i := range g.veh {
		g.veh[i].baleAcc = 0
	}
	for y := 0; y < WH; y++ {
		for x := 0; x < WW; x++ {
			g.recolor(x, y)
		}
	}
	g.rebake(old)
}

// rebake clears removed bales, then redraws the bales left in the field and
// the stack on the pad (bales can overlap one another).
func (g *Game) rebake(removed []Bale) {
	for _, b := range removed {
		g.bakeBale(b, false)
	}
	for _, b := range g.bales {
		g.bakeBale(b, true)
	}
	for n := 0; n < min(g.Stored, StackMax); n++ {
		g.bakeBale(stackSpot(n), true)
	}
}

const StackMax = 72

// stackSpot is where the n-th delivered bale sits on the pad.
func stackSpot(n int) Bale {
	return Bale{PadX0 + 2 + n%6*6, PadY0 + 4 + n%24/6*6 - n/24*2}
}
