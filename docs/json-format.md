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
    "width": 800,
    "height": 600
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
| `window.width` | int | `800` | logical resolution (letterboxed) |
| `window.height` | int | `600` | logical resolution |
| `game.target_fps` | int | `60` | target frame rate |
| `game.initial_scene` | string | `"main"` | scene shown first |

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

`camera` fields: `x`, `y` (view **center**, world coords), `zoom` (scale, center
anchored), `smoothing` (0 = snap), `lock_x`, `lock_y`.

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

A component entry appears in `objects`/`components` arrays everywhere:

```json
{ "kind": "@Sprite", "name": "sprite", "args": { "texture": "assets/player.png", "width": 32, "height": 32 } }
```

| Field | Type | Notes |
|---|---|---|
| `kind` | string | `@Name` for built-ins, the Go struct type name for user components (e.g. `PlayerComponent`) |
| `name` | string | unique **within the object**; used to look components up by name |
| `args` | object | export variables — keys are the component's `json` tags (see the rule above) |

Component `name` is how you address a specific component (e.g. an `@Animator`
targets a sprite by name, and `core.GetFromNamed` looks it up). Two components of
the same kind can coexist under different names (e.g. several `@Sprite`s).

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
| `math.Color` | `{ "r": 255, "g": 0, "b": 0, "a": 255 }` (0–255) |
| `*bool` | `true` / `false` (e.g. `@Sprite` `visible`) |

Nested structs use their own `json` tags: `acceleration {x,y}`, `offset {x,y}`,
`tint {r,g,b,a}`.

## Color and vector shorthand

- **Hex string** — used only for a scene's `background_color` (`"#RRGGBB"`, etc.).
- **`{r,g,b,a}` object** — used for component args that take a `math.Color` (e.g.
  `@Sprite` `tint`). Channels are `uint8`, 0–255. Omitted `a` defaults to 255.
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
        { "kind": "@Collider", "name": "body", "args": { "width": 800, "height": 40, "mode": "solid" } }
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
