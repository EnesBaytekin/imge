#pragma once

#include "imge/core/Singleton.hpp"
#include <cstdint>

namespace imge {

/**
 * Key codes (following common conventions)
 */
enum class Key : uint32_t {
    // Letters
    A = 0, B, C, D, E, F, G, H, I, J, K, L, M, N, O, P, Q, R, S, T, U, V, W, X, Y, Z,

    // Numbers
    Num0, Num1, Num2, Num3, Num4, Num5, Num6, Num7, Num8, Num9,

    // Special keys
    Space, Enter, Escape, Backspace, Tab,
    Left, Right, Up, Down,
    Home, End, PageUp, PageDown,
    Insert, Delete,

    // Function keys
    F1, F2, F3, F4, F5, F6, F7, F8, F9, F10, F11, F12,

    // Modifiers
    LeftShift, RightShift, LeftCtrl, RightCtrl,
    LeftAlt, RightAlt, LeftSuper, RightSuper
};

/**
 * Mouse buttons
 */
enum class MouseButton : uint32_t {
    Left = 1,
    Middle = 2,
    Right = 3,
    X1 = 4,
    X2 = 5
};

/**
 * Input service - abstract interface for input handling
 * Platform-specific implementations inherit from this
 */
class Input : public Singleton<Input> {
public:
    virtual ~Input() = default;

    /**
     * Update input state (call once per frame)
     */
    virtual void update() = 0;

    /**
     * Check if a key is currently pressed
     * @param key Key to check
     * @return true if key is pressed
     */
    [[nodiscard]] virtual bool isKeyPressed(Key key) const = 0;

    /**
     * Check if a key was just pressed this frame
     * @param key Key to check
     * @return true if key was just pressed
     */
    [[nodiscard]] virtual bool isKeyJustPressed(Key key) const = 0;

    /**
     * Check if a key was just released this frame
     * @param key Key to check
     * @return true if key was just released
     */
    [[nodiscard]] virtual bool isKeyJustReleased(Key key) const = 0;

    /**
     * Get mouse position
     * @return Pair of (x, y) coordinates
     */
    [[nodiscard]] virtual std::pair<int, int> getMousePosition() const = 0;

    /**
     * Check if a mouse button is currently pressed
     * @param button Button to check
     * @return true if button is pressed
     */
    [[nodiscard]] virtual bool isMouseButtonPressed(MouseButton button) const = 0;

    /**
     * Check if a mouse button was just pressed this frame
     * @param button Button to check
     * @return true if button was just pressed
     */
    [[nodiscard]] virtual bool isMouseButtonJustPressed(MouseButton button) const = 0;

    /**
     * Check if a mouse button was just released this frame
     * @param button Button to check
     * @return true if button was just released
     */
    [[nodiscard]] virtual bool isMouseButtonJustReleased(MouseButton button) const = 0;

    /**
     * Get mouse wheel movement
     * @return Pair of (x, y) scroll amounts
     */
    [[nodiscard]] virtual std::pair<float, float> getMouseWheel() const = 0;
};

} // namespace imge
