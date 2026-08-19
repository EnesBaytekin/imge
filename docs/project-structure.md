# Project structure

An IMGE project is a directory containing a `game.imge` marker file plus whatever
files your game needs. **The layout is free-form** — the build scans the whole
project root and finds components, scenes, objects, and assets wherever you put
them, and every path you write is **relative to the project root**.

`imge init` scaffolds a conventional layout to get you started:

```
mygame/
├── game.imge          # window title/size, FPS, initial scene
├── components/        # your Go components
├── scenes/            # scene definitions (.scene)
├── objects/           # object templates (.obj)
└── assets/            # images (PNG/JPEG) and sounds (WAV/MP3/OGG)
```

…but **none of these directories is required**. Only `game.imge` is (the `imge`
tool checks for it — it marks the project root). You are free to reorganize, nest,
or flatten everything — the engine does not care. For example, this is equally
valid:

```
game-demo/
├── game.imge
└── objects/
    ├── player/
    │   ├── images/
    │   │   ├── idle.png
    │   │   └── run.png
    │   ├── player.go      # a component
    │   └── player.obj     # an object template
    └── enemy/
        └── …
```

Here `player.go` is a component that lives under `objects/player/`. Its `kind` is
its struct type name (e.g. `PlayerComponent`), independent of where the file sits.
Its `idle.png` is referenced as `objects/player/images/idle.png`. Nothing has to be
in a fixed folder.

## The one rule: root-relative paths

Every path you write — an asset, a `.obj` `file` reference, a component `kind` —
is relative to the project root:

```json
{ "file": "objects/player/player.obj", "transform": { "position": { "x": 120, "y": 80 } } }
```

```json
{ "kind": "PlayerComponent", "name": "brain", "args": { "speed": 120 } }
```

```json
{ "kind": "@Sprite", "name": "sprite", "args": { "texture": "objects/player/images/idle.png" } }
```

The only place a path isn't root-relative is inside a `.go` source file's `import`
statements — those are ordinary Go import paths, unaffected by this.

## `game.imge`

The single source of truth for window and game settings. See
[game.imge reference](json-format.md#gameimge).

```json
{
  "name": "My Game",
  "format_version": 1,
  "window": { "title": "My Game", "width": 800, "height": 600 },
  "game": { "target_fps": 60, "initial_scene": "main" }
}
```

- `name` — the game name (used for the output filename, slugified).
- `window.title` — window title.
- `window.width` / `window.height` — the **logical** resolution. The window/browser
  letterboxes this size (preserving aspect ratio) rather than stretching.
- `game.target_fps` — target frame rate (default 60).
- `game.initial_scene` — the name of the scene to show first. **This must match the
  `name` field inside the `.scene` file** (or its filename if the `name` field is
  absent — see [scenes](scenes-and-camera.md#loading-scenes)).

## Scene files (`.scene`)

Each `.scene` file is one scene. Every `*.scene` file under the project root is
loaded at startup (at any depth), and registered by its `name` field — the filename
with the `.scene` suffix stripped is the fallback when `name` is absent. Two scenes
with the same `name` collide, so keep names unique across the whole project. See
[scene format](json-format.md#scene-files-scene).

## Object templates (`.obj`)

Reusable object templates — an object's components, tags, and depth, but **no
transform** (position/rotation/scale are set where the object is placed). A scene
places a template via a root-relative `file` reference:

```json
{ "file": "objects/coin.obj", "transform": { "position": { "x": 120, "y": 80 } } }
```

`file` is read as a project-root-relative path, exactly as written — write the full
path from the root (e.g. `objects/coin.obj`, or `entities/enemy/enemy.obj`).

## Assets

Images and sounds, embedded into the build. Textures and audio are referenced by a
project-root-relative path, taken **exactly as written**: `"texture": "player.png"`
means `<project root>/player.png`, and `"texture": "assets/player.png"` means the
`assets/` directory. There is no `assets/` fallback — write the full path to the
file you mean (e.g. `assets/player.png` or `objects/player/images/idle.png`).

Supported formats:

- **Images** — PNG, JPEG (pixel-art style: use PNG; Ebitengine scales with nearest-neighbour).
- **Audio** — WAV, MP3, OGG (detected by file extension).

## Components (`.go`)

Your custom components. Each file declares `package components` and exports exactly
one struct that embeds `core.BaseComponent` — that one type is what the build tool
registers, keyed by its **struct type name** (e.g. `PlayerComponent`). The build
scans the **whole project root** for these files, so they can live anywhere. See
[Custom components](custom-components.md) for the full contract.

## Generated / build artifacts

- `imge_build/` — build output (see [the `imge` tool](cli.md#output)). Safe to delete.
- `.imge_build/` — a temporary directory the builder cleans up; never commit it.

The `.imge_build`, `imge_build`, `.git`, and `node_modules` directories are excluded
from the scan and the embed, so you can nest projects or keep a build around without
confusing the builder.

## Next

- [JSON format](json-format.md) — every field, in detail.
- [The `imge` tool](cli.md) — building and running.
- [Custom components](custom-components.md) — the Go side.
