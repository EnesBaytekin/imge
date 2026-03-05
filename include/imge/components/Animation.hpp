#pragma once

#include "imge/core/Component.hpp"

#include <SDL2/SDL.h>
#include <string>
#include <vector>

namespace imge {

/**
 * Animation component - renders sprite sheet animation
 * Part of builtin components (prefixed with @)
 */
class Animation : public Component {
public:
    struct AnimationData {
        std::string file;
        int frameWidth = 32;
        int frameHeight = 32;
        std::vector<int> frames;  // Frame indices to play (empty = all sequential)
        float speed = 10.0f;      // Frames per second
        bool loop = true;
    };

    /**
     * Constructor
     * @param animOrData Animation file path or animation data
     * @param pivotX Pivot X
     * @param pivotY Pivot Y
     */
    Animation(const AnimationData& data,
              const std::string& pivotX = "0",
              const std::string& pivotY = "0");

    ~Animation() override;

    void onCreate(Object* owner) override;
    void onUpdate(Object* owner) override;
    void onDraw(Object* owner) override;
    void fromJSON(const nlohmann::json& j) override;

private:
    SDL_Texture* texture = nullptr;
    int pivotX = 0;
    int pivotY = 0;
    int width = 0;
    int height = 0;

    // Animation data
    int frameWidth = 32;
    int frameHeight = 32;
    std::vector<SDL_Rect> frames;
    float speed = 10.0f;
    bool loop = true;

    // Runtime state
    int currentFrame = 0;
    float timer = 0.0f;
    bool playing = true;

    /**
     * Load animation from sprite sheet
     */
    void _loadFromSpriteSheet(const std::string& file);

    /**
     * Parse pivot value to pixel coordinate
     */
    [[nodiscard]] int _parsePivot(const std::string& val, int maxVal) const;
};

} // namespace imge
