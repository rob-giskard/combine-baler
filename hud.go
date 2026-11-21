package main

// Heads-up display: mostly icons. Text is kept for the menus and the
// morning report.

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
)

var palette = map[byte]uint32{
	'o': 0x5A3418, 'Y': 0xFFD23E, 'W': 0xFFF3B0, 'D': 0xD9A520, 'L': 0xFFE07A, 'B': 0xF2C24A, 'd': 0xD8A02E,
	'E': 0xA8781E, 'r': 0x7A121A, 'R': 0xC92F36, 's': 0xB8862E, 'G': 0x9AC83A, 'y': 0xFFE06A, 'M': 0xDDE6FF,
	'w': 0xF4EEE6, 'b': 0x8A5A34, 'c': 0xD9B38C, 'g': 0x9FDDEE, 'u': 0x3A7BD5, 'x': 0xE0402E, 'p': 0xFFC8BE,
	'k': 0x55555E, 'h': 0xFFF8D0, 'q': 0xD8402A, 'f': 0xFFFFFF, 'n': 0x8A5A14,
}

var artCoin = []string{
	"..nnnnn..",
	".nYYYYYn.",
	"nYWYYYYYn",
	"nYWYYDYYn",
	"nYYYDDYYn",
	"nYYYDDYYn",
	"nYYYYDYYn",
	".nYYYYYn.",
	"..nnnnn..",
}
var artBale = []string{
	"..oooooooooo..",
	".oLLLLLLLLLLo.",
	"oBBdBBBdBBBdBo",
	"oBBdBBBdBBBdBo",
	"oBBdBBBdBBBdBo",
	"oBBdBBBdBBBdBo",
	"oBBdBBBdBBBdBo",
	"oEEEEEEEEEEEEo",
	".oEEEEEEEEEEo.",
	"..oooooooooo..",
}
var artRoof = []string{
	"......rrrr......",
	"....rrRRRRrr....",
	"..rrRRRRRRRRrr..",
	"rrRRRRRRRRRRRRrr",
	"rrrrrrrrrrrrrrrr",
	".o............o.",
}
var artStalk = []string{
	"....Y....",
	"...YYY...",
	"...YYY...",
	".YY.Y.YY.",
	".YYYsYYY.",
	"..YYsYY..",
	".YY.s.YY.",
	".YYYsYYY.",
	"..YYsYY..",
	".YY.s.YY.",
	".YYYsYYY.",
	"..YYsYY..",
	"....s....",
	"....s..G.",
	"....s.GG.",
	"....sGG..",
	"....sG...",
	"....s....",
}
var artSun = []string{
	"....y....",
	".y..y..y.",
	"..yYYYy..",
	"..YYYYY..",
	"yyYYYYYyy",
	"..YYYYY..",
	"..yYYYy..",
	".y..y..y.",
	"....y....",
}
var artMoon = []string{
	"...MMM...",
	"..MMM....",
	".MMM.....",
	".MMM.....",
	".MMM.....",
	".MMMM....",
	"..MMMM.M.",
	"...MMMMM.",
	"....MMM..",
}
var artItem = [NItems][]string{{ // coffee
	"....w..w....",
	"...w..w.....",
	"....w..w....",
	".wwwwwwwwww.",
	".wwwwwwwwww.",
	"..bbbbbbbb..",
	"..bbccccbb..",
	"..bbccccbb..",
	"..bbbbbbbb..",
	"...bbbbbb...",
	"...bbbbbb...",
	"...bbbbbb...",
}, { // fizzy water
	".....qq.....",
	".....gg.....",
	".....gg.....",
	"....gggg....",
	"...gggggg...",
	"...gfgggg...",
	"...ggggfg...",
	"...uuuuuu...",
	"...uuuuuu...",
	"...ggfggg...",
	"...gggggg...",
	"....gggg....",
}, { // bouncy ball
	"....xxxx....",
	"..xxxxxxxx..",
	".xxppxxxxxx.",
	".xppxxxxxxx.",
	"xxpxxxxxxxxx",
	"xxxxxxxxxxxx",
	"wwwwwwwwwwww",
	"wwwwwwwwwwww",
	".xxxxxxxxxx.",
	".xxxxxxxxxx.",
	"..xxxxxxxx..",
	"....xxxx....",
}, { // headlamps
	"............",
	"y....YY.....",
	".y..YYYY....",
	"...YYhhYY.kk",
	"y.YYhhhhYYkk",
	"..YYhhhhYYkk",
	"y.YYhhhhYYkk",
	"...YYhhYY.kk",
	".y..YYYY....",
	"y....YY.....",
	"............",
	"............",
}, { // egg sandwich
	"............",
	"............",
	"...cccccc...",
	"..cWWWWWWc..",
	".cWWWWWWWWc.",
	".GGGGGGGGGG.",
	".wwwYYYwwww.",
	".wwYYYYYwww.",
	".cccccccccc.",
	".cWWWWWWWWc.",
	"..cccccccc..",
	"............",
}, { // cinnamon bun
	"............",
	"...bbbbbb...",
	"..bccccccb..",
	".bccbbbbccb.",
	".bcbccccbcb.",
	".bcbcbbcbcb.",
	".bcbcbccbcb.",
	".bcbccbbcbb.",
	".bccbbbccb..",
	"..bcccccb...",
	"...bbbbb....",
	"............",
}, { // kite
	".....xx.....",
	"....xxxx....",
	"...xxxxYY...",
	"..xxxxYYYY..",
	".xxxxYYYYYY.",
	"..YYYYxxxx..",
	"...YYYxxx...",
	"....YYxx....",
	".....w......",
	"......w.x...",
	".......w....",
	"......x.w...",
}, { // faster engines
	"....kkkk....",
	"...kffffk...",
	"..kfk..kfk..",
	".kfk.k..kfk.",
	".kf...k..fk.",
	".kf....kkfk.",
	".kf......fk.",
	"..kk....kk..",
	"............",
	"..yy.yy.yy..",
	".yy.yy.yy...",
	"............",
}, { // bigger truck bed
	"............",
	"oo........oo",
	"obBBBBBBBBbo",
	"obBdBBBdBBbo",
	"obBBBBBBBBbo",
	"oobBBBBBBboo",
	"ooBBBdBBBBoo",
	"obBBBBBBBBbo",
	"obbbbbbbbbbo",
	"oooooooooooo",
	".kk......kk.",
	"............",
}, { // wide reaper
	"............",
	"kkkkkkkkkkkk",
	"qxxxxxxxxxxq",
	"qwwwwwwwwwwq",
	"qxxxxxxxxxxq",
	"qwwwwwwwwwwq",
	"qqqqqqqqqqqq",
	"....GGGG....",
	"...GGGGGG...",
	"...GgggGG...",
	"...GGGGGG...",
	"............",
}}

var artBoost = []string{
	"...ff....",
	"....ff...",
	"ff...ff..",
	".ff...ff.",
	"..ff...ff",
	".ff...ff.",
	"ff...ff..",
	"....ff...",
	"...ff....",
}
var artBelly = []string{ // egg sandwich: full belly
	"..ccccc..",
	".cWWWWWc.",
	"cWWWWWWWc",
	"GGGGGGGGG",
	"wwYYYwwww",
	"ccccccccc",
	".cWWWWWc.",
	"..ccccc..",
}
var artBun = []string{
	"..bbbbb..",
	".bcccccb.",
	"bccbbbccb",
	"bcbcccbcb",
	"bcbcbcbcb",
	"bccbbccbb",
	".bcccccb.",
	"..bbbbb..",
}

// art draws string pixel art; with outline it gets a dark rim first.
func (g *Game) art(x, y, sc int, rows []string, outline bool) {
	if outline {
		for _, d := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
			for j, row := range rows {
				for i := 0; i < len(row); i++ {
					if row[i] != '.' {
						g.rect(x+i*sc+d[0], y+j*sc+d[1], x+(i+1)*sc+d[0], y+(j+1)*sc+d[1], 0x5A3418)
					}
				}
			}
		}
	}
	for j, row := range rows {
		for i := 0; i < len(row); i++ {
			if c, ok := palette[row[i]]; ok {
				g.rect(x+i*sc, y+j*sc, x+(i+1)*sc, y+(j+1)*sc, c)
			}
		}
	}
}

// keyCap draws a floating key with a small label over it.
func (g *Game) keyCap(bx, by int, key, label string) {
	by = max(by, g.clipY0+26)
	g.textO(bx-textW(label, 1)/2, by-22, label, 1, 0xFFFFFF)
	sc := 2
	if len(key) > 1 { // SPACE, NUM9, F13: smaller letters on a wider cap
		sc = 1
	}
	hw := max(9, (textW(key, sc)+7)/2)
	g.rect(bx-hw-1, by-11, bx+hw+1, by+10, 0x5A3418)
	g.rect(bx-hw, by-10, bx+hw, by+9, 0xFFF8E8)
	g.rect(bx-hw, by+6, bx+hw, by+9, 0xD8C8A8)
	if sc == 2 {
		g.text(bx-5, by-9, key, 2, 0x5A3418)
	} else {
		g.text(bx-textW(key, 1)/2, by-5, key, 1, 0x5A3418)
	}
	g.rect(bx-3, by+10, bx+3, by+12, 0x5A3418)
	g.rect(bx-1, by+12, bx+1, by+14, 0x5A3418)
}

func (g *Game) useKey(i int) string { return g.keyOf(i, AUse) }

// prompt floats a key cap over whatever a farmer on foot can use.
func (g *Game) prompt(i int, cm cam) {
	p := &g.pl[i]
	if p.veh >= 0 || p.shop || g.State != StPlay {
		return
	}
	kind, _, x, y := g.nearThing(p)
	if kind == UseNone {
		return
	}
	lift := 11.0
	if kind != UseVehicle {
		lift = 4
	}
	bob := int(math.Round(math.Sin(g.clock*5) * 2))
	g.keyCap(int(cm.sx(x)), int(cm.sy(y)-lift*cm.z)-14+bob, g.useKey(i), [5]string{"", "GET IN", "SLEEP", "SHOP", "HOSE"}[kind])
}

// hint is only shown for the first minute of the first day.
func (g *Game) hint(i int) string {
	p := &g.pl[i]
	if g.Day > 1 || g.hintT > 45 || p.veh < 0 {
		return ""
	}
	use, act := g.useKey(i), g.keyOf(i, AReap)
	if g.veh[p.veh].kind == VTruck {
		return "DRIVE OVER BALES   " + g.keyOf(i, ABoost) + " - BOOST"
	}
	return act + " - REAPER   " + use + " - EXIT"
}

func (g *Game) hud(scr int, slots []slot) {
	if g.State == StSettings {
		g.settingsScreen()
		return
	}
	if g.State == StTitle {
		g.rectA(0, 0, SW, SH, 0x2A1808, 160)
		g.textC(44, "COMBINE-BALER", 5, 0xFFE06A)
		opts := [4]string{"1 PLAYER", "2 PLAYERS", "3 PLAYERS", "SETTINGS"}
		for i, o := range opts {
			c := uint32(0xD8C8A8)
			if i == g.menuSel {
				c = 0xFFE06A
				o = "- " + o + " -"
			}
			g.textC(104+i*20, o, 2, c)
		}
		y := 194
		km := &g.Set.Keys
		line := func(p int) string {
			return fmt.Sprintf("PLAYER %d   %s   %s REAPER   %s USE   %s BOOST   %s HORN", p+1, moveKeys(km[p]),
				keyName(km[p][AReap]), keyName(km[p][AUse]), keyName(km[p][ABoost]), keyName(km[p][AHorn]))
		}
		switch g.menuSel {
		case 0:
			g.textC(y, line(0), 1, 0xFFFFFF)
			g.textC(y+12, "THE SECOND SET OF KEYS WORKS TOO", 1, 0xFFF4D6)
		case 1, 2:
			for p := 0; p <= g.menuSel; p++ {
				g.textC(y+p*12, line(p), 1, 0xFFFFFF)
			}
		default:
			g.textC(y, "KEYS, TREE COLLISIONS, MUSIC, ONE WINDOW PER PLAYER", 1, 0xFFFFFF)
		}
		g.textC(y+44, "HAUL BALES TO THE BARN - THEY SELL AT SUNDOWN - SLEEP IN THE BARN", 1, 0xFFF4D6)
		g.textC(y+58, "M  WHOLE FARM   P  SAVE PICTURE   F2  MUSIC   ESC  PAUSE", 1, 0xFFF4D6)
		if int(g.clock*2)%2 == 0 {
			g.textC(300, "UP / DOWN TO CHOOSE - ENTER TO START", 2, 0xFFE06A)
		}
		return
	}

	// top left: sun or moon, day number, daylight bar
	night := g.Night()
	if night {
		g.art(8, 7, 2, artMoon, true)
	} else {
		g.art(8, 7, 2, artSun, true)
	}
	day := fmt.Sprint(g.Day)
	g.textO(32, 9, day, 2, 0xFFFFFF)
	g.textO(32+textW(day, 2)+8, 16, g.ClockString(), 1, 0xFFF4D6)
	g.rect(9, 29, 93, 36, 0x5A3418)
	if night {
		g.rect(10, 30, 92, 35, 0x2A3566)
		g.rect(10, 30, 10+int(82*math.Min(1, (g.T-DayLen)/NightLen)), 35, 0xDDE6FF)
	} else {
		g.rect(10, 30, 92, 35, 0xB8905A)
		g.rect(10, 30, 10+int(82*g.T/DayLen), 35, 0xFFFFFF)
	}

	// right edge: the standing wheat, draining as it is reaped
	bx, top, bot := SW-15, 34, SH-10
	g.art(bx-1, 8, 1, artStalk, true)
	g.rect(bx-1, top-1, bx+8, bot+1, 0x5A3418)
	g.rect(bx, top, bx+7, bot, 0x8A5A34)
	left := 1 - float64(g.Stubble+g.Flat)/float64(g.WheatTotal)
	fy := bot - int(math.Ceil(float64(bot-top)*left))
	g.rect(bx, fy, bx+7, bot, 0xF4B82E)
	g.rect(bx, fy, bx+2, bot, 0xFFDC62)
	if fy < bot {
		g.rect(bx, fy, bx+7, fy+1, 0xFFF3B0)
	}

	// lower left: bales in the field, bales under the barn roof, coins
	x, y := 8, SH-24
	num := func(n int) {
		s := fmt.Sprint(n)
		g.textO(x, y+4, s, 2, 0xFFFFFF)
		x += textW(s, 2) + 12
	}
	g.art(x, y+6, 1, artBale, false)
	x += 18
	num(len(g.bales))
	g.art(x, y, 1, artRoof, false)
	g.art(x+1, y+6, 1, artBale, false)
	x += 20
	num(g.Stored)
	g.art(x, y+5, 1, artCoin, true)
	x += 13
	cx := x
	num(g.Coins)
	if g.cashFlash > 0 {
		g.textO(cx, y-10-int((4-g.cashFlash)*4), fmt.Sprintf("+%d", g.cashN), 2, 0xFFD23E)
	}

	for _, sl := range slots {
		i, r := sl.pl, sl.r
		p := &g.pl[i]
		mid := r.x + r.w/2
		g.clip0, g.clip1, g.clipY0, g.clipY1 = r.x, r.x+r.w, r.y, r.y+r.h
		if p.tired > 0.03 { // sleepiness
			g.textO(mid-24, r.y+8, "Z", 1, 0xDDE6FF)
			g.rect(mid-15, r.y+8, mid+17, r.y+15, 0x5A3418)
			g.rect(mid-14, r.y+9, mid+16, r.y+14, 0x2A3566)
			g.rect(mid-14, r.y+9, mid-14+int(30*p.tired), r.y+14, 0xDDE6FF)
		}
		if g.State == StPlay {
			g.status(i, mid, r.y+20)
			if p.shop {
				g.shopPanel(i, r)
			}
			if h := g.hint(i); h != "" {
				g.textO(r.x+9, r.y+r.h-42, h, 1, 0xFFFFFF)
			}
		}
		g.clip0, g.clip1, g.clipY0, g.clipY1 = 0, SW, 0, SH
	}
	if g.toastT > 0 {
		g.textC(SH-58, g.toast, 1, 0xFFFFFF)
	}

	if scr > 0 { // the second window only mirrors what the first one says
		switch g.State {
		case StPause:
			g.rectA(0, 0, SW, SH, 0x2A1808, 130)
			g.textC(150, "PAUSED", 4, 0xFFE06A)
		case StSummary:
			g.rectA(0, 0, SW, SH, 0x2A1808, 150)
			g.textC(150, "GOOD MORNING", 4, 0xFFE06A)
		}
		return
	}
	if g.State == StPause {
		g.rectA(0, 0, SW, SH, 0x2A1808, 130)
		g.textC(104, "PAUSED", 4, 0xFFE06A)
		g.textC(154, "ESC   BACK TO THE FIELD", 1, 0xFFFFFF)
		g.textC(168, "N     GO TO BED NOW", 1, 0xFFFFFF)
		g.textC(182, "S     SETTINGS", 1, 0xFFFFFF)
		g.textC(196, "M     MAIN MENU (STARTS A NEW FARM)", 1, 0xFFFFFF)
		g.textC(210, "F2    MUSIC ON / OFF", 1, 0xFFFFFF)
		g.textC(224, "Q     QUIT", 1, 0xFFFFFF)
	}
	if g.State == StSummary {
		g.summary()
	}
}

// moveKeys names a player's four movement keys briefly: WASD, ARROWS,
// NUMPAD 8456.
func moveKeys(km Keymap) string {
	k := [4]uint8{km[AUp], km[ALeft], km[ADown], km[ARight]}
	if k == [4]uint8{KUp, KLeft, KDown, KRight} {
		return "ARROWS"
	}
	short, num := "", "NUMPAD "
	for _, vk := range k {
		n := keyName(vk)
		if len(n) == 1 {
			short += n
		} else {
			short = "-"
		}
		if vk >= 0x60 && vk <= 0x69 {
			num += string(rune('0' + vk - 0x60))
		} else {
			num = "-"
		}
	}
	switch {
	case len(short) == 4:
		return short
	case len(num) == 11:
		return num
	}
	return keyName(k[0]) + " " + keyName(k[1]) + " " + keyName(k[2]) + " " + keyName(k[3])
}

// status is a player's little row of icons at the top of their view: mud
// on their machine, the truck boost, today's food.
func (g *Game) status(i, mid, y int) {
	p := &g.pl[i]
	var row []func(x int) int
	if p.veh >= 0 {
		v := &g.veh[p.veh]
		row = append(row, func(x int) int {
			g.mudClock(x+5, y+5, 5, v.mud)
			return 13
		})
		if v.kind == VTruck {
			row = append(row, func(x int) int {
				g.rect(x-1, y-1, x+11, y+11, 0x5A3418)
				g.rect(x, y, x+10, y+10, 0x2A3566)
				ready := 1 - math.Max(0, v.boostCD)/BoostCD
				g.rect(x, y+10-int(10*ready), x+10, y+10, 0x3A7BD5)
				if v.boostCD <= 0 && v.mud < 1 {
					g.art(x+1, y+1, 1, artBoost, false)
				}
				return 14
			})
		}
	}
	if p.fed {
		row = append(row, func(x int) int { g.art(x, y+1, 1, artBelly, true); return 12 })
	}
	if p.bun {
		row = append(row, func(x int) int { g.art(x, y+1, 1, artBun, true); return 12 })
	}
	w := 0
	for range row {
		w += 13
	}
	x := mid - w/2
	for _, f := range row {
		x += f(x)
	}
}

// shopPanel is the vending machine. Four goods show at a time; the row
// slides along as the selection moves.
func (g *Game) shopPanel(i int, r rect) {
	p := &g.pl[i]
	w, h := 140, 78
	px := r.x + r.w/2 - w/2
	py := min(r.y+r.h/2+24, r.y+r.h-h-6)
	if p.shake > 0 {
		px += int(math.Sin(p.shake*60) * 3)
	}
	g.rect(px-2, py-2, px+w+2, py+h+2, 0x1C6B55)
	g.rect(px, py, px+w, py+h, 0xFFF8E8)
	g.rect(px, py, px+w, py+13, 0x2FA07E)
	g.art(px+5, py+2, 1, artCoin, false)
	g.text(px+18, py+3, fmt.Sprint(g.Coins), 1, 0xFFFFFF)
	g.text(px+w-48, py+3, "< "+g.useKey(i)+" >", 1, 0xFFFFFF)
	first := max(0, min(p.shopSel-1, NItems-4))
	if first > 0 {
		g.text(px-1, py+36, "<", 1, 0x1C6B55)
	}
	if first+4 < NItems {
		g.text(px+w-4, py+36, ">", 1, 0x1C6B55)
	}
	for k := first; k < first+4; k++ {
		sx, sy := px+6+(k-first)*33, py+17
		if k == p.shopSel {
			g.rect(sx-2, sy-2, sx+30, sy+44, 0xFFD23E)
			g.rect(sx, sy, sx+28, sy+42, 0xFFF3B0)
			sy -= int(math.Round(math.Abs(math.Sin(g.clock*5)) * 2))
		}
		g.art(sx+2, sy+2, 2, artItem[k], false)
		if k == ItemEngine { // level pips
			for l := 0; l < EngineMax; l++ {
				c := uint32(0xD8C8A8)
				if l < g.Engine {
					c = 0x2FA07E
				}
				g.rect(sx+5+l*7, sy+26, sx+10+l*7, sy+28, c)
			}
		}
		if g.soldOut(i, k) { // already have it: a green tick
			for t := 0; t < 4; t++ {
				g.rect(sx+8+t, sy+32+t, sx+10+t, sy+34+t, 0x2FA07E)
			}
			for t := 0; t < 7; t++ {
				g.rect(sx+12+t, sy+35-t, sx+14+t, sy+37-t, 0x2FA07E)
			}
			continue
		}
		g.art(sx+3, sy+30, 1, artCoin, false)
		c := uint32(0x5A3418)
		if g.Coins < g.price(k) {
			c = 0xC8402A
		}
		g.text(sx+15, sy+31, fmt.Sprint(g.price(k)), 1, c)
	}
	name := g.itemName(p.shopSel)
	g.text(px+(w-textW(name, 1))/2, py+h-12, name, 1, 0x5A3418)
}

func (g *Game) summary() {
	g.rectA(0, 0, SW, SH, 0x2A1808, 150)
	g.rect(98, 28, 542, 332, 0x5A3418)
	g.rect(100, 30, 540, 330, 0xFFF0D0)
	g.text(116, 40, fmt.Sprintf("GOOD MORNING - DAY %d IS DONE", g.Day), 2, 0x7A3B1C)
	// the farm as it stands: 1 px per 5 cells
	tx, ty := 116, 62
	g.rect(tx-2, ty-2, tx+322, ty+202, 0x7A3B1C)
	for y := 0; y < 200; y++ {
		for x := 0; x < 320; x++ {
			g.fb[(ty+y)*SW+tx+x] = g.cellColor(x*5+2, y*5+2)
		}
	}
	x := 450
	dark, mid := uint32(0x5A3418), uint32(0xB5601E)
	g.art(x, 62, 1, artStalk[:12], false)
	g.text(x+14, 62, "CUT", 1, mid)
	g.text(x+14, 72, fmt.Sprintf("%.1f T", g.Tonnes(g.CutToday)), 1, dark)
	g.text(x, 92, "ALL DAYS", 1, mid)
	g.text(x, 102, fmt.Sprintf("%.1f T", g.Tonnes(g.CutTotal)), 2, dark)
	g.text(x, 124, "WHEAT LEFT", 1, mid)
	g.text(x, 134, fmt.Sprintf("%.0f%%", 100-g.FieldPct()), 2, dark)
	g.art(x, 158, 1, artBale, false)
	g.text(x+18, 160, fmt.Sprintf("%d SOLD", g.Cashed), 1, dark)
	g.art(x+2, 176, 1, artCoin, false)
	g.text(x+18, 177, fmt.Sprint(g.Coins), 2, dark)
	g.text(x, 202, "LEDGER", 1, mid)
	from := max(0, len(g.Log)-4)
	for i, l := range g.Log[from:] {
		g.text(x, 212+i*10, fmt.Sprintf("D%d %.1fT %dB", l.Day, l.Tonnes, l.Bales), 1, dark)
	}
	g.text(116, 272, "ENTER  WAKE UP, KEEP THE FIELD AS IT IS", 1, dark)
	g.text(116, 283, "R      WAKE UP TO FRESHLY REPLANTED WHEAT", 1, dark)
	g.text(116, 294, "P      SAVE THIS FARM AS A PNG PICTURE", 1, dark)
	g.text(116, 305, "ESC    QUIT", 1, dark)
	if g.toastT > 0 {
		g.text(240, 305, g.toast, 1, mid)
	}
}

// SavePicture writes the whole farm, as cut so far, to a PNG (1 px per
// cell) with the current stats on a strip underneath.
func (g *Game) SavePicture(dir string) (string, error) {
	const strip = 30
	img := image.NewRGBA(image.Rect(0, 0, WW, WH+strip))
	set := func(x, y int, c uint32) {
		img.SetRGBA(x, y, color.RGBA{uint8(c >> 16), uint8(c >> 8), uint8(c), 255})
	}
	for y := 0; y < WH+strip; y++ {
		for x := 0; x < WW; x++ {
			if y < WH {
				set(x, y, g.cellColor(x, y))
			} else {
				set(x, y, 0x5A3418)
			}
		}
	}
	line := fmt.Sprintf("COMBINE-BALER    DAY %d  %s    CUT %.1f T    WHEAT LEFT %.0f%%    BALES MADE %d    COINS %d",
		g.Day, g.ClockString(), g.Tonnes(g.CutTotal), 100-g.FieldPct(), g.BaleCount, g.Coins)
	x := 16
	for _, ch := range line {
		if gl, ok := font[ch]; ok {
			for i := 0; i < 35; i++ {
				if gl[i] == '#' {
					for k := 0; k < 4; k++ {
						set(x+(i%5)*2+k%2, WH+8+(i/5)*2+k/2, 0xFFF0D0)
					}
				}
			}
		}
		x += 12
	}
	g.saveN++
	name := fmt.Sprintf("combine_baler_day%d_%d.png", g.Day, g.saveN)
	f, err := os.Create(dir + name)
	if err != nil {
		return "", err
	}
	defer f.Close()
	return name, png.Encode(f, img)
}
