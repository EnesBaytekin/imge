#include "imge/core/Engine.hpp"
#include <iostream>

int main(int argc, char* argv[]) {
    (void)argc;
    (void)argv;

    std::cout << "[IMGE] Game starting..." << std::endl;

    auto* engine = imge::Engine::getInstance();

    // Try to load and add main_scene.json
    auto scene = imge::Scene::fromFile("scenes/main_scene.json");
    if (scene) {
        std::cout << "[IMGE] main_scene.json found, loading scene..." << std::endl;
        engine->addScene("main", scene);
        engine->setScene("main");
    } else {
        std::cout << "[IMGE] Creating default scene..." << std::endl;
        auto defaultScene = std::make_shared<imge::Scene>();
        engine->addScene("default", defaultScene);
        engine->setScene("default");
    }

    // Register custom components
    // Note: Your custom components are automatically registered via IMGE_REGISTER_COMPONENT
    // Make sure all your scripts include IMGE_REGISTER_COMPONENT(YourClassName)

    std::cout << "[IMGE] Components registered, creating engine..." << std::endl;

    // Initialize and run
    engine->init(800, 600, "Game");
    std::cout << "[IMGE] Engine initialized, starting main loop..." << std::endl;
    engine->run();

    return 0;
}
