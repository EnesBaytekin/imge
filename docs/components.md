# Built-in Components

> ← [Documentation index](README.md) · [Custom components](custom-components.md) · [Events](events.md)

IMGE ships a small library of **built-in components**. They are not special — they
implement the same `core.Component` interface as user components, and at build time
they are merged into the project's single `components` package, so any component
(built-in or custom) can reach any other component's concrete methods directly.

Every component's configurable fields are its **export variables**: JSON-tagged
fields populated from the component's `args` object in `.obj`/`.scene` files. All
args are **snake_case**.

Each component below shows a **copy-paste `args` example** with default values —
paste the `{ "kind", "name", "args" }` entry into an object's `components` array and
adjust. Every arg has a default except a few marked **(required)**.

## The component model

- **Kind** — `@Name` for built-ins (e.g. `@Sprite`), the struct type name for user
  components (e.g. `PlayerComponent`).
- **Lifecycle** — `Initialize()` once after the scene is fully assembled, then
  `Update(ctx)` every frame, then `Draw(renderer)`. Draw order within an object is
  by `draw_layer` (a field every component inherits via `BaseComponent`; lower
  draws first, equal keeps insertion order).
- **Universal arg** — every component (built-in and custom) accepts one extra arg
  inherited from `BaseComponent`: `draw_layer` (int, default 0), which orders its
  `Draw` relative to the object's other components.
- **Accessing siblings** — `core.GetFrom[*T](owner)` (first by insertion order),
  `core.GetAllFrom[*T](owner)`, `core.GetFromNamed[*T](owner, name)`. All return a
  nil pointer when nothing matches; nil-check.
- **Requires** — optional `Requires() []string` declares the kinds a component
  needs (informational; the build tool warns about missing ones).
- **Events** — `Emit(name, data)` broadcasts to every component that subscribed via
  `On(name, handler)`, delivered after all `Update` calls finish. Events carry a
  `Source` (the emitting component); the `@StateMachine` filters transitions by it.

## Quick reference

| Kind | Purpose |
|------|---------|
| `@Sprite` | Draws a texture (or a frame of a sprite sheet) at the owner transform |
| `@Animator` | Plays named clips by driving a `@Sprite`'s frame, visibility and flip |
| `@Spin` | Rotates the owner over time |
| `@Mover` | Collision-aware movement (the capability other components drive) |
| `@Velocity` | Per-axis velocity state, integrated through `@Mover` each frame |
| `@Gravity` | Applies acceleration (default gravity) to `@Velocity` |
| `@Friction` | Damps `@Velocity` toward zero on an axis |
| `@Bounce` | Constant velocity that reflects off blocked collisions |
| `@PlayerController` | WASD / arrow keys → movement (arcade/overhead) |
| `@Chase` | Moves toward the nearest tagged object |
| `@Follow` | Smoothly lerps toward a tagged object + offset |
| `@Patrol` | Walks a list of waypoints (loop or ping-pong) |
| `@Wander` | Moves in a random direction, re-rolled on an interval |
| `@Collider` | Rectangle shape; solid/pushable/trigger, overlap tracking |
| `@Health` | HP pool; emits `damaged` / `died` |
| `@Damage` | Applies damage to overlapping tagged targets |
| `@StateMachine` | Named states + event-driven transitions |
| `@Sound` | One-shot sound effect or looping music |
| `@TimedDespawn` | Destroys the owner after N seconds |

## Rendering

### `@Sprite`
Draws `texture` at the owner's transform. Frame slicing is derived from the
texture size — no frame count is configured.

```json
{
  "kind": "@Sprite", "name": "sprite",
  "args": {
    "texture": "assets/player.png",
    "frame_width": 0, "frame_height": 0, "frame": 0,
    "width": 0, "height": 0,
    "flip_x": false, "flip_y": false,
    "tint": { "r": 255, "g": 255, "b": 255, "a": 255 },
    "offset": { "x": 0, "y": 0 },
    "visible": true
  }
}
```

- Args: `texture` (**required**), `frame_width` (0), `frame_height` (0), `frame`
  (0), `width` (0 = natural), `height` (0 = natural), `flip_x` (false), `flip_y`
  (false), `tint` (white), `offset {x,y}` (0,0), `visible` (true).
- `frame_width = 0` → whole texture is one frame. `frame_width > 0` and
  `frame_height = 0` → a single horizontal strip. Both `> 0` → a grid cut
  row-by-row. `FrameCount()` is always computed from the texture size.
- `width`/`height` are the display size (0 = natural frame size).
- Methods: `SetFrame`, `SetTexture`, `SetTint`, `SetFlipX/Y`, `SetVisible`,
  `SetOffset`, `IsVisible`, `FrameCount`.

### `@Animator`
Owns the animation state of the sprites named in its clips: it drives their frame,
makes exactly one visible at a time, and mirrors its own flip onto all of them.
Sprites not named in a clip are left untouched.

```json
{
  "kind": "@Animator", "name": "animator",
  "args": {
    "clips": [
      { "sprite": "sprite", "fps": 12, "loop": true }
    ],
    "default": "sprite", "flip_x": false, "flip_y": false
  }
}
```

- Args: `clips [{sprite, fps, loop}]`, `default`, `flip_x`, `flip_y`.
  - `clips[].sprite` (**required**) is the name of the `@Sprite` it animates;
    `fps` (12), `loop` (false).
  - `default` (`""` → first clip's sprite).
- A clip's id is the **name** of the `@Sprite` it animates (`sprite` field).
- Emits `animation_finished` (data = sprite name) when a non-looping clip ends.
- Requires `@Sprite`. Methods: `Play`, `Stop`, `IsPlaying`, `SetFlipX/Y`.

### `@Spin`

```json
{ "kind": "@Spin", "name": "spin", "args": { "speed": 3 } }
```

- Args: `speed` (3, radians per second). Adds `speed * dt` to owner rotation.

## Movement

The pattern: **behavior components compute a displacement and hand it to `@Mover`
for collision resolution.** Without a `@Mover`, they move the position directly
(teleport, no collision).

### `@Mover` (capability)

```json
{ "kind": "@Mover", "name": "mover", "args": {} }
```

- No args. Methods: `Teleport(x, y)`, `Move(dx, dy)`, `MoveTowards(target, dist)`.
- Resolves collisions per axis (slides along walls); solids block, pushables are
  pushed, triggers ignored. Emits `blocked_collision` (data = blocking object) when
  blocked.

### `@Velocity` (capability)

```json
{ "kind": "@Velocity", "name": "velocity", "args": { "vx": 0, "vy": 0 } }
```

- Args: `vx` (0), `vy` (0) — initial velocity. Integrates velocity through `@Mover`
  each frame, and zeroes an axis when the mover blocks it.
- Methods: `SetVelocity`, `Velocity`, `AddVelocity`.

### `@Gravity`

```json
{ "kind": "@Gravity", "name": "gravity", "args": { "acceleration": { "x": 0, "y": 980 }, "max_speed": 0 } }
```

- Args: `acceleration {x,y}` (default `{0, 980}`), `max_speed` (0 = no cap).
- Adds `acceleration * dt` to `@Velocity`, clamped to `max_speed`. Requires
  `@Velocity`.

### `@Friction`

```json
{ "kind": "@Friction", "name": "friction", "args": { "amount": 0, "axes": "both" } }
```

- Args: `amount` (0, px/s reduction), `axes` (`"both"`, or `"x"` / `"y"`).
- Damps `@Velocity` toward zero. Requires `@Velocity`.

### `@Bounce`

```json
{ "kind": "@Bounce", "name": "bounce", "args": { "vx": 100, "vy": 0 } }
```

- Args: `vx` (0), `vy` (0) — constant velocity. If both are 0, `vx` becomes 100.
- Reflects the blocked axis and emits `bounce` (data = surface normal `Vector2`).

### `@PlayerController`

```json
{ "kind": "@PlayerController", "name": "controller", "args": { "speed": 200 } }
```

- Args: `speed` (200). WASD / arrows → `@Mover`. Arcade/overhead movement; for
  velocity-based platformer feel, write a custom component that sets `@Velocity`.

### `@Chase`

```json
{ "kind": "@Chase", "name": "chase", "args": { "speed": 60, "target_tag": "player", "stop_distance": 0 } }
```

- Args: `speed` (60), `target_tag` (`"player"`), `stop_distance` (0 = disabled).
  Moves toward the nearest object with `target_tag`, stopping within
  `stop_distance`.

### `@Follow`

```json
{ "kind": "@Follow", "name": "follow", "args": { "target_tag": "player", "lerp": 0.2, "offset": { "x": 0, "y": 0 } } }
```

- Args: `target_tag` (`"player"`), `lerp` (0.2, per-frame smoothing 0..1),
  `offset {x,y}` (0,0). Lerps position toward the first tagged object + offset. No
  collision.

### `@Patrol`

```json
{ "kind": "@Patrol", "name": "patrol", "args": { "points": [ { "x": 0, "y": 0 }, { "x": 100, "y": 0 } ], "speed": 60, "ping_pong": false } }
```

- Args: `points [{x,y}…]` (needs ≥ 2 to move), `speed` (60), `ping_pong` (false).
  Walks between waypoints, looping (or ping-ponging) through them.

### `@Wander`

```json
{ "kind": "@Wander", "name": "wander", "args": { "speed": 40, "change_interval": 1.5 } }
```

- Args: `speed` (40), `change_interval` (1.5 s). Random direction, re-rolled every
  `change_interval` seconds.

## Collision & combat

### `@Collider`

```json
{ "kind": "@Collider", "name": "hitbox", "args": { "width": 32, "height": 32, "offset": { "x": 0, "y": 0 }, "mode": "solid", "push_factor": 1, "collides_with": [] } }
```

- Args: `width` (32), `height` (32), `offset {x,y}` (0,0), `mode` (`"solid"`),
  `push_factor` (1), `collides_with []` (empty).
- `mode`: `solid` (blocks), `pushable` (pushed by movers; `push_factor` scales the
  slide, 0 = immovable, 1 = full), `trigger` (detects only).
- `offset` shifts the rectangle relative to the owner's position (top-left corner).
  Use it when the sprite and hitbox have different origins.
- `collides_with` lists tags to interact with (empty = everything).
- Tracks overlaps each frame and emits `collision_enter` / `collision_exit`
  (data = other object).
- Methods: `GetBounds`, `SetSize`, `SetOffset`, `GetSize`, `CheckOverlap`,
  `ContainsPoint`, `GetOverlaps` (sorted by ID).

### `@Health`

```json
{ "kind": "@Health", "name": "health", "args": { "max": 100 } }
```

- Args: `max` (100). `Damage(amount)` / `Heal(amount)`.
- Emits `damaged` (data = amount) or `died` (data = owner). `IsDead()`, `Current()`.

### `@Damage`

```json
{ "kind": "@Damage", "name": "damage", "args": { "amount": 1, "target_tags": [], "cooldown": 0 } }
```

- Args: `amount` (1), `target_tags []` (empty = any), `cooldown` (0 = every frame).
  Applies damage to overlapping objects that have `@Health`, filtered by
  `target_tags`, once per `cooldown` seconds. Requires `@Collider`.

## Logic

### `@StateMachine`
A component-level state machine. The machine is in one named state at a time; each
state lists the transitions that may leave it.

```json
{
  "kind": "@StateMachine", "name": "state",
  "args": {
    "initial": "idle",
    "states": [
      { "name": "idle", "transitions": [ { "event": "start", "from": "", "to": "run", "delay": 0 } ] },
      { "name": "run",  "transitions": [] }
    ]
  }
}
```

- Args: `initial` (`""` → first state), `states [{name, transitions [{event, from, to, delay}]}]`.
  - `states[].name` (**required**) — the state key.
  - `transitions[].event` (**required**) — the trigger name.
  - `transitions[].from` (`""` = manual only) — see below.
  - `transitions[].to` (**required**) — target state.
  - `transitions[].delay` (0) — seconds to wait before the transition.
- A transition fires when its `event` occurs while the machine is in that state:
  - `from = ""` → manual only, reached via `Trigger(event)`.
  - `from = "component"` → the named component **on the same object**.
  - `from = "object.component"` → the named component on the named object.
  - `from = "scene"` → any component (source ignored).
  - `delay` → seconds to wait before the transition (0 = immediate).
- Emits `state_entered` (data = new state) and `state_exited` (data = previous
  state) on every change.
- Methods: `Trigger(event)`, `SetState(name)`, `Current()`, `Previous()`,
  `TimeInState()`, `JustEntered()`.

## Audio & misc

### `@Sound`

```json
{ "kind": "@Sound", "name": "sound", "args": { "sound": "assets/pickup.wav", "volume": 1, "loop": false, "play_on_start": false } }
```

- Args: `sound` (**required**), `volume` (1, 0–1), `loop` (false), `play_on_start`
  (false).
- `play_on_start` auto-plays on the first frame; otherwise drive via
  `Play(ctx)` / `Stop(ctx)`.

### `@TimedDespawn`

```json
{ "kind": "@TimedDespawn", "name": "despawn", "args": { "lifetime": 5 } }
```

- Args: `lifetime` (5 s). Destroys the owner after `lifetime`.

## Events reference

| Event | Emitted by | Data |
|-------|-----------|------|
| `blocked_collision` | `@Mover` | blocking object |
| `bounce` | `@Bounce` | surface normal `Vector2` |
| `collision_enter` / `collision_exit` | `@Collider` | the other object |
| `damaged` / `died` | `@Health` | amount / owner |
| `animation_finished` | `@Animator` | sprite name |
| `state_entered` / `state_exited` | `@StateMachine` | state name |

## How they combine

- **Movement stack** — `@PlayerController` (or `@Chase`/`@Patrol`/`@Wander`) +
  `@Mover` + `@Collider` = collision-aware motion. `@Velocity` + `@Gravity` +
  `@Friction` is the platformer alternative: they mutate `@Velocity`, which
  integrates through `@Mover`.
- **Animation** — `@Sprite` + `@Animator`: the animator drives the sprite's frame,
  visibility and flip; give one object several sprites and an animator to swap
  between them.
- **Combat** — `@Collider` (trigger or solid) + `@Health` + `@Damage`: `@Damage`
  reads its collider's overlaps and calls `Health.Damage`, which emits `damaged` /
  `died` for anything else to react to.
- **State machine** — `@StateMachine` listens to the events above: e.g. transition
  on `animation_finished` (from `@Animator`) or `died` (from `@Health`), and emit
  `state_entered`/`state_exited` so other components can react to state changes
  without polling.

## Scene-level (not components)

- **Camera** — `scene.camera {x, y, zoom, smoothing, lock_x, lock_y}`. `x`/`y` are
  the view's top-left corner in world coordinates; `zoom` is the scale factor
  (center anchored). `smoothing` lerps toward a follow target (0 = snap). Drive it
  from a custom component via `scene.Camera.Follow(obj)`.
- **`ui` flag** — an object with `"ui": true` is drawn in screen space (pixels),
  ignores the camera, and draws on top of all world objects.

## Related

- [Events](events.md) — the `Emit`/`On` model and the full event table.
- [Custom components](custom-components.md) — how these are all just the same
  `core.Component` interface you implement yourself.
- [JSON format](json-format.md) — the exact `args` key names (snake_case) for the
  tables above.
