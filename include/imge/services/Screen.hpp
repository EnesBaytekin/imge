#pragma once

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

protected:
    static Screen* instance;
};

} // namespace imge
