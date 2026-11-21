package main

// Settings: per-player key bindings, tree collisions, music and windows.
// They live in combine_baler_settings.txt next to the exe and are edited on
// the settings screen (title menu or pause menu).

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// fileDir is where the game reads and writes its files: next to the exe.
// Tests point it at out/.
var fileDir = func() string {
	if exe, err := os.Executable(); err == nil {
		return filepath.Dir(exe) + string(os.PathSeparator)
	}
	return ""
}()

const settingsFile = "combine_baler_settings.txt"

// player actions that can be bound to a key
const (
	AUp = iota
	ADown
	ALeft
	ARight
	AReap
	AUse
	ABoost
	AHorn
	NActions
)

var actionNames = [NActions]string{"UP", "DOWN", "LEFT", "RIGHT", "REAPER / SPRAY", "USE", "TRUCK BOOST", "HORN"}

type Keymap [NActions]uint8

// numpad virtual keys (Num Lock on)
const (
	KNum1 = 0x61
	KNum3 = 0x63
	KNum4 = 0x64
	KNum5 = 0x65
	KNum6 = 0x66
	KNum7 = 0x67
	KNum8 = 0x68
	KNum9 = 0x69
)

var defaultKeys = [3]Keymap{
	{KW, KS, KA, KD, KSpace, KE, KQ, KR},
	{KUp, KDown, KLeft, KRight, KL, KK, KO, KI},
	{KNum8, KNum5, KNum4, KNum6, KNum7, KNum9, KNum1, KNum3},
}

type Settings struct {
	Keys    [3]Keymap
	Smooth  bool // trees and map objects: smooth slide instead of bouncy
	MusicOn bool
	Windows int  // 1, or 2 = one window per player (two for three players)
	Win2P23 bool // three players on two windows: the second shows players 2 and 3
}

func defaultSettings() Settings {
	return Settings{Keys: defaultKeys, MusicOn: true, Windows: 1}
}

func loadSettings() Settings {
	s := defaultSettings()
	f, err := os.Open(fileDir + settingsFile)
	if err != nil {
		return s
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line, _, _ := strings.Cut(sc.Text(), "#")
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		k, v = strings.TrimSpace(k), strings.TrimSpace(v)
		switch k {
		case "trees":
			s.Smooth = v == "smooth"
		case "music":
			s.MusicOn = v != "off"
		case "windows":
			if v == "2" {
				s.Windows = 2
			}
		case "window2":
			s.Win2P23 = v == "players23"
		default:
			var p int
			var act string
			if n, _ := fmt.Sscanf(k, "p%d.%s", &p, &act); n != 2 || p < 1 || p > 3 {
				continue
			}
			for a, name := range actionKeys {
				if name == act {
					if vk, err := strconv.ParseUint(v, 0, 8); err == nil && vk > 0 {
						s.Keys[p-1][a] = uint8(vk)
					}
				}
			}
		}
	}
	return s
}

var actionKeys = [NActions]string{"up", "down", "left", "right", "reaper", "use", "boost", "horn"}

func (s *Settings) save() error {
	var b strings.Builder
	b.WriteString("# Combine-Baler settings. Keys are Windows virtual key codes.\n")
	b.WriteString("# Edit them in the game: title menu or pause menu, SETTINGS.\n\n")
	trees := "bouncy"
	if s.Smooth {
		trees = "smooth"
	}
	music := "on"
	if !s.MusicOn {
		music = "off"
	}
	w2 := "player3"
	if s.Win2P23 {
		w2 = "players23"
	}
	fmt.Fprintf(&b, "trees = %s\nmusic = %s\nwindows = %d\nwindow2 = %s\n", trees, music, s.Windows, w2)
	for p := range s.Keys {
		b.WriteString("\n")
		for a, vk := range s.Keys[p] {
			fmt.Fprintf(&b, "p%d.%s = 0x%02X  # %s\n", p+1, actionKeys[a], vk, keyName(vk))
		}
	}
	return os.WriteFile(fileDir+settingsFile, []byte(b.String()), 0644)
}

// keyName is how a virtual key is written on screen.
func keyName(vk uint8) string {
	switch {
	case vk >= '0' && vk <= '9', vk >= 'A' && vk <= 'Z':
		return string(rune(vk))
	case vk >= 0x60 && vk <= 0x69:
		return fmt.Sprintf("NUM%d", vk-0x60)
	case vk >= 0x70 && vk <= 0x87:
		return fmt.Sprintf("F%d", vk-0x6F)
	}
	switch vk {
	case KSpace:
		return "SPACE"
	case KEnter:
		return "ENTER"
	case KLeft:
		return "LEFT"
	case KRight:
		return "RIGHT"
	case KUp:
		return "UP"
	case KDown:
		return "DOWN"
	case 0x08:
		return "BACKSPACE"
	case 0x09:
		return "TAB"
	case 0x10:
		return "SHIFT"
	case 0x11:
		return "CTRL"
	case 0x12:
		return "ALT"
	case 0x14:
		return "CAPS"
	case 0x21:
		return "PGUP"
	case 0x22:
		return "PGDN"
	case 0x23:
		return "END"
	case 0x24:
		return "HOME"
	case 0x2D:
		return "INS"
	case 0x2E:
		return "DEL"
	case 0x6A:
		return "NUM*"
	case 0x6B:
		return "NUM+"
	case 0x6D:
		return "NUM-"
	case 0x6E:
		return "NUM."
	case 0x6F:
		return "NUM/"
	case 0xBA:
		return ";"
	case 0xBB:
		return "="
	case 0xBC:
		return ","
	case 0xBD:
		return "-"
	case 0xBE:
		return "."
	case 0xBF:
		return "/"
	case 0xDB:
		return "["
	case 0xDD:
		return "]"
	case 0xDE:
		return "'"
	case 0xAD:
		return "MUTE"
	case 0xAE:
		return "VOL-"
	case 0xAF:
		return "VOL+"
	}
	return fmt.Sprintf("KEY %02X", vk)
}

// keyOf names player i's key for an action. With one player both the
// first and second sets work; the first is shown.
func (g *Game) keyOf(i, a int) string {
	if g.NPlayers == 1 {
		i = 0
	}
	return keyName(g.Set.Keys[i][a])
}

// ---------------------------------------------------------------- screen

// settings pages: 0 main, 1..3 keys for that player
const (
	setTrees = iota
	setMusic
	setWindows
	setWin2
	setKeys1
	setKeys2
	setKeys3
	setReset
	setBack
	setRows
)

func (g *Game) openSettings() {
	g.setFrom, g.State = g.State, StSettings
	g.setPage, g.setSel, g.waitKey = 0, 0, false
	g.Snd.Play(SSelect)
}

func (g *Game) settingsInput(hit func(int) bool, keys *[256]bool) {
	if g.waitKey {
		for k := 1; k < 256; k++ {
			if !keys[k] || g.prev[k] {
				continue
			}
			g.waitKey = false
			if k == KEsc {
				return
			}
			g.bind(g.setPage-1, g.setSel, uint8(k))
			g.Snd.Play(SSelect)
			return
		}
		return
	}
	rows := setRows
	if g.setPage > 0 {
		rows = NActions + 1
	}
	switch {
	case hit(KUp) || hit(KW):
		g.setSel = (g.setSel + rows - 1) % rows
		g.Snd.Play(SMenu)
	case hit(KDown) || hit(KS):
		g.setSel = (g.setSel + 1) % rows
		g.Snd.Play(SMenu)
	case hit(KEsc):
		g.settingsBack()
	case hit(KEnter) || hit(KSpace) || hit(KLeft) || hit(KRight) || hit(KA) || hit(KD):
		if g.setPage > 0 {
			if g.setSel == NActions {
				g.settingsBack()
			} else {
				g.waitKey = true
				g.Snd.Play(SMenu)
			}
			return
		}
		switch g.setSel {
		case setTrees:
			g.Set.Smooth = !g.Set.Smooth
		case setMusic:
			g.Snd.ToggleMusic()
			g.Set.MusicOn = g.Snd.MusicOn
		case setWindows:
			g.Set.Windows = 3 - g.Set.Windows
		case setWin2:
			g.Set.Win2P23 = !g.Set.Win2P23
		case setKeys1, setKeys2, setKeys3:
			g.setPage, g.setSel = g.setSel-setKeys1+1, 0
		case setReset:
			g.Set.Keys = defaultKeys
			g.say("KEYS RESET")
		case setBack:
			g.settingsBack()
			return
		}
		g.Snd.Play(SSelect)
		g.Set.save()
	}
}

func (g *Game) settingsBack() {
	g.Snd.Play(SMenu)
	if g.setPage > 0 {
		g.setSel, g.setPage = g.setPage-1+setKeys1, 0
		return
	}
	g.Set.save()
	g.State = g.setFrom
	for i := range g.pl { // window layout may have changed
		g.pl[i].zoom = g.zoomTarget(i)
		g.updateCamera(i)
	}
}

// bind gives player p's action a the key vk. If some binding already uses
// vk, it takes this action's old key instead, so nothing is left unbound.
func (g *Game) bind(p, a int, vk uint8) {
	old := g.Set.Keys[p][a]
	for q := range g.Set.Keys {
		for b, k := range g.Set.Keys[q] {
			if k == vk && (q != p || b != a) {
				g.Set.Keys[q][b] = old
			}
		}
	}
	g.Set.Keys[p][a] = vk
	g.Set.save()
}

func (g *Game) settingsScreen() {
	g.rectA(0, 0, SW, SH, 0x2A1808, 175)
	row := func(y int, sel bool, label, value string) {
		c, v := uint32(0xD8C8A8), uint32(0xFFF4D6)
		if sel {
			g.rect(110, y-4, 530, y+11, 0x7A3B1C)
			c, v = 0xFFE06A, 0xFFE06A
		}
		g.textO(122, y, label, 1, c)
		g.textO(518-textW(value, 1), y, value, 1, v)
	}
	if g.setPage == 0 {
		g.textC(36, "SETTINGS", 3, 0xFFE06A)
		trees := "BOUNCY"
		if g.Set.Smooth {
			trees = "SMOOTH"
		}
		music := "ON"
		if !g.Set.MusicOn {
			music = "OFF"
		}
		wins := "ONE"
		if g.Set.Windows == 2 {
			wins = "ONE PER PLAYER"
		}
		w2 := "PLAYER 3"
		if g.Set.Win2P23 {
			w2 = "PLAYERS 2 AND 3"
		}
		vals := [setRows][2]string{
			{"TREES AND OBJECTS", trees}, {"MUSIC", music}, {"WINDOWS", wins},
			{"3 PLAYERS: SECOND WINDOW SHOWS", w2},
			{"PLAYER 1 KEYS", ">"}, {"PLAYER 2 KEYS", ">"}, {"PLAYER 3 KEYS", ">"},
			{"RESET ALL KEYS", ""}, {"BACK", ""},
		}
		for k, r := range vals {
			row(80+k*20, k == g.setSel, r[0], r[1])
		}
		help := [setRows]string{
			"BOUNCY: TREES ARE SOFT AND SPRINGY.  SMOOTH: SLIDE PAST, DRIVE UNDER THE LEAVES",
			"F2 ALSO SWITCHES THE MUSIC",
			"ONE PER PLAYER: DRAG A WINDOW TO YOUR OTHER MONITOR",
			"WITH THREE PLAYERS ON TWO WINDOWS",
			"", "", "",
			"BACK TO WASD, ARROWS AND THE NUMPAD", "",
		}
		g.textC(274, help[g.setSel], 1, 0xFFF4D6)
		g.textC(300, "UP / DOWN CHOOSE   ENTER CHANGE   ESC BACK", 1, 0xFFE06A)
		return
	}
	p := g.setPage - 1
	g.textC(36, fmt.Sprintf("PLAYER %d KEYS", p+1), 3, 0xFFE06A)
	for a := 0; a < NActions; a++ {
		v := keyName(g.Set.Keys[p][a])
		if g.waitKey && a == g.setSel {
			v = "PRESS A KEY"
			if int(g.clock*3)%2 == 0 {
				v = ""
			}
		}
		row(80+a*20, a == g.setSel, actionNames[a], v)
	}
	row(80+NActions*20, g.setSel == NActions, "BACK", "")
	if p == 2 {
		g.textC(274, "THE NUMPAD NEEDS NUM LOCK ON", 1, 0xFFF4D6)
	}
	g.textC(288, "ANY KEY WORKS, ALSO ON A SECOND KEYBOARD OR MACRO PAD", 1, 0xFFF4D6)
	g.textC(300, "ENTER THEN PRESS THE NEW KEY   ESC BACK", 1, 0xFFE06A)
}
