package components

import (
	"fmt"
	stdmath "math"
	"strconv"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// ColorPicker is a color chooser: a swatch that shows the currently selected
// color and, when clicked, opens a panel with the classic picker controls —
// an SV gradient square, a hue bar, an alpha bar, RGB(A) and hex entry fields,
// an old-vs-new preview, and an OK button that commits the choice.
//
// Picking and committing:
//   - click the swatch to open the panel (the in-progress color starts from the
//     committed one); click anywhere else (or press Escape) to cancel,
//   - press+drag on the square to set saturation/value, on the hue bar to set hue,
//     and on the alpha bar to set opacity,
//   - click an R/G/B/A or Hex field and type a value; the color follows the field
//     as you type, and Enter applies,
//   - click OK (or press Enter) to commit: the swatch switches to the new color,
//     the panel closes, and Event fires. Anything short of OK discards the change.
//
// Appearance: the closed swatch is the committed color inset by Padding, framed by
// a nine-sliced Texture (with Border) or, when no texture, a flat BorderColor, with
// an optional outline (BorderThickness). The panel is a flat PanelColor or a
// nine-sliced PanelTexture, and the square/bars/fields/OK all have flat-color
// fallbacks. Defaults can come from styles.imge under the "@ColorPicker" key.
//
// Export variables (JSON args): color, event, padding {left, top, right, bottom},
// texture, border {left, top, right, bottom}, border_color, border_thickness,
// panel_color, panel_texture, panel_border, square_size, bar_width, font_id, size,
// text_color, field_color, accent_color, marker_color, offset, width, height,
// visible, enabled, blocking, group, draw_layer.
type ColorPickerComponent struct {
	core.BaseUIComponent

	// Color is the committed color (the value a handler reads via GetColor()).
	Color math.Color `json:"color"`

	// Event is emitted on commit, with the picker itself as the data (so a handler
	// can read GetColor() and GetName()/GetOwner()), matching @Button/@Slider/@ComboBox.
	Event string `json:"event"`

	// Padding insets the color fill from the swatch edges, so the frame (texture or
	// border color) stays visible around it.
	Padding math.Border `json:"padding"`

	// Texture/Border opt the swatch into nine-slice rendering; empty texture = flat
	// fill. PanelTexture/PanelBorder do the same for the open panel.
	Texture      string      `json:"texture"`
	Border       math.Border `json:"border"`
	PanelTexture string      `json:"panel_texture"`
	PanelBorder  math.Border `json:"panel_border"`

	// BorderColor/BorderThickness stroke the swatch (and preview swatches). A
	// transparent color means no outline; thickness 0 defaults to 1.
	BorderColor     math.Color `json:"border_color"`
	BorderThickness float64    `json:"border_thickness"`

	// PanelColor is the panel's flat fill when PanelTexture is empty.
	PanelColor math.Color `json:"panel_color"`

	// SquareSize is the SV square side; BarWidth is the hue/alpha bar width.
	SquareSize float64 `json:"square_size"`
	BarWidth   float64 `json:"bar_width"`

	// FontID/Size set the panel text font. Size 0 (default) = 12 (crisp).
	FontID string  `json:"font_id"`
	Size   float64 `json:"size"`

	// TextColor is the field value text; FieldColor the input-box fill; AccentColor
	// the OK button; MarkerColor the square/bar cursors.
	TextColor   math.Color `json:"text_color"`
	FieldColor  math.Color `json:"field_color"`
	AccentColor math.Color `json:"accent_color"`
	MarkerColor math.Color `json:"marker_color"`

	// open is whether the panel is shown; working is the in-progress color while it
	// is. viewportH is the logical screen height, captured each frame in Draw.
	open      bool
	working   math.Color
	viewportH float64

	// dragRegion is which continuous control a held drag is adjusting (see the
	// drag* constants); activeField is the text field being edited (field* constants).
	dragRegion  int
	activeField int
	editR       string
	editG       string
	editB       string
	editA       string
	editHex     string
	pressedOK   bool
	hoverOK     bool
}

// drag-region constants for the uiSlider gesture.
const (
	dragNone = iota
	dragSquare
	dragHue
	dragAlpha
)

// text-field constants (fieldNone = no field active).
const (
	fieldNone = -1
	fieldR    = 0
	fieldG    = 1
	fieldB    = 2
	fieldA    = 3
	fieldHex  = 4
)

// panel layout constants (logical units).
const (
	cpMargin   = 8.0  // outer margin of the panel
	cpGap      = 6.0  // gap between stacked rows / bars
	cpFieldGap = 4.0  // gap between the R/G/B/A boxes
	cpFieldH   = 20.0 // input-box height
	cpPrevH    = 20.0 // old/new preview height
	cpOKH      = 22.0 // OK button height

	cpFieldValueSize = 6.0 // value text inside the R/G/B/A and Hex boxes (small + crisp)
)

// Initialize defaults colors, sizes, and flags, and opts into blocking + keyboard
// focus (a picker is inherently focusable for typing and outside-click-to-close).
func (c *ColorPickerComponent) Initialize() {
	if c.Color == (math.Color{}) {
		c.Color = math.White
	}
	if c.PanelColor == (math.Color{}) {
		c.PanelColor = math.NewColor(40, 40, 54, 255)
	}
	if c.BorderColor == (math.Color{}) {
		c.BorderColor = math.NewColor(74, 74, 106, 255)
	}
	if c.BorderThickness <= 0 {
		c.BorderThickness = 1
	}
	if c.TextColor == (math.Color{}) {
		c.TextColor = math.White
	}
	if c.FieldColor == (math.Color{}) {
		c.FieldColor = math.NewColor(24, 24, 34, 255)
	}
	if c.AccentColor == (math.Color{}) {
		c.AccentColor = math.NewColor(90, 158, 90, 255)
	}
	if c.MarkerColor == (math.Color{}) {
		c.MarkerColor = math.White
	}
	if c.Size <= 0 {
		c.Size = 12
	}
	if c.SquareSize <= 0 {
		c.SquareSize = 88
	}
	if c.BarWidth <= 0 {
		c.BarWidth = 14
	}
	if c.Blocking == nil {
		b := true
		c.Blocking = &b
	}
	c.Focusable = true
	c.working = c.Color
	c.activeField = fieldNone
}

// GetColor returns the committed color.
func (c *ColorPickerComponent) GetColor() math.Color { return c.Color }

// SetColor sets the committed color silently (no Event). When the panel is closed it
// also seeds the working color, so reopening starts from the new value.
func (c *ColorPickerComponent) SetColor(col math.Color) {
	c.Color = col
	if !c.open {
		c.working = col
	}
}

// Contains reports whether p is over the swatch or, when open, the panel — so the
// manager hit-tests the whole popup as one element.
func (c *ColorPickerComponent) Contains(p math.Vector2) bool {
	if c.headerRect().ContainsPoint(p) {
		return true
	}
	if c.open {
		return c.panelRect().ContainsPoint(p)
	}
	return false
}

// PointerMove tracks whether the pointer is over the OK button (for its hover tint).
// Called by a @UIManager every frame the pointer is over the element.
func (c *ColorPickerComponent) PointerMove(pos math.Vector2) {
	if !c.open {
		c.hoverOK = false
		return
	}
	c.hoverOK = c.okRect().ContainsPoint(pos)
}

// PointerLeave clears the hover state. Called by a @UIManager.
func (c *ColorPickerComponent) PointerLeave() { c.hoverOK = false }

// Press opens the panel (when closed) or records the press for release handling
// (OK button, text field activation). The square/hue/alpha drag is owned by the
// uiSlider gesture (BeginAdjust), not here. Called by a @UIManager on press.
func (c *ColorPickerComponent) Press(pos math.Vector2) {
	if !c.IsEnabled() {
		return
	}
	if !c.open {
		c.openPanel()
		return
	}
	c.pressedOK = c.okRect().ContainsPoint(pos)
	if c.pressedOK {
		c.activeField = fieldNone
		return
	}
	if f := c.fieldAt(pos); f != fieldNone {
		c.activateField(f)
	} else {
		c.activeField = fieldNone
	}
}

// Release commits when the OK button was pressed and released over itself. Called by
// a @UIManager on release.
func (c *ColorPickerComponent) Release(pos math.Vector2) {
	if !c.open {
		c.pressedOK = false
		return
	}
	if c.pressedOK && c.okRect().ContainsPoint(pos) {
		c.commit()
	}
	c.pressedOK = false
}

// BeginAdjust starts a drag gesture on whichever continuous control the pointer hit
// (square/hue/alpha), applying the position immediately for press-to-jump. Called by
// a @UIManager on press.
func (c *ColorPickerComponent) BeginAdjust(pos math.Vector2) {
	if !c.IsEnabled() || !c.open {
		c.dragRegion = dragNone
		return
	}
	c.dragRegion = c.regionAt(pos)
	if c.dragRegion != dragNone {
		c.activeField = fieldNone
		c.applyDrag(pos)
	}
}

// Adjust updates the dragged control from the pointer position. Called by a
// @UIManager on every held frame (even if the pointer has left the element).
func (c *ColorPickerComponent) Adjust(pos math.Vector2) {
	if c.dragRegion == dragNone {
		return
	}
	c.applyDrag(pos)
}

// EndAdjust ends the drag gesture. Called by a @UIManager on release.
func (c *ColorPickerComponent) EndAdjust() { c.dragRegion = dragNone }

// SetFocused cancels (discards the working color) and closes when focus leaves, e.g.
// a click outside. Opening stays on Press so the first click doesn't both focus and
// open.
func (c *ColorPickerComponent) SetFocused(focused bool) {
	if !focused && c.open {
		c.cancel()
	}
}

// HandleInput handles keyboard: Enter/Space opens when closed; with a field active it
// routes typing (InputChars + Backspace) and Enter-applies/Escape-blurs; otherwise
// Enter commits and Escape cancels. Called by a @UIManager while focused.
func (c *ColorPickerComponent) HandleInput(ctx *core.Context) {
	if !c.IsEnabled() {
		return
	}
	in := ctx.Input

	if !c.open {
		if in.IsKeyJustPressed(core.KeyEnter) || in.IsKeyJustPressed(core.KeySpace) {
			c.openPanel()
		}
		return
	}

	if c.activeField != fieldNone {
		c.handleFieldInput(in)
		return
	}

	if in.IsKeyJustPressed(core.KeyEscape) {
		c.cancel()
		return
	}
	if in.IsKeyJustPressed(core.KeyEnter) {
		c.commit()
	}
}

// ============================================================================
// Lifecycle helpers
// ============================================================================

func (c *ColorPickerComponent) openPanel() {
	c.open = true
	c.working = c.Color
	c.activeField = fieldNone
	c.dragRegion = dragNone
	c.pressedOK = false
	c.hoverOK = false
}

func (c *ColorPickerComponent) closePanel() {
	c.open = false
	c.activeField = fieldNone
	c.dragRegion = dragNone
	c.pressedOK = false
	c.hoverOK = false
}

// commit sets the committed color to the working color (emitting Event only on an
// actual change) and closes the panel.
func (c *ColorPickerComponent) commit() {
	if c.working != c.Color {
		c.Color = c.working
		if c.Event != "" {
			c.Emit(c.Event, c)
		}
	}
	c.closePanel()
}

// cancel discards the working color and closes the panel.
func (c *ColorPickerComponent) cancel() { c.closePanel() }

// regionAt reports which continuous control the point is over.
func (c *ColorPickerComponent) regionAt(pos math.Vector2) int {
	if c.squareRect().ContainsPoint(pos) {
		return dragSquare
	}
	if c.hueRect().ContainsPoint(pos) {
		return dragHue
	}
	if c.alphaRect().ContainsPoint(pos) {
		return dragAlpha
	}
	return dragNone
}

// applyDrag maps the pointer position to a value for the dragged control and writes
// it into working, carrying the unchanged channels (square/hue keep alpha; alpha
// keeps RGB).
func (c *ColorPickerComponent) applyDrag(pos math.Vector2) {
	switch c.dragRegion {
	case dragSquare:
		h, _, _ := rgbToHSV(c.working)
		sq := c.squareRect()
		s := clampf((pos.X-sq.Left())/sq.Width(), 0, 1)
		v := 1 - clampf((pos.Y-sq.Top())/sq.Height(), 0, 1)
		c.working = hsvToRGB(h, s, v).WithAlpha(c.working.A)
	case dragHue:
		_, s, v := rgbToHSV(c.working)
		hb := c.hueRect()
		f := clampf((pos.Y-hb.Top())/hb.Height(), 0, 1)
		h := 360 * (1 - f)
		if h >= 360 {
			h = 359.999
		}
		c.working = hsvToRGB(h, s, v).WithAlpha(c.working.A)
	case dragAlpha:
		ab := c.alphaRect()
		f := clampf((pos.Y-ab.Top())/ab.Height(), 0, 1)
		c.working = c.working.WithAlpha(uint8(stdmath.Round((1 - f) * 255)))
	}
}

func (c *ColorPickerComponent) fieldAt(pos math.Vector2) int {
	for i := 0; i < 4; i++ {
		if c.fieldRect(i).ContainsPoint(pos) {
			return i
		}
	}
	if c.hexRect().ContainsPoint(pos) {
		return fieldHex
	}
	return fieldNone
}

// activateField seeds the field's edit buffer from the working color and makes it the
// active field.
func (c *ColorPickerComponent) activateField(f int) {
	c.activeField = f
	switch f {
	case fieldR:
		c.editR = strconv.Itoa(int(c.working.R))
	case fieldG:
		c.editG = strconv.Itoa(int(c.working.G))
	case fieldB:
		c.editB = strconv.Itoa(int(c.working.B))
	case fieldA:
		c.editA = strconv.Itoa(int(c.working.A))
	case fieldHex:
		c.editHex = formatHex(c.working)
	}
}

// handleFieldInput edits the active field: Escape blurs, Enter applies, Backspace
// deletes, and typed characters append; any successful edit re-applies the field to
// the working color.
func (c *ColorPickerComponent) handleFieldInput(in core.Input) {
	if in.IsKeyJustPressed(core.KeyEscape) {
		c.activeField = fieldNone
		return
	}
	if in.IsKeyJustPressed(core.KeyEnter) {
		c.commitField()
		return
	}
	changed := false
	if in.IsKeyJustPressed(core.KeyBackspace) {
		c.editBackspace()
		changed = true
	}
	for _, r := range in.InputChars() {
		if c.editAppend(r) {
			changed = true
		}
	}
	if changed {
		c.applyFieldLive()
	}
}

// editAppend appends r to the active field's buffer if it is a valid character and
// the field's max length isn't reached. Returns whether it appended.
func (c *ColorPickerComponent) editAppend(r rune) bool {
	buf := c.editBuffer()
	switch c.activeField {
	case fieldR, fieldG, fieldB, fieldA:
		if r >= '0' && r <= '9' && len(*buf) < 3 {
			*buf += string(r)
			return true
		}
	case fieldHex:
		if len(*buf) >= 9 {
			return false
		}
		if r == '#' && *buf == "" {
			*buf = "#"
			return true
		}
		if isHexRune(r) {
			*buf += string(r)
			return true
		}
	}
	return false
}

func (c *ColorPickerComponent) editBackspace() {
	buf := c.editBuffer()
	if len(*buf) > 0 {
		*buf = (*buf)[:len(*buf)-1]
	}
}

// editBuffer returns a pointer to the active field's edit buffer.
func (c *ColorPickerComponent) editBuffer() *string {
	switch c.activeField {
	case fieldR:
		return &c.editR
	case fieldG:
		return &c.editG
	case fieldB:
		return &c.editB
	case fieldA:
		return &c.editA
	default:
		return &c.editHex
	}
}

// applyFieldLive applies a (valid) active-field buffer to the working color without
// resetting invalid/partial input.
func (c *ColorPickerComponent) applyFieldLive() {
	switch c.activeField {
	case fieldR:
		if v, ok := parseByte(c.editR); ok {
			c.working.R = v
		}
	case fieldG:
		if v, ok := parseByte(c.editG); ok {
			c.working.G = v
		}
	case fieldB:
		if v, ok := parseByte(c.editB); ok {
			c.working.B = v
		}
	case fieldA:
		if v, ok := parseByte(c.editA); ok {
			c.working.A = v
		}
	case fieldHex:
		if col, err := math.ParseHex(c.editHex); err == nil {
			c.working = col
		}
	}
}

// commitField applies the active field on Enter, resetting the buffer to the working
// value when the input was invalid, and clears the active field.
func (c *ColorPickerComponent) commitField() {
	switch c.activeField {
	case fieldR:
		if v, ok := parseByte(c.editR); ok {
			c.working.R = v
		} else {
			c.editR = strconv.Itoa(int(c.working.R))
		}
	case fieldG:
		if v, ok := parseByte(c.editG); ok {
			c.working.G = v
		} else {
			c.editG = strconv.Itoa(int(c.working.G))
		}
	case fieldB:
		if v, ok := parseByte(c.editB); ok {
			c.working.B = v
		} else {
			c.editB = strconv.Itoa(int(c.working.B))
		}
	case fieldA:
		if v, ok := parseByte(c.editA); ok {
			c.working.A = v
		} else {
			c.editA = strconv.Itoa(int(c.working.A))
		}
	case fieldHex:
		if col, err := math.ParseHex(c.editHex); err == nil {
			c.working = col
		} else {
			c.editHex = formatHex(c.working)
		}
	}
	c.activeField = fieldNone
}

// fieldText returns what a field should display: its edit buffer while active,
// otherwise the live working value.
func (c *ColorPickerComponent) fieldText(f int) string {
	switch f {
	case fieldR:
		if c.activeField == fieldR {
			return c.editR
		}
		return strconv.Itoa(int(c.working.R))
	case fieldG:
		if c.activeField == fieldG {
			return c.editG
		}
		return strconv.Itoa(int(c.working.G))
	case fieldB:
		if c.activeField == fieldB {
			return c.editB
		}
		return strconv.Itoa(int(c.working.B))
	case fieldA:
		if c.activeField == fieldA {
			return c.editA
		}
		return strconv.Itoa(int(c.working.A))
	case fieldHex:
		if c.activeField == fieldHex {
			return c.editHex
		}
		return formatHex(c.working)
	}
	return ""
}

// ============================================================================
// Geometry
// ============================================================================

func (c *ColorPickerComponent) headerRect() math.Rect { return c.Rect() }

func (c *ColorPickerComponent) squareSize() float64 {
	if c.SquareSize <= 0 {
		return 88
	}
	return c.SquareSize
}

func (c *ColorPickerComponent) barWidth() float64 {
	if c.BarWidth <= 0 {
		return 14
	}
	return c.BarWidth
}

func (c *ColorPickerComponent) textSize() float64 {
	if c.Size <= 0 {
		return 12
	}
	return c.Size
}

// panelRect returns the open panel rectangle, below the swatch or above it when it
// would overflow the bottom of the screen.
func (c *ColorPickerComponent) panelRect() math.Rect {
	hr := c.headerRect()
	w := c.panelWidth()
	h := c.panelHeight()
	if c.openUp() {
		return math.NewRect(hr.Left(), hr.Top()-h, w, h)
	}
	return math.NewRect(hr.Left(), hr.Bottom(), w, h)
}

func (c *ColorPickerComponent) panelWidth() float64 {
	return 2*cpMargin + c.squareSize() + 2*cpGap + 2*c.barWidth()
}

func (c *ColorPickerComponent) panelHeight() float64 {
	return 2*cpMargin + c.squareSize() + 4*cpGap + cpPrevH + 2*cpFieldH + cpOKH
}

// openUp reports whether the panel should open above the swatch (native-combobox
// placement).
func (c *ColorPickerComponent) openUp() bool {
	if c.viewportH <= 0 {
		return false
	}
	h := c.panelHeight()
	hr := c.headerRect()
	below := c.viewportH - hr.Bottom()
	above := hr.Top()
	if h <= below {
		return false
	}
	if h <= above {
		return true
	}
	return above > below
}

func (c *ColorPickerComponent) squareRect() math.Rect {
	pr := c.panelRect()
	return math.NewRect(pr.Left()+cpMargin, pr.Top()+cpMargin, c.squareSize(), c.squareSize())
}

func (c *ColorPickerComponent) hueRect() math.Rect {
	sq := c.squareRect()
	return math.NewRect(sq.Right()+cpGap, sq.Top(), c.barWidth(), sq.Height())
}

func (c *ColorPickerComponent) alphaRect() math.Rect {
	h := c.hueRect()
	return math.NewRect(h.Right()+cpGap, h.Top(), c.barWidth(), h.Height())
}

func (c *ColorPickerComponent) previewRect() math.Rect {
	sq := c.squareRect()
	pr := c.panelRect()
	return math.NewRect(sq.Left(), sq.Bottom()+cpGap, pr.Width()-2*cpMargin, cpPrevH)
}

func (c *ColorPickerComponent) fieldBoxY() float64 {
	return c.previewRect().Bottom() + cpGap
}

func (c *ColorPickerComponent) fieldRect(i int) math.Rect {
	pr := c.panelRect()
	fw := c.fieldWidth()
	x := pr.Left() + cpMargin + float64(i)*(fw+cpFieldGap)
	return math.NewRect(x, c.fieldBoxY(), fw, cpFieldH)
}

func (c *ColorPickerComponent) fieldWidth() float64 {
	inner := c.panelWidth() - 2*cpMargin
	return (inner - 3*cpFieldGap) / 4
}

func (c *ColorPickerComponent) hexRect() math.Rect {
	pr := c.panelRect()
	y := c.fieldBoxY() + cpFieldH + cpGap
	return math.NewRect(pr.Left()+cpMargin, y, pr.Width()-2*cpMargin, cpFieldH)
}

func (c *ColorPickerComponent) okRect() math.Rect {
	hr := c.hexRect()
	return math.NewRect(hr.Left(), hr.Bottom()+cpGap, hr.Width(), cpOKH)
}

// swatchColorRect returns the swatch's color fill area: the header rect inset by the
// configured padding.
func (c *ColorPickerComponent) swatchColorRect() math.Rect {
	hr := c.headerRect()
	return math.NewRect(
		hr.Left()+c.Padding.Left,
		hr.Top()+c.Padding.Top,
		hr.Width()-c.Padding.Left-c.Padding.Right,
		hr.Height()-c.Padding.Top-c.Padding.Bottom,
	)
}

// ============================================================================
// Draw
// ============================================================================

func (c *ColorPickerComponent) Draw(r core.Renderer) {
	if !c.IsVisible() {
		return
	}
	if _, h := r.GetViewportSize(); h > 0 {
		c.viewportH = float64(h)
	}

	c.drawSwatch(r)
	if c.open {
		c.drawPanel(r)
	}
}

func (c *ColorPickerComponent) drawSwatch(r core.Renderer) {
	hr := c.headerRect()
	if c.Texture != "" {
		core.DrawNineSlice(r, c.Texture, c.Border, hr)
	} else if c.BorderColor.A > 0 {
		r.DrawRect(hr, c.BorderColor)
	}
	cr := c.swatchColorRect()
	c.drawCheckerboard(r, cr, 4)
	r.DrawRect(cr, c.Color)
	c.drawOutline(r, hr)
}

func (c *ColorPickerComponent) drawPanel(r core.Renderer) {
	pr := c.panelRect()
	if c.PanelTexture != "" {
		core.DrawNineSlice(r, c.PanelTexture, c.PanelBorder, pr)
	} else {
		r.DrawRect(pr, c.PanelColor)
	}
	c.drawOutline(r, pr)

	c.drawSquare(r)
	c.drawHueBar(r)
	c.drawAlphaBar(r)
	c.drawPreview(r)
	c.drawFields(r)
	c.drawOK(r)
}

// drawSquare renders the SV square as two layered strip passes: an opaque horizontal
// white→hue sweep, then a transparent→black vertical sweep blended on top, which is
// exactly HSB (corners white / hue / black / black).
func (c *ColorPickerComponent) drawSquare(r core.Renderer) {
	sq := c.squareRect()
	x0 := stdmath.Floor(sq.Left())
	y0 := stdmath.Floor(sq.Top())
	n := c.gradSteps()
	h, _, _ := rgbToHSV(c.working)
	hue := hsvToRGB(h, 1, 1)

	for i := 0; i < n; i++ {
		t := float64(i) / float64(n)
		r.DrawRect(math.NewRect(x0+float64(i), y0, 1, c.squareSize()), math.White.Lerp(hue, t))
	}
	for i := 0; i < n; i++ {
		t := float64(i) / float64(n)
		r.DrawRect(math.NewRect(x0, y0+float64(i), c.squareSize(), 1), math.Black.WithAlphaFloat(t))
	}

	_, s, v := rgbToHSV(c.working)
	c.drawCursor(r, math.NewVector2(sq.Left()+s*sq.Width(), sq.Top()+(1-v)*sq.Height()))
}

// drawHueBar renders the vertical hue spectrum (red at top, spectrum down to red at
// bottom) and its marker.
func (c *ColorPickerComponent) drawHueBar(r core.Renderer) {
	hb := c.hueRect()
	y0 := stdmath.Floor(hb.Top())
	n := c.gradSteps()
	for i := 0; i < n; i++ {
		t := float64(i) / float64(n)
		r.DrawRect(math.NewRect(hb.Left(), y0+float64(i), c.barWidth(), 1), hsvToRGB(360*(1-t), 1, 1))
	}
	h, _, _ := rgbToHSV(c.working)
	c.drawBarMarker(r, hb, hb.Top()+(1-h/360)*hb.Height())
}

// drawAlphaBar renders the current color sweeping from opaque (top) to transparent
// (bottom) over a checkerboard, plus its marker.
func (c *ColorPickerComponent) drawAlphaBar(r core.Renderer) {
	ab := c.alphaRect()
	c.drawCheckerboard(r, ab, 4)
	y0 := stdmath.Floor(ab.Top())
	n := c.gradSteps()
	for i := 0; i < n; i++ {
		t := float64(i) / float64(n)
		r.DrawRect(math.NewRect(ab.Left(), y0+float64(i), c.barWidth(), 1), c.working.WithAlphaFloat(1-t))
	}
	a := float64(c.working.A) / 255
	c.drawBarMarker(r, ab, ab.Top()+(1-a)*ab.Height())
}

// drawPreview draws the committed (old) and working (new) colors side by side.
func (c *ColorPickerComponent) drawPreview(r core.Renderer) {
	pv := c.previewRect()
	half := (pv.Width() - cpFieldGap) / 2
	oldR := math.NewRect(pv.Left(), pv.Top(), half, pv.Height())
	newR := math.NewRect(pv.Left()+half+cpFieldGap, pv.Top(), half, pv.Height())
	c.drawCheckerboard(r, oldR, 4)
	c.drawCheckerboard(r, newR, 4)
	r.DrawRect(oldR, c.Color)
	r.DrawRect(newR, c.working)
	c.drawOutline(r, oldR)
	c.drawOutline(r, newR)
}

func (c *ColorPickerComponent) drawFields(r core.Renderer) {
	labels := [4]string{"R", "G", "B", "A"}
	for i := 0; i < 4; i++ {
		fr := c.fieldRect(i)
		r.DrawRect(fr, c.FieldColor)
		if c.activeField == i {
			r.DrawRectOutline(fr, c.AccentColor, 1)
		}
		r.DrawText(labels[i], c.FontID, 6, math.NewVector2(fr.Left()+3, fr.Center().Y-3), c.labelColor())
		c.drawFieldValue(r, fr, c.fieldText(i))
	}

	hr := c.hexRect()
	r.DrawRect(hr, c.FieldColor)
	if c.activeField == fieldHex {
		r.DrawRectOutline(hr, c.AccentColor, 1)
	}
	r.DrawText("Hex", c.FontID, 6, math.NewVector2(hr.Left()+3, hr.Center().Y-3), c.labelColor())
	c.drawFieldValue(r, hr, c.fieldText(fieldHex))
}

func (c *ColorPickerComponent) drawOK(r core.Renderer) {
	ok := c.okRect()
	col := c.AccentColor
	if c.hoverOK {
		col = c.AccentColor.Lerp(math.White, 0.15)
	}
	r.DrawRect(ok, col)
	tw, th := r.MeasureText("OK", c.FontID, c.textSize())
	r.DrawText("OK", c.FontID, c.textSize(), math.NewVector2(ok.Center().X-tw/2, ok.Center().Y-th/2), c.TextColor)
}

// drawFieldValue centers the given text in the box.
func (c *ColorPickerComponent) drawFieldValue(r core.Renderer, rect math.Rect, text string) {
	if text == "" {
		return
	}
	tw, th := r.MeasureText(text, c.FontID, cpFieldValueSize)
	x := rect.Center().X - tw/2
	y := rect.Center().Y - th/2
	r.DrawText(text, c.FontID, cpFieldValueSize, math.NewVector2(x, y), c.TextColor)
}

func (c *ColorPickerComponent) drawOutline(r core.Renderer, rect math.Rect) {
	if c.BorderThickness > 0 && c.BorderColor.A > 0 {
		r.DrawRectOutline(rect, c.BorderColor, c.BorderThickness)
	}
}

func (c *ColorPickerComponent) drawCursor(r core.Renderer, center math.Vector2) {
	r.DrawCircleOutline(center, 5, math.Black, 2)
	r.DrawCircleOutline(center, 5, c.MarkerColor, 1)
}

func (c *ColorPickerComponent) drawBarMarker(r core.Renderer, bar math.Rect, y float64) {
	y = clampf(y, bar.Top(), bar.Bottom())
	r.DrawLine(math.NewVector2(bar.Left(), y), math.NewVector2(bar.Right(), y), math.Black, 3)
	r.DrawLine(math.NewVector2(bar.Left(), y), math.NewVector2(bar.Right(), y), c.MarkerColor, 1)
}

// drawCheckerboard fills rect with a two-tone checker of the given cell size, so
// partial transparency reads correctly.
func (c *ColorPickerComponent) drawCheckerboard(r core.Renderer, rect math.Rect, cell float64) {
	if rect.Width() <= 0 || rect.Height() <= 0 || cell <= 0 {
		return
	}
	c1 := math.NewColor(80, 80, 80, 255)
	c2 := math.NewColor(140, 140, 140, 255)
	x0, y0 := rect.Left(), rect.Top()
	for y := 0; y0+float64(y)*cell < rect.Bottom(); y++ {
		cy := y0 + float64(y)*cell
		for x := 0; x0+float64(x)*cell < rect.Right(); x++ {
			cx := x0 + float64(x)*cell
			col := c1
			if (x+y)%2 == 1 {
				col = c2
			}
			cw := cell
			if cx+cw > rect.Right() {
				cw = rect.Right() - cx
			}
			ch := cell
			if cy+ch > rect.Bottom() {
				ch = rect.Bottom() - cy
			}
			r.DrawRect(math.NewRect(cx, cy, cw, ch), col)
		}
	}
}

func (c *ColorPickerComponent) labelColor() math.Color { return c.TextColor.Scale(0.6) }

// gradSteps is the number of 1-logical-unit gradient strips (the square and bars are
// all squareSize tall).
func (c *ColorPickerComponent) gradSteps() int {
	n := int(stdmath.Ceil(c.squareSize()))
	if n < 2 {
		return 2
	}
	return n
}

// ============================================================================
// Color helpers
// ============================================================================

// rgbToHSV converts a color to hue [0,360), saturation [0,1], value [0,1].
func rgbToHSV(c math.Color) (h, s, v float64) {
	r := float64(c.R) / 255
	g := float64(c.G) / 255
	b := float64(c.B) / 255
	max := stdmath.Max(r, stdmath.Max(g, b))
	min := stdmath.Min(r, stdmath.Min(g, b))
	d := max - min

	v = max
	if max == 0 {
		s = 0
	} else {
		s = d / max
	}
	if d == 0 {
		h = 0
	} else {
		switch max {
		case r:
			h = 60 * ((g - b) / d)
		case g:
			h = 60*((b-r)/d) + 120
		default:
			h = 60*((r-g)/d) + 240
		}
		if h < 0 {
			h += 360
		}
	}
	return h, s, v
}

// hsvToRGB converts hue [0,360), saturation [0,1], value [0,1] to an opaque color.
func hsvToRGB(h, s, v float64) math.Color {
	h = stdmath.Mod(h, 360)
	if h < 0 {
		h += 360
	}
	cc := v * s
	x := cc * (1 - stdmath.Abs(stdmath.Mod(h/60, 2)-1))
	m := v - cc

	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = cc, x, 0
	case h < 120:
		r, g, b = x, cc, 0
	case h < 180:
		r, g, b = 0, cc, x
	case h < 240:
		r, g, b = 0, x, cc
	case h < 300:
		r, g, b = x, 0, cc
	default:
		r, g, b = cc, 0, x
	}
	return math.NewColorFromFloats(r+m, g+m, b+m, 1)
}

// formatHex returns "#RRGGBB" when opaque and "#RRGGBBAA" otherwise.
func formatHex(c math.Color) string {
	if c.A == 255 {
		return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
	}
	return fmt.Sprintf("#%02X%02X%02X%02X", c.R, c.G, c.B, c.A)
}

// parseByte parses a 0-255 decimal string, clamping out-of-range values.
func parseByte(s string) (uint8, bool) {
	if s == "" {
		return 0, false
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return 0, false
	}
	if n < 0 {
		n = 0
	}
	if n > 255 {
		n = 255
	}
	return uint8(n), true
}

func isHexRune(r rune) bool {
	return (r >= '0' && r <= '9') || (r >= 'a' && r <= 'f') || (r >= 'A' && r <= 'F')
}

func clampf(v, lo, hi float64) float64 {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
