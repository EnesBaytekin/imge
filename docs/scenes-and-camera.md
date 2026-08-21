# Scenes & camera

A **scene** is a named collection of objects that update and draw together. Only
one scene is **active** at a time; the game switches between them.

## Scene anatomy

A `Scene` holds:

- `Objects` — the objects, keyed by ID.
- `Name` — its identifier (from the `.scene` file's `name` field).
- `BackgroundColor` — the clear color each frame (from `background_color`).
- `Camera` — the optional viewport (nil = world coords are screen coords).
- `Active` — whether it updates/draws.
- `EventManager` — its own event queue and subscriptions (events are scene-scoped).

## Loading scenes

At startup, the generated entrypoint walks the project root and loads **every**
`*.scene` file it finds (at any depth), registering each as a scene. The initial
scene is whatever `game.initial_scene` names (or `"main"` by default). The scene's
registered name comes from the file's `name` field; if absent, the loader uses the
filename without the `.scene` suffix.

> **`initial_scene` must match a scene's `name`** (or filename). If it doesn't, no
> scene matches and the game starts with no active scene (a black screen). If a
> scene file fails to load (e.g. a bad `file` reference inside it), the loader logs
> a warning and **skips that scene** — so `SwitchScene` to it will return `false`.

## The per-frame update order

`Scene.Update` runs, in order:

1. `frame++` — increments the scene's frame counter.
2. `ctx.Scene = s` — so components can reach their scene via `c.GetScene()`.
3. For each object: `initializeComponents()` (first time only), then `Update(ctx)`.
4. `Camera.Tick()` — advance the camera toward its follow target (after objects moved).
5. `EventManager.Process()` — deliver every queued event.
6. `removeDestroyedObjects()` — actually drop objects marked `Destroy()`ed.

Because events are delivered **after** all updates, a component that emits an event
in its `Update` can't observe the handlers running until the same frame's end —
and a component that reacts to an event sees the world as it was after all updates.

## The per-frame draw order

`Scene.Draw`:

1. Re-sorts objects by depth if any changed.
2. Applies the camera transform (if a camera exists).
3. Draws **world** objects (`UI == false`) in depth order.
4. Resets the camera (raw screen space).
5. Draws **UI** objects (`UI == true`) in depth order, on top.

## Scene switching

Two ways to change the active scene:

| API | Timing | Notes |
|---|---|---|
| `game.SetActiveScene(name)` | immediate | returns `false` if the scene doesn't exist |
| `game.SwitchScene(name)` | deferred | **recommended** — queues the switch to apply at the start of the next `Game.Update` |

Always prefer `SwitchScene` when switching from inside a component's `Update`:
deferring avoids drawing the new scene before its objects' `Initialize()` has run
(which is exactly the crash that an immediate switch from mid-update would cause).

From inside a component, reach the game via `ctx.Game`:

```go
func (c *Menu) Update(ctx *core.Context) {
    if ctx.Input.IsKeyJustPressed(core.KeyEnter) {
        ctx.Game.SwitchScene("game")   // queued; applied next frame
    }
}
```

`SwitchScene` (and `SetActiveScene`) return `false` when the named scene isn't
registered — check that value if a switch seems to silently do nothing (a common
cause is the target `.scene` failing to load, or a `name` mismatch).

The active scene's objects are **not** reset between visits — switching away and
back preserves object state. If you need a fresh scene each time, reset objects in
a component's `Initialize` (which runs again on the newly-added object) or manage
the reset yourself.

## The camera

A scene's `Camera` (optional) defines the viewport: what part of the world is
visible. It's core-level state on the scene, not an object or component.

| Field | Meaning |
|---|---|
| `X` / `Y` | the view's **top-left corner**, in world coordinates |
| `Zoom` | scale factor, anchored at the viewport center (1 = 1:1) |
| `Smoothing` | how much the camera eases toward its target each frame (0 = snap; 0.1 = smooth trail) |
| `LockX` / `LockY` | freeze that axis (e.g. a side-scroller locks Y) |

Methods (drive these from a custom component):

| Method | What it does |
|---|---|
| `Follow(obj)` | track an object's position |
| `FollowPoint(x, y)` | track a fixed world point |
| `LookAt(x, y)` | center on a point immediately, stop following |
| `StopFollow()` | stop following, stay where it is |
| `WorldToScreen(p)` / `ScreenToWorld(p)` | coordinate conversion (mouse ↔ world) |
| `Tick()` | called automatically each frame by the scene |

`Smoothing` semantics: `0 < smoothing < 1` lerps toward the target (`X += (tx - X)
* smoothing`); any other value snaps. `LockX`/`LockY` suppress movement on that
axis regardless of smoothing.

A scene with **no camera** (`camera` omitted, or `scene.Camera == nil`) draws with
world = screen: object position equals pixel position. A camera at `(0, 0, 1)` is
the same — the world origin `(0, 0)` sits at the **top-left** of the screen.

### Example: follow the player

```go
func (c *Director) Initialize() {
    s := c.GetScene()
    if s == nil || s.Camera == nil { return }
    if players := s.FindObjectsWithTag("player"); len(players) > 0 {
        s.Camera.Follow(players[0])
    }
}
```

## Querying a scene

From a component, `c.GetScene()` returns the scene (nil before the object is added):

- `scene.FindObjectsWithTag(tag)` — all objects with a tag.
- `scene.GetObjectByName(name)` — by unique name.
- `scene.GetObjectByID(id)` — by ID.
- `scene.FrameNumber()` — the current frame count (1-based), used by
  `@StateMachine.JustEntered()` to detect "this frame".

## Next

- [Objects](objects.md) — the objects a scene contains.
- [Events](events.md) — how components talk within a scene.
- [Camera in the JSON reference](json-format.md#scene-files-scene) — configuring it declaratively.
