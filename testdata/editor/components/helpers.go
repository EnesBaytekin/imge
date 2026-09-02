// Package components holds the editor's UI components. This file is a plain
// helper file: it declares no component struct, only the free functions,
// constants, types, and variables the component files share. The build tool
// copies it into the generated `components` package verbatim, and its codegen
// passes over it (it contributes no component kind) — see build/registry.go.
package components

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// ============================================================================
// Scrollbar helpers (shared by the editor panels that scroll overflowing content).
// ============================================================================

// There is no engine-level scrollbar primitive (the @Slider is a @UIManager widget;
// the editor panels read input directly), so the track/thumb geometry and drag math
// live here and the panels own their scroll offset. A panel shows the scrollbar only
// when its content is taller than the track (contentH > track.Height()).

// scrollThumbH returns the scrollbar thumb height for a track of the given height
// showing contentH pixels of content, or 0 when the content fits (no scrollbar).
func scrollThumbH(trackH, contentH float64) float64 {
	if contentH <= trackH {
		return 0
	}
	h := trackH * trackH / contentH
	if h < 12 {
		h = 12
	}
	if h > trackH {
		h = trackH
	}
	return h
}

// scrollThumb returns the thumb rect within track for the current scroll offset, plus
// whether a scrollbar is needed. scroll and maxScroll are in content pixels.
func scrollThumb(track math.Rect, contentH, scroll, maxScroll float64) (math.Rect, bool) {
	thumbH := scrollThumbH(track.Height(), contentH)
	if thumbH == 0 || maxScroll <= 0 {
		return math.Rect{}, false
	}
	t := scroll / maxScroll
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	travel := track.Height() - thumbH
	return math.NewRect(track.X(), track.Y()+t*travel, track.Width(), thumbH), true
}

// scrollFromThumb maps a mouse Y (with a grab offset within the thumb) to the scroll
// offset that puts the thumb there, clamped to [0, maxScroll].
func scrollFromThumb(track math.Rect, contentH, maxScroll, mouseY, grab float64) float64 {
	thumbH := scrollThumbH(track.Height(), contentH)
	travel := track.Height() - thumbH
	if travel <= 0 || maxScroll <= 0 {
		return 0
	}
	t := (mouseY - grab - track.Y()) / travel
	if t < 0 {
		t = 0
	}
	if t > 1 {
		t = 1
	}
	return t * maxScroll
}

// drawScrollbar draws a scrollbar track and thumb.
func drawScrollbar(r core.Renderer, track math.Rect, thumb math.Rect, trackColor, thumbColor math.Color) {
	r.DrawRect(track, trackColor)
	r.DrawRect(thumb, thumbColor)
}

// ============================================================================
// Reflection helpers: the runtime argument schema of a component.
// ============================================================================

// argField describes one component argument discovered by reflection.
type argField struct {
	name     string // json tag
	field    reflect.StructField
	value    reflect.Value
	editable bool
}

// enumerateArgs returns the exported, json-tagged fields of a component (including
// those promoted from the embedded BaseComponent/BaseUIComponent) in declaration
// order. It is the schema the args window renders and edits.
func enumerateArgs(comp core.Component) []argField {
	if comp == nil {
		return nil
	}
	v := reflect.ValueOf(comp)
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil
		}
		v = v.Elem()
	}
	t := v.Type()
	var out []argField
	for _, f := range reflect.VisibleFields(t) {
		if f.Anonymous {
			continue // the embedded base struct itself; its tagged fields are promoted below
		}
		tag := strings.Split(f.Tag.Get("json"), ",")[0]
		if tag == "" || tag == "-" {
			continue
		}
		fv := v.FieldByIndex(f.Index)
		if !fv.CanInterface() || !fv.CanSet() {
			continue
		}
		out = append(out, argField{
			name:     tag,
			field:    f,
			value:    fv,
			editable: isEditable(fv),
		})
	}
	return out
}

// isEditable reports whether a field can be parsed from a string. Pointers (e.g.
// *bool Visible/Enabled) and unknown structs/slices/maps are read-only for now.
func isEditable(v reflect.Value) bool {
	if v.Kind() == reflect.Ptr {
		return false
	}
	switch v.Kind() {
	case reflect.String, reflect.Bool,
		reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return true
	}
	t := v.Type()
	return t == reflect.TypeOf(math.Color{}) ||
		t == reflect.TypeOf(math.Vector2{}) ||
		t == reflect.TypeOf(math.Border{})
}

// formatArg renders a field's current value as the text shown (and seeded into) the
// value column.
func formatArg(v reflect.Value) string {
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return "<unset>"
		}
		return formatArg(v.Elem())
	}
	switch v.Kind() {
	case reflect.String:
		return v.String()
	case reflect.Bool:
		return strconv.FormatBool(v.Bool())
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(v.Int(), 10)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(v.Uint(), 10)
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(v.Float(), 'g', -1, 64)
	}
	t := v.Type()
	switch {
	case t == reflect.TypeOf(math.Color{}):
		return formatColorHex(v.Interface().(math.Color))
	case t == reflect.TypeOf(math.Vector2{}):
		vec := v.Interface().(math.Vector2)
		return fmt.Sprintf("%g, %g", vec.X, vec.Y)
	case t == reflect.TypeOf(math.Border{}):
		b := v.Interface().(math.Border)
		return fmt.Sprintf("%g, %g, %g, %g", b.Left, b.Top, b.Right, b.Bottom)
	}
	return fmt.Sprintf("%v", v.Interface())
}

// setArg parses a string and writes it into a field. It is the inverse of formatArg.
func setArg(v reflect.Value, s string) error {
	if v.Kind() == reflect.Ptr {
		return fmt.Errorf("pointer fields are read-only")
	}
	t := v.Type()
	switch {
	case t == reflect.TypeOf(math.Color{}):
		c, err := math.ParseHex(s)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(c))
		return nil
	case t == reflect.TypeOf(math.Vector2{}):
		x, y, err := parseTwoFloats(s)
		if err != nil {
			return err
		}
		v.Set(reflect.ValueOf(math.NewVector2(x, y)))
		return nil
	case t == reflect.TypeOf(math.Border{}):
		parts, err := parseFloats(s)
		if err != nil {
			return err
		}
		if len(parts) != 4 {
			return fmt.Errorf("border needs 4 numbers (left, top, right, bottom)")
		}
		v.Set(reflect.ValueOf(math.Border{Left: parts[0], Top: parts[1], Right: parts[2], Bottom: parts[3]}))
		return nil
	}
	switch v.Kind() {
	case reflect.String:
		v.SetString(s)
		return nil
	case reflect.Bool:
		b, err := parseBool(s)
		if err != nil {
			return err
		}
		v.SetBool(b)
		return nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		n, err := strconv.ParseInt(strings.TrimSpace(s), 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetInt(n)
		return nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		n, err := strconv.ParseUint(strings.TrimSpace(s), 10, v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetUint(n)
		return nil
	case reflect.Float32, reflect.Float64:
		f, err := strconv.ParseFloat(strings.TrimSpace(s), v.Type().Bits())
		if err != nil {
			return err
		}
		v.SetFloat(f)
		return nil
	}
	return fmt.Errorf("unsupported field type %s", v.Type())
}

// formatColorHex renders a color as #RRGGBB (opaque) or #RRGGBBAA.
func formatColorHex(c math.Color) string {
	if c.A == 255 {
		return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
	}
	return fmt.Sprintf("#%02X%02X%02X%02X", c.R, c.G, c.B, c.A)
}

// parseTwoFloats parses a "x, y" string.
func parseTwoFloats(s string) (float64, float64, error) {
	parts, err := parseFloats(s)
	if err != nil {
		return 0, 0, err
	}
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("expected two numbers (x, y)")
	}
	return parts[0], parts[1], nil
}

// parseFloats splits a string on commas and parses each token as a float. A malformed
// token fails the whole parse rather than being silently skipped.
func parseFloats(s string) ([]float64, error) {
	var out []float64
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			return nil, fmt.Errorf("empty value in %q", s)
		}
		f, err := strconv.ParseFloat(part, 64)
		if err != nil {
			return nil, fmt.Errorf("bad number %q", part)
		}
		out = append(out, f)
	}
	return out, nil
}

// parseFloat parses a single float.
func parseFloat(s string) (float64, error) {
	return strconv.ParseFloat(strings.TrimSpace(s), 64)
}

// formatFloat renders a float without trailing zeros, so it round-trips through
// parseFloat exactly.
func formatFloat(v float64) string {
	return strconv.FormatFloat(v, 'g', -1, 64)
}

// parseBool accepts a set of true/false spellings, shared by the object-property
// inspector and the component-args reflection writer.
func parseBool(s string) (bool, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "true", "1", "yes", "on":
		return true, nil
	case "false", "0", "no", "off":
		return false, nil
	}
	return false, fmt.Errorf("expected true/false")
}

// ============================================================================
// Shared field-widget plumbing (used by the args window and the inspector).
// ============================================================================

// fieldKind selects which engine widget backs an editable field.
type fieldKind int

const (
	kindText fieldKind = iota
	kindCheck
	kindColor
)

// fieldWidget is the common surface of the engine widgets a field value uses: enough
// to attach, position, show/hide, and clip them, with the rest reached by type
// assertion.
type fieldWidget interface {
	core.Component
	SetOffset(math.Vector2)
	SetVisible(bool)
	SetClipRect(math.Rect)
	ClearClipRect()
}

// fieldBinding ties one editable field to its engine widget: how to read/write the
// model (get/apply, string round-trip), the typed getter for live-sync refresh, and
// the widget component itself. Everything commits through commitString so undo stays a
// single string round-trip regardless of widget type.
type fieldBinding struct {
	key        string
	row        int // index into the field list (for scroll positioning)
	col        int // part index within the row (0 = first widget)
	parts      int // number of widgets sharing the row's value column
	kind       fieldKind
	get        func() string
	apply      func(string) error
	getBool    func() bool
	getColor   func() math.Color
	widget     fieldWidget
	old        string // last committed value
	wasFocused bool   // TextInput blur tracking
}

// makeFieldWidget creates the engine widget for a binding, attaches it to the window
// object, positions it, and seeds its current value. The widget is initialized
// manually: the object's own Initialize already ran, so AddComponent will not call it.
func makeFieldWidget(b *fieldBinding, owner *core.Object, pos math.Vector2, valueW, h float64, fontID string, size float64, valueText math.Color) fieldWidget {
	var comp fieldWidget
	switch b.kind {
	case kindCheck:
		cb := &CheckBoxComponent{}
		cb.Text = "" // the host draws the field name label
		cb.BoxSize = h
		cb.FontID = fontID
		cb.Size = size
		cb.TextColor = valueText
		cb.Width = h
		cb.Height = h
		cb.DrawLayer = 1
		comp = cb
	case kindColor:
		cp := &ColorPickerComponent{}
		cp.FontID = fontID
		cp.Size = size
		cp.TextColor = valueText
		cp.Width = h
		cp.Height = h
		cp.DrawLayer = 1
		comp = cp
	default: // kindText
		ti := &TextInputComponent{}
		ti.FontID = fontID
		ti.Size = size
		ti.TextColor = valueText
		ti.BackgroundColor = fieldBackground
		ti.OutlineColor = fieldOutline
		ti.OutlineThickness = 1
		ti.Width = valueW
		ti.Height = h
		ti.DrawLayer = 1
		comp = ti
	}
	comp.SetName("val_" + b.key)
	comp.SetOffset(pos.Subtract(owner.Transform.Position))
	if err := owner.AddComponent(comp); err != nil {
		return nil
	}
	comp.Initialize()
	// Seed the value AFTER Initialize: ColorPicker resets a zero color to white, so a
	// transparent #00000000 must be applied here, not before.
	switch b.kind {
	case kindText:
		comp.(*TextInputComponent).Text = b.get()
	case kindCheck:
		comp.(*CheckBoxComponent).SetChecked(b.getBool())
	case kindColor:
		comp.(*ColorPickerComponent).SetColor(b.getColor())
	}
	return comp
}

// commitString applies a committed widget value through the binding's string round-trip
// and records one undo entry when it changed. It returns the parse error (TextInput
// only; bool/color cannot fail).
func commitString(b *fieldBinding, s string) error {
	if s == b.old {
		return nil
	}
	old := b.old
	if err := b.apply(s); err != nil {
		return err
	}
	history.record(
		func() { _ = b.apply(old) },
		func() { _ = b.apply(s) },
	)
	b.old = s
	return nil
}

// pollCommits detects committed widget changes each frame — a CheckBox toggle, a
// ColorPicker commit, a TextInput Enter or blur — and records one undo entry per
// change. TextInput parse failures tint the box error-red and keep focus (Enter) or
// revert the text (blur).
func pollCommits(bindings []fieldBinding, ctx *core.Context, valueText, errorColor math.Color) {
	for i := range bindings {
		b := &bindings[i]
		switch b.kind {
		case kindCheck:
			cb := b.widget.(*CheckBoxComponent)
			_ = commitString(b, strconv.FormatBool(cb.GetChecked()))
		case kindColor:
			cp := b.widget.(*ColorPickerComponent)
			_ = commitString(b, formatColorHex(cp.GetColor()))
		default:
			ti := b.widget.(*TextInputComponent)
			focused := ti.IsFocused()
			if focused && ctx.Input.IsKeyJustPressed(core.KeyEnter) {
				if err := commitString(b, ti.Text); err != nil {
					ti.TextColor = errorColor
				} else {
					ti.TextColor = valueText
				}
			}
			if b.wasFocused && !focused {
				if err := commitString(b, ti.Text); err != nil {
					ti.Text = b.get() // revert on blur-error
				}
				ti.TextColor = valueText
			}
			b.wasFocused = focused
		}
	}
}

// refreshWidgets live-syncs each binding's widget to the current model value, skipping
// a focused TextInput so an in-progress edit is never overwritten. SetChecked/SetColor
// are silent; SetColor's `if !open` guard keeps an open picker panel's working color.
func refreshWidgets(bindings []fieldBinding) {
	for i := range bindings {
		b := &bindings[i]
		switch b.kind {
		case kindText:
			ti := b.widget.(*TextInputComponent)
			if ti.IsFocused() {
				continue
			}
			ti.Text = b.get()
		case kindCheck:
			b.widget.(*CheckBoxComponent).SetChecked(b.getBool())
		case kindColor:
			b.widget.(*ColorPickerComponent).SetColor(b.getColor())
		}
	}
}

// syncBindings runs the per-frame widget pass — commit polling then live-sync — for a
// host's bindings. The order matters: a TextInput blur must commit BEFORE its text is
// resynced from the model, or the edit is silently overwritten.
func syncBindings(bindings []fieldBinding, ctx *core.Context, valueText, errorColor math.Color) {
	pollCommits(bindings, ctx, valueText, errorColor)
	refreshWidgets(bindings)
}

// fieldPartGap is the horizontal gap between side-by-side part widgets of a split
// field (e.g. a Vector2's x and y boxes).
const fieldPartGap = 4.0

// fieldBackground / fieldOutline are the TextInput value-box chrome: a dark inset fill
// with a thin border, so text boxes read as clearly-clickable fields against the window.
var (
	fieldBackground = math.NewColor(0x10, 0x13, 0x1c, 0xff)
	fieldOutline    = math.NewColor(0x3a, 0x42, 0x57, 0xff)
)

// partWidth is the width of one part widget within a row's value column.
func partWidth(fullW float64, parts int) float64 {
	if parts <= 1 {
		return fullW
	}
	return (fullW - float64(parts-1)*fieldPartGap) / float64(parts)
}

// partX is the left x of one part widget within a row's value column.
func partX(valX float64, col int, partW float64) float64 {
	return valX + float64(col)*(partW+fieldPartGap)
}

// layoutWidgets positions each binding's widget at its row (and part slot). Rows that
// scrolled fully out of the body are hidden; rows only partly visible stay visible and
// are clipped to the body, so their overflowing part is cut off instead of blinking out
// whole — a realistic scroll feel.
func layoutWidgets(bindings []fieldBinding, owner *core.Object, bodyTop, valX, valW, scroll, rowHeight, bodyBottom float64) {
	// The value column's body clip: every widget draws only inside this region.
	clip := math.NewRect(valX, bodyTop, valW, bodyBottom-bodyTop)
	for i := range bindings {
		b := &bindings[i]
		if b.widget == nil {
			continue
		}
		y := bodyTop + float64(b.row)*rowHeight - scroll
		if y+rowHeight <= bodyTop || y >= bodyBottom {
			b.widget.SetVisible(false)
			b.widget.ClearClipRect()
			continue
		}
		b.widget.SetVisible(true)
		pw := partWidth(valW, b.parts)
		x := partX(valX, b.col, pw)
		b.widget.SetOffset(math.NewVector2(x, y).Subtract(owner.Transform.Position))
		b.widget.SetClipRect(clip)
	}
}

// removeWidgets detaches each binding's widget from the owner. RemoveComponent is
// synchronous and unsubscribes events, so it is safe to call mid-frame.
func removeWidgets(bindings []fieldBinding, owner *core.Object) {
	for i := range bindings {
		if bindings[i].widget != nil {
			owner.RemoveComponent(bindings[i].widget.GetName())
		}
	}
}

// lookupUIManager resolves the editor's @UIManager by object name. It is the source of
// truth for whether any managed widget holds keyboard focus.
func lookupUIManager(scene *core.Scene) *UIManagerComponent {
	if scene == nil {
		return nil
	}
	if obj := scene.GetObjectByName("ui_root"); obj != nil {
		return core.GetFrom[*UIManagerComponent](obj)
	}
	return nil
}

// raiseToFront brings obj to the front of its layer via the @UIManager, so a window can
// be raised programmatically (e.g. when opened). Clicking a window is raised by the
// manager itself (its blocking surface is the pointer target), so panels no longer call
// this on click. A no-op when the manager is absent or auto-raise is off.
func raiseToFront(scene *core.Scene, obj *core.Object) {
	if mgr := lookupUIManager(scene); mgr != nil {
		mgr.RaiseToFront(obj)
	}
}

// pointerOwnedElsewhere reports whether a UI object other than owner is the topmost
// pointer target at pos, so a custom panel that reads ctx.Input directly should yield.
// It is the panel-side half of the @UIManager's blocking occlusion: a blocking panel is
// the exclusive target while it is topmost, and yields to any window drawn above it —
// the same rule managed widgets follow for free.
func pointerOwnedElsewhere(scene *core.Scene, owner *core.Object, pos math.Vector2) bool {
	mgr := lookupUIManager(scene)
	if mgr == nil {
		return false
	}
	top := mgr.TopmostObjectAt(pos)
	return top != nil && top != owner
}

// ============================================================================
// Undo/redo history (committed field edits).
// ============================================================================

// editStep is one reversible edit: an undo closure (restore the prior state) and a
// redo closure (re-apply the new state). The closures capture the target object/field
// and its old/new values at record time, so an entry stays correct no matter how the
// editor's UI state changes afterward. This one shape lets every edit source share a
// single history: component-arg writes, object-property edits, and viewport drags.
type editStep struct {
	undo func()
	redo func()
}

// editHistory is the editor-wide undo stack. It lives at package level so any panel can
// record edits and the toolbar triggers undo/redo by shortcut.
type editHistory struct {
	undoStack []editStep
	redoStack []editStep
}

var history editHistory

// maxHistory bounds the undo stack so a long editing session can't grow unbounded.
const maxHistory = 100

// record pushes a reversible edit and clears the redo stack (a fresh edit invalidates
// the redo chain, matching every editor).
func (h *editHistory) record(undo, redo func()) {
	h.undoStack = append(h.undoStack, editStep{undo, redo})
	if len(h.undoStack) > maxHistory {
		h.undoStack = h.undoStack[len(h.undoStack)-maxHistory:]
	}
	h.redoStack = h.redoStack[:0]
}

// clear drops all history. Called when the target project switches, since undo entries
// reference the previous project's live objects and must not apply across documents.
func (h *editHistory) clear() {
	h.undoStack = nil
	h.redoStack = nil
}

// undo reverts the most recent edit and moves it to the redo stack. It returns false
// when there is nothing to undo.
func (h *editHistory) undo() bool {
	if len(h.undoStack) == 0 {
		return false
	}
	step := h.undoStack[len(h.undoStack)-1]
	h.undoStack = h.undoStack[:len(h.undoStack)-1]
	step.undo()
	h.redoStack = append(h.redoStack, step)
	return true
}

// redo re-applies the most recently undone edit and moves it back to the undo stack.
func (h *editHistory) redo() bool {
	if len(h.redoStack) == 0 {
		return false
	}
	step := h.redoStack[len(h.redoStack)-1]
	h.redoStack = h.redoStack[:len(h.redoStack)-1]
	step.redo()
	h.undoStack = append(h.undoStack, step)
	return true
}
