#include "imge/impl/WebGLEngine.hpp"

#include <emscripten.h>
#include <emscripten/html5.h>
#include <iostream>

namespace imge {

WebGLEngine::WebGLEngine()
    : renderer(std::make_unique<EmscriptenRenderer>()),
      input(std::make_unique<EmscriptenInput>()) {
    setInstance(this);
}

WebGLEngine::~WebGLEngine() {
    // Emscripten cleanup is automatic
}

void WebGLEngine::init(int width, int height, const std::string& title) {
    // Set canvas size
    emscripten_set_canvas_element_size("#canvas", width, height);

    // Initialize all services
    renderer->init(width, height, title);

    Engine::init(width, height, title);
    emscriptenInitialized = true;
}

void WebGLEngine::mainLoopCallback(void* userData) {
    auto* engine = static_cast<WebGLEngine*>(userData);

    if (!engine->running) {
        return;
    }

    // Update input
    engine->input->update();

    // Calculate delta time
    static auto lastTime = emscripten_performance_now();
    auto currentTime = emscripten_performance_now();
    float dt = static_cast<float>((currentTime - lastTime) / 1000.0);
    lastTime = currentTime;

    // Update time
    Time::getInstance()->update(dt);

    // Update scene
    auto* scene = engine->getCurrentScene();
    if (scene) {
        scene->update();
        engine->renderer->clear();
        scene->draw();
        engine->renderer->present();
    }
}

void WebGLEngine::run() {
    if (!scenes.empty()) {
        running = true;
    }

    // Use Emscripten's main loop mechanism
    // This integrates with the browser's requestAnimationFrame
    emscripten_set_main_loop_arg(mainLoopCallback, this, 0, true);
}

} // namespace imge
