#pragma once

#include "imge/core/Singleton.hpp"

namespace imge {

/**
 * Time service - manages delta time and FPS
 */
class Time : public Singleton<Time> {
public:
    float deltaTime = 0.0f;   // Time since last frame (seconds)
    float totalTime = 0.0f;   // Total time since start (seconds)
    int frameCount = 0;        // Total frames since start
    float fps = 60.0f;         // Current FPS
    float targetFps = 60.0f;   // Target FPS
    float fixedDeltaTime = 1.0f / 60.0f; // Fixed timestep for physics

    /**
     * Update time values (call once per frame)
     * @param dt Delta time for this frame
     */
    void update(float dt) {
        deltaTime = dt;
        totalTime += dt;
        frameCount++;

        // Calculate FPS (simple moving average)
        static float fpsAccumulator = 0.0f;
        static int fpsSamples = 0;
        fpsAccumulator += dt;
        fpsSamples++;

        if (fpsAccumulator >= 0.5f) { // Update every 0.5 seconds
            fps = static_cast<float>(fpsSamples) / fpsAccumulator;
            fpsAccumulator = 0.0f;
            fpsSamples = 0;
        }
    }
};

} // namespace imge
