# Gameplay improvements: plan

Branch `gameplay-improvements`, cut from `prototype` at `588a1f0`.
Written 2026-11-21. Each milestone is one commit that keeps `go test .` green
and adds a few lines to `game_test.go` that play the feature, with frames in
`out/` to check by eye.

## What the code does today (why some of this is needed)

- **Getting stuck on trees** has two causes. Every tree crown is baked as a
  *solid* 13-cell disc (`bakeFarm`, `oDisc(..., true)`), so the wall is much
  wider than the trunk. And when a vehicle is blocked, `drive` reverses its
  speed at 25% and does not slide, so a glancing touch stops it dead and it
  jitters. Walking already slides (x and y are tested separately).
- **Keys are hard-coded** in `Game.input` and in the HUD strings (`useKey`,
  `hint`, title screen). There is no settings screen and nothing is saved.
- **Two players is hard-coded** in `split`, `view`, `updateCamera`,
  `zoomTarget`, `setup`, `useKey` and the title menu.
- There are **no barrels** in the yard; the yard clutter is one crate and two
  terracotta pots. Both go.

## Milestone order

Settings come first because boost, horn and player 3 all need their own keys.

### 1. Settings screen and rebindable keys

- Title menu becomes `1 PLAYER / 2 PLAYERS / 3 PLAYERS / SETTINGS`.
- Settings screen (also reachable from the pause menu):
  - **Collisions:** `SMOOTH` / `BOUNCY` (see milestone 2).
  - **Music:** on / off (same as F2).
  - **Keys, per player:** up, down, left, right, reaper, use, boost, horn.
    Select a row, press the new key. A key already used by another action is
    swapped, so nothing ends up unbound.
- `Keymap` struct per player replaces the constants in `Game.input`; HUD
  strings read key names from it (`keyName(vk)`).
- Saved as `combine_baler_settings.txt` (plain `key = value` lines) **next to
  the exe**. The same `exeDir()` helper fixes known bug 3 (saved pictures
  landing in the working directory).
- Default keys:

  |          | Move    | Reaper | Use | Boost | Horn |
  |----------|---------|--------|-----|-------|------|
  | Player 1 | WASD    | Space  | E   | Q     | R    |
  | Player 2 | Arrows  | L      | K   | O     | I    |
  | Player 3 | Numpad 8 4 5 6 | Numpad 7 | Numpad 9 | Numpad 1 | Numpad 3 |

  In one-player mode both P1 and P2 bindings work, as now.

### 2. Collisions and trees

- **Tree crowns stop being solid.** Only the trunk (radius about 3) blocks.
  Crowns move to a new canopy layer drawn *after* vehicles and farmers, and
  fade to half opacity over a vehicle or farmer underneath, so you can drive
  under a tree's edge and still see yourself.
- **Remove** the crate and the terracotta pots.
- **Contact normal:** when a probe hits, sample `solidAt` around the hit
  point to get the wall direction (vehicle-to-vehicle: the line between
  centres).
- **Smooth (default):** remove only the part of the motion going into the
  wall, keep the sideways part, so the machine slides along trees, the barn
  and other machines, losing about 15% speed. A small push-out frees a
  machine that starts a frame overlapping something, so nothing can wedge.
- **Bouncy (exaggerated):** bounce off with about 90% of the speed plus an
  extra kick, a short spin, a dust puff and a boing. Vehicles get a knockback
  velocity (`kx, ky`, decays over about half a second) separate from their
  driving speed. Machines push each other apart, parked ones included.
  Farmers on foot near a moving machine are pushed outward with a little hop
  and wobble. Never damage, never a fail state.
- **More autumn trees:** four leaf palettes (orange, red, gold, russet) and
  three sizes, picked per tree by `hash`. About a dozen more around the yard
  and along the grass lanes, plus one to three lone trees inside some
  fields. Wheat under a trunk becomes grass and leaves `WheatTotal`; the
  crown's shadow darkens the wheat around it.

### 3. Truck boost

- Boost key while driving a **truck**: a quick dash (instant speed to about
  1.6x top speed), a big puff of grey-white smoke from the exhaust, then a
  lingering speed-up that fades over about 2.5 s. Cooldown about 4 s.
- Ready indicator: a small arrow icon next to the truck in the HUD that fills
  back in during the cooldown.
- Sound: a short whoosh, plus a deeper engine note while boosted.
- Boost does not work while clogged with mud (milestone 4), so cleaning
  matters.

### 4. Mud and the hose station

- Each vehicle gets `mud` (0 to 1). It builds while working: a reaping combine
  fills in about 90 s of reaping; a truck fills in about 150 s of driving on
  wheat or stubble. Grass and the yard add nothing.
- **Effect:** none below 50%; from 50% to 100% top speed drops steadily to
  about 55%. At 100% the machine is "clogged" (puffs of dark smoke, a slower
  engine note).
- **Gauge:** a brown circle that fills clockwise like a clock face, in each
  player's HUD next to the bale icons, for the vehicle they are in. It also
  floats small above any machine past 75%. The vehicle sprite gets brown
  speckles that grow with the mud.
- **Hose station:** a tap and coiled green hose on the barn's left wall,
  where the crate stood. On foot, press Use at the tap (`UseHose`,
  "WASH" key cap). The farmer sprays the nearest machine within reach: water
  particles arc onto it and its mud drains over about 3 s. Cleaned machines
  sparkle for a moment.

### 5. Shop: two new items

- **Better engines:** permanent +20% top speed for every machine, one-time
  purchase, about 15 coins. Shows the green tick once bought, like the
  headlamps.
- **Egg sandwich:** about 3 coins. Removes half the current tiredness, and for
  the rest of the day tiredness builds at half speed ("full belly" icon in
  the HUD).
- The vending panel grows from 4 to 6 items (about 210 px wide, still fits a
  third of the screen in 3-player). Two new 14x14 icons in the `artItem`
  tables.

### 6. Car horns

- Horn key in a combine or a truck. Combine: low two-tone honk. Truck:
  higher double beep. About 0.4 s between honks so it cannot be machine-gunned.
- A small ring pops out of the vehicle when it honks, so the other players
  see it in their views too.
- A farmer on foot close to a honking machine does a startled little hop.

### 7. Three-player mode

- `setup(3)`: three combines and three trucks; player 3's machines get a third
  tint (blue). New parking spots in the yard, checked against the new trees.
- **Screen layout:** three vertical strips, 212 px wide each, full height.
  This keeps the camera code as it is (it already works in vertical strips);
  zoom levels get a third column in `zoomTarget`.
- Generalise every `NPlayers == 2` check to "split when more than one
  player"; `view(i)` returns strip `i` of `NPlayers`.
- HUD per strip: bale icons, mud gauge, boost icon, prompts and shop panel
  laid out for the narrower strip.
- Tests: `TestThreePlayers` (scripted) and a 3-player fuzz run.

## Open questions for the designer

1. **Boost on trucks only, or combines too?** The plan follows the brief
   (trucks only).
2. **Hose: farmer sprays on foot, or a drive-through wash?** The plan has the
   farmer spray it, which is cosier; drive-through is quicker in play.
3. **Better engines: one-time +20%, or three levels (+10% each, rising
   price)?**
4. **Egg sandwich effect:** is "half tiredness now, tiredness builds half as
   fast the rest of the day" right, or should it do something else?
5. **3-player layout:** three vertical strips (planned), or two on top and
   one wide view at the bottom?
6. **3-player keys:** the numpad (needs Num Lock on), or skip that and go
   straight to gamepad support for players 2 and 3?
7. **Lone trees in fields:** they make drawing harder (you steer around them).
   Fine, or keep them only on lanes?

## Untouched here

Known bugs 1 and 2 from `HANDOFF.md` (ball pushed into a wall, bale pickup
erasing ground art) are not in scope, but bug 1 falls out of milestone 2 if
balls go through the same solid check; bug 3 is fixed in milestone 1.
