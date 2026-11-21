package main

// Screens and views. A screen is one window's 640x360 framebuffer; it holds
// one to three player views. Layouts:
//   1 player: full screen
//   2 players: side by side (or one window each)
//   3 players: two on top, one wide below (or split over two windows)

type rect struct{ x, y, w, h int }

type slot struct {
	pl int
	r  rect
}

var (
	rFull = rect{0, 0, SW, SH}
	rL    = rect{0, 0, SW/2 - 1, SH}
	rR    = rect{SW/2 + 1, 0, SW/2 - 1, SH}
	rTL   = rect{0, 0, SW/2 - 1, SH/2 - 1}
	rTR   = rect{SW/2 + 1, 0, SW/2 - 1, SH/2 - 1}
	rB    = rect{0, SH/2 + 1, SW, SH/2 - 1}
)

// Screens is how many windows the game wants right now.
func (g *Game) Screens() int {
	if g.Set.Windows == 2 && g.NPlayers >= 2 && g.State != StTitle {
		return 2
	}
	return 1
}

func (g *Game) layout(s int) []slot {
	n := g.NPlayers
	if g.State == StTitle {
		n = 1
	}
	if g.Screens() == 1 {
		switch n {
		case 1:
			return []slot{{0, rFull}}
		case 2:
			return []slot{{0, rL}, {1, rR}}
		}
		return []slot{{0, rTL}, {1, rTR}, {2, rB}}
	}
	switch {
	case n == 2 && s == 0:
		return []slot{{0, rFull}}
	case n == 2:
		return []slot{{1, rFull}}
	case g.Set.Win2P23 && s == 0:
		return []slot{{0, rFull}}
	case g.Set.Win2P23:
		return []slot{{1, rL}, {2, rR}}
	case s == 0:
		return []slot{{0, rL}, {1, rR}}
	}
	return []slot{{2, rFull}}
}

// viewOf returns the rectangle player i is drawn into.
func (g *Game) viewOf(i int) rect {
	for s := 0; s < g.Screens(); s++ {
		for _, sl := range g.layout(s) {
			if sl.pl == i {
				return sl.r
			}
		}
	}
	return rFull
}

func (g *Game) camOf(i int, r rect) cam {
	p := &g.pl[i]
	return cam{p.camX, p.camY, p.zoom, r}
}

// Render draws the first screen into g.fb.
func (g *Game) Render() { g.RenderScreen(0) }

// RenderScreen draws screen s into its own framebuffer and returns it.
func (g *Game) RenderScreen(s int) []uint32 {
	if g.fbs[s] == nil {
		g.fbs[s] = make([]uint32, SW*SH)
	}
	g.fb = g.fbs[s]
	slots := g.layout(s)
	if len(slots) > 1 {
		g.rect(0, 0, SW, SH, 0x5A3418) // the gaps between views
	}
	for _, sl := range slots {
		g.renderView(sl.pl, sl.r)
	}
	g.lighting(slots)
	g.hud(s, slots)
	return g.fb
}
