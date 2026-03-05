#include "imge/components/Image.hpp"
#include "imge/impl/SDL2Renderer.hpp"

#include <SDL2/SDL_image.h>
#include <stdexcept>

namespace imge {

Image::Image(const std::string& imageOrPath,
             const std::string& pivotX_,
             const std::string& pivotY_)
    : imagePath(imageOrPath)
    , pivotX(0)
    , pivotY(0)
{
    // Note: Texture loading happens in onCreate when renderer is available
    // Store pivot values as strings for now
    // This is a simplified version - full implementation would parse immediately
    (void)pivotX_;
    (void)pivotY_;
}

Image::~Image() {
    if (texture) {
        SDL_DestroyTexture(texture);
        texture = nullptr;
    }
}

void Image::onCreate(Object* owner) {
    // Load image when renderer is available
    auto* renderer = static_cast<SDL2Renderer*>(Screen::getInstance());
    if (!renderer || !renderer->getRenderer()) {
        return;
    }

    SDL_Surface* surface = IMG_Load(imagePath.c_str());
    if (!surface) {
        return; // Failed to load
    }

    texture = SDL_CreateTextureFromSurface(renderer->getRenderer(), surface);
    SDL_FreeSurface(surface);

    if (texture) {
        SDL_QueryTexture(texture, nullptr, nullptr, &width, &height);
    }
}

void Image::onDraw(Object* owner) {
    if (!texture) {
        return;
    }

    auto* renderer = static_cast<SDL2Renderer*>(Screen::getInstance());
    if (!renderer || !renderer->getRenderer()) {
        return;
    }

    SDL_Rect destRect;
    destRect.x = static_cast<int>(owner->x - pivotX);
    destRect.y = static_cast<int>(owner->y - pivotY);
    destRect.w = width;
    destRect.h = height;

    SDL_RenderCopy(renderer->getRenderer(), texture, nullptr, &destRect);
}

void Image::fromJSON(const nlohmann::json& j) {
    if (j.contains("file")) {
        imagePath = j["file"];
    }

    // Note: Pivot values need to be parsed from JSON
    // This is a simplified version
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
        return maxVal / 2;
    } else if (val == "end") {
        return maxVal - 1;
    } else {
        try {
            return std::stoi(val);
        } catch (...) {
            return 0;
        }
    }
}

} // namespace imge
