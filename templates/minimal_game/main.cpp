#include "imge/config.hpp"
#include "imge/core/Object.hpp"
#include "imge/core/Scene.hpp"

#include "PlayerController.hpp"

#include <iostream>

int main() {
    std::cout << "=== Minimal IMGE Game ===" << std::endl;
    std::cout << "Controls: WASD to move, ESC to quit" << std::endl;

    try {
        // Engine automatically uses selected implementation
        // SDL2 for desktop, WebGL for browser (see CMake)
        imge::EngineImpl engine;

        // Create scene
        auto scene = std::make_shared<imge::Scene>();
        scene->width = 800;
        scene->height = 600;
        scene->backgroundColor = "#222222";

        // Create player object
        auto player = std::make_shared<imge::Object>(400.0f, 300.0f, "player");
        player->addTag("player");
        player->depth = 10.0f;

        // Add player controller
        auto controller = std::shared_ptr<PlayerController>(new PlayerController());
        player->addComponent(controller, "PlayerController");

        scene->addObject(player);

        // Add scene and run
        engine.addScene("main", scene);
        engine.init(800, 600, "Minimal IMGE Game");
        engine.run();

    } catch (const std::exception& e) {
        std::cerr << "Error: " << e.what() << std::endl;
        return 1;
    }

    std::cout << "Game ended!" << std::endl;
    return 0;
}
