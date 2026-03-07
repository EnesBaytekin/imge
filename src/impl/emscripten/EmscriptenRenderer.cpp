#include "imge/impl/EmscriptenRenderer.hpp"

#include <emscripten.h>
#include <emscripten/html5.h>
#include <iostream>
#include <stdexcept>
#include <string>

namespace imge {

// Global state for color
static uint8_t currentR = 255, currentG = 255, currentB = 255, currentA = 255;

EmscriptenRenderer::EmscriptenRenderer() {
    setInstance(this);
}

EmscriptenRenderer::~EmscriptenRenderer() = default;

void EmscriptenRenderer::init(int width_, int height_, const std::string& title) {
    width = width_;
    height = height_;

    // Set page title
    EM_ASM_({ document.title = UTF8ToString($0); }, title.c_str());

    open = true;
}

void EmscriptenRenderer::clear() {
    // Set background color (stored as 0xRRGGBBAA)
    uint8_t r = (backgroundColor >> 24) & 0xFF;
    uint8_t g = (backgroundColor >> 16) & 0xFF;
    uint8_t b = (backgroundColor >> 8) & 0xFF;
    uint8_t a = backgroundColor & 0xFF;

    // Clear canvas with background color
    EM_ASM_({
        var canvas = document.getElementById('canvas');
        if (!canvas) {
            console.error('[Renderer] Canvas element not found!');
            return;
        }
        var ctx = canvas.getContext('2d');
        if (!ctx) {
            console.error('[Renderer] Failed to get 2d context!');
            return;
        }
        ctx.fillStyle = 'rgba(' + $0 + ',' + $1 + ',' + $2 + ',' + $3 + ')';
        ctx.fillRect(0, 0, canvas.width, canvas.height);
    }, r, g, b, (double)a / 255.0);
}

void EmscriptenRenderer::present() {
    // In Emscripten, present is automatic (canvas is double-buffered)
}

void EmscriptenRenderer::setBackgroundColor(const std::string& color) {
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

void EmscriptenRenderer::setBackgroundImage(const std::string& filename) {
    // TODO: Implement background image support
    (void)filename;
}

int EmscriptenRenderer::getWidth() const {
    return width;
}

int EmscriptenRenderer::getHeight() const {
    return height;
}

bool EmscriptenRenderer::isOpen() const {
    return open;
}

void EmscriptenRenderer::close() {
    open = false;
}

void EmscriptenRenderer::setColor(uint8_t r, uint8_t g, uint8_t b, uint8_t a) {
    currentR = r;
    currentG = g;
    currentB = b;
    currentA = a;
}

void EmscriptenRenderer::drawRect(float x, float y, float width, float height) {
    EM_ASM_({
        var canvas = document.getElementById('canvas');
        if (!canvas) {
            console.error('[Renderer] drawRect: Canvas not found');
            return;
        }
        var ctx = canvas.getContext('2d');
        ctx.fillStyle = 'rgba(' + $0 + ',' + $1 + ',' + $2 + ',' + $3 + ')';
        ctx.fillRect($4, $5, $6, $7);
    }, currentR, currentG, currentB, (double)currentA / 255.0, x, y, width, height);
}

void EmscriptenRenderer::drawRectOutline(float x, float y, float width, float height) {
    EM_ASM_({
        var canvas = document.getElementById('canvas');
        if (!canvas) {
            console.error('[Renderer] drawRectOutline: Canvas not found');
            return;
        }
        var ctx = canvas.getContext('2d');
        ctx.strokeStyle = 'rgba(' + $0 + ',' + $1 + ',' + $2 + ',' + $3 + ')';
        ctx.lineWidth = 1;
        ctx.strokeRect($4, $5, $6, $7);
    }, currentR, currentG, currentB, (double)currentA / 255.0, x, y, width, height);
}

void EmscriptenRenderer::drawTexture(void* textureId, float x, float y, float width, float height) {
    // textureId is actually a string pointer to image path
    const char* imagePath = static_cast<const char*>(textureId);

    EM_ASM_({
        var imagePath = UTF8ToString($0);
        var x = $1;
        var y = $2;
        var width = $3;
        var height = $4;

        var canvas = document.getElementById('canvas');
        if (!canvas) return;
        var ctx = canvas.getContext('2d');

        // Check if image is already loaded
        if (!window.imgeImages) {
            window.imgeImages = {};
        }

        var img = window.imgeImages[imagePath];
        if (!img) {
            // Read file from Emscripten virtual filesystem and create Blob URL
            var filePath = imagePath;

            // Try to read the file using Emscripten's FS
            try {
                var data = FS.readFile(filePath);
                var blob = new Blob([data], { type: 'image/png' });
                var url = URL.createObjectURL(blob);

                img = new Image();
                img.src = url;
                window.imgeImages[imagePath] = img;
            } catch (e) {
                console.error('[JS] Failed to read file from FS:', filePath, e);
                return;
            }
        }

        // Draw image
        if (img.complete) {
            ctx.drawImage(img, x, y, width, height);
        } else {
            img.onload = function() {
                ctx.drawImage(img, x, y, width, height);
            };
        }
    }, imagePath, x, y, width, height);
}

} // namespace imge
