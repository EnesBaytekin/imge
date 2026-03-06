#include "imge/config.hpp"
#include "imge/core/Object.hpp"
#include "imge/core/Scene.hpp"

// Game scripts - custom components
#include "scripts/PlayerController.hpp"
#include "scripts/RectangleRenderer.hpp"

#include <iostream>

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
        auto rectRenderer = std::shared_ptr<RectangleRenderer>(
            new RectangleRenderer(50, 150, 255, 50.0f, 50.0f));
        player->addComponent(rectRenderer, "renderer");

        // Add player controller component
        auto controller = std::shared_ptr<PlayerController>(
            new PlayerController());
        player->addComponent(controller, "controller");

        scene->addObject(player);

        // Add scene to engine
        engine.addScene("main", scene);

        // Initialize engine
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
