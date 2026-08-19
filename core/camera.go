// Package core contains platform-agnostic game engine logic.
// This file defines the Camera — the scene's viewport into the world.
package core

import "github.com/EnesBaytekin/imge/core/math"

// Camera defines what part of the world a scene renders. It is core-level state on
// a Scene (not an object or component), so a scene can follow an object and the
// renderer can transform world coordinates into screen coordinates.
//
// X/Y are the view CENTER in world coordinates. Zoom scales around the viewport
// center (1 = 1:1). Smoothing eases the camera toward its follow target: 0 snaps
// instantly (the default), while a small value like 0.1 trails smoothly. LockX /
// LockY stop the camera from moving along that axis (e.g. a side-scroller locks Y).
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

// NewCamera returns a camera centered at the origin with 1x zoom.
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
	c.X = x
	c.Y = y
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

	if c.Smoothing > 0 && c.Smoothing < 1 {
		if !c.LockX {
			c.X += (tx - c.X) * c.Smoothing
		}
		if !c.LockY {
			c.Y += (ty - c.Y) * c.Smoothing
		}
		return
	}

	if !c.LockX {
		c.X = tx
	}
	if !c.LockY {
		c.Y = ty
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
		(world.X-c.X)*z+c.viewportW/2,
		(world.Y-c.Y)*z+c.viewportH/2,
	)
}

// ScreenToWorld converts a screen point to world coordinates.
func (c *Camera) ScreenToWorld(screen math.Vector2) math.Vector2 {
	z := c.Zoom
	if z <= 0 {
		z = 1
	}
	return math.NewVector2(
		(screen.X-c.viewportW/2)/z+c.X,
		(screen.Y-c.viewportH/2)/z+c.Y,
	)
}
