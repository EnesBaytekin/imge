#pragma once

#include "imge/core/Component.hpp"
#include <string>

namespace imge {

/**
 * Image component - renders a sprite
 * Platform-agnostic - rendering handled by Screen abstraction
 */
class Image : public Component {
public:
    /**
     * Constructor
     * @param imagePath Path to image file
     * @param pivotX Pivot X (0, "center", "end", or pixel value)
     * @param pivotY Pivot Y (0, "center", "end", or pixel value)
     */
    Image(const std::string& imagePath,
          const std::string& pivotX = "0",
          const std::string& pivotY = "0");

    ~Image() override = default;

    void onCreate(Object* owner) override;
    void onDraw(Object* owner) override;
    void fromJSON(const nlohmann::json& j) override;

    /**
     * Set pivot after initialization
     * @param x Pivot X
     * @param y Pivot Y
     */
    void setPivot(const std::string& x, const std::string& y);

    /**
     * Get image path (for renderer to use)
     */
    const std::string& getPath() const { return imagePath; }

    /**
     * Get width (may be 0 if not loaded yet)
     */
    int getWidth() const { return width; }

    /**
     * Get height (may be 0 if not loaded yet)
     */
    int getHeight() const { return height; }

private:
    int pivotX = 0;
    int pivotY = 0;
    int width = 0;
    int height = 0;
    std::string imagePath;
    std::string pivotXStr = "0";
    std::string pivotYStr = "0";
    void* textureHandle = nullptr;  // Opaque handle for renderer

    /**
     * Parse pivot value to pixel coordinate
     * @param val Pivot value ("center", "end", or pixel value)
     * @param maxVal Maximum value (width or height)
     * @return Pixel coordinate
     */
    [[nodiscard]] int _parsePivot(const std::string& val, int maxVal) const;
};

} // namespace imge
