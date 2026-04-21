package sdl

import (
	"fmt"
	"log"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
	"github.com/veandco/go-sdl2/sdl"
)

// SDLInput implements core.Input interface using SDL2.
type SDLInput struct {
	window *SDLWindow // Reference to window for setting shouldClose flag

	// Keyboard state tracking
	currentKeys map[core.KeyCode]bool
	prevKeys    map[core.KeyCode]bool

	// Mouse state tracking
	currentMouseButtons map[core.MouseButton]bool
	prevMouseButtons    map[core.MouseButton]bool
	mousePosition       math.Vector2
	mouseDelta          math.Vector2
	mouseScroll         math.Vector2

	// SDL keycode to engine KeyCode mapping
	keyMap map[sdl.Keycode]core.KeyCode
	// Reverse mapping for debugging
	reverseKeyMap map[core.KeyCode]sdl.Keycode
}

// NewSDLInput creates a new SDLInput instance.
func NewSDLInput(window *SDLWindow) *SDLInput {
	input := &SDLInput{
		window: window,
		currentKeys: make(map[core.KeyCode]bool),
		prevKeys: make(map[core.KeyCode]bool),
		currentMouseButtons: make(map[core.MouseButton]bool),
		prevMouseButtons: make(map[core.MouseButton]bool),
		mousePosition: math.Vector2{X: 0, Y: 0},
		mouseDelta: math.Vector2{X: 0, Y: 0},
		mouseScroll: math.Vector2{X: 0, Y: 0},
		keyMap: make(map[sdl.Keycode]core.KeyCode),
		reverseKeyMap: make(map[core.KeyCode]sdl.Keycode),
	}

	// Initialize key mappings
	input.initKeyMappings()

	return input
}

// initKeyMappings initializes the SDL keycode to engine KeyCode mapping.
func (i *SDLInput) initKeyMappings() {
	// Alphabet keys
	i.keyMap[sdl.K_a] = core.KeyA
	i.keyMap[sdl.K_b] = core.KeyB
	i.keyMap[sdl.K_c] = core.KeyC
	i.keyMap[sdl.K_d] = core.KeyD
	i.keyMap[sdl.K_e] = core.KeyE
	i.keyMap[sdl.K_f] = core.KeyF
	i.keyMap[sdl.K_g] = core.KeyG
	i.keyMap[sdl.K_h] = core.KeyH
	i.keyMap[sdl.K_i] = core.KeyI
	i.keyMap[sdl.K_j] = core.KeyJ
	i.keyMap[sdl.K_k] = core.KeyK
	i.keyMap[sdl.K_l] = core.KeyL
	i.keyMap[sdl.K_m] = core.KeyM
	i.keyMap[sdl.K_n] = core.KeyN
	i.keyMap[sdl.K_o] = core.KeyO
	i.keyMap[sdl.K_p] = core.KeyP
	i.keyMap[sdl.K_q] = core.KeyQ
	i.keyMap[sdl.K_r] = core.KeyR
	i.keyMap[sdl.K_s] = core.KeyS
	i.keyMap[sdl.K_t] = core.KeyT
	i.keyMap[sdl.K_u] = core.KeyU
	i.keyMap[sdl.K_v] = core.KeyV
	i.keyMap[sdl.K_w] = core.KeyW
	i.keyMap[sdl.K_x] = core.KeyX
	i.keyMap[sdl.K_y] = core.KeyY
	i.keyMap[sdl.K_z] = core.KeyZ

	// Number keys
	i.keyMap[sdl.K_0] = core.Key0
	i.keyMap[sdl.K_1] = core.Key1
	i.keyMap[sdl.K_2] = core.Key2
	i.keyMap[sdl.K_3] = core.Key3
	i.keyMap[sdl.K_4] = core.Key4
	i.keyMap[sdl.K_5] = core.Key5
	i.keyMap[sdl.K_6] = core.Key6
	i.keyMap[sdl.K_7] = core.Key7
	i.keyMap[sdl.K_8] = core.Key8
	i.keyMap[sdl.K_9] = core.Key9

	// Special keys
	i.keyMap[sdl.K_SPACE] = core.KeySpace
	i.keyMap[sdl.K_RETURN] = core.KeyEnter
	i.keyMap[sdl.K_ESCAPE] = core.KeyEscape
	i.keyMap[sdl.K_BACKSPACE] = core.KeyBackspace
	i.keyMap[sdl.K_TAB] = core.KeyTab
	i.keyMap[sdl.K_LSHIFT] = core.KeyShift
	i.keyMap[sdl.K_RSHIFT] = core.KeyShift
	i.keyMap[sdl.K_LCTRL] = core.KeyControl
	i.keyMap[sdl.K_RCTRL] = core.KeyControl
	i.keyMap[sdl.K_LALT] = core.KeyAlt
	i.keyMap[sdl.K_RALT] = core.KeyAlt
	i.keyMap[sdl.K_LEFT] = core.KeyLeft
	i.keyMap[sdl.K_RIGHT] = core.KeyRight
	i.keyMap[sdl.K_UP] = core.KeyUp
	i.keyMap[sdl.K_DOWN] = core.KeyDown

	// Function keys
	i.keyMap[sdl.K_F1] = core.KeyF1
	i.keyMap[sdl.K_F2] = core.KeyF2
	i.keyMap[sdl.K_F3] = core.KeyF3
	i.keyMap[sdl.K_F4] = core.KeyF4
	i.keyMap[sdl.K_F5] = core.KeyF5
	i.keyMap[sdl.K_F6] = core.KeyF6
	i.keyMap[sdl.K_F7] = core.KeyF7
	i.keyMap[sdl.K_F8] = core.KeyF8
	i.keyMap[sdl.K_F9] = core.KeyF9
	i.keyMap[sdl.K_F10] = core.KeyF10
	i.keyMap[sdl.K_F11] = core.KeyF11
	i.keyMap[sdl.K_F12] = core.KeyF12

	// Build reverse mapping for debugging
	for sdlKey, engineKey := range i.keyMap {
		i.reverseKeyMap[engineKey] = sdlKey
	}
}

// toEngineKeyCode converts SDL keycode to engine KeyCode.
// Returns KeyUnknown if mapping not found.
func (i *SDLInput) toEngineKeyCode(sdlKey sdl.Keycode) core.KeyCode {
	if engineKey, ok := i.keyMap[sdlKey]; ok {
		return engineKey
	}
	return core.KeyUnknown
}

// toSDLMouseButton converts engine MouseButton to SDL button constant.
func toSDLMouseButton(button core.MouseButton) uint8 {
	switch button {
	case core.MouseButtonLeft:
		return sdl.BUTTON_LEFT
	case core.MouseButtonRight:
		return sdl.BUTTON_RIGHT
	case core.MouseButtonMiddle:
		return sdl.BUTTON_MIDDLE
	case core.MouseButton4:
		return sdl.BUTTON_X1
	case core.MouseButton5:
		return sdl.BUTTON_X2
	default:
		return 0
	}
}

// toEngineMouseButton converts SDL button to engine MouseButton.
func toEngineMouseButton(sdlButton uint8) core.MouseButton {
	switch sdlButton {
	case sdl.BUTTON_LEFT:
		return core.MouseButtonLeft
	case sdl.BUTTON_RIGHT:
		return core.MouseButtonRight
	case sdl.BUTTON_MIDDLE:
		return core.MouseButtonMiddle
	case sdl.BUTTON_X1:
		return core.MouseButton4
	case sdl.BUTTON_X2:
		return core.MouseButton5
	default:
		return core.MouseButtonLeft // Default fallback
	}
}

// IsKeyPressed checks if a key is currently pressed.
func (i *SDLInput) IsKeyPressed(key core.KeyCode) bool {
	return i.currentKeys[key]
}

// IsKeyJustPressed checks if a key was pressed this frame (not held).
func (i *SDLInput) IsKeyJustPressed(key core.KeyCode) bool {
	return i.currentKeys[key] && !i.prevKeys[key]
}

// IsKeyJustReleased checks if a key was released this frame.
func (i *SDLInput) IsKeyJustReleased(key core.KeyCode) bool {
	return !i.currentKeys[key] && i.prevKeys[key]
}

// IsMouseButtonPressed checks if a mouse button is currently pressed.
func (i *SDLInput) IsMouseButtonPressed(button core.MouseButton) bool {
	return i.currentMouseButtons[button]
}

// IsMouseButtonJustPressed checks if a mouse button was pressed this frame.
func (i *SDLInput) IsMouseButtonJustPressed(button core.MouseButton) bool {
	return i.currentMouseButtons[button] && !i.prevMouseButtons[button]
}

// IsMouseButtonJustReleased checks if a mouse button was released this frame.
func (i *SDLInput) IsMouseButtonJustReleased(button core.MouseButton) bool {
	return !i.currentMouseButtons[button] && i.prevMouseButtons[button]
}

// GetMousePosition returns the current mouse position in screen coordinates.
func (i *SDLInput) GetMousePosition() math.Vector2 {
	return i.mousePosition
}

// GetMouseDelta returns the mouse movement since last frame.
func (i *SDLInput) GetMouseDelta() math.Vector2 {
	return i.mouseDelta
}

// GetMouseScroll returns the mouse wheel scroll delta.
func (i *SDLInput) GetMouseScroll() math.Vector2 {
	return i.mouseScroll
}

// Update should be called once per frame to update input state.
// Polls SDL events and updates keyboard/mouse state.
func (i *SDLInput) Update() {
	// Save previous state
	for key, pressed := range i.currentKeys {
		i.prevKeys[key] = pressed
	}
	for button, pressed := range i.currentMouseButtons {
		i.prevMouseButtons[button] = pressed
	}

	// Reset mouse delta and scroll for this frame
	i.mouseDelta = math.Vector2{X: 0, Y: 0}
	i.mouseScroll = math.Vector2{X: 0, Y: 0}

	// Poll SDL events
	for event := sdl.PollEvent(); event != nil; event = sdl.PollEvent() {
		switch e := event.(type) {
		case *sdl.QuitEvent:
			// Window close requested
			if i.window != nil {
				i.window.SetShouldClose(true)
			}
			log.Println("SDL QuitEvent received")

		case *sdl.KeyboardEvent:
			engineKey := i.toEngineKeyCode(e.Keysym.Sym)
			if engineKey == core.KeyUnknown {
				// Skip unmapped keys
				continue
			}

			if e.Type == sdl.KEYDOWN {
				i.currentKeys[engineKey] = true
			} else if e.Type == sdl.KEYUP {
				i.currentKeys[engineKey] = false
			}

		case *sdl.MouseMotionEvent:
			// prevX, prevY := i.mousePosition.X, i.mousePosition.Y
			i.mousePosition = math.Vector2{X: float64(e.X), Y: float64(e.Y)}
			i.mouseDelta = math.Vector2{X: float64(e.XRel), Y: float64(e.YRel)}
			// Alternatively: i.mouseDelta = math.Vector2{X: i.mousePosition.X - prevX, Y: i.mousePosition.Y - prevY}

		case *sdl.MouseButtonEvent:
			engineButton := toEngineMouseButton(e.Button)
			if e.Type == sdl.MOUSEBUTTONDOWN {
				i.currentMouseButtons[engineButton] = true
			} else if e.Type == sdl.MOUSEBUTTONUP {
				i.currentMouseButtons[engineButton] = false
			}

		case *sdl.MouseWheelEvent:
			i.mouseScroll = math.Vector2{X: float64(e.X), Y: float64(e.Y)}

		case *sdl.WindowEvent:
			// Handle window events if needed (resize, focus, etc.)
			switch e.Event {
			case sdl.WINDOWEVENT_CLOSE:
				if i.window != nil {
					i.window.SetShouldClose(true)
				}
			}
		}
	}

	// Additionally, get current keyboard state for keys that might not generate events
	// (e.g., held keys between frames)
	i.updateKeyboardState()
}

// updateKeyboardState updates keyboard state from SDL_GetKeyboardState.
// This ensures we detect keys held across frames.
func (i *SDLInput) updateKeyboardState() {
	// Note: This approach requires converting SDL scancodes to keycodes
	// For simplicity, we're relying on events for now
	// Can be enhanced later for more accurate key state tracking
}

// Cleanup releases SDL input resources.
func (i *SDLInput) Cleanup() {
	// No SDL resources to clean up specifically for input
}