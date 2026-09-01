package core

import (
	"testing"

	"github.com/EnesBaytekin/imge/core/math"
)

// debugTestComponent implements DebugDrawer and DebugBoundsProvider, recording each
// DrawDebug call so tests can assert the overlay pass ran (and with which Selected).
type debugTestComponent struct {
	BaseComponent
	calls []DebugInfo
}

func (c *debugTestComponent) DrawDebug(r Renderer, info DebugInfo) {
	c.calls = append(c.calls, info)
}

func (c *debugTestComponent) DebugBounds() math.Rect {
	return math.NewRect(0, 0, 10, 10)
}

// nopRenderer satisfies Renderer with no-ops, enough for Scene.Draw to run.
type nopRenderer struct{}

func (nopRenderer) Clear(math.Color)                                             {}
func (nopRenderer) DrawRect(math.Rect, math.Color)                               {}
func (nopRenderer) DrawRectOutline(math.Rect, math.Color, float64)               {}
func (nopRenderer) DrawCircle(math.Vector2, float64, math.Color)                 {}
func (nopRenderer) DrawCircleOutline(math.Vector2, float64, math.Color, float64) {}
func (nopRenderer) DrawLine(math.Vector2, math.Vector2, math.Color, float64)     {}
func (nopRenderer) DrawTexture(string, math.Rect, math.Vector2, math.Vector2, float64, math.ColorTransform) {
}
func (nopRenderer) GetTextureSize(string) (float64, float64) { return 0, 0 }
func (nopRenderer) DrawText(string, string, float64, math.Vector2, math.Color) {
}
func (nopRenderer) MeasureText(string, string, float64) (float64, float64) { return 0, 0 }
func (nopRenderer) DrawTextWrapped(string, string, float64, float64, WrapMode, bool, math.Vector2, math.Color) {
}
func (nopRenderer) MeasureTextWrapped(string, string, float64, float64, WrapMode, bool) (float64, float64) {
	return 0, 0
}
func (nopRenderer) SetCamera(float64, float64, float64) {}
func (nopRenderer) Present()                             {}
func (nopRenderer) SetViewport(int, int)                 {}
func (nopRenderer) GetViewportSize() (int, int)          { return 0, 0 }
func (nopRenderer) SetClipRect(math.Rect)                {}
func (nopRenderer) ClearClip()                           {}

func newDebugScene() (*Scene, *debugTestComponent) {
	scene := NewScene("test")
	obj := NewObject("obj")
	comp := &debugTestComponent{}
	comp.SetName("dbg")
	_ = obj.AddComponent(comp)
	_ = scene.AddObject(obj)
	return scene, comp
}

func TestDebugOverlayDisabledByDefault(t *testing.T) {
	scene, comp := newDebugScene()
	scene.Draw(nopRenderer{})
	if len(comp.calls) != 0 {
		t.Fatalf("DrawDebug called %d times with debug off, want 0", len(comp.calls))
	}
}

func TestDebugOverlayRunsWhenEnabled(t *testing.T) {
	scene, comp := newDebugScene()
	scene.SetDebugDraw(true)
	scene.Draw(nopRenderer{})
	if len(comp.calls) != 1 {
		t.Fatalf("DrawDebug called %d times, want 1", len(comp.calls))
	}
	if comp.calls[0].Selected {
		t.Fatal("unselected component reported Selected=true")
	}
}

func TestDebugOverlayMarksSelection(t *testing.T) {
	scene, comp := newDebugScene()
	scene.SetDebugDraw(true)
	scene.SetDebugSelection(comp)
	scene.Draw(nopRenderer{})
	if len(comp.calls) != 1 || !comp.calls[0].Selected {
		t.Fatalf("selected component not marked: calls=%v", comp.calls)
	}
}

func TestDebugOverlayClearsSelection(t *testing.T) {
	scene, comp := newDebugScene()
	scene.SetDebugDraw(true)
	scene.SetDebugSelection(comp)
	scene.SetDebugSelection(nil)
	scene.Draw(nopRenderer{})
	if len(comp.calls) != 1 || comp.calls[0].Selected {
		t.Fatalf("cleared selection still marked selected: calls=%v", comp.calls)
	}
}
