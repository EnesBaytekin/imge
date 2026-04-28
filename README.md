# IMGE — 2D Game Engine in Go

IMGE is a 2D game engine being developed in Go with a component-based architecture and an event-driven communication system. It is in active development.

## Motivation

Go is fast to compile, easy to reason about, and produces single-binary outputs. IMGE brings these strengths to 2D game development by embedding Go as the scripting language — no bindings, no DSL, just Go.

## Architecture

- **Component-based**: Game objects are composed of reusable components (built-in `@Hitbox`, `@Movement`, or user-defined). Components are registered with a factory pattern.
- **Ping-Pong Event System**: Components communicate through a deferred event queue. `Ping()` emits an event, the EventManager queues it between frames, and subscribers receive it via `Pong()`.
- **Platform layer**: Rendering, input, and audio are abstracted behind a `Platform` interface. Currently SDL2 is the primary implementation, with a mock platform for headless testing.

## Features

- **Built-in components** (`@Hitbox`, `@Movement`) with collision detection
- **User components** — write any `.go` component, register it, use it in objects
- **JSON-defined scenes and objects** — compose your game world from `.scene` and `.obj` files
- **Ping-Pong event bus** — decoupled communication between components
- **Tag-based object queries** — find objects by tag at runtime
- **Depth-based rendering** — control draw order per object

## CLI Tool (`imge`)

The `imge` CLI tool creates the game binary:

```sh
imge init              # Scaffold a new project
imge build sdl         # Build for SDL2 (desktop)
imge build sdl --clean # Clean build
imge version           # Show engine version
```

The build pipeline generates a `go.mod` pointing to the engine's GitHub release, fetches the engine automatically, compiles the user's components together with the engine, and produces a single standalone binary.

## Project Structure

```
my-game/
├── game.json           # Game config (window size, title, FPS, initial scene)
├── components/         # User-defined Go components
├── scenes/             # Scene definitions (.scene)
├── objects/            # Object templates (.obj)
└── assets/             # Game assets
```

## Supported Platforms

| Platform | Status |
|----------|--------|
| SDL2     | Working — desktop builds (Linux, macOS, Windows) |
| Mock     | Headless testing |
| Web/WASM | Planned |
| Desktop  | Planned (native packaging) |

## Development Status

IMGE is in early development. While the core systems are functional, expect rough edges:

- **Editor**: A visual editor is planned to make scene/object editing more accessible.
- **Web builds**: WASM export will be added once the platform abstraction stabilizes.
- **Audio**: SDL audio device opens, but playback APIs are being built out.
- **Missing features**: Several convenience APIs and component types are still being implemented.

## License

MIT
