# Built-in Components (Faz 2)

IMGE ships a small library of **built-in components**. They are not special — they
use exactly the same `core.Component` interface as user components. At build time
they are merged into the project's single `components` package, so any component
(built-in or custom) can reach any other component's concrete methods directly.

## One component type, two jobs

There is a single `Component` interface. How a component *behaves* is a label, not
a type distinction:

- **Capability** — a component that mostly *exposes methods and data* for other
  components to drive (e.g. `@Mover`, `@Velocity`, `@Sprite`). It does little or
  nothing on its own each frame.
- **Behavior** — a component that *changes the scene on its own* each frame
  (e.g. `@Chase`, `@Gravity`, `@Spin`), typically by calling a capability's methods.

The distinction is documentation, not a second interface.

## Naming & kinds

Built-in types are named **without** a `Component` suffix (`Collider`, not
`ColliderComponent`). The kind identifier strips the suffix when present, so both
`Collider` and a hypothetical `FooComponent` map to `@Collider` / `@Foo`:

```go
func builtinKind(typeName string) string { return "@" + strings.TrimSuffix(typeName, "Component") }
```

User components keep their file path as kind (`components/player.go`).

## Accessing other components

A component reaches a sibling component (on the same owner object) by type and,
optionally, by name. There is **no** `core.Get(from Component)`; lookups take the
object directly and are deterministic (insertion order):

```go
collider := core.GetFrom[*Collider](owner)        // first Collider, insertion order
sprites  := core.GetAllFrom[*Sprite](owner)       // every Sprite
hud      := core.GetFromNamed[*Health](owner, "playerHealth") // by name
```

`GetFrom` returns the zero value (a `nil` pointer for pointer types) when nothing
matches — callers nil-check.

## Dependencies

A component may declare the kinds it needs by implementing the optional
`Dependable` interface. It is informational: the build tool reads it to warn when
an object uses a component without the components it declares it needs.

```go
func (a *Animator) Requires() []string { return []string{"@Sprite"} }
```

## The built-in set

### Capabilities

| Kind        | Type       | Purpose |
|-------------|------------|---------|
| `@Collider` | `Collider` | Rectangle shape, movement mode, tag-filtered overlap tracking |
| `@Mover`    | `Mover`    | Collision-aware movement (`Teleport`, `Move`, `MoveTowards`) |
| `@Velocity` | `Velocity` | Per-axis velocity state; integrates through `@Mover` |
| `@Gravity`  | `Gravity`  | Vector acceleration + max speed applied to `@Velocity` |
| `@Friction` | `Friction` | Per-axis linear velocity damping |
| `@Sprite`   | `Sprite`   | Draws a texture / texture region at the owner transform |
| `@Animator` | `Animator` | Named frame clips, drives `@Sprite`'s source rect |
| `@Sound`    | `Sound`    | One-shot sound or looping music |

### Behaviors

| Kind               | Type               | Purpose |
|--------------------|--------------------|---------|
| `@PlayerController`| `PlayerController` | WASD / arrow keys → `@Mover` |
| `@Chase`           | `Chase`            | Move toward the nearest tagged target |
| `@Follow`          | `Follow`           | Lerp toward a tagged target + offset |
| `@Patrol`          | `Patrol`           | Walk a list of waypoints (loop or ping-pong) |
| `@Wander`          | `Wander`           | Random direction, re-rolled on an interval |
| `@Bounce`          | `Bounce`           | Constant velocity that reflects off collisions |
| `@Spin`            | `Spin`             | Rotate the owner over time |
| `@TimedDespawn`    | `TimedDespawn`     | Destroy the owner after N seconds |
| `@Health`          | `Health`           | HP pool; emits `damaged` / `died` |
| `@Damage`          | `Damage`           | Applies damage to overlapping tagged targets |

## The collision model

`@Collider` is the shape. Its `mode` decides how movement resolves against it:

- **`solid`** — blocks movers outright (walls, ground).
- **`pushable`** — a mover pushes it; `pushFactor` scales how far it slides
  (`0` = immovable, `1` = full, default `1`).
- **`trigger`** — never blocks or gets pushed; it only *detects* overlaps.

`@Mover.Move` resolves each axis independently (slide-along-walls), so a diagonal
move that hits a wall still glides along the open axis. Push chains are allowed up
to a small recursion depth. When a move is blocked, `@Mover` emits
`blocked_collision` with the blocking object as data.

`collidesWith` is an optional list of object tags. Empty means "collide with
everything"; otherwise the collider only interacts with objects carrying one of
those tags (resolved in O(1) via the scene's tag index).

`@Collider` tracks overlaps each frame and emits `collision_enter` /
`collision_exit` (data = the other object), which is how "entered an area" /
"left an area" is detected. `GetOverlaps()` returns the current overlapping
objects.

## Events

Events are scene-global by name: `Emit(name, data)` is delivered to **every**
component that subscribed via `On(name, handler)` after all `Update` calls finish.
Context (who/what caused it) travels in `data` — e.g. `blocked_collision` carries
the blocking object, `damaged` carries the amount.

## An example

```json
{
  "kind": "@Collider", "name": "body", "args": { "width": 32, "height": 32, "mode": "pushable" }
},
{
  "kind": "@Mover", "name": "mover", "args": {}
}
```

```go
// custom controller: read input, move through the collider-aware @Mover
func (c *Player) Update(ctx *core.Context) {
	owner := c.GetOwner()
	dx := 0.0
	if ctx.Input.IsKeyPressed(core.KeyA) { dx = -1 }
	if ctx.Input.IsKeyPressed(core.KeyD) { dx = 1 }
	if m := core.GetFrom[*Mover](owner); m != nil {
		m.Move(dx*c.Speed*ctx.DeltaTime(), 0)
	}
}
```
