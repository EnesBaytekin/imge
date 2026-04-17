package mock

import (
	"fmt"

	"github.com/EnesBaytekin/imge/internal/core/math"
)

// MockRenderer implements core.Renderer interface with debug prints.
type MockRenderer struct{}

// Clear clears the entire screen with the specified color.
func (r *MockRenderer) Clear(color math.Color) {
	fmt.Printf("[MockRenderer] Clear(color=%v)\n", color)
}

// DrawRect draws a filled rectangle.
func (r *MockRenderer) DrawRect(rect math.Rect, color math.Color) {
	fmt.Printf("[MockRenderer] DrawRect(rect=%v, color=%v)\n", rect, color)
}

// DrawRectOutline draws a rectangle outline (border only).
func (r *MockRenderer) DrawRectOutline(rect math.Rect, color math.Color, thickness float64) {
	fmt.Printf("[MockRenderer] DrawRectOutline(rect=%v, color=%v, thickness=%f)\n", rect, color, thickness)
}

// DrawCircle draws a filled circle.
func (r *MockRenderer) DrawCircle(center math.Vector2, radius float64, color math.Color) {
	fmt.Printf("[MockRenderer] DrawCircle(center=%v, radius=%f, color=%v)\n", center, radius, color)
}

// DrawCircleOutline draws a circle outline.
func (r *MockRenderer) DrawCircleOutline(center math.Vector2, radius float64, color math.Color, thickness float64) {
	fmt.Printf("[MockRenderer] DrawCircleOutline(center=%v, radius=%f, color=%v, thickness=%f)\n", center, radius, color, thickness)
}

// DrawLine draws a line between two points.
func (r *MockRenderer) DrawLine(start, end math.Vector2, color math.Color, thickness float64) {
	fmt.Printf("[MockRenderer] DrawLine(start=%v, end=%v, color=%v, thickness=%f)\n", start, end, color, thickness)
}

// DrawTexture draws a texture (image) at the specified position with transformations.
func (r *MockRenderer) DrawTexture(textureID string, position math.Vector2, scale math.Vector2, rotation float64, tint math.Color) {
	fmt.Printf("[MockRenderer] DrawTexture(id=%s, position=%v, scale=%v, rotation=%f, tint=%v)\n",
		textureID, position, scale, rotation, tint)
}

// Present presents the rendered frame to the screen (swap buffers).
func (r *MockRenderer) Present() {
	fmt.Println("[MockRenderer] Present()")
}

// SetViewport sets the rendering viewport size.
func (r *MockRenderer) SetViewport(width, height int) {
	fmt.Printf("[MockRenderer] SetViewport(width=%d, height=%d)\n", width, height)
}

// GetViewportSize returns the current viewport size.
func (r *MockRenderer) GetViewportSize() (width, height int) {
	fmt.Println("[MockRenderer] GetViewportSize() called")
	return 800, 600 // Default mock size
}