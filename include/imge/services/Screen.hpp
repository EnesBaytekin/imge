#pragma once

#include <cstdint>
#include <optional>
#include <string>

namespace imge {

/**
 * Screen service - abstract interface for rendering
 * Platform-specific implementations (SDL2, OpenGL, etc.) inherit from this
 *
 * Uses pointer-based singleton pattern to allow abstract base class
 */
class Screen {
public:
    virtual ~Screen() = default;

    /**
     * Get the singleton instance
     * @return Pointer to the screen service instance or nullptr if not set
     */
    static Screen* getInstance() {
        return instance;
    }

    /**
     * Set the singleton instance (called by concrete implementation)
     * @param inst Pointer to the concrete implementation
     */
    static void setInstance(Screen* inst) {
        instance = inst;
    }

    /**
     * Initialize the screen/window
     * @param width Window width
     * @param height Window height
     * @param title Window title
     */
    virtual void init(int width, int height, const std::string& title = "IMGE Game") = 0;

    /**
     * Clear the screen with background color or image
     */
    virtual void clear() = 0;

    /**
     * Present the rendered frame
     */
    virtual void present() = 0;

    /**
     * Set background color (hex format: "#RRGGBB")
     * @param color Hex color string
     */
    virtual void setBackgroundColor(const std::string& color) = 0;

    /**
     * Set background image
     * @param filename Path to image file
     */
    virtual void setBackgroundImage(const std::string& filename) = 0;

    /**
     * Get screen width
     */
    [[nodiscard]] virtual int getWidth() const = 0;

    /**
     * Get screen height
     */
    [[nodiscard]] virtual int getHeight() const = 0;

    /**
     * Check if window is still open
     */
    [[nodiscard]] virtual bool isOpen() const = 0;

    /**
     * Close the window
     */
    virtual void close() = 0;

    /**
     * Drawing primitives - abstract interface for rendering
     * These allow custom components to draw without platform-specific code
     */

    /**
     * Set the current drawing color
     * @param r Red component (0-255)
     * @param g Green component (0-255)
     * @param b Blue component (0-255)
     * @param a Alpha component (0-255, default 255)
     */
    virtual void setColor(uint8_t r, uint8_t g, uint8_t b, uint8_t a = 255) = 0;

    /**
     * Draw a filled rectangle
     * @param x X position (top-left corner)
     * @param y Y position (top-left corner)
     * @param width Rectangle width
     * @param height Rectangle height
     */
    virtual void drawRect(float x, float y, float width, float height) = 0;

    /**
     * Draw a rectangle outline
     * @param x X position (top-left corner)
     * @param y Y position (top-left corner)
     * @param width Rectangle width
     * @param height Rectangle height
     */
    virtual void drawRectOutline(float x, float y, float width, float height) = 0;

protected:
    static Screen* instance;
};

} // namespace imge
