#include "imge/impl/SDL2Engine.hpp"

#include <iostream>
#include <stdexcept>

namespace imge {

SDL2Engine::SDL2Engine()
    : renderer(std::make_unique<SDL2Renderer>()),
      input(std::make_unique<SDL2Input>()),
      audio(std::make_unique<SDL2Audio>()) {
    setInstance(this);
}

SDL2Engine::~SDL2Engine() {
    // Destroy SDL2-dependent services BEFORE quitting SDL
    // This ensures proper cleanup order
    renderer.reset();
    input.reset();
    audio.reset();

    if (sdlInitialized) {
        SDL_Quit();
    }
}

void SDL2Engine::init(int width, int height, const std::string& title) {
    // Initialize SDL2
    if (SDL_Init(SDL_INIT_VIDEO | SDL_INIT_AUDIO) < 0) {
        throw std::runtime_error("Failed to initialize SDL2: " + std::string(SDL_GetError()));
    }
    sdlInitialized = true;

    // Initialize all services
    renderer->init(width, height, title);
    audio->init();

    Engine::init(width, height, title);
}

void SDL2Engine::run() {
    if (!scenes.empty()) running = true;

    auto lastTime = std::chrono::high_resolution_clock::now();

    while (running) {
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

        input->update();

        // Check for ESC key to quit
        if (input->isKeyPressed(imge::Key::Escape)) {
            running = false;
        }

        Time::getInstance()->update(dt);

        auto* scene = getCurrentScene();
        if (scene) {
            scene->update();
            renderer->clear();
            scene->draw();
            renderer->present();
        }

        SDL_Delay(1);
    }

    renderer->close();
}

} // namespace imge
