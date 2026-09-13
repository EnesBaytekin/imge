package components

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// tooltip is the single shared tooltip the editor displays over symbol-only buttons
// (add/remove/duplicate object, scene and tag controls, component controls, …). A
// panel sets it during Update when the cursor hovers one of its buttons (see
// showTooltip); the TooltipComponent draws it last (top layer) and clears it at the end
// of its Draw, so it vanishes the frame after the cursor leaves.
type tooltipState struct {
	text   string
	anchor math.Vector2 // cursor position, in screen (UI) space
	active bool
}

var tooltip tooltipState

// showTooltip records a tooltip for this frame, anchored near the cursor. It overrides
// any earlier call this frame, so the topmost hovered control wins.
func showTooltip(text string, anchor math.Vector2) {
	tooltip.text = text
	tooltip.anchor = anchor
	tooltip.active = true
}

// TooltipComponent draws the current tooltip (if any) on top of every other panel. It
// is a transparent, non-blocking surface: it never swallows pointer events, so the
// cursor still reaches the button beneath it. Its own rect/transform are unused — it
// positions the tooltip from the recorded anchor and the live viewport size.
type TooltipComponent struct {
	core.BaseUIComponent

	Background  math.Color `json:"background"`
	BorderColor math.Color `json:"border_color"`
	TextColor   math.Color `json:"text_color"`

	FontID   string  `json:"font_id"`
	FontSize float64 `json:"font_size"`

	// TooltipPad is the space between the text and the panel edge.
	TooltipPad float64 `json:"tooltip_pad"`
	// AnchorOffset is the gap between the cursor and the panel's top-left corner.
	AnchorOffset float64 `json:"anchor_offset"`
}

func (t *TooltipComponent) Initialize() {
	if t.Background == (math.Color{}) {
		t.Background = math.NewColor(0x2a, 0x2f, 0x3e, 0xf5)
	}
	if t.BorderColor == (math.Color{}) {
		t.BorderColor = math.NewColor(0x4a, 0x55, 0x70, 0xff)
	}
	if t.TextColor == (math.Color{}) {
		t.TextColor = math.NewColor(0xf0, 0xf0, 0xf5, 0xff)
	}
	if t.FontSize <= 0 {
		t.FontSize = 6
	}
	if t.TooltipPad <= 0 {
		t.TooltipPad = 5
	}
	if t.AnchorOffset <= 0 {
		t.AnchorOffset = 12
	}
}

func (t *TooltipComponent) Draw(r core.Renderer) {
	// The tooltip is single-shot: draw it now, then clear it so it never lingers past
	// the frame the cursor was over a button.
	defer func() { tooltip = tooltipState{} }()
	if !tooltip.active || tooltip.text == "" {
		return
	}

	vw, vh := r.GetViewportSize()
	fw, fh := float64(vw), float64(vh)
	if fw <= 0 || fh <= 0 {
		return
	}

	tw, th := r.MeasureText(tooltip.text, t.FontID, t.FontSize)
	pad := t.TooltipPad
	w := tw + pad*2
	h := th + pad*2

	// Below-right of the cursor by default; flip above when it would clip the bottom
	// edge, and clamp horizontally so the panel always stays on screen.
	x := tooltip.anchor.X + t.AnchorOffset
	y := tooltip.anchor.Y + t.AnchorOffset
	if x+w > fw-2 {
		x = fw - w - 2
	}
	if y+h > fh-2 {
		y = tooltip.anchor.Y - h - 4
	}
	if x < 2 {
		x = 2
	}
	if y < 2 {
		y = 2
	}

	rect := math.NewRect(x, y, w, h)
	r.SetClipRect(math.NewRect(0, 0, fw, fh))
	r.DrawRect(rect, t.Background)
	r.DrawRectOutline(rect, t.BorderColor, 1)
	r.DrawText(tooltip.text, t.FontID, t.FontSize, math.NewVector2(x+pad, y+pad), t.TextColor)
	r.ClearClip()
}
