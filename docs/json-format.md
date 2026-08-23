# JSON format reference

This is the complete format for every JSON file IMGE reads. All numbers are
floats unless stated otherwise; booleans are `true`/`false`.

> **One rule to memorize — `args` keys must match the `json` tags exactly.**
> A component's configurable fields are populated by unmarshaling the component's
> `args` object into its exported, JSON-tagged fields. Matching is **exact** (the
> tags are all **snake_case** for built-ins): `collides_with`, `target_tag`,
> `max_speed`, `stop_distance`, `target_tags`, `change_interval`, `play_on_start`,
> `frame_width`, `push_factor`, `flip_x`, …
>
> camelCase keys (`collidesWith`, `maxSpeed`, …) are **silently ignored** — Go's
> `encoding/json` does not fuzzy-match against the tag name. If a value seems to be
> ignored, check the key casing against the component's tag in
> [Components](components.md).

## `game.imge`

```json
{
  "name": "My Game",
  "format_version": 1,
  "window": {
    "title": "My Game",
    "width": 640,
    "height": 360,
    "fullscreen": false,
    "pixel_per_unit": 1
  },
  "game": {
    "target_fps": 60,
    "initial_scene": "main"
  }
}
```

| Field | Type | Default | Notes |
|---|---|---|---|
| `name` | string | `"My Game"` | slugified into the output filename |
| `format_version` | int | `1` | the IMGE project-format version this project targets (see below) |
| `window.title` | string | `"My IMGE Game"` | window title |
| `window.width` | int | `640` | logical resolution (the renderer draws at this size) |
| `window.height` | int | `360` | logical resolution |
| `window.fullscreen` | bool | `false` | start the game fullscreen |
| `window.pixel_per_unit` | int | `1` | framebuffer pixels per logical unit (sub-unit render precision; see below) |
| `game.target_fps` | int | `60` | target frame rate |
| `game.initial_scene` | string | `"main"` | scene shown first |

`width`/`height` is the **logical** resolution the game is rendered at — not the
window size. Each frame is drawn to a render target of `width × height` logical
units, then scaled into the window/browser with letterboxing to preserve the
aspect ratio (desktop opens at the largest integer scale that fits your monitor;
the web build fills the browser). Press **F11** (or **Alt+Enter**) to toggle
fullscreen at any time; the window is always locked to its fit size.

`pixel_per_unit` (default `1`) is how many framebuffer pixels each logical unit
occupies along one axis. The actual render target is `width×pixel_per_unit` by
`height×pixel_per_unit`, and the window auto-fits to **that** size. At `1` the
engine is pixel-perfect: a position like `54.8` snaps to the nearest whole pixel,
which can make smooth fractional movement appear to jitter. Raising it to `2`,
`4`, etc. lets the rasterizer show half-, quarter-, … pixel positions, so motion
stays smooth without changing anything in the game world — positions, collider
sizes, and camera math are all unaffected; only the render resolution (and the
mouse coordinate mapping, which is normalized back to logical units) changes.
There is no upper limit, but the framebuffer (and its memory) grows with the
square of the value, so keep it as small as your game's smoothness needs.

`format_version` is **not** a free-form game-version field — it's an engine-owned
number that pins the project format (the `game.imge` schema, the scene/object JSON
shape, the component-kind scheme, and the path rules). The `imge` tool validates it
at build time: a project declaring a newer format than the tool knows is rejected
("update the imge tool"), and an older one needs migrating. It is **not** bumped on
every engine release — only when the format actually changes — and it is not the
place to record your game's own version (use VCS tags for that).

## Scene files (`.scene`)

```json
{
  "name": "main",
  "background_color": "#8fd0ff",
  "camera": { "x": 80, "y": 270, "zoom": 1, "smoothing": 0.1, "lock_y": true },
  "objects": [ /* … */ ]
}
```

| Field | Type | Notes |
|---|---|---|
| `name` | string | scene identifier (used by `initial_scene` and `SwitchScene`) |
| `background_color` | string | hex color `#RGB` / `#RGBA` / `#RRGGBB` / `#RRGGBBAA` |
| `camera` | object | optional initial camera (see [Camera](scenes-and-camera.md#camera)) |
| `objects` | array | the objects to place, in order |

`camera` fields: `x`, `y` (view's **top-left corner**, world coords), `zoom` (scale,
center anchored), `smoothing` (0 = snap), `lock_x`, `lock_y`.

### Scene objects

Each entry in `objects` is either a **file reference** or an **inline definition**.

**File reference** (reuse a `.obj` template, overriding transform/depth):

```json
{ "file": "objects/coin.obj", "transform": { "position": { "x": 120, "y": 80 } } }
```

**Inline definition** (full object written directly):

```json
{
  "name": "wall",
  "depth": 0,
  "ui": false,
  "tags": ["wall"],
  "transform": { "position": { "x": 0, "y": 0 } },
  "components": [ /* … */ ]
}
```

| Field | Type | Notes |
|---|---|---|
| `file` | string | project-root-relative path to a `.obj` template (e.g. `objects/coin.obj` or `entities/enemy/enemy.obj`) |
| `name` | string | object name (inline objects; template objects use the template's `name`) |
| `transform` | object | `position {x,y}`, `rotation` (radians), `scale {x,y}` |
| `depth` | number | draw order (higher = on top). On a file ref, overrides the template's depth |
| `ui` | bool | screen-space object (ignores camera, drawn on top) |
| `tags` | string[] | tags for filtering |
| `components` | array | component instances |

A file reference merges the template's components/tags/depth with the scene's
transform (and depth) override. `ui` is `template.ui || scene.ui`.

## Object files (`.obj`)

A template — identical to an inline scene object but with **no transform**:

```json
{
  "name": "Crate",
  "depth": 2,
  "ui": false,
  "tags": ["crate"],
  "components": [ /* … */ ]
}
```

`name`, `depth`, `ui`, `tags`, `components` — same meaning as above.

## Component instances

Every object (inline or template) has a `components` array. Each element is one
**component instance** — a specific component with a kind, a name, and the args
that configure it:

```json
{ "kind": "@Sprite", "name": "sprite", "args": { "texture": "assets/player.png", "width": 32, "height": 32 } }
```

| Field | Type | Notes |
|---|---|---|
| `kind` | string | `@Name` for built-ins (e.g. `@Sprite`, `@Collider`); the Go struct type name for user components (e.g. `PlayerComponent`) |
| `name` | string | unique **within the object**; used to look components up by name |
| `args` | object | export variables — keys are the component's `json` tags (see the rule above) |

### `kind`

How the build tool maps a kind to a component:

- Built-ins start with `@` and are the struct name without the `Component`
  suffix: `@Sprite`, `@Animator`, `@Collider`, …
- User components use the Go struct type name, e.g. `PlayerComponent` (declared
  in the project's `components/` package). A kind that matches neither produces a
  build error ("unknown component").

### `name`

The name is scoped to the object — the same name can be reused on different
objects. It is how one component addresses another:

- An `@Animator` targets the `@Sprite` it animates **by that sprite's `name`**
  (the `sprite` field of each clip).
- `@StateMachine` transition `from` scopes reference component names.
- `core.GetFromNamed[*T](owner, name)` looks a component up by name.

Two components of the same kind can coexist under different names (e.g. several
`@Sprite`s). Different kinds coexist too — a solid body (`@Collider`) plus a
trigger hitbox (`@Trigger`) on the same object. `name` may be omitted (defaults
to `""`), but any component you need to address — a sprite driven by an animator,
a trigger a `@Damage` reads — should be named.

### `args`

`args` is a JSON object whose keys are the component's **export variables**: the
`json` tags on its exported struct fields. It may be `{}` (or omitted) for a
component that takes none (e.g. `@Mover`). Every built-in's tags, defaults, and a
copy-paste example live in [Components](components.md) — paste the entry into an
object's `components` array and adjust.

Keys are matched **exactly** (snake_case for built-ins); a wrong case is silently
ignored, not an error.

## `args` value types

The values inside `args` map to Go types:

| Go type | JSON |
|---|---|
| `float64` / `int` | number (`42`, `0.5`) |
| `string` | string |
| `bool` | `true` / `false` |
| `[]string` | array of strings (`["player", "wall"]`) |
| `[]math.Vector2` | array of `{x,y}` (e.g. `@Patrol` `points`) |
| `[]Clip` | array of objects (e.g. `@Animator` `clips`) |
| `math.Vector2` | `{ "x": 0, "y": 0 }` |
| `math.Color` | `{ "r": 255, "g": 0, "b": 0, "a": 255 }` (0–255) or `"#RRGGBB"` |
| `math.ColorTransform` | `{ "grayscale": false, "hue": 0, "hue_to": {…}, "tint": {…}, "solid": false }` (e.g. `@Sprite` `color`) |
| `*bool` | `true` / `false` (e.g. `@Sprite` `visible`) |

Nested structs use their own `json` tags: `acceleration {x,y}`, `offset {x,y}`,
`color {grayscale, hue, hue_to {r,g,b,a}, tint {r,g,b,a}, solid}`.

## Color and vector shorthand

- **Hex string** — a scene's `background_color`, or any `math.Color` arg (e.g.
  `@Sprite` `color.tint`, `color.hue_to`): `"#RGB"`, `"#RGBA"`, `"#RRGGBB"`, or
  `"#RRGGBBAA"` (the leading `#` is optional).
- **`{r,g,b,a}` object** — also accepted for `math.Color` args. Channels are
  `uint8`, 0–255. Omitted `a` defaults to 255.
- **`{x,y}` object** — any `math.Vector2` (positions, velocities, offsets, points).

## Full worked example

```json
{
  "name": "main",
  "background_color": "#141430",
  "camera": { "x": 80, "y": 270, "zoom": 1, "smoothing": 0.1, "lock_y": true },
  "objects": [
    {
      "name": "ground",
      "tags": ["platform"],
      "transform": { "position": { "x": 0, "y": 500 } },
      "components": [
        { "kind": "@Sprite", "name": "sprite", "args": { "texture": "assets/platform.png", "width": 800, "height": 40 } },
        { "kind": "@Collider", "name": "body", "args": { "width": 800, "height": 40 } }
      ]
    },
    { "file": "objects/coin.obj", "transform": { "position": { "x": 380, "y": 370 } } }
  ]
}
```

## Next

- [Objects](objects.md) — what these fields *do* at runtime.
- [Scenes & camera](scenes-and-camera.md) — scene loading and switching.
- [Components](components.md) — every component's exact `args` tags.
