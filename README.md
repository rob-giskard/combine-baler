# Combine-Baler

A cosy harvest prototype for Windows. Drive a combine through the wheat and
draw in it, haul the bales to the barn with the truck, sell them at sundown,
spend the coins at the vending machine, sleep in the barn. One or two players
(split screen).

## Controls

|                 | Drive / walk   | Reaper | Use (get in/out, sleep, shop) |
|-----------------|----------------|--------|-------------------------------|
| One player      | WASD or arrows | Space  | E                             |
| Player 1 of two | WASD           | Space  | E                             |
| Player 2 of two | Arrows         | L      | K                             |

M whole-farm view, P save a picture of the farm (with stats), F2 music,
Esc pause. In the vending machine: left/right choose, reaper or use key
buys, up/down closes.

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
| `game.go`         | Game state, update loop, machines, farmers, world rendering      |
| `world.go`        | Field grid, wheat/stubble colouring, barn and yard art, bales    |
| `cosy.go`         | Day and night, tiredness, sundown sale, sleep, shop, balls, light |
| `hud.go`          | Icon HUD, vending panel, morning report, picture export          |
| `audio.go`        | Software synth: music, engines, effects                          |
| `font.go`         | 5x7 bitmap font                                                  |
| `main_windows.go` | Win32 window, keyboard, blit, winmm audio output                 |
| `game_test.go`    | Scripted play-throughs, random-input runs, audio level check     |

The core renders into a 640x360 software framebuffer and is platform
independent, so the tests run headless anywhere (`go test .` also dumps
frames into `out/`). Only `main_windows.go` touches Windows.
