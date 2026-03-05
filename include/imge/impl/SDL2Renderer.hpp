#pragma once

#include "imge/services/Screen.hpp"

#include <SDL2/SDL.h>
#include <string>

namespace imge {

/**
 * SDL2 implementation of Screen service
 */
class SDL2Renderer : public Screen {
public:
    SDL2Renderer();
    ~SDL2Renderer() override;

    void init(int width, int height, const std::string& title) override;
    void clear() override;
    void present() override;
    void setBackgroundColor(const std::string& color) override;
    void setBackgroundImage(const std::string& filename) override;
    [[nodiscard]] int getWidth() const override;
    [[nodiscard]] int getHeight() const override;
    [[nodiscard]] bool isOpen() const override;
    void close() override;

    /**
     * Get the SDL renderer
     */
    [[nodiscard]] SDL_Renderer* getRenderer() const {
        return renderer;
    }

    /**
     * Get the SDL window
     */
    [[nodiscard]] SDL_Window* getWindow() const {
        return window;
    }

private:
    SDL_Window* window = nullptr;
    SDL_Renderer* renderer = nullptr;
    int width = 0;
    int height = 0;
    bool open = false;
    uint32_t backgroundColor = 0xFF000000; // Black by default
};

} // namespace imge
