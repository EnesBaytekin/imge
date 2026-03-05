#include "imge/core/Engine.hpp"
#include "imge/impl/SDL2Audio.hpp"
#include "imge/impl/SDL2Input.hpp"
#include "imge/impl/SDL2Renderer.hpp"

#include <SDL2/SDL.h>
#include <chrono>
#include <stdexcept>

namespace imge {

/**
 * SDL2 implementation of the Engine main loop for example game
 */
class SDL2Engine : public Engine {
public:
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
 * Simple example game - demonstrates basic engine usage
 */
int main(int argc, char* argv[]) {
    std::cout << "IMGE Simple Game Example" << std::endl;
    std::cout << "Controls: WASD to move, ESC to quit" << std::endl;

    // Initialize SDL2
    if (SDL_Init(SDL_INIT_VIDEO | SDL_INIT_AUDIO) < 0) {
        std::cerr << "Failed to initialize SDL2: " << SDL_GetError() << std::endl;
        return 1;
    }

    try {
        // Create engine instance
        auto* engine = static_cast<imge::SDL2Engine*>(imge::Engine::getInstance());

        // Create a simple scene
        auto scene = std::make_shared<imge::Scene>();
        scene->width = 800;
        scene->height = 600;
        scene->backgroundColor = "#222222";

        // Add a simple object
        auto player = std::make_shared<imge::Object>(400.0f, 300.0f, "player");
        player->addTag("player");
        player->depth = 10.0f;
        scene->addObject(player);

        // Add scene to engine
        engine->addScene("main", scene);

        // Initialize engine
        engine->init(800, 600, "IMGE Simple Game");

        // Run the engine
        engine->run();

    } catch (const std::exception& e) {
        std::cerr << "Error: " << e.what() << std::endl;
        SDL_Quit();
        return 1;
    }

    // Cleanup
    SDL_Quit();

    std::cout << "Game ended successfully!" << std::endl;
    return 0;
}
