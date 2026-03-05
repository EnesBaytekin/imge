#include "imge/core/Engine.hpp"
#include "imge/impl/SDL2Audio.hpp"
#include "imge/impl/SDL2Input.hpp"
#include "imge/impl/SDL2Renderer.hpp"

#include <SDL2/SDL.h>
#include <chrono>
#include <stdexcept>

namespace imge {

/**
 * SDL2 implementation of the Engine main loop
 */
class SDL2Engine : public Engine {
public:
    SDL2Engine() {
        setInstance(this);
    }

    void run() override {
        if (!scenes.empty()) {
            running = true;
        }

        auto* renderer = static_cast<SDL2Renderer*>(Screen::getInstance());
        auto* input = static_cast<SDL2Input*>(Input::getInstance());
        auto* time = Time::getInstance();

        auto lastTime = std::chrono::high_resolution_clock::now();

        while (running) {
            // Calculate delta time
            auto currentTime = std::chrono::high_resolution_clock::now();
            std::chrono::duration<float> delta = currentTime - lastTime;
            lastTime = currentTime;
            float dt = delta.count();

            // Handle SDL events
            SDL_Event event;
            while (SDL_PollEvent(&event)) {
                if (event.type == SDL_QUIT) {
                    running = false;
                }
            }

            // Update input
            input->update();

            // Update time
            time->update(dt);

            // Update scene
            auto* scene = getCurrentScene();
            if (scene) {
                scene->update();

                // Clear screen
                renderer->clear();

                // Draw scene
                scene->draw();

                // Present frame
                renderer->present();
            }

            // Cap FPS
            SDL_Delay(1);
        }

        // Cleanup
        renderer->close();
    }
};

} // namespace imge

/**
 * Main entry point for SDL2 implementation
 */
int main(int argc, char* argv[]) {
    (void)argc;
    (void)argv;

    // Initialize SDL2
    if (SDL_Init(SDL_INIT_VIDEO | SDL_INIT_AUDIO) < 0) {
        return 1;
    }

    try {
        // Create SDL2 service instances (these register themselves as singletons)
        imge::SDL2Renderer renderer;
        imge::SDL2Input input;
        imge::SDL2Audio audio;

        // Create engine instance (registers itself)
        imge::SDL2Engine engine;

        // TODO: Load scene from command line argument
        // For now, just initialize with default parameters
        engine.init(800, 600, "IMGE Game");

        // Run the engine
        engine.run();

    } catch (const std::exception& e) {
        SDL_Quit();
        return 1;
    }

    // Cleanup
    SDL_Quit();

    return 0;
}
