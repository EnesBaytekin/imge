package core

import (
	"testing"

	"github.com/EnesBaytekin/imge/core/math"
)

// pickTestComponent reports a configurable DebugBounds for hit-testing. It does not
// implement DrawDebug — picking only needs DebugBoundsProvider.
type pickTestComponent struct {
	BaseComponent
	rect math.Rect
}

func (c *pickTestComponent) DebugBounds() math.Rect { return c.rect }

// newPickObject adds an object to the scene holding one pickTestComponent with the
// given bounds. depth is set before AddObject so the scene's draw-order sort is
// deterministic (higher depth draws on top).
func newPickObject(scene *Scene, name string, rect math.Rect, depth float64) *Object {
	obj := NewObject(name)
	obj.Depth = depth
	comp := &pickTestComponent{rect: rect}
	comp.SetName(name + "_pick")
	_ = obj.AddComponent(comp)
	_ = scene.AddObject(obj)
	return obj
}

func TestPickReturnsTopmostHit(t *testing.T) {
	scene := NewScene("t")
	back := newPickObject(scene, "back", math.NewRect(0, 0, 100, 100), 0)
	front := newPickObject(scene, "front", math.NewRect(20, 20, 40, 40), 1)

	// A point inside both picks the topmost object (front).
	if got := scene.Pick(math.NewVector2(30, 30)); got == nil || got.GetOwner() != front {
		t.Fatalf("Pick(30,30) = %v, want owner %q", got, "front")
	}

	// A point only in back picks back.
	if got := scene.Pick(math.NewVector2(5, 5)); got == nil || got.GetOwner() != back {
		t.Fatalf("Pick(5,5) = %v, want owner %q", got, "back")
	}

	// A point outside everything returns nil.
	if got := scene.Pick(math.NewVector2(200, 200)); got != nil {
		t.Fatalf("Pick(200,200) = %v, want nil", got)
	}
}

func TestPickSkipsUIObjects(t *testing.T) {
	scene := NewScene("t")
	world := newPickObject(scene, "world", math.NewRect(0, 0, 100, 100), 0)

	ui := newPickObject(scene, "ui", math.NewRect(0, 0, 100, 100), 10)
	ui.UI = true // drawn on top, but UI must be excluded from world picking

	if got := scene.Pick(math.NewVector2(50, 50)); got == nil || got.GetOwner() != world {
		t.Fatalf("Pick(50,50) = %v, want world object (UI skipped)", got)
	}
}

func TestPickSkipsInactiveAndDestroyed(t *testing.T) {
	scene := NewScene("t")
	inactive := newPickObject(scene, "inactive", math.NewRect(0, 0, 50, 50), 0)
	inactive.Active = false

	destroyed := newPickObject(scene, "destroyed", math.NewRect(60, 60, 50, 50), 0)
	destroyed.Destroy()

	if got := scene.Pick(math.NewVector2(25, 25)); got != nil {
		t.Fatalf("Pick on inactive object = %v, want nil", got)
	}
	if got := scene.Pick(math.NewVector2(85, 85)); got != nil {
		t.Fatalf("Pick on destroyed object = %v, want nil", got)
	}
}
