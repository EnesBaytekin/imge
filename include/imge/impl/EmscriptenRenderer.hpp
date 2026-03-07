#pragma once

#include "imge/services/Screen.hpp"

#include <emscripten.h>
#include <emscripten/html5.h>
#include <string>

namespace imge {

/**
 * Emscripten implementation of Screen service
 * Uses HTML5 Canvas 2D Context for rendering
 */
class EmscriptenRenderer : public Screen {
public:
    EmscriptenRenderer();
    ~EmscriptenRenderer() override;

    void init(int width, int height, const std::string& title) override;
    void clear() override;
    void present() override;
    void setBackgroundColor(const std::string& color) override;
    void setBackgroundImage(const std::string& filename) override;
    [[nodiscard]] int getWidth() const override;
    [[nodiscard]] int getHeight() const override;
    [[nodiscard]] bool isOpen() const override;
    void close() override;

    // Drawing primitives
    void setColor(uint8_t r, uint8_t g, uint8_t b, uint8_t a = 255) override;
    void drawRect(float x, float y, float width, float height) override;
    void drawRectOutline(float x, float y, float width, float height) override;
    void drawTexture(void* textureId, float x, float y, float width, float height) override;

private:
    int width = 0;
    int height = 0;
    bool open = false;
    uint32_t backgroundColor = 0x000000FF; // Black

    // Helper to get canvas 2D context
    [[nodiscard]] EMSCRIPTEN_WEBGL_CONTEXT_HANDLE getContext() const;
};

} // namespace imge
