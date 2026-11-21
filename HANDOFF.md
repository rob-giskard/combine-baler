# Combine-Baler: handoff

First written 2026-11-14 (commit `282d3c7`); updated 2026-11-21 on the
`gameplay-improvements` branch, which carries out `PLAN.md`.
Read this first in the next sess## Where things stand

- Branch `prototype` holds the first playable version; PR #1 brings it to
  `master`. Branch `gameplay-improvements` adds everything in `PLAN.md`:
  settings and rebindable keys, smooth/bouncy collisions, lane trees, truck
  boost with tyre tracks, mud and the hose, seven new shop items, horns,
  three players, one window per player.
- Playable loop: reap and draw, haul bales to the pad in front of the barn,
  bales sell at 20:00 for 1 coin each, work at night and get tired, sleep at
  the barn door, spend coins at the vending machine. One to three players.

### What has and has not been verified

| Area | State |
|---|---|
| Game rules and rendering | Scripted sessions for 1, 2 and 3 players, one test per new feature, plus 200,000 frames of random input per mode (`go test .`). Frames were checked by eye. |
| Windows layer (`main_windows.go`) | Compiles and vets. Never launched: the build machine has no Windows or Wine. Keyboard is now polled with `GetAsyncKeyState`; the second window opens on the other monitor if there is one. |
| Sound output (winmm) | Never played. New sounds (horns, boost, spray) only checked as code. |
| Game feel, balance | Untested by a human: boost strength, mud rates, crown springiness and shop prices are all first guesses. |

The first thing the next session needs is the designer's report from
actually running the exe: does it start, does sound play, how does it feel,
and do both windows take keys at once.

## Known bugs

1. ~~A ball can be shoved inside a wall.~~ Fixed: the nudge goes through
   `solidAt`.
2. **Removing a bale erases ground art under it.** `bakeBale(b, false)` in
   `world.go` sets the object layer to empty for the bale's cells. Ground
   shadows and the pad's corner posts live in that same layer. Fix: keep
   bales in their own layer, or restore what was there.
3. ~~Saved pictures go to the working directory.~~ Fixed: pictures, the
   settings file and the crash report go next to the exe (`fileDir`).

Smaller loose ends: more than 72 bales on the pad are counted and sold but
not drawn; `StoredToday` in `game.go` is written and never read.

## Tuning knobs added on this branch

Top of `machines.go`: hose length, boost time, dash time and cooldown,
engine step, big bed capacity, mud rates (`MudCombine`, `MudTruck`), wash
time, spray range. Shop prices: `items` and `enginePrice` in `cosy.go`.
Crown springiness: `treeSprings` in `trees.go` (the `34` and radius terms).
Tree layout: `plantTrees` in `trees.go`.

d again (standard Win32 menu behaviour, not
checked).

## Three possible improvements

1. **Save the farm when the farmer sleeps.** Coins, headlamps, the day
   number and the cut field are lost when the game closes. The field is one
   byte per cell, so a compressed save file is small.
2. **Give drawing a purpose.** Tonnage is still earned fastest by straight
   stripes. Candidates: a daily commission (an outline to match in a named
   field), or a photo album that keeps each day's picture.
3. **Gamepad support.** Two players on one keyboard is cramped. XInput can
   be called the same dependency-free way as the rest of the Windows layer.

Also open from the last message to the designer, still unanswered: coin
balance (a bale drops about every two seconds of reaping), whether both
players must be at the barn to sleep, whether replanting should cost seed
from the machine, and whether the ball should do more than bounce.

## Three things worth knowing as the developer

1. **The game can be checked without Windows.** Everything except
   `main_windows.go` is platform independent and draws into a plain
   640x360 pixel buffer. `go test .` plays scripted sessions and writes
   numbered screenshots into `out/`. Looking through that folder after a
   change is the fastest way to see what it did. A new feature should get a
   few lines in `game_test.go` that play it.
2. **Most tuning is a number at the top of a file.** Day length, cells per
   bale, truck capacity and the farm grid are constants at the top of
   `game.go`. Bale price, night length and the shop's items and prices are
   at the top of `cosy.go`. Camera zoom levels are in `zoomTarget`. These
   can be changed without touching any logic.
3. **Art is text.** Icons and the farmer are small grids of letters with a
   colour per letter (`palette` and the `art...` tables in `hud.go`,
   `farmerArt` in `game.go`). The barn and yard are rectangles and discs in
   `bakeFarm` in `world.go`, in farm-local coordinates; the last argument
   says whether the thing blocks movement. Editing those tables is how to
   repaint anything.

## How to report a problem

- If the game shows an error box, send `combine_baler_crash.txt` from the
  game's folder.
- For anything visual, press P and send the saved picture, or a screenshot.
- For sound, say whether effects play with the music off (F2).
