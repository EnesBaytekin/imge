package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// CheckBox is a toggle control: a small square box with an optional label, which
// flips between checked and unchecked when clicked and emits an event.
//
// Appearance: each state can use its own nine-sliced texture — Texture (unchecked)
// and CheckedTexture (checked). When a state has no texture it falls back to a flat
// box: BoxColor/CheckedColor fill, OutlineColor outline, and — when checked — a
// CheckColor check mark drawn inside the box. BoxSize is the square's side and Gap
// the spacing before the label.
//
// Label: the checkbox draws its own label (Text) to the right of the box, vertically
// centered, with FontID/Size/TextColor. It is part of the clickable area, so clicking
// the label also toggles.
//
// Interaction: hovering and clicking are detected against the element's whole rect
// (box + label). On click (press then release inside) the checkbox flips Checked and
// emits Event (if non-empty) with itself as the data, so a handler reads
// GetChecked(). A disabled checkbox (enabled: false) still draws but ignores input.
//
// Export variables (JSON args): checked, text, font_id, size, text_color, texture,
// checked_texture, border {left, top, right, bottom}, checked_border, box_color,
// checked_color, check_color, outline_color, outline_thickness, box_size, gap, event,
// offset, width, height, visible, enabled, blocking, group, draw_layer.
type CheckBoxComponent struct {
	core.BaseUIComponent

	// Checked is the current state; a handler reads it via GetChecked().
	Checked bool `json:"checked"`

	// Text is the label drawn to the right of the box (empty = no label).
	Text string `json:"text"`

	// FontID and Size set the label font. Size 0 (default) = 12, a crisp multiple of
	// the built-in pixel font's 6-unit design grid.
	FontID    string  `json:"font_id"`
	Size      float64 `json:"size"`

	// TextColor is the label color.
	TextColor math.Color `json:"text_color"`

	// Texture/Border opt the unchecked box into nine-slice rendering;
	// CheckedTexture/CheckedBorder do the same for the checked box. An empty texture
	// for a state means the flat fallback (fill + outline + optional check mark).
	Texture        string      `json:"texture"`
	CheckedTexture string      `json:"checked_texture"`
	Border         math.Border `json:"border"`
	CheckedBorder  math.Border `json:"checked_border"`

	// BoxColor/CheckedColor are the flat fills for the unchecked/checked box when the
	// corresponding texture is empty. CheckColor is the check mark drawn in the flat
	// checked box. OutlineColor/OutlineThickness stroke the flat box.
	BoxColor         math.Color `json:"box_color"`
	CheckedColor     math.Color `json:"checked_color"`
	CheckColor       math.Color `json:"check_color"`
	OutlineColor     math.Color `json:"outline_color"`
	OutlineThickness float64    `json:"outline_thickness"`

	// BoxSize is the square box side; Gap the spacing between the box and the label.
	BoxSize float64 `json:"box_size"`
	Gap     float64 `json:"gap"`

	// Event is emitted on toggle, with the checkbox itself as the data.
	Event string `json:"event"`

	hovered bool
	pressed bool
}

// Initialize defaults the colors, sizes, and gaps, and makes the checkbox block
// pointer events (so it occludes whatever is drawn behind it).
func (c *CheckBoxComponent) Initialize() {
	if c.TextColor == (math.Color{}) {
		c.TextColor = math.White
	}
	if c.BoxColor == (math.Color{}) {
		c.BoxColor = math.NewColor(30, 30, 42, 255)
	}
	if c.CheckedColor == (math.Color{}) {
		c.CheckedColor = c.BoxColor
	}
	if c.CheckColor == (math.Color{}) {
		c.CheckColor = math.White
	}
	if c.OutlineColor == (math.Color{}) {
		c.OutlineColor = math.NewColor(74, 74, 106, 255)
	}
	if c.OutlineThickness <= 0 {
		c.OutlineThickness = 1
	}
	if c.Size <= 0 {
		c.Size = 12
	}
	if c.BoxSize <= 0 {
		c.BoxSize = 16
	}
	if c.Gap <= 0 {
		c.Gap = 6
	}
	if c.Blocking == nil {
		b := true
		c.Blocking = &b
	}
}

// GetChecked returns the current state.
func (c *CheckBoxComponent) GetChecked() bool { return c.Checked }

// SetChecked sets the state silently (no Event).
func (c *CheckBoxComponent) SetChecked(v bool) { c.Checked = v }

// SetHovered sets the hover state, driven by a @UIManager.
func (c *CheckBoxComponent) SetHovered(hovered bool) { c.hovered = hovered }

// SetPressed sets the pressed state, driven by a @UIManager.
func (c *CheckBoxComponent) SetPressed(pressed bool) { c.pressed = pressed }

// Activate flips the checked state and emits Event (if non-empty) with the checkbox
// itself as the data. Called by a @UIManager when a press is released inside.
func (c *CheckBoxComponent) Activate() {
	c.Checked = !c.Checked
	if c.Event != "" {
		c.Emit(c.Event, c)
	}
}

// Draw renders the box, then the label.
func (c *CheckBoxComponent) Draw(r core.Renderer) {
	if !c.IsVisible() {
		return
	}
	c.drawBox(r)
	c.drawLabel(r)
}

// boxRect returns the square box, vertically centered at the element's left edge.
func (c *CheckBoxComponent) boxRect() math.Rect {
	r := c.Rect()
	size := c.boxSize()
	return math.NewRect(r.Left(), r.Center().Y-size/2, size, size)
}

// drawBox draws the current state's box: a nine-sliced texture, or the flat fallback
// (fill, check mark when checked, and outline).
func (c *CheckBoxComponent) drawBox(r core.Renderer) {
	br := c.boxRect()
	flat := false
	if c.Checked {
		if c.CheckedTexture != "" {
			core.DrawNineSlice(r, c.CheckedTexture, c.checkedBorder(), br)
		} else {
			r.DrawRect(br, c.checkedColor())
			c.drawCheck(r, br)
			flat = true
		}
	} else {
		if c.Texture != "" {
			core.DrawNineSlice(r, c.Texture, c.Border, br)
		} else {
			r.DrawRect(br, c.boxColor())
			flat = true
		}
	}
	if flat {
		c.drawOutline(r, br)
	}
}

// drawCheck draws the check mark as two strokes inside the box.
func (c *CheckBoxComponent) drawCheck(r core.Renderer, br math.Rect) {
	t := br.Width() * 0.12
	if t < 1 {
		t = 1
	}
	p1 := math.NewVector2(br.Left()+br.Width()*0.22, br.Top()+br.Height()*0.55)
	p2 := math.NewVector2(br.Left()+br.Width()*0.42, br.Top()+br.Height()*0.78)
	p3 := math.NewVector2(br.Left()+br.Width()*0.80, br.Top()+br.Height()*0.26)
	r.DrawLine(p1, p2, c.CheckColor, t)
	r.DrawLine(p2, p3, c.CheckColor, t)
}

// drawOutline strokes the flat box with the outline color.
func (c *CheckBoxComponent) drawOutline(r core.Renderer, br math.Rect) {
	if c.OutlineThickness > 0 && c.OutlineColor.A > 0 {
		r.DrawRectOutline(br, c.outlineColor(), c.OutlineThickness)
	}
}

// drawLabel draws the label to the right of the box, vertically centered.
func (c *CheckBoxComponent) drawLabel(r core.Renderer) {
	if c.Text == "" {
		return
	}
	br := c.boxRect()
	size := c.textSize()
	_, th := r.MeasureText(c.Text, c.FontID, size)
	x := br.Right() + c.gap()
	y := c.Rect().Center().Y - th/2
	r.DrawText(c.Text, c.FontID, size, math.NewVector2(x, y), c.TextColor)
}

// boxColor returns the unchecked flat fill, darkened slightly while pressed.
func (c *CheckBoxComponent) boxColor() math.Color {
	col := c.BoxColor
	if c.pressed {
		col = col.Lerp(math.Black, 0.15)
	}
	return col
}

// checkedColor returns the checked flat fill, darkened slightly while pressed.
func (c *CheckBoxComponent) checkedColor() math.Color {
	col := c.CheckedColor
	if c.pressed {
		col = col.Lerp(math.Black, 0.15)
	}
	return col
}

// outlineColor returns the flat outline color, brightened slightly while hovered.
func (c *CheckBoxComponent) outlineColor() math.Color {
	col := c.OutlineColor
	if c.hovered {
		col = col.Lerp(math.White, 0.2)
	}
	return col
}

// checkedBorder returns the checked texture's nine-slice border, falling back to the
// unchecked Border when CheckedBorder is left zero.
func (c *CheckBoxComponent) checkedBorder() math.Border {
	if c.CheckedBorder != (math.Border{}) {
		return c.CheckedBorder
	}
	return c.Border
}

// boxSize returns the square box side.
func (c *CheckBoxComponent) boxSize() float64 {
	if c.BoxSize <= 0 {
		return 16
	}
	return c.BoxSize
}

// gap returns the spacing between the box and the label.
func (c *CheckBoxComponent) gap() float64 {
	if c.Gap <= 0 {
		return 6
	}
	return c.Gap
}

// textSize returns the label font size.
func (c *CheckBoxComponent) textSize() float64 {
	if c.Size <= 0 {
		return 12
	}
	return c.Size
}
