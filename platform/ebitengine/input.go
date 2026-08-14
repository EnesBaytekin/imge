package ebitengine

import (
	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// Input implements core.Input using Ebitengine's input API.
type Input struct {
	mousePosition math.Vector2
	prevMouse     math.Vector2
	mouseDelta    math.Vector2
	mouseScroll   math.Vector2
}

func newInput() *Input {
	return &Input{}
}

// keyMap maps engine key codes to Ebitengine keys.
var keyMap = map[core.KeyCode]ebiten.Key{
	core.KeyA: ebiten.KeyA, core.KeyB: ebiten.KeyB, core.KeyC: ebiten.KeyC,
	core.KeyD: ebiten.KeyD, core.KeyE: ebiten.KeyE, core.KeyF: ebiten.KeyF,
	core.KeyG: ebiten.KeyG, core.KeyH: ebiten.KeyH, core.KeyI: ebiten.KeyI,
	core.KeyJ: ebiten.KeyJ, core.KeyK: ebiten.KeyK, core.KeyL: ebiten.KeyL,
	core.KeyM: ebiten.KeyM, core.KeyN: ebiten.KeyN, core.KeyO: ebiten.KeyO,
	core.KeyP: ebiten.KeyP, core.KeyQ: ebiten.KeyQ, core.KeyR: ebiten.KeyR,
	core.KeyS: ebiten.KeyS, core.KeyT: ebiten.KeyT, core.KeyU: ebiten.KeyU,
	core.KeyV: ebiten.KeyV, core.KeyW: ebiten.KeyW, core.KeyX: ebiten.KeyX,
	core.KeyY: ebiten.KeyY, core.KeyZ: ebiten.KeyZ,

	core.Key0: ebiten.Key0, core.Key1: ebiten.Key1, core.Key2: ebiten.Key2,
	core.Key3: ebiten.Key3, core.Key4: ebiten.Key4, core.Key5: ebiten.Key5,
	core.Key6: ebiten.Key6, core.Key7: ebiten.Key7, core.Key8: ebiten.Key8,
	core.Key9: ebiten.Key9,

	core.KeySpace:     ebiten.KeySpace,
	core.KeyEnter:     ebiten.KeyEnter,
	core.KeyEscape:    ebiten.KeyEscape,
	core.KeyBackspace: ebiten.KeyBackspace,
	core.KeyTab:       ebiten.KeyTab,
	core.KeyShift:     ebiten.KeyShift,
	core.KeyControl:   ebiten.KeyControl,
	core.KeyAlt:       ebiten.KeyAlt,
	core.KeyLeft:      ebiten.KeyArrowLeft,
	core.KeyRight:     ebiten.KeyArrowRight,
	core.KeyUp:        ebiten.KeyArrowUp,
	core.KeyDown:      ebiten.KeyArrowDown,

	core.KeyF1: ebiten.KeyF1, core.KeyF2: ebiten.KeyF2, core.KeyF3: ebiten.KeyF3,
	core.KeyF4: ebiten.KeyF4, core.KeyF5: ebiten.KeyF5, core.KeyF6: ebiten.KeyF6,
	core.KeyF7: ebiten.KeyF7, core.KeyF8: ebiten.KeyF8, core.KeyF9: ebiten.KeyF9,
	core.KeyF10: ebiten.KeyF10, core.KeyF11: ebiten.KeyF11, core.KeyF12: ebiten.KeyF12,
}

// mouseMap maps engine mouse buttons to Ebitengine mouse buttons.
var mouseMap = map[core.MouseButton]ebiten.MouseButton{
	core.MouseButtonLeft:   ebiten.MouseButtonLeft,
	core.MouseButtonRight:  ebiten.MouseButtonRight,
	core.MouseButtonMiddle: ebiten.MouseButtonMiddle,
	core.MouseButton4:      ebiten.MouseButton3,
	core.MouseButton5:      ebiten.MouseButton4,
}

// IsKeyPressed reports whether the key is currently held.
func (i *Input) IsKeyPressed(key core.KeyCode) bool {
	return ebiten.IsKeyPressed(keyMap[key])
}

// IsKeyJustPressed reports whether the key was pressed this frame.
func (i *Input) IsKeyJustPressed(key core.KeyCode) bool {
	return inpututil.IsKeyJustPressed(keyMap[key])
}

// IsKeyJustReleased reports whether the key was released this frame.
func (i *Input) IsKeyJustReleased(key core.KeyCode) bool {
	return inpututil.IsKeyJustReleased(keyMap[key])
}

// IsMouseButtonPressed reports whether the mouse button is currently held.
func (i *Input) IsMouseButtonPressed(button core.MouseButton) bool {
	return ebiten.IsMouseButtonPressed(mouseMap[button])
}

// IsMouseButtonJustPressed reports whether the mouse button was pressed this frame.
func (i *Input) IsMouseButtonJustPressed(button core.MouseButton) bool {
	return inpututil.IsMouseButtonJustPressed(mouseMap[button])
}

// IsMouseButtonJustReleased reports whether the mouse button was released this frame.
func (i *Input) IsMouseButtonJustReleased(button core.MouseButton) bool {
	return inpututil.IsMouseButtonJustReleased(mouseMap[button])
}

// GetMousePosition returns the current mouse position in screen coordinates.
func (i *Input) GetMousePosition() math.Vector2 {
	return i.mousePosition
}

// GetMouseDelta returns the mouse movement since last frame.
func (i *Input) GetMouseDelta() math.Vector2 {
	return i.mouseDelta
}

// GetMouseScroll returns the mouse wheel scroll delta.
func (i *Input) GetMouseScroll() math.Vector2 {
	return i.mouseScroll
}

// Update snapshots the mouse state for this frame. Called once per frame by the runner.
func (i *Input) Update() {
	x, y := ebiten.CursorPosition()
	i.mousePosition = math.NewVector2(float64(x), float64(y))
	i.mouseDelta = i.mousePosition.Subtract(i.prevMouse)
	i.prevMouse = i.mousePosition

	wx, wy := ebiten.Wheel()
	i.mouseScroll = math.NewVector2(wx, wy)
}
