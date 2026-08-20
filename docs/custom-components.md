# Custom components

Built-in components cover rendering, movement, collision, and combat — but your
game's *rules* live in **custom components**: small Go files you write. They are
first-class citizens, identical to built-ins, and at build time they're merged into
the same package so any component can call any other's methods directly.

## The contract

A custom component file:

1. Lives anywhere in the project (convention: `components/`), named `*.go`, and
   declares `package components`.
2. Exports **exactly one** struct that embeds `core.BaseComponent`.
3. Implements whichever of `Initialize` / `Update` / `Draw` / `OnEnable` /
   `OnDisable` it needs (BaseComponent provides empty defaults).
4. Optionally implements `Requires() []string` to declare the kinds it depends on.

That single exported type is what the build tool registers, keyed by its **struct
type name**. You reference it in JSON by that name as its `kind`:

```json
{ "kind": "Enemy", "name": "brain", "args": { "speed": 190 } }
```

## A minimal component

```go
// components/enemy.go
package components

import "github.com/EnesBaytekin/imge/core"

type Enemy struct {
    core.BaseComponent
    Speed float64 `json:"speed"` // export variable
}

func (c *Enemy) Initialize() {
    if c.Speed <= 0 { c.Speed = 60 }
}

func (c *Enemy) Update(ctx *core.Context) {
    // logic here, using ctx.DeltaTime(), ctx.Input, c.GetScene(), c.GetOwner(), ...
}
```

Key points:

- **Export variables** — exported, JSON-tagged fields are populated from the
  component's `args` in `.obj`/`.scene` files. The JSON key must match the `json`
  tag **exactly** (snake_case is the engine convention; see
  [the JSON rule](json-format.md#json-format-reference)). Unexported fields stay
  private state.
- **`Initialize()`** runs once, after the object is in a fully-loaded scene — so
  `c.GetScene()` works and you can look up sibling components here.
- **`Update(ctx)`** runs every frame.
- **`Draw(renderer)`** runs every frame, for custom rendering.

## Reaching sibling components

The engine is deliberately flat: components on the same object call each other's
concrete methods directly. From a component:

```go
owner := c.GetOwner()

velocity := core.GetFrom[*Velocity](owner)          // first of type (insertion order)
sprites   := core.GetAllFrom[*Sprite](owner)        // all of type
run       := core.GetFromNamed[*Sprite](owner, "run") // the one named "run"
```

`core.GetFrom` / `GetFromNamed` return a nil pointer when nothing matches — always
nil-check. `GetFromNamed` is O(1) and what you want when several components share a
type (e.g. several `@Sprite`s).

## Declaring dependencies

```go
func (c *Enemy) Requires() []string {
    return []string{"@Velocity", "@Mover", "@Collider"}
}
```

`Requires` is **informational**: the build tool warns when an object uses a
component without also giving it the components it declares it needs. It doesn't
auto-add anything — you still fetch siblings with `core.GetFrom` and nil-check.

## Events

```go
func (c *Enemy) Initialize() {
    c.On("damaged", func(data any) { /* react */ })
}
func (c *Enemy) Update(ctx *core.Context) {
    c.Emit("scream", nil)
}
```

See [Events](events.md) for the full model (queueing, timing, `Source`).

## Scene and game access

```go
func (c *Enemy) Update(ctx *core.Context) {
    scene := c.GetScene()                  // the scene containing the owner (nil before added)
    scene.FindObjectsWithTag("player")  // query the world
    ctx.Game.SwitchScene("game_over")   // switch scenes
    ctx.Input.IsKeyJustPressed(core.KeyE)
    ctx.Audio.PlaySound("assets/hit.wav", 0.8, 1.0)
}
```

The `Context` passed to `Update` bundles `Input`, `Audio`, `Time` (plus `Scene` and
`Game`), with a `DeltaTime()` helper. See the interfaces in
[the engine source](../core/interfaces.go) for the full surface.

## Renderer drawing

`Draw` receives a `core.Renderer` with shape and texture primitives (all in world
coordinates, under the camera):

```go
func (c *Enemy) Draw(r core.Renderer) {
    r.DrawRect(math.NewRect(0, 0, 32, 32), math.Red)
    r.DrawCircle(c.GetOwner().GetPosition(), 8, math.Green)
    r.DrawLine(math.NewVector2(0,0), math.NewVector2(10,10), math.White, 2)
}
```

Methods: `DrawRect`, `DrawRectOutline`, `DrawCircle`, `DrawCircleOutline`,
`DrawLine`, `DrawTexture`, `GetTextureSize`, `GetViewportSize`. Shapes are drawn
without anti-aliasing (crisp pixel art).

## Component draw order within an object

Every component inherits a `draw_layer` field (from `BaseComponent`). Lower layers
draw first; equal layers keep insertion order. Set it via the `draw_layer` JSON arg
or the `DrawLayer` field. Useful when one object needs several draw calls in a
specific order (e.g. a shadow under a sprite).

## Naming and the one-type rule

- One component **type** per file — the build tool errors if a file exports more
  than one `core.BaseComponent`-embedding struct (or none).
- Two component files must not share a type name: all components are merged into
  one `components` package, so type names are already unique by Go's rules. Where
  the file lives doesn't matter to the `kind` (a component can be moved without
  touching its JSON).
- The component's `kind` is its struct type name, so the `name` field in JSON is
  the instance name (unique within the object) — the same component type can be
  instantiated several times under different names.

## Build-time validation

`imge build` checks, before compiling:

- every `kind` referenced in your `.obj`/`.scene` files is registered (typos in a
  kind are caught here, as an error);
- no duplicate kinds or duplicate type names;
- `Requires()` dependencies are present on the same object (warnings, not errors).

## A realistic example

A crate that breaks when destroyed, spawning a coin:

```go
// components/crate.go
package components

import "github.com/EnesBaytekin/imge/core"

type Crate struct {
    core.BaseComponent
}

func (c *Crate) Requires() []string { return []string{"@Health", "@Collider", "@Sprite"} }

func (c *Crate) Initialize() {
    c.On("died", func(data any) {
        obj, ok := data.(*core.Object)
        if !ok || obj != c.GetOwner() { return } // ignore other objects' deaths
        c.spawnCoin()
    })
}

func (c *Crate) spawnCoin() {
    scene := c.GetScene()
    if scene == nil { return }
    scene.InstantiateFromTemplate("objects/coin.obj",
        &math.Transform{ Position: c.GetOwner().GetPosition() })
    c.GetOwner().Destroy()
}
```

Wire it up in JSON:

```json
{
  "name": "crate",
  "tags": ["crate"],
  "components": [
    { "kind": "@Sprite", "name": "sprite", "args": { "texture": "assets/crate.png", "width": 24, "height": 24 } },
    { "kind": "@Collider", "name": "hitbox", "args": { "width": 24, "height": 24, "mode": "pushable" } },
    { "kind": "@Health", "name": "health", "args": { "max": 1 } },
    { "kind": "Crate", "name": "crate", "args": {} }
  ]
}
```

## Next

- [Components](components.md) — the built-in library you'll compose with.
- [Events](events.md) — the event contract used above.
- [Cookbook](cookbook.md) — more worked components.
