# The `imge` tool

`imge` is the command-line tool that scaffolds, builds, and runs your game. Run it
from your project directory (the directory containing `game.imge`).

## Commands

| Command | What it does |
|---|---|
| `imge init` | Scaffold a blank project (empty directory only) |
| `imge init sample` | Scaffold the sample platformer demo (`sample` / `demo` / `example` all work) |
| `imge build [flags]` | Build the game |
| `imge run` | Build and launch the desktop game |
| `imge version` | Print the engine version |
| `imge help` | Print usage (also `-h` / `--help`) |

`imge run` builds a native desktop binary and executes it immediately.

## `imge init`

- **Refuses to run in a non-empty directory** — it prints a warning and exits,
  rather than risk overwriting your files. Initialize in an empty folder.
- `imge init` → blank project. `imge init sample` → the playable platformer demo
  (a good reference for custom components and the JSON formats).

## `imge build`

With no flags, builds a native desktop executable for the current machine.

### Flags

| Flag | Meaning |
|---|---|
| `--linux` / `--windows` / `--macos` | target OS (omit all to target every OS when combined with `--arch`) |
| `--amd64` / `--arm64` | target architecture (omit all to build both) |
| `--web` | build the web (WASM) bundle |
| `--all` | build every supported target (skips what it can't) |

You can also pass the positional `web` / `wasm` after `build` (e.g. `imge build web`)
as shorthand for `--web`.

### Target resolution

| Invocation | Result |
|---|---|
| `imge build` | native desktop for this machine |
| `imge build --windows` | Windows, amd64 + arm64 |
| `imge build --windows --amd64` | Windows amd64 only |
| `imge build --amd64` | amd64 for every OS (linux, macos, windows) |
| `imge build --web` (or `imge build web`) | web bundle only |
| `imge build --windows --web` | Windows (both archs) + web |
| `imge build --all` | every buildable target (skips what it can't) |

### Cross-compilation support

| Target | From anywhere? | Notes |
|---|---|---|
| native (your OS/arch) | ✓ | — |
| `windows/amd64`, `windows/arm64` | ✓ | pure Go, no C toolchain needed |
| `darwin/…` | ✗ | Ebitengine uses Cgo (GLFW/Metal) — build on a Mac or CI |
| non-native Linux (e.g. `linux/arm64` from amd64) | ✗ | needs a C cross toolchain + sysroot |
| `web` | ✓ | pure Go → WASM |

Unsupported targets are skipped with a clear message rather than failing the whole
run. A native Linux build requires the X11/GL/ALSA headers listed in
[Getting started](getting-started.md#prerequisites).

### Output

- **Desktop** — `imge_build/<name>_<os>-<arch>` (`.exe` on Windows). `<name>` is the
  `game.imge` `name`, slugified to lowercase ASCII (`My Game` → `my-game`). The
  executable is self-contained — copy it anywhere.
- **Web** — `imge_build/web/` containing `index.html`, `game.wasm`, and
  `wasm_exec.js`. Serve it over HTTP (a browser won't load WASM from `file://`):

  ```sh
  cd imge_build/web
  python3 -m http.server 8000   # open http://localhost:8000/
  ```

## What the build does

1. **Analyzes** the project — walks the whole root once, classifying `.go`
   components, `.scene` files, `.obj` templates, and other data files, wherever
   they live.
2. **Generates** a self-contained Go module in a temp directory: it embeds the
   pure-Go engine source, merges your components with the built-ins into one
   `components` package, and auto-generates `components/registry.go` (which
   registers every component — you never write an `init()` yourself).
3. **Validates** — errors on unknown component kinds, duplicate kinds/type names,
   and warns when an object uses a component without the components it declares it
   needs via `Requires()`.
4. **Compiles** with `go build`, then copies the artifact into `imge_build/`.

For desktop, the whole project tree is embedded and extracted to a temp directory
at startup (the game runs from there); for web, it's read directly from the
embedded filesystem. Either way, the root-relative paths inside your JSON files
resolve exactly as they do during development.

## Next

- [Project structure](project-structure.md) — what the tool reads.
- [Custom components](custom-components.md) — how the auto-registration works.
