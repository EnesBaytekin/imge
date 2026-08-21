package components

import (
	"testing"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// testObject builds an object at (x, y) with the given components (each must
// already be named). It panics on error, keeping the tests terse.
func testObject(name string, x, y float64, comps ...core.Component) *core.Object {
	obj := core.NewObject(name)
	for _, c := range comps {
		if err := obj.AddComponent(c); err != nil {
			panic(err)
		}
	}
	obj.SetPosition(x, y)
	return obj
}

// testCollider returns a named collider of the given size and push factor.
func testCollider(name string, w, h, push float64) *Collider {
	c := &Collider{Width: w, Height: h, PushFactor: push}
	c.SetName(name)
	return c
}

// testMover returns a named mover.
func testMover(name string) *Mover {
	m := &Mover{}
	m.SetName(name)
	return m
}

// mustAdd adds obj to scene, panicking on error.
func mustAdd(scene *core.Scene, obj *core.Object) {
	if err := scene.AddObject(obj); err != nil {
		panic(err)
	}
}

// getMover returns the first @Mover on obj.
func getMover(obj *core.Object) *Mover { return core.GetFrom[*Mover](obj) }

func TestMoverBlockedBySolid(t *testing.T) {
	scene := core.NewScene("main")
	player := testObject("player", 0, 0,
		testCollider("body", 32, 32, 0),
		testMover("mover"),
	)
	wall := testObject("wall", 32, 0, testCollider("body", 32, 32, 0))
	mustAdd(scene, player)
	mustAdd(scene, wall)

	res := getMover(player).Move(10, 0)
	if res.X {
		t.Fatalf("expected X movement to be blocked")
	}
	if player.Transform.Position.X != 0 {
		t.Fatalf("player should not have moved, got x=%v", player.Transform.Position.X)
	}
}

// TestMoverBlockedByCompoundSolid is the DVD regression: an object carries several
// solid colliders, and only one of them is near the mover. The mover must test the
// whole compound body, not just the first collider.
func TestMoverBlockedByCompoundSolid(t *testing.T) {
	scene := core.NewScene("main")

	walls := core.NewObject("walls")
	c1 := testCollider("left", 10, 100, 0)
	c1.Offset = math.NewVector2(0, 0)
	c2 := testCollider("right", 10, 100, 0)
	c2.Offset = math.NewVector2(100, 0)
	if err := walls.AddComponent(c1); err != nil {
		t.Fatal(err)
	}
	if err := walls.AddComponent(c2); err != nil {
		t.Fatal(err)
	}
	walls.SetPosition(0, 0)
	mustAdd(scene, walls)

	player := testObject("player", 95, 0,
		testCollider("body", 10, 10, 0),
		testMover("mover"),
	)
	mustAdd(scene, player)

	res := getMover(player).Move(10, 0)
	if res.X {
		t.Fatalf("expected the second collider to block, got X applied")
	}
	if player.Transform.Position.X != 95 {
		t.Fatalf("player should not have moved, got x=%v", player.Transform.Position.X)
	}
}

func TestMoverPushesPushableTogether(t *testing.T) {
	scene := core.NewScene("main")
	crate := testObject("crate", 32, 0,
		testCollider("body", 32, 32, 1),
		testMover("mover"),
	)
	player := testObject("player", 0, 0,
		testCollider("body", 32, 32, 0),
		testMover("mover"),
	)
	mustAdd(scene, crate)
	mustAdd(scene, player)

	res := getMover(player).Move(10, 0)
	if !res.X {
		t.Fatalf("expected push to succeed")
	}
	// Both advance the same distance (weightless crate => full speed), staying flush.
	if player.Transform.Position.X != 10 || crate.Transform.Position.X != 42 {
		t.Fatalf("expected player=10 crate=42, got player=%v crate=%v",
			player.Transform.Position.X, crate.Transform.Position.X)
	}
}

func TestMoverPushSlowsBothByFactor(t *testing.T) {
	scene := core.NewScene("main")
	crate := testObject("crate", 32, 0,
		testCollider("body", 32, 32, 0.5),
		testMover("mover"),
	)
	player := testObject("player", 0, 0,
		testCollider("body", 32, 32, 0),
		testMover("mover"),
	)
	mustAdd(scene, crate)
	mustAdd(scene, player)

	getMover(player).Move(10, 0)
	// f = (1-0) * 0.5 = 0.5 => both move 5 of the requested 10.
	if player.Transform.Position.X != 5 || crate.Transform.Position.X != 37 {
		t.Fatalf("expected player=5 crate=37, got player=%v crate=%v",
			player.Transform.Position.X, crate.Transform.Position.X)
	}
}

func TestMoverBlockedByWallBehindPushable(t *testing.T) {
	scene := core.NewScene("main")
	wall := testObject("wall", 64, 0, testCollider("body", 32, 32, 0))
	crate := testObject("crate", 32, 0,
		testCollider("body", 32, 32, 1),
		testMover("mover"),
	)
	player := testObject("player", 0, 0,
		testCollider("body", 32, 32, 0),
		testMover("mover"),
	)
	mustAdd(scene, wall)
	mustAdd(scene, crate)
	mustAdd(scene, player)

	res := getMover(player).Move(10, 0)
	if res.X {
		t.Fatalf("expected the push to be blocked by the wall behind the crate")
	}
	if player.Transform.Position.X != 0 || crate.Transform.Position.X != 32 {
		t.Fatalf("neither should move: player=%v crate=%v",
			player.Transform.Position.X, crate.Transform.Position.X)
	}
}
