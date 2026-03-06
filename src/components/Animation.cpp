#include "imge/components/Animation.hpp"
#include "imge/core/Object.hpp"
#include "imge/impl/SDL2Renderer.hpp"
#include "imge/services/Time.hpp"

#include <SDL2/SDL_image.h>
#include <algorithm>

namespace imge {

Animation::Animation(const AnimationData& data_,
                     const std::string& pivotX_,
                     const std::string& pivotY_)
    : data(data_)
    , frameWidth(data_.frameWidth)
    , frameHeight(data_.frameHeight)
    , speed(data_.speed)
    , loop(data_.loop)
{
    // Note: Texture loading happens in onCreate when renderer is available
    (void)pivotX_;
    (void)pivotY_;
}

Animation::~Animation() {
    if (texture) {
        SDL_DestroyTexture(texture);
        texture = nullptr;
    }
}

void Animation::onCreate(Object* owner) {
    // Load animation when renderer is available
    _loadFromSpriteSheet(data.file);

    // Parse pivot values
    pivotX = _parsePivot("center", width);   // Default to center
    pivotY = _parsePivot("center", height);  // Default to center

    // Build frame rectangles
    frames.clear();

    if (data.frames.empty()) {
        // Use all frames sequentially
        int columns = width / frameWidth;
        int rows = height / frameHeight;

        for (int row = 0; row < rows; ++row) {
            for (int col = 0; col < columns; ++col) {
                SDL_Rect frame;
                frame.x = col * frameWidth;
                frame.y = row * frameHeight;
                frame.w = frameWidth;
                frame.h = frameHeight;
                frames.push_back(frame);
            }
        }
    } else {
        // Use specified frames
        for (int frameIndex : data.frames) {
            SDL_Rect frame;
            frame.x = (frameIndex % (width / frameWidth)) * frameWidth;
            frame.y = (frameIndex / (width / frameWidth)) * frameHeight;
            frame.w = frameWidth;
            frame.h = frameHeight;
            frames.push_back(frame);
        }
    }
}

void Animation::onUpdate(Object* owner) {
    if (!playing || frames.empty()) {
        return;
    }

    // Get delta time
    float dt = owner ? 0.0f : Time::getInstance()->deltaTime;

    timer += dt;

    if (timer >= 1.0f / speed) {
        timer -= 1.0f / speed;
        currentFrame++;

        if (currentFrame >= static_cast<int>(frames.size())) {
            if (loop) {
                currentFrame = 0;
            } else {
                currentFrame = static_cast<int>(frames.size()) - 1;
                playing = false;
            }
        }
    }
}

void Animation::onDraw(Object* owner) {
    if (!texture || frames.empty()) {
        return;
    }

    auto* renderer = static_cast<SDL2Renderer*>(Screen::getInstance());
    if (!renderer || !renderer->getRenderer()) {
        return;
    }

    if (currentFrame >= static_cast<int>(frames.size())) {
        currentFrame = 0;
    }

    const SDL_Rect& srcRect = frames[currentFrame];

    SDL_Rect destRect;
    destRect.x = static_cast<int>(owner->x - pivotX);
    destRect.y = static_cast<int>(owner->y - pivotY);
    destRect.w = frameWidth;
    destRect.h = frameHeight;

    SDL_RenderCopy(renderer->getRenderer(), texture, &srcRect, &destRect);
}

void Animation::fromJSON(const nlohmann::json& j) {
    AnimationData data;

    if (j.contains("file")) {
        data.file = j["file"];
    }

    if (j.contains("frame_width")) {
        data.frameWidth = j["frame_width"];
    }

    if (j.contains("frame_height")) {
        data.frameHeight = j["frame_height"];
    }

    if (j.contains("frames")) {
        for (const auto& frame : j["frames"]) {
            data.frames.push_back(frame.get<int>());
        }
    }

    if (j.contains("speed")) {
        data.speed = j["speed"];
    }

    if (j.contains("loop")) {
        data.loop = j["loop"];
    }

    // Update internal state
    frameWidth = data.frameWidth;
    frameHeight = data.frameHeight;
    speed = data.speed;
    loop = data.loop;
}

void Animation::_loadFromSpriteSheet(const std::string& file) {
    auto* renderer = static_cast<SDL2Renderer*>(Screen::getInstance());
    if (!renderer || !renderer->getRenderer()) {
        return;
    }

    SDL_Surface* surface = IMG_Load(file.c_str());
    if (!surface) {
        return; // Failed to load
    }

    texture = SDL_CreateTextureFromSurface(renderer->getRenderer(), surface);
    SDL_FreeSurface(surface);

    if (texture) {
        SDL_QueryTexture(texture, nullptr, nullptr, &width, &height);
    }
}

int Animation::_parsePivot(const std::string& val, int maxVal) const {
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
