// Package core contains platform-agnostic game engine logic.
// This file defines the Camera — the scene's viewport into the world.
package core

import "github.com/EnesBaytekin/imge/core/math"

// Camera defines what part of the world a scene renders. It is core-level state on
// a Scene (not an object or component), so a scene can follow an object and the
// renderer can transform world coordinates into screen coordinates.
//
// X/Y are the world coordinates of the viewport's TOP-LEFT corner, so the world
// origin (0,0) appears at the top-left of the screen. Zoom scales the view around
// its center (1 = 1:1). Smoothing eases the camera toward its follow target: 0
// snaps instantly (the default), while a small value like 0.1 trails smoothly.
// LockX / LockY stop the camera from moving along that axis (e.g. a side-scroller
// locks Y).
//
// A scene with no camera (Camera == nil) draws with world = screen (the default).
type Camera struct {
	X    float64 `json:"x"`
	Y    float64 `json:"y"`
	Zoom float64 `json:"zoom"`

	Smoothing float64 `json:"smoothing"`
	LockX     bool    `json:"lock_x"`
	LockY     bool    `json:"lock_y"`

	target    *Object
	targetX   float64
	targetY   float64
	following bool

	viewportW float64
	viewportH float64
}

// NewCamera returns a camera with the view's top-left corner at the origin and 1x zoom.
func NewCamera() *Camera {
	return &Camera{Zoom: 1}
}

// Follow makes the camera track an object's position (its transform origin).
func (c *Camera) Follow(obj *Object) {
	c.target = obj
	c.following = true
}

// FollowPoint makes the camera track a fixed world point.
func (c *Camera) FollowPoint(x, y float64) {
	c.target = nil
	c.targetX = x
	c.targetY = y
	c.following = true
}

// LookAt immediately centers the camera on a point and stops following.
func (c *Camera) LookAt(x, y float64) {
	c.following = false
	c.target = nil
	c.centerOn(x, y)
}

// centerOn sets the camera so the given world point sits at the viewport center.
func (c *Camera) centerOn(x, y float64) {
	z := c.Zoom
	if z <= 0 {
		z = 1
	}
	c.X = x - c.viewportW/(2*z)
	c.Y = y - c.viewportH/(2*z)
}

// StopFollow stops following, leaving the camera where it is.
func (c *Camera) StopFollow() {
	c.following = false
	c.target = nil
}

// Tick advances the camera toward its follow target, applying smoothing. Called
// once per frame by the scene after objects update.
func (c *Camera) Tick() {
	if !c.following {
		return
	}

	tx, ty := c.targetX, c.targetY
	if c.target != nil {
		tx = c.target.Transform.Position.X
		ty = c.target.Transform.Position.Y
	}

	// X/Y is the view's top-left corner, so center the target by aiming the
	// top-left at (target - half the viewport). The viewport is set before Tick
	// runs (see Scene.Draw).
	z := c.Zoom
	if z <= 0 {
		z = 1
	}
	gx := tx - c.viewportW/(2*z)
	gy := ty - c.viewportH/(2*z)

	if c.Smoothing > 0 && c.Smoothing < 1 {
		if !c.LockX {
			c.X += (gx - c.X) * c.Smoothing
		}
		if !c.LockY {
			c.Y += (gy - c.Y) * c.Smoothing
		}
		return
	}

	if !c.LockX {
		c.X = gx
	}
	if !c.LockY {
		c.Y = gy
	}
}

// setViewport records the render target size used by WorldToScreen/ScreenToWorld.
// Called by the scene each frame before drawing.
func (c *Camera) setViewport(width, height float64) {
	c.viewportW = width
	c.viewportH = height
}

// WorldToScreen converts a world point to screen coordinates.
func (c *Camera) WorldToScreen(world math.Vector2) math.Vector2 {
	z := c.Zoom
	if z <= 0 {
		z = 1
	}
	return math.NewVector2(
		(world.X-c.X)*z,
		(world.Y-c.Y)*z,
	)
}

// ScreenToWorld converts a screen point to world coordinates.
func (c *Camera) ScreenToWorld(screen math.Vector2) math.Vector2 {
	z := c.Zoom
	if z <= 0 {
		z = 1
	}
	return math.NewVector2(
		screen.X/z+c.X,
		screen.Y/z+c.Y,
	)
}
