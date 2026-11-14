package main

import (
	"encoding/binary"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"testing"
)

const dt = 1.0 / 60

func dump(t *testing.T, g *Game, name string) {
	g.Render()
	img := image.NewRGBA(image.Rect(0, 0, SW, SH))
	for i, c := range g.fb {
		img.SetRGBA(i%SW, i/SW, color.RGBA{uint8(c >> 16), uint8(c >> 8), uint8(c), 255})
	}
	os.MkdirAll("out", 0755)
	f, err := os.Create("out/" + name + ".png")
	if err != nil {
		t.Fatal(err)
	}
	png.Encode(f, img)
	f.Close()
}

// check asserts the things that must hold on every single frame.
func check(g *Game) {
	seated := 0
	for i, p := range g.pl {
		if p.veh >= 0 {
			if g.veh[p.veh].driver != i {
				panic("driver bookkeeping broken")
			}
			seated++
		} else if g.walkBlocked(p.x, p.y) {
			panic("farmer inside something solid")
		}
	}
	loads := 0
	for _, v := range g.veh {
		if v.driver >= 0 {
			seated--
		}
		if g.solidAt(v.x, v.y) {
			panic("vehicle inside something solid")
		}
		if v.load < 0 || v.load > TruckCap {
			panic("truck load out of range")
		}
		loads += v.load
	}
	if seated != 0 {
		panic("driver count mismatch")
	}
	if g.Stored+loads+len(g.bales) > g.BaleCount {
		panic("bales appeared from nowhere")
	}
}

func step(g *Game, k *[256]bool) {
	g.Update(dt, k)
	check(g)
}

func run(g *Game, secs float64, held ...int) {
	var k [256]bool
	for _, h := range held {
		k[h] = true
	}
	for i := 0; i < int(secs*60); i++ {
		step(g, &k)
	}
}

func tap(g *Game, key int) {
	var k [256]bool
	k[key] = true
	step(g, &k)
	k[key] = false
	step(g, &k)
}

type pad struct{ up, down, left, right, act, use int }

var pads = [2]pad{{KW, KS, KA, KD, KSpace, KE}, {KUp, KDown, KLeft, KRight, KL, KK}}

// driveTo steers player pi's vehicle towards a point; stops when done() or close.
func driveTo(g *Game, pi int, tx, ty float64, done func() bool) bool {
	pd := pads[pi]
	lx, ly := -1.0, -1.0
	for n := 0; n < 60*60; n++ {
		v := &g.veh[g.pl[pi].veh]
		dx, dy := tx-v.x, ty-v.y
		if (done != nil && done()) || (done == nil && math.Hypot(dx, dy) < 5) {
			return true
		}
		diff := math.Atan2(dx, -dy) - v.ang
		for diff > math.Pi {
			diff -= 2 * math.Pi
		}
		for diff < -math.Pi {
			diff += 2 * math.Pi
		}
		var k [256]bool
		k[pd.left] = diff < -0.04
		k[pd.right] = diff > 0.04
		k[pd.up] = !(math.Abs(diff) > 0.5 && v.speed > 8)
		step(g, &k)
		if n%90 == 89 { // wedged against another machine: back out and go round
			if math.Hypot(v.x-lx, v.y-ly) < 2 {
				run(g, 1.2, pd.down, pd.left)
				run(g, 0.8, pd.up)
			}
			lx, ly = v.x, v.y
		}
	}
	return false
}

func walkTo(g *Game, pi int, tx, ty float64) bool {
	pd := pads[pi]
	for n := 0; n < 60*90; n++ {
		p := &g.pl[pi]
		dx, dy := tx-p.x, ty-p.y
		if math.Hypot(dx, dy) < 2 {
			return true
		}
		var k [256]bool
		k[pd.left], k[pd.right] = dx < -0.7, dx > 0.7
		k[pd.up], k[pd.down] = dy < -0.7, dy > 0.7
		step(g, &k)
	}
	return false
}

func TestSinglePlayer(t *testing.T) {
	g := NewGame()
	if g.WheatTotal < 1200000 {
		t.Fatalf("wheat %d", g.WheatTotal)
	}
	for i := range g.veh {
		if g.blocked(i, g.veh[i].x, g.veh[i].y, g.veh[i].ang) {
			t.Fatal("start position blocked")
		}
	}
	dump(t, g, "01_menu")
	tap(g, KEnter)
	if g.State != StPlay || g.NPlayers != 1 || len(g.veh) != 2 || g.pl[0].veh != 0 {
		t.Fatal("single player start")
	}
	c := &g.veh[0]
	y0 := c.y
	run(g, 3, KUp) // arrows also work for one player
	if c.y <= y0+40 || g.CutToday != 0 || g.pl[0].zoom != 4 {
		t.Fatalf("drive: y %.1f cut %d zoom %.2f", c.y, g.CutToday, g.pl[0].zoom)
	}
	dump(t, g, "02_drive_close")

	tap(g, KSpace)
	run(g, 5, KW)
	if g.CutToday < 1500 || math.Abs(g.pl[0].zoom-1.6) > 0.05 {
		t.Fatalf("reap: cut %d zoom %.2f", g.CutToday, g.pl[0].zoom)
	}
	w := 0
	for x := 0; x < WW; x++ {
		if g.tile[int(c.y-20)*WW+x] == TStubble {
			w++
		}
	}
	if w < 17 || w > 20 {
		t.Fatalf("swathe width %d", w)
	}
	run(g, 5, KW, KD)
	run(g, 4, KW)
	run(g, 4, KW, KA)
	run(g, 4, KW)
	dump(t, g, "03_reaping")
	made := g.BaleCount
	if made < 5 || len(g.bales) != made {
		t.Fatalf("bales %d", made)
	}

	// climb out, walk back to the truck, climb in
	tap(g, KE)
	p := &g.pl[0]
	if p.veh != -1 || c.driver != -1 || c.header {
		t.Fatal("exit combine")
	}
	cut := g.CutToday
	run(g, 1, KA)
	dump(t, g, "04_on_foot")
	tr := &g.veh[1]
	if !walkTo(g, 0, tr.x-8.5, tr.y+18) || !walkTo(g, 0, tr.x-8.5, tr.y) {
		t.Fatalf("walk to truck stuck at %.0f,%.0f", p.x, p.y)
	}
	if g.CutToday != cut {
		t.Fatal("walking cut wheat")
	}
	dump(t, g, "05_at_truck")
	tap(g, KE)
	if p.veh != 1 || tr.driver != 0 {
		t.Fatal("enter truck")
	}

	// collect every bale
	for len(g.bales) > 0 && tr.load < TruckCap {
		b, l := g.bales[0], tr.load
		if !driveTo(g, 0, float64(b.x)+3, float64(b.y)+2.5, func() bool { return tr.load > l }) {
			t.Fatalf("could not reach bale at %d,%d (truck %.0f,%.0f)", b.x, b.y, tr.x, tr.y)
		}
	}
	if tr.load != min(made, TruckCap) || len(g.bales) != made-tr.load {
		t.Fatalf("load %d field %d made %d", tr.load, len(g.bales), made)
	}
	dump(t, g, "06_truck_loaded")
	// field must show no ghost bale pixels
	ghosts := 0
	for i, o := range g.obj {
		if o == 0xF2C24A && g.tile[i] == TStubble {
			ghosts++
		}
	}
	if len(g.bales) == 0 && ghosts != 0 {
		t.Fatalf("%d ghost bale pixels", ghosts)
	}

	// deliver
	load := tr.load
	px, py := float64(PadX0+PadX1)/2, float64(PadY0+PadY1)/2
	if !driveTo(g, 0, px-10, py+70, nil) || !driveTo(g, 0, px, py+2, func() bool { return tr.load == 0 }) {
		t.Fatalf("delivery failed, truck at %.0f,%.0f load %d", tr.x, tr.y, tr.load)
	}
	if g.Stored != load {
		t.Fatalf("stored %d of %d", g.Stored, load)
	}
	dump(t, g, "07_delivered")

	tap(g, KM)
	run(g, 3)
	dump(t, g, "08_map")
	tap(g, KM)

	// pause freezes the clock
	tap(g, KEsc)
	t0 := g.T
	run(g, 1, KW)
	if g.State != StPause || g.T != t0 {
		t.Fatal("pause")
	}
	tap(g, KEsc)

	// climb out, walk to the barn door and sleep: the bales sell overnight
	tap(g, KE)
	if p.veh != -1 {
		t.Fatal("exit truck")
	}
	if !walkTo(g, 0, doorX, doorY+8) || !walkTo(g, 0, doorX, doorY+1) {
		t.Fatalf("walk to barn stuck at %.0f,%.0f", p.x, p.y)
	}
	if kind, _, _, _ := g.nearThing(p); kind != UseDoor {
		t.Fatal("no sleep prompt at the barn door")
	}
	dump(t, g, "09_sleep_prompt")
	tap(g, KE)
	if g.State != StSummary || len(g.Log) != 1 || g.Log[0].Bales != load || g.Coins != load*BalePrice || g.Stored != 0 {
		t.Fatalf("sleep: state %d coins %d stored %d", g.State, g.Coins, g.Stored)
	}
	tap(g, KP)
	dump(t, g, "10_summary")
	if _, err := os.Stat("combine_baler_day1_1.png"); err != nil {
		t.Fatal("picture not saved")
	}
	os.Rename("combine_baler_day1_1.png", "out/11_saved_picture.png")

	total, stub := g.CutTotal, g.Stubble
	tap(g, KEnter)
	if g.State != StPlay || g.Day != 2 || g.CutToday != 0 || g.Cashed != 0 || g.CutTotal != total || g.Stubble != stub {
		t.Fatal("day 2")
	}
	if p.veh != -1 || tr.driver != -1 || c.driver != -1 || p.tired != 0 {
		t.Fatal("farmer should wake up on foot at the barn")
	}

	// vending machine
	if !walkTo(g, 0, shopX, shopY+1) {
		t.Fatal("walk to vending machine")
	}
	dump(t, g, "12_shop_prompt")
	tap(g, KE)
	if !p.shop {
		t.Fatal("shop did not open")
	}
	g.Coins = 6
	x0 := p.x
	run(g, 0.5, KS, KD) // held keys must not walk the farmer away or spam
	run(g, 0.1)
	if !p.shop || p.shopSel != 1 {
		t.Fatal("held keys should move the selection once and keep the shop open")
	}
	p.shopSel = 0
	if p.x != x0 {
		t.Fatal("walked while shopping")
	}
	tap(g, KD)
	tap(g, KSpace) // fizzy water
	if g.Coins != 5 || p.fizz <= 0 {
		t.Fatalf("fizzy: coins %d", g.Coins)
	}
	tap(g, KD)
	tap(g, KSpace) // bouncy ball
	if g.Coins != 1 || len(g.balls) != 1 {
		t.Fatalf("ball: coins %d balls %d", g.Coins, len(g.balls))
	}
	tap(g, KD)
	tap(g, KSpace) // headlamps: cannot afford
	if g.Coins != 1 || g.Lamps {
		t.Fatal("bought lamps without coins")
	}
	dump(t, g, "13_shop")
	tap(g, KS)
	if p.shop {
		t.Fatal("shop did not close")
	}
	// kick the ball about
	bl := &g.balls[0]
	run(g, 2)
	bx, by := bl.x, bl.y
	walkTo(g, 0, bl.x, bl.y)
	run(g, 1)
	if bl.x == bx && bl.y == by {
		t.Fatal("ball did not move when kicked")
	}

	// sundown sells what is on the pad
	g.Stored = 5
	g.rebake(nil)
	sp := stackSpot(2)
	if g.obj[(sp.y+2)*WW+sp.x+2] == 0 {
		t.Fatal("stack not drawn")
	}
	g.T = DayLen - 1
	run(g, 2)
	if g.Coins != 6 || g.Stored != 0 || g.Cashed != 5 || g.obj[(sp.y+2)*WW+sp.x+2] != 0 {
		t.Fatalf("sundown: coins %d stored %d", g.Coins, g.Stored)
	}
	dump(t, g, "14_sundown")

	// night work makes the farmer tired and slow; coffee fixes it
	run(g, 70)
	if p.tired < 0.35 {
		t.Fatalf("tired %.2f", p.tired)
	}
	walkTo(g, 0, shopX, shopY+1)
	dump(t, g, "15_night_tired")
	g.Coins = 20
	tap(g, KE)
	p.shopSel = ItemLamps
	tap(g, KSpace)
	p.shopSel = ItemCoffee
	tap(g, KSpace)
	if !g.Lamps || p.tired > 0.01 || g.Coins != 6 {
		t.Fatalf("lamps %v tired %.2f coins %d", g.Lamps, p.tired, g.Coins)
	}
	tap(g, KSpace) // coffee when wide awake: paid for, nothing happens
	if g.Coins != 4 || p.tired > 0.01 {
		t.Fatal("second coffee")
	}
	tap(g, KW)
	walkTo(g, 0, tr.x, tr.y+12)
	tap(g, KE)
	if p.veh != 1 {
		t.Fatalf("night truck: farmer %.0f,%.0f truck %.0f,%.0f", p.x, p.y, tr.x, tr.y)
	}
	run(g, 1.5, KS)
	dump(t, g, "16_night_drive")

	// working through the night rolls into the next day, still tired
	for n := 0; g.Day == 2 && n < 100; n++ {
		run(g, 5)
	}
	if g.Day != 3 || g.State != StPlay || len(g.Log) != 2 || p.tired < 0.5 || g.T > 6 {
		t.Fatalf("all-nighter: day %d tired %.2f", g.Day, p.tired)
	}

	// bed from the pause menu, wake up to replanted wheat
	tap(g, KEsc)
	tap(g, KN)
	if g.State != StSummary || len(g.Log) != 3 {
		t.Fatal("go to bed from pause")
	}
	tap(g, KR)
	if g.Day != 4 || g.Stubble != 0 || len(g.bales) != 0 || p.tired != 0 {
		t.Fatal("replant")
	}
	n := 0
	for _, tl := range g.tile {
		if tl == TWheat {
			n++
		}
	}
	if n != g.WheatTotal {
		t.Fatal("wheat not restored")
	}

	// back to the main menu resets the farm
	tap(g, KEsc)
	tap(g, KM)
	if g.State != StTitle || g.Day != 1 || g.Coins != 0 || g.CutTotal != 0 || g.Lamps {
		t.Fatal("main menu reset")
	}
}

func TestTwoPlayers(t *testing.T) {
	g := NewGame()
	tap(g, KDown)
	dump(t, g, "20_menu_two")
	tap(g, KEnter)
	if g.NPlayers != 2 || len(g.veh) != 4 || g.pl[0].veh != 0 || g.pl[1].veh != 2 {
		t.Fatal("two player start")
	}
	a, b := &g.veh[0], &g.veh[2]
	dump(t, g, "21_split_start")

	// each pad moves only its own machine
	ay, by := a.y, b.y
	run(g, 2, KUp)
	if a.y != ay || b.y <= by+10 {
		t.Fatal("arrows should drive player 2 only")
	}
	run(g, 2) // let it roll to a stop
	by = b.y
	run(g, 2, KW)
	if b.y != by || a.y <= ay+10 {
		t.Fatal("WASD should drive player 1 only")
	}
	tap(g, KSpace)
	if !a.header || b.header {
		t.Fatal("space is player 1's reaper")
	}
	tap(g, KL)
	if !b.header {
		t.Fatal("L is player 2's reaper")
	}
	run(g, 7, KW, KUp)
	run(g, 4, KW, KUp, KD, KLeft)
	run(g, 3, KW, KUp)
	if g.pl[0].zoom > 1.4 || g.pl[1].zoom > 1.4 || g.BaleCount < 4 {
		t.Fatalf("zoom %.2f %.2f bales %d", g.pl[0].zoom, g.pl[1].zoom, g.BaleCount)
	}
	dump(t, g, "22_split_reaping")

	// player 2 hops out and fetches a truck while player 1 keeps reaping
	tap(g, KK)
	if g.pl[1].veh != -1 || g.pl[0].veh != 0 {
		t.Fatal("K is player 2's exit")
	}
	tap(g, KE)
	tap(g, KE) // player 1 out and straight back in
	if g.pl[0].veh != 0 {
		t.Fatal("player 1 re-enter")
	}
	tr := &g.veh[3]
	if !walkTo(g, 1, tr.x+8.5, tr.y+18) || !walkTo(g, 1, tr.x+8.5, tr.y) {
		t.Fatal("player 2 walk")
	}
	dump(t, g, "23_split_p2_on_foot")
	tap(g, KK)
	if g.pl[1].veh != 3 {
		t.Fatal("player 2 truck")
	}
	run(g, 2, KUp) // pull out of the yard, clear of the other truck
	bl := g.bales[0]
	if !driveTo(g, 1, float64(bl.x)+3, float64(bl.y)+2.5, func() bool { return tr.load > 0 }) {
		t.Fatalf("player 2 bale at %d,%d: truck %.0f,%.0f ang %.2f speed %.1f; combines %.0f,%.0f %.0f,%.0f", bl.x, bl.y, tr.x, tr.y, tr.ang, tr.speed, a.x, a.y, b.x, b.y)
	}
	dump(t, g, "24_split_truck")
	tap(g, KM)
	run(g, 3)
	dump(t, g, "25_split_map")
	// a seat is for one: player 1 cannot take player 2's truck
	tap(g, KE)
	walkTo(g, 0, tr.x-8.5, tr.y)
	tap(g, KE)
	if g.pl[0].veh == 3 {
		t.Fatal("stole an occupied truck")
	}
}

// random key mashing in both modes: check() must hold on every frame.
func TestFuzz(t *testing.T) {
	for mode := 0; mode < 2; mode++ {
		g := NewGame()
		if mode == 1 {
			tap(g, KDown)
		}
		tap(g, KEnter)
		tap(g, KSpace)
		tap(g, KL)
		run(g, 10, KW, KUp) // out into the wheat first
		var k [256]bool
		seed := uint32(777 + mode)
		rnd := func(n uint32) uint32 {
			seed = seed*1664525 + 1013904223
			return (seed >> 10) % n
		}
		hold := []int{KUp, KDown, KLeft, KRight, KW, KA, KS, KD}
		rare := []int{KSpace, KL, KE, KK, KSpace, KL, KM, KEsc, KEnter, KN, KR, KF2}
		for i := 0; i < 200000; i++ {
			if rnd(25) == 0 {
				key := hold[rnd(8)]
				k[key] = !k[key] || rnd(3) == 0 && (key == KUp || key == KW)
			}
			for _, r := range rare {
				k[r] = false
			}
			if rnd(150) == 0 {
				k[rare[rnd(uint32(len(rare)))]] = true
			}
			if g.State == StPause {
				k[KM] = false // do not wipe the farm; that path is tested elsewhere
			}
			step(g, &k)
			g.Quit = false
			if g.State == StTitle { // went back to the menu: start again
				k = [256]bool{}
				tap(g, KEnter)
			}
			if i%1499 == 0 {
				g.Render()
			}
		}
		t.Logf("mode %d: day %d, %.1f t, %d bales made, %d stored", mode+1, g.Day, g.Tonnes(g.CutTotal), g.BaleCount, g.Stored)
	}
}

func TestAudio(t *testing.T) {
	a := NewAudio()
	buf := make([]int16, SampleRate/60)
	var all []int16
	for f := 0; f < 60*40; f++ {
		a.SetState(f > 300, 0.8, f > 900, 1, f > 600 && f < 1500)
		if f%45 == 0 {
			a.Play(f / 45 % 11)
		}
		a.Fill(buf)
		all = append(all, buf...)
	}
	peak, sum := 0.0, 0.0
	for _, s := range all {
		v := math.Abs(float64(s))
		peak = math.Max(peak, v)
		sum += v * v
	}
	rms := math.Sqrt(sum / float64(len(all)))
	t.Logf("peak %.0f rms %.0f", peak, rms)
	if peak > 26000 || rms < 800 || rms > 7000 {
		t.Fatal("levels out of the cosy range")
	}
	os.MkdirAll("out", 0755)
	f, _ := os.Create("out/audio_preview.wav")
	hdr := []any{[]byte("RIFF"), uint32(36 + len(all)*2), []byte("WAVEfmt "), uint32(16), uint16(1), uint16(1),
		uint32(SampleRate), uint32(SampleRate * 2), uint16(2), uint16(16), []byte("data"), uint32(len(all) * 2)}
	for _, h := range hdr {
		binary.Write(f, binary.LittleEndian, h)
	}
	binary.Write(f, binary.LittleEndian, all)
	f.Close()

	// music off and nothing running must be true silence
	a = NewAudio()
	a.ToggleMusic()
	a.Fill(buf)
	for _, s := range buf {
		if s != 0 {
			t.Fatal("not silent")
		}
	}
}

func BenchmarkFrame(b *testing.B) {
	g := NewGame()
	tap(g, KDown)
	tap(g, KEnter)
	var k [256]bool
	k[KUp], k[KW] = true, true
	for i := 0; i < b.N; i++ {
		g.Update(dt, &k)
		g.Render()
	}
}
