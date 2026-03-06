#pragma once

#include "imge/core/Engine.hpp"
#include "imge/impl/SDL2Audio.hpp"
#include "imge/impl/SDL2Input.hpp"
#include "imge/impl/SDL2Renderer.hpp"
#include "imge/services/Input.hpp"
#include "imge/services/Screen.hpp"
#include "imge/services/Time.hpp"

#include <SDL2/SDL.h>
#include <chrono>
#include <memory>

namespace imge {

/**
 * SDL2 Engine implementation
 * Platform-specific engine implementation using SDL2
 * All SDL2 details are hidden inside this class
 */
class SDL2Engine : public Engine {
public:
    SDL2Engine();
    ~SDL2Engine() override;

    /**
     * Initialize SDL2 and all services
     * @param width Window width
     * @param height Window height
     * @param title Window title
     */
    void init(int width, int height, const std::string& title) override;

    /**
     * Main game loop with SDL2 event handling
     */
    void run() override;

private:
    // SDL2-specific service instances (owned by this engine)
    std::unique_ptr<SDL2Renderer> renderer;
    std::unique_ptr<SDL2Input> input;
    std::unique_ptr<SDL2Audio> audio;
    bool sdlInitialized = false;
};

} // namespace imge
