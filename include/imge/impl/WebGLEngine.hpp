#pragma once

#include "imge/core/Engine.hpp"
#include "imge/impl/EmscriptenInput.hpp"
#include "imge/impl/EmscriptenRenderer.hpp"
#include "imge/services/Input.hpp"
#include "imge/services/Screen.hpp"
#include "imge/services/Time.hpp"

#include <memory>

namespace imge {

/**
 * WebAssembly Engine implementation
 * Platform-specific engine implementation using Emscripten and HTML5 Canvas
 * All WebAssembly/Emscripten details are hidden inside this class
 */
class WebGLEngine : public Engine {
public:
    WebGLEngine();
    ~WebGLEngine() override;

    /**
     * Initialize Emscripten and all services
     * @param width Canvas width
     * @param height Canvas height
     * @param title Page title (window title in browser)
     */
    void init(int width, int height, const std::string& title) override;

    /**
     * Main game loop with Emscripten event handling
     * Uses emscripten_set_main_loop for browser compatibility
     */
    void run() override;

private:
    // Emscripten-specific service instances (owned by this engine)
    std::unique_ptr<EmscriptenRenderer> renderer;
    std::unique_ptr<EmscriptenInput> input;
    bool emscriptenInitialized = false;

    // Static callback for Emscripten main loop
    static void mainLoopCallback(void* userData);
    bool running = false;
};

} // namespace imge
