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
  nil pointer when nothing matches; nil-check. See
  [Custom components → Reaching sibling components](custom-components.md#reaching-sibling-components)
  for the full table.
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
| `@Collider` | Rectangle physics body; solid or pushable (multiple = compound body) |
| `@Trigger` | Sensor region; detects overlaps, emits `trigger_enter`/`trigger_exit` |
| `@Health` | HP pool; emits `damaged` / `died` |
| `@Damage` | Applies damage to a `@Trigger`'s overlapping targets |
| `@StateMachine` | Named states + event-driven transitions |
| `@Sound` | One-shot sound effect or looping music |
| `@TimedDespawn` | Destroys the owner after N seconds |
| `@Panel` | Filled rectangle: flat color or nine-sliced texture, optional outline |
| `@Label` | Single-line or width-wrapped text |
| `@Button` | Clickable element (state textures + centered text) that emits an event on click |
| `@TextInput` | Single-line editable text field with caret, focus, and placeholder |

## Colors

Every color arg — `color`, `text_color`, `background_color`, `outline_color`, the
predefined sprite tint fields, and a scene's `background_color` — accepts a color in
**either** of two forms:

- **Hex string** — `"#RRGGBB"` or `"#RRGGBBAA"`. Alpha is optional and defaults to
  opaque (255). Shorthand `#RGB` / `#RGBA` works, and the leading `#` is optional:
  `"#f00"`, `"ff0000"`, and `"#ff000080"` are all valid.
- **RGBA object** — `{ "r": 255, "g": 0, "b": 0, "a": 255 }`. `a` is optional
  (defaults to 255); `r`/`g`/`b` default to 0.

The examples in this reference use hex (shorter and more readable), but both forms
decode to the same color and are interchangeable:

```json
{ "color": "#ff4a6b" }                                        // opaque pink-red
{ "color": "#ff4a6b80" }                                      // 50% alpha
{ "color": { "r": 255, "g": 74, "b": 107, "a": 255 } }        // same as the first
```

Alpha `0` is fully transparent — e.g. a panel's `outline_color` defaults to
transparent so there is no outline unless you give it one.

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
    "color": {
      "grayscale": false,
      "hue": 0,
      "hue_to": null,
      "tint": { "r": 255, "g": 255, "b": 255, "a": 255 },
      "solid": false
    },
    "offset": { "x": 0, "y": 0 },
    "visible": true
  }
}
```

- Args: `texture` (**required**), `frame_width` (0), `frame_height` (0), `frame`
  (0), `width` (0 = natural), `height` (0 = natural), `flip_x` (false), `flip_y`
  (false), `color` (identity), `offset {x,y}` (0,0), `visible` (true).
- `color` recolors the texture at draw time, applied in this order:
  `grayscale` (desaturate) → `hue` / `hue_to` (rotate) → `tint` (multiply) →
  `solid` (silhouette). See [Color effects](color-effects.md) for the full guide
  with visual side-by-side comparisons.
- `frame_width = 0` → whole texture is one frame. `frame_width > 0` and
  `frame_height = 0` → a single horizontal strip. Both `> 0` → a grid cut
  row-by-row. `FrameCount()` is always computed from the texture size.
- `width`/`height` are the display size (0 = natural frame size).
- Methods: `SetFrame`, `SetTexture`, `SetColor`, `SetFlipX/Y`, `SetVisible`,
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
- Resolves collisions per axis (slides along walls): solids block, pushables are
  pushed (mover and obstacle advance together at a reduced speed, never
  interpenetrating). `@Trigger`s never affect movement. Emits `blocked_collision`
  (data = blocking object) when blocked.

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
{ "kind": "@Collider", "name": "hitbox", "args": { "width": 32, "height": 32, "offset": { "x": 0, "y": 0 }, "push_factor": 0, "collides_with": [] } }
```

- Args: `width` (32), `height` (32), `offset {x,y}` (0,0), `push_factor` (0),
  `collides_with []` (empty).
- `push_factor` sets how the collider responds when a mover hits it: **0 = solid**
  (blocks outright, the default), and a value in **(0, 1] = pushable** — the higher
  the value, the lighter (easier to push; 1 = weightless).
- Multiple `@Collider`s on one object form a **single compound body**: a mover tests
  the whole union of rectangles and treats them as one physical object.
- `@Collider` is **pure physics** — it never emits events or tracks overlaps. Use
  `@Trigger` for detection.
- `offset` shifts the rectangle relative to the owner's position (top-left corner).
  Use it when the sprite and hitbox have different origins.
- `collides_with` lists tags to interact with (empty = everything).
- Methods: `GetBounds`, `SetSize`, `SetOffset`, `GetSize`, `CheckOverlap`,
  `ContainsPoint`.

### `@Trigger`

```json
{ "kind": "@Trigger", "name": "trigger", "args": { "width": 32, "height": 32, "offset": { "x": 0, "y": 0 }, "collides_with": [] } }
```

- Args: `width` (32), `height` (32), `offset {x,y}` (0,0), `collides_with []`
  (empty).
- A sensor region: it **never blocks or is pushed** — it only detects when another
  object's `@Collider` shapes overlap it. An object is detected when *any* of its
  colliders overlaps (compound-aware).
- Tracks overlaps each frame and emits `trigger_enter` / `trigger_exit`
  (data = other object).
- Multiple `@Trigger`s on one object are independent sensors, each with its own
  area.
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
{ "kind": "@Damage", "name": "damage", "args": { "amount": 1, "target_tags": [], "cooldown": 0, "trigger": "" } }
```

- Args: `amount` (1), `target_tags []` (empty = any), `cooldown` (0 = every frame),
  `trigger` (`""` = the owner's first `@Trigger`; otherwise the name of a specific
  `@Trigger`).
- Reads the target `@Trigger`'s overlaps and applies damage to overlapping objects
  that have `@Health`, filtered by `target_tags`, once per `cooldown` seconds.
  Requires `@Trigger`.

## UI

UI is built the same way as everything else, with one convention: **an object with
`"ui": true` is a window or container, and each element inside it is a UI
component.** The object's `Position` is the window's top-left; each element's
`offset {x,y}` is its own top-left *relative to that window*, and its `width` /
`height` are its extent. This keeps a whole window (and its many buttons, labels,
panels) on a single object instead of one object per element.

Every UI component embeds `BaseUIComponent`, which adds these common args on top of
`draw_layer`:

| Arg | Default | Meaning |
|-----|---------|---------|
| `offset {x,y}` | 0,0 | element top-left, relative to the owner object's position |
| `width` / `height` | 0 | element extent in logical units |
| `visible` | true | whether the element draws |
| `enabled` | true | whether the element receives input (a disabled element still draws) |
| `focusable` | false | whether it can take keyboard focus (used by the upcoming `@UIManager`) |
| `group` | "" | free-form label; the editor renders same-`group` elements as a folder |

### `@Panel`
Draws a filled rectangle. It has two mutually exclusive fills, plus an optional
outline drawn **over** whichever fill is used:

- **Flat color** — when `texture` is empty, it fills with `color`.
- **Nine-slice** — when `texture` is set, it nine-slices that texture with `border`
  (corners keep their natural size, edges and center stretch). `color` is ignored.
- **Outline** — when `outline_color` is non-transparent and `outline_thickness > 0`,
  a stroke is drawn over the fill, independent of the fill choice.

Works in world space too (e.g. a platform block on a non-UI object).

**Flat color fill:**
```json
{ "kind": "@Panel", "name": "bg", "args": { "color": "#14141e", "width": 200, "height": 100 } }
```

**Nine-sliced texture** (corners stay sharp at any size):
```json
{ "kind": "@Panel", "name": "bg", "args": {
  "texture": "assets/panel.png",
  "border": { "left": 4, "top": 4, "right": 4, "bottom": 4 },
  "width": 200, "height": 100
} }
```

**Flat color + outline** (a bordered box):
```json
{ "kind": "@Panel", "name": "bg", "args": {
  "color": "#14141e",
  "outline_color": "#3b3b4d", "outline_thickness": 1,
  "width": 200, "height": 100
} }
```

- Args: `color` (opaque black; the flat fill when no texture), `texture` (`""` =
  flat fill), `border {left,top,right,bottom}` (slice inset in texture pixels),
  `outline_color` (transparent = none), `outline_thickness` (0).

### `@Label`
Draws a single line of text, or wraps to `max_width` when `max_width > 0`. No
interaction, no background. Text is left-aligned (alignment is deferred to a later
phase).

```json
{
  "kind": "@Label", "name": "title",
  "args": {
    "text": "Inventory", "font_id": "", "size": 0,
    "color": "#ffffff",
    "max_width": 0, "wrap": "word"
  }
}
```

- Args: `text`, `font_id` (`""` = built-in pixel font), `size` (0 = font default),
  `color` (white), `max_width` (0 = single line), `wrap` (`"word"` \| `"char"` \|
  `"clip"`). See [Rendering → Text](rendering.md) for `wrap` behavior.

### `@Button`
A clickable element: a background (flat `color`, or nine-sliced textures per state)
and centered text. Hover and click are detected against its rect; on click (press
then release inside) it emits its `event`.

**The background** is a set of texture paths — `normal`, `hover`, `pressed`,
`disabled` — each nine-sliced with `border` when set. On each frame the button picks
one texture for its current state, and a state with no texture **falls back to
`normal`**. If `normal` is also empty, the button fills with flat `color` instead.
There is **no automatic tinting**: hover/pressed/disabled are explicit textures you
supply, not color effects applied to `normal`.

```json
{
  "kind": "@Button", "name": "ok",
  "args": {
    "text": "OK", "font_id": "", "size": 0,
    "text_color": "#ffffff",
    "normal": "assets/btn.png",
    "hover": "assets/btn_hover.png",
    "pressed": "assets/btn_pressed.png",
    "disabled": "assets/btn_disabled.png",
    "border": { "left": 4, "top": 4, "right": 4, "bottom": 4 },
    "color": "#3c3c46",
    "event": "confirm"
  }
}
```

- Args: `text`, `font_id`, `size`, `text_color` (white),
  `normal`/`hover`/`pressed`/`disabled` (texture paths; a state with no texture
  falls back to `normal`, and `normal = ""` falls back to `color`), `border`,
  `color` (flat fill used only when `normal` is empty), `event` (event name emitted
  on click; empty = none).
- The click event's data is the **button component** (reach its owner via
  `GetOwner()`), so a handler can tell which button was pressed and act on the
  window it belongs to.
- **`enabled: false`** means "draws but not clickable". Set it from code
  (`SetEnabled(false)`) for logic like "disable Next until the form is valid", and
  give the button a `disabled` texture so the non-interactive state is visually
  distinct. Occlusion (a button *behind* another panel) is a separate,
  `@UIManager`-level concern, not this flag.

### `@TextInput`
A single-line editable text field with classic textbox behavior:

- **Click** to focus (blinking caret); click elsewhere to blur.
- **Type** to insert characters at the caret; **Backspace** deletes the rune before
  it; **←/→** move the caret; **Enter** submits.
- When the text grows wider than the box it **scrolls horizontally** so the caret
  stays visible (the beginning or end scrolls out of view, exactly like a normal
  one-line field).
- When empty and unfocused it can show a `placeholder` (in `placeholder_color`).

```json
{
  "kind": "@TextInput", "name": "name_field",
  "args": {
    "text": "", "font_id": "", "size": 0,
    "text_color": "#ffffff",
    "placeholder_color": "#808080",
    "background_color": "#1e1e28",
    "texture": "", "border": { "left": 4, "top": 4, "right": 4, "bottom": 4 },
    "placeholder": "Enter name…", "max_length": 0,
    "event": "submitted",
    "width": 160, "height": 24
  }
}
```

- Args: `text` (initial value), `font_id`, `size`, `text_color` (white),
  `placeholder_color` (gray), `background_color` (transparent = none), `texture` +
  `border` (nine-sliced background; `""` = flat `background_color`), `placeholder`,
  `max_length` (0 = unlimited), `event` (event name emitted on Enter; empty = none).
- The submit event's data is the **text string**.
- The caret is a rune index, so Unicode input inserts and deletes by rune. A
  two-pixel horizontal padding (`inputPadX`) is reserved inside the box, so the text
  never touches the border/edge.
- Focuses itself on click (per-component for now; Faz 3's `@UIManager` will
  coordinate focus across elements).

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
| `trigger_enter` / `trigger_exit` | `@Trigger` | the other object |
| `damaged` / `died` | `@Health` | amount / owner |
| `animation_finished` | `@Animator` | sprite name |
| `state_entered` / `state_exited` | `@StateMachine` | state name |

`@Button` and `@TextInput` emit a **configurable** event — the name is whatever you
put in their `event` arg, not a fixed name. `@Button` emits it on click with the
button component as data; `@TextInput` emits it on Enter with the text string as
data.

## How they combine

- **Movement stack** — `@PlayerController` (or `@Chase`/`@Patrol`/`@Wander`) +
  `@Mover` + `@Collider` = collision-aware motion. `@Velocity` + `@Gravity` +
  `@Friction` is the platformer alternative: they mutate `@Velocity`, which
  integrates through `@Mover`.
- **Animation** — `@Sprite` + `@Animator`: the animator drives the sprite's frame,
  visibility and flip; give one object several sprites and an animator to swap
  between them.
- **Combat** — `@Trigger` + `@Health` + `@Damage`: `@Damage` reads its trigger's
  overlaps and calls `Health.Damage`, which emits `damaged` / `died` for anything
  else to react to.
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
