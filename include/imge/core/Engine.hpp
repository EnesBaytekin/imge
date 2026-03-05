#pragma once

#include "imge/core/Scene.hpp"
#include "imge/core/Singleton.hpp"
#include "imge/services/Audio.hpp"
#include "imge/services/Input.hpp"
#include "imge/services/Screen.hpp"
#include "imge/services/Time.hpp"

#include <memory>
#include <string>
#include <unordered_map>

namespace imge {

/**
 * Engine singleton - main game loop and scene management
 */
class Engine : public Singleton<Engine> {
public:
    ~Engine() override = default;

    /**
     * Initialize the engine
     * @param width Window width
     * @param height Window height
     * @param title Window title
     */
    virtual void init(int width, int height, const std::string& title = "IMGE Game");

    /**
     * Add a scene to the engine
     * @param name Scene name/identifier
     * @param scene Scene to add
     */
    void addScene(const std::string& name, std::shared_ptr<Scene> scene);

    /**
     * Set the active scene
     * @param name Scene name
     */
    void setScene(const std::string& name);

    /**
     * Get the current active scene
     * @return Pointer to current scene or nullptr if no scene is active
     */
    [[nodiscard]] Scene* getCurrentScene();

    /**
     * Get delta time
     * @return Delta time in seconds
     */
    [[nodiscard]] float getDeltaTime() const {
        return Time::getInstance()->deltaTime;
    }

    /**
     * Main game loop (abstract - must be implemented by platform-specific code)
     */
    virtual void run() = 0;

    /**
     * Stop the game loop
     */
    void stop() {
        running = false;
    }

    /**
     * Check if engine is still running
     */
    [[nodiscard]] bool isRunning() const {
        return running;
    }

protected:
    bool running = false;
    std::string currentSceneName;
    std::unordered_map<std::string, std::shared_ptr<Scene>> scenes;
};

inline void Engine::init(int width, int height, const std::string& title) {
    // Initialize screen (implementation-specific)
    Screen::getInstance()->init(width, height, title);

    // Initialize audio
    Audio::getInstance()->init();

    running = false;
}

inline void Engine::addScene(const std::string& name, std::shared_ptr<Scene> scene) {
    scenes[name] = scene;

    // Set as current scene if it's the first scene
    if (currentSceneName.empty()) {
        currentSceneName = name;
    }
}

inline void Engine::setScene(const std::string& name) {
    if (scenes.find(name) != scenes.end()) {
        currentSceneName = name;
    }
}

inline Scene* Engine::getCurrentScene() {
    auto it = scenes.find(currentSceneName);
    if (it != scenes.end()) {
        return it->second.get();
    }
    return nullptr;
}

} // namespace imge
