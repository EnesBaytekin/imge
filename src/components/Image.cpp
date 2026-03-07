#include "imge/components/Image.hpp"
#include "imge/core/Object.hpp"
#include "imge/impl/SDL2Renderer.hpp"
#include "imge/services/Screen.hpp"

#include <SDL2/SDL_image.h>
#include <stdexcept>
#include <iostream>

namespace imge {

Image::Image(const std::string& imageOrPath,
             const std::string& pivotX_,
             const std::string& pivotY_)
    : pivotX(0)
    , pivotY(0)
    , imagePath(imageOrPath)
{
    // Note: Texture loading happens in onCreate when renderer is available
    // Store pivot values as strings for now
    // This is a simplified version - full implementation would parse immediately
    (void)pivotX_;
    (void)pivotY_;
}

Image::~Image() {
    if (texture) {
        SDL_DestroyTexture(static_cast<SDL_Texture*>(texture));
        texture = nullptr;
    }
}

void Image::onCreate(Object* owner) {
    (void)owner;

    // Load surface
    SDL_Surface* surface = IMG_Load(imagePath.c_str());
    if (!surface) {
        return;
    }

    // Get the Screen implementation (which is SDL2Renderer)
    auto* screen = Screen::getInstance();
    if (!screen) {
        SDL_FreeSurface(surface);
        return;
    }

    // Cast to implementation to get SDL_Renderer
    auto* rendererImpl = static_cast<SDL2Renderer*>(screen);
    if (!rendererImpl->getRenderer()) {
        SDL_FreeSurface(surface);
        return;
    }

    texture = SDL_CreateTextureFromSurface(rendererImpl->getRenderer(), surface);
    if (texture) {
        SDL_QueryTexture(static_cast<SDL_Texture*>(texture), nullptr, nullptr, &width, &height);
    }

    SDL_FreeSurface(surface);
}

void Image::onDraw(Object* owner) {
    if (!texture) {
        return;
    }

    auto* screen = Screen::getInstance();
    if (!screen) {
        return;
    }

    // Use abstract drawTexture method
    screen->drawTexture(texture, owner->x - pivotX, owner->y - pivotY, width, height);
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
