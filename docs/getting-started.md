# Getting started

This walks you from a clean machine to a running IMGE game.

## Prerequisites

- **Go 1.24+** — the `imge` tool shells out to the Go toolchain to compile your
  game. Download it from [go.dev/dl](https://go.dev/dl/).
- **The `imge` CLI** — a single binary, downloaded from the
  [latest release](https://github.com/EnesBaytekin/imge/releases) and placed on
  your `PATH`.

> **Native Linux builds** additionally need a C toolchain and system headers
> (Ebitengine links GLFW → X11/GL and oto → ALSA via pkg-config). If `imge build`
> complains about missing dependencies, install them once:
>
> ```sh
> sudo apt update && sudo apt install -y build-essential pkg-config \
>   libgl1-mesa-dev libx11-dev libxcursor-dev libxrandr-dev \
>   libxinerama-dev libxi-dev libxxf86vm-dev libasound2-dev
> ```
>
> Web and Windows cross-builds are pure Go and don't need any of this.

## 1. Scaffold a project

```sh
mkdir mygame && cd mygame
imge init            # blank project
# or: imge init sample   — the sample platformer demo
```

`imge init` only works in an **empty** directory (it refuses otherwise, to avoid
overwriting your files). It creates the project skeleton described in
[Project structure](project-structure.md).

## 2. Run it

```sh
imge run
```

`imge run` builds and launches the desktop game. For a blank project this opens a
window showing the initial scene; for the sample, you get a playable platformer
(WASD/arrows to move, Space to jump, E to attack).

## 3. Make it yours

The two places you spend your time:

1. **`.scene` files** — place objects and give them components (the level layout).
   `imge init` puts them in `scenes/`, but they can live anywhere (see
   [Project structure](project-structure.md)).
2. **`.go` components** — write custom components for game-specific behavior.
   Conventionally in `components/`, but the build finds them anywhere under the
   project root.

The [JSON format](json-format.md) reference covers the scene/object files in full;
[Custom components](custom-components.md) covers the Go side.

## A minimal first object

Add this to `scenes/main.scene` to put a 32×32 sprite on screen:

```json
{
  "name": "main",
  "background_color": "#1a1a2e",
  "objects": [
    {
      "name": "hero",
      "transform": { "position": { "x": 100, "y": 100 } },
      "components": [
        { "kind": "@Sprite", "name": "sprite", "args": { "texture": "assets/player.png", "width": 32, "height": 32 } }
      ]
    }
  ]
}
```

Drop a `player.png` into `assets/` and reference it as `assets/player.png` — every
path is project-root-relative, so `player.png` alone means `<project root>/player.png`.
Then `imge run`.

## Where to go next

- [Project structure](project-structure.md) — understand the layout.
- [Objects](objects.md) — the object model.
- [Scenes & camera](scenes-and-camera.md) — scenes, switching, camera.
- [Components](components.md) — the built-in component library.
- [Cookbook](cookbook.md) — worked patterns for common game features.
