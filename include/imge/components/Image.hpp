#pragma once

#include "imge/core/Component.hpp"
#include "imge/core/Vec2.hpp"

#include <SDL2/SDL.h>
#include <string>

namespace imge {

/**
 * Image component - renders a sprite
 * Part of builtin components (prefixed with @)
 */
class Image : public Component {
public:
    /**
     * Constructor
     * @param imageOrPath Path to image file
     * @param pivotX Pivot X (0, "center", "end", or pixel value)
     * @param pivotY Pivot Y (0, "center", "end", or pixel value)
     */
    Image(const std::string& imageOrPath,
          const std::string& pivotX = "0",
          const std::string& pivotY = "0");

    ~Image() override;

    void onCreate(Object* owner) override;
    void onDraw(Object* owner) override;
    void fromJSON(const nlohmann::json& j) override;

    /**
     * Set pivot after initialization
     * @param x Pivot X
     * @param y Pivot Y
     */
    void setPivot(const std::string& x, const std::string& y);

private:
    SDL_Texture* texture = nullptr;
    int pivotX = 0;
    int pivotY = 0;
    int width = 0;
    int height = 0;
    std::string imagePath;

    /**
     * Parse pivot value to pixel coordinate
     * @param val Pivot value ("center", "end", or pixel value)
     * @param maxVal Maximum value (width or height)
     * @return Pixel coordinate
     */
    [[nodiscard]] int _parsePivot(const std::string& val, int maxVal) const;
};

} // namespace imge
