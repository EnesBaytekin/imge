package components

import (
	stdmath "math"
	"testing"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// axisAlignedQuad returns the four corners of an axis-aligned rect as a quad.
func axisAlignedQuad(x, y, w, h float64) [4]math.Vector2 {
	return [4]math.Vector2{
		math.NewVector2(x, y),
		math.NewVector2(x+w, y),
		math.NewVector2(x+w, y+h),
		math.NewVector2(x, y+h),
	}
}

// quadAABB returns the axis-aligned bounding box of a quad.
func quadAABB(q [4]math.Vector2) math.Rect {
	minX, minY := q[0].X, q[0].Y
	maxX, maxY := q[0].X, q[0].Y
	for _, p := range q[1:] {
		minX = mathMin(minX, p.X)
		minY = mathMin(minY, p.Y)
		maxX = mathMax(maxX, p.X)
		maxY = mathMax(maxY, p.Y)
	}
	return math.NewRect(minX, minY, maxX-minX, maxY-minY)
}

func mathMin(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func mathMax(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// TestOBBContactMatchesAABBForAxisAligned verifies the OBB swept solver gives the same
// first-contact distance as the trusted axis-aligned solver for unrotated shapes, across
// directions and both movement axes. This is the invariant that keeps the new rotated path
// from regressing the common case.
func TestOBBContactMatchesAABBForAxisAligned(t *testing.T) {
	cases := []struct {
		axis    int
		dir     float64
		mover   [4]math.Vector2
		obst    [4]math.Vector2
		maxDist float64
	}{
		// Moving +X toward a wall.
		{0, 1, axisAlignedQuad(0, 0, 10, 10), axisAlignedQuad(20, 0, 10, 10), 50},
		// Moving +X but the wall is out of reach.
		{0, 1, axisAlignedQuad(0, 0, 10, 10), axisAlignedQuad(20, 0, 10, 10), 5},
		// Moving -X toward a wall to the left.
		{0, -1, axisAlignedQuad(30, 0, 10, 10), axisAlignedQuad(0, 0, 10, 10), 50},
		// Moving +Y toward a platform below.
		{1, 1, axisAlignedQuad(0, 0, 10, 10), axisAlignedQuad(0, 20, 10, 10), 50},
		// Moving -Y toward a ceiling above.
		{1, -1, axisAlignedQuad(0, 30, 10, 10), axisAlignedQuad(0, 0, 10, 10), 50},
		// Overlapping already.
		{0, 1, axisAlignedQuad(0, 0, 10, 10), axisAlignedQuad(5, 0, 10, 10), 50},
		// Tunneling: a thin obstacle with a large step must stop at first contact.
		{0, 1, axisAlignedQuad(0, 0, 10, 10), axisAlignedQuad(20, 0, 5, 10), 100},
		// No perpendicular overlap: obstacle off to the side.
		{0, 1, axisAlignedQuad(0, 0, 10, 10), axisAlignedQuad(20, 100, 10, 10), 50},
	}

	for i, tc := range cases {
		mr := quadAABB(tc.mover)
		or := quadAABB(tc.obst)
		want := contactAlong(tc.axis, tc.dir, mr, or, tc.maxDist)
		got := obbContactAlong(tc.axis, tc.dir, tc.mover, tc.obst, tc.maxDist)
		// The OBB solver stops overlapEpsilon short of contact to avoid leaving the
		// mover a hair inside an obstacle, so it may be slightly *under* the exact
		// axis-aligned result but never further than the tolerance above it.
		if got > want+1e-9 || got < want-overlapEpsilon-1e-9 {
			t.Fatalf("case %d: obbContactAlong = %v, contactAlong = %v", i, got, want)
		}
	}
}

// TestQuadOverlapRotatedVsAABB verifies that a rotated quad only overlaps when its true
// shape overlaps, not merely when its enclosing AABB overlaps the other shape's AABB.
func TestQuadOverlapRotatedVsAABB(t *testing.T) {
	// A unit square rotated 45 degrees about the origin becomes a diamond:
	// (0,0), (√2/2, √2/2), (0, √2), (-√2/2, √2/2).
	s := stdmath.Sqrt(2) / 2
	diamond := [4]math.Vector2{
		math.NewVector2(0, 0),
		math.NewVector2(s, s),
		math.NewVector2(0, 2*s),
		math.NewVector2(-s, s),
	}

	// A small square high and to the right of the diamond's apex: inside the diamond's
	// AABB ([-s, s] x [0, 2s]) but outside the actual diamond.
	outside := axisAlignedQuad(0.4, 1.3, 0.2, 0.2)

	if quadOverlap(outside, diamond) {
		t.Fatalf("mover outside the diamond should not overlap")
	}
	if !quadAABB(outside).Overlaps(quadAABB(diamond)) {
		t.Fatalf("test setup: mover AABB should overlap the diamond AABB")
	}

	// A small square inside the diamond must overlap.
	inside := axisAlignedQuad(0.4, 0.7, 0.2, 0.2)
	if !quadOverlap(inside, diamond) {
		t.Fatalf("mover inside the diamond should overlap")
	}
}

// TestScenePickRespectsRotation verifies the editor's click-selection (Scene.Pick) uses the
// rotated shape, not the enclosing AABB.
func TestScenePickRespectsRotation(t *testing.T) {
	scene := core.NewScene("main")
	obj := core.NewObject("rotated")
	c := &Collider{Width: 32, Height: 32}
	c.SetName("body")
	if err := obj.AddComponent(c); err != nil {
		t.Fatal(err)
	}
	obj.SetPosition(0, 0)
	obj.Transform.Rotation = stdmath.Pi / 4 // 45° CCW about the origin
	mustAdd(scene, obj)

	// Inside the enclosing AABB but outside the rotated diamond: must not select.
	if got := scene.Pick(math.NewVector2(20, 40)); got != nil {
		t.Fatalf("point outside the rotated quad should not pick, got %v", got)
	}
	// Inside the rotated diamond: must select the collider.
	if got := scene.Pick(math.NewVector2(0, 20)); got == nil {
		t.Fatalf("point inside the rotated quad should pick the collider")
	}
}

// TestColliderContainsPointRotated verifies the hit-test follows the rotated quad, not the
// enclosing AABB.
func TestColliderContainsPointRotated(t *testing.T) {
	obj := core.NewObject("rotated")
	c := &Collider{Width: 32, Height: 32}
	c.SetName("body")
	if err := obj.AddComponent(c); err != nil {
		t.Fatal(err)
	}
	obj.SetPosition(0, 0)
	obj.Transform.Rotation = stdmath.Pi / 2 // rotate 90 deg CCW about the origin

	// World quad after a 90° rotation about the origin: [-32,0] x [0,32].
	// (16,16) is the unrotated center; rotated it is outside the new quad.
	if c.ContainsPoint(math.NewVector2(16, 16)) {
		t.Fatalf("(16,16) should be outside the 90°-rotated quad")
	}
	// (0,16) is inside the rotated quad.
	if !c.ContainsPoint(math.NewVector2(-16, 16)) {
		t.Fatalf("(-16,16) should be inside the 90°-rotated quad")
	}
}

// TestDiagonalIntoAngledWallDoesNotWedge reproduces the bug where a mover walking
// diagonally into a rotated (OBB) wall could end a hair inside it and then be unable to
// move out. The mover must stop without overlapping and must be able to retreat.
func TestDiagonalIntoAngledWallDoesNotWedge(t *testing.T) {
	// Directions approaching the wall from several sides, matching the normalized
	// diagonal input a PlayerController produces.
	dirs := []math.Vector2{
		{X: 1, Y: -1}, {X: 1, Y: 1}, {X: -1, Y: -1}, {X: -1, Y: 1}, {X: 1, Y: 0}, {X: 0, Y: 1},
	}
	for _, d := range dirs {
		d = d.Normalize()
		scene := core.NewScene("main")
		wall := testObject("wall", 200, 200, testCollider("body", 64, 64, 0))
		wall.Transform.Rotation = stdmath.Pi / 4 // 45° diamond
		mustAdd(scene, wall)

		start := math.NewVector2(200-d.X*150, 200-d.Y*150)
		player := testObject("player", start.X, start.Y, testCollider("body", 32, 32, 0), testMover("mover"))
		mustAdd(scene, player)

		m := getMover(player)
		wc := core.GetFrom[*Collider](wall)
		pc := core.GetFrom[*Collider](player)

		for i := 0; i < 400; i++ {
			m.Move(d.X*2, d.Y*2)
			if quadOverlap(pc.corners(), wc.corners()) {
				t.Fatalf("dir=%v: mover overlapped wall at step %d", d, i)
			}
		}

		// After pressing into the wall, the mover must be able to retreat freely.
		if res := m.Move(-d.X*100, -d.Y*100); !res.Moved() {
			t.Fatalf("dir=%v: mover could not retreat after resting against wall", d)
		}
		if quadOverlap(pc.corners(), wc.corners()) {
			t.Fatalf("dir=%v: mover still overlapping after retreat", d)
		}
	}
}
