# IMGE Platformer Demo

A small pixel-art platformer that exercises as much of IMGE's built-in component
library as possible, plus a handful of custom components. It doubles as a
playground for the engine's Faz 1 component API (export variables, events,
`Requires()` dependency declarations, and the `@Builtin` / `components/*.go`
kind system).

## Controls

| Input            | Action              |
|------------------|---------------------|
| `A` / `←`        | Move left           |
| `D` / `→`        | Move right          |
| `Space` / `W` / `↑` | Jump             |
| `E` / `X`        | Attack (break crates) |

## Goal

Collect **5 gold coins** (2 lying in the open, 3 hidden inside crates), then
touch the **door** in the top-right to win. The door stays locked until every
coin is collected — approach it early and a yellow `!` warning pops up. Spikes
hurt you; if your hearts run out you respawn at the start.

- **Crates** can be pushed around (walk into them) *or* broken (press `E` next
  to one). Breaking one pops out a coin.
- **Coins** bob up and down on a sine wave and spin until you grab them.

## Project layout

```
game.json               window / FPS / initial scene
assets/                 PNG sprite sheets + WAV sounds
scenes/main.scene       the whole level (inline objects + file refs)
objects/crate.obj       reusable "crate" template
objects/gold.obj        reusable "gold" template
components/*.go         custom components (merged with the built-ins at build time)
```

## How the level maps to the component system

### Built-in components

| Component        | Used by                              | What it demonstrates |
|------------------|--------------------------------------|----------------------|
| `@Collider`      | everything physical                  | `solid` / `pushable` / `trigger` modes, `collidesWith` tag filtering, `collision_enter` events |
| `@Mover`         | player, crates, walker, bat, ball    | axis-by-axis collision resolution, push chains |
| `@Velocity`      | player, crates, walker               | per-axis speed state |
| `@Gravity`       | player, crates, walker               | acceleration + max-speed clamp |
| `@Friction`      | player                               | per-axis damping (slippery ground) |
| `@Sprite`        | everything visible                   | tinting, flip, sub-region drawing |
| `@Animator`      | player, door                         | multi-clip sprite-sheet animation (idle / run / jump, closed / open) |
| `@Spin`          | coins, enemies, poof particles       | rotation over time |
| `@Health`        | player, crates                       | HP pool + `died` / `damaged` events |
| `@Damage`        | spikes                               | overlap damage on a cooldown |
| `@Chase`         | bat                                  | home in on the nearest `player`, stop at range |
| `@Patrol`        | walker                               | loop between waypoints |
| `@Wander`        | firefly                              | random-direction drift |
| `@Bounce`        | ball                                 | reflect velocity off walls |
| `@Follow`        | pet                                  | lerp-trail the player with an offset |
| `@TimedDespawn`  | poof particle                        | lifetime auto-destroy |
| `@Sound`         | door                                 | one-shot sound effect |

### Custom components (`components/`)

| File        | Type             | What it does |
|-------------|------------------|--------------|
| `player.go` | `PlayerComponent`| reads input, writes `@Velocity`, picks anim clips, attacks crates, respawns on `died` |
| `gold.go`   | `GoldComponent`  | sine-wave bob, collects on `collision_enter` with the player |
| `crate.go`  | `CrateComponent` | breaks on `died`, spawns a coin + a poof particle |
| `door.go`   | `DoorComponent`  | unlocks on `all_gold`, emits `win` on touch, draws the `!` warning |
| `game.go`   | `GameComponent`  | HUD (coins + hearts), tracks progress, shows the win overlay |

### Event flow

```
player attacks crate   → Health.damage → "died"        → CrateComponent breaks
                       → spawns gold    → "collision_enter" (player) → GoldComponent
                       → "gold_collected" → GameComponent counts up
                       → "all_gold" (at 5)  → DoorComponent unlocks
                       → player touches   → "win"         → GameComponent overlay
```

## Build & run

```sh
imge run          # build and launch natively
imge build        # native binary  → imge_build/<name>_<os>-<arch>
imge build --web  # WASM bundle    → imge_build/web/
```

For the web build:

```sh
cd imge_build/web && python3 -m http.server 8000
# open http://localhost:8000/
```
