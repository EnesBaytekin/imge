package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// ImageComponent renders a texture (optionally a region of a horizontal sprite
// sheet) at the owner's transform. Set frameWidth/frameHeight/frameCount for
// sprite-sheet animation; set width/height to scale the result to a display size.
type ImageComponent struct {
	core.BaseComponent

	texturePath string
	width       float64 // display width (0 = natural)
	height      float64 // display height (0 = natural)
	tint        math.Color

	// Sprite sheet animation (horizontal strip)
	frameWidth   float64
	frameHeight  float64
	frameCount   int
	currentFrame int
	fps          float64
	frameTimer   float64
	playing      bool
	loop         bool
}

// Initialize parses component configuration from JSON args.
// Supported args: texture, width, height, tint, frameWidth, frameHeight,
// frameCount, fps, loop.
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
			if l, ok := argMap["loop"].(bool); ok {
				c.loop = l
			}
		}
	}

	if c.tint == (math.Color{}) {
		c.tint = math.White
	}
	if c.fps <= 0 {
		c.fps = 12
	}

	// Auto-play sprite sheets; a single-frame image stays static.
	if c.frameCount > 1 {
		c.playing = true
		c.loop = true
	}

	return nil
}

// Update advances the animation frame based on delta time.
func (c *ImageComponent) Update(ctx *core.ComponentContext) {
	if !c.playing || c.frameCount <= 1 || c.fps <= 0 {
		return
	}

	c.frameTimer += ctx.Time.DeltaTime()
	frameDuration := 1.0 / c.fps

	for c.frameTimer >= frameDuration {
		c.frameTimer -= frameDuration
		c.currentFrame++

		if c.currentFrame >= c.frameCount {
			if c.loop {
				c.currentFrame = 0
			} else {
				c.currentFrame = c.frameCount - 1
				c.playing = false
				break
			}
		}
	}
}

// Draw renders the current frame at the owner's transform.
func (c *ImageComponent) Draw(renderer core.Renderer) {
	owner := c.GetOwner()
	if owner == nil || c.texturePath == "" {
		return
	}

	var src math.Rect
	var naturalW, naturalH float64

	if c.frameWidth > 0 && c.frameHeight > 0 && c.frameCount > 0 {
		col := c.currentFrame % c.frameCount
		src = math.NewRect(float64(col)*c.frameWidth, 0, c.frameWidth, c.frameHeight)
		naturalW, naturalH = c.frameWidth, c.frameHeight
	} else {
		naturalW, naturalH = renderer.GetTextureSize(c.texturePath)
	}

	// Scale the source region up to the requested display size (if any), then
	// apply the owner's own scale on top.
	scale := owner.Transform.Scale
	if c.width > 0 && naturalW > 0 {
		scale.X *= c.width / naturalW
	}
	if c.height > 0 && naturalH > 0 {
		scale.Y *= c.height / naturalH
	}

	renderer.DrawTexture(c.texturePath, src, owner.Transform.Position, scale, owner.Transform.Rotation, c.tint)
}

// SetTexture sets the texture path to render.
func (c *ImageComponent) SetTexture(path string) {
	c.texturePath = path
}

// PlayAnimation starts sprite-sheet playback at the given speed.
func (c *ImageComponent) PlayAnimation(fps float64, loop bool) {
	c.fps = fps
	c.loop = loop
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

func init() {
	core.RegisterComponent("@Image", func(args []interface{}) (core.Component, error) {
		return &ImageComponent{}, nil
	})
}
