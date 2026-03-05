#include "imge/impl/SDL2Renderer.hpp"

#include <stdexcept>

namespace imge {

SDL2Renderer::SDL2Renderer() {
    setInstance(this);
}

SDL2Renderer::~SDL2Renderer() {
    if (renderer) {
        SDL_DestroyRenderer(renderer);
        renderer = nullptr;
    }
    if (window) {
        SDL_DestroyWindow(window);
        window = nullptr;
    }
}

void SDL2Renderer::init(int width_, int height_, const std::string& title) {
    width = width_;
    height = height_;

    // Create window
    window = SDL_CreateWindow(
        title.c_str(),
        SDL_WINDOWPOS_CENTERED,
        SDL_WINDOWPOS_CENTERED,
        width,
        height,
        SDL_WINDOW_SHOWN
    );

    if (!window) {
        throw std::runtime_error("Failed to create SDL window: " + std::string(SDL_GetError()));
    }

    // Create renderer
    renderer = SDL_CreateRenderer(
        window,
        -1,
        SDL_RENDERER_ACCELERATED | SDL_RENDERER_PRESENTVSYNC
    );

    if (!renderer) {
        SDL_DestroyWindow(window);
        window = nullptr;
        throw std::runtime_error("Failed to create SDL renderer: " + std::string(SDL_GetError()));
    }

    open = true;
}

void SDL2Renderer::clear() {
    // Set background color
    uint8_t r = (backgroundColor >> 24) & 0xFF;
    uint8_t g = (backgroundColor >> 16) & 0xFF;
    uint8_t b = (backgroundColor >> 8) & 0xFF;
    uint8_t a = backgroundColor & 0xFF;

    SDL_SetRenderDrawColor(renderer, r, g, b, a);
    SDL_RenderClear(renderer);
}

void SDL2Renderer::present() {
    SDL_RenderPresent(renderer);
}

void SDL2Renderer::setBackgroundColor(const std::string& color) {
    // Parse hex color string (#RRGGBB or #RRGGBBAA)
    if (color.empty() || color[0] != '#') {
        return; // Invalid format
    }

    std::string hex = color.substr(1);
    uint32_t colorValue = std::stoul(hex, nullptr, 16);

    // Handle #RRGGBB (6 hex digits)
    if (hex.length() == 6) {
        backgroundColor = (colorValue << 8) | 0xFF; // Add full alpha
    }
    // Handle #RRGGBBAA (8 hex digits)
    else if (hex.length() == 8) {
        backgroundColor = colorValue;
    }
}

void SDL2Renderer::setBackgroundImage(const std::string& filename) {
    // TODO: Implement background image support
    (void)filename;
}

int SDL2Renderer::getWidth() const {
    return width;
}

int SDL2Renderer::getHeight() const {
    return height;
}

bool SDL2Renderer::isOpen() const {
    return open;
}

void SDL2Renderer::close() {
    open = false;
}

void SDL2Renderer::setColor(uint8_t r, uint8_t g, uint8_t b, uint8_t a) {
    SDL_SetRenderDrawColor(renderer, r, g, b, a);
}

void SDL2Renderer::drawRect(float x, float y, float width, float height) {
    SDL_Rect rect{
        static_cast<int>(x),
        static_cast<int>(y),
        static_cast<int>(width),
        static_cast<int>(height)
    };
    SDL_RenderFillRect(renderer, &rect);
}

void SDL2Renderer::drawRectOutline(float x, float y, float width, float height) {
    SDL_Rect rect{
        static_cast<int>(x),
        static_cast<int>(y),
        static_cast<int>(width),
        static_cast<int>(height)
    };
    SDL_RenderDrawRect(renderer, &rect);
}

} // namespace imge
