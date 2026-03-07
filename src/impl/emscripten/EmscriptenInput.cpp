#include "imge/impl/EmscriptenInput.hpp"

#include <emscripten.h>
#include <emscripten/html5.h>
#include <cstring>
#include <iostream>

namespace imge {

// Static instance for callbacks
static EmscriptenInput* instance = nullptr;

EmscriptenInput::EmscriptenInput() {
    setInstance(this);
    instance = this;

    // Register keyboard event callbacks
    emscripten_set_keydown_callback(EMSCRIPTEN_EVENT_TARGET_DOCUMENT, this, 1, keyCallback);
    emscripten_set_keyup_callback(EMSCRIPTEN_EVENT_TARGET_DOCUMENT, this, 1, keyCallback);

    // Register mouse event callbacks
    emscripten_set_mousemove_callback(EMSCRIPTEN_EVENT_TARGET_DOCUMENT, this, 1, mouseCallback);
    emscripten_set_mousedown_callback(EMSCRIPTEN_EVENT_TARGET_DOCUMENT, this, 1, mouseCallback);
    emscripten_set_mouseup_callback(EMSCRIPTEN_EVENT_TARGET_DOCUMENT, this, 1, mouseCallback);

    // Register wheel callback
    emscripten_set_wheel_callback(EMSCRIPTEN_EVENT_TARGET_DOCUMENT, this, 1, wheelCallback);
}

EmscriptenInput::~EmscriptenInput() {
    instance = nullptr;
}

void EmscriptenInput::update() {
    // Store previous keys
    previousKeys = currentKeys;

    // Clear just pressed/released tracking
    justPressedKeys.clear();
    justReleasedKeys.clear();

    // Clear wheel
    wheelX = 0.0f;
    wheelY = 0.0f;
}

bool EmscriptenInput::isKeyPressed(Key key) const {
    return currentKeys.find(key) != currentKeys.end();
}

bool EmscriptenInput::isKeyJustPressed(Key key) const {
    return justPressedKeys.find(key) != justPressedKeys.end();
}

bool EmscriptenInput::isKeyJustReleased(Key key) const {
    return justReleasedKeys.find(key) != justReleasedKeys.end();
}

bool EmscriptenInput::isMouseButtonPressed(MouseButton button) const {
    return currentMouseButtons.find(button) != currentMouseButtons.end();
}

std::pair<int, int> EmscriptenInput::getMousePosition() const {
    return {mouseX, mouseY};
}

std::pair<float, float> EmscriptenInput::getMouseWheel() const {
    return {wheelX, wheelY};
}

EM_BOOL EmscriptenInput::keyCallback(int eventType, const EmscriptenKeyboardEvent* keyEvent, void* userData) {
    auto* input = static_cast<EmscriptenInput*>(userData);

    Key key = jsKeyCodeToKey(keyEvent->code);

    if (eventType == EMSCRIPTEN_EVENT_KEYDOWN) {
        if (input->currentKeys.find(key) == input->currentKeys.end()) {
            input->justPressedKeys.insert(key);
        }
        input->currentKeys.insert(key);
    } else if (eventType == EMSCRIPTEN_EVENT_KEYUP) {
        input->currentKeys.erase(key);
        input->justReleasedKeys.insert(key);
    }

    return EM_TRUE;
}

EM_BOOL EmscriptenInput::mouseCallback(int eventType, const EmscriptenMouseEvent* mouseEvent, void* userData) {
    auto* input = static_cast<EmscriptenInput*>(userData);

    input->mouseX = mouseEvent->canvasX;
    input->mouseY = mouseEvent->canvasY;

    if (eventType == EMSCRIPTEN_EVENT_MOUSEDOWN) {
        MouseButton button = static_cast<MouseButton>(mouseEvent->button);
        input->currentMouseButtons.insert(button);
    } else if (eventType == EMSCRIPTEN_EVENT_MOUSEUP) {
        MouseButton button = static_cast<MouseButton>(mouseEvent->button);
        input->currentMouseButtons.erase(button);
    }

    return EM_TRUE;
}

EM_BOOL EmscriptenInput::wheelCallback(int eventType, const EmscriptenWheelEvent* wheelEvent, void* userData) {
    (void)eventType;
    auto* input = static_cast<EmscriptenInput*>(userData);

    input->wheelX = static_cast<float>(wheelEvent->deltaX);
    input->wheelY = static_cast<float>(wheelEvent->deltaY);

    return EM_TRUE;
}

Key EmscriptenInput::jsKeyCodeToKey(const char* code) {
    // Map JavaScript KeyboardEvent.code to our Key enum
    // See: https://developer.mozilla.org/en-US/docs/Web/API/KeyboardEvent/code

    if (strcmp(code, "KeyA") == 0) return Key::A;
    if (strcmp(code, "KeyB") == 0) return Key::B;
    if (strcmp(code, "KeyC") == 0) return Key::C;
    if (strcmp(code, "KeyD") == 0) return Key::D;
    if (strcmp(code, "KeyE") == 0) return Key::E;
    if (strcmp(code, "KeyF") == 0) return Key::F;
    if (strcmp(code, "KeyG") == 0) return Key::G;
    if (strcmp(code, "KeyH") == 0) return Key::H;
    if (strcmp(code, "KeyI") == 0) return Key::I;
    if (strcmp(code, "KeyJ") == 0) return Key::J;
    if (strcmp(code, "KeyK") == 0) return Key::K;
    if (strcmp(code, "KeyL") == 0) return Key::L;
    if (strcmp(code, "KeyM") == 0) return Key::M;
    if (strcmp(code, "KeyN") == 0) return Key::N;
    if (strcmp(code, "KeyO") == 0) return Key::O;
    if (strcmp(code, "KeyP") == 0) return Key::P;
    if (strcmp(code, "KeyQ") == 0) return Key::Q;
    if (strcmp(code, "KeyR") == 0) return Key::R;
    if (strcmp(code, "KeyS") == 0) return Key::S;
    if (strcmp(code, "KeyT") == 0) return Key::T;
    if (strcmp(code, "KeyU") == 0) return Key::U;
    if (strcmp(code, "KeyV") == 0) return Key::V;
    if (strcmp(code, "KeyW") == 0) return Key::W;
    if (strcmp(code, "KeyX") == 0) return Key::X;
    if (strcmp(code, "KeyY") == 0) return Key::Y;
    if (strcmp(code, "KeyZ") == 0) return Key::Z;

    if (strcmp(code, "Digit0") == 0) return Key::Num0;
    if (strcmp(code, "Digit1") == 0) return Key::Num1;
    if (strcmp(code, "Digit2") == 0) return Key::Num2;
    if (strcmp(code, "Digit3") == 0) return Key::Num3;
    if (strcmp(code, "Digit4") == 0) return Key::Num4;
    if (strcmp(code, "Digit5") == 0) return Key::Num5;
    if (strcmp(code, "Digit6") == 0) return Key::Num6;
    if (strcmp(code, "Digit7") == 0) return Key::Num7;
    if (strcmp(code, "Digit8") == 0) return Key::Num8;
    if (strcmp(code, "Digit9") == 0) return Key::Num9;

    if (strcmp(code, "Space") == 0) return Key::Space;
    if (strcmp(code, "Enter") == 0) return Key::Enter;
    if (strcmp(code, "Escape") == 0) return Key::Escape;
    if (strcmp(code, "ArrowLeft") == 0) return Key::Left;
    if (strcmp(code, "ArrowRight") == 0) return Key::Right;
    if (strcmp(code, "ArrowUp") == 0) return Key::Up;
    if (strcmp(code, "ArrowDown") == 0) return Key::Down;

    // Default fallback
    return Key::A;
}

} // namespace imge
