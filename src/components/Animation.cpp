#include "imge/components/Animation.hpp"
#include "imge/core/Object.hpp"
#include "imge/services/Screen.hpp"
#include "imge/services/Time.hpp"

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
    (void)pivotX_;
    (void)pivotY_;
}

Animation::~Animation() = default;

void Animation::onCreate(Object* owner) {
    // Parse pivot values
    pivotX = _parsePivot("center", width);   // Default to center
    pivotY = _parsePivot("center", height);  // Default to center

    // Build frame rectangles (will be loaded by renderer)
    frames.clear();

    // TODO: Load actual sprite sheet dimensions
    // For now, assume single frame
    FrameRect frame;
    frame.x = 0;
    frame.y = 0;
    frame.width = frameWidth;
    frame.height = frameHeight;
    frames.push_back(frame);

    (void)owner;
}

void Animation::onUpdate(Object* owner) {
    if (!playing) return;

    auto* time = Time::getInstance();
    timer += time->deltaTime * speed;

    if (timer >= 1.0f) {
        timer = 0.0f;
        currentFrame++;

        if (currentFrame >= frames.size()) {
            if (loop) {
                currentFrame = 0;
            } else {
                playing = false;
                currentFrame = frames.size() - 1;
            }
        }
    }

    (void)owner;
}

void Animation::onDraw(Object* owner) {
    // TODO: Implement animation rendering
    // For now, this is a stub - Animation component needs platform-specific implementation
    (void)owner;
}

void Animation::fromJSON(const nlohmann::json& j) {
    if (j.contains("file")) {
        data.file = j["file"];
    }

    if (j.contains("frameWidth")) {
        data.frameWidth = j["frameWidth"];
        frameWidth = data.frameWidth;
    }

    if (j.contains("frameHeight")) {
        data.frameHeight = j["frameHeight"];
        frameHeight = data.frameHeight;
    }

    if (j.contains("frames")) {
        data.frames = j["frames"].get<std::vector<int>>();
    }

    if (j.contains("speed")) {
        data.speed = j["speed"];
        speed = data.speed;
    }

    if (j.contains("loop")) {
        data.loop = j["loop"];
        loop = data.loop;
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

void Animation::_loadFromSpriteSheet(const std::string& file) {
    // TODO: Platform-specific sprite sheet loading
    // This will be implemented by renderer
    (void)file;
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
