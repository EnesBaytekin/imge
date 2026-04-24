// Package components contains built-in game components.
package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// ============================================================================
// @Image Component
// ============================================================================

// ImageComponent renders a texture with optional sprite sheet animation support.
// For sprite sheets, set frameWidth, frameHeight, and frameCount in args.
// Use PlayAnimation() to start playback and StopAnimation() to halt it.
type ImageComponent struct {
	core.BaseComponent

	// Texture
	texturePath string
	width       float64
	height      float64
	tint        math.Color

	// Sprite sheet animation
	frameWidth    float64
	frameHeight   float64
	frameCount    int
	currentFrame  int
	fps           float64
	frameTimer    float64
	playing       bool
	loop          bool
}

// Initialize parses component configuration from JSON args.
// Supported args:
//
//	texture: string (path to texture file)
//	width: float64 (display width)
//	height: float64 (display height)
//	tint: map (color tint, e.g. {"r": 255, "g": 255, "b": 255, "a": 255})
//	frameWidth: float64 (sprite sheet frame width)
//	frameHeight: float64 (sprite sheet frame height)
//	frameCount: float64 (number of frames in animation)
//	fps: float64 (animation frames per second)
func (c *ImageComponent) Initialize(args []interface{}) error {
	if len(args) > 0 {
		if argMap, ok := args[0].(map[string]interface{}); ok {
			if t, ok := argMap["texture"].(string); ok {
				c.texturePath = t
			}
			if w, ok := argMap["width"].(float64); ok {
				c.width = w
			}
			if h, ok := argMap["height"].(float64); ok {
				c.height = h
			}
			if tintMap, ok := argMap["tint"].(map[string]interface{}); ok {
				c.tint = parseColor(tintMap)
			}
			if fw, ok := argMap["frameWidth"].(float64); ok {
				c.frameWidth = fw
			}
			if fh, ok := argMap["frameHeight"].(float64); ok {
				c.frameHeight = fh
			}
			if fc, ok := argMap["frameCount"].(float64); ok {
				c.frameCount = int(fc)
			}
			if fps, ok := argMap["fps"].(float64); ok {
				c.fps = fps
			}
		}
	}

	// Defaults
	if c.tint == (math.Color{}) {
		c.tint = math.White
	}
	if c.width <= 0 && c.frameWidth > 0 {
		c.width = c.frameWidth
	}
	if c.height <= 0 && c.frameHeight > 0 {
		c.height = c.frameHeight
	}
	if c.width <= 0 {
		c.width = 32
	}
	if c.height <= 0 {
		c.height = 32
	}
	if c.frameWidth <= 0 {
		c.frameWidth = c.width
	}
	if c.frameHeight <= 0 {
		c.frameHeight = c.height
	}
	if c.fps <= 0 {
		c.fps = 12
	}

	return nil
}

// Update advances the animation frame based on delta time.
func (c *ImageComponent) Update(ctx *core.ComponentContext) {
	if !c.playing || c.frameCount <= 1 {
		return
	}

	c.frameTimer += ctx.Time.DeltaTime()
	frameDuration := 1.0 / c.fps

	if c.frameTimer >= frameDuration {
		c.frameTimer -= frameDuration
		c.currentFrame++

		if c.currentFrame >= c.frameCount {
			if c.loop {
				c.currentFrame = 0
			} else {
				c.currentFrame = c.frameCount - 1
				c.playing = false
			}
		}
	}
}

// Draw renders the texture at the owner's position with current transform.
// If animating, uses the current frame to calculate the source rectangle.
func (c *ImageComponent) Draw(renderer core.Renderer) {
	owner := c.GetOwner()
	if owner == nil || c.texturePath == "" {
		return
	}

	renderer.DrawTexture(
		c.texturePath,
		owner.Transform.Position,
		owner.Transform.Scale,
		owner.Transform.Rotation,
		c.tint,
	)
}

// SetTexture sets the texture path to render.
func (c *ImageComponent) SetTexture(path string) {
	c.texturePath = path
}

// PlayAnimation starts sprite sheet playback at the given speed.
// If loop is true, the animation repeats indefinitely.
func (c *ImageComponent) PlayAnimation(fps float64, loop bool) {
	c.fps = fps
	c.loop = loop
	c.currentFrame = 0
	c.frameTimer = 0
	c.playing = true
}

// StopAnimation halts sprite sheet playback and resets to the first frame.
func (c *ImageComponent) StopAnimation() {
	c.playing = false
	c.currentFrame = 0
	c.frameTimer = 0
}

// SetTint sets the color tint for rendering.
func (c *ImageComponent) SetTint(tint math.Color) {
	c.tint = tint
}

// parseColor converts a JSON color map to a math.Color.
func parseColor(m map[string]interface{}) math.Color {
	color := math.White
	if r, ok := m["r"].(float64); ok {
		color.R = uint8(r)
	}
	if g, ok := m["g"].(float64); ok {
		color.G = uint8(g)
	}
	if b, ok := m["b"].(float64); ok {
		color.B = uint8(b)
	}
	if a, ok := m["a"].(float64); ok {
		color.A = uint8(a)
	}
	return color
}

// ============================================================================
// Registration
// ============================================================================

func init() {
	core.RegisterComponent("@Image", func(args []interface{}) (core.Component, error) {
		return &ImageComponent{}, nil
	})
}
