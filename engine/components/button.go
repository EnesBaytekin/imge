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
// override it for their state, and a state with no texture falls back to normal.
// When normal is empty the button fills with color instead. There is no automatic
// tinting on hover — each visual state is an explicit texture you provide.
//
// Interaction: hovering and clicking are detected against the button's rect. On
// click (press then release inside) the button emits Event (if non-empty) with the
// button itself as the data, so a handler can reach the owner via GetOwner(). A
// disabled button (enabled: false) still draws but ignores input; give it a
// disabled texture so the non-interactive state is visually distinct. enabled is
// the flag you control for "interactive or not" — e.g. disable a Next button until
// the form is valid. Occlusion (a button behind another panel) is a separate,
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
	if tex := b.currentTexture(); tex != "" {
		core.DrawNineSlice(r, tex, b.Border, rect)
	} else {
		r.DrawRect(rect, b.Color)
	}
	if b.Text == "" {
		return
	}
	tw, th := r.MeasureText(b.Text, b.FontID, b.Size)
	x := rect.X() + (rect.Width()-tw)/2
	y := rect.Y() + (rect.Height()-th)/2
	r.DrawText(b.Text, b.FontID, b.Size, math.NewVector2(x, y), b.TextColor)
}

// currentTexture returns the texture for the current visual state, falling back to
// normal and finally to "" (flat color). A disabled button shows its disabled
// texture (falling back to normal) regardless of hover/pressed.
func (b *ButtonComponent) currentTexture() string {
	if !b.IsEnabled() {
		if b.DisabledTexture != "" {
			return b.DisabledTexture
		}
		return b.NormalTexture
	}
	switch {
	case b.pressed && b.PressedTexture != "":
		return b.PressedTexture
	case b.hovered && b.HoverTexture != "":
		return b.HoverTexture
	default:
		return b.NormalTexture
	}
}
