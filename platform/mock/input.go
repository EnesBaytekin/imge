package mock

import (
	"fmt"

	"github.com/EnesBaytekin/imge/core"
	"github.com/EnesBaytekin/imge/core/math"
)

// MockInput implements core.Input interface with debug prints.
type MockInput struct {
	mousePosition math.Vector2
	mouseDelta    math.Vector2
	mouseScroll   math.Vector2
}

// IsKeyPressed checks if a key is currently pressed.
func (i *MockInput) IsKeyPressed(key core.KeyCode) bool {
	fmt.Printf("[MockInput] IsKeyPressed(key=%d)\n", key)
	return false // Always false in mock
}

// IsKeyJustPressed checks if a key was pressed this frame (not held).
func (i *MockInput) IsKeyJustPressed(key core.KeyCode) bool {
	fmt.Printf("[MockInput] IsKeyJustPressed(key=%d)\n", key)
	return false // Always false in mock
}

// IsKeyJustReleased checks if a key was released this frame.
func (i *MockInput) IsKeyJustReleased(key core.KeyCode) bool {
	fmt.Printf("[MockInput] IsKeyJustReleased(key=%d)\n", key)
	return false // Always false in mock
}

// IsMouseButtonPressed checks if a mouse button is currently pressed.
func (i *MockInput) IsMouseButtonPressed(button core.MouseButton) bool {
	fmt.Printf("[MockInput] IsMouseButtonPressed(button=%d)\n", button)
	return false // Always false in mock
}

// IsMouseButtonJustPressed checks if a mouse button was pressed this frame.
func (i *MockInput) IsMouseButtonJustPressed(button core.MouseButton) bool {
	fmt.Printf("[MockInput] IsMouseButtonJustPressed(button=%d)\n", button)
	return false // Always false in mock
}

// IsMouseButtonJustReleased checks if a mouse button was released this frame.
func (i *MockInput) IsMouseButtonJustReleased(button core.MouseButton) bool {
	fmt.Printf("[MockInput] IsMouseButtonJustReleased(button=%d)\n", button)
	return false // Always false in mock
}

// GetMousePosition returns the current mouse position in screen coordinates.
func (i *MockInput) GetMousePosition() math.Vector2 {
	fmt.Println("[MockInput] GetMousePosition() called")
	return i.mousePosition
}

// GetMouseDelta returns the mouse movement since last frame.
func (i *MockInput) GetMouseDelta() math.Vector2 {
	fmt.Println("[MockInput] GetMouseDelta() called")
	return i.mouseDelta
}

// GetMouseScroll returns the mouse wheel scroll delta.
func (i *MockInput) GetMouseScroll() math.Vector2 {
	fmt.Println("[MockInput] GetMouseScroll() called")
	return i.mouseScroll
}

// Update should be called once per frame to update input state.
func (i *MockInput) Update() {
	fmt.Println("[MockInput] Update() called")
	// Update mock mouse position slightly for testing
	i.mousePosition = math.Vector2{X: i.mousePosition.X + 0.5, Y: i.mousePosition.Y + 0.3}
	i.mouseDelta = math.Vector2{X: 0.5, Y: 0.3}
}