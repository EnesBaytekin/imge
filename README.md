# IMGE — Minimal 2D Game Engine in Go

**IMGE** (Minimal Game Engine) is a 2D pixel-art game engine. You describe your game with
JSON files (scenes, objects) and small Go "components", then `imge build` compiles it into a
single self-contained executable — or a web (WASM) bundle. It's built on
[Ebitengine](https://ebitengine.org/), a pure-Go 2D game library, so the engine itself is Go
all the way down — no C/C++ engine to link against.

## What you need

- **Go 1.24+** — `imge` uses the Go toolchain to compile your game. Get it from
  [go.dev/dl](https://go.dev/dl/).
- **The `imge` CLI** — download the binary for your OS/architecture from the
  [latest release](https://github.com/EnesBaytekin/imge/releases) and put it on your `PATH`.

Release assets are named `imge_<os>_<arch>` (`.exe` on Windows):

| File | Platform |
| --- | --- |
| `imge_linux_amd64` / `imge_linux_arm64` | Linux (Intel/AMD 64-bit / ARM 64-bit) |
| `imge_windows_amd64.exe` / `imge_windows_arm64.exe` | Windows (64-bit / ARM 64-bit) |
| `imge_darwin_amd64` / `imge_darwin_arm64` | macOS (Intel / Apple Silicon) |

> The first build needs internet once: `imge` fetches Ebitengine from the Go module proxy.
> The engine source is embedded inside the `imge` binary, so you don't fetch it separately.

## Quick start

```sh
mkdir mygame && cd mygame
imge init      # scaffold a project (only works in an empty directory)
imge run       # build and launch — move with WASD, enemies chase you
```

`imge init` creates a project like this:

```
mygame/
├── game.json      # window title/size, FPS, initial scene
├── components/    # your Go components (any nesting depth)
├── scenes/        # scene definitions (.scene)
├── objects/       # object templates (.obj)
└── assets/        # images and sounds
```

## How a game is made

A game is a set of **objects** placed into **scenes**. Each object is a list of
**components**. Built-in components come with the engine; user components are small Go files
you write in `components/`.

- **Objects** (`objects/*.obj`) — JSON: a name, depth, tags, and a list of components.
- **Scenes** (`scenes/*.scene`) — JSON: a background color and the objects to place.
- **Components** (`components/*.go`) — Go structs with `Update`/`Draw` methods, registered in
  `init()`. Built-ins: `@Hitbox`, `@Movement`, `@Image`, `@Sound`.
- **Assets** (`assets/`) — PNG/JPEG images and WAV/MP3/OGG sounds, embedded into the build.

Example — give an object a sprite and a hitbox:

```json
{ "kind": "@Image",  "name": "sprite", "args": { "texture": "player.png", "width": 32, "height": 32 } },
{ "kind": "@Hitbox", "name": "hitbox", "args": { "width": 32, "height": 32 } }
```

## Building

```sh
imge build                    # native build for your machine
imge build --windows          # Windows, amd64 + arm64
imge build --amd64            # amd64 for every OS
imge build --windows --amd64  # Windows amd64 only
imge build --web              # web (WASM) bundle
imge build --windows --web    # Windows (both archs) + web
imge build --all              # every buildable target (skips what it can't)
```

Flags: `--linux --windows --macos` pick the OS, `--amd64 --arm64` pick the architecture
(omit either to target all of them), `--web` builds the web bundle, `--all` builds everything
it can.

Output goes to `imge_build/`:

- **Desktop**: `imge_build/<name>_<os>-<arch>` (`.exe` on Windows) — a single self-contained
  executable; copy it anywhere and run it.
- **Web**: `imge_build/web/` — serve it locally:

```sh
cd imge_build/web
python3 -m http.server 8000   # then open http://localhost:8000/
```

### Cross-compilation

Windows (amd64/arm64) builds from any host (pure Go). macOS and non-native Linux targets need
Ebitengine's Cgo (GLFW), so build those natively or via CI — `imge build` prints which
targets it skipped and why.

## License

MIT
