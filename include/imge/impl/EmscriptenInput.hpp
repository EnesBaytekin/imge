#pragma once

#include "imge/services/Input.hpp"

#include <emscripten.h>
#include <emscripten/html5.h>
#include <map>
#include <set>

namespace imge {

/**
 * Emscripten implementation of Input service
 * Captures browser keyboard and mouse events
 */
class EmscriptenInput : public Input {
public:
    EmscriptenInput();
    ~EmscriptenInput() override;

    void update() override;

    [[nodiscard]] bool isKeyPressed(Key key) const override;
    [[nodiscard]] bool isKeyJustPressed(Key key) const override;
    [[nodiscard]] bool isKeyJustReleased(Key key) const override;

    [[nodiscard]] bool isMouseButtonPressed(MouseButton button) const override;
    [[nodiscard]] std::pair<int, int> getMousePosition() const override;
    [[nodiscard]] std::pair<float, float> getMouseWheel() const override;

private:
    std::set<Key> currentKeys;
    std::set<Key> previousKeys;
    std::set<Key> justPressedKeys;
    std::set<Key> justReleasedKeys;

    std::set<MouseButton> currentMouseButtons;
    int mouseX = 0;
    int mouseY = 0;
    float wheelX = 0.0f;
    float wheelY = 0.0f;

    // Static callbacks for Emscripten
    static EM_BOOL keyCallback(int eventType, const EmscriptenKeyboardEvent* keyEvent, void* userData);
    static EM_BOOL mouseCallback(int eventType, const EmscriptenMouseEvent* mouseEvent, void* userData);
    static EM_BOOL wheelCallback(int eventType, const EmscriptenWheelEvent* wheelEvent, void* userData);

    // Convert JavaScript keycode to our Key enum
    static Key jsKeyCodeToKey(const char* code);
};

} // namespace imge
