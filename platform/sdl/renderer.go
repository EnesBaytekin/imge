package sdl

import (
	"fmt"
	"log"

	"github.com/EnesBaytekin/imge/core/math"
	"github.com/veandco/go-sdl2/sdl"
	"github.com/veandco/go-sdl2/sdl_image"
)

// SDLRenderer implements core.Renderer interface using SDL2.
type SDLRenderer struct {
	renderer *sdl.Renderer
	window   *SDLWindow // Reference to window for viewport size
	textures map[string]*sdl.Texture // Texture cache
	viewportWidth  int
	viewportHeight int
}

// Init initializes the SDL renderer with the given window.
// Must be called after window.Create().
func (r *SDLRenderer) Init(window *SDLWindow) error {
	if window == nil {
		return fmt.Errorf("window is nil")
	}

	sdlWindow := window.GetSDLWindow()
	if sdlWindow == nil {
		return fmt.Errorf("SDL window is nil, call window.Create() first")
	}

	// Create SDL renderer
	renderer, err := sdl.CreateRenderer(sdlWindow, -1, sdl.RENDERER_ACCELERATED|sdl.RENDERER_PRESENTVSYNC)
	if err != nil {
		// Fallback to software renderer
		renderer, err = sdl.CreateRenderer(sdlWindow, -1, sdl.RENDERER_SOFTWARE)
		if err != nil {
			return fmt.Errorf("failed to create SDL renderer: %v", err)
		}
		log.Println("Using software SDL renderer (hardware accelerated not available)")
	} else {
		log.Println("Using hardware accelerated SDL renderer")
	}

	r.renderer = renderer
	r.window = window
	r.textures = make(map[string]*sdl.Texture)

	// Get window size for viewport
	width, height := window.GetSize()
	r.viewportWidth = width
	r.viewportHeight = height

	// Initialize SDL_image for texture loading
	if err := img.Init(img.INIT_PNG | img.INIT_JPG); err != nil {
		log.Printf("Warning: SDL_image initialization failed: %v", err)
	}

	log.Printf("SDL renderer initialized: %dx%d", width, height)
	return nil
}

// Clear clears the entire screen with the specified color.
func (r *SDLRenderer) Clear(color math.Color) {
	if r.renderer == nil {
		return
	}

	// math.Color already uses uint8 (0-255), same as SDL
	r.renderer.SetDrawColor(color.R, color.G, color.B, color.A)
	r.renderer.Clear()
}

// DrawRect draws a filled rectangle.
func (r *SDLRenderer) DrawRect(rect math.Rect, color math.Color) {
	if r.renderer == nil {
		return
	}

	r.renderer.SetDrawColor(color.R, color.G, color.B, color.A)

	sdlRect := sdl.Rect{
		X: int32(rect.X),
		Y: int32(rect.Y),
		W: int32(rect.Width),
		H: int32(rect.Height),
	}

	r.renderer.FillRect(&sdlRect)
}

// DrawRectOutline draws a rectangle outline (border only).
func (r *SDLRenderer) DrawRectOutline(rect math.Rect, color math.Color, thickness float64) {
	if r.renderer == nil {
		return
	}

	r.renderer.SetDrawColor(color.R, color.G, color.B, color.A)

	// Convert thickness to int32 (approx)
	thick := int32(thickness)
	if thick < 1 {
		thick = 1
	}

	// Draw four lines for rectangle outline
	// Top line
	topRect := sdl.Rect{
		X: int32(rect.X),
		Y: int32(rect.Y),
		W: int32(rect.Width),
		H: thick,
	}
	r.renderer.FillRect(&topRect)

	// Bottom line
	bottomRect := sdl.Rect{
		X: int32(rect.X),
		Y: int32(rect.Y + rect.Height - float64(thick)),
		W: int32(rect.Width),
		H: thick,
	}
	r.renderer.FillRect(&bottomRect)

	// Left line
	leftRect := sdl.Rect{
		X: int32(rect.X),
		Y: int32(rect.Y) + thick,
		W: thick,
		H: int32(rect.Height) - thick*2,
	}
	r.renderer.FillRect(&leftRect)

	// Right line
	rightRect := sdl.Rect{
		X: int32(rect.X + rect.Width - float64(thick)),
		Y: int32(rect.Y) + thick,
		W: thick,
		H: int32(rect.Height) - thick*2,
	}
	r.renderer.FillRect(&rightRect)
}

// DrawCircle draws a filled circle.
// Note: SDL2 doesn't have built-in circle drawing, so we approximate with filled polygons.
// For simplicity, we'll draw a filled circle using multiple triangles (fan).
func (r *SDLRenderer) DrawCircle(center math.Vector2, radius float64, color math.Color) {
	if r.renderer == nil {
		return
	}

	// Simple implementation: draw a filled circle using multiple triangles
	// For now, draw a rectangle as placeholder
	// TODO: Implement proper circle drawing
	rect := math.Rect{
		X: center.X - radius,
		Y: center.Y - radius,
		Width: radius * 2,
		Height: radius * 2,
	}
	r.DrawRect(rect, color)
}

// DrawCircleOutline draws a circle outline.
func (r *SDLRenderer) DrawCircleOutline(center math.Vector2, radius float64, color math.Color, thickness float64) {
	if r.renderer == nil {
		return
	}

	// Simple placeholder: draw rectangle outline
	// TODO: Implement proper circle outline drawing
	rect := math.Rect{
		X: center.X - radius,
		Y: center.Y - radius,
		Width: radius * 2,
		Height: radius * 2,
	}
	r.DrawRectOutline(rect, color, thickness)
}

// DrawLine draws a line between two points.
func (r *SDLRenderer) DrawLine(start, end math.Vector2, color math.Color, thickness float64) {
	if r.renderer == nil {
		return
	}

	r.renderer.SetDrawColor(color.R, color.G, color.B, color.A)

	// SDL2 doesn't support line thickness directly, so we draw a thick line using multiple pixels
	// For now, draw a thin line (thickness = 1)
	// TODO: Implement proper thick line drawing
	r.renderer.DrawLine(int32(start.X), int32(start.Y), int32(end.X), int32(end.Y))
}

// DrawTexture draws a texture (image) at the specified position with transformations.
func (r *SDLRenderer) DrawTexture(textureID string, position math.Vector2, scale math.Vector2, rotation float64, tint math.Color) {
	if r.renderer == nil {
		return
	}

	texture, exists := r.textures[textureID]
	if !exists {
		log.Printf("Texture not found: %s", textureID)
		// Draw a colored rectangle as fallback
		rect := math.Rect{
			X: position.X,
			Y: position.Y,
			Width: 64 * scale.X,  // Default size
			Height: 64 * scale.Y,
		}
		r.DrawRect(rect, tint)
		return
	}

	// TODO: Implement texture rendering with rotation and tint
	// For now, just draw the texture at position
	_, _, texWidth, texHeight, _ := texture.Query()

	dstRect := sdl.Rect{
		X: int32(position.X),
		Y: int32(position.Y),
		W: int32(float64(texWidth) * scale.X),
		H: int32(float64(texHeight) * scale.Y),
	}

	// Apply tint color modulation
	texture.SetColorMod(tint.R, tint.G, tint.B)
	texture.SetAlphaMod(tint.A)

	r.renderer.Copy(texture, nil, &dstRect)
}

// LoadTexture loads a texture from file and stores it in the cache.
func (r *SDLRenderer) LoadTexture(textureID, filePath string) error {
	if r.renderer == nil {
		return fmt.Errorf("renderer not initialized")
	}

	// Check if already loaded
	if _, exists := r.textures[textureID]; exists {
		return nil // Already loaded
	}

	// Load image surface
	surface, err := img.Load(filePath)
	if err != nil {
		return fmt.Errorf("failed to load image %s: %v", filePath, err)
	}
	defer surface.Free()

	// Create texture from surface
	texture, err := r.renderer.CreateTextureFromSurface(surface)
	if err != nil {
		return fmt.Errorf("failed to create texture from %s: %v", filePath, err)
	}

	r.textures[textureID] = texture
	log.Printf("Texture loaded: %s -> %s", textureID, filePath)
	return nil
}

// Present presents the rendered frame to the screen (swap buffers).
func (r *SDLRenderer) Present() {
	if r.renderer != nil {
		r.renderer.Present()
	}
}

// SetViewport sets the rendering viewport size.
func (r *SDLRenderer) SetViewport(width, height int) {
	if r.renderer != nil {
		r.viewportWidth = width
		r.viewportHeight = height

		// SDL viewport is a rectangle within the render target
		viewport := sdl.Rect{
			X: 0,
			Y: 0,
			W: int32(width),
			H: int32(height),
		}
		r.renderer.SetViewport(&viewport)
	}
}

// GetViewportSize returns the current viewport size.
func (r *SDLRenderer) GetViewportSize() (width, height int) {
	return r.viewportWidth, r.viewportHeight
}

// Cleanup releases SDL renderer resources.
func (r *SDLRenderer) Cleanup() {
	// Release textures
	for id, texture := range r.textures {
		texture.Destroy()
		delete(r.textures, id)
	}

	// Destroy renderer
	if r.renderer != nil {
		r.renderer.Destroy()
		r.renderer = nil
	}

	img.Quit()
}