package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// ImageComponent renders a texture (optionally a region of a horizontal sprite
// sheet) at the owner's transform. Set FrameWidth/FrameHeight/FrameCount for
// sprite-sheet animation; set Width/Height to scale the result to a display size.
//
// Export variables (JSON args): texture, width, height, tint, frameWidth,
// frameHeight, frameCount, fps, loop.
type ImageComponent struct {
	core.BaseComponent

	Texture string     `json:"texture"`
	Width   float64    `json:"width"`  // display width (0 = natural)
	Height  float64    `json:"height"` // display height (0 = natural)
	Tint    math.Color `json:"tint"`

	// Sprite sheet animation (horizontal strip)
	FrameWidth  float64 `json:"frameWidth"`
	FrameHeight float64 `json:"frameHeight"`
	FrameCount  int     `json:"frameCount"`
	FPS         float64 `json:"fps"`
	Loop        bool    `json:"loop"`

	// local animation state
	currentFrame int
	frameTimer   float64
	playing      bool
}

// Initialize applies defaults. A sprite sheet (FrameCount > 1) auto-plays.
func (c *ImageComponent) Initialize() {
	if c.Tint == (math.Color{}) {
		c.Tint = math.White
	}
	if c.FPS <= 0 {
		c.FPS = 12
	}
	if c.FrameCount > 1 {
		c.playing = true
		c.Loop = true
	}
}

// Update advances the animation frame based on delta time.
func (c *ImageComponent) Update(ctx *core.Context) {
	if !c.playing || c.FrameCount <= 1 || c.FPS <= 0 {
		return
	}

	c.frameTimer += ctx.DeltaTime()
	frameDuration := 1.0 / c.FPS

	for c.frameTimer >= frameDuration {
		c.frameTimer -= frameDuration
		c.currentFrame++

		if c.currentFrame >= c.FrameCount {
			if c.Loop {
				c.currentFrame = 0
			} else {
				c.currentFrame = c.FrameCount - 1
				c.playing = false
				break
			}
		}
	}
}

// Draw renders the current frame at the owner's transform.
func (c *ImageComponent) Draw(renderer core.Renderer) {
	owner := c.GetOwner()
	if owner == nil || c.Texture == "" {
		return
	}

	var src math.Rect
	var naturalW, naturalH float64

	if c.FrameWidth > 0 && c.FrameHeight > 0 && c.FrameCount > 0 {
		col := c.currentFrame % c.FrameCount
		src = math.NewRect(float64(col)*c.FrameWidth, 0, c.FrameWidth, c.FrameHeight)
		naturalW, naturalH = c.FrameWidth, c.FrameHeight
	} else {
		naturalW, naturalH = renderer.GetTextureSize(c.Texture)
	}

	// Scale the source region up to the requested display size (if any), then
	// apply the owner's own scale on top.
	scale := owner.Transform.Scale
	if c.Width > 0 && naturalW > 0 {
		scale.X *= c.Width / naturalW
	}
	if c.Height > 0 && naturalH > 0 {
		scale.Y *= c.Height / naturalH
	}

	renderer.DrawTexture(c.Texture, src, owner.Transform.Position, scale, owner.Transform.Rotation, c.Tint)
}

// SetTexture sets the texture path to render.
func (c *ImageComponent) SetTexture(path string) {
	c.Texture = path
}

// PlayAnimation starts sprite-sheet playback at the given speed.
func (c *ImageComponent) PlayAnimation(fps float64, loop bool) {
	c.FPS = fps
	c.Loop = loop
	c.currentFrame = 0
	c.frameTimer = 0
	c.playing = true
}

// StopAnimation halts sprite-sheet playback and resets to the first frame.
func (c *ImageComponent) StopAnimation() {
	c.playing = false
	c.currentFrame = 0
	c.frameTimer = 0
}

// SetTint sets the color tint for rendering.
func (c *ImageComponent) SetTint(tint math.Color) {
	c.Tint = tint
}
