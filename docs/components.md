# Built-in Components

> ← [Documentation index](README.md) · [Custom components](custom-components.md) · [Events](events.md)

IMGE ships a small library of **built-in components**. They are not special — they
implement the same `core.Component` interface as user components, and at build time
they are merged into the project's single `components` package, so any component
(built-in or custom) can reach any other component's concrete methods directly.

Every component's configurable fields are its **export variables**: JSON-tagged
fields populated from the component's `args` object in `.obj`/`.scene` files. All
args are **snake_case**.

## The component model

- **Kind** — `@Name` for built-ins (e.g. `@Sprite`), the struct type name for user
  components (e.g. `PlayerComponent`).
- **Lifecycle** — `Initialize()` once after the scene is fully assembled, then
  `Update(ctx)` every frame, then `Draw(renderer)`. Draw order within an object is
  by `draw_layer` (a field every component inherits via `BaseComponent`; lower
  draws first, equal keeps insertion order).
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

- Args: `texture`, `frame_width`, `frame_height`, `frame`, `width`, `height`,
  `flip_x`, `flip_y`, `tint`, `offset {x,y}`, `visible`.
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

- Args: `clips [{sprite, fps, loop}]`, `default`, `flip_x`, `flip_y`.
- A clip's id is the **name** of the `@Sprite` it animates (`sprite` field).
- Emits `animation_finished` (data = sprite name) when a non-looping clip ends.
- Requires `@Sprite`. Methods: `Play`, `Stop`, `IsPlaying`, `SetFlipX/Y`.

### `@Spin`
- Args: `speed` (radians per second). Adds `speed * dt` to owner rotation.

## Movement

The pattern: **behavior components compute a displacement and hand it to `@Mover`
for collision resolution.** Without a `@Mover`, they move the position directly
(teleport, no collision).

### `@Mover` (capability)
- No args. Methods: `Teleport(x, y)`, `Move(dx, dy)`, `MoveTowards(target, dist)`.
- Resolves collisions per axis (slides along walls); solids block, pushables are
  pushed, triggers ignored. Emits `blocked_collision` (data = blocking object) when
  blocked.

### `@Velocity` (capability)
- Args: `vx`, `vy` (initial). Integrates velocity through `@Mover` each frame, and
  zeroes an axis when the mover blocks it.
- Methods: `SetVelocity`, `Velocity`, `AddVelocity`.

### `@Gravity`
- Args: `acceleration {x,y}` (default `{0, 980}`), `max_speed` (0 = no cap).
- Adds `acceleration * dt` to `@Velocity`, clamped to `max_speed`. Requires
  `@Velocity`.

### `@Friction`
- Args: `amount` (px/s reduction), `axes` (`"x"`, `"y"`, or `"both"`).
- Damps `@Velocity` toward zero. Requires `@Velocity`.

### `@Bounce`
- Args: `vx`, `vy`. Moves at constant velocity; reflects the blocked axis and emits
  `bounce` (data = surface normal `Vector2`).

### `@PlayerController`
- Args: `speed`. WASD / arrows → `@Mover`. Arcade/overhead movement; for
  velocity-based platformer feel, write a custom component that sets `@Velocity`.

### `@Chase`
- Args: `speed`, `target_tag`, `stop_distance`. Moves toward the nearest object
  with `target_tag`, stopping within `stop_distance`.

### `@Follow`
- Args: `target_tag`, `lerp`, `offset {x,y}`. Lerps position toward the first
  tagged object + offset. No collision.

### `@Patrol`
- Args: `points [{x,y}…]`, `speed`, `ping_pong`. Walks between waypoints, looping
  (or ping-ponging) through them.

### `@Wander`
- Args: `speed`, `change_interval`. Random direction, re-rolled every
  `change_interval` seconds.

## Collision & combat

### `@Collider`
- Args: `width`, `height`, `offset {x,y}`, `mode`, `push_factor`, `collides_with []`.
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
- Args: `max` (default 100). `Damage(amount)` / `Heal(amount)`.
- Emits `damaged` (data = amount) or `died` (data = owner). `IsDead()`, `Current()`.

### `@Damage`
- Args: `amount`, `target_tags []`, `cooldown`. Applies damage to overlapping
  objects that have `@Health`, filtered by `target_tags`, once per `cooldown`
  seconds. Requires `@Collider`.

## Logic

### `@StateMachine`
A component-level state machine. The machine is in one named state at a time; each
state lists the transitions that may leave it.

- Args: `initial`, `states [{name, transitions [{event, from, to, delay}]}]`.
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
- Args: `sound`, `volume` (0–1), `loop`, `play_on_start`.
- `play_on_start` auto-plays on the first frame; otherwise drive via
  `Play(ctx)` / `Stop(ctx)`.

### `@TimedDespawn`
- Args: `lifetime` (seconds). Destroys the owner after `lifetime`.

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
  the view center in world coordinates; `zoom` is the scale factor (center
  anchored). `smoothing` lerps toward a follow target (0 = snap). Drive it from a
  custom component via `scene.Camera.Follow(obj)`.
- **`ui` flag** — an object with `"ui": true` is drawn in screen space (pixels),
  ignores the camera, and draws on top of all world objects.

## Related

- [Events](events.md) — the `Emit`/`On` model and the full event table.
- [Custom components](custom-components.md) — how these are all just the same
  `core.Component` interface you implement yourself.
- [JSON format](json-format.md) — the exact `args` key names (snake_case) for the
  tables above.
