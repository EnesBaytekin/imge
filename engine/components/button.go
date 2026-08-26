package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// Button is a clickable UI element. It draws a background (a flat color, or
// nine-sliced textures per state) and centered text, and emits an event when
// clicked.
//
// Background: normal/hover/pressed/disabled are optional texture paths. When
// normal is set the button nine-slices it with border; hover/pressed/disabled
// override it for their state. When normal is empty the button fills with color.
//
// State tinting (the default when a state has no texture of its own): the flat
// color — or the base/normal texture, when a state falls back to it — is tinted per
// state so transitions stay visible: hover brightens, pressed darkens, disabled
// desaturates. A state that provides its own texture is drawn untinted.
//
// Interaction: hovering and clicking are detected against the button's rect. On
// click (press then release inside) the button emits Event (if non-empty) with the
// button itself as the data, so a handler can reach the owner via GetOwner(). A
// disabled button (enabled: false) still draws but ignores input. enabled is the
// flag you control for "interactive or not" — e.g. disable a Next button until the
// form is valid. Occlusion (a button behind another panel) is a separate,
// manager-level concern, not this flag.
//
// Export variables (JSON args): text, font_id, size, text_color, normal, hover,
// pressed, disabled, border {left, top, right, bottom}, color, event, offset,
// width, height, visible, enabled, group, draw_layer.
type ButtonComponent struct {
	core.BaseUIComponent

	Text      string     `json:"text"`
	FontID    string     `json:"font_id"`
	Size      float64    `json:"size"`
	TextColor math.Color `json:"text_color"`

	NormalTexture   string `json:"normal"`
	HoverTexture    string `json:"hover"`
	PressedTexture  string `json:"pressed"`
	DisabledTexture string `json:"disabled"`

	Border math.Border `json:"border"`
	Color  math.Color  `json:"color"` // flat fill when no texture is set

	Event string `json:"event"` // event name emitted on click

	hovered bool
	pressed bool
}

// Initialize defaults the text color to white when not specified.
func (b *ButtonComponent) Initialize() {
	if b.TextColor == (math.Color{}) {
		b.TextColor = math.White
	}
}

func (b *ButtonComponent) Update(ctx *core.Context) {
	b.hovered = false
	if !b.IsVisible() || !b.IsEnabled() {
		b.pressed = false
		return
	}

	pos := ctx.Input.GetMousePosition()
	b.hovered = b.Contains(pos)

	if ctx.Input.IsMouseButtonJustPressed(core.MouseButtonLeft) {
		if b.hovered {
			b.pressed = true
		}
	}
	if ctx.Input.IsMouseButtonJustReleased(core.MouseButtonLeft) {
		if b.pressed && b.hovered {
			if b.Event != "" {
				b.Emit(b.Event, b)
			}
		}
		b.pressed = false
	}
}

func (b *ButtonComponent) Draw(r core.Renderer) {
	if !b.IsVisible() {
		return
	}
	rect := b.Rect()

	if tex, explicit := b.currentState(); tex != "" {
		transform := math.ColorTransform{}
		if !explicit {
			transform = b.stateTransform()
		}
		core.DrawNineSliceTransform(r, tex, b.Border, rect, transform)
	} else {
		r.DrawRect(rect, b.stateColor())
	}

	if b.Text == "" {
		return
	}
	tw, th := r.MeasureText(b.Text, b.FontID, b.Size)
	x := rect.X() + (rect.Width()-tw)/2
	y := rect.Y() + (rect.Height()-th)/2
	r.DrawText(b.Text, b.FontID, b.Size, math.NewVector2(x, y), b.TextColor)
}

// currentState returns the texture to draw and whether it is an explicit
// state-specific texture. A state with no texture of its own falls back to the
// base/normal texture (or "" for flat color) with explicit=false, so the caller can
// apply the state tint as the default behavior.
func (b *ButtonComponent) currentState() (texture string, explicit bool) {
	if !b.IsEnabled() {
		if b.DisabledTexture != "" {
			return b.DisabledTexture, true
		}
		return b.NormalTexture, false
	}
	switch {
	case b.pressed && b.PressedTexture != "":
		return b.PressedTexture, true
	case b.hovered && b.HoverTexture != "":
		return b.HoverTexture, true
	default:
		return b.NormalTexture, false
	}
}

// stateColor returns the flat (no-texture) background color for the current state:
// base color for normal, lighter on hover, darker when pressed, desaturated when
// disabled.
func (b *ButtonComponent) stateColor() math.Color {
	c := b.Color
	switch {
	case !b.IsEnabled():
		return desaturate(c).Scale(0.65)
	case b.pressed:
		return c.Lerp(math.Black, 0.2)
	case b.hovered:
		return c.Lerp(math.White, 0.15)
	default:
		return c
	}
}

// stateTransform returns the color transform applied to a base texture when the
// current state has no texture of its own, mirroring stateColor's look: brighten on
// hover, darken when pressed, desaturate when disabled. The identity transform for
// normal.
func (b *ButtonComponent) stateTransform() math.ColorTransform {
	switch {
	case !b.IsEnabled():
		return math.ColorTransform{Grayscale: true, Brightness: -0.3}
	case b.pressed:
		return math.ColorTransform{Brightness: -0.25}
	case b.hovered:
		return math.ColorTransform{Brightness: 0.15}
	default:
		return math.ColorTransform{}
	}
}

// desaturate returns the color's grayscale equivalent (luma in all three RGB
// channels), keeping alpha.
func desaturate(c math.Color) math.Color {
	r, g, b, a := c.ToFloats()
	luma := 0.299*r + 0.587*g + 0.114*b
	return math.NewColorFromFloats(luma, luma, luma, a)
}
