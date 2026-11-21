package main

import (
	"math"
	"os"
	"testing"
)

// start1 begins a one-player game with the farmer in the combine.
func start1(t *testing.T) *Game {
	g := NewGame()
	g.Set = defaultSettings()
	tap(g, KEnter)
	if g.State != StPlay {
		t.Fatal("start")
	}
	return g
}

// seat puts player i straight into vehicle j.
func seat(g *Game, i, j int) {
	p := &g.pl[i]
	if p.veh >= 0 {
		g.veh[p.veh].driver = -1
	}
	p.veh = j
	g.veh[j].driver = i
}

// laneTree is a tree out on the farm (not one of the yard's).
func laneTree(g *Game) *Tree {
	for i := 8; i < len(g.trees); i++ {
		t := &g.trees[i]
		if g.tileAt(int(t.x), int(t.baseY())+40) == TWheat || g.tileAt(int(t.x), int(t.baseY())+40) == TGrass {
			return t
		}
	}
	return nil
}

func TestTreesPlanted(t *testing.T) {
	g := NewGame()
	if len(g.trees) < 20 {
		t.Fatalf("only %d trees", len(g.trees))
	}
	pals := map[int]bool{}
	overhang := 0
	for i := range g.trees {
		tr := &g.trees[i]
		pals[tr.pal] = true
		if g.tileAt(int(tr.x), int(tr.baseY())) != TGrass {
			t.Fatalf("tree %d trunk not on grass", i)
		}
		if g.cellKind(tr.x, tr.baseY()-0.5) != CTree {
			t.Fatalf("tree %d trunk not solid", i)
		}
		for a := 0.0; a < 2*math.Pi; a += 0.3 {
			if g.tileAt(int(tr.x+math.Cos(a)*tr.r), int(tr.y+math.Sin(a)*tr.r)) == TWheat {
				overhang++
				break
			}
		}
	}
	if len(pals) != 4 || overhang < 10 {
		t.Fatalf("palettes %d, crowns over wheat %d", len(pals), overhang)
	}
	// no crate, no pots
	for i := range g.obj {
		if g.obj[i] == 0xB5773A && g.tile[i] == TGrass || g.obj[i] == 0xC8592A {
			t.Fatal("crate or pot still in the yard")
		}
	}
}

// aim places the truck below tree tr, offset sideways by off, facing north.
func aim(g *Game, tr *Tree, off, dist float64) *Vehicle {
	seat(g, 0, 1)
	v := &g.veh[1]
	v.x, v.y, v.ang, v.speed = tr.x+off, tr.baseY()+dist, 0, 0
	g.pl[0].zoom = g.zoomTarget(0)
	g.updateCamera(0)
	return v
}

func TestSmoothCollisions(t *testing.T) {
	g := start1(t)
	g.Set.Smooth = true
	tr := laneTree(g)
	// glancing: slides past the trunk instead of sticking to it
	v := aim(g, tr, 6, 30)
	run(g, 2.5, KW)
	if v.y > tr.baseY()-15 {
		t.Fatalf("glancing hit got stuck: truck %.1f,%.1f trunk %.1f,%.1f", v.x, v.y, tr.x, tr.baseY())
	}
	dump(t, g, "30_smooth_slide")
	// head on: stalls against the trunk, backs away cleanly
	v = aim(g, tr, 0, 30)
	run(g, 2.5, KW)
	if v.y < tr.baseY() || v.y > tr.baseY()+16 {
		t.Fatalf("head on: truck at %.1f, trunk %.1f", v.y, tr.baseY())
	}
	y := v.y
	run(g, 1.5, KS)
	if v.y < y+8 {
		t.Fatal("could not back away from the tree")
	}
	// the barn always slides, even in bouncy mode
	g.Set.Smooth = false
	v.x, v.y, v.ang, v.speed, v.kx, v.ky = doorX+30, doorY+20, -0.5, 0, 0, 0
	run(g, 2, KW)
	if v.kx != 0 || v.ky != 0 {
		t.Fatal("the barn bounced the truck")
	}
}

func TestBouncyTrees(t *testing.T) {
	g := start1(t)
	tr := laneTree(g)
	v := aim(g, tr, 0, 45)
	closest := 1e9
	for n := 0; n < 60*3; n++ {
		var k [256]bool
		k[KW] = true
		step(g, &k)
		closest = math.Min(closest, math.Hypot(v.x-tr.x, v.y-tr.baseY()))
	}
	if closest < 8 {
		t.Fatalf("soft crown let the truck reach the trunk (%.1f)", closest)
	}
	dump(t, g, "31_bouncy_crown")
	run(g, 1)
	if math.Hypot(v.x-tr.x, v.y-tr.baseY()) < closest+2 {
		t.Fatal("crown did not push the truck back out")
	}
	// a farmer walking into a trunk bounces off with a hop
	tap(g, KE)
	p := &g.pl[0]
	p.x, p.y = tr.x, tr.baseY()+10
	for n := 0; n < 120 && p.hop <= 0; n++ {
		var k [256]bool
		k[KW] = true
		step(g, &k)
	}
	if p.hop <= 0 {
		t.Fatal("farmer did not bounce")
	}
	dump(t, g, "32_farmer_bounce")
}

func TestBoost(t *testing.T) {
	g := start1(t)
	seat(g, 0, 1)
	v := &g.veh[1]
	// out in a field of standing wheat
	x0, y0, _, y1 := plot(1, 2)
	v.x, v.y, v.ang = float64(x0+100), float64(y1-20), 0
	run(g, 1, KW)
	z := g.pl[0].zoom
	base := v.speed
	tap(g, KQ)
	if v.boost <= 0 || v.speed < base*1.4 {
		t.Fatalf("boost: %.1f -> %.1f", base, v.speed)
	}
	run(g, DashTime+0.1, KW)
	if g.Flat < 40 || g.tileAt(int(v.x+4.5), int(v.y+8)) == TStubble {
		t.Fatalf("flattened %d cells", g.Flat)
	}
	if g.pl[0].zoom >= z {
		t.Fatal("no zoom out while boosting")
	}
	dump(t, g, "33_boost")
	flat := g.Flat
	tap(g, KQ) // still cooling down
	run(g, BoostTime-DashTime, KW)
	if g.Flat != flat || v.boost > 0 {
		t.Fatalf("boost on cooldown fired again: flat %d->%d boost %.2f cd %.2f", flat, g.Flat, v.boost, v.boostCD)
	}
	if v.speed > 46*1.05 {
		t.Fatalf("speed-up did not fade: %.1f", v.speed)
	}
	_ = y0
	dump(t, g, "34_tracks")
	// flattened wheat cannot be baled, and grows back on replant
	g.sleep()
	tap(g, KR)
	if g.Flat != 0 || g.tileAt(int(v.x), int(v.y)) == TFlat {
		t.Fatal("replant should restore pressed wheat")
	}
	// clogged trucks cannot boost
	seat(g, 0, 1)
	v.mud, v.boostCD = 1, 0
	tap(g, KQ)
	if v.boost > 0 {
		t.Fatal("boosted while clogged")
	}
	// the combine has no boost
	seat(g, 0, 0)
	tap(g, KQ)
	if g.veh[0].boost > 0 {
		t.Fatal("combine boosted")
	}
}

func TestMudAndHose(t *testing.T) {
	g := start1(t)
	c := &g.veh[0]
	run(g, 3, KW)
	tap(g, KSpace)
	run(g, 10, KW)
	if c.mud < 10/MudCombine*0.8 || c.mud > 10/MudCombine*1.2 {
		t.Fatalf("mud after 10 s reaping %.3f", c.mud)
	}
	// a clogged combine is slow
	tap(g, KSpace)
	c.mud = 1
	top := 0.0
	for n := 0; n < 180; n++ {
		var k [256]bool
		k[KW] = true
		step(g, &k)
		top = math.Max(top, c.speed)
	}
	if top > 32*0.56 || top < 32*0.5 {
		t.Fatalf("clogged top speed %.1f", top)
	}
	dump(t, g, "35_clogged")
	// park it by the tap, climb out, take the hose, wash it
	c.x, c.y, c.ang, c.speed = tapX+4, tapY+30, math.Pi, 0
	tap(g, KE)
	p := &g.pl[0]
	if !walkTo(g, 0, tapX, tapY) {
		t.Fatal("walk to tap")
	}
	if kind, _, _, _ := g.nearThing(p); kind != UseHose {
		t.Fatal("no hose prompt")
	}
	dump(t, g, "36_hose_prompt")
	tap(g, KE)
	if !p.hose {
		t.Fatal("did not take the hose")
	}
	walkTo(g, 0, c.x, c.y-17)
	run(g, 0.05, KS) // face the combine
	run(g, WashTime*0.5, KSpace)
	dump(t, g, "37_washing")
	run(g, WashTime*0.6, KSpace)
	if c.mud != 0 || c.sparkle <= 0 {
		t.Fatalf("still muddy %.2f", c.mud)
	}
	tap(g, KE) // hang the hose up
	if p.hose {
		t.Fatal("hose not hung up")
	}
	// walking off with the hose snaps it back
	walkTo(g, 0, tapX, tapY)
	tap(g, KE)
	run(g, 7, KS)
	if p.hose {
		t.Fatal("hose stretched forever")
	}
	// trucks get muddy driving the field
	seat(g, 0, 1)
	tr := &g.veh[1]
	tr.x, tr.y, tr.ang = float64(Margin+150), float64(Margin+200), 0
	run(g, 20, KW, KD)
	if tr.mud < 0.05 {
		t.Fatalf("truck mud %.3f", tr.mud)
	}
}

func TestShopExtras(t *testing.T) {
	g := start1(t)
	tap(g, KE)
	p := &g.pl[0]
	if !walkTo(g, 0, shopX, shopY+1) {
		t.Fatal("walk to shop")
	}
	tap(g, KE)
	g.Coins = 200
	buy := func(k int) {
		p.shopSel = k
		tap(g, KSpace)
	}
	p.tired = 0.6
	buy(ItemEgg)
	if p.tired > 0.31 || !p.fed || g.Coins != 197 {
		t.Fatalf("egg: tired %.2f coins %d", p.tired, g.Coins)
	}
	buy(ItemBun)
	if !p.bun {
		t.Fatal("bun")
	}
	for k := 0; k < EngineMax+1; k++ {
		buy(ItemEngine)
	}
	if g.Engine != EngineMax || g.Coins != 194-10-20-35 {
		t.Fatalf("engines %d coins %d", g.Engine, g.Coins)
	}
	buy(ItemBed)
	buy(ItemReaper)
	buy(ItemKite)
	if !g.BigBed || !g.WideReap || !p.kite || g.truckCap() != BigBedCap {
		t.Fatal("upgrades")
	}
	coins := g.Coins
	buy(ItemKite) // one kite each
	if g.Coins != coins {
		t.Fatal("bought a second kite")
	}
	p.shopSel = NItems - 2
	dump(t, g, "38_shop_scrolled")
	tap(g, KS)
	// kite flies downwind of the farmer
	kx, ky := p.kiteX, p.kiteY
	run(g, 3, KA)
	if math.Hypot(p.kiteX-kx, p.kiteY-ky) < 5 {
		t.Fatal("kite did not fly")
	}
	dump(t, g, "39_kite")
	// faster engines and the bun: a quicker truck
	p.tired = 0
	tr := &g.veh[1]
	walkTo(g, 0, tr.x-8.5, tr.y)
	tap(g, KE)
	if p.veh != 1 {
		t.Fatal("into the truck")
	}
	run(g, 2, KW)
	if tr.speed < 46*1.36*1.1*0.95 {
		t.Fatalf("upgraded speed %.1f", tr.speed)
	}
	// a wider swath
	seat(g, 0, 0)
	c := &g.veh[0]
	c.x, c.y, c.ang = float64(Margin+150), float64(Margin+60), math.Pi
	tap(g, KSpace)
	run(g, 3, KW)
	w := 0
	for x := 0; x < WW; x++ {
		if g.tile[int(c.y-20)*WW+x] == TStubble {
			w++
		}
	}
	if w < 23 || w > 26 {
		t.Fatalf("wide swath %d", w)
	}
	dump(t, g, "40_wide_reaper")
	// the bun wears off overnight
	g.sleep()
	tap(g, KEnter)
	if p.bun || p.fed || !p.kite {
		t.Fatal("food should last one day, the kite for good")
	}
}

func TestHorns(t *testing.T) {
	g := NewGame()
	g.Set = defaultSettings()
	tap(g, KDown)
	tap(g, KEnter)
	ring := func() int {
		n := 0
		for _, pt := range g.parts {
			if pt.ring {
				n++
			}
		}
		return n
	}
	// player 2 out on foot next to player 1's combine
	tap(g, KK)
	q := &g.pl[1]
	walkTo(g, 1, g.veh[0].x+14, g.veh[0].y)
	tap(g, KR)
	if ring() != 1 || q.hop <= 0 {
		t.Fatalf("horn: rings %d hop %.2f", ring(), q.hop)
	}
	dump(t, g, "41_horn")
	tap(g, KR) // too soon for another
	if ring() != 1 {
		t.Fatal("horn spam")
	}
	run(g, 0.5)
	tap(g, KR)
	if ring() < 1 || g.veh[0].hornCD <= 0 {
		t.Fatal("second honk")
	}
}

func TestThreePlayers(t *testing.T) {
	g := NewGame()
	g.Set = defaultSettings()
	tap(g, KDown)
	tap(g, KDown)
	dump(t, g, "50_menu_three")
	tap(g, KEnter)
	if g.NPlayers != 3 || len(g.veh) != 6 || g.pl[2].veh != 4 {
		t.Fatal("three player start")
	}
	for i := range g.veh {
		if g.blocked(i, g.veh[i].x, g.veh[i].y, g.veh[i].ang) {
			t.Fatalf("machine %d parked blocked", i)
		}
	}
	if g.viewOf(0) != rTL || g.viewOf(1) != rTR || g.viewOf(2) != rB {
		t.Fatal("layout")
	}
	dump(t, g, "51_three_start")
	c := &g.veh[4]
	y := c.y
	run(g, 2, KNum8)
	if c.y < y+20 || g.veh[0].y != g.veh[0].sy || g.veh[2].y != g.veh[2].sy {
		t.Fatal("numpad should drive player 3 only")
	}
	tap(g, KNum7)
	if !c.header {
		t.Fatal("numpad 7 is player 3's reaper")
	}
	run(g, 6, KNum8, KW, KUp)
	run(g, 3, KNum8, KNum4, KW, KD, KUp, KLeft)
	dump(t, g, "52_three_reaping")
	tap(g, KNum9)
	if g.pl[2].veh != -1 {
		t.Fatal("numpad 9 is player 3's use")
	}
	tap(g, KM)
	run(g, 3)
	dump(t, g, "53_three_map")
	tap(g, KM)

	// one window per player
	g.Set.Windows = 2
	if g.Screens() != 2 || g.viewOf(2) != rFull || g.viewOf(0) != rL {
		t.Fatal("two windows, player 3 alone")
	}
	run(g, 3)
	dumpScreen(t, g, 1, "54_window2_p3")
	g.Set.Win2P23 = true
	run(g, 3)
	if g.viewOf(0) != rFull || g.viewOf(2) != rR {
		t.Fatal("two windows, players 2 and 3 together")
	}
	dump(t, g, "55_window1_p1")
	dumpScreen(t, g, 1, "56_window2_p23")
	tap(g, KEsc)
	dumpScreen(t, g, 1, "57_window2_paused")
	tap(g, KEsc)
	// everyone wakes up somewhere free
	g.sleep()
	tap(g, KEnter)
	for i := range g.pl {
		if g.walkBlocked(g.pl[i].x, g.pl[i].y) {
			t.Fatal("woke up inside the barn")
		}
	}
}

func dumpScreen(t *testing.T, g *Game, s int, name string) {
	fb := g.RenderScreen(s)
	saved := g.fb
	g.fb = fb
	g.fbs[0], fb = fb, g.fbs[0]
	dumpFB(t, g.fb, name)
	g.fbs[0] = fb
	g.fb = saved
}

func TestSettings(t *testing.T) {
	os.Remove(fileDir + settingsFile)
	defer os.Remove(fileDir + settingsFile)
	g := NewGame()
	if g.Set != defaultSettings() {
		t.Fatal("defaults")
	}
	for k := 0; k < 3; k++ {
		tap(g, KDown)
	}
	tap(g, KEnter)
	if g.State != StSettings {
		t.Fatal("settings did not open")
	}
	tap(g, KEnter) // trees: bouncy -> smooth
	if !g.Set.Smooth {
		t.Fatal("tree toggle")
	}
	dump(t, g, "60_settings")
	for g.setSel != setKeys1 {
		tap(g, KDown)
	}
	tap(g, KEnter)
	if g.setPage != 1 {
		t.Fatal("keys page")
	}
	for g.setSel != ABoost {
		tap(g, KDown)
	}
	tap(g, KEnter)
	if !g.waitKey {
		t.Fatal("not waiting for a key")
	}
	dump(t, g, "61_settings_keys")
	tap(g, KL) // already player 2's reaper: the two swap
	if g.Set.Keys[0][ABoost] != KL || g.Set.Keys[1][AReap] != KQ {
		t.Fatalf("swap: p1 boost %X p2 reaper %X", g.Set.Keys[0][ABoost], g.Set.Keys[1][AReap])
	}
	tap(g, KEnter)
	tap(g, 0x7C) // F13 from a macro pad
	if g.Set.Keys[0][ABoost] != 0x7C || keyName(0x7C) != "F13" {
		t.Fatal("F13")
	}
	tap(g, KEsc)
	tap(g, KEsc)
	if g.State != StTitle {
		t.Fatal("back to the title")
	}
	// saved next to the game and read back
	if s := loadSettings(); s != g.Set {
		t.Fatalf("reload: %+v", s)
	}
	// the new keys drive the game
	g.menuSel = 0
	tap(g, KEnter)
	g.pl[0].veh = -1
	g.veh[0].driver = -1
	seat(g, 0, 1)
	var k [256]bool
	k[0x7C] = true
	step(g, &k)
	if g.veh[1].boost <= 0 {
		t.Fatal("F13 did not boost")
	}
	os.Remove(fileDir + settingsFile)
}
