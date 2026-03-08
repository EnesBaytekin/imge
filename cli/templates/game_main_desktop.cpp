#include "imge/impl/SDL2Engine.hpp"
#include "scripts/ComponentRegistry.hpp"
#include <iostream>

int main(int argc, char* argv[]) {
    (void)argc;
    (void)argv;

    std::cout << "[IMGE] Game starting..." << std::endl;

    // Create SDL2 engine instance
    auto engine = std::make_unique<imge::SDL2Engine>();
    imge::Engine::setInstance(engine.get());

    // Register custom components
    std::cout << "[IMGE] Registering custom components..." << std::endl;
    registerComponents();

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

    // Initialize and run
    engine->init(800, 600, "Game");
    std::cout << "[IMGE] Engine initialized, starting main loop..." << std::endl;
    engine->run();

    return 0;
}
