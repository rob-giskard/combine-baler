# Combine-Baler

A cosy harvest prototype for Windows. Drive a combine through the wheat and
draw in it, haul the bales to the barn with the truck, sell them at sundown,
spend the coins at the vending machine, sleep in the barn. One to three
players (split screen, or one window per player across two monitors).

## Controls

Default keys; every one can be changed in SETTINGS (title or pause menu).

|          | Drive / walk   | Reaper, hose spray | Use (get in/out, sleep, shop, hose) | Truck boost | Horn  |
|----------|----------------|--------------------|-------------------------------------|-------------|-------|
| Player 1 | WASD           | Space              | E                                   | Q           | R     |
| Player 2 | Arrows         | L                  | K                                   | O           | I     |
| Player 3 | Numpad 8 4 5 6 | Numpad 7           | Numpad 9                            | Numpad 1    | Numpad 3 |

With one player, both the player 1 and player 2 keys work. The numpad needs
Num Lock on. Any key can be bound, including keys from a second keyboard or
a macro pad (set the pad's own software to send F13 to F24, then bind those).

M whole-farm view, P save a picture of the farm (with stats), F2 music,
Esc pause. In the vending machine: left/right choose, reaper or use key
buys, up/down closes.

## Things to know

- **Trees** stand on the grass lanes with their crowns over the wheat. With
  the default BOUNCY setting the crowns are soft springs and the trunks
  bounce you off; SMOOTH lets you slide past and drive under the leaves.
  The barn and other machines always slide.
- **Truck boost:** a dash, a puff of smoke and a speed-up that fades over
  3.5 seconds; ready again 4 seconds after the last one. The dash presses the wheat flat in two tyre tracks (it grows
  back at replanting).
- **The hose** hangs on the tap at the barn's left wall, just for fun:
  take it (use key), aim, hold the reaper key to spray. Soaked machines
  sparkle and the bouncy ball gets pushed along. Walk too far and the hose
  springs back.
- **Vending machine:** coffee, fizzy water, egg sandwich (tiredness halved and
  slower for the day), cinnamon bun (+10% speed for the day), bouncy ball,
  kite, headlamps, three levels of faster engines, a bigger truck bed, a
  wide reaper. Coins and upgrades are shared.
- **Settings** are saved in `combine_baler_settings.txt` next to the exe.

## Build

Needs only [Go](https://go.dev/dl/) 1.24 or newer; there are no dependencies.

    build.bat          Windows: vet, test, build build\CombineBaler.exe
    ./build.sh         Linux/macOS: the same, cross-compiled
    build.bat notest   skip the tests

Every push to `prototype` is also built by GitHub Actions; the exe is
attached to the run as the `CombineBaler` artifact.

## Layout

| File              | What it holds                                                    |
|-------------------|------------------------------------------------------------------|
| `game.go`         | Game state, update loop, driving, farmers, world rendering       |
| `world.go`        | Field grid, wheat/stubble colouring, barn and yard art, bales    |
| `trees.go`        | Tree planting, canopy, soft bouncy crowns                        |
| `collide.go`      | Contact normals, sliding, bouncing, unsticking                   |
| `machines.go`     | Boost and tyre tracks, hose, horns, kite                         |
| `cosy.go`         | Day and night, tiredness, sundown sale, sleep, shop, balls, light |
| `views.go`        | Screen layouts for 1-3 players and one or two windows            |
| `settings.go`     | Key bindings, settings file, settings screen                     |
| `hud.go`          | Icon HUD, vending panel, morning report, picture export          |
| `audio.go`        | Software synth: music, engines, effects                          |
| `font.go`         | 5x7 bitmap font                                                  |
| `main_windows.go` | Win32 windows, keyboard polling, blit, winmm audio output        |
| `game_test.go`    | Scripted play-throughs, random-input runs, audio level check     |
| `features_test.go`| Trees, collisions, boost, hose, shop, horns, 3 players, settings |

The core renders into a 640x360 software framebuffer and is platform
independent, so the tests run headless anywhere (`go test .` also dumps
frames into `out/`). Only `main_windows.go` touches Windows.
