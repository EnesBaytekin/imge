# IMGE Game

Your new IMGE game. This is a blank starting point — a single empty scene and
the standard project layout, ready for you to drop in objects and components.

## Project layout

```
game.json             window / FPS / initial scene
scenes/main.scene     the starting scene (add objects here)
objects/              reusable object templates (*.obj)
components/           custom component scripts (Go)
assets/               images (PNG) and sounds (WAV)
```

## Adding an object

Objects are placed in `scenes/*.scene` (inline) or as reusable `objects/*.obj`
templates referenced by `"file"`. Here's an inline object that draws a 32x32
sprite and gives it a solid collider:

```json
{
  "name": "box",
  "tags": ["platform"],
  "transform": { "position": { "x": 100, "y": 300 } },
  "components": [
    { "kind": "@Sprite",   "args": { "texture": "box.png", "width": 32, "height": 32 } },
    { "kind": "@Collider", "args": { "width": 32, "height": 32, "mode": "solid" } }
  ]
}
```

Built-in components are named `@Name` (`@Sprite`, `@Collider`, `@Mover`,
`@Velocity`, `@Gravity`, `@Friction`, `@Animator`, `@Health`, `@Damage`,
`@Spin`, `@Chase`, `@Patrol`, `@Wander`, `@Bounce`, `@Follow`, `@Sound`, …).
A component in `components/` gets a `kind` equal to its project-relative path
(e.g. `components/player.go`).

## Writing a custom component

Create `components/foo.go` with one exported struct that embeds
`core.BaseComponent`. Exported JSON-tagged fields become "export variables"
you can set from an object's `args`:

```go
package components

import "github.com/EnesBaytekin/imge/core"

type FooComponent struct {
	core.BaseComponent
	Speed float64 `json:"speed"`
}

func (c *FooComponent) Initialize() {
	if c.Speed <= 0 {
		c.Speed = 100
	}
}

func (c *FooComponent) Update(ctx *core.Context) {
	// runs every frame
}
```

## Build & run

```sh
imge run           # build and launch natively
imge build         # native binary  → imge_build/<name>_<os>-<arch>
imge build --web   # WASM bundle    → imge_build/web/
```
