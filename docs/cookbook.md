# Cookbook

Worked recipes showing how components combine. Each recipe is a scene snippet plus
(where needed) a custom component. Adjust names, tags, and numbers to your game.

> Cross-references: [Components](components.md) for the args tables,
> [JSON format](json-format.md) for the field reference, [Events](events.md) for
> the event contract.

## A platformer player

The movement stack: `@Velocity` holds speed, `@Gravity` adds acceleration,
`@Friction` damps horizontal drift, `@Mover` resolves collisions through the
`@Collider`. A custom component reads input and writes `@Velocity`.

```json
{
  "name": "player",
  "depth": 10,
  "tags": ["player"],
  "transform": { "position": { "x": 120, "y": 470 } },
  "components": [
    { "kind": "@Collider", "name": "body", "args": { "width": 28, "height": 32, "mode": "solid", "collides_with": ["platform", "wall"] } },
    { "kind": "@Mover", "name": "mover", "args": {} },
    { "kind": "@Velocity", "name": "velocity", "args": {} },
    { "kind": "@Gravity", "name": "gravity", "args": { "acceleration": { "x": 0, "y": 980 }, "max_speed": 700 } },
    { "kind": "@Friction", "name": "friction", "args": { "amount": 900, "axes": "x" } },
    { "kind": "Player", "name": "player", "args": { "speed": 190, "jump_speed": 560 } }
  ]
}
```

```go
// components/player.go
package components

import "github.com/EnesBaytekin/imge/core"

type Player struct {
    core.BaseComponent
    Speed     float64 `json:"speed"`
    JumpSpeed float64 `json:"jump_speed"`
}

func (c *Player) Initialize() {
    if c.Speed <= 0 { c.Speed = 190 }
    if c.JumpSpeed <= 0 { c.JumpSpeed = 560 }
}

func (c *Player) Update(ctx *core.Context) {
    v := core.GetFrom[*Velocity](c.GetOwner())
    if v == nil { return }
    vx, vy := v.Velocity()

    dir := 0.0
    if ctx.Input.IsKeyPressed(core.KeyLeft) || ctx.Input.IsKeyPressed(core.KeyA) { dir = -1 }
    if ctx.Input.IsKeyPressed(core.KeyRight) || ctx.Input.IsKeyPressed(core.KeyD) { dir = 1 }
    vx = dir * c.Speed

    if (ctx.Input.IsKeyJustPressed(core.KeySpace) || ctx.Input.IsKeyJustPressed(core.KeyUp)) {
        vy = -c.JumpSpeed
    }
    v.SetVelocity(vx, vy)
}
```

For a quick **overhead/arcade** character with no physics, swap the whole stack for
`@PlayerController` (WASD/arrows → `@Mover`) plus a `@Collider`.

## An enemy that chases

`@Chase` homes in on the nearest `player`; a trigger `@Collider` + `@Damage` hurts
them on overlap, on a cooldown. `@Sprite` + `@Spin` make it visible.

```json
{
  "name": "bat",
  "tags": ["enemy"],
  "components": [
    { "kind": "@Sprite", "name": "sprite", "args": { "texture": "assets/orb.png", "width": 20, "height": 20 } },
    { "kind": "@Spin", "name": "spin", "args": { "speed": 6 } },
    { "kind": "@Collider", "name": "hitbox", "args": { "width": 20, "height": 20, "mode": "trigger", "collides_with": ["player"] } },
    { "kind": "@Chase", "name": "chase", "args": { "speed": 90, "target_tag": "player", "stop_distance": 40 } },
    { "kind": "@Damage", "name": "damage", "args": { "amount": 1, "target_tags": ["player"], "cooldown": 1.0 } }
  ]
}
```

Variants: `@Patrol` for back-and-forth guards (give it `points`, and add
`@Velocity`/`@Gravity`/`@Collider` if they should fall), `@Wander` for ambient
drift, `@Bounce` for a reflecting projectile, `@Follow` for a companion.

## A collectible pickup

A trigger collider detects the player; a custom component handles the pickup. The
coin bobs (custom) and spins (`@Spin`).

```json
{
  "name": "coin",
  "tags": ["coin"],
  "components": [
    { "kind": "@Sprite", "name": "sprite", "args": { "texture": "assets/coin.png", "width": 20, "height": 20 } },
    { "kind": "@Spin", "name": "spin", "args": { "speed": 3 } },
    { "kind": "@Collider", "name": "hitbox", "args": { "width": 20, "height": 20, "mode": "trigger", "collides_with": ["player"] } },
    { "kind": "Coin", "name": "coin", "args": {} }
  ]
}
```

```go
// components/coin.go
package components

import "github.com/EnesBaytekin/imge/core"

type Coin struct { core.BaseComponent }

func (c *Coin) Initialize() {
    c.On("collision_enter", func(data any) {
        other, ok := data.(*core.Object)
        if ok && other.HasTag("player") {
            c.Emit("coin_collected", c.GetOwner())
            c.GetOwner().Destroy()
        }
    })
}
```

## Health, damage, and death

Give the player `@Health`; hazards use `@Damage`. The `died` event lets a custom
component react (respawn, game over).

```json
{ "kind": "@Health", "name": "health", "args": { "max": 3 } }
```

```json
{
  "name": "spikes",
  "tags": ["hazard"],
  "components": [
    { "kind": "@Collider", "name": "hitbox", "args": { "width": 24, "height": 8, "mode": "trigger", "collides_with": ["player"] } },
    { "kind": "@Damage", "name": "damage", "args": { "amount": 1, "target_tags": ["player"], "cooldown": 1 } }
  ]
}
```

```go
func (c *Game) Initialize() {
    c.On("died", func(data any) {
        obj, ok := data.(*core.Object)
        if !ok || !obj.HasTag("player") { return }
        // respawn logic…
    })
}
```

Note `@Health.Damage` emits `died` with the owner object as `data` — filter by tag
so other objects' deaths don't trigger the respawn.

## Scene transitions

A menu component switches scenes on input. `SwitchScene` is deferred, so it's safe
to call from `Update`.

```go
// components/menu.go
package components

import "github.com/EnesBaytekin/imge/core"

type Menu struct { core.BaseComponent }

func (c *Menu) Update(ctx *core.Context) {
    if ctx.Input.IsKeyJustPressed(core.KeyEnter) {
        ctx.Game.SwitchScene("game")
    }
}
```

In `game.imge` set `"initial_scene": "menu"`, and have `scenes/menu.scene` and
`scenes/game.scene` each with `"name"` matching. See
[Scenes & camera](scenes-and-camera.md#scene-switching).

## A HUD in screen space

UI objects (`"ui": true`) ignore the camera and draw in screen pixels — perfect for
hearts and counters. Draw with a custom component's `Draw(renderer)`.

```json
{
  "name": "hud",
  "ui": true,
  "depth": 100,
  "components": [ { "kind": "HUD", "name": "hud", "args": {} } ]
}
```

```go
// components/hud.go
package components

import "github.com/EnesBaytekin/imge/core"
import "github.com/EnesBaytekin/imge/core/math"

type HUD struct { core.BaseComponent; collected int }

func (c *HUD) Initialize() { c.On("coin_collected", func(any) { c.collected++ }) }

func (c *HUD) Draw(r core.Renderer) {
    r.DrawCircle(math.NewVector2(30, 30), 10, math.Red) // a "heart" at screen (30,30)
    r.DrawRect(math.NewRect(10, 50, float64(c.collected*20), 8), math.Yellow)
}
```

## A door that opens when all coins are gathered

Compose events: coins emit `coin_collected`; a counter emits `all_coins`; the door
listens and switches its `@Animator` clip.

```go
// components/game.go — counts and broadcasts progress
func (c *Game) Initialize() {
    c.On("coin_collected", func(any) {
        c.collected++
        if c.collected >= c.Total { c.Emit("all_coins", nil) }
    })
}
```

```go
// components/door.go — opens on all_coins, wins on touch
func (c *Door) Initialize() {
    c.On("all_coins", func(any) { core.GetFrom[*Animator](c.GetOwner()).Play("open") })
    c.On("collision_enter", func(data any) {
        if o, ok := data.(*core.Object); ok && o.HasTag("player") && c.open {
            c.Emit("win", nil)
        }
    })
}
```

The door object carries `@Sprite` (closed) + `@Sprite` (open) + `@Animator`
(`closed`/`open` clips) + a trigger `@Collider` + a `Door` component.

## State-machine driven behavior

Use `@StateMachine` when an object has distinct states with event-driven
transitions. The player below has `idle → run → jump → hurt` states; input and
events drive the transitions.

```json
{
  "kind": "@StateMachine",
  "name": "fsm",
  "args": {
    "initial": "idle",
    "states": [
      { "name": "idle", "transitions": [
        { "event": "move", "from": "component", "to": "run" },
        { "event": "jump", "from": "component", "to": "jump" }
      ]},
      { "name": "run", "transitions": [
        { "event": "idle", "from": "component", "to": "idle" },
        { "event": "jump", "from": "component", "to": "jump" }
      ]},
      { "name": "jump", "transitions": [
        { "event": "landed", "from": "component", "to": "idle" }
      ]},
      { "name": "hurt", "transitions": [
        { "event": "recover", "from": "", "to": "idle", "delay": 0.5 }
      ]}
    ]
  }
}
```

Drive it from a custom component:

```go
c.Emit("move", nil)                       // -> run (from "component" = this component)
fsm := core.GetFrom[*StateMachine](owner)
fsm.Trigger("recover")                    // manual transition (from ""), after a delay
state := fsm.Current()                    // read the state
if fsm.JustEntered() { /* one-shot on entry */ }
```

See [Components → `@StateMachine`](components.md#statemachine) for the full `from`
scope table and `JustEntered` timing.

## Next

- [Components](components.md) — every built-in, with args and methods.
- [Events](events.md) — the event model behind these recipes.
- [Custom components](custom-components.md) — the Go patterns used above.
