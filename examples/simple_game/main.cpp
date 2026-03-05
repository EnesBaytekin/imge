#include "imge/core/Engine.hpp"
#include "imge/impl/SDL2Audio.hpp"
#include "imge/impl/SDL2Input.hpp"
#include "imge/impl/SDL2Renderer.hpp"
#include "imge/services/Input.hpp"
#include "imge/services/Screen.hpp"
#include "imge/services/Time.hpp"

#include <SDL2/SDL.h>
#include <cmath>
#include <iostream>

namespace imge {

/**
 * Player controller component - WASD movement
 * PLATFORM-AGNOSTIC - No SDL2 dependencies!
 */
class PlayerController : public Component {
public:
    float speed = 200.0f;

    void onUpdate(Object* owner) override {
        // Use abstract interface - no casting needed!
        auto* input = Input::getInstance();
        auto dt = Time::getInstance()->deltaTime;

        float dx = 0.0f;
        float dy = 0.0f;

        if (input->isKeyPressed(Key::W)) dy -= 1.0f;
        if (input->isKeyPressed(Key::S)) dy += 1.0f;
        if (input->isKeyPressed(Key::A)) dx -= 1.0f;
        if (input->isKeyPressed(Key::D)) dx += 1.0f;

        // Normalize diagonal movement
        if (dx != 0.0f && dy != 0.0f) {
            float length = std::sqrt(dx * dx + dy * dy);
            dx /= length;
            dy /= length;
        }

        owner->x += dx * speed * dt;
        owner->y += dy * speed * dt;

        // Check for ESC key to quit
        if (input->isKeyPressed(Key::Escape)) {
            Engine::getInstance()->stop();
        }
    }
};

/**
 * Simple rectangle renderer component - draws a colored rectangle
 * PLATFORM-AGNOSTIC - Uses abstract Screen interface only!
 */
class RectangleRenderer : public Component {
public:
    uint8_t r = 255, g = 255, b = 255, a = 255;
    float width = 50.0f;
    float height = 50.0f;

    RectangleRenderer(uint8_t r_, uint8_t g_, uint8_t b_,
                     float w = 50.0f, float h = 50.0f)
        : r(r_), g(g_), b(b_), width(w), height(h) {}

    void onDraw(Object* owner) override {
        // Use abstract Screen interface - no SDL2 dependency!
        auto* screen = Screen::getInstance();

        // Set color and draw rectangle
        screen->setColor(r, g, b, a);
        screen->drawRect(owner->x - width / 2, owner->y - height / 2, width, height);
    }
};

/**
 * SDL2 implementation of the Engine main loop for example game
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

        // Use abstract interfaces - no casting needed!
        auto* renderer = Screen::getInstance();
        auto* input = Input::getInstance();
        auto* time = Time::getInstance();

        auto lastTime = std::chrono::high_resolution_clock::now();

        while (running) {
            // Calculate delta time
            auto currentTime = std::chrono::high_resolution_clock::now();
            std::chrono::duration<float> delta = currentTime - lastTime;
            lastTime = currentTime;
            float dt = delta.count();

            // Handle SDL events (platform-specific, stays in implementation)
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
    (void)argc;
    (void)argv;

    std::cout << "IMGE Simple Game Example" << std::endl;
    std::cout << "Controls: WASD to move, ESC to quit" << std::endl;

    // Initialize SDL2
    if (SDL_Init(SDL_INIT_VIDEO | SDL_INIT_AUDIO) < 0) {
        std::cerr << "Failed to initialize SDL2: " << SDL_GetError() << std::endl;
        return 1;
    }

    try {
        // Create SDL2 service instances (these register themselves as singletons)
        imge::SDL2Renderer renderer;
        imge::SDL2Input input;
        imge::SDL2Audio audio;

        // Create engine instance (registers itself)
        imge::SDL2Engine engine;

        // Create a simple scene
        auto scene = std::make_shared<imge::Scene>();
        scene->width = 800;
        scene->height = 600;
        scene->backgroundColor = "#222222";

        // Add a simple object
        auto player = std::make_shared<imge::Object>(400.0f, 300.0f, "player");
        player->addTag("player");
        player->depth = 10.0f;

        // Add renderer component (blue rectangle)
        auto rectRenderer = std::shared_ptr<imge::RectangleRenderer>(
            new imge::RectangleRenderer(50, 150, 255, 50.0f, 50.0f));
        player->addComponent(rectRenderer, "renderer");

        // Add player controller component
        auto controller = std::shared_ptr<imge::PlayerController>(
            new imge::PlayerController());
        player->addComponent(controller, "controller");

        scene->addObject(player);

        // Add scene to engine
        engine.addScene("main", scene);

        // Initialize engine (now Screen::getInstance() and Audio::getInstance() will work)
        engine.init(800, 600, "IMGE Simple Game");

        // Run the engine
        engine.run();

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
