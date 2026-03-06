#include "imge/config.hpp"
#include "imge/core/Object.hpp"
#include "imge/core/Scene.hpp"
#include "imge/services/Input.hpp"
#include "imge/services/Screen.hpp"
#include "imge/services/Time.hpp"

#include <cmath>
#include <iostream>

namespace game {

/**
 * Player controller component - WASD movement
 * PLATFORM-AGNOSTIC - No implementation dependencies!
 */
class PlayerController : public imge::Component {
public:
    float speed = 200.0f;

    void onUpdate(imge::Object* owner) override {
        // Use abstract interface - no casting needed!
        auto* input = imge::Input::getInstance();
        auto dt = imge::Time::getInstance()->deltaTime;

        float dx = 0.0f;
        float dy = 0.0f;

        if (input->isKeyPressed(imge::Key::W)) dy -= 1.0f;
        if (input->isKeyPressed(imge::Key::S)) dy += 1.0f;
        if (input->isKeyPressed(imge::Key::A)) dx -= 1.0f;
        if (input->isKeyPressed(imge::Key::D)) dx += 1.0f;

        // Normalize diagonal movement
        if (dx != 0.0f && dy != 0.0f) {
            float length = std::sqrt(dx * dx + dy * dy);
            dx /= length;
            dy /= length;
        }

        owner->x += dx * speed * dt;
        owner->y += dy * speed * dt;

        // Check for ESC key to quit
        if (input->isKeyPressed(imge::Key::Escape)) {
            imge::Engine::getInstance()->stop();
        }
    }
};

/**
 * Simple rectangle renderer component - draws a colored rectangle
 * PLATFORM-AGNOSTIC - Uses abstract Screen interface only!
 */
class RectangleRenderer : public imge::Component {
public:
    uint8_t r = 255, g = 255, b = 255, a = 255;
    float width = 50.0f;
    float height = 50.0f;

    RectangleRenderer(uint8_t r_, uint8_t g_, uint8_t b_,
                     float w = 50.0f, float h = 50.0f)
        : r(r_), g(g_), b(b_), width(w), height(h) {}

    void onDraw(imge::Object* owner) override {
        // Use abstract Screen interface - no implementation dependency!
        auto* screen = imge::Screen::getInstance();

        // Set color and draw rectangle
        screen->setColor(r, g, b, a);
        screen->drawRect(owner->x - width / 2, owner->y - height / 2, width, height);
    }
};

} // namespace game

/**
 * Simple example game - demonstrates basic engine usage
 */
int main(int argc, char* argv[]) {
    (void)argc;
    (void)argv;

    std::cout << "IMGE Simple Game Example" << std::endl;
    std::cout << "Controls: WASD to move, ESC to quit" << std::endl;

    try {
        // Create engine
        // Platform implementation is selected by build configuration
        imge::EngineImpl engine;

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
        auto rectRenderer = std::shared_ptr<game::RectangleRenderer>(
            new game::RectangleRenderer(50, 150, 255, 50.0f, 50.0f));
        player->addComponent(rectRenderer, "renderer");

        // Add player controller component
        auto controller = std::shared_ptr<game::PlayerController>(
            new game::PlayerController());
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
        return 1;
    }

    std::cout << "Game ended successfully!" << std::endl;
    return 0;
}
