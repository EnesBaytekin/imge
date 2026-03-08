#include "imge/components/Image.hpp"
#include "imge/core/Object.hpp"
#include "imge/services/Screen.hpp"
#include <iostream>

namespace imge {

Image::Image(const std::string& imagePath_,
             const std::string& pivotX_,
             const std::string& pivotY_)
    : imagePath(imagePath_)
    , pivotXStr(pivotX_)
    , pivotYStr(pivotY_)
    , textureHandle(nullptr)
{
}

void Image::onCreate(Object* owner) {
    auto* screen = Screen::getInstance();
    if (!screen) return;

    // Load texture
    int texWidth = 0, texHeight = 0;
    textureHandle = screen->loadTexture(imagePath, texWidth, texHeight);

    if (!textureHandle) {
        std::cerr << "Failed to load image: " << imagePath << std::endl;
        return;
    }

    // Store actual image dimensions
    width = texWidth;
    height = texHeight;

    // Parse pivot values with actual dimensions
    pivotX = _parsePivot(pivotXStr, width);
    pivotY = _parsePivot(pivotYStr, height);

    (void)owner;
}

void Image::onDraw(Object* owner) {
    auto* screen = Screen::getInstance();
    if (!screen || !textureHandle) return;

    // Draw the loaded texture
    screen->drawTexture(textureHandle, owner->x - pivotX, owner->y - pivotY, width, height);
}

void Image::fromJSON(const nlohmann::json& j) {
    if (j.contains("file")) {
        imagePath = j["file"];
    }

    if (j.contains("pivotX")) {
        std::string pivotXVal = j["pivotX"];
        pivotX = _parsePivot(pivotXVal, width);
    }

    if (j.contains("pivotY")) {
        std::string pivotYVal = j["pivotY"];
        pivotY = _parsePivot(pivotYVal, height);
    }
}

void Image::setPivot(const std::string& x, const std::string& y) {
    pivotX = _parsePivot(x, width);
    pivotY = _parsePivot(y, height);
}

int Image::_parsePivot(const std::string& val, int maxVal) const {
    if (val == "center") {
        return maxVal > 0 ? maxVal / 2 : 16;
    } else if (val == "end") {
        return maxVal > 0 ? maxVal - 1 : 31;
    } else {
        try {
            return std::stoi(val);
        } catch (...) {
            return 0;
        }
    }
}

} // namespace imge
