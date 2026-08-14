# IMGE — 2D Game Engine in Go

**I**MGE is **M**inimal **G**ame **E**ngine. A 2D pixel-art game engine in Go with a component-based architecture and an event-driven communication system, built on [Ebitengine](https://ebitengine.org/) (pure Go — no SDL, no CGO toolchain to manage).

## Architecture

- **Component-based**: Game objects are composed of reusable components (built-in `@Hitbox`, `@Movement`, or user-defined). Components are registered with a factory pattern.
- **Ping-Pong Event System**: Components communicate through a deferred event queue. `Ping()` emits an event, the EventManager queues it between frames, and subscribers receive it via `Pong()`.
- **Platform layer**: Rendering, input, and audio are abstracted behind a `Platform` interface. The Ebitengine implementation drives the game loop through a runner adapter, leaving the platform-agnostic core untouched.

## Features

- **Built-in components** (`@Hitbox`, `@Movement`) with collision detection
- **User components** — write any `.go` component, register it, use it in objects
- **JSON-defined scenes and objects** — compose your game world from `.scene` and `.obj` files
- **Ping-Pong event bus** — decoupled communication between components
- **Tag-based object queries** — find objects by tag at runtime
- **Depth-based rendering** — control draw order per object

## Installing the CLI

Download the `imge` binary for your platform from the [latest release](https://github.com/EnesBaytekin/imge/releases), or build it from source:

```sh
git clone https://github.com/EnesBaytekin/imge
cd imge
go build -o imge ./cmd/imge
```

Requirements:

- **Go 1.24+** — `imge` compiles the game by invoking the Go toolchain.
- **Internet** on the first build — game dependencies (Ebitengine) are fetched from the Go module proxy. The engine source itself is embedded in the `imge` binary, so no GitHub fetch of the engine is needed.

## CLI Tool (`imge`)

```sh
imge init              # Scaffold a new project in the current directory
imge build             # Build a native executable (default: desktop)
imge build web         # Build a WebAssembly bundle into web/
imge run               # Build and run the game locally
imge version           # Show engine version
```

`imge build` (target `desktop`) produces a **single self-contained executable** — `game` (or `game.exe` on Windows). Assets, scenes, and objects are embedded into the binary, so you can copy it anywhere and run it standalone.

`imge build web` produces a `web/` directory (`game.wasm` + `index.html` + `wasm_exec.js`). Serve it locally with:

```sh
cd web
python3 -m http.server 8000
```

then open <http://localhost:8000/> in a browser.

## Project Structure

```
my-game/
├── game.json           # Game config (window size, title, FPS, initial scene)
├── components/         # User-defined Go components
├── scenes/             # Scene definitions (.scene)
├── objects/            # Object templates (.obj)
└── assets/             # Game assets (images, sounds)
```

## Quick Start

```sh
mkdir mygame && cd mygame
imge init
imge run        # opens a window — move with WASD, enemies chase you
```

## Development Status

IMGE is in early development. While the core systems are functional, expect rough edges:

- **Editor**: A visual editor is planned to make scene/object editing more accessible.
- **Web assets**: Web builds run scenes/objects/components, but texture/audio asset loading on the web target is still being wired up.
- **Missing features**: Several convenience APIs and component types are still being implemented.

## License

MIT
