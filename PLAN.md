# Gameplay improvements: plan

**Status 2026-11-21: all eight milestones built, tested headless; waiting for
the designer to play the exe.** Later the same day the designer dropped the
mud mechanic (milestone 4): the hose stays as a fun activity only. See `HANDOFF.md` for what is verified.

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
- **Scope of the setting (designer's choice):** the Smooth/Bouncy toggle
  applies only to trees and other map objects. The barn and other machines
  always use the smooth slide below, never the bounce.
- **Smooth:** remove only the part of the motion going into the
  wall, keep the sideways part, so the machine slides along trees, the barn
  and other machines, losing about 15% speed. A small push-out frees a
  machine that starts a frame overlapping something, so nothing can wedge.
- **Bouncy (default for trees and map objects):** bounce off with about 90% of the speed plus an
  extra kick, a short spin, a dust puff and a boing. Vehicles get a knockback
  velocity (`kx, ky`, decays over about half a second) separate from their
  driving speed. A farmer on foot who walks into a tree bounces off it with
  a little hop and wobble. Never damage, never a fail state.
- **More autumn trees, on grass only:** four leaf palettes (orange, red,
  gold, russet) and three sizes, picked per tree by `hash`. About a dozen
  more around the yard and along the grass lanes between fields. No trees
  inside fields.
- **Soft, bouncy crowns (designer's choice):** lane trees are big enough
  that their crowns hang over the edge of the wheat. The trunk is hard; the
  crown is a soft spring bumper that a machine can sink into a little
  before it is pushed back out with a wobble. That makes trimming the wheat
  under an overhanging crown a small precision challenge. Crowns go
  see-through over a machine so the driver can see what they are cutting.

### 3. Truck boost

- Boost key while driving a **truck**: a quick dash (instant speed to about
  1.6x top speed), a big puff of grey-white smoke from the exhaust, then a
  lingering speed-up that fades over about 2.5 s. Cooldown about 4 s.
- Ready indicator: a small arrow icon next to the truck in the HUD that fills
  back in during the cooldown.
- Sound: a short whoosh, plus a deeper engine note while boosted.
- Boost does not work while clogged with mud (milestone 4), so cleaning
  matters.
- **Tyre marks:** boosting across wheat flattens two tyre stripes for the
  length of the dash. Flattened wheat is a new cell state: it can't be
  reaped into bales, looks pressed down and darker, and springs back at
  replanting. (It cuts into drawings, by design.)
- **Camera:** the view zooms out while boosting and eases back in as the
  speed-up fades.

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
  where the crate stood.
  - On foot, Use at the tap picks up the hose ("HOSE" key cap). The hose is
    drawn as a line from the tap to the farmer and has a fixed length.
  - The farmer walks around holding it and aims where he faces; the reaper
    key sprays. The hose reaches about four truck lengths; walking further
    drops it and it snaps back to the tap. Water particles arc out, and a machine they land on loses
    mud (about 3 s for a full clean). Cleaned machines sparkle for a moment.
  - Use again (or walking back to the tap) hangs the hose up.

### 5. Shop: upgrades, food and a fun item

- **Engine upgrades:** several levels (planned: 3), each a permanent
  top-speed boost for every machine at a rising price. The icon shows the
  level reached.
- **Bigger truck bed:** one level, more bales per trip.
- **Wider reaper:** one level, the combine cuts a wider swath.
- **Egg sandwich:** about 3 coins. Removes half the current tiredness, and for
  the rest of the day tiredness builds at half speed ("full belly" icon in
  the HUD).
- **Cinnamon bun:** +10% speed for the rest of the day.
- **Kite:** the fun item. Once bought, the farmer flies a kite on a string
  that trails behind him and drifts with the wind, also while driving.
- With 7+ items the vending panel scrolls: it shows a window of items and
  slides along as you move the selection, so it fits a narrow view.

### 6. Car horns

- Horn key in a combine or a truck. Combine: low two-tone honk. Truck:
  higher double beep. About 0.4 s between honks so it cannot be machine-gunned.
- A small ring pops out of the vehicle when it honks, so the other players
  see it in their views too.
- A farmer on foot close to a honking machine does a startled little hop.

### 7. Three-player mode

- `setup(3)`: three combines and three trucks; player 3's machines get a third
  tint (blue). New parking spots in the yard, checked against the new trees.
- Two players keep today's side-by-side split.
- **Screen layout for three (designer's choice):** two views side by side on top,
  one wide view below. Views become rectangles (x, y, w, h) instead of
  full-height strips, so `cam`, `view`, `updateCamera`, `renderView`,
  lighting and the HUD all take a view height too.
- Generalise every `NPlayers == 2` check to "split when more than one
  player".
- HUD per view: bale icons, mud gauge, boost icon, prompts and shop panel
  laid out for the half-height views.

### 8. One window per player (spread across monitors)

- Settings option **Views: one window / a window per player.**
- In window-per-player mode each player's view gets its own Win32 window at
  the full 640x360 resolution (scaled up), which can be dragged to another
  monitor and maximised. The game core already renders each view
  separately, so this is mostly platform work in `main_windows.go`.
- Keyboard input must not depend on which window has focus, otherwise only
  one player could drive at a time. Keys are read with `GetAsyncKeyState`
  every frame instead of from window messages.
- Shared screens (title, pause, morning report) show in the first window.
- Windows are normal resizable windows (the designer has two monitors), not
  borderless fullscreen. With three players, the first window keeps two
  views and the second window gets one, or the reverse; set in settings.

### Extra input devices (mini keyboard with knob)

Nothing special to build: any key a device sends can be bound in the
settings screen by pressing it. Small macro pads usually come with their
own software that sets what each key and the knob send. Best practice is
to set them to keys nobody types on, such as F13 to F24, then bind those in
the game. If the device sends media keys (volume up/down), those also
change Windows' volume, so remapping is better.
- Tests: `TestThreePlayers` (scripted) and a 3-player fuzz run.

## Decisions made (2026-11-21)

1. Boost: trucks only; boosting on a field leaves muddy tyre marks.
2. Hose: the farmer gets out, picks up the hose with the use key, holds it
   and aims at the machine.
3. Engine upgrades: several levels, plus one more fun item like the ball.
4. Egg sandwich as planned; add a cinnamon bun (+10% speed for the day).
5. 3-player: two views on top, one wide view below; also an option for a
   window per player to spread across monitors.
6. Player 3 on the numpad; the designer's mini keyboard (3 keys and a knob)
   is bound through the settings screen.
7. Trees on grass only, with soft bouncy crowns overhanging the wheat.

## Decisions made, round 2

1. Fun item: a kite.
2. Upgrades: engine levels, plus one level each of truck capacity and reaper
   width.
3. Boost tyre marks flatten wheat; the camera zooms out while boosting.
4. Hose: sprays with the reaper key where the farmer faces; reach about four
   truck lengths; drops back to the tap when stretched too far.
5. Coins and upgrades stay shared.
6. Sleeping: anyone at the barn door still ends the day for everyone.
7. Smooth/Bouncy applies only to trees and map objects, never the barn or
   other machines; bouncy is the default.
8. Window per player: normal resizable windows; two monitors.
9. Two players keep the side-by-side split.

## Untouched here

Known bugs 1 and 2 from `HANDOFF.md` (ball pushed into a wall, bale pickup
erasing ground art) are not in scope, but bug 1 falls out of milestone 2 if
balls go through the same solid check; bug 3 is fixed in milestone 1.
